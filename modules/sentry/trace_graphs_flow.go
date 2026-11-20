package sentry

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// flowSpan is one service's part of a request: from its first entry to the
// end of its last.
type flowSpan struct {
	service    string
	start, end time.Time
	isError    bool
}

type flowRequest struct {
	traceID, name string
	start, end    time.Time
	isError       bool
	services      map[string]*flowSpan
}

const (
	flowDefaultLimit = 500
	flowMaxLimit     = 5000
)

// handleFlow lists the requests of a period as rows across services: each
// trace with the time every service spent on it, plus per-bucket counts of
// all requests in the period.
func handleFlow(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := flowDefaultLimit
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = min(n, flowMaxLimit)
			}
		}
		from, to := graphPeriod(r)
		entries, err := loadGraphEntries(db, r, from, to)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		spanErrors, err := loadSpanErrors(db, r, from, to)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		byTrace := map[string]*flowRequest{}
		roots := map[string]graphEntry{}
		var seen []time.Time
		for _, e := range entries {
			if e.service == "" {
				e.service = "unknown"
			}
			seen = append(seen, e.start)
			req, ok := byTrace[e.traceID]
			if !ok {
				req = &flowRequest{traceID: e.traceID, start: e.start, end: e.end, services: map[string]*flowSpan{}}
				byTrace[e.traceID] = req
			}
			if e.start.Before(req.start) {
				req.start = e.start
			}
			if e.end.After(req.end) {
				req.end = e.end
			}
			fs, ok := req.services[e.service]
			if !ok {
				fs = &flowSpan{service: e.service, start: e.start, end: e.end}
				req.services[e.service] = fs
			}
			if e.start.Before(fs.start) {
				fs.start = e.start
			}
			if e.end.After(fs.end) {
				fs.end = e.end
			}
			if e.isError() || spanErrors[[2]string{e.traceID, e.service}] {
				fs.isError = true
				req.isError = true
			}
			// Root: no parent span, else the earliest.
			cur, ok := roots[e.traceID]
			isRoot := !e.parentSpanID.Valid || e.parentSpanID.String == ""
			curRoot := ok && (!cur.parentSpanID.Valid || cur.parentSpanID.String == "")
			if !ok || (isRoot && !curRoot) || (isRoot == curRoot && e.start.Before(cur.start)) {
				roots[e.traceID] = e
			}
		}
		for id, e := range roots {
			byTrace[id].name = e.name
		}

		from, to = resolveBounds(from, to, seen)
		b := newBuckets(from, to)
		all := b.series()

		requests := make([]*flowRequest, 0, len(byTrace))
		for _, req := range byTrace {
			requests = append(requests, req)
			b.add(all, req.start, req.isError)
		}
		sort.Slice(requests, func(i, j int) bool {
			if !requests[i].start.Equal(requests[j].start) {
				return requests[i].start.After(requests[j].start)
			}
			return requests[i].traceID < requests[j].traceID
		})

		total := len(requests)
		truncated := total > limit
		if truncated {
			requests = requests[:limit]
		}

		out := make([]map[string]any, 0, len(requests))
		for _, req := range requests {
			parts := make([]*flowSpan, 0, len(req.services))
			for _, fs := range req.services {
				parts = append(parts, fs)
			}
			sort.Slice(parts, func(i, j int) bool {
				if !parts[i].start.Equal(parts[j].start) {
					return parts[i].start.Before(parts[j].start)
				}
				return parts[i].service < parts[j].service
			})
			ps := make([]map[string]any, 0, len(parts))
			for _, fs := range parts {
				ps = append(ps, map[string]any{
					"service":  fs.service,
					"start_ts": formatTS(fs.start),
					"end_ts":   formatTS(fs.end),
					"is_error": fs.isError,
				})
			}
			out = append(out, map[string]any{
				"trace_id":    req.traceID,
				"name":        req.name,
				"start_ts":    formatTS(req.start),
				"end_ts":      formatTS(req.end),
				"duration_ms": req.end.Sub(req.start).Milliseconds(),
				"is_error":    req.isError,
				"services":    ps,
			})
		}

		apiJSON(w, map[string]any{
			"from":           formatTS(from),
			"to":             formatTS(to),
			"bucket_seconds": int(b.step / time.Second),
			"total":          total,
			"truncated":      truncated,
			"services":       flowServiceOrder(byTrace),
			"buckets":        b.json(all),
			"requests":       out,
		})
	}
}

// flowServiceOrder puts callers before callees: by how long after its
// request's start a service starts on average, then by name.
func flowServiceOrder(byTrace map[string]*flowRequest) []string {
	sum := map[string]time.Duration{}
	n := map[string]int{}
	for _, req := range byTrace {
		for name, fs := range req.services {
			sum[name] += fs.start.Sub(req.start)
			n[name]++
		}
	}
	names := sortedKeys(n)
	avg := func(s string) time.Duration { return sum[s] / time.Duration(n[s]) }
	sort.SliceStable(names, func(i, j int) bool { return avg(names[i]) < avg(names[j]) })
	return names
}

// loadSpanErrors marks the (trace, service) pairs with a failed span.
func loadSpanErrors(db *sql.DB, r *http.Request, from, to time.Time) (map[[2]string]bool, error) {
	pc, pa := projectClause(r, "s.project_id")
	tc, ta := periodConds("s.start_ts", from, to)
	rows, err := db.Query(
		`SELECT DISTINCT s.trace_id, COALESCE(s.service_name, 'unknown') FROM sentry_spans s
		WHERE s.is_error = 1`+tc+pc, append(ta, pa...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[[2]string]bool{}
	for rows.Next() {
		var trace, service string
		if err := rows.Scan(&trace, &service); err != nil {
			return nil, err
		}
		out[[2]string{trace, service}] = true
	}
	return out, rows.Err()
}

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
