package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// graphFixture stores two requests through shop-web → checkout-api →
// payments-worker, as the OTel stand sends them, plus a db call.
//
//	tr-err 11:30:00: shop-web's call to checkout-api fails; checkout-api
//	                 is still working 50 ms after shop-web stopped waiting.
//	tr-ok  11:31:30: all ok.
func graphFixture(t *testing.T) (*sql.DB, *http.ServeMux) {
	t.Helper()
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)
	noon := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return noon }
	t.Cleanup(func() { now = time.Now })

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	txn := func(id, trace, service, name, status, parent, start, end string, dur int) {
		var p any
		if parent != "" {
			p = parent
		}
		exec(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, status, start_ts, end_ts, duration_ms, payload, project_id, service_name, parent_span_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', 'shop', ?, ?)`, id, id, trace, name, status, start, end, dur, service, p)
	}
	span := func(id, trace, txnID, service, peer, peerType, desc, start, end string, dur, isErr int) {
		var tx any
		if txnID != "" {
			tx = txnID
		}
		exec(`INSERT INTO sentry_spans (id, span_id, transaction_id, trace_id, service_name, peer_address, peer_type, description, start_ts, end_ts, duration_ms, is_error, project_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'shop')`, id, id, tx, trace, service, peer, peerType, desc, start, end, dur, isErr)
	}
	for _, tr := range []string{"tr-err", "tr-ok"} {
		exec(`INSERT INTO sentry_traces (trace_id) VALUES (?)`, tr)
	}

	txn("web-1", "tr-err", "shop-web", "POST /cart/checkout", "internal_error", "", "2026-11-16T11:30:00Z", "2026-11-16T11:30:00.2Z", 200)
	span("web-1-call", "tr-err", "web-1", "shop-web", "checkout-api", "http", "POST http://checkout-api/orders", "2026-11-16T11:30:00.01Z", "2026-11-16T11:30:00.11Z", 100, 1)
	txn("co-1", "tr-err", "checkout-api", "POST /orders", "ok", "web-1-call", "2026-11-16T11:30:00.02Z", "2026-11-16T11:30:00.16Z", 140)
	span("co-1-db", "tr-err", "", "checkout-api", "postgres:5432", "db", "INSERT orders", "2026-11-16T11:30:00.03Z", "2026-11-16T11:30:00.04Z", 10, 0)
	span("co-1-call", "tr-err", "", "checkout-api", "127.0.0.1:18302", "http", "HTTP POST", "2026-11-16T11:30:00.05Z", "2026-11-16T11:30:00.15Z", 100, 0)
	txn("pw-1", "tr-err", "payments-worker", "POST /charge", "ok", "co-1-call", "2026-11-16T11:30:00.06Z", "2026-11-16T11:30:00.14Z", 80)

	txn("web-2", "tr-ok", "shop-web", "POST /cart/checkout", "ok", "", "2026-11-16T11:31:30Z", "2026-11-16T11:31:30.3Z", 300)
	span("web-2-call", "tr-ok", "web-2", "shop-web", "checkout-api", "http", "POST http://checkout-api/orders", "2026-11-16T11:31:30.01Z", "2026-11-16T11:31:30.25Z", 240, 0)
	txn("co-2", "tr-ok", "checkout-api", "POST /orders", "ok", "web-2-call", "2026-11-16T11:31:30.02Z", "2026-11-16T11:31:30.24Z", 220)
	span("co-2-call", "tr-ok", "", "checkout-api", "127.0.0.1:18302", "http", "HTTP POST", "2026-11-16T11:31:30.05Z", "2026-11-16T11:31:30.2Z", 150, 0)
	txn("pw-2", "tr-ok", "payments-worker", "POST /charge", "ok", "co-2-call", "2026-11-16T11:31:30.06Z", "2026-11-16T11:31:30.19Z", 130)

	// Outside the period below: an hour earlier.
	exec(`INSERT INTO sentry_traces (trace_id) VALUES ('tr-old')`)
	txn("web-old", "tr-old", "shop-web", "GET /", "ok", "", "2026-11-16T10:00:00Z", "2026-11-16T10:00:00.1Z", 100)
	return db, mux
}

func getJSON(t *testing.T, mux *http.ServeMux, target string, out any) {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
	if w.Code != 200 {
		t.Fatalf("GET %s: status = %d: %s", target, w.Code, w.Body)
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("GET %s: %v: %s", target, err, w.Body)
	}
}

type graphStats struct {
	RequestCount int     `json:"request_count"`
	ErrorCount   int     `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	RPM          float64 `json:"rpm"`
	Avg          *int    `json:"avg_duration_ms"`
	P95          *int    `json:"p95_duration_ms"`
	Buckets      []struct {
		T      string `json:"t"`
		Count  int    `json:"count"`
		Errors int    `json:"errors"`
	} `json:"buckets"`
}

func TestServiceMapPeriod(t *testing.T) {
	_, mux := graphFixture(t)
	var resp struct {
		From, To      string
		BucketSeconds int `json:"bucket_seconds"`
		Nodes         []struct {
			graphStats
			ID, Kind string
		}
		Edges []struct {
			graphStats
			Source, Target string
			OpType         string `json:"op_type"`
			TraceCount     int    `json:"trace_count"`
			Traces         []struct {
				TraceID string `json:"trace_id"`
				IsError bool   `json:"is_error"`
			}
		}
	}
	getJSON(t, mux, "/api/sentry/service-map?project=shop&from=2026-11-16T11:30:00Z&to=2026-11-16T11:39:59Z", &resp)

	if resp.From != "2026-11-16T11:30:00Z" || resp.To != "2026-11-16T11:39:59Z" || resp.BucketSeconds != 60 {
		t.Errorf("period = %s..%s step %d", resp.From, resp.To, resp.BucketSeconds)
	}
	nodes := map[string]int{}
	for i, n := range resp.Nodes {
		nodes[n.ID+"/"+n.Kind] = i
	}
	for _, want := range []string{"shop-web/service", "checkout-api/service", "payments-worker/service", "postgres:5432/external"} {
		if _, ok := nodes[want]; !ok {
			t.Errorf("node %s missing: %v", want, nodes)
		}
	}
	if len(resp.Nodes) != 4 {
		t.Errorf("nodes = %v", nodes)
	}
	web := resp.Nodes[nodes["shop-web/service"]]
	// Two requests in a 10-minute period, one failed; p95 of {200, 300} = 300.
	if web.RequestCount != 2 || web.ErrorCount != 1 || web.ErrorRate != 0.5 || web.RPM != 0.2 || *web.P95 != 300 || *web.Avg != 250 {
		t.Errorf("shop-web = %+v", web.graphStats)
	}
	if len(web.Buckets) != 10 || web.Buckets[0].T != "2026-11-16T11:30:00Z" || web.Buckets[0].Count != 1 || web.Buckets[0].Errors != 1 || web.Buckets[1].Count != 1 {
		t.Errorf("shop-web buckets = %+v", web.Buckets)
	}

	edges := map[string]int{}
	for i, e := range resp.Edges {
		edges[e.Source+"->"+e.Target] = i
	}
	if got := len(resp.Edges); got != 3 {
		t.Errorf("edges = %v", edges)
	}
	call := resp.Edges[edges["shop-web->checkout-api"]]
	if call.RequestCount != 2 || call.ErrorCount != 1 || *call.P95 != 240 || call.OpType != "http" || call.TraceCount != 2 {
		t.Errorf("shop-web->checkout-api = %+v", call)
	}
	if len(call.Traces) != 2 || call.Traces[0].TraceID != "tr-err" || !call.Traces[0].IsError {
		t.Errorf("edge traces, errors first = %+v", call.Traces)
	}
	if _, ok := edges["checkout-api->payments-worker"]; !ok {
		t.Errorf("checkout-api->payments-worker missing: %v", edges)
	}
}

func TestServiceMapEdgeTraces(t *testing.T) {
	_, mux := graphFixture(t)
	var resp struct {
		Total  int
		Traces []struct {
			TraceID string `json:"trace_id"`
			IsError bool   `json:"is_error"`
		}
	}
	q := url.Values{"source": {"shop-web"}, "target": {"checkout-api"}, "window": {"all"}, "limit": {"1"}}
	getJSON(t, mux, "/api/sentry/service-map/edge?"+q.Encode(), &resp)
	if resp.Total != 2 || len(resp.Traces) != 1 || resp.Traces[0].TraceID != "tr-err" {
		t.Errorf("page 1 = %+v", resp)
	}
	q.Set("page", "2")
	getJSON(t, mux, "/api/sentry/service-map/edge?"+q.Encode(), &resp)
	if resp.Total != 2 || len(resp.Traces) != 1 || resp.Traces[0].TraceID != "tr-ok" {
		t.Errorf("page 2 = %+v", resp)
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/service-map/edge?source=a", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("no target: status = %d", w.Code)
	}
}

func TestFlow(t *testing.T) {
	_, mux := graphFixture(t)
	var resp struct {
		Total     int
		Truncated bool
		Services  []string
		Buckets   []struct{ Count, Errors int }
		Requests  []struct {
			TraceID  string `json:"trace_id"`
			Name     string
			Duration int64  `json:"duration_ms"`
			IsError  bool   `json:"is_error"`
			StartTS  string `json:"start_ts"`
			Services []struct {
				Service string
				StartTS string `json:"start_ts"`
				EndTS   string `json:"end_ts"`
				IsError bool   `json:"is_error"`
			}
		}
	}
	getJSON(t, mux, "/api/sentry/flow?project=shop&from=2026-11-16T11:30:00Z&to=2026-11-16T11:39:59Z", &resp)

	if got := strings.Join(resp.Services, " "); got != "shop-web checkout-api payments-worker" {
		t.Errorf("services = %s", got)
	}
	if resp.Total != 2 || resp.Truncated || len(resp.Requests) != 2 {
		t.Fatalf("total %d truncated %v requests %d", resp.Total, resp.Truncated, len(resp.Requests))
	}
	newest, failed := resp.Requests[0], resp.Requests[1]
	if newest.TraceID != "tr-ok" || failed.TraceID != "tr-err" {
		t.Errorf("order = %s, %s", newest.TraceID, failed.TraceID)
	}
	if failed.Name != "POST /cart/checkout" || !failed.IsError || failed.Duration != 200 || len(failed.Services) != 3 {
		t.Errorf("tr-err = %+v", failed)
	}
	// shop-web failed (its transaction and its call); checkout-api did not.
	if s := failed.Services[0]; s.Service != "shop-web" || !s.IsError {
		t.Errorf("tr-err shop-web = %+v", s)
	}
	if s := failed.Services[1]; s.Service != "checkout-api" || s.IsError || s.StartTS != "2026-11-16T11:30:00.02Z" || s.EndTS != "2026-11-16T11:30:00.16Z" {
		t.Errorf("tr-err checkout-api = %+v", s)
	}
	sum, errs := 0, 0
	for _, b := range resp.Buckets {
		sum += b.Count
		errs += b.Errors
	}
	if sum != 2 || errs != 1 {
		t.Errorf("buckets sum %d errors %d", sum, errs)
	}

	getJSON(t, mux, "/api/sentry/flow?project=shop&window=all&limit=1", &resp)
	if resp.Total != 3 || !resp.Truncated || len(resp.Requests) != 1 || resp.Requests[0].TraceID != "tr-ok" {
		t.Errorf("all time, limit 1: total %d truncated %v %+v", resp.Total, resp.Truncated, resp.Requests)
	}
}

func TestTraceSequence(t *testing.T) {
	_, mux := graphFixture(t)
	var resp struct {
		Services []string
		Calls    []struct {
			SpanID   string `json:"span_id"`
			From, To string
			OpType   string `json:"op_type"`
			IsError  bool   `json:"is_error"`
			LateMS   *int64 `json:"late_ms"`
			Callee   *struct {
				Service       string
				TransactionID *string `json:"transaction_id"`
			}
		}
	}
	getJSON(t, mux, "/api/sentry/traces/tr-err/sequence", &resp)

	if got := strings.Join(resp.Services, " "); got != "shop-web checkout-api payments-worker" {
		t.Errorf("services = %s", got)
	}
	var got []string
	for _, c := range resp.Calls {
		got = append(got, c.From+"->"+c.To)
	}
	if strings.Join(got, " ") != "shop-web->checkout-api checkout-api->postgres:5432 checkout-api->payments-worker" {
		t.Fatalf("calls = %v", got)
	}
	first := resp.Calls[0]
	if !first.IsError || first.Callee == nil || first.Callee.Service != "checkout-api" || *first.Callee.TransactionID != "co-1" {
		t.Errorf("first call = %+v", first)
	}
	// checkout-api ended 50 ms after shop-web's call did.
	if first.LateMS == nil || *first.LateMS != 50 {
		t.Errorf("late_ms = %v, want 50", first.LateMS)
	}
	if db := resp.Calls[1]; db.Callee != nil || db.OpType != "db" || db.LateMS != nil {
		t.Errorf("db call = %+v", db)
	}
	// payments-worker ended 10 ms before checkout-api stopped waiting.
	if pw := resp.Calls[2]; pw.LateMS == nil || *pw.LateMS != -10 {
		t.Errorf("payments call late_ms = %v, want -10", pw.LateMS)
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/traces/nope/sequence", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown trace: status = %d", w.Code)
	}
}

func TestGraphPeriodAndBuckets(t *testing.T) {
	noon := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return noon }
	t.Cleanup(func() { now = time.Now })

	for _, tt := range []struct {
		query    string
		from, to time.Time
	}{
		{"", noon.Add(-time.Hour), time.Time{}},
		{"window=15", noon.Add(-15 * time.Minute), time.Time{}},
		{"window=24h", noon.Add(-24 * time.Hour), time.Time{}},
		{"window=all", time.Time{}, time.Time{}},
		{"from=2026-11-16T10:00:00Z&window=all", time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC), time.Time{}},
	} {
		from, to := graphPeriod(httptest.NewRequest("GET", "/x?"+tt.query, nil))
		if !from.Equal(tt.from) || !to.Equal(tt.to) {
			t.Errorf("%q: %v..%v, want %v..%v", tt.query, from, to, tt.from, tt.to)
		}
	}

	for _, tt := range []struct {
		span time.Duration
		step time.Duration
	}{
		{10 * time.Minute, time.Minute},
		{time.Hour, time.Minute},
		{2 * time.Hour, 5 * time.Minute},
		{24 * time.Hour, 30 * time.Minute},
		{7 * 24 * time.Hour, 3 * time.Hour},
	} {
		b := newBuckets(noon.Add(-tt.span), noon)
		if b.step != tt.step || b.n > maxBuckets+1 {
			t.Errorf("%v: step %v n %d, want step %v", tt.span, b.step, b.n, tt.step)
		}
	}

	s := stats{}
	b := newBuckets(noon.Add(-time.Hour), noon)
	for i := 1; i <= 100; i++ {
		s.add(b, noon, sql.NullInt64{Int64: int64(i), Valid: true}, false)
	}
	if got := s.json(b, 60)["p95_duration_ms"].(*int); *got != 95 {
		t.Errorf("p95 of 1..100 = %d", *got)
	}
}

// The period graphs read every call and entry of the period: on 3000
// three-service traces they must stay fast.
func TestTraceGraphs_LargeDBStaysFast(t *testing.T) {
	if testing.Short() {
		t.Skip("timing probe")
	}
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)
	base := time.Now().UTC().Add(-50 * time.Minute)
	pad := strings.Repeat("x", 30<<10)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	ts := func(t time.Time) string { return t.Format(time.RFC3339Nano) }
	for i := 0; i < 3000; i++ {
		tr := fmt.Sprintf("%032x", i)
		st := base.Add(time.Duration(i) * time.Second)
		ins := func(q string, args ...any) {
			if _, err := tx.Exec(q, args...); err != nil {
				t.Fatal(err)
			}
		}
		ins(`INSERT INTO sentry_traces (trace_id) VALUES (?)`, tr)
		for j, svc := range []string{"shop-web", "checkout-api", "payments-worker"} {
			parent := any(nil)
			if j > 0 {
				parent = fmt.Sprintf("c%d-%d", i, j-1)
			}
			ins(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, status, start_ts, end_ts, duration_ms, payload, project_id, service_name, parent_span_id)
				VALUES (?, ?, ?, 'POST /x', 'ok', ?, ?, 100, ?, 'default', ?, ?)`,
				fmt.Sprintf("t%d-%d", i, j), fmt.Sprintf("e%d-%d", i, j), tr, ts(st.Add(time.Duration(j)*time.Millisecond)), ts(st.Add(100*time.Millisecond)), `{"pad":"`+pad+`"}`, svc, parent)
			if j < 2 {
				ins(`INSERT INTO sentry_spans (id, span_id, trace_id, service_name, peer_address, peer_type, start_ts, end_ts, duration_ms, is_error, project_id)
					VALUES (?, ?, ?, ?, 'next', 'http', ?, ?, 90, 0, 'default')`,
					fmt.Sprintf("c%d-%d", i, j), fmt.Sprintf("c%d-%d", i, j), tr, svc, ts(st.Add(time.Duration(j)*time.Millisecond)), ts(st.Add(95*time.Millisecond)))
			}
			for k := 0; k < 5; k++ {
				ins(`INSERT INTO sentry_spans (id, span_id, trace_id, service_name, peer_address, peer_type, start_ts, duration_ms, is_error, project_id)
					VALUES (?, ?, ?, ?, 'db:5432', 'db', ?, 5, 0, 'default')`,
					fmt.Sprintf("d%d-%d-%d", i, j, k), fmt.Sprintf("d%d-%d-%d", i, j, k), tr, svc, ts(st))
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/api/sentry/service-map?window=all", "/api/sentry/flow?window=all"} {
		start := time.Now()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		took := time.Since(start)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", target, w.Code, w.Body)
		}
		t.Logf("%s: %v, %d KB", target, took, w.Body.Len()>>10)
		if took > time.Second {
			t.Errorf("%s took %v", target, took)
		}
	}
}
