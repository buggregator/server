package sentry

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// Check-in statuses. The SDK sends the first three; the server writes the
// others: missed (no check-in for a slot), timeout (in_progress longer than
// max_runtime), unknown (the slot fell while Buggregator was down).
const (
	cronInProgress = "in_progress"
	cronOK         = "ok"
	cronError      = "error"
	cronMissed     = "missed"
	cronTimeout    = "timeout"
	cronUnknown    = "unknown"
)

// Where a check-in came from.
const (
	cronSourceSentry = "sentry"
	cronSourcePing   = "ping"
)

const (
	// maxCronMonitors caps auto-created monitors per project: a script that
	// puts a run id or a date into the slug stops at 500 monitors instead of
	// growing the list without end. Ingestion is open, so it is no limit on
	// a client that varies the project too.
	maxCronMonitors = 500
	// maxPingBody is how much of a ping body (the job's output) is kept: its tail.
	maxPingBody = 10 << 10

	defaultCheckinMargin = 1  // minutes, Sentry's default
	defaultMaxRuntime    = 30 // minutes, Sentry's default
)

var (
	errTooManyMonitors = errors.New("too many cron monitors in this project")
	errBadCheckIn      = errors.New("invalid check-in")
	slugPattern        = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
)

// cronConfig is the monitor part of a check-in: Sentry's monitor_config or
// the query of a ping. Nil fields keep what the monitor has.
type cronConfig struct {
	Schedule      *cronSchedule
	Timezone      *string
	CheckinMargin *int
	MaxRuntime    *int
}

// cronCheckIn is one check-in as received.
type cronCheckIn struct {
	Project     string
	Slug        string
	CheckInID   string
	Status      string
	Environment string
	TraceID     string
	Source      string
	Body        string
	Duration    *float64 // seconds
	Config      *cronConfig
}

// validate checks the slug, the status and the schedule of the config.
func (c *cronCheckIn) validate() error {
	if !slugPattern.MatchString(c.Slug) {
		return fmt.Errorf("%w: monitor slug %q", errBadCheckIn, c.Slug)
	}
	switch c.Status {
	case cronInProgress, cronOK, cronError:
	default:
		return fmt.Errorf("%w: status %q", errBadCheckIn, c.Status)
	}
	if c.Config != nil {
		m := cronMonitorRow{TZ: "UTC"}
		m.apply(c.Config)
		if _, err := m.Schedule.compile(m.TZ); err != nil {
			return err
		}
	}
	return nil
}

// cronMonitorRow is a cron_monitors row.
type cronMonitorRow struct {
	ID            int64
	Project       string
	Slug          string
	Source        string
	Schedule      cronSchedule
	TZ            string
	CheckinMargin int
	MaxRuntime    int
	LastStatus    string
	LastCheckinAt sql.NullString
	NextExpected  sql.NullString
	CreatedAt     string
}

func (m *cronMonitorRow) apply(c *cronConfig) {
	if c.Schedule != nil {
		m.Schedule = *c.Schedule
	}
	if c.Timezone != nil {
		m.TZ = *c.Timezone
		if m.TZ == "" {
			m.TZ = "UTC"
		}
	}
	if c.CheckinMargin != nil && *c.CheckinMargin >= 0 {
		m.CheckinMargin = *c.CheckinMargin
	}
	if c.MaxRuntime != nil && *c.MaxRuntime > 0 {
		m.MaxRuntime = *c.MaxRuntime
	}
}

// next is the monitor's first slot after t; ok is false without a schedule.
func (m *cronMonitorRow) next(t time.Time) (time.Time, bool) {
	fn, err := m.Schedule.compile(m.TZ)
	if err != nil || fn == nil {
		return time.Time{}, false
	}
	n := fn(t)
	return n, !n.IsZero()
}

const cronMonitorCols = `id, project_id, slug, source, schedule_type, schedule_value, schedule_unit, timezone,
	checkin_margin, max_runtime, last_status, last_checkin_at, next_expected_at, created_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanCronMonitor(s rowScanner) (cronMonitorRow, error) {
	var m cronMonitorRow
	err := s.Scan(&m.ID, &m.Project, &m.Slug, &m.Source, &m.Schedule.Type, &m.Schedule.Value, &m.Schedule.Unit,
		&m.TZ, &m.CheckinMargin, &m.MaxRuntime, &m.LastStatus, &m.LastCheckinAt, &m.NextExpected, &m.CreatedAt)
	return m, err
}

type queryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

func findCronMonitor(q queryer, project, slug string) (cronMonitorRow, error) {
	return scanCronMonitor(q.QueryRow(`SELECT `+cronMonitorCols+` FROM cron_monitors WHERE project_id = ? AND slug = ?`,
		projectKey(project), slug))
}

// recordCheckIn stores a check-in: it creates the monitor on its first
// check-in, applies the config, opens or closes a run, and moves the
// monitor's next expected slot. It returns the run's check_in_id.
func recordCheckIn(db *sql.DB, c cronCheckIn, now time.Time) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}
	project := projectKey(c.Project)
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	m, err := findCronMonitor(tx, project, c.Slug)
	if errors.Is(err, sql.ErrNoRows) {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM cron_monitors WHERE project_id = ?`, project).Scan(&n); err != nil {
			return "", err
		}
		if n >= maxCronMonitors {
			return "", errTooManyMonitors
		}
		res, err := tx.Exec(`INSERT INTO cron_monitors (project_id, slug, source, checkin_margin, max_runtime, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, project, c.Slug, c.Source, defaultCheckinMargin, defaultMaxRuntime, cronTime(now))
		if err != nil {
			return "", err
		}
		m = cronMonitorRow{Project: project, Slug: c.Slug, Source: c.Source, TZ: "UTC",
			CheckinMargin: defaultCheckinMargin, MaxRuntime: defaultMaxRuntime}
		m.ID, _ = res.LastInsertId()
	} else if err != nil {
		return "", err
	}
	if c.Config != nil {
		m.apply(c.Config)
	}

	id, err := writeRun(tx, m, c, now)
	if err != nil {
		return "", err
	}

	// The monitor shows its newest run. The next slot counts from now: a run
	// that just ended expects the slot after it, never one already passed.
	var next any
	if n, ok := m.next(now); ok {
		next = cronTime(n)
	}
	if _, err := tx.Exec(`UPDATE cron_monitors SET schedule_type = ?, schedule_value = ?, schedule_unit = ?, timezone = ?,
		checkin_margin = ?, max_runtime = ?, last_checkin_at = ?, next_expected_at = ?,
		last_status = COALESCE((SELECT status FROM cron_checkins WHERE monitor_id = ? ORDER BY started_at DESC, id DESC LIMIT 1), '')
		WHERE id = ?`,
		m.Schedule.Type, m.Schedule.Value, m.Schedule.Unit, m.TZ, m.CheckinMargin, m.MaxRuntime,
		cronTime(now), next, m.ID, m.ID); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

// writeRun opens a run (in_progress) or closes one (ok, error). A closing
// check-in finds its run by check_in_id; a ping without one closes the
// newest open ping run. With no run to close, it is a run of its own.
// Requests can arrive out of order: an in_progress for a run that already
// ended changes nothing.
func writeRun(tx *sql.Tx, m cronMonitorRow, c cronCheckIn, now time.Time) (string, error) {
	var (
		rowID   int64
		started string
		status  string
	)
	var err error
	switch {
	case c.CheckInID != "":
		err = tx.QueryRow(`SELECT id, started_at, status FROM cron_checkins WHERE monitor_id = ? AND check_in_id = ?`,
			m.ID, c.CheckInID).Scan(&rowID, &started, &status)
	case c.Source == cronSourcePing && c.Status != cronInProgress:
		err = tx.QueryRow(`SELECT id, started_at, status, check_in_id FROM cron_checkins
			WHERE monitor_id = ? AND status = 'in_progress' AND source = 'ping' ORDER BY started_at DESC, id DESC LIMIT 1`,
			m.ID).Scan(&rowID, &started, &status, &c.CheckInID)
	default:
		err = sql.ErrNoRows
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if c.CheckInID == "" {
		c.CheckInID = strings.ReplaceAll(event.GenerateUUID(), "-", "")
	}

	if rowID != 0 {
		if c.Status == cronInProgress {
			return c.CheckInID, nil
		}
		dur := durationMS(c.Duration)
		if dur == nil {
			ms := now.Sub(parseCronTime(started)).Milliseconds()
			dur = &ms
		}
		// A run the server already called timeout keeps that status.
		newStatus := c.Status
		if status != cronInProgress {
			newStatus = status
		}
		_, err := tx.Exec(`UPDATE cron_checkins SET status = ?, finished_at = ?, duration_ms = ?,
			environment = CASE WHEN ? <> '' THEN ? ELSE environment END,
			trace_id = CASE WHEN ? <> '' THEN ? ELSE trace_id END,
			body = CASE WHEN ? <> '' THEN ? ELSE body END WHERE id = ?`,
			newStatus, cronTime(now), *dur, c.Environment, c.Environment, c.TraceID, c.TraceID, c.Body, c.Body, rowID)
		return c.CheckInID, err
	}

	start := now
	var finished any
	dur := durationMS(c.Duration)
	if c.Status != cronInProgress {
		finished = cronTime(now)
		if dur != nil {
			start = now.Add(-time.Duration(*dur) * time.Millisecond)
		}
	} else {
		dur = nil
	}
	_, err = tx.Exec(`INSERT INTO cron_checkins (monitor_id, check_in_id, status, started_at, finished_at, duration_ms,
		environment, trace_id, source, body) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, c.CheckInID, c.Status, cronTime(start), finished, nullableInt(dur), c.Environment, c.TraceID, c.Source, c.Body)
	return c.CheckInID, err
}

func durationMS(sec *float64) *int64 {
	if sec == nil || *sec < 0 || math.IsNaN(*sec) || math.IsInf(*sec, 0) {
		return nil
	}
	// Clamp to a year: a huge value would overflow int64 milliseconds.
	ms := int64(math.Round(math.Min(*sec, 365*24*3600) * 1000))
	return &ms
}

func nullableInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// bodyTail keeps the last maxPingBody bytes of a ping body as valid UTF-8.
func bodyTail(b []byte) string {
	if len(b) > maxPingBody {
		b = b[len(b)-maxPingBody:]
	}
	return strings.ToValidUTF8(string(b), "")
}
