package ray

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_Priority(t *testing.T) {
	h := &handler{}
	if h.Priority() != 20 {
		t.Errorf("Priority = %d, want 20", h.Priority())
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
		{"POST with detected type", "POST", map[string]string{"X-Buggregator-Detected-Type": "ray"}, true},
		{"POST with Ray user-agent", "POST", map[string]string{"User-Agent": "Ray 2.0"}, true},
		{"POST with ray lowercase user-agent", "POST", map[string]string{"User-Agent": "ray/1.0"}, true},
		{"GET request", "GET", nil, false},
		{"POST without ray indicators", "POST", nil, false},
		{"POST with non-ray user-agent", "POST", map[string]string{"User-Agent": "Mozilla/5.0"}, false},
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

	t.Run("extracts UUID from payload", func(t *testing.T) {
		body := `{"uuid":"ray-uuid-123","payloads":[{"type":"log"}]}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc.UUID != "ray-uuid-123" {
			t.Errorf("UUID = %q, want %q", inc.UUID, "ray-uuid-123")
		}
		if inc.Type != "ray" {
			t.Errorf("Type = %q, want %q", inc.Type, "ray")
		}
	})

	t.Run("empty UUID when not in payload", func(t *testing.T) {
		body := `{"payloads":[]}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc.UUID != "" {
			t.Errorf("UUID = %q, want empty", inc.UUID)
		}
	})
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToPreview returns payload as-is", func(t *testing.T) {
		payload := json.RawMessage(`{"uuid":"123","payloads":[]}`)
		preview, err := m.ToPreview(payload)
		if err != nil {
			t.Fatal(err)
		}
		if string(preview) != string(payload) {
			t.Errorf("preview should equal payload")
		}
	})

	t.Run("ToSearchableText extracts content", func(t *testing.T) {
		payload := json.RawMessage(`{"payloads":[{"content":{"value":"hello world"}}]}`)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "hello world") {
			t.Errorf("text = %q, expected to contain 'hello world'", text)
		}
	})
}
