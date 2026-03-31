package sentry

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func handleTracesList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pagination(r, 50)

		countQuery := `SELECT COUNT(*) FROM sentry_transactions`
		var total int
		db.QueryRow(countQuery).Scan(&total)

		rows, err := db.Query(
			`SELECT t.trace_id, t.id, t.transaction_name, t.op, t.status, t.duration_ms,
				tr.span_count, tr.error_count, t.start_ts
			FROM sentry_transactions t
			JOIN sentry_traces tr ON tr.trace_id = t.trace_id
			ORDER BY t.start_ts DESC
			LIMIT ? OFFSET ?`, limit, offset,
		)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Collect rows first to release the DB connection (SQLite single-writer).
		type traceRow struct {
			traceID, txnID, txnName    string
			op, status                 sql.NullString
			durationMS                 sql.NullInt64
			spanCount, errorCount      int
			startTS                    sql.NullString
		}
		var traceRows []traceRow
		for rows.Next() {
			var tr traceRow
			if err := rows.Scan(&tr.traceID, &tr.txnID, &tr.txnName, &tr.op, &tr.status, &tr.durationMS, &tr.spanCount, &tr.errorCount, &tr.startTS); err != nil {
				rows.Close()
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			traceRows = append(traceRows, tr)
		}
		rows.Close()

		// Now build response with preview spans (connection is free).
		var data []map[string]any
		for _, tr := range traceRows {
			previewSpans := loadPreviewSpans(db, tr.traceID, tr.startTS, 3)

			data = append(data, map[string]any{
				"trace_id":         tr.traceID,
				"transaction_id":   tr.txnID,
				"transaction_name": tr.txnName,
				"op":               scanNullString(tr.op),
				"status":           scanNullString(tr.status),
				"duration_ms":      scanNullInt(tr.durationMS),
				"span_count":       tr.spanCount,
				"error_count":      tr.errorCount,
				"received_at":      scanNullString(tr.startTS),
				"preview_spans":    previewSpans,
			})
		}

		if data == nil {
			data = []map[string]any{}
		}

		apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
	}
}

func handleTraceDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := r.PathValue("traceId")

		// Load transaction.
		var (
			txnID, txnName string
			op, status     sql.NullString
			startTS, endTS sql.NullString
			durationMS     sql.NullInt64
		)
		err := db.QueryRow(
			`SELECT id, transaction_name, op, status, start_ts, end_ts, duration_ms
			FROM sentry_transactions WHERE trace_id = ? ORDER BY start_ts LIMIT 1`, traceID,
		).Scan(&txnID, &txnName, &op, &status, &startTS, &endTS, &durationMS)
		if err == sql.ErrNoRows {
			apiError(w, "trace not found", http.StatusNotFound)
			return
		}
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Load all spans.
		spans := loadAllSpans(db, traceID, startTS)

		// Load related errors.
		relatedErrors := loadRelatedErrors(db, traceID)

		// Count related logs.
		var relatedLogCount int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_logs WHERE trace_id = ?`, traceID).Scan(&relatedLogCount)

		result := map[string]any{
			"trace_id": traceID,
			"transaction": map[string]any{
				"id":               txnID,
				"transaction_name": txnName,
				"op":               scanNullString(op),
				"status":           scanNullString(status),
				"start_ts":         scanNullString(startTS),
				"end_ts":           scanNullString(endTS),
				"duration_ms":      scanNullInt(durationMS),
			},
			"spans":             spans,
			"related_errors":    relatedErrors,
			"related_log_count": relatedLogCount,
		}

		apiJSON(w, result)
	}
}

func loadAllSpans(db *sql.DB, traceID string, txnStartTS sql.NullString) []map[string]any {
	rows, err := db.Query(
		`SELECT span_id, parent_span_id, op, description, status, start_ts, duration_ms,
			peer_type, peer_address, is_error
		FROM sentry_spans WHERE trace_id = ? ORDER BY start_ts`, traceID,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			spanID                            string
			parentSpanID, op, description     sql.NullString
			status                            sql.NullString
			sStartTS                          sql.NullString
			durationMS                        sql.NullInt64
			peerType, peerAddress             sql.NullString
			isError                           int
		)
		if err := rows.Scan(&spanID, &parentSpanID, &op, &description, &status, &sStartTS, &durationMS, &peerType, &peerAddress, &isError); err != nil {
			continue
		}

		startOffsetMS := computeOffsetMS(txnStartTS, sStartTS)

		result = append(result, map[string]any{
			"span_id":         spanID,
			"parent_span_id":  scanNullString(parentSpanID),
			"op":              scanNullString(op),
			"description":     scanNullString(description),
			"status":          scanNullString(status),
			"start_offset_ms": startOffsetMS,
			"duration_ms":     scanNullInt(durationMS),
			"peer_type":       scanNullString(peerType),
			"peer_address":    scanNullString(peerAddress),
			"is_error":        isError != 0,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result
}

func loadRelatedErrors(db *sql.DB, traceID string) []map[string]any {
	// Use a single query to avoid nested connection issues with SQLite.
	rows, err := db.Query(
		`SELECT e.event_id, se.exception_type, e.received_at
		FROM sentry_error_events e
		LEFT JOIN sentry_exceptions se ON se.error_event_id = e.id AND se.position = 0
		WHERE e.trace_id = ? ORDER BY e.received_at DESC`, traceID,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			id         string
			excType    sql.NullString
			receivedAt string
		)
		if err := rows.Scan(&id, &excType, &receivedAt); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"id":             id,
			"exception_type": scanNullString(excType),
			"received_at":    receivedAt,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result
}

// parseRFC3339 tries multiple time formats for stored timestamps.
func parseRFC3339(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}
