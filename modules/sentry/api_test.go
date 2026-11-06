package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// seedTestData populates the test DB with sample data for API tests.
func seedTestData(t *testing.T, mux *http.ServeMux) {
	t.Helper()
	db := setupTestDB(t)
	registerAPI(mux, db)

	// Insert error events.
	for i, evt := range []struct {
		eventID string
		level   string
		excType string
		excVal  string
		traceID string
	}{
		{"evt-1", "error", "RuntimeException", "Something broke", "trace-abc"},
		{"evt-2", "error", "RuntimeException", "Something broke", "trace-abc"},
		{"evt-3", "warning", "LogicException", "Bad logic", ""},
	} {
		payload, _ := json.Marshal(map[string]any{
			"event_id":    evt.eventID,
			"level":       evt.level,
			"environment": "production",
			"exception":   map[string]any{"values": []map[string]any{{"type": evt.excType, "value": evt.excVal}}},
		})
		var ev ErrorEvent
		json.Unmarshal(payload, &ev)
		if evt.traceID != "" {
			ev.Contexts = &Contexts{Trace: &TraceContext{TraceID: evt.traceID, SpanID: "span-" + evt.eventID}}
		}
		_, err := storeErrorEvent(db, &ev, payload, "default")
		if err != nil {
			t.Fatalf("seed error event %d: %v", i, err)
		}
	}

	// Insert a transaction with spans.
	txnPayload, _ := json.Marshal(map[string]any{
		"event_id":        "txn-1",
		"type":            "transaction",
		"transaction":     "GET /api/users",
		"start_timestamp": 1678272500.0,
		"timestamp":       1678272503.0,
		"contexts":        map[string]any{"trace": map[string]any{"trace_id": "trace-abc", "span_id": "root"}},
		"spans": []map[string]any{
			{"span_id": "s1", "trace_id": "trace-abc", "op": "db.query", "description": "SELECT *", "start_timestamp": 1678272501.0, "timestamp": 1678272502.0, "status": "ok", "data": map[string]string{"server.address": "mysql:3306"}},
			{"span_id": "s2", "trace_id": "trace-abc", "op": "http.client", "description": "stripe.com", "start_timestamp": 1678272502.0, "timestamp": 1678272502.5, "status": "internal_error", "data": map[string]string{"http.url": "https://stripe.com/charge"}},
		},
	})
	var txn Transaction
	json.Unmarshal(txnPayload, &txn)
	storeTransaction(db, &txn, txnPayload, "")

	// Insert logs.
	storeLogs(db, []LogRecord{
		{TraceID: "trace-abc", Level: "info", Body: "Request started", SeverityNumber: 9},
		{TraceID: "trace-abc", Level: "warning", Body: "Slow query", SeverityNumber: 13},
		{Level: "error", Body: "Connection refused", SeverityNumber: 17},
	}, "")

	// Insert metrics.
	storeMetrics(db, []MetricRecord{
		{TraceID: "trace-abc", Name: "http.client.duration", Type: "distribution", Value: 12.5, Unit: "millisecond", Timestamp: "1678272501"},
		{TraceID: "trace-abc", Name: "http.client.duration", Type: "distribution", Value: 40, Unit: "millisecond", Timestamp: "1678272502"},
		{Name: "order.status", Type: "gauge", Value: 3, Timestamp: "1678272503"},
	}, "")
}

func TestAPIExceptionsChronological(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/exceptions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 3 {
		t.Fatalf("data length = %d, want 3", len(data))
	}

	// First item should be most recent.
	first := data[0].(map[string]any)
	if first["level"] == nil {
		t.Error("expected level field")
	}

	// Check occurrence_count for RuntimeException events.
	for _, item := range data {
		m := item.(map[string]any)
		if m["exception_type"] == "RuntimeException" {
			count := int(m["occurrence_count"].(float64))
			if count != 2 {
				t.Errorf("occurrence_count = %d, want 2 for RuntimeException", count)
			}
		}
	}

	meta := resp["meta"].(map[string]any)
	if int(meta["total"].(float64)) != 3 {
		t.Errorf("meta.total = %v, want 3", meta["total"])
	}
}

func TestAPIExceptionsGrouped(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/exceptions?grouped=true", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("data length = %d, want 2 (2 distinct fingerprints)", len(data))
	}

	// First group should have count 2 (RuntimeException).
	first := data[0].(map[string]any)
	if int(first["count"].(float64)) != 2 {
		t.Errorf("first group count = %v, want 2", first["count"])
	}
}

func TestAPIExceptionsFilterLevel(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/exceptions?level=warning", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("data length = %d, want 1 (only warning)", len(data))
	}
}

func TestAPIExceptionDetail(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	// Get the first event's ID.
	req := httptest.NewRequest("GET", "/api/sentry/exceptions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	items := listResp["data"].([]any)

	// Find one with trace_id.
	var eventWithTrace map[string]any
	for _, item := range items {
		m := item.(map[string]any)
		if m["trace_id"] != nil && m["trace_id"] != "" {
			eventWithTrace = m
			break
		}
	}
	if eventWithTrace == nil {
		t.Fatal("no event with trace_id found")
	}

	id := eventWithTrace["id"].(string)
	req = httptest.NewRequest("GET", "/api/sentry/exceptions/"+id, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var detail map[string]any
	json.Unmarshal(w.Body.Bytes(), &detail)

	if detail["exceptions"] == nil {
		t.Error("expected exceptions array")
	}
	if detail["trace_summary"] == nil {
		t.Error("expected trace_summary for event with trace_id")
	}

	ts := detail["trace_summary"].(map[string]any)
	if ts["transaction_name"] != "GET /api/users" {
		t.Errorf("trace_summary.transaction_name = %v", ts["transaction_name"])
	}
	if ts["preview_spans"] == nil {
		t.Error("expected preview_spans in trace_summary")
	}
}

// TestAPIExceptionDetailByEventID covers Fix B: the detail endpoint must resolve
// by event_id (the key the list and the frontend route use) and still return the
// populated exceptions/breadcrumbs keyed off the internal id.
func TestAPIExceptionDetailByEventID(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/exceptions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var listResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &listResp)
	first := listResp["data"].([]any)[0].(map[string]any)
	eventID := first["event_id"].(string)
	internalID := first["id"].(string)

	req = httptest.NewRequest("GET", "/api/sentry/exceptions/"+eventID, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200 for event_id lookup", w.Code)
	}

	var detail map[string]any
	json.Unmarshal(w.Body.Bytes(), &detail)

	if detail["id"] != internalID {
		t.Errorf("detail id = %v, want internal id %q", detail["id"], internalID)
	}
	excs, ok := detail["exceptions"].([]any)
	if !ok || len(excs) == 0 {
		t.Errorf("exceptions empty for event_id lookup, want populated (got %v)", detail["exceptions"])
	}
}

func TestAPIExceptionDetailNotFound(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/exceptions/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPITracesList(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/traces", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("data length = %d, want 1", len(data))
	}

	trace := data[0].(map[string]any)
	if trace["transaction_name"] != "GET /api/users" {
		t.Errorf("transaction_name = %v", trace["transaction_name"])
	}
	if trace["preview_spans"] == nil {
		t.Error("expected preview_spans")
	}
	spans := trace["preview_spans"].([]any)
	if len(spans) < 1 {
		t.Error("expected at least 1 preview span")
	}
}

func TestAPITraceDetail(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/traces/trace-abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["trace_id"] != "trace-abc" {
		t.Errorf("trace_id = %v", resp["trace_id"])
	}

	txn := resp["transaction"].(map[string]any)
	if txn["transaction_name"] != "GET /api/users" {
		t.Errorf("transaction_name = %v", txn["transaction_name"])
	}

	spans := resp["spans"].([]any)
	if len(spans) != 2 {
		t.Errorf("spans length = %d, want 2", len(spans))
	}

	// Verify related errors.
	relatedErrors := resp["related_errors"].([]any)
	if len(relatedErrors) != 2 {
		t.Errorf("related_errors = %d, want 2", len(relatedErrors))
	}

	// Verify related log count.
	logCount := int(resp["related_log_count"].(float64))
	if logCount != 2 {
		t.Errorf("related_log_count = %d, want 2", logCount)
	}
}

func TestAPITraceDetailNotFound(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/traces/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestAPILogsList(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/logs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 3 {
		t.Fatalf("data length = %d, want 3", len(data))
	}
}

func TestAPILogsFilterByTraceID(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/logs?trace_id=trace-abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 2 {
		t.Fatalf("data length = %d, want 2 (only trace-abc)", len(data))
	}
}

func TestAPIMetricsList(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/metrics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].([]any)
	if len(data) != 3 {
		t.Fatalf("data length = %d, want 3", len(data))
	}

	newest := data[0].(map[string]any)
	if newest["name"] != "order.status" || newest["type"] != "gauge" || newest["value"] != 3.0 {
		t.Errorf("newest metric = %v, want the order.status gauge", newest)
	}
}

func TestAPIMetricsFilter(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	tests := []struct {
		query string
		want  int
	}{
		{"name=http.client.duration", 2},
		{"type=gauge", 1},
		{"trace_id=trace-abc", 2},
		{"name=http.client.duration&type=gauge", 0},
		{"search=client", 2},
		{"search=CLIENT.dur", 2},
		{"search=order", 1},
		{"search=%25", 0},
		{"search=http_client", 0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/sentry/metrics?"+tt.query, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			var resp map[string]any
			json.Unmarshal(w.Body.Bytes(), &resp)

			data := resp["data"].([]any)
			if len(data) != tt.want {
				t.Errorf("data length = %d, want %d", len(data), tt.want)
			}
		})
	}
}

func TestAPIServiceMap(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/service-map", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should have nodes and edges from the span data.
	if resp["nodes"] == nil {
		t.Error("expected nodes")
	}
	if resp["edges"] == nil {
		t.Error("expected edges")
	}
	if resp["window_minutes"] == nil {
		t.Error("expected window_minutes")
	}
}

func TestAPICounts(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	req := httptest.NewRequest("GET", "/api/sentry/counts", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if int(resp["exceptions"].(float64)) != 3 {
		t.Errorf("exceptions = %v, want 3", resp["exceptions"])
	}
	if int(resp["traces"].(float64)) != 1 {
		t.Errorf("traces = %v, want 1", resp["traces"])
	}
	if int(resp["logs"].(float64)) != 3 {
		t.Errorf("logs = %v, want 3", resp["logs"])
	}
	if int(resp["metrics"].(float64)) != 3 {
		t.Errorf("metrics = %v, want 3", resp["metrics"])
	}
}

func TestAPIEmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)

	endpoints := []string{
		"/api/sentry/exceptions",
		"/api/sentry/exceptions?grouped=true",
		"/api/sentry/traces",
		"/api/sentry/logs",
		"/api/sentry/service-map",
		"/api/sentry/counts",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req := httptest.NewRequest("GET", ep, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("status = %d, want 200 for %s", w.Code, ep)
			}
		})
	}
}

// pagedField fetches the given URL, which has a query already, page by page, one row each, and returns the field of every row.
func pagedField(t *testing.T, mux *http.ServeMux, url, field string, pages int) []string {
	t.Helper()
	var got []string
	for page := 1; page <= pages; page++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("%s&limit=1&page=%d", url, page), nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		for _, row := range resp["data"].([]any) {
			got = append(got, row.(map[string]any)[field].(string))
		}
	}
	return got
}

// With a filter SQLite sorts instead of walking the timestamp index backwards, and
// rows of the same timestamp came in insertion order, so pages could repeat or skip rows.
func TestAPIMetricsSameTimestampNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)

	storeMetrics(db, []MetricRecord{
		{Name: "first", Type: "counter", TraceID: "trace-abc", Value: 1, Timestamp: "1678272501"},
		{Name: "second", Type: "counter", TraceID: "trace-abc", Value: 1, Timestamp: "1678272501"},
		{Name: "third", Type: "counter", TraceID: "trace-abc", Value: 1, Timestamp: "1678272501"},
	}, "")

	// trace_id makes SQLite use its index and sort; without the rowid
	// tie-breaker the rows come back in insertion order.
	got := pagedField(t, mux, "/api/sentry/metrics?trace_id=trace-abc", "name", 3)
	want := []string{"third", "second", "first"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

func TestAPILogsSameTimestampNewestFirst(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)

	var logs []LogRecord
	json.Unmarshal([]byte(`[
		{"trace_id": "trace-abc", "level": "info", "body": "first", "timestamp": 1678272501},
		{"trace_id": "trace-abc", "level": "info", "body": "second", "timestamp": 1678272501},
		{"trace_id": "trace-abc", "level": "info", "body": "third", "timestamp": 1678272501}
	]`), &logs)
	storeLogs(db, logs, "")

	got := pagedField(t, mux, "/api/sentry/logs?trace_id=trace-abc", "body", 3)
	want := []string{"third", "second", "first"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

func TestAPILogsFilterByLevels(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	tests := []struct {
		query string
		want  int
	}{
		{"level=info", 1},
		{"level=warning,error", 2},
		{"level=warn,warning", 1},
		{"level=warn,%20warning", 1},
		{"level=info,,%20", 1},
		{"level=fatal", 0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/sentry/logs?"+tt.query, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			var resp map[string]any
			json.Unmarshal(w.Body.Bytes(), &resp)

			data := resp["data"].([]any)
			if len(data) != tt.want {
				t.Errorf("data length = %d, want %d", len(data), tt.want)
			}
		})
	}
}

// The exceptions list could not be narrowed to a project: the column is filled
// in on write, but the handler had no reader for it, so selecting a project in
// the UI showed every project's errors. Same for environment, and for the
// period there was no filter at all.
func TestAPIExceptionsFilters(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)

	seed := func(eventID, env, project string) {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{
			"event_id":    eventID,
			"level":       "error",
			"environment": env,
			"exception":   map[string]any{"values": []map[string]any{{"type": "RuntimeException", "value": eventID}}},
		})
		var ev ErrorEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			t.Fatal(err)
		}
		if _, err := storeErrorEvent(db, &ev, payload, project); err != nil {
			t.Fatalf("seed %s: %v", eventID, err)
		}
	}
	seed("evt-prod-a", "production", "shop")
	seed("evt-prod-b", "production", "shop")
	seed("evt-stage", "staging", "shop")
	seed("evt-other", "production", "billing")

	count := func(query string) int {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/sentry/exceptions"+query, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status = %d for %q", w.Code, query)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		return len(resp["data"].([]any))
	}

	cases := map[string]struct {
		query string
		want  int
	}{
		"no filters":         {"", 4},
		"by project":         {"?project=shop", 3},
		"by environment":     {"?environment=staging", 1},
		"project and env":    {"?project=shop&environment=production", 2},
		"grouped by project": {"?grouped=true&project=billing", 1},
		"within window":      {"?window=24h", 4},
		"outside window":     {"?from=2030-01-01", 0},
		"grouped in window":  {"?grouped=true&from=2030-01-01", 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := count(tc.query); got != tc.want {
				t.Errorf("%q returned %d events, want %d", tc.query, got, tc.want)
			}
		})
	}
}

// The tab counters counted the whole database, so they disagreed with the lists
// they label as soon as a project or a period was selected.
func TestAPICountsFollowFilters(t *testing.T) {
	mux := http.NewServeMux()
	seedTestData(t, mux)

	counts := func(query string) map[string]any {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/sentry/counts"+query, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	if got := counts("?project=default")["exceptions"]; int(got.(float64)) != 3 {
		t.Errorf("exceptions for the seeded project = %v, want 3", got)
	}
	if got := counts("?project=nonexistent")["exceptions"]; int(got.(float64)) != 0 {
		t.Errorf("exceptions for an unknown project = %v, want 0", got)
	}
	if got := counts("?from=2030-01-01")["exceptions"]; int(got.(float64)) != 0 {
		t.Errorf("exceptions in a future window = %v, want 0", got)
	}
}

// A standard DSN (/api/1/...) carries no project name: the main events table
// stores the event under "default", and the sentry tables must use the same
// key, or ?project=default (what the UI sends) finds nothing.
func TestAPIStandardDSNProjectIsDefault(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)
	h := &handler{db: db}

	// A row stored before the fix, with no project.
	if _, err := db.Exec(`INSERT INTO sentry_error_events (id, event_id, project_id, level, fingerprint, payload)
		VALUES ('old-row', 'old-1', NULL, 'error', 'fp-old', '{}')`); err != nil {
		t.Fatal(err)
	}
	sqlBytes, err := migrations.ReadFile("migrations/2024_01_01_000028_default_project_for_sentry_errors.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		t.Fatal(err)
	}

	// A new event after the migration: stored under "default" by the code.
	r := httptest.NewRequest("POST", "/api/1/store", strings.NewReader(
		`{"event_id":"dsn-1","level":"error","exception":{"values":[{"type":"RuntimeException","value":"boom"}]}}`))
	if _, err := h.Handle(r); err != nil {
		t.Fatal(err)
	}

	get := func(url string) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	if got := get("/api/sentry/counts?project=default")["exceptions"]; got != float64(2) {
		t.Errorf("counts exceptions for project=default = %v, want 2", got)
	}
	data, _ := get("/api/sentry/exceptions?project=default")["data"].([]any)
	if len(data) != 2 {
		t.Errorf("exceptions for project=default = %d, want 2", len(data))
	}
}

// The metrics tab counter and the metrics list take the same period.
func TestAPIMetricsWindowMatchesCount(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db)

	storeMetrics(db, []MetricRecord{
		{Name: "old", Type: "counter", Value: 1, Timestamp: "1678272501"},   // 2023
		{Name: "new-1", Type: "counter", Value: 1, Timestamp: "1893456000"}, // 2030
		{Name: "new-2", Type: "counter", Value: 1, Timestamp: "1893456001"},
	}, "")

	get := func(url string) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	for _, q := range []string{"from=2029-01-01", "to=2024-01-01", "from=2029-01-01&to=2030-01-01T00:00:00Z"} {
		count := get("/api/sentry/counts?" + q)["metrics"]
		data, _ := get("/api/sentry/metrics?" + q)["data"].([]any)
		if count != float64(len(data)) {
			t.Errorf("%s: counts metrics = %v, list = %d", q, count, len(data))
		}
	}
	if data, _ := get("/api/sentry/metrics?from=2029-01-01")["data"].([]any); len(data) != 2 {
		t.Errorf("metrics from 2029 = %d, want 2", len(data))
	}
}
