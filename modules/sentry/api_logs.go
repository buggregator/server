package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func handleLogsList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pagination(r, 100)
		q := r.URL.Query()

		// LEFT JOIN sentry_traces to check if the referenced trace actually exists.
		query := `SELECT l.id, l.trace_id, l.span_id, l.level, l.severity_number, l.body, l.attributes, l.log_ts,
			CASE WHEN t.trace_id IS NOT NULL THEN 1 ELSE 0 END AS trace_exists
			FROM sentry_logs l
			LEFT JOIN sentry_traces t ON t.trace_id = l.trace_id`
		countQuery := `SELECT COUNT(*) FROM sentry_logs l`

		var conditions []string
		var args []any

		if v := q.Get("level"); v != "" {
			conditions = append(conditions, "l.level = ?")
			args = append(args, v)
		}
		if v := q.Get("trace_id"); v != "" {
			conditions = append(conditions, "l.trace_id = ?")
			args = append(args, v)
		}

		where := ""
		if len(conditions) > 0 {
			where = " WHERE " + strings.Join(conditions, " AND ")
		}

		query += where + " ORDER BY l.log_ts DESC LIMIT ? OFFSET ?"
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
				id, level, body string
				traceID, spanID sql.NullString
				sevNum          sql.NullInt64
				attrsStr        sql.NullString
				logTS           sql.NullString
				traceExists     int
			)
			if err := rows.Scan(&id, &traceID, &spanID, &level, &sevNum, &body, &attrsStr, &logTS, &traceExists); err != nil {
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var attrs any
			if attrsStr.Valid {
				json.Unmarshal([]byte(attrsStr.String), &attrs)
			}

			data = append(data, map[string]any{
				"id":              id,
				"level":           level,
				"severity_number": scanNullInt(sevNum),
				"body":            body,
				"trace_id":        scanNullString(traceID),
				"span_id":         scanNullString(spanID),
				"log_ts":          scanNullString(logTS),
				"attributes":      attrs,
				"trace_exists":    traceExists != 0,
			})
		}

		if data == nil {
			data = []map[string]any{}
		}

		apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
	}
}
