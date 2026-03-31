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

	t.Run("envelope without valid payload fallback", func(t *testing.T) {
		body := []byte(`{"event_id":"test-id"}
{"type":"event"}
not valid json`)
		inc, err := h.handleEnvelope(body, "")
		if err != nil {
			t.Fatal(err)
		}
		// Should fallback to wrapping entire envelope
		var p map[string]any
		json.Unmarshal(inc.Payload, &p)
		if _, ok := p["envelope"]; !ok {
			t.Error("expected fallback envelope wrapping")
		}
	})
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
