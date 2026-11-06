package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

// handleMetricsList lists stored trace metrics, newest first.
// Filters: name, type (counter, gauge, distribution), trace_id.
func handleMetricsList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pagination(r, 100)
		q := r.URL.Query()

		query := `SELECT id, trace_id, span_id, name, type, value, unit, attributes, metric_ts
			FROM sentry_metrics`
		countQuery := `SELECT COUNT(*) FROM sentry_metrics`

		var conditions []string
		var args []any

		for _, filter := range []string{"name", "type", "trace_id"} {
			if v := q.Get(filter); v != "" {
				conditions = append(conditions, filter+" = ?")
				args = append(args, v)
			}
		}

		where := ""
		if len(conditions) > 0 {
			where = " WHERE " + strings.Join(conditions, " AND ")
		}

		query += where + " ORDER BY metric_ts DESC LIMIT ? OFFSET ?"
		countQuery += where

		var total int
		db.QueryRow(countQuery, args...).Scan(&total)

		queryArgs := append(args, limit, offset)
		rows, err := db.Query(query, queryArgs...)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var data []map[string]any
		for rows.Next() {
			var (
				id, name, metricType string
				value                float64
				traceID, spanID      sql.NullString
				unit                 sql.NullString
				attrsStr             sql.NullString
				metricTS             sql.NullString
			)
			if err := rows.Scan(&id, &traceID, &spanID, &name, &metricType, &value, &unit, &attrsStr, &metricTS); err != nil {
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var attrs any
			if attrsStr.Valid {
				json.Unmarshal([]byte(attrsStr.String), &attrs)
			}

			data = append(data, map[string]any{
				"id":         id,
				"name":       name,
				"type":       metricType,
				"value":      value,
				"unit":       scanNullString(unit),
				"trace_id":   scanNullString(traceID),
				"span_id":    scanNullString(spanID),
				"metric_ts":  scanNullString(metricTS),
				"attributes": attrs,
			})
		}

		if data == nil {
			data = []map[string]any{}
		}

		apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
	}
}
