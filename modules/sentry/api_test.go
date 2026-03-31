package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	storeTransaction(db, &txn, txnPayload)

	// Insert logs.
	storeLogs(db, []LogRecord{
		{TraceID: "trace-abc", Level: "info", Body: "Request started", SeverityNumber: 9},
		{TraceID: "trace-abc", Level: "warning", Body: "Slow query", SeverityNumber: 13},
		{Level: "error", Body: "Connection refused", SeverityNumber: 17},
	})
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
