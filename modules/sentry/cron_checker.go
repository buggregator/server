package sentry

import (
	"database/sql"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// cronTick is how often the checker looks for missed and hung runs.
const cronTick = 30 * time.Second

// cronDownGap: a heartbeat older than this means the server (or the
// machine) was not running, not just a slow tick.
const cronDownGap = 3 * cronTick

// cronWindow is a span of time the server was down.
type cronWindow struct{ From, To time.Time }

// cronChecker finds missed slots and hung runs. It remembers the downtimes
// it saw since start, so a slot that fell into one is unknown, not missed.
type cronChecker struct {
	db        *sql.DB
	mu        sync.Mutex
	downtimes []cronWindow
	notify    func(project string, kinds []string)
}

// run ticks until the process exits.
func (c *cronChecker) run() {
	t := time.NewTicker(cronTick)
	defer t.Stop()
	for now := range t.C {
		c.tick(now)
	}
}

// tick runs one check at now and writes the heartbeat.
func (c *cronChecker) tick(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now = now.UTC()

	var alive string
	err := c.db.QueryRow(`SELECT alive_at FROM cron_heartbeat WHERE id = 1`).Scan(&alive)
	if err == nil {
		if last := parseCronTime(alive); !last.IsZero() && now.Sub(last) > cronDownGap {
			c.downtimes = append(c.downtimes, cronWindow{From: last, To: now})
			if len(c.downtimes) > 20 {
				c.downtimes = c.downtimes[len(c.downtimes)-20:]
			}
		}
	} else if err != sql.ErrNoRows {
		slog.Warn("cron: cannot read heartbeat", "err", err)
		return
	}

	projects, err := checkCrons(c.db, now, c.downtimes)
	if err != nil {
		slog.Warn("cron: check failed", "err", err)
	}
	if _, err := c.db.Exec(`INSERT INTO cron_heartbeat (id, alive_at) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET alive_at = excluded.alive_at`, cronTime(now)); err != nil {
		slog.Warn("cron: cannot write heartbeat", "err", err)
	}
	if c.notify != nil {
		for _, p := range projects {
			c.notify(p, []string{"cron"})
		}
	}
}

// checkCrons marks hung runs timeout and slots without a check-in missed
// (unknown when the slot's margin overlaps a downtime). It returns the
// projects it changed.
func checkCrons(db *sql.DB, now time.Time, downtimes []cronWindow) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	changed := map[int64]string{}

	// Hung runs: in_progress for longer than the monitor's max_runtime.
	rows, err := tx.Query(`SELECT c.id, c.monitor_id, c.started_at, m.max_runtime, m.project_id
		FROM cron_checkins c JOIN cron_monitors m ON m.id = c.monitor_id WHERE c.status = 'in_progress'`)
	if err != nil {
		return nil, err
	}
	type hung struct{ id, monitor int64 }
	var timeouts []hung
	for rows.Next() {
		var (
			id, monitor int64
			started     string
			maxRuntime  int
			project     string
		)
		if err := rows.Scan(&id, &monitor, &started, &maxRuntime, &project); err != nil {
			rows.Close()
			return nil, err
		}
		if now.Sub(parseCronTime(started)) > time.Duration(maxRuntime)*time.Minute {
			timeouts = append(timeouts, hung{id, monitor})
			changed[monitor] = project
		}
	}
	rows.Close()
	for _, h := range timeouts {
		if _, err := tx.Exec(`UPDATE cron_checkins SET status = 'timeout' WHERE id = ?`, h.id); err != nil {
			return nil, err
		}
		// The slots the hung run sat on are its fault, not missed runs:
		// expect the next one after now.
		m, err := scanCronMonitor(tx.QueryRow(`SELECT `+cronMonitorCols+` FROM cron_monitors WHERE id = ?`, h.monitor))
		if err != nil {
			return nil, err
		}
		var next any
		if n, ok := m.next(now); ok {
			next = cronTime(n)
		}
		if _, err := tx.Exec(`UPDATE cron_monitors SET next_expected_at = ? WHERE id = ?`, next, h.monitor); err != nil {
			return nil, err
		}
	}

	// Missed slots. A monitor with a run in progress is not missed: a long
	// run may span a slot, and timeout covers one that hangs.
	rows, err = tx.Query(`SELECT `+cronMonitorCols+` FROM cron_monitors m
		WHERE next_expected_at IS NOT NULL AND next_expected_at < ?
		AND NOT EXISTS (SELECT 1 FROM cron_checkins c WHERE c.monitor_id = m.id AND c.status = 'in_progress')`,
		cronTime(now))
	if err != nil {
		return nil, err
	}
	var due []cronMonitorRow
	for rows.Next() {
		m, err := scanCronMonitor(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		due = append(due, m)
	}
	rows.Close()
	for _, m := range due {
		slot := parseCronTime(m.NextExpected.String)
		margin := time.Duration(m.CheckinMargin) * time.Minute
		if !now.After(slot.Add(margin)) {
			continue
		}
		status := cronMissed
		for _, w := range downtimes {
			if slot.Before(w.To) && slot.Add(margin).After(w.From) {
				status = cronUnknown
				break
			}
		}
		id := strings.ReplaceAll(event.GenerateUUID(), "-", "")
		if _, err := tx.Exec(`INSERT INTO cron_checkins (monitor_id, check_in_id, status, started_at, source, server_made, expected_at)
			VALUES (?, ?, ?, ?, ?, 1, ?)`, m.ID, id, status, cronTime(slot), m.Source, cronTime(slot)); err != nil {
			return nil, err
		}
		// One row per gap: skip the slots whose margin has passed too.
		next, ok := m.next(slot)
		for i := 0; ok && i < 1000 && now.After(next.Add(margin)); i++ {
			next, ok = m.next(next)
		}
		if ok && now.After(next.Add(margin)) {
			next, ok = m.next(now.Add(-margin))
		}
		var nextVal any
		if ok {
			nextVal = cronTime(next)
		}
		if _, err := tx.Exec(`UPDATE cron_monitors SET next_expected_at = ? WHERE id = ?`, nextVal, m.ID); err != nil {
			return nil, err
		}
		changed[m.ID] = m.Project
	}

	seen := map[string]bool{}
	var projects []string
	for id, p := range changed {
		if _, err := tx.Exec(`UPDATE cron_monitors SET last_status = COALESCE((SELECT status FROM cron_checkins
			WHERE monitor_id = ? ORDER BY started_at DESC, id DESC LIMIT 1), '') WHERE id = ?`, id, id); err != nil {
			return nil, err
		}
		if !seen[p] {
			seen[p] = true
			projects = append(projects, p)
		}
	}
	return projects, tx.Commit()
}
