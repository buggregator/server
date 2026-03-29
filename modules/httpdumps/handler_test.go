package httpdumps

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_Priority(t *testing.T) {
	h := &handler{}
	if h.Priority() != 9999 {
		t.Errorf("Priority = %d, want 9999", h.Priority())
	}
}

func TestHandler_Match(t *testing.T) {
	h := &handler{}

	// HTTP dump matches everything
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		r := httptest.NewRequest(method, "/anything", nil)
		if !h.Match(r) {
			t.Errorf("Match(%s) = false, want true", method)
		}
	}
}

func TestHandler_Handle_GET(t *testing.T) {
	h := &handler{}
	r := httptest.NewRequest("GET", "/test/path?foo=bar", nil)
	r.Header.Set("X-Custom", "value")
	r.Host = "example.com"

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc.Type != "http-dump" {
		t.Errorf("Type = %q, want %q", inc.Type, "http-dump")
	}
	if inc.UUID == "" {
		t.Error("UUID should be generated")
	}

	var payload map[string]any
	json.Unmarshal(inc.Payload, &payload)

	if payload["host"] != "example.com" {
		t.Errorf("host = %v", payload["host"])
	}

	req, _ := payload["request"].(map[string]any)
	if req == nil {
		t.Fatal("request is nil")
	}
	if req["method"] != "GET" {
		t.Errorf("method = %v", req["method"])
	}
	if req["uri"] != "test/path" {
		t.Errorf("uri = %v", req["uri"])
	}

	query, _ := req["query"].(map[string]any)
	if query["foo"] != "bar" {
		t.Errorf("query.foo = %v", query["foo"])
	}
}

func TestHandler_Handle_POST_JSON(t *testing.T) {
	h := &handler{}
	body := `{"key":"value"}`
	r := httptest.NewRequest("POST", "/api/endpoint", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Host = "localhost:8000"

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	json.Unmarshal(inc.Payload, &payload)

	req, _ := payload["request"].(map[string]any)
	post, _ := req["post"].(map[string]any)
	if post["key"] != "value" {
		t.Errorf("post.key = %v", post["key"])
	}
}

func TestHandler_Handle_POST_PlainBody(t *testing.T) {
	h := &handler{}
	r := httptest.NewRequest("POST", "/form", strings.NewReader("raw body content"))
	r.Header.Set("Content-Type", "text/plain")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	json.Unmarshal(inc.Payload, &payload)

	req, _ := payload["request"].(map[string]any)
	if req["body"] != "raw body content" {
		t.Errorf("body = %v", req["body"])
	}
}

func TestHandler_Handle_Cookies(t *testing.T) {
	h := &handler{}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Cookie", "session=abc123; theme=dark")

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	json.Unmarshal(inc.Payload, &payload)

	req, _ := payload["request"].(map[string]any)
	cookies, _ := req["cookies"].(map[string]any)
	if cookies["session"] != "abc123" {
		t.Errorf("cookies.session = %v", cookies["session"])
	}
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToPreview returns as-is", func(t *testing.T) {
		payload := json.RawMessage(`{"host":"example.com","request":{"method":"GET"}}`)
		p, err := m.ToPreview(payload)
		if err != nil {
			t.Fatal(err)
		}
		if string(p) != string(payload) {
			t.Error("expected passthrough")
		}
	})

	t.Run("ToSearchableText", func(t *testing.T) {
		payload := json.RawMessage(`{"host":"example.com","request":{"method":"GET","uri":"api/test"}}`)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "example.com") {
			t.Error("expected host in text")
		}
		if !strings.Contains(text, "GET") {
			t.Error("expected method in text")
		}
	})
}
