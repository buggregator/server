package sentry

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
)

// now is the clock of the service-map window; tests pin it.
var now = time.Now

func handleServiceMap(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		windowMinutes := 60
		if v := r.URL.Query().Get("window"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10080 {
				windowMinutes = n
			}
		}

		// Build edges from spans with peer_address. An outgoing call to a
		// service we also receive spans from is drawn to that service, not to
		// its address: the callee's span (span v2) or transaction (contexts.trace)
		// names the call's span as its parent. Only another service counts: a
		// child of the call in the caller itself (a retry, a sub-span) is not
		// the callee. Databases, caches and queues keep their address.
		pc, pa := projectClause(r, "s.project_id")
		edgeRows, err := db.Query(
			`WITH calls AS (
				SELECT
					COALESCE(s.service_name, 'unknown') AS source,
					s.peer_address, s.peer_type, s.is_error, s.duration_ms, s.span_id, s.trace_id,
					CASE WHEN s.peer_type IN ('db', 'cache', 'queue') THEN NULL ELSE COALESCE(
						(SELECT d.service_name FROM sentry_spans d
							WHERE d.trace_id = s.trace_id AND d.parent_span_id = s.span_id
								AND d.service_name <> COALESCE(s.service_name, '') LIMIT 1),
						(SELECT sp.service_name FROM sentry_transactions t
							JOIN sentry_spans sp ON sp.transaction_id = t.id
							WHERE t.trace_id = s.trace_id
								AND json_extract(t.payload, '$.contexts.trace.parent_span_id') = s.span_id
								AND sp.service_name <> COALESCE(s.service_name, '') LIMIT 1)
					) END AS callee
				FROM sentry_spans s
				WHERE s.peer_address IS NOT NULL
					AND s.start_ts >= ?`+pc+`
			)
			SELECT
				source,
				COALESCE(callee, peer_address) AS target,
				peer_type AS op_type,
				COUNT(*) AS request_count,
				SUM(CASE WHEN is_error = 1 THEN 1 ELSE 0 END) AS error_count,
				CAST(AVG(duration_ms) AS INTEGER) AS avg_duration_ms
			FROM calls
			GROUP BY source, target, op_type
			ORDER BY request_count DESC`,
			append([]any{windowStart(now(), windowMinutes)}, pa...)...,
		)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer edgeRows.Close()

		nodeMap := make(map[string]*serviceNode)
		var edges []map[string]any

		for edgeRows.Next() {
			var (
				source, target, opType      string
				requestCount, errorCount    int
				avgDuration                 sql.NullInt64
			)
			if err := edgeRows.Scan(&source, &target, &opType, &requestCount, &errorCount, &avgDuration); err != nil {
				continue
			}

			edges = append(edges, map[string]any{
				"source":          source,
				"target":          target,
				"op_type":         opType,
				"request_count":   requestCount,
				"error_count":     errorCount,
				"avg_duration_ms": scanNullInt(avgDuration),
			})

			// Accumulate node stats.
			if _, ok := nodeMap[source]; !ok {
				nodeMap[source] = &serviceNode{ID: source, Label: source}
			}
			nodeMap[source].RequestCount += requestCount
			nodeMap[source].ErrorCount += errorCount

			if _, ok := nodeMap[target]; !ok {
				nodeMap[target] = &serviceNode{ID: target, Label: labelForNode(target, opType)}
			}
			nodeMap[target].RequestCount += requestCount
			nodeMap[target].ErrorCount += errorCount
			if avgDuration.Valid {
				nodeMap[target].TotalDuration += avgDuration.Int64 * int64(requestCount)
				nodeMap[target].DurationSamples += requestCount
			}
		}

		var nodes []map[string]any
		for _, n := range nodeMap {
			var avgDur *int
			if n.DurationSamples > 0 {
				v := int(n.TotalDuration / int64(n.DurationSamples))
				avgDur = &v
			}
			nodes = append(nodes, map[string]any{
				"id":              n.ID,
				"label":           n.Label,
				"request_count":   n.RequestCount,
				"error_count":     n.ErrorCount,
				"avg_duration_ms": avgDur,
			})
		}

		if nodes == nil {
			nodes = []map[string]any{}
		}
		if edges == nil {
			edges = []map[string]any{}
		}

		apiJSON(w, map[string]any{
			"nodes":          nodes,
			"edges":          edges,
			"window_minutes": windowMinutes,
		})
	}
}

// windowStart is the lower bound of the window in the format of span start_ts
// (ISO, "2026-11-16T11:00:00.5Z"). A datetime() bound ("2026-11-16 11:00:00")
// compared as text let every span of the same UTC date pass: 'T' > ' '.
func windowStart(t time.Time, minutes int) string {
	return t.UTC().Add(-time.Duration(minutes) * time.Minute).Format("2006-01-02T15:04:05")
}

type serviceNode struct {
	ID              string
	Label           string
	RequestCount    int
	ErrorCount      int
	TotalDuration   int64
	DurationSamples int
}

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
