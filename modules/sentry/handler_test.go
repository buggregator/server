package sentry

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_Priority(t *testing.T) {
	h := &handler{}
	if h.Priority() != 10 {
		t.Errorf("Priority = %d, want 10", h.Priority())
	}
}

func TestHandler_Match(t *testing.T) {
	h := &handler{}

	tests := []struct {
		name   string
		method string
		path   string
		header map[string]string
		want   bool
	}{
		{"POST with detected type", "POST", "/", map[string]string{"X-Buggregator-Detected-Type": "sentry"}, true},
		{"POST with sentry auth", "POST", "/", map[string]string{"X-Sentry-Auth": "Sentry sentry_key=abc"}, true},
		{"POST to /store", "POST", "/api/123/store", nil, true},
		{"POST to /envelope", "POST", "/api/123/envelope", nil, true},
		{"GET request", "GET", "/api/123/store", nil, false},
		{"POST to random path", "POST", "/random", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.header {
				r.Header.Set(k, v)
			}
			if got := h.Match(r); got != tt.want {
				t.Errorf("Match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Handle_JSON(t *testing.T) {
	h := &handler{}
	payload := `{"event_id":"abc123","message":"test error","level":"error"}`
	r := httptest.NewRequest("POST", "/api/myproject/store", strings.NewReader(payload))

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want %q", inc.Type, "sentry")
	}
	if inc.UUID != "abc123" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "abc123")
	}
	if inc.Project != "myproject" {
		t.Errorf("Project = %q, want %q", inc.Project, "myproject")
	}
}

func TestHandler_Handle_Envelope(t *testing.T) {
	h := &handler{}
	envelope := `{"event_id":"env-uuid"}
{"type":"event","length":25}
{"message":"from envelope"}`

	r := httptest.NewRequest("POST", "/api/proj/store", strings.NewReader(envelope))
	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc.UUID != "env-uuid" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "env-uuid")
	}

	var p map[string]any
	json.Unmarshal(inc.Payload, &p)
	if p["message"] != "from envelope" {
		t.Errorf("payload message = %v", p["message"])
	}
}

func TestDecompress(t *testing.T) {
	t.Run("plain text passthrough", func(t *testing.T) {
		data := []byte("hello world")
		got := decompress(data, "")
		if string(got) != "hello world" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("gzip with header", func(t *testing.T) {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write([]byte("compressed"))
		w.Close()

		got := decompress(buf.Bytes(), "gzip")
		if string(got) != "compressed" {
			t.Errorf("got %q, want %q", got, "compressed")
		}
	})

	t.Run("gzip auto-detect by magic bytes", func(t *testing.T) {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write([]byte("auto-detected"))
		w.Close()

		got := decompress(buf.Bytes(), "")
		if string(got) != "auto-detected" {
			t.Errorf("got %q, want %q", got, "auto-detected")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		got := decompress([]byte{}, "gzip")
		if len(got) != 0 {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestExtractProject(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/myproject/store", "myproject"},
		{"/api/123/envelope", "123"},
		{"/", ""},
		{"/api", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := extractProject(tt.path); got != tt.want {
				t.Errorf("extractProject(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestHandleEnvelope(t *testing.T) {
	h := &handler{} // no db — structured storage silently skipped

	t.Run("valid envelope", func(t *testing.T) {
		body := []byte(`{"event_id":"test-id"}
{"type":"event"}
{"message":"payload data"}`)
		inc, err := h.handleEnvelope(body, "proj")
		if err != nil {
			t.Fatal(err)
		}
		if inc.UUID != "test-id" {
			t.Errorf("UUID = %q, want %q", inc.UUID, "test-id")
		}
		if inc.Project != "proj" {
			t.Errorf("Project = %q, want %q", inc.Project, "proj")
		}
	})

	t.Run("envelope without valid payload returns nil", func(t *testing.T) {
		body := []byte(`{"event_id":"test-id"}
{"type":"event"}
not valid json`)
		inc, err := h.handleEnvelope(body, "")
		if err != nil {
			t.Fatal(err)
		}
		// Envelopes with no valid JSON payload return nil — the handler matched
		// but there's nothing to store in the canonical events table.
		if inc != nil {
			t.Errorf("expected nil incoming for invalid payload, got %+v", inc)
		}
	})
}

// TestHandler_SDKv3_PlainJSON tests Sentry PHP SDK v3 without tracing.
// v3 sends plain JSON to /api/{project}/store with event_id in payload.
func TestHandler_SDKv3_PlainJSON(t *testing.T) {
	h := &handler{}

	// Real payload structure from sentry/sentry 3.22.1 (no tracing → plain JSON to /store).
	payload := `{"event_id":"8f78970685be481994063baacda75197","timestamp":1774960590.323397,"platform":"php","sdk":{"name":"sentry.php","version":"3.22.1"},"level":"error","server_name":"web-01","release":"1.0.0","environment":"production","exception":{"values":[{"type":"RuntimeException","value":"Something went wrong"}]}}`

	r := httptest.NewRequest("POST", "/api/1/store", strings.NewReader(payload))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/3.22.1, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want %q", inc.Type, "sentry")
	}
	if inc.UUID != "8f78970685be481994063baacda75197" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "8f78970685be481994063baacda75197")
	}
	if inc.Project != "1" {
		t.Errorf("Project = %q, want %q", inc.Project, "1")
	}

	// Payload must contain event_id so the frontend can identify the event.
	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["event_id"] != "8f78970685be481994063baacda75197" {
		t.Errorf("payload event_id = %v, want %q", parsed["event_id"], "8f78970685be481994063baacda75197")
	}
	if parsed["level"] != "error" {
		t.Errorf("payload level = %v, want %q", parsed["level"], "error")
	}

	exc, ok := parsed["exception"].(map[string]any)
	if !ok {
		t.Fatal("payload missing exception")
	}
	vals, ok := exc["values"].([]any)
	if !ok || len(vals) == 0 {
		t.Fatal("payload exception.values is empty")
	}
}

// TestHandler_SDKv3_Envelope tests Sentry PHP SDK v3 with tracing enabled.
// v3 sends envelope to /api/{project}/envelope with event_id in BOTH envelope header AND item payload.
func TestHandler_SDKv3_Envelope(t *testing.T) {
	h := &handler{}

	// Real payload structure from sentry/sentry 3.22.1 (tracing enabled → envelope to /envelope).
	envelope := `{"event_id":"3663da0e4e864104a584bf6a053b3ab9","sent_at":"2026-03-31T12:36:30Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"3.22.1"}}
{"type":"event","content_type":"application/json"}
{"event_id":"3663da0e4e864104a584bf6a053b3ab9","timestamp":1774960590.323862,"platform":"php","sdk":{"name":"sentry.php","version":"3.22.1"},"level":"error","server_name":"web-01","release":"1.0.0","environment":"production","exception":{"values":[{"type":"RuntimeException","value":"Something went wrong"}]}}`

	r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(envelope))
	r.Header.Set("Content-Type", "application/x-sentry-envelope")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/3.22.1, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want %q", inc.Type, "sentry")
	}
	if inc.UUID != "3663da0e4e864104a584bf6a053b3ab9" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "3663da0e4e864104a584bf6a053b3ab9")
	}

	// Payload must contain event_id so the frontend can identify the event.
	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["event_id"] != "3663da0e4e864104a584bf6a053b3ab9" {
		t.Errorf("payload event_id = %v, want %q", parsed["event_id"], "3663da0e4e864104a584bf6a053b3ab9")
	}
	if parsed["level"] != "error" {
		t.Errorf("payload level = %v, want %q", parsed["level"], "error")
	}
}

// TestHandler_SDKv4_Envelope tests Sentry PHP SDK v4.
// v4 ALWAYS sends envelope to /api/{project}/envelope.
// CRITICAL DIFFERENCE: v4 puts event_id ONLY in envelope header, NOT in item payload.
// The handler must inject event_id into the payload so the frontend can render the event.
func TestHandler_SDKv4_Envelope(t *testing.T) {
	h := &handler{}

	// Real payload structure from sentry/sentry 4.23.1.
	// Note: item payload does NOT contain event_id — only the envelope header does.
	envelope := `{"sent_at":"2026-03-31T12:31:55Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"4.23.1","packages":[{"name":"composer:sentry/sentry","version":"4.23.1"}]},"event_id":"cf37ef540db04de7ab27dcd8df08034d"}
{"type":"event","content_type":"application/json"}
{"timestamp":1774960315.279867,"platform":"php","sdk":{"name":"sentry.php","version":"4.23.1","packages":[{"name":"composer:sentry/sentry","version":"4.23.1"}]},"level":"error","server_name":"web-01","release":"1.0.0","environment":"production","exception":{"values":[{"type":"RuntimeException","value":"Something went wrong"}]}}`

	r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(envelope))
	r.Header.Set("Content-Type", "application/x-sentry-envelope")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/4.23.1, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want %q", inc.Type, "sentry")
	}
	if inc.UUID != "cf37ef540db04de7ab27dcd8df08034d" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "cf37ef540db04de7ab27dcd8df08034d")
	}

	// CRITICAL: Payload MUST contain event_id even though v4 SDK doesn't include it.
	// The handler should inject event_id from the envelope header.
	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["event_id"] != "cf37ef540db04de7ab27dcd8df08034d" {
		t.Errorf("payload event_id = %v, want %q (should be injected from envelope header)", parsed["event_id"], "cf37ef540db04de7ab27dcd8df08034d")
	}
	if parsed["level"] != "error" {
		t.Errorf("payload level = %v, want %q", parsed["level"], "error")
	}
	if parsed["environment"] != "production" {
		t.Errorf("payload environment = %v, want %q", parsed["environment"], "production")
	}

	// Exception data must be preserved.
	exc, ok := parsed["exception"].(map[string]any)
	if !ok {
		t.Fatal("payload missing exception")
	}
	vals, ok := exc["values"].([]any)
	if !ok || len(vals) == 0 {
		t.Fatal("payload exception.values is empty")
	}
	first := vals[0].(map[string]any)
	if first["type"] != "RuntimeException" {
		t.Errorf("exception type = %v, want %q", first["type"], "RuntimeException")
	}
}

// TestHandler_SDKv4_Envelope_MessageEvent tests a v4 SDK simple message event (no exception).
func TestHandler_SDKv4_Envelope_MessageEvent(t *testing.T) {
	h := &handler{}

	envelope := `{"sent_at":"2026-03-31T12:00:00Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"4.23.1"},"event_id":"aabbccdd11223344aabbccdd11223344"}
{"type":"event","content_type":"application/json"}
{"timestamp":1774960000.0,"platform":"php","sdk":{"name":"sentry.php","version":"4.23.1"},"level":"info","message":"Hello from v4"}`

	r := httptest.NewRequest("POST", "/api/myproject/envelope/", strings.NewReader(envelope))
	r.Header.Set("Content-Type", "application/x-sentry-envelope")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/4.23.1, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.UUID != "aabbccdd11223344aabbccdd11223344" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "aabbccdd11223344aabbccdd11223344")
	}
	if inc.Project != "myproject" {
		t.Errorf("Project = %q, want %q", inc.Project, "myproject")
	}

	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["event_id"] != "aabbccdd11223344aabbccdd11223344" {
		t.Errorf("payload event_id = %v, want %q", parsed["event_id"], "aabbccdd11223344aabbccdd11223344")
	}
	if parsed["message"] != "Hello from v4" {
		t.Errorf("payload message = %v, want %q", parsed["message"], "Hello from v4")
	}
}

// TestHandler_SDKv2_PlainJSON tests Sentry PHP SDK v2.
// v2 always sends plain JSON to /api/{project}/store with ISO 8601 timestamp string.
func TestHandler_SDKv2_PlainJSON(t *testing.T) {
	h := &handler{}

	// Real payload structure from sentry/sentry 2.5.2.
	// Note: timestamp is ISO 8601 string, not float.
	payload := `{"event_id":"197468f20c2b4697a0ae0d236330b0b0","timestamp":"2026-03-31T12:44:44Z","level":"error","platform":"php","sdk":{"name":"sentry.php","version":"2.5.2"},"server_name":"web-01","release":"1.0.0","environment":"production","message":"Something went wrong","contexts":{"os":{"name":"Linux","version":"6.17.0"},"runtime":{"name":"php","version":"8.5.3"}}}`

	r := httptest.NewRequest("POST", "/api/1/store", strings.NewReader(payload))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/2.5.2, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.Type != "sentry" {
		t.Errorf("Type = %q, want %q", inc.Type, "sentry")
	}
	if inc.UUID != "197468f20c2b4697a0ae0d236330b0b0" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "197468f20c2b4697a0ae0d236330b0b0")
	}

	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if parsed["event_id"] != "197468f20c2b4697a0ae0d236330b0b0" {
		t.Errorf("payload event_id = %v, want %q", parsed["event_id"], "197468f20c2b4697a0ae0d236330b0b0")
	}
	// v2 timestamp is ISO string — must be preserved.
	if parsed["timestamp"] != "2026-03-31T12:44:44Z" {
		t.Errorf("payload timestamp = %v, want %q", parsed["timestamp"], "2026-03-31T12:44:44Z")
	}
}

// TestHandler_SDKv2_PlainJSON_WithDB tests that v2 events with ISO timestamps
// are stored correctly in structured storage.
func TestHandler_SDKv2_PlainJSON_WithDB(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	payload := `{"event_id":"v2-db-test","timestamp":"2026-03-31T12:44:44Z","level":"error","platform":"php","sdk":{"name":"sentry.php","version":"2.5.2"},"server_name":"web-01","environment":"production","message":"v2 db test","exception":{"values":[{"type":"RuntimeException","value":"v2 boom"}]}}`

	r := httptest.NewRequest("POST", "/api/1/store", strings.NewReader(payload))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/2.5.2, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil")
	}

	// Verify event was stored in structured storage.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentry_error_events WHERE event_id = 'v2-db-test'").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Errorf("sentry_error_events count = %d, want 1 (v2 ISO timestamp must be handled)", count)
	}

	// Verify exception was stored.
	err = db.QueryRow("SELECT COUNT(*) FROM sentry_exceptions").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Errorf("sentry_exceptions count = %d, want 1", count)
	}
}

// TestHandler_MessageAsObject tests that events with structured message objects
// (message with params, used by v3/v4 with parameterized messages) are handled.
func TestHandler_MessageAsObject(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	// When SDK sends message with params, "message" is an object, not a string.
	payload := `{"event_id":"msg-obj-test","timestamp":1774960590.0,"level":"warning","platform":"php","message":{"message":"User %s performed %s","params":["alice","login"],"formatted":"User alice performed login"}}`

	r := httptest.NewRequest("POST", "/api/1/store", strings.NewReader(payload))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil")
	}
	if inc.UUID != "msg-obj-test" {
		t.Errorf("UUID = %q, want %q", inc.UUID, "msg-obj-test")
	}

	// Verify event was stored in structured storage (should not fail on object message).
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sentry_error_events WHERE event_id = 'msg-obj-test'").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Errorf("sentry_error_events count = %d, want 1 (object message must not break structured storage)", count)
	}
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToPreview extracts fields", func(t *testing.T) {
		payload := json.RawMessage(`{
			"message":"test error",
			"level":"error",
			"platform":"php",
			"environment":"production",
			"server_name":"web-1",
			"event_id":"abc",
			"extra_field":"should be excluded"
		}`)
		preview, err := m.ToPreview(payload)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		json.Unmarshal(preview, &p)
		if p["message"] != "test error" {
			t.Error("missing message")
		}
		if p["level"] != "error" {
			t.Error("missing level")
		}
		if _, ok := p["extra_field"]; ok {
			t.Error("extra_field should not be in preview")
		}
	})

	t.Run("ToSearchableText", func(t *testing.T) {
		payload := json.RawMessage(`{"message":"test","level":"error","environment":"prod"}`)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "test") || !strings.Contains(text, "error") {
			t.Errorf("text = %q, expected to contain message and level", text)
		}
	})
}
