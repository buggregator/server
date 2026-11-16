// Package retention deletes old events in the background when the config
// asks for it. By default nothing is deleted.
package retention

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// Interval is how often the cleaner runs.
const Interval = time.Minute

// batch bounds one DELETE: the UUID list is a set of SQL parameters.
const batch = 500

// Policy says what to delete. Zero values switch a rule off; pinned events
// are never deleted and do not count towards MaxEvents.
type Policy struct {
	MaxAge    time.Duration // delete events older than this
	MaxEvents int           // keep at most this many unpinned events, newest first
}

// On reports whether any rule is set.
func (p Policy) On() bool { return p.MaxAge > 0 || p.MaxEvents > 0 }

// Pruner deletes rows that have no event card older than cutoff (Sentry
// logs, metrics, traces); module.AgePruner satisfies it.
type Pruner interface {
	PruneBefore(ctx context.Context, cutoff time.Time) (int, error)
}

// Cleaner applies a Policy. Deletes go through the store, so attachments and
// module data go with the events, as with a delete from the UI.
type Cleaner struct {
	db        *sql.DB
	store     event.Store
	policy    Policy
	onDeleted func(uuid, project string)
	pruners   []Pruner
	now       func() time.Time
}

// New returns a cleaner. onDeleted (may be nil) is called for each deleted
// event, so open UIs drop it. pruners age out rows with no event card when
// MaxAge is set.
func New(db *sql.DB, store event.Store, policy Policy, onDeleted func(uuid, project string), pruners ...Pruner) *Cleaner {
	return &Cleaner{db: db, store: store, policy: policy, onDeleted: onDeleted, pruners: pruners, now: time.Now}
}

// Run cleans at once and then every Interval until ctx is done.
func (c *Cleaner) Run(ctx context.Context) {
	if !c.policy.On() {
		return
	}
	slog.Info("retention on", "max_age", c.policy.MaxAge.String(), "max_events", c.policy.MaxEvents)
	t := time.NewTicker(Interval)
	defer t.Stop()
	for {
		if n, err := c.RunOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("retention: cleanup failed", "deleted", n, "err", err)
		} else if n > 0 {
			slog.Info("retention: deleted events", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

type victim struct{ uuid, project string }

// RunOnce deletes what the policy allows to delete and returns how many rows
// went: events, and with MaxAge the pruners' rows.
func (c *Cleaner) RunOnce(ctx context.Context) (int, error) {
	total := 0
	if c.policy.MaxAge > 0 {
		cutoff := float64(c.now().Add(-c.policy.MaxAge).UnixNano()) / 1e9
		n, err := c.deleteWhile(ctx, `SELECT uuid, COALESCE(project, '') FROM events
			WHERE is_pinned = 0 AND CAST(timestamp AS REAL) < ? LIMIT ?`, cutoff, batch)
		total += n
		if err != nil {
			return total, err
		}
		for _, p := range c.pruners {
			n, err := p.PruneBefore(ctx, c.now().Add(-c.policy.MaxAge))
			total += n
			if err != nil {
				return total, err
			}
		}
	}
	if c.policy.MaxEvents > 0 {
		n, err := c.deleteWhile(ctx, `SELECT uuid, COALESCE(project, '') FROM events
			WHERE is_pinned = 0 ORDER BY CAST(timestamp AS REAL) DESC LIMIT ? OFFSET ?`, batch, c.policy.MaxEvents)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// deleteWhile selects a batch with query and deletes it until none is left.
func (c *Cleaner) deleteWhile(ctx context.Context, query string, args ...any) (int, error) {
	total := 0
	// A bound so a delete that removes nothing cannot spin; the next run
	// goes on where this one stopped.
	for range 1000 {
		victims, err := c.selectVictims(ctx, query, args...)
		if err != nil || len(victims) == 0 {
			return total, err
		}
		uuids := make([]string, len(victims))
		for i, v := range victims {
			uuids[i] = v.uuid
		}
		if err := c.store.DeleteAll(ctx, event.DeleteOptions{UUIDs: uuids, KeepPinned: true}); err != nil {
			return total, err
		}
		total += len(victims)
		if c.onDeleted != nil {
			for _, v := range victims {
				c.onDeleted(v.uuid, v.project)
			}
		}
	}
	return total, nil
}

func (c *Cleaner) selectVictims(ctx context.Context, query string, args ...any) ([]victim, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []victim
	for rows.Next() {
		var v victim
		if err := rows.Scan(&v.uuid, &v.project); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
