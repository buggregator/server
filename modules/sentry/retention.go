package sentry

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"
)

// The Sentry tables hold rows the events table does not: the structured copy
// of an error (deleted here with its event card), and logs, metrics,
// transactions and spans, which have no card at all (aged out by retention).

// deleteBatch bounds the UUIDs of one statement (SQL parameters).
const deleteBatch = 500

// OnEventsDeleted removes the structured copy of deleted error events.
func (m *Module) OnEventsDeleted(uuids []string) {
	if m.db == nil || len(uuids) == 0 {
		return
	}
	if err := deleteErrorEvents(context.Background(), m.db, uuids); err != nil {
		slog.Warn("sentry: failed to delete rows of deleted events", "err", err)
	}
}

func deleteErrorEvents(ctx context.Context, db *sql.DB, uuids []string) error {
	for start := 0; start < len(uuids); start += deleteBatch {
		chunk := uuids[start:min(start+deleteBatch, len(uuids))]
		in := "(" + strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",") + ")"
		args := make([]any, len(chunk))
		for i, u := range chunk {
			args[i] = u
		}
		ids := `(SELECT id FROM sentry_error_events WHERE event_id IN ` + in + `)`
		if err := inTx(ctx, db, func(tx *sql.Tx) error {
			for _, st := range []string{
				`DELETE FROM sentry_breadcrumbs WHERE error_event_id IN ` + ids,
				`DELETE FROM sentry_exceptions WHERE error_event_id IN ` + ids,
				`DELETE FROM sentry_error_events WHERE event_id IN ` + in,
			} {
				if _, err := tx.ExecContext(ctx, st, args...); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// PruneBefore deletes Sentry rows older than cutoff that no event card keeps:
// logs and metrics by received_at, transactions and spans by start_ts (or,
// without one, by their trace's last_seen), and
// structured errors whose card is already gone. Errors of a kept (pinned)
// card stay. It implements module.AgePruner for retention.max_age.
func (m *Module) PruneBefore(ctx context.Context, cutoff time.Time) (int, error) {
	if m.db == nil {
		return 0, nil
	}
	return pruneBefore(ctx, m.db, cutoff)
}

func pruneBefore(ctx context.Context, db *sql.DB, cutoff time.Time) (int, error) {
	// received_at is datetime('now'); start_ts is ISO with T and Z. Each bound
	// is in its column's format so the text comparison is by time.
	dt := cutoff.UTC().Format("2006-01-02 15:04:05")
	iso := cutoff.UTC().Format("2006-01-02T15:04:05")
	orphan := `received_at < ? AND event_id NOT IN (SELECT uuid FROM events)`
	// A span or transaction sent without a start time ages with its trace's
	// last write.
	noStart := `start_ts IS NULL AND trace_id IN (SELECT trace_id FROM sentry_traces WHERE last_seen < ?)`

	deleted := 0
	err := inTx(ctx, db, func(tx *sql.Tx) error {
		stmts := []struct {
			sql   string
			arg   string
			count bool
		}{
			{`CREATE TEMP TABLE IF NOT EXISTS sentry_clear_traces (trace_id TEXT PRIMARY KEY)`, "", false},
			{`DELETE FROM sentry_clear_traces`, "", false},
			{`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_spans WHERE start_ts < ?`, iso, false},
			{`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_transactions WHERE start_ts < ?`, iso, false},
			{`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_spans WHERE ` + noStart, dt, false},
			{`INSERT OR IGNORE INTO sentry_clear_traces SELECT trace_id FROM sentry_transactions WHERE ` + noStart, dt, false},
			{`DELETE FROM sentry_spans WHERE transaction_id IN (SELECT id FROM sentry_transactions WHERE ` + noStart + `)`, dt, true},
			{`DELETE FROM sentry_spans WHERE ` + noStart, dt, true},
			{`DELETE FROM sentry_transactions WHERE ` + noStart, dt, true},
			{`DELETE FROM sentry_spans WHERE transaction_id IN (SELECT id FROM sentry_transactions WHERE start_ts < ?)`, iso, true},
			{`DELETE FROM sentry_spans WHERE start_ts < ?`, iso, true},
			{`DELETE FROM sentry_transactions WHERE start_ts < ?`, iso, true},
			{`DELETE FROM sentry_logs WHERE received_at < ?`, dt, true},
			{`DELETE FROM sentry_metrics WHERE received_at < ?`, dt, true},
			{`DELETE FROM sentry_breadcrumbs WHERE error_event_id IN (SELECT id FROM sentry_error_events WHERE ` + orphan + `)`, dt, false},
			{`DELETE FROM sentry_exceptions WHERE error_event_id IN (SELECT id FROM sentry_error_events WHERE ` + orphan + `)`, dt, false},
			{`DELETE FROM sentry_error_events WHERE ` + orphan, dt, true},
		}
		for _, st := range stmts {
			var args []any
			if st.arg != "" {
				args = []any{st.arg}
			}
			res, err := tx.ExecContext(ctx, st.sql, args...)
			if err != nil {
				return err
			}
			if st.count {
				n, _ := res.RowsAffected()
				deleted += int(n)
			}
		}
		for _, st := range tidyTracesStmts {
			if _, err := tx.ExecContext(ctx, st); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

func inTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
