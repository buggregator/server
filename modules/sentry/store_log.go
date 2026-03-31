package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// storeLogs batch inserts Sentry native log items.
func storeLogs(db *sql.DB, logs []LogRecord) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO sentry_logs
			(id, trace_id, span_id, level, severity_number, body, attributes, log_ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare sentry_logs: %w", err)
	}
	defer stmt.Close()

	for _, log := range logs {
		id := event.GenerateUUID()

		var attrs *string
		if log.Attributes != nil {
			b, _ := json.Marshal(log.Attributes)
			s := string(b)
			attrs = &s
		}

		var sevNum *int
		if log.SeverityNumber > 0 {
			v := log.SeverityNumber
			sevNum = &v
		}

		logTS := parseLogTimestamp(log.Timestamp)

		_, err = stmt.Exec(
			id,
			nullIfEmpty(log.TraceID),
			nullIfEmpty(log.SpanID),
			log.Level,
			sevNum,
			log.Body,
			attrs,
			logTS,
		)
		if err != nil {
			return fmt.Errorf("insert sentry_logs: %w", err)
		}
	}

	return tx.Commit()
}

// parseLogTimestamp handles both float epoch seconds and string ISO timestamps.
func parseLogTimestamp(ts json.Number) *string {
	if ts == "" {
		return nil
	}

	if f, err := strconv.ParseFloat(string(ts), 64); err == nil {
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		t := time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
		return &t
	}

	s := string(ts)
	return &s
}
