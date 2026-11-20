package sentry

import (
	"database/sql"
	"net/http"
	"sort"
	"time"
)

// handleTraceSequence lists a trace's calls in time order: who called whom,
// how long the caller waited, and whether the callee was still working after
// the caller stopped waiting (late_ms > 0).
func handleTraceSequence(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := r.PathValue("traceId")
		pc, pa := projectClause(r, "project_id")
		args := append([]any{traceID}, pa...)

		type txn struct {
			id, service string
			start, end  time.Time
			duration    sql.NullInt64
			isError     bool
		}
		txns := map[string]txn{}          // by id
		txnByParent := map[string][]txn{} // by parent_span_id
		tRows, err := db.Query(`SELECT id, COALESCE(service_name, ''), parent_span_id, COALESCE(status, ''),
				start_ts, end_ts, duration_ms
			FROM sentry_transactions WHERE trace_id = ?`+pc, args...)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for tRows.Next() {
			var t txn
			var parent, start, end sql.NullString
			var status string
			if err := tRows.Scan(&t.id, &t.service, &parent, &status, &start, &end, &t.duration); err != nil {
				tRows.Close()
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			t.start, t.end, t.isError = parseTS(start.String), parseTS(end.String), txnIsError(status)
			txns[t.id] = t
			if parent.Valid && parent.String != "" {
				txnByParent[parent.String] = append(txnByParent[parent.String], t)
			}
		}
		tRows.Close()

		type span struct {
			spanID, txnID, parent, service, op, desc, status, peerType, peerAddr string
			start, end                                                           time.Time
			duration                                                             sql.NullInt64
			isError                                                              bool
		}
		var spans []span
		spanByParent := map[string][]int{}
		sRows, err := db.Query(`SELECT COALESCE(span_id, ''), COALESCE(transaction_id, ''), COALESCE(parent_span_id, ''),
				COALESCE(service_name, ''), COALESCE(op, ''), COALESCE(description, ''), COALESCE(status, ''),
				COALESCE(peer_type, ''), COALESCE(peer_address, ''), start_ts, end_ts, duration_ms, COALESCE(is_error, 0)
			FROM sentry_spans WHERE trace_id = ?`+pc, args...)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for sRows.Next() {
			var s span
			var start, end sql.NullString
			var isErr int
			if err := sRows.Scan(&s.spanID, &s.txnID, &s.parent, &s.service, &s.op, &s.desc, &s.status,
				&s.peerType, &s.peerAddr, &start, &end, &s.duration, &isErr); err != nil {
				sRows.Close()
				apiError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.start, s.end, s.isError = parseTS(start.String), parseTS(end.String), isErr == 1
			if s.service == "" {
				s.service = txns[s.txnID].service
			}
			if s.parent != "" {
				spanByParent[s.parent] = append(spanByParent[s.parent], len(spans))
			}
			spans = append(spans, s)
		}
		sRows.Close()

		if len(txns) == 0 && len(spans) == 0 {
			apiError(w, "trace not found", http.StatusNotFound)
			return
		}

		type call struct {
			start time.Time
			json  map[string]any
		}
		var calls []call
		order := map[string]time.Time{}
		note := func(service string, t time.Time) {
			if service == "" {
				return
			}
			if cur, ok := order[service]; !ok || t.Before(cur) {
				order[service] = t
			}
		}
		for _, t := range txns {
			note(t.service, t.start)
		}
		for _, s := range spans {
			from := s.service
			if from == "" {
				from = "unknown"
			}
			var callee map[string]any
			var calleeEnd time.Time
			if s.peerType != "db" && s.peerType != "cache" && s.peerType != "queue" && s.spanID != "" {
				for _, t := range txnByParent[s.spanID] {
					if t.service != "" && t.service != s.service {
						callee = map[string]any{
							"service": t.service, "start_ts": formatTS(t.start), "end_ts": formatTS(t.end),
							"duration_ms": scanNullInt(t.duration), "is_error": t.isError, "transaction_id": t.id,
						}
						calleeEnd = t.end
						break
					}
				}
				if callee == nil {
					for _, i := range spanByParent[s.spanID] {
						d := spans[i]
						if d.service != "" && d.service != s.service {
							callee = map[string]any{
								"service": d.service, "start_ts": formatTS(d.start), "end_ts": formatTS(d.end),
								"duration_ms": scanNullInt(d.duration), "is_error": d.isError, "transaction_id": nil,
							}
							calleeEnd = d.end
							break
						}
					}
				}
			}
			if callee == nil && s.peerAddr == "" {
				continue
			}
			to := s.peerAddr
			if callee != nil {
				to = callee["service"].(string)
				// A span-only callee (span v2, no transaction) needs its lane too.
				note(to, parseTS(callee["start_ts"].(string)))
			}
			opType := s.peerType
			if opType == "" {
				opType = "unknown"
			}
			var late any
			if callee != nil && !calleeEnd.IsZero() && !s.end.IsZero() {
				late = calleeEnd.Sub(s.end).Milliseconds()
			}
			note(from, s.start)
			calls = append(calls, call{s.start, map[string]any{
				"span_id":     s.spanID,
				"from":        from,
				"to":          to,
				"op_type":     opType,
				"description": s.desc,
				"start_ts":    formatTS(s.start),
				"end_ts":      formatTS(s.end),
				"duration_ms": scanNullInt(s.duration),
				"is_error":    s.isError,
				"status":      s.status,
				"callee":      callee,
				"late_ms":     late,
			}})
		}
		sort.SliceStable(calls, func(i, j int) bool { return calls[i].start.Before(calls[j].start) })
		callsJSON := make([]map[string]any, len(calls))
		for i, c := range calls {
			callsJSON[i] = c.json
		}

		services := sortedKeys(order)
		sort.SliceStable(services, func(i, j int) bool { return order[services[i]].Before(order[services[j]]) })

		apiJSON(w, map[string]any{
			"trace_id": traceID,
			"services": services,
			"calls":    callsJSON,
		})
	}
}
