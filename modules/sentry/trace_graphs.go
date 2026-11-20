package sentry

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	httpserver "github.com/buggregator/server/v2/internal/server/http"
)

// Shared parts of the trace graphs: the service map over a period, the
// request flow and a trace's call sequence. Rows are read once per request
// and aggregated here: SQLite has no percentile, and the same rows feed
// counts, p95, time buckets and trace lists.

const isoSecond = "2006-01-02T15:04:05"

// graphPeriod reads the period of a graph request.
//
//	from, to — unix seconds or ISO, as the other sentry lists;
//	window   — "24h", "15m", or a plain number of minutes (the service map's
//	           old parameter); "all" means no bound.
//
// Nothing given: the last 60 minutes, what the service map always showed.
// A zero bound means unbounded.
func graphPeriod(r *http.Request) (from, to time.Time) {
	q := r.URL.Query()
	from, to = parseWhen(q.Get("from")), parseWhen(q.Get("to"))
	if !from.IsZero() || !to.IsZero() {
		return from, to
	}
	w := strings.TrimSpace(q.Get("window"))
	switch {
	case w == "":
		return now().UTC().Add(-60 * time.Minute), time.Time{}
	case w == "all" || w == "0":
		return time.Time{}, time.Time{}
	}
	if n, err := strconv.Atoi(w); err == nil && n > 0 && n <= 10080 {
		return now().UTC().Add(-time.Duration(n) * time.Minute), time.Time{}
	}
	if d, err := httpserver.ParseWindow(w); err == nil && d > 0 {
		return now().UTC().Add(-d), time.Time{}
	}
	return now().UTC().Add(-60 * time.Minute), time.Time{}
}

// periodConds narrows an ISO start_ts column to the period.
func periodConds(column string, from, to time.Time) (string, []any) {
	var sqlPart string
	var args []any
	if !from.IsZero() {
		sqlPart += " AND " + column + " >= ?"
		args = append(args, from.UTC().Format(isoSecond))
	}
	if !to.IsZero() {
		// Inclusive of the whole last second: "…:05.7Z" > "…:05".
		sqlPart += " AND " + column + " < ?"
		args = append(args, to.UTC().Add(time.Second).Format(isoSecond))
	}
	return sqlPart, args
}

// parseTS reads a stored start_ts/end_ts (ISO with zone).
func parseTS(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

func formatTS(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// resolveBounds fills an unbounded side from the data seen; a still empty
// period becomes the last hour.
func resolveBounds(from, to time.Time, seen []time.Time) (time.Time, time.Time) {
	if to.IsZero() {
		to = now().UTC()
		for _, t := range seen {
			if t.After(to) {
				to = t
			}
		}
	}
	if from.IsZero() {
		from = to
		for _, t := range seen {
			if !t.IsZero() && t.Before(from) {
				from = t
			}
		}
		if !from.Before(to) {
			from = to.Add(-time.Hour)
		}
	}
	return from, to
}

// bucketSteps are the bucket sizes a period is cut into, smallest first.
var bucketSteps = []time.Duration{
	time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour, 7 * 24 * time.Hour,
}

const maxBuckets = 60

// buckets cut a period into at most maxBuckets equal steps, aligned to the
// step (a 5-minute bucket starts at :00, :05, …).
type buckets struct {
	start time.Time
	step  time.Duration
	n     int
}

func newBuckets(from, to time.Time) buckets {
	span := to.Sub(from)
	step := bucketSteps[len(bucketSteps)-1]
	for _, s := range bucketSteps {
		if span <= s*maxBuckets {
			step = s
			break
		}
	}
	start := from.Truncate(step)
	n := int(to.Sub(start)/step) + 1
	if n > maxBuckets+1 {
		n = maxBuckets + 1
	}
	return buckets{start: start, step: step, n: n}
}

// series counts requests and errors per bucket.
type series struct {
	counts, errors []int
}

func (b buckets) series() series {
	return series{counts: make([]int, b.n), errors: make([]int, b.n)}
}

func (b buckets) add(s series, t time.Time, isErr bool) {
	i := int(t.Sub(b.start) / b.step)
	if t.IsZero() || i < 0 || i >= b.n {
		return
	}
	s.counts[i]++
	if isErr {
		s.errors[i]++
	}
}

func (b buckets) json(s series) []map[string]any {
	out := make([]map[string]any, b.n)
	for i := range out {
		out[i] = map[string]any{
			"t":      formatTS(b.start.Add(time.Duration(i) * b.step)),
			"count":  s.counts[i],
			"errors": s.errors[i],
		}
	}
	return out
}

// stats collects a group's counts and durations.
type stats struct {
	count, errors int
	durations     []int
	series        series
}

func (s *stats) add(b buckets, t time.Time, durationMS sql.NullInt64, isErr bool) {
	if s.series.counts == nil {
		s.series = b.series()
	}
	s.count++
	if isErr {
		s.errors++
	}
	if durationMS.Valid {
		s.durations = append(s.durations, int(durationMS.Int64))
	}
	b.add(s.series, t, isErr)
}

func (s *stats) json(b buckets, minutes float64) map[string]any {
	if s.series.counts == nil {
		s.series = b.series()
	}
	var avg, p95 *int
	if len(s.durations) > 0 {
		sort.Ints(s.durations)
		sum := 0
		for _, d := range s.durations {
			sum += d
		}
		a := sum / len(s.durations)
		avg = &a
		// Nearest rank.
		idx := (95*len(s.durations)+99)/100 - 1
		p := s.durations[idx]
		p95 = &p
	}
	rate := 0.0
	if s.count > 0 {
		rate = float64(s.errors) / float64(s.count)
	}
	rpm := 0.0
	if minutes > 0 {
		rpm = float64(s.count) / minutes
	}
	return map[string]any{
		"request_count":   s.count,
		"error_count":     s.errors,
		"error_rate":      round3(rate),
		"rpm":             round3(rpm),
		"avg_duration_ms": avg,
		"p95_duration_ms": p95,
		"buckets":         b.json(s.series),
	}
}

func round3(f float64) float64 {
	return float64(int64(f*1000+0.5)) / 1000
}

// txnIsError: a transaction failed when its status is set and not ok.
func txnIsError(status string) bool {
	return status != "" && status != "ok" && status != "unset"
}

// graphCall is one outgoing call of a service in the period: a span with a
// peer, drawn to the service that received it when that service reports too.
type graphCall struct {
	source, target, opType string
	isError                bool
	duration               sql.NullInt64
	start                  time.Time
	traceID                string
}

// loadGraphCalls reads the calls in the period. The callee is the other
// service whose entry transaction, or span, names the call as its parent.
// Databases, caches and queues keep their address.
func loadGraphCalls(db *sql.DB, r *http.Request, from, to time.Time) ([]graphCall, error) {
	pc, pa := projectClause(r, "s.project_id")
	tc, ta := periodConds("s.start_ts", from, to)
	rows, err := db.Query(
		`SELECT
			COALESCE(s.service_name, 'unknown'),
			COALESCE(CASE WHEN s.peer_type IN ('db', 'cache', 'queue') THEN NULL ELSE COALESCE(
				(SELECT t.service_name FROM sentry_transactions t
					WHERE t.trace_id = s.trace_id AND t.parent_span_id = s.span_id
						AND t.service_name <> COALESCE(s.service_name, '') LIMIT 1),
				(SELECT d.service_name FROM sentry_spans d
					WHERE d.trace_id = s.trace_id AND d.parent_span_id = s.span_id
						AND d.service_name <> COALESCE(s.service_name, '') LIMIT 1),
				(SELECT sp.service_name FROM sentry_transactions t
					JOIN sentry_spans sp ON sp.transaction_id = t.id
					WHERE t.trace_id = s.trace_id AND t.parent_span_id = s.span_id
						AND sp.service_name <> COALESCE(s.service_name, '') LIMIT 1)
			) END, s.peer_address),
			COALESCE(s.peer_type, 'unknown'), s.is_error, s.duration_ms, s.start_ts, s.trace_id
		FROM sentry_spans s
		WHERE s.peer_address IS NOT NULL`+tc+pc,
		append(ta, pa...)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []graphCall
	for rows.Next() {
		var c graphCall
		var isErr sql.NullInt64
		var start sql.NullString
		if err := rows.Scan(&c.source, &c.target, &c.opType, &isErr, &c.duration, &start, &c.traceID); err != nil {
			return nil, err
		}
		c.isError = isErr.Int64 == 1
		c.start = parseTS(start.String)
		out = append(out, c)
	}
	return out, rows.Err()
}

// graphEntry is a service's entry transaction: one request it served.
type graphEntry struct {
	id, traceID, service, name, status string
	parentSpanID                       sql.NullString
	start, end                         time.Time
	duration                           sql.NullInt64
}

func (e graphEntry) isError() bool { return txnIsError(e.status) }

func loadGraphEntries(db *sql.DB, r *http.Request, from, to time.Time) ([]graphEntry, error) {
	pc, pa := projectClause(r, "t.project_id")
	tc, ta := periodConds("t.start_ts", from, to)
	rows, err := db.Query(
		`SELECT t.id, t.trace_id, COALESCE(t.service_name, ''), COALESCE(t.transaction_name, ''),
			COALESCE(t.status, ''), t.parent_span_id, t.start_ts, t.end_ts, t.duration_ms
		FROM sentry_transactions t
		WHERE 1 = 1`+tc+pc,
		append(ta, pa...)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []graphEntry
	for rows.Next() {
		var e graphEntry
		var start, end sql.NullString
		if err := rows.Scan(&e.id, &e.traceID, &e.service, &e.name, &e.status, &e.parentSpanID, &start, &end, &e.duration); err != nil {
			return nil, err
		}
		e.start, e.end = parseTS(start.String), parseTS(end.String)
		out = append(out, e)
	}
	return out, rows.Err()
}

// edgeTrace is a trace through an edge, for its drill-down list.
type edgeTrace struct {
	traceID  string
	start    time.Time
	duration int64
	isError  bool
}

// edgeTraces groups an edge's calls by trace: errors first, then newest.
func edgeTraces(calls []graphCall) []edgeTrace {
	byID := map[string]*edgeTrace{}
	var order []*edgeTrace
	for _, c := range calls {
		t, ok := byID[c.traceID]
		if !ok {
			t = &edgeTrace{traceID: c.traceID, start: c.start}
			byID[c.traceID] = t
			order = append(order, t)
		}
		if c.start.Before(t.start) {
			t.start = c.start
		}
		if c.duration.Valid && c.duration.Int64 > t.duration {
			t.duration = c.duration.Int64
		}
		t.isError = t.isError || c.isError
	}
	sort.SliceStable(order, func(i, j int) bool {
		if order[i].isError != order[j].isError {
			return order[i].isError
		}
		if !order[i].start.Equal(order[j].start) {
			return order[i].start.After(order[j].start)
		}
		return order[i].traceID < order[j].traceID
	})
	out := make([]edgeTrace, len(order))
	for i, t := range order {
		out[i] = *t
	}
	return out
}

func edgeTracesJSON(list []edgeTrace) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, map[string]any{
			"trace_id":    t.traceID,
			"start_ts":    formatTS(t.start),
			"duration_ms": t.duration,
			"is_error":    t.isError,
		})
	}
	return out
}

const edgeTracesInline = 20

// handleServiceMap draws services and the calls between them over a period:
// per node and edge, requests, errors, rpm, avg and p95 duration, and time
// buckets; per edge the traces through it.
func handleServiceMap(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to := graphPeriod(r)
		calls, err := loadGraphCalls(db, r, from, to)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries, err := loadGraphEntries(db, r, from, to)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var seen []time.Time
		for _, c := range calls {
			seen = append(seen, c.start)
		}
		for _, e := range entries {
			seen = append(seen, e.start)
		}
		from, to = resolveBounds(from, to, seen)
		b := newBuckets(from, to)
		minutes := to.Sub(from).Minutes()

		// Services: whoever served a request or made a call.
		services := map[string]*stats{}
		service := func(name string) *stats {
			s, ok := services[name]
			if !ok {
				s = &stats{}
				services[name] = s
			}
			return s
		}
		// A transaction without a service name (stored before 000036) is no
		// node of its own: its spans still draw its calls.
		for _, e := range entries {
			if e.service != "" {
				service(e.service).add(b, e.start, e.duration, e.isError())
			}
		}
		for _, c := range calls {
			service(c.source)
		}

		type edgeKey struct{ source, target, opType string }
		edgeStats := map[edgeKey]*stats{}
		edgeCalls := map[edgeKey][]graphCall{}
		var keys []edgeKey
		for _, c := range calls {
			k := edgeKey{c.source, c.target, c.opType}
			s, ok := edgeStats[k]
			if !ok {
				s = &stats{}
				edgeStats[k] = s
				keys = append(keys, k)
			}
			s.add(b, c.start, c.duration, c.isError)
			edgeCalls[k] = append(edgeCalls[k], c)
		}
		sort.Slice(keys, func(i, j int) bool {
			ci, cj := edgeStats[keys[i]].count, edgeStats[keys[j]].count
			if ci != cj {
				return ci > cj
			}
			if keys[i].source != keys[j].source {
				return keys[i].source < keys[j].source
			}
			return keys[i].target < keys[j].target
		})

		// External nodes: targets that are not services, measured by the
		// calls into them.
		externals := map[string]*stats{}
		externalType := map[string]string{}
		edges := make([]map[string]any, 0, len(keys))
		for _, k := range keys {
			if _, ok := services[k.target]; !ok {
				s, ok := externals[k.target]
				if !ok {
					s = &stats{}
					externals[k.target] = s
					externalType[k.target] = k.opType
				}
				for _, c := range edgeCalls[k] {
					s.add(b, c.start, c.duration, c.isError)
				}
			}
			e := edgeStats[k].json(b, minutes)
			e["source"], e["target"], e["op_type"] = k.source, k.target, k.opType
			traces := edgeTraces(edgeCalls[k])
			e["trace_count"] = len(traces)
			if len(traces) > edgeTracesInline {
				traces = traces[:edgeTracesInline]
			}
			e["traces"] = edgeTracesJSON(traces)
			edges = append(edges, e)
		}

		nodes := make([]map[string]any, 0, len(services)+len(externals))
		for _, name := range sortedKeys(services) {
			n := services[name].json(b, minutes)
			n["id"], n["label"], n["kind"] = name, name, "service"
			nodes = append(nodes, n)
		}
		for _, name := range sortedKeys(externals) {
			n := externals[name].json(b, minutes)
			n["id"], n["label"], n["kind"] = name, labelForNode(name, externalType[name]), "external"
			nodes = append(nodes, n)
		}

		apiJSON(w, map[string]any{
			"from":           formatTS(from),
			"to":             formatTS(to),
			"bucket_seconds": int(b.step / time.Second),
			"window_minutes": int(minutes + 0.5),
			"nodes":          nodes,
			"edges":          edges,
		})
	}
}

// handleServiceMapEdge lists the traces through one edge, errors first.
func handleServiceMapEdge(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		source, target := q.Get("source"), q.Get("target")
		if source == "" || target == "" {
			apiError(w, "source and target are required", http.StatusBadRequest)
			return
		}
		limit, offset := pagination(r, 50)
		if v := q.Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		from, to := graphPeriod(r)
		calls, err := loadGraphCalls(db, r, from, to)
		if err != nil {
			apiError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var mine []graphCall
		for _, c := range calls {
			if c.source == source && c.target == target {
				mine = append(mine, c)
			}
		}
		traces := edgeTraces(mine)
		total := len(traces)
		if offset > len(traces) {
			offset = len(traces)
		}
		traces = traces[offset:]
		if len(traces) > limit {
			traces = traces[:limit]
		}
		apiJSON(w, map[string]any{"total": total, "traces": edgeTracesJSON(traces)})
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
