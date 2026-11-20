package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// rootOrder sorts a trace's transactions (alias r) root first: the one with
// no parent span (Sentry: contexts.trace.parent_span_id, a segment:
// parent_span_id), then the earliest.
const rootOrder = `CASE WHEN json_valid(r.payload) AND COALESCE(
		NULLIF(json_extract(r.payload, '$.contexts.trace.parent_span_id'), ''),
		NULLIF(json_extract(r.payload, '$.parent_span_id'), '')) IS NOT NULL THEN 1 ELSE 0 END,
	r.start_ts, r.id`

func handleTracesList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pagination(r, 50)
		source, ok := sourceParam(r)
		if !ok {
			apiError(w, "source must be sentry or otel", http.StatusBadRequest)
			return
		}

		// One row per trace: its root transaction, the one with no parent span
		// (Sentry: contexts.trace.parent_span_id, a segment: parent_span_id),
		// else the earliest. A trace through three services is one row.
		rootClause, rootArgs := projectClause(r, "r.project_id")
		conditions := []string{`t.id = (SELECT r.id FROM sentry_transactions r
			WHERE r.trace_id = t.trace_id` + rootClause + `
			ORDER BY ` + rootOrder + `
			LIMIT 1)`}
		args := append([]any{}, rootArgs...)

		// The filters match a trace when any of its transactions matches.
		// Period: start_ts of a transaction is in ISO form, index idx_sentry_txn_start.
		var filters []string
		var filterArgs []any
		filters, filterArgs = appendTimeConditions(filters, filterArgs, r, "f.start_ts", true)
		filters, filterArgs = appendProject(filters, filterArgs, r, "f.project_id")
		if v := r.URL.Query().Get("environment"); v != "" {
			filters = append(filters, "f.environment = ?")
			filterArgs = append(filterArgs, v)
		}
		if v := r.URL.Query().Get("release"); v != "" {
			filters = append(filters, "f.release = ?")
			filterArgs = append(filterArgs, v)
		}
		if len(filters) > 0 {
			conditions = append(conditions, `EXISTS (SELECT 1 FROM sentry_transactions f
				WHERE f.trace_id = t.trace_id AND `+strings.Join(filters, " AND ")+`)`)
			args = append(args, filterArgs...)
		}
		// A trace matches a source when any of its transactions or spans
		// came from it: a mixed Sentry + OTel trace matches both.
		if source != "" {
			conditions = append(conditions, `(EXISTS (SELECT 1 FROM sentry_transactions t2 WHERE t2.trace_id = t.trace_id AND t2.source = ?)
				OR EXISTS (SELECT 1 FROM sentry_spans s WHERE s.trace_id = t.trace_id AND s.source = ?))`)
			args = append(args, source, source)
		}
		where := " WHERE " + strings.Join(conditions, " AND ")

		var total int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions t`+where, args...).Scan(&total)

		queryArgs := append(append([]any{}, args...), limit, offset)
		rows, err := db.Query(
			`SELECT t.trace_id, t.id, t.transaction_name, t.op, t.status, t.duration_ms,
				tr.span_count, tr.error_count, t.start_ts
			FROM sentry_transactions t
			JOIN sentry_traces tr ON tr.trace_id = t.trace_id`+where+`
			ORDER BY t.start_ts DESC
			LIMIT ? OFFSET ?`, queryArgs...,
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
		pc, pa := projectClause(r, "project_id")
		var data []map[string]any
		for _, tr := range traceRows {
			previewSpans := loadPreviewSpans(db, tr.traceID, tr.startTS, 3, pc, pa)
			sources, services := traceSourcesServices(db, tr.traceID, pc, pa)

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
				"sources":          sources,
				"services":         services,
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
		// ?project: only that project's part of the trace (a distributed trace
		// can hold spans of several projects).
		pc, pa := projectClause(r, "project_id")
		targs := append([]any{traceID}, pa...)

		// Load transaction.
		var (
			txnID, txnName string
			op, status     sql.NullString
			startTS, endTS sql.NullString
			durationMS     sql.NullInt64
			txnSource      string
			txnService     sql.NullString
		)
		err := db.QueryRow(
			`SELECT id, transaction_name, op, status, start_ts, end_ts, duration_ms, source, service_name
			FROM sentry_transactions r WHERE trace_id = ?`+pc+` ORDER BY `+rootOrder+` LIMIT 1`, targs...,
		).Scan(&txnID, &txnName, &op, &status, &startTS, &endTS, &durationMS, &txnSource, &txnService)
		if err == sql.ErrNoRows {
			apiError(w, "trace not found", http.StatusNotFound)
			return
		}
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Load all spans.
		spans, rootSpanID := loadAllSpans(db, traceID, txnID, startTS, pc, pa)
		sources, _ := traceSourcesServices(db, traceID, pc, pa)

		// Load related errors.
		relatedErrors := loadRelatedErrors(db, traceID, pc, pa)

		// Count related logs.
		var relatedLogCount int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_logs WHERE trace_id = ?`+pc, targs...).Scan(&relatedLogCount)

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
				"sources":          sources,
				"span_id":          nullIfEmpty(rootSpanID),
				"source":           txnSource,
				"service_name":     scanNullString(txnService),
			},
			"spans":             spans,
			"related_errors":    relatedErrors,
			"related_log_count": relatedLogCount,
		}

		apiJSON(w, result)
	}
}

// loadAllSpans returns the trace's spans and the span id of the root
// transaction rootTxnID (shown as the header, not listed as a span).
func loadAllSpans(db *sql.DB, traceID, rootTxnID string, txnStartTS sql.NullString, projectClause string, projectArgs []any) ([]map[string]any, string) {
	rows, err := db.Query(
		`SELECT span_id, parent_span_id, op, description, status, start_ts, duration_ms,
			peer_type, peer_address, is_error, source, service_name, attributes
		FROM sentry_spans WHERE trace_id = ?`+projectClause+` ORDER BY start_ts`, append([]any{traceID}, projectArgs...)...,
	)
	if err != nil {
		return []map[string]any{}, ""
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
			spanSrc                           string
			serviceName, attrsStr             sql.NullString
		)
		if err := rows.Scan(&spanID, &parentSpanID, &op, &description, &status, &sStartTS, &durationMS, &peerType, &peerAddress, &isError, &spanSrc, &serviceName, &attrsStr); err != nil {
			continue
		}
		var attrs any
		if attrsStr.Valid {
			_ = json.Unmarshal([]byte(attrsStr.String), &attrs)
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
			"source":          spanSrc,
			"service_name":    scanNullString(serviceName),
			"attributes":      attrs,
		})
	}

	// Every other transaction is the span where a request entered another
	// service: list it too, so the spans form one tree under the root
	// (the header). A span row with the same id wins.
	seen := make(map[string]bool, len(result))
	for _, s := range result {
		seen[s["span_id"].(string)] = true
	}
	rootSpanID := ""
	for _, s := range loadTransactionSpans(db, traceID, txnStartTS, projectClause, projectArgs) {
		if s["transaction_id"] == rootTxnID {
			rootSpanID = s["span_id"].(string)
			continue
		}
		if !seen[s["span_id"].(string)] {
			result = append(result, s)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i]["start_offset_ms"].(int) < result[j]["start_offset_ms"].(int)
	})

	if result == nil {
		result = []map[string]any{}
	}
	return result, rootSpanID
}

// loadTransactionSpans returns a trace's transactions as span entries, in
// the shape loadAllSpans uses, marked is_transaction.
func loadTransactionSpans(db *sql.DB, traceID string, txnStartTS sql.NullString, projectClause string, projectArgs []any) []map[string]any {
	rows, err := db.Query(
		`SELECT id, transaction_name, op, status, start_ts, duration_ms, source, service_name, payload
		FROM sentry_transactions WHERE trace_id = ?`+projectClause, append([]any{traceID}, projectArgs...)...,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		var (
			id, name, src, payload string
			op, status, startTS    sql.NullString
			serviceName            sql.NullString
			durationMS             sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &op, &status, &startTS, &durationMS, &src, &serviceName, &payload); err != nil {
			continue
		}
		// A segment's payload is the span (span_id, parent_span_id; OTel
		// attributes, or Sentry data); a Sentry transaction keeps them in
		// contexts.trace.
		var p struct {
			SpanID       string         `json:"span_id"`
			ParentSpanID string         `json:"parent_span_id"`
			Attributes   map[string]any `json:"attributes"`
			Data         map[string]any `json:"data"`
			Contexts     struct {
				Trace struct {
					SpanID       string         `json:"span_id"`
					ParentSpanID string         `json:"parent_span_id"`
					Data         map[string]any `json:"data"`
				} `json:"trace"`
			} `json:"contexts"`
		}
		_ = json.Unmarshal([]byte(payload), &p)
		spanID := firstNonEmpty(p.SpanID, p.Contexts.Trace.SpanID)
		if spanID == "" {
			continue
		}
		var attrs any
		switch {
		case p.Attributes != nil:
			attrs = p.Attributes
		case p.Data != nil:
			attrs = p.Data
		case p.Contexts.Trace.Data != nil:
			attrs = p.Contexts.Trace.Data
		}
		result = append(result, map[string]any{
			"span_id":         spanID,
			"parent_span_id":  nullIfEmpty(firstNonEmpty(p.ParentSpanID, p.Contexts.Trace.ParentSpanID)),
			"op":              scanNullString(op),
			"description":     name,
			"status":          scanNullString(status),
			"start_offset_ms": computeOffsetMS(txnStartTS, startTS),
			"duration_ms":     scanNullInt(durationMS),
			"peer_type":       nil,
			"peer_address":    nil,
			"is_error":        isErrorStatus(status.String),
			"source":          src,
			"service_name":    scanNullString(serviceName),
			"attributes":      attrs,
			"is_transaction":  true,
			"transaction_id":  id,
		})
	}
	return result
}

// traceSourcesServices lists where a trace's data came from ("sentry",
// "otel"; Sentry first) and its services in order of their first span.
func traceSourcesServices(db *sql.DB, traceID string, projectClause string, projectArgs []any) ([]string, []string) {
	args := append(append(append([]any{traceID}, projectArgs...), traceID), projectArgs...)

	sources := []string{}
	if rows, err := db.Query(
		`SELECT DISTINCT source FROM (
			SELECT source FROM sentry_transactions WHERE trace_id = ?`+projectClause+`
			UNION ALL
			SELECT source FROM sentry_spans WHERE trace_id = ?`+projectClause+`)`, args...,
	); err == nil {
		for rows.Next() {
			var s string
			if rows.Scan(&s) == nil {
				sources = append(sources, s)
			}
		}
		rows.Close()
	}
	sort.Slice(sources, func(i, j int) bool {
		if (sources[i] == sourceSentry) != (sources[j] == sourceSentry) {
			return sources[i] == sourceSentry
		}
		return sources[i] < sources[j]
	})

	services := []string{}
	if rows, err := db.Query(
		`SELECT service_name FROM (
			SELECT service_name, start_ts FROM sentry_transactions WHERE trace_id = ? AND service_name IS NOT NULL`+projectClause+`
			UNION ALL
			SELECT service_name, start_ts FROM sentry_spans WHERE trace_id = ? AND service_name IS NOT NULL`+projectClause+`)
		GROUP BY service_name ORDER BY MIN(start_ts), service_name`, args...,
	); err == nil {
		for rows.Next() {
			var s string
			if rows.Scan(&s) == nil {
				services = append(services, s)
			}
		}
		rows.Close()
	}
	return sources, services
}

func loadRelatedErrors(db *sql.DB, traceID string, projectClause string, projectArgs []any) []map[string]any {
	// Use a single query to avoid nested connection issues with SQLite.
	rows, err := db.Query(
		`SELECT e.event_id, se.exception_type, ` + tsUTC("e.received_at") + `
		FROM sentry_error_events e
		LEFT JOIN sentry_exceptions se ON se.error_event_id = e.id AND se.position = 0
		WHERE e.trace_id = ?`+strings.Replace(projectClause, "project_id", "e.project_id", 1)+` ORDER BY e.received_at DESC`, append([]any{traceID}, projectArgs...)...,
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
