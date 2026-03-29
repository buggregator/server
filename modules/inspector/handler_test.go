package inspector

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_Priority(t *testing.T) {
	h := &handler{}
	if h.Priority() != 30 {
		t.Errorf("Priority = %d, want 30", h.Priority())
	}
}

func TestHandler_Match(t *testing.T) {
	h := &handler{}

	tests := []struct {
		name   string
		method string
		header map[string]string
		want   bool
	}{
		{"POST with detected type", "POST", map[string]string{"X-Buggregator-Detected-Type": "inspector"}, true},
		{"POST with X-Inspector-Key", "POST", map[string]string{"X-Inspector-Key": "abc"}, true},
		{"POST with X-Inspector-Version", "POST", map[string]string{"X-Inspector-Version": "1.0"}, true},
		{"GET request", "GET", nil, false},
		{"POST without inspector headers", "POST", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", nil)
			for k, v := range tt.header {
				r.Header.Set(k, v)
			}
			if got := h.Match(r); got != tt.want {
				t.Errorf("Match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	h := &handler{}

	t.Run("base64-encoded JSON", func(t *testing.T) {
		payload := `[{"type":"request","model":"App\\Http"}]`
		encoded := base64.StdEncoding.EncodeToString([]byte(payload))
		r := httptest.NewRequest("POST", "/", strings.NewReader(encoded))

		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc.Type != "inspector" {
			t.Errorf("Type = %q, want %q", inc.Type, "inspector")
		}

		var data []map[string]any
		json.Unmarshal(inc.Payload, &data)
		if len(data) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(data))
		}
		if data[0]["type"] != "request" {
			t.Error("expected type=request")
		}
	})

	t.Run("raw JSON", func(t *testing.T) {
		payload := `[{"type":"segment"}]`
		r := httptest.NewRequest("POST", "/", strings.NewReader(payload))

		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc.Type != "inspector" {
			t.Errorf("Type = %q, want %q", inc.Type, "inspector")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/", strings.NewReader("not json at all"))
		_, err := h.Handle(r)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToPreview returns as-is", func(t *testing.T) {
		payload := json.RawMessage(`[{"type":"request"}]`)
		p, err := m.ToPreview(payload)
		if err != nil {
			t.Fatal(err)
		}
		if string(p) != string(payload) {
			t.Error("expected passthrough")
		}
	})

	t.Run("ToSearchableText", func(t *testing.T) {
		payload := json.RawMessage(`[{"type":"request","model":"UserController","host":{"hostname":"web-1"}}]`)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "request") {
			t.Error("expected 'request' in text")
		}
		if !strings.Contains(text, "UserController") {
			t.Error("expected 'UserController' in text")
		}
		if !strings.Contains(text, "web-1") {
			t.Error("expected 'web-1' in text")
		}
	})
}
