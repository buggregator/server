package sentry

import (
	"database/sql"
	"net/http"
	"strconv"
)

func handleServiceMap(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		windowMinutes := 60
		if v := r.URL.Query().Get("window"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10080 {
				windowMinutes = n
			}
		}

		// Build edges from spans with peer_address.
		edgeRows, err := db.Query(
			`SELECT
				COALESCE(service_name, 'unknown') as source,
				peer_address as target,
				peer_type as op_type,
				COUNT(*) as request_count,
				SUM(CASE WHEN is_error = 1 THEN 1 ELSE 0 END) as error_count,
				CAST(AVG(duration_ms) AS INTEGER) as avg_duration_ms
			FROM sentry_spans
			WHERE peer_address IS NOT NULL
				AND start_ts >= datetime('now', ? || ' minutes')
			GROUP BY source, target, op_type
			ORDER BY request_count DESC`,
			strconv.Itoa(-windowMinutes),
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
		var exceptions, traces, logs int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_error_events`).Scan(&exceptions)
		db.QueryRow(`SELECT COUNT(*) FROM sentry_traces`).Scan(&traces)
		db.QueryRow(`SELECT COUNT(*) FROM sentry_logs`).Scan(&logs)

		apiJSON(w, map[string]any{
			"exceptions": exceptions,
			"traces":     traces,
			"logs":       logs,
		})
	}
}
