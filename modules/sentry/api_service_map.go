package sentry

import (
	"database/sql"
	"net/http"
	"time"
)

// now is the clock of the service-map window; tests pin it.
var now = time.Now

func labelForNode(address, opType string) string {
	switch opType {
	case "db":
		return "DB: " + address
	case "cache":
		return "Cache: " + address
	case "queue":
		return "Queue: " + address
	default:
		return address
	}
}

func handleCounts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var exceptions, traces, logs, metrics int
		// The counters follow the same project and period as the lists they label.
		// They used to count the whole database, so the project selected in the UI
		// had no effect on the numbers next to the tabs.
		var errConds, txnConds, logConds, metricConds []string
		var errArgs, txnArgs, logArgs, metricArgs []any
		errConds, errArgs = appendTimeConditions(errConds, errArgs, r, "received_at", false)
		txnConds, txnArgs = appendTimeConditions(txnConds, txnArgs, r, "start_ts", true)
		logConds, logArgs = appendTimeConditions(logConds, logArgs, r, "log_ts", true)
		metricConds, metricArgs = appendTimeConditions(metricConds, metricArgs, r, "metric_ts", true)
		errConds, errArgs = appendProject(errConds, errArgs, r, "project_id")
		txnConds, txnArgs = appendProject(txnConds, txnArgs, r, "project_id")
		logConds, logArgs = appendProject(logConds, logArgs, r, "project_id")
		metricConds, metricArgs = appendProject(metricConds, metricArgs, r, "project_id")

		db.QueryRow(`SELECT COUNT(*) FROM sentry_error_events`+whereOf(errConds), errArgs...).Scan(&exceptions)
		// Counted over sentry_transactions rather than sentry_traces: that is what
		// the traces list shows (one row per trace), and it is the table carrying
		// a timestamp to filter on.
		db.QueryRow(`SELECT COUNT(DISTINCT trace_id) FROM sentry_transactions`+whereOf(txnConds), txnArgs...).Scan(&traces)
		db.QueryRow(`SELECT COUNT(*) FROM sentry_logs`+whereOf(logConds), logArgs...).Scan(&logs)
		db.QueryRow(`SELECT COUNT(*) FROM sentry_metrics`+whereOf(metricConds), metricArgs...).Scan(&metrics)

		apiJSON(w, map[string]any{
			"exceptions": exceptions,
			"traces":     traces,
			"logs":       logs,
			"metrics":    metrics,
		})
	}
}
