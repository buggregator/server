package sentry

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
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
		// Sentry PHP SDK 4.x posts to /api/{id}/envelope/ with a trailing slash.
		{"POST to /envelope/ trailing slash", "POST", "/api/123/envelope/", nil, true},
		{"POST to /store/ trailing slash", "POST", "/api/123/store/", nil, true},
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
	payload := `{"message":"from envelope"}`
	envelope := "{\"event_id\":\"env-uuid\"}\n" +
		"{\"type\":\"event\",\"length\":" + strconv.Itoa(len(payload)) + "}\n" +
		payload

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

func TestHandler_Handle_EnvelopeMultilinePayload(t *testing.T) {
	// Payload contains literal newlines — must rely on length, not newline split.
	h := &handler{}
	payload := "{\n  \"message\": \"multi\\nline\"\n}"
	envelope := "{\"event_id\":\"env-uuid\"}\n" +
		"{\"type\":\"event\",\"length\":" + strconv.Itoa(len(payload)) + "}\n" +
		payload

	r := httptest.NewRequest("POST", "/api/proj/store", strings.NewReader(envelope))
	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc == nil {
		t.Fatal("incoming = nil")
	}
	var p map[string]any
	if err := json.Unmarshal(inc.Payload, &p); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if p["message"] != "multi\nline" {
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

	// sentry-php sends Content-Encoding: zstd by default since 4.x. Without zstd
	// support the body stays compressed, fails to parse as an envelope and the
	// event is dropped silently — the request still answers 200.
	t.Run("zstd with header", func(t *testing.T) {
		var buf bytes.Buffer
		w, err := zstd.NewWriter(&buf)
		if err != nil {
			t.Fatalf("zstd writer: %v", err)
		}
		w.Write([]byte("zstd payload"))
		w.Close()

		got := decompress(buf.Bytes(), "zstd")
		if string(got) != "zstd payload" {
			t.Errorf("got %q, want %q", got, "zstd payload")
		}
	})

	t.Run("zstd auto-detect by magic bytes", func(t *testing.T) {
		var buf bytes.Buffer
		w, err := zstd.NewWriter(&buf)
		if err != nil {
			t.Fatalf("zstd writer: %v", err)
		}
		w.Write([]byte("zstd auto-detected"))
		w.Close()

		got := decompress(buf.Bytes(), "")
		if string(got) != "zstd auto-detected" {
			t.Errorf("got %q, want %q", got, "zstd auto-detected")
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

func TestResolveProject(t *testing.T) {
	tests := []struct {
		segment string
		want    string
	}{
		{"1", ""},        // numeric DSN id (Sentry SDK-forced) → default
		{"123", ""},      // multi-digit numeric → default
		{"my-app", "my-app"},
		{"", ""},
		{"app2", "app2"}, // trailing digit is a named project
		{"2team", "2team"},
		{"١٢٣", "١٢٣"},   // Arabic-Indic digits are named, not numeric (ASCII only)
		{"１２３", "１２３"}, // fullwidth digits are named, not numeric
	}

	for _, tt := range tests {
		t.Run(tt.segment, func(t *testing.T) {
			if got := resolveProject(tt.segment); got != tt.want {
				t.Errorf("resolveProject(%q) = %q, want %q", tt.segment, got, tt.want)
			}
		})
	}
}

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"007", true},
		{"0", true},
		{"", false},
		{"+1", false},
		{" 1", false},
		{"1.0", false},
		{"abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isAllDigits(tt.s); got != tt.want {
				t.Errorf("isAllDigits(%q) = %v, want %v", tt.s, got, tt.want)
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

// traceMetricEnvelope is a "trace_metric" envelope as the Sentry PHP SDK 4.x sends it.
const traceMetricEnvelope = `{"event_id":"metric-envelope","sent_at":"2026-10-16T10:00:00Z"}
{"type":"trace_metric","item_count":2,"content_type":"application/vnd.sentry.items.trace-metric+json"}
{"items":[{"timestamp":1788420877.523564,"trace_id":"53c16245360049e1a0cd3041dede1ec7","span_id":"9ada5f708f96d465","name":"order.status","value":3,"unit":null,"type":"gauge","attributes":{"status":{"type":"string","value":"qa"}}},{"timestamp":1788420877.6,"trace_id":"53c16245360049e1a0cd3041dede1ec7","span_id":"9ada5f708f96d465","name":"http.client.duration","value":12.5,"unit":"millisecond","type":"distribution","attributes":{"server.address":{"type":"string","value":"api"}}}]}
`

func TestHandleEnvelope_TraceMetrics(t *testing.T) {
	t.Run("stored, not turned into a feed event", func(t *testing.T) {
		db := setupTestDB(t)
		h := &handler{db: db}

		inc, err := h.handleEnvelope([]byte(traceMetricEnvelope), "proj")
		if err != nil {
			t.Fatal(err)
		}
		if inc != nil {
			t.Errorf("expected no feed event for a metrics-only envelope, got %+v", inc)
		}

		var count int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_metrics`).Scan(&count)
		if count != 2 {
			t.Errorf("metric count = %d, want 2", count)
		}
	})

	t.Run("dropped without a database", func(t *testing.T) {
		h := &handler{}

		inc, err := h.handleEnvelope([]byte(traceMetricEnvelope), "proj")
		if err != nil {
			t.Fatal(err)
		}
		if inc != nil {
			t.Errorf("expected no feed event for a metrics-only envelope, got %+v", inc)
		}
	})
}

// TestHandler_NumericProject_Envelope routes a numeric DSN envelope to the
// default project (empty project) — the JS SDK path of issue #338.
func TestHandler_NumericProject_Envelope(t *testing.T) {
	h := &handler{}

	body := `{"event_id":"env-1"}
{"type":"event"}
{"event_id":"env-1","level":"error","exception":{"values":[{"type":"E","value":"v"}]}}`
	r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(body))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=test")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if inc == nil {
		t.Fatal("Handle() returned nil, want non-nil incoming event")
	}
	if inc.Project != "" {
		t.Errorf("Project = %q, want %q (routes to default)", inc.Project, "")
	}
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
	// A numeric DSN project id (forced by the Sentry SDKs) resolves to empty so
	// the event lands in the default project downstream — see issue #338.
	if inc.Project != "" {
		t.Errorf("Project = %q, want %q", inc.Project, "")
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

// TestHandler_SDKv4_Envelope_TransactionEventIDFromHeader verifies transaction
// items are persisted with event_id from the envelope header when omitted from
// the transaction payload (Sentry PHP SDK v4+ shape).
func TestHandler_SDKv4_Envelope_TransactionEventIDFromHeader(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	first := `{"sent_at":"2026-06-13T03:40:25Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"4.27.0"},"event_id":"b74b9242f81441c5b36d603a8c1fde33"}
{"type":"transaction","content_type":"application/json"}
{"timestamp":1781322025.308429,"platform":"php","sdk":{"name":"sentry.php","version":"4.27.0"},"start_timestamp":1781322025.138361,"transaction":"/somelink","server_name":"d78b669f473f","environment":"dev","contexts":{"trace":{"span_id":"cbd2ed7ada284a07","trace_id":"43cf0f64290d425fb2a52696eb56e5c3","op":"http.server"}},"spans":[],"transaction_info":{"source":"custom"}}`

	second := `{"sent_at":"2026-06-13T03:40:25Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"4.27.0"},"event_id":"a58b0b34bc9c48e0be0ede07a93b3f4a"}
{"type":"transaction","content_type":"application/json"}
{"timestamp":1781322025.521973,"platform":"php","sdk":{"name":"sentry.php","version":"4.27.0"},"start_timestamp":1781322025.472156,"transaction":"/somelink/anotherone","server_name":"d78b669f473f","environment":"dev","contexts":{"trace":{"span_id":"c92f027e3d8d401d","trace_id":"51f437611edf44bda6dadb1703dbde98","op":"http.server"}},"spans":[],"transaction_info":{"source":"custom"}}`

	for i, envelope := range []string{first, second} {
		r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(envelope))
		r.Header.Set("Content-Type", "application/x-sentry-envelope")
		r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_client=sentry.php/4.27.0, sentry_key=test")

		inc, err := h.Handle(r)
		if err != nil {
			t.Fatalf("Handle() [%d] error: %v", i, err)
		}
		if inc != nil {
			t.Fatalf("Handle() [%d] = %+v, want nil (transactions should not create feed events)", i, inc)
		}
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions`).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("sentry_transactions count = %d, want 2", count)
	}

	for _, id := range []string{"b74b9242f81441c5b36d603a8c1fde33", "a58b0b34bc9c48e0be0ede07a93b3f4a"} {
		var exists int
		err := db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions WHERE event_id = ?`, id).Scan(&exists)
		if err != nil {
			t.Fatalf("event_id query failed for %s: %v", id, err)
		}
		if exists != 1 {
			t.Fatalf("event_id %s count = %d, want 1", id, exists)
		}
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

// v4Envelope builds an envelope shaped like sentry/sentry 4.34.0 output
// (Serializer/PayloadSerializer.php): header line, then item header/payload pairs.
func v4Envelope(eventID string, items ...string) string {
	header := `{"sent_at":"2026-10-23T12:00:00Z","dsn":"http://test@localhost:8000/1","sdk":{"name":"sentry.php","version":"4.34.0"}`
	if eventID != "" {
		header += `,"event_id":"` + eventID + `"`
	}
	return header + "}\n" + strings.Join(items, "\n")
}

// TestHandler_MessageWithMixedParams covers captureMessage() with non-string
// params, which every SDK version serializes as {"message","params","formatted"}.
func TestHandler_MessageWithMixedParams_WithDB(t *testing.T) {
	message := `{"message":"Cart %s has %d items, paid: %s","params":["c-1",3,true],"formatted":"Cart c-1 has 3 items, paid: 1"}`

	tests := []struct {
		name    string
		path    string
		body    string
		eventID string
	}{
		{
			name:    "v2 plain JSON",
			path:    "/api/1/store/",
			body:    `{"event_id":"a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1","timestamp":"2026-10-23T12:00:00Z","level":"info","platform":"php","message":` + message + `}`,
			eventID: "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1",
		},
		{
			name: "v4 envelope",
			path: "/api/1/envelope/",
			body: v4Envelope("b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2",
				`{"type":"event","content_type":"application/json"}`,
				`{"timestamp":1792756800.25,"platform":"php","sdk":{"name":"sentry.php","version":"4.34.0"},"level":"info","message":`+message+`}`),
			eventID: "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			h := &handler{db: db}

			r := httptest.NewRequest("POST", tt.path, strings.NewReader(tt.body))
			r.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7, sentry_key=test")

			inc, err := h.Handle(r)
			if err != nil {
				t.Fatalf("Handle() error: %v", err)
			}
			if inc == nil {
				t.Fatal("Handle() returned nil")
			}
			if inc.UUID != tt.eventID {
				t.Errorf("UUID = %q, want %q", inc.UUID, tt.eventID)
			}

			var parsed map[string]any
			if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
				t.Fatalf("failed to parse payload: %v", err)
			}
			if parsed["event_id"] != tt.eventID {
				t.Errorf("payload event_id = %v, want %q", parsed["event_id"], tt.eventID)
			}

			var count int
			if err := db.QueryRow("SELECT COUNT(*) FROM sentry_error_events WHERE event_id = ?", tt.eventID).Scan(&count); err != nil {
				t.Fatalf("query error: %v", err)
			}
			if count != 1 {
				t.Errorf("sentry_error_events count = %d, want 1 (mixed message params must not break structured storage)", count)
			}
		})
	}
}

// TestHandler_SDKv4_Envelope_UnparseableEventKeepsEventID checks that an event
// the structured parser rejects still reaches the feed with its event_id.
func TestHandler_SDKv4_Envelope_UnparseableEventKeepsEventID(t *testing.T) {
	h := &handler{}

	// "level" as a number does not fit ErrorEvent.
	body := v4Envelope("c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3",
		`{"type":"event","content_type":"application/json"}`,
		`{"timestamp":1792756800.25,"platform":"php","level":40,"message":"odd"}`)

	inc, err := h.handleEnvelope([]byte(body), "1")
	if err != nil {
		t.Fatal(err)
	}
	if inc == nil {
		t.Fatal("expected an incoming event")
	}
	if inc.UUID != "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3" {
		t.Errorf("UUID = %q, want envelope event_id", inc.UUID)
	}
	var parsed map[string]any
	if err := json.Unmarshal(inc.Payload, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["event_id"] != "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3" {
		t.Errorf("payload event_id = %v, want it injected from the envelope header", parsed["event_id"])
	}
}

// TestHandler_SDKv4_Transaction_WithDB covers v4 transactions: event_id only in
// the envelope header, span data with non-string values (GuzzleTracingMiddleware).
func TestHandler_SDKv4_Transaction_WithDB(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	txn := func(eventID, traceID string) string {
		return v4Envelope(eventID,
			`{"type":"transaction","content_type":"application/json"}`,
			`{"timestamp":1792756801.5,"platform":"php","sdk":{"name":"sentry.php","version":"4.34.0"},"start_timestamp":1792756800.0,"transaction":"GET /orders","contexts":{"trace":{"trace_id":"`+traceID+`","span_id":"1000000000000001","op":"http.server","origin":"manual"}},"spans":[{"span_id":"2000000000000002","trace_id":"`+traceID+`","start_timestamp":1792756800.1,"origin":"auto.http.guzzle","parent_span_id":"1000000000000001","timestamp":1792756800.4,"status":"ok","description":"GET https://api.example.com/stock","op":"http.client","data":{"http.url":"https://api.example.com/stock","http.request.method":"GET","http.request.body.size":0,"http.response.body.size":512,"http.response.status_code":200}}]}`)
	}

	for _, body := range []string{
		txn("d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		txn("e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	} {
		r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(body))
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatalf("Handle() error: %v", err)
		}
		if inc != nil {
			t.Errorf("transaction must not create a feed event, got %+v", inc)
		}
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions WHERE event_id IN (?, ?)`,
		"d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4", "e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5").Scan(&count)
	if count != 2 {
		t.Errorf("sentry_transactions count = %d, want 2 (event_id must come from the envelope header)", count)
	}

	var peerType, peerAddress, attrs string
	err := db.QueryRow(`SELECT peer_type, peer_address, attributes FROM sentry_spans WHERE trace_id = ?`,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Scan(&peerType, &peerAddress, &attrs)
	if err != nil {
		t.Fatalf("span not stored: %v", err)
	}
	if peerType != "http" || peerAddress != "api.example.com" {
		t.Errorf("peer = %q %q, want http api.example.com", peerType, peerAddress)
	}
	// Span data values are normalized to strings (SpanData).
	if !strings.Contains(attrs, `"http.response.status_code":"200"`) {
		t.Errorf("attributes = %s, want the numeric status code kept", attrs)
	}
}

// TestHandler_SDKv4_SpanV2Item_WithDB covers the "span" item (Span v2) that
// SDK v4 adds after a transaction for gen_ai spans.
func TestHandler_SDKv4_SpanV2Item_WithDB(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}

	body := v4Envelope("f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6",
		`{"type":"transaction","content_type":"application/json"}`,
		`{"timestamp":1792756803.0,"platform":"php","start_timestamp":1792756800.0,"transaction":"POST /chat","contexts":{"trace":{"trace_id":"cccccccccccccccccccccccccccccccc","span_id":"1000000000000001"}},"spans":[]}`,
		`{"type":"span","item_count":1,"content_type":"application/vnd.sentry.items.span.v2+json"}`,
		`{"items":[{"trace_id":"cccccccccccccccccccccccccccccccc","span_id":"3000000000000003","name":"chat claude","is_segment":false,"start_timestamp":1792756800.5,"attributes":{"sentry.op":{"type":"string","value":"gen_ai.chat"},"gen_ai.usage.input_tokens":{"type":"integer","value":42}},"status":"ok","end_timestamp":1792756802.5,"parent_span_id":"1000000000000001"}],"version":2}`)

	inc, err := h.handleEnvelope([]byte(body), "1")
	if err != nil {
		t.Fatal(err)
	}
	if inc != nil {
		t.Errorf("transaction with span item must not create a feed event, got payload %s", inc.Payload)
	}

	var op, description, attrs string
	var durationMS int
	err = db.QueryRow(`SELECT op, description, duration_ms, attributes FROM sentry_spans WHERE span_id = ?`,
		"3000000000000003").Scan(&op, &description, &durationMS, &attrs)
	if err != nil {
		t.Fatalf("span v2 not stored: %v", err)
	}
	if op != "gen_ai.chat" {
		t.Errorf("op = %q, want gen_ai.chat (from sentry.op attribute)", op)
	}
	if description != "chat claude" {
		t.Errorf("description = %q, want span name", description)
	}
	if durationMS != 2000 {
		t.Errorf("duration_ms = %d, want 2000 (from end_timestamp)", durationMS)
	}
	if !strings.Contains(attrs, `"gen_ai.usage.input_tokens":"42"`) {
		t.Errorf("attributes = %s, want typed attribute values unwrapped", attrs)
	}
}

// TestHandler_SDKv4_NonFeedEnvelopes checks that envelopes without an event
// never become cards in the feed.
func TestHandler_SDKv4_NonFeedEnvelopes(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "check_in",
			body: v4Envelope("a7a7a7a7a7a7a7a7a7a7a7a7a7a7a7a7",
				`{"type":"check_in","content_type":"application/json"}`,
				`{"check_in_id":"a7a7a7a7a7a7a7a7a7a7a7a7a7a7a7a7","monitor_slug":"nightly","status":"ok","duration":1.5,"release":null,"environment":"production"}`),
		},
		{
			name: "client_report",
			body: "{}\n" + `{"type":"client_report"}` + "\n" +
				`{"timestamp":1792756800.0,"discarded_events":[{"category":"error","reason":"sample_rate","quantity":2}]}`,
		},
		{
			name: "trace_metric",
			body: v4Envelope("",
				`{"type":"trace_metric","item_count":1,"content_type":"application/vnd.sentry.items.trace-metric+json"}`,
				`{"items":[{"timestamp":1792756800.0,"trace_id":"dddddddddddddddddddddddddddddddd","span_id":"4000000000000004","name":"orders.placed","value":1,"unit":null,"type":"counter","attributes":{}}]}`),
		},
		{
			name: "transaction with profile",
			body: v4Envelope("b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8",
				`{"type":"transaction","content_type":"application/json"}`,
				`{"timestamp":1792756801.0,"platform":"php","start_timestamp":1792756800.0,"transaction":"GET /","contexts":{"trace":{"trace_id":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","span_id":"1000000000000001"}},"spans":[]}`,
				`{"type":"profile","content_type":"application/json"}`,
				`{"device":{"architecture":"x86_64"},"os":{"name":"Linux"},"platform":"php","profile":{"frames":[],"samples":[],"stacks":[]},"version":"1"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{}
			r := httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(tt.body))
			inc, err := h.Handle(r)
			if err != nil {
				t.Fatalf("Handle() error: %v", err)
			}
			if inc != nil {
				t.Errorf("expected no feed event, got payload %s", inc.Payload)
			}
		})
	}
}
