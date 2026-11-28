package sentry

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
)

// The MCP tools sentry_traces_list and sentry_trace_get read the same data
// as GET /api/sentry/traces and /api/sentry/traces/{id}, cut down for an AI
// agent: no preview spans or raw attributes, spans as a tree, N+1 groups
// and queue job attempts of the trace.

// TracesQuery holds the filters of sentry_traces_list.
type TracesQuery struct {
	Project       string
	Name          string // the root transaction's name contains it
	MinDurationMS int    // the root transaction took at least this long
	NPlusOne      bool   // only traces with an N+1 query group
	Window        string // 15m, 1h, 24h, 7d
	Limit         int
}

const (
	agentTracesDefault = 20
	agentTracesMax     = 100
	// agentNPlusOneScan caps how many traces, newest first, the N+1 filter
	// looks through: the check runs per trace, not in SQL.
	agentNPlusOneScan = 1000
	agentSpansDefault = 300
	agentSpansMax     = 2000
	agentDescMax      = 500
)

// TracesForAgent lists traces, newest first, each with its N+1 group count.
func TracesForAgent(db *sql.DB, in TracesQuery) (map[string]any, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = agentTracesDefault
	}
	if limit > agentTracesMax {
		limit = agentTracesMax
	}
	q := url.Values{}
	for k, v := range map[string]string{"project": in.Project, "name": in.Name, "window": in.Window} {
		if v != "" {
			q.Set(k, v)
		}
	}
	if in.MinDurationMS > 0 {
		q.Set("min_duration", strconv.Itoa(in.MinDurationMS))
	}
	pc, pa := "", []any(nil)
	if in.Project != "" {
		pc, pa = " AND project_id = ?", []any{in.Project}
	}

	pageSize := limit
	if in.NPlusOne {
		pageSize = agentTracesMax
	}
	q.Set("limit", strconv.Itoa(pageSize))

	data := []map[string]any{}
	scanned, total := 0, 0
	for page := 1; ; page++ {
		q.Set("page", strconv.Itoa(page))
		res, _, err := tracesList(db, &http.Request{URL: &url.URL{RawQuery: q.Encode()}})
		if err != nil {
			return nil, err
		}
		total = res["meta"].(map[string]any)["total"].(int)
		rows := res["data"].([]map[string]any)
		for _, row := range rows {
			scanned++
			groups := loadNPlusOne(db, row["trace_id"].(string), pc, pa)
			if in.NPlusOne && len(groups) == 0 {
				continue
			}
			delete(row, "preview_spans")
			row["n_plus_one_groups"] = len(groups)
			data = append(data, row)
			if len(data) == limit {
				break
			}
		}
		if !in.NPlusOne || len(data) == limit || len(rows) < pageSize || scanned >= agentNPlusOneScan {
			break
		}
	}

	meta := map[string]any{"returned": len(data), "total": total}
	if in.NPlusOne {
		// total counts traces before the N+1 check.
		meta["scanned"] = scanned
	}
	return map[string]any{"data": data, "meta": meta}, nil
}

// TraceForAgent returns one trace: the root transaction, the spans as a
// tree (at most maxSpans, earliest first), N+1 groups, errors and queue job
// attempts. nil when there is no such trace.
func TraceForAgent(db *sql.DB, traceID, project string, maxSpans int) (map[string]any, error) {
	if maxSpans <= 0 {
		maxSpans = agentSpansDefault
	}
	if maxSpans > agentSpansMax {
		maxSpans = agentSpansMax
	}
	q := url.Values{}
	pc, pa := "", []any(nil)
	if project != "" {
		q.Set("project", project)
		pc, pa = " AND project_id = ?", []any{project}
	}
	res, code, err := traceDetail(db, &http.Request{URL: &url.URL{RawQuery: q.Encode()}}, traceID)
	if code == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	spans := res["spans"].([]map[string]any)
	tree := spanTree(spans, maxSpans)

	groups := res["n_plus_one"].([]map[string]any)
	for _, g := range groups {
		ids := g["span_ids"].([]string)
		if len(ids) > 3 {
			ids = ids[:3]
		}
		delete(g, "span_ids")
		g["example_span_ids"] = ids
	}

	ext := loadTraceExtent(db, traceID, pc, pa)
	out := map[string]any{
		"trace_id":          traceID,
		"duration_ms":       scanNullInt(ext.durationMS),
		"transaction":       res["transaction"],
		"span_count":        len(spans),
		"spans":             tree,
		"n_plus_one":        groups,
		"errors":            res["related_errors"],
		"related_log_count": res["related_log_count"],
		"queue_jobs":        traceQueueJobs(db, traceID, pc, pa),
	}
	if len(spans) > maxSpans {
		out["spans_truncated"] = map[string]any{"shown": maxSpans, "total": len(spans)}
	}
	return out, nil
}

// spanTree nests the first max spans (sorted by start) under their parents.
// A span whose parent is not among them (the root transaction, or one cut
// off) is a top-level node.
func spanTree(spans []map[string]any, max int) []map[string]any {
	if len(spans) > max {
		spans = spans[:max]
	}
	nodes := make(map[string]map[string]any, len(spans))
	order := make([]map[string]any, 0, len(spans))
	for _, s := range spans {
		n := agentSpan(s)
		nodes[s["span_id"].(string)] = n
		order = append(order, n)
	}
	roots := []map[string]any{}
	for i, s := range spans {
		n := order[i]
		parentID, _ := s["parent_span_id"].(string)
		if p := nodes[parentID]; p != nil && parentID != s["span_id"] {
			kids, _ := p["children"].([]map[string]any)
			p["children"] = append(kids, n)
			continue
		}
		roots = append(roots, n)
	}
	return roots
}

// agentSpan keeps the fields an agent reads, the code place from the
// attributes, and drops empty ones.
func agentSpan(s map[string]any) map[string]any {
	n := map[string]any{
		"span_id":         s["span_id"],
		"start_offset_ms": s["start_offset_ms"],
		"duration_ms":     s["duration_ms"],
	}
	for _, k := range []string{"op", "status", "service_name", "peer_type"} {
		if v, ok := s[k].(string); ok && v != "" && v != "unknown" {
			n[k] = v
		}
	}
	if d, ok := s["description"].(string); ok && d != "" {
		if r := []rune(d); len(r) > agentDescMax {
			d = string(r[:agentDescMax]) + "…"
		}
		n["description"] = d
	}
	if e, _ := s["is_error"].(bool); e {
		n["is_error"] = true
	}
	if t, _ := s["is_transaction"].(bool); t {
		n["is_transaction"] = true
		n["transaction_id"] = s["transaction_id"]
	}
	if attrs, ok := s["attributes"].(map[string]any); ok {
		if file, _ := attrs["code.filepath"].(string); file != "" {
			origin := file
			switch line := attrs["code.lineno"].(type) {
			case float64:
				origin += ":" + strconv.Itoa(int(line))
			case string:
				if line != "" {
					origin += ":" + line
				}
			}
			n["origin"] = origin
		}
	}
	return n
}

// traceQueueJobs lists the queue job attempts that ran in the trace.
func traceQueueJobs(db *sql.DB, traceID, projectClause string, projectArgs []any) []map[string]any {
	out := []map[string]any{}
	rows, err := db.Query(`SELECT transaction_id, message_id, job, queue, attempt, status, duration_ms, latency_ms, started_at, finished_at
		FROM sentry_queue_jobs WHERE trace_id = ?`+projectClause+` ORDER BY started_at, attempt`,
		append([]any{traceID}, projectArgs...)...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			txnID, job, queue, status string
			messageID, started, ended sql.NullString
			attempt                   int
			duration, latency         sql.NullInt64
		)
		if rows.Scan(&txnID, &messageID, &job, &queue, &attempt, &status, &duration, &latency, &started, &ended) != nil {
			continue
		}
		out = append(out, map[string]any{
			"transaction_id": txnID,
			"message_id":     scanNullString(messageID),
			"job":            job,
			"queue":          queue,
			"attempt":        attempt,
			"status":         status,
			"duration_ms":    scanNullInt(duration),
			"latency_ms":     scanNullInt(latency),
			"started_at":     scanNullString(started),
			"finished_at":    scanNullString(ended),
		})
	}
	return out
}
