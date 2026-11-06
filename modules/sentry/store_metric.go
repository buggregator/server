package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/buggregator/go-buggregator/internal/event"
)

// storeMetrics batch inserts Sentry trace metric items.
func storeMetrics(db *sql.DB, metrics []MetricRecord) error {
	if len(metrics) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO sentry_metrics
			(id, trace_id, span_id, name, type, value, unit, attributes, metric_ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare sentry_metrics: %w", err)
	}
	defer stmt.Close()

	for _, m := range metrics {
		if m.Name == "" {
			continue
		}

		var attrs *string
		if m.Attributes != nil {
			b, _ := json.Marshal(m.Attributes)
			s := string(b)
			attrs = &s
		}

		_, err = stmt.Exec(
			event.GenerateUUID(),
			nullIfEmpty(m.TraceID),
			nullIfEmpty(m.SpanID),
			m.Name,
			m.Type,
			m.Value,
			nullIfEmpty(m.Unit),
			attrs,
			parseLogTimestamp(m.Timestamp),
		)
		if err != nil {
			return fmt.Errorf("insert sentry_metrics: %w", err)
		}
	}

	return tx.Commit()
}
