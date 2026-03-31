package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// storeErrorEvent stores a parsed error event with its exceptions and breadcrumbs
// in a single DB transaction.
func storeErrorEvent(db *sql.DB, ev *ErrorEvent, payload json.RawMessage, projectID string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	id := event.GenerateUUID()
	fingerprint := computeFingerprint(ev)

	level := ev.Level
	if level == "" {
		level = "error"
	}

	var handled *int
	if ev.Exception != nil && len(ev.Exception.Values) > 0 {
		if m := ev.Exception.Values[0].Mechanism; m != nil && m.Handled != nil {
			v := 0
			if *m.Handled {
				v = 1
			}
			handled = &v
		}
	}

	eventTS := parseTimestamp(ev.Timestamp.Number())

	_, err = tx.Exec(
		`INSERT INTO sentry_error_events
			(id, event_id, project_id, trace_id, span_id, level, environment, release, server_name, platform, "transaction", fingerprint, handled, event_ts, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		ev.EventID,
		nullIfEmpty(projectID),
		nullIfEmpty(ev.traceID()),
		nullIfEmpty(ev.spanID()),
		level,
		nullIfEmpty(ev.Environment),
		nullIfEmpty(ev.Release),
		nullIfEmpty(ev.ServerName),
		nullIfEmpty(ev.Platform),
		nullIfEmpty(ev.Transaction),
		fingerprint,
		handled,
		eventTS,
		string(payload),
	)
	if err != nil {
		return "", fmt.Errorf("insert sentry_error_events: %w", err)
	}

	// Insert exceptions.
	if ev.Exception != nil {
		for i, exc := range ev.Exception.Values {
			excID := event.GenerateUUID()
			var excHandled *int
			if exc.Mechanism != nil && exc.Mechanism.Handled != nil {
				v := 0
				if *exc.Mechanism.Handled {
					v = 1
				}
				excHandled = &v
			}

			var stacktrace *string
			if exc.Stacktrace != nil {
				s := string(exc.Stacktrace)
				stacktrace = &s
			}

			_, err = tx.Exec(
				`INSERT INTO sentry_exceptions
					(id, error_event_id, position, exception_type, exception_value, mechanism_type, handled, stacktrace)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				excID, id, i,
				nullIfEmpty(exc.Type),
				nullIfEmpty(exc.Value),
				mechanismType(exc.Mechanism),
				excHandled,
				stacktrace,
			)
			if err != nil {
				return "", fmt.Errorf("insert sentry_exceptions: %w", err)
			}
		}
	}

	// Insert breadcrumbs.
	if ev.Breadcrumbs != nil {
		for _, bc := range ev.Breadcrumbs.Values {
			bcID := event.GenerateUUID()
			bcTS := parseTimestamp(bc.Timestamp)

			var data *string
			if bc.Data != nil {
				s := string(bc.Data)
				data = &s
			}

			_, err = tx.Exec(
				`INSERT INTO sentry_breadcrumbs
					(id, error_event_id, bc_type, category, level, message, bc_timestamp, data)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				bcID, id,
				nullIfEmpty(bc.Type),
				nullIfEmpty(bc.Category),
				nullIfEmpty(bc.Level),
				nullIfEmpty(bc.Message),
				bcTS,
				data,
			)
			if err != nil {
				return "", fmt.Errorf("insert sentry_breadcrumbs: %w", err)
			}
		}
	}

	return id, tx.Commit()
}

func mechanismType(m *Mechanism) *string {
	if m == nil || m.Type == "" {
		return nil
	}
	return &m.Type
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parseTimestamp converts a Sentry timestamp (float64 epoch seconds or ISO string) to an RFC3339 string.
func parseTimestamp(ts json.Number) *string {
	if ts == "" {
		return nil
	}

	// Try as float (epoch seconds).
	if f, err := strconv.ParseFloat(string(ts), 64); err == nil {
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		t := time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
		return &t
	}

	// Try as string (ISO 8601).
	s := string(ts)
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return &s
	}

	return &s
}
