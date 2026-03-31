package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func handleExceptionsList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		grouped := q.Get("grouped") == "true"

		if grouped {
			handleExceptionsGrouped(db, w, r)
		} else {
			handleExceptionsChronological(db, w, r)
		}
	}
}

func handleExceptionsGrouped(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r, 50)
	q := r.URL.Query()

	query := `SELECT
		g.fingerprint,
		se.exception_type,
		se.exception_value,
		g.count,
		g.first_seen,
		g.last_seen,
		g.level,
		g.handled,
		g.sample_event_id
	FROM (
		SELECT
			e.fingerprint,
			COUNT(*) as count,
			MIN(e.received_at) as first_seen,
			MAX(e.received_at) as last_seen,
			MAX(e.level) as level,
			MIN(e.handled) as handled,
			MAX(e.event_id) as sample_event_id
		FROM sentry_error_events e`

	countQuery := `SELECT COUNT(DISTINCT e.fingerprint) FROM sentry_error_events e`

	var conditions []string
	var args []any

	if v := q.Get("level"); v != "" {
		conditions = append(conditions, "e.level = ?")
		args = append(args, v)
	}
	if v := q.Get("handled"); v != "" {
		if v == "true" {
			conditions = append(conditions, "e.handled = 1")
		} else {
			conditions = append(conditions, "e.handled = 0")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	query += where + ` GROUP BY e.fingerprint
	) g
	LEFT JOIN sentry_error_events sample_e ON sample_e.event_id = g.sample_event_id
	LEFT JOIN sentry_exceptions se ON se.error_event_id = sample_e.id AND se.position = 0
	ORDER BY g.count DESC LIMIT ? OFFSET ?`
	countQuery += where

	// Get total.
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
			fingerprint, sampleEventID string
			excType, excValue          sql.NullString
			level                      sql.NullString
			handled                    sql.NullBool
			count                      int
			firstSeen, lastSeen        string
		)
		if err := rows.Scan(&fingerprint, &excType, &excValue, &count, &firstSeen, &lastSeen, &level, &handled, &sampleEventID); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = append(data, map[string]any{
			"fingerprint":     fingerprint,
			"exception_type":  scanNullString(excType),
			"exception_value": scanNullString(excValue),
			"count":           count,
			"first_seen":      firstSeen,
			"last_seen":       lastSeen,
			"level":           scanNullString(level),
			"handled":         scanNullBool(handled),
			"sample_event_id": sampleEventID,
		})
	}

	if data == nil {
		data = []map[string]any{}
	}

	apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
}

func handleExceptionsChronological(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r, 50)
	q := r.URL.Query()

	query := `SELECT
		e.id, e.event_id, e.fingerprint,
		(SELECT COUNT(*) FROM sentry_error_events e2 WHERE e2.fingerprint = e.fingerprint) as occurrence_count,
		(SELECT se.exception_type FROM sentry_exceptions se WHERE se.error_event_id = e.id AND se.position = 0 LIMIT 1) as exception_type,
		(SELECT se.exception_value FROM sentry_exceptions se WHERE se.error_event_id = e.id AND se.position = 0 LIMIT 1) as exception_value,
		e.level, e.handled, e."transaction", e.received_at, e.trace_id
	FROM sentry_error_events e`

	countQuery := `SELECT COUNT(*) FROM sentry_error_events e`

	var conditions []string
	var args []any

	if v := q.Get("level"); v != "" {
		conditions = append(conditions, "e.level = ?")
		args = append(args, v)
	}
	if v := q.Get("handled"); v != "" {
		if v == "true" {
			conditions = append(conditions, "e.handled = 1")
		} else {
			conditions = append(conditions, "e.handled = 0")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	query += where + " ORDER BY e.received_at DESC LIMIT ? OFFSET ?"
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
			id, eventID, fingerprint string
			occurrenceCount          int
			excType, excValue        sql.NullString
			level                    string
			handled                  sql.NullBool
			txn                      sql.NullString
			receivedAt               string
			traceID                  sql.NullString
		)
		if err := rows.Scan(&id, &eventID, &fingerprint, &occurrenceCount, &excType, &excValue, &level, &handled, &txn, &receivedAt, &traceID); err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = append(data, map[string]any{
			"id":               id,
			"event_id":         eventID,
			"fingerprint":      fingerprint,
			"occurrence_count": occurrenceCount,
			"exception_type":   scanNullString(excType),
			"exception_value":  scanNullString(excValue),
			"level":            level,
			"handled":          scanNullBool(handled),
			"transaction":      scanNullString(txn),
			"received_at":      receivedAt,
			"trace_id":         scanNullString(traceID),
		})
	}

	if data == nil {
		data = []map[string]any{}
	}

	apiJSON(w, listResponse(data, total, currentPage(offset, limit), limit))
}

func handleExceptionDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		// Load error event.
		var (
			eventID, fingerprint, level, receivedAt, payloadStr string
			handled                                             sql.NullBool
			platform, environment, serverName, txn, release     sql.NullString
			traceID, spanID                                     sql.NullString
			eventTS                                             sql.NullString
		)
		err := db.QueryRow(
			`SELECT event_id, fingerprint, level, handled, platform, environment, server_name,
				"transaction", release, trace_id, span_id, received_at, event_ts, payload
			FROM sentry_error_events WHERE id = ?`, id,
		).Scan(&eventID, &fingerprint, &level, &handled, &platform, &environment,
			&serverName, &txn, &release, &traceID, &spanID, &receivedAt, &eventTS, &payloadStr)
		if err == sql.ErrNoRows {
			apiError(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Load exceptions.
		exceptions := loadExceptions(db, id)

		// Load breadcrumbs.
		breadcrumbs := loadBreadcrumbs(db, id)

		// Load trace summary if trace_id exists.
		var traceSummary any
		if traceID.Valid && traceID.String != "" {
			traceSummary = loadTraceSummary(db, traceID.String)
		}

		// Parse payload.
		var payload json.RawMessage
		if payloadStr != "" {
			payload = json.RawMessage(payloadStr)
		}

		result := map[string]any{
			"id":            id,
			"event_id":      eventID,
			"fingerprint":   fingerprint,
			"level":         level,
			"handled":       scanNullBool(handled),
			"platform":      scanNullString(platform),
			"environment":   scanNullString(environment),
			"server_name":   scanNullString(serverName),
			"transaction":   scanNullString(txn),
			"release":       scanNullString(release),
			"trace_id":      scanNullString(traceID),
			"span_id":       scanNullString(spanID),
			"received_at":   receivedAt,
			"event_ts":      scanNullString(eventTS),
			"exceptions":    exceptions,
			"breadcrumbs":   breadcrumbs,
			"trace_summary": traceSummary,
			"payload":       payload,
		}

		apiJSON(w, result)
	}
}

func loadExceptions(db *sql.DB, errorEventID string) []map[string]any {
	rows, err := db.Query(
		`SELECT position, exception_type, exception_value, mechanism_type, handled, stacktrace
		FROM sentry_exceptions WHERE error_event_id = ? ORDER BY position`, errorEventID,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			position       int
			excType        sql.NullString
			excValue       sql.NullString
			mechType       sql.NullString
			handled        sql.NullBool
			stacktraceStr  sql.NullString
		)
		if err := rows.Scan(&position, &excType, &excValue, &mechType, &handled, &stacktraceStr); err != nil {
			continue
		}

		var stacktrace any
		if stacktraceStr.Valid {
			json.Unmarshal([]byte(stacktraceStr.String), &stacktrace)
		}

		result = append(result, map[string]any{
			"position":       position,
			"exception_type": scanNullString(excType),
			"exception_value": scanNullString(excValue),
			"mechanism_type": scanNullString(mechType),
			"handled":        scanNullBool(handled),
			"stacktrace":     stacktrace,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result
}

func loadBreadcrumbs(db *sql.DB, errorEventID string) []map[string]any {
	rows, err := db.Query(
		`SELECT bc_type, category, level, message, bc_timestamp, data
		FROM sentry_breadcrumbs WHERE error_event_id = ? ORDER BY bc_timestamp`, errorEventID,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			bcType, category, level, message sql.NullString
			bcTimestamp                       sql.NullString
			dataStr                          sql.NullString
		)
		if err := rows.Scan(&bcType, &category, &level, &message, &bcTimestamp, &dataStr); err != nil {
			continue
		}

		var data any
		if dataStr.Valid {
			json.Unmarshal([]byte(dataStr.String), &data)
		}

		result = append(result, map[string]any{
			"type":      scanNullString(bcType),
			"category":  scanNullString(category),
			"level":     scanNullString(level),
			"message":   scanNullString(message),
			"timestamp": scanNullString(bcTimestamp),
			"data":      data,
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result
}

func loadTraceSummary(db *sql.DB, traceID string) map[string]any {
	// Load transaction for this trace.
	var (
		txnName    sql.NullString
		op         sql.NullString
		durationMS sql.NullInt64
		startTS    sql.NullString
	)

	err := db.QueryRow(
		`SELECT transaction_name, op, duration_ms, start_ts
		FROM sentry_transactions WHERE trace_id = ? ORDER BY start_ts LIMIT 1`, traceID,
	).Scan(&txnName, &op, &durationMS, &startTS)
	if err != nil {
		return nil
	}

	// Get span count.
	var spanCount int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_spans WHERE trace_id = ?`, traceID).Scan(&spanCount)

	// Load top 3 spans by duration for preview.
	previewSpans := loadPreviewSpans(db, traceID, startTS, 3)

	return map[string]any{
		"trace_id":         traceID,
		"transaction_name": scanNullString(txnName),
		"op":               scanNullString(op),
		"duration_ms":      scanNullInt(durationMS),
		"span_count":       spanCount,
		"preview_spans":    previewSpans,
	}
}

// loadPreviewSpans loads the top N spans by duration for mini-waterfall preview.
func loadPreviewSpans(db *sql.DB, traceID string, txnStartTS sql.NullString, limit int) []map[string]any {
	rows, err := db.Query(
		fmt.Sprintf(`SELECT span_id, op, description, start_ts, duration_ms, is_error, peer_type
		FROM sentry_spans WHERE trace_id = ? ORDER BY duration_ms DESC LIMIT %d`, limit), traceID,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			spanID, op  string
			description sql.NullString
			sStartTS    sql.NullString
			durationMS  sql.NullInt64
			isError     int
			peerType    sql.NullString
		)
		if err := rows.Scan(&spanID, &op, &description, &sStartTS, &durationMS, &isError, &peerType); err != nil {
			continue
		}

		startOffsetMS := computeOffsetMS(txnStartTS, sStartTS)

		result = append(result, map[string]any{
			"span_id":         spanID,
			"op":              op,
			"description":     scanNullString(description),
			"start_offset_ms": startOffsetMS,
			"duration_ms":     scanNullInt(durationMS),
			"is_error":        isError != 0,
			"peer_type":       scanNullString(peerType),
		})
	}

	if result == nil {
		result = []map[string]any{}
	}
	return result
}

// computeOffsetMS calculates the offset in ms between two RFC3339 timestamps.
func computeOffsetMS(baseTSStr, spanTSStr sql.NullString) int {
	if !baseTSStr.Valid || !spanTSStr.Valid {
		return 0
	}
	baseTime, err1 := parseRFC3339(baseTSStr.String)
	spanTime, err2 := parseRFC3339(spanTSStr.String)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(spanTime.Sub(baseTime).Milliseconds())
}
