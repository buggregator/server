package sentry

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// handleFeedbackList lists user feedback, newest first.
//
//	GET /api/sentry/feedback?event_id=…&project=…&from=…&to=…&window=…&page=…&limit=…
//
// event_id narrows to the feedback on one error (dashes and case ignored).
func handleFeedbackList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pagination(r, 100)

		var conditions []string
		var args []any
		if v := normalizeEventID(r.URL.Query().Get("event_id")); v != "" {
			conditions = append(conditions, "lower(replace(event_id, '-', '')) = ?")
			args = append(args, v)
		}
		conditions, args = appendTimeConditions(conditions, args, r, "received_at", false)
		conditions, args = appendProject(conditions, args, r, "project_id")
		where := whereOf(conditions)

		var total int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_feedback`+where, args...).Scan(&total)

		rows, err := db.Query(
			`SELECT id, project_id, event_id, feedback_id, name, email, message, url, source, payload, `+tsUTC("received_at")+`
			FROM sentry_feedback`+where+` ORDER BY received_at DESC, rowid DESC LIMIT ? OFFSET ?`,
			append(args, limit, offset)...,
		)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		data := []map[string]any{}
		for rows.Next() {
			var id, project, eventID, feedbackID, name, email, message, url, source, receivedAt string
			var payload sql.NullString
			if err := rows.Scan(&id, &project, &eventID, &feedbackID, &name, &email, &message, &url, &source, &payload, &receivedAt); err != nil {
				continue
			}
			item := map[string]any{
				"id":          id,
				"project":     project,
				"event_id":    eventID,
				"feedback_id": feedbackID,
				"name":        name,
				"email":       email,
				"message":     message,
				"url":         url,
				"source":      source,
				"received_at": receivedAt,
				"payload":     nil,
			}
			if payload.Valid && json.Valid([]byte(payload.String)) {
				item["payload"] = json.RawMessage(payload.String)
			}
			data = append(data, item)
		}

		apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
	}
}
