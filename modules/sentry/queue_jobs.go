package sentry

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
)

// queueJobOp is the op of a transaction that is one attempt of a queue job
// (sentry-laravel QueueIntegration; otlp.go maps OTel messaging spans to it).
const queueJobOp = "queue.process"

// queueJobRow is one attempt of a queue job, read from its transaction.
type queueJobRow struct {
	TransactionID, TraceID, Project string
	Job, Status                     string
	StartTS, EndTS                  *string
	DurationMS                      *int
	// attr reads a messaging.* attribute of the transaction.
	attr func(key string) any
}

// storeQueueJob writes the attempt into sentry_queue_jobs, in the
// transaction's own database transaction.
func storeQueueJob(dbTx *sql.Tx, j queueJobRow) error {
	str := func(key string) string {
		switch v := j.attr(key).(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
		return ""
	}
	num := func(key string) (float64, bool) {
		switch v := j.attr(key).(type) {
		case float64:
			return v, true
		case string:
			f, err := strconv.ParseFloat(v, 64)
			return f, err == nil
		}
		return 0, false
	}

	attempt := 1
	if n, ok := num("messaging.message.retry.count"); ok && n >= 0 {
		attempt = int(n) + 1
	}
	var latencyMS *int
	if l, ok := num("messaging.message.receive.latency"); ok && l >= 0 {
		// sentry-laravel sends seconds (microtime(true) - pushedAt), the
		// Sentry spec and other SDKs milliseconds.
		if str("messaging.system") == "laravel" {
			l *= 1000
		}
		ms := int(math.Round(l))
		latencyMS = &ms
	}

	_, err := dbTx.Exec(
		`INSERT OR IGNORE INTO sentry_queue_jobs
			(transaction_id, message_id, trace_id, project_id, job, queue, connection, attempt, status, duration_ms, latency_ms, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.TransactionID, nullIfEmpty(str("messaging.message.id")), j.TraceID, j.Project,
		j.Job, str("messaging.destination.name"), str("messaging.destination.connection"),
		attempt, j.Status, j.DurationMS, latencyMS, j.StartTS, j.EndTS,
	)
	if err != nil {
		return fmt.Errorf("insert sentry_queue_jobs: %w", err)
	}
	return nil
}
