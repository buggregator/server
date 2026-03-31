package sentry

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/buggregator/go-buggregator/internal/storage"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Run sentry migrations.
	migrator := storage.NewMigrator(db)
	if err := migrator.AddFromFS("sentry", migrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Run(); err != nil {
		t.Fatal(err)
	}

	return db
}

func TestStoreErrorEvent(t *testing.T) {
	db := setupTestDB(t)

	payload := json.RawMessage(`{
		"event_id": "abc123",
		"level": "error",
		"platform": "php",
		"environment": "production",
		"server_name": "web-1",
		"transaction": "GET /users",
		"exception": {
			"values": [
				{
					"type": "RuntimeException",
					"value": "Something went wrong",
					"mechanism": {"type": "generic", "handled": false},
					"stacktrace": {"frames": [{"filename": "app.php", "lineno": 42}]}
				},
				{
					"type": "LogicException",
					"value": "Root cause",
					"mechanism": {"type": "chained", "handled": true}
				}
			]
		},
		"breadcrumbs": {
			"values": [
				{"type": "http", "category": "request", "level": "info", "message": "GET /api"},
				{"type": "query", "category": "db", "level": "debug", "message": "SELECT *"}
			]
		},
		"contexts": {
			"trace": {"trace_id": "aabbccdd", "span_id": "1234"}
		}
	}`)

	var ev ErrorEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		t.Fatal(err)
	}

	id, err := storeErrorEvent(db, &ev, payload, "myproject")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Verify sentry_error_events row.
	var level, fingerprint, traceID string
	var handled int
	err = db.QueryRow(
		`SELECT level, fingerprint, trace_id, handled FROM sentry_error_events WHERE event_id = ?`, "abc123",
	).Scan(&level, &fingerprint, &traceID, &handled)
	if err != nil {
		t.Fatal(err)
	}
	if level != "error" {
		t.Errorf("level = %q, want 'error'", level)
	}
	if fingerprint == "" {
		t.Error("fingerprint should not be empty")
	}
	if traceID != "aabbccdd" {
		t.Errorf("trace_id = %q, want 'aabbccdd'", traceID)
	}
	if handled != 0 {
		t.Errorf("handled = %d, want 0 (false)", handled)
	}

	// Verify exceptions (2 rows).
	var excCount int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_exceptions WHERE error_event_id = ?`, id).Scan(&excCount)
	if excCount != 2 {
		t.Errorf("exception count = %d, want 2", excCount)
	}

	// Verify exception positions.
	var excType string
	db.QueryRow(`SELECT exception_type FROM sentry_exceptions WHERE error_event_id = ? AND position = 0`, id).Scan(&excType)
	if excType != "RuntimeException" {
		t.Errorf("exception[0].type = %q, want 'RuntimeException'", excType)
	}
	db.QueryRow(`SELECT exception_type FROM sentry_exceptions WHERE error_event_id = ? AND position = 1`, id).Scan(&excType)
	if excType != "LogicException" {
		t.Errorf("exception[1].type = %q, want 'LogicException'", excType)
	}

	// Verify breadcrumbs (2 rows).
	var bcCount int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_breadcrumbs WHERE error_event_id = ?`, id).Scan(&bcCount)
	if bcCount != 2 {
		t.Errorf("breadcrumb count = %d, want 2", bcCount)
	}
}

func TestStoreErrorEvent_SameFingerprint(t *testing.T) {
	db := setupTestDB(t)

	payload := json.RawMessage(`{
		"event_id": "evt-1",
		"level": "error",
		"exception": {"values": [{"type": "RuntimeException", "value": "Same error"}]}
	}`)

	var ev ErrorEvent
	json.Unmarshal(payload, &ev)
	storeErrorEvent(db, &ev, payload, "")

	// Send same exception type+value with different event_id.
	payload2 := json.RawMessage(`{
		"event_id": "evt-2",
		"level": "error",
		"exception": {"values": [{"type": "RuntimeException", "value": "Same error"}]}
	}`)
	json.Unmarshal(payload2, &ev)
	ev.EventID = "evt-2"
	storeErrorEvent(db, &ev, payload2, "")

	payload3 := json.RawMessage(`{
		"event_id": "evt-3",
		"level": "error",
		"exception": {"values": [{"type": "RuntimeException", "value": "Same error"}]}
	}`)
	json.Unmarshal(payload3, &ev)
	ev.EventID = "evt-3"
	storeErrorEvent(db, &ev, payload3, "")

	// All 3 should have the same fingerprint.
	rows, _ := db.Query(`SELECT fingerprint FROM sentry_error_events ORDER BY event_id`)
	defer rows.Close()

	var fingerprints []string
	for rows.Next() {
		var fp string
		rows.Scan(&fp)
		fingerprints = append(fingerprints, fp)
	}

	if len(fingerprints) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(fingerprints))
	}
	if fingerprints[0] != fingerprints[1] || fingerprints[1] != fingerprints[2] {
		t.Errorf("fingerprints should all match: %v", fingerprints)
	}
}

func TestStoreTransaction(t *testing.T) {
	db := setupTestDB(t)

	payload := json.RawMessage(`{
		"event_id": "txn-1",
		"type": "transaction",
		"transaction": "GET /api/users",
		"op": "http.server",
		"status": "ok",
		"start_timestamp": 1678272500.0,
		"timestamp": 1678272505.844,
		"environment": "production",
		"contexts": {"trace": {"trace_id": "aabbccdd11223344", "span_id": "root-span"}},
		"spans": [
			{
				"span_id": "span-1",
				"parent_span_id": "root-span",
				"trace_id": "aabbccdd11223344",
				"op": "db.query",
				"description": "SELECT * FROM users",
				"status": "ok",
				"start_timestamp": 1678272501.0,
				"timestamp": 1678272502.0,
				"data": {"server.address": "db.example.com", "db.name": "mydb"}
			},
			{
				"span_id": "span-2",
				"parent_span_id": "root-span",
				"trace_id": "aabbccdd11223344",
				"op": "http.client",
				"description": "POST https://api.example.com/notify",
				"status": "internal_error",
				"start_timestamp": 1678272503.0,
				"timestamp": 1678272504.5,
				"data": {"http.url": "https://api.example.com/notify"}
			}
		]
	}`)

	var txn Transaction
	if err := json.Unmarshal(payload, &txn); err != nil {
		t.Fatal(err)
	}

	id, err := storeTransaction(db, &txn, payload)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Verify sentry_traces.
	var spanCount, errorCount int
	err = db.QueryRow(`SELECT span_count, error_count FROM sentry_traces WHERE trace_id = ?`, "aabbccdd11223344").Scan(&spanCount, &errorCount)
	if err != nil {
		t.Fatal(err)
	}
	if spanCount != 2 {
		t.Errorf("span_count = %d, want 2", spanCount)
	}
	if errorCount != 1 {
		t.Errorf("error_count = %d, want 1", errorCount)
	}

	// Verify sentry_transactions.
	var txnName string
	var durationMS int
	err = db.QueryRow(`SELECT transaction_name, duration_ms FROM sentry_transactions WHERE event_id = ?`, "txn-1").Scan(&txnName, &durationMS)
	if err != nil {
		t.Fatal(err)
	}
	if txnName != "GET /api/users" {
		t.Errorf("transaction_name = %q, want 'GET /api/users'", txnName)
	}
	if durationMS != 5844 {
		t.Errorf("duration_ms = %d, want 5844", durationMS)
	}

	// Verify sentry_spans.
	var spansCount int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_spans WHERE trace_id = ?`, "aabbccdd11223344").Scan(&spansCount)
	if spansCount != 2 {
		t.Errorf("spans count = %d, want 2", spansCount)
	}

	// Verify db span classification.
	var peerType, peerAddress string
	db.QueryRow(`SELECT peer_type, peer_address FROM sentry_spans WHERE span_id = ?`, "span-1").Scan(&peerType, &peerAddress)
	if peerType != "db" {
		t.Errorf("span-1 peer_type = %q, want 'db'", peerType)
	}
	if peerAddress != "db.example.com" {
		t.Errorf("span-1 peer_address = %q, want 'db.example.com'", peerAddress)
	}

	// Verify http span classification.
	var httpPeerType, httpPeerAddress string
	var isError int
	db.QueryRow(`SELECT peer_type, peer_address, is_error FROM sentry_spans WHERE span_id = ?`, "span-2").Scan(&httpPeerType, &httpPeerAddress, &isError)
	if httpPeerType != "http" {
		t.Errorf("span-2 peer_type = %q, want 'http'", httpPeerType)
	}
	if httpPeerAddress != "api.example.com" {
		t.Errorf("span-2 peer_address = %q, want 'api.example.com'", httpPeerAddress)
	}
	if isError != 1 {
		t.Errorf("span-2 is_error = %d, want 1", isError)
	}
}

func TestStoreLogs(t *testing.T) {
	db := setupTestDB(t)

	logs := []LogRecord{
		{TraceID: "trace-1", SpanID: "span-1", Level: "info", SeverityNumber: 9, Body: "Request started"},
		{TraceID: "trace-1", Level: "warning", SeverityNumber: 13, Body: "Slow query detected"},
		{Level: "error", SeverityNumber: 17, Body: "Connection refused"},
		{Level: "debug", Body: "Cache hit"},
		{Level: "info", Body: "Request completed"},
	}

	err := storeLogs(db, logs)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_logs`).Scan(&count)
	if count != 5 {
		t.Errorf("log count = %d, want 5", count)
	}

	// Verify trace-linked log.
	var traceID, level string
	db.QueryRow(`SELECT trace_id, level FROM sentry_logs WHERE body = ?`, "Request started").Scan(&traceID, &level)
	if traceID != "trace-1" {
		t.Errorf("trace_id = %q, want 'trace-1'", traceID)
	}
	if level != "info" {
		t.Errorf("level = %q, want 'info'", level)
	}
}

func TestFingerprint(t *testing.T) {
	t.Run("exception event", func(t *testing.T) {
		ev := &ErrorEvent{
			Exception: &ExceptionList{
				Values: []ExceptionValue{
					{Type: "RuntimeException", Value: "Something went wrong"},
				},
			},
		}
		fp := computeFingerprint(ev)
		if len(fp) != 16 {
			t.Errorf("fingerprint length = %d, want 16", len(fp))
		}
	})

	t.Run("message event", func(t *testing.T) {
		ev := &ErrorEvent{Message: "Something happened"}
		fp := computeFingerprint(ev)
		if len(fp) != 16 {
			t.Errorf("fingerprint length = %d, want 16", len(fp))
		}
	})

	t.Run("same exception produces same fingerprint", func(t *testing.T) {
		ev1 := &ErrorEvent{Exception: &ExceptionList{Values: []ExceptionValue{{Type: "Err", Value: "msg"}}}}
		ev2 := &ErrorEvent{Exception: &ExceptionList{Values: []ExceptionValue{{Type: "Err", Value: "msg"}}}}
		if computeFingerprint(ev1) != computeFingerprint(ev2) {
			t.Error("same exception should produce same fingerprint")
		}
	})

	t.Run("different exception produces different fingerprint", func(t *testing.T) {
		ev1 := &ErrorEvent{Exception: &ExceptionList{Values: []ExceptionValue{{Type: "ErrA", Value: "msg"}}}}
		ev2 := &ErrorEvent{Exception: &ExceptionList{Values: []ExceptionValue{{Type: "ErrB", Value: "msg"}}}}
		if computeFingerprint(ev1) == computeFingerprint(ev2) {
			t.Error("different exceptions should produce different fingerprints")
		}
	})
}

func TestSpanClassifier(t *testing.T) {
	tests := []struct {
		name         string
		span         RawSpan
		wantType     string
		wantAddress  string
	}{
		{
			name:        "db.query",
			span:        RawSpan{Op: "db.query", Data: map[string]string{"server.address": "db.host.com"}},
			wantType:    "db",
			wantAddress: "db.host.com",
		},
		{
			name:        "db with db.name fallback",
			span:        RawSpan{Op: "db.redis", Data: map[string]string{"db.name": "cache-0"}},
			wantType:    "db",
			wantAddress: "cache-0",
		},
		{
			name:        "http.client",
			span:        RawSpan{Op: "http.client", Data: map[string]string{"http.url": "https://api.example.com:8080/path"}},
			wantType:    "http",
			wantAddress: "api.example.com:8080",
		},
		{
			name:        "cache.get",
			span:        RawSpan{Op: "cache.get", Data: map[string]string{"server.address": "redis.local"}},
			wantType:    "cache",
			wantAddress: "redis.local",
		},
		{
			name:        "queue.publish",
			span:        RawSpan{Op: "queue.publish", Data: map[string]string{"messaging.destination": "orders-queue"}},
			wantType:    "queue",
			wantAddress: "orders-queue",
		},
		{
			name:     "unknown op",
			span:     RawSpan{Op: "custom.op"},
			wantType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerType, peerAddress := classifySpan(tt.span)
			if peerType != tt.wantType {
				t.Errorf("peerType = %q, want %q", peerType, tt.wantType)
			}
			if peerAddress != tt.wantAddress {
				t.Errorf("peerAddress = %q, want %q", peerAddress, tt.wantAddress)
			}
		})
	}
}

func TestEnvelopeParser(t *testing.T) {
	t.Run("multi-item envelope", func(t *testing.T) {
		body := []byte(`{"event_id":"env-id","sent_at":"2023-01-01T00:00:00Z"}
{"type":"event","length":0}
{"event_id":"evt-1","message":"test"}
{"type":"session","length":0}
{"sid":"sess-1"}`)

		header, items, err := parseEnvelopeItems(body)
		if err != nil {
			t.Fatal(err)
		}
		if header.EventID != "env-id" {
			t.Errorf("header.EventID = %q, want 'env-id'", header.EventID)
		}
		if len(items) != 2 {
			t.Fatalf("items count = %d, want 2", len(items))
		}
		if items[0].Type != "event" {
			t.Errorf("items[0].Type = %q, want 'event'", items[0].Type)
		}
		if items[1].Type != "session" {
			t.Errorf("items[1].Type = %q, want 'session'", items[1].Type)
		}
	})

	t.Run("empty envelope", func(t *testing.T) {
		_, items, err := parseEnvelopeItems([]byte{})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 0 {
			t.Errorf("expected no items, got %d", len(items))
		}
	})
}

func TestSessionItemDiscarded(t *testing.T) {
	h := &handler{}

	body := []byte(`{"event_id":"sess-env"}
{"type":"session","length":0}
{"sid":"test-session"}`)

	inc, err := h.handleEnvelope(body, "proj")
	if err != nil {
		t.Fatal(err)
	}

	// Should still return a valid response (no error).
	if inc == nil {
		t.Fatal("expected non-nil incoming event")
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want 'sentry'", inc.Type)
	}
}

func TestHandlerWithDB(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	body := []byte(`{"event_id":"full-test"}
{"type":"event","length":0}
{"event_id":"full-test","level":"warning","message":"test message","exception":{"values":[{"type":"TestException","value":"test"}]}}`)

	inc, err := h.handleEnvelope(body, "testproject")
	if err != nil {
		t.Fatal(err)
	}

	if inc.UUID != "full-test" {
		t.Errorf("UUID = %q, want 'full-test'", inc.UUID)
	}

	// Verify structured data was stored.
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_error_events`).Scan(&count)
	if count != 1 {
		t.Errorf("error events = %d, want 1", count)
	}

	var excCount int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_exceptions`).Scan(&excCount)
	if excCount != 1 {
		t.Errorf("exceptions = %d, want 1", excCount)
	}
}
