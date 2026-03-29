package sms

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGateway_Detect(t *testing.T) {
	t.Run("twilio detected", func(t *testing.T) {
		gw := findByName("twilio")
		body := map[string]any{"MessageSid": "SM123", "Body": "Hello"}
		if !gw.detect(body) {
			t.Error("expected twilio to be detected")
		}
	})

	t.Run("twilio missing field", func(t *testing.T) {
		gw := findByName("twilio")
		body := map[string]any{"MessageSid": "SM123"}
		if gw.detect(body) {
			t.Error("expected twilio not detected without Body")
		}
	})

	t.Run("generic always matches", func(t *testing.T) {
		gw := findByName("generic")
		if !gw.detect(map[string]any{}) {
			t.Error("generic should always match")
		}
	})
}

func TestGateway_Parse(t *testing.T) {
	gw := findByName("twilio")
	body := map[string]any{
		"MessageSid": "SM123",
		"From":       "+1234567890",
		"To":         "+0987654321",
		"Body":       "Hello World",
	}
	msg := gw.parse(body)
	if msg.From != "+1234567890" {
		t.Errorf("From = %q", msg.From)
	}
	if msg.To != "+0987654321" {
		t.Errorf("To = %q", msg.To)
	}
	if msg.Message != "Hello World" {
		t.Errorf("Message = %q", msg.Message)
	}
	if msg.Gateway != "twilio" {
		t.Errorf("Gateway = %q", msg.Gateway)
	}
}

func TestDetectGatewayFromBody(t *testing.T) {
	t.Run("detects twilio", func(t *testing.T) {
		body := map[string]any{"MessageSid": "SM123", "Body": "Hi"}
		gw := detectGatewayFromBody(body)
		if gw == nil || gw.Name != "twilio" {
			t.Errorf("expected twilio, got %v", gw)
		}
	})

	t.Run("falls back to generic", func(t *testing.T) {
		body := map[string]any{"random": "data"}
		gw := detectGatewayFromBody(body)
		if gw == nil || gw.Name != "generic" {
			t.Errorf("expected generic, got %v", gw)
		}
	})
}

func TestExtractFirst(t *testing.T) {
	body := map[string]any{
		"From":   "+123",
		"sender": "",
		"count":  float64(42),
	}

	if got := extractFirst(body, []string{"missing", "From"}); got != "+123" {
		t.Errorf("got %q, want %q", got, "+123")
	}

	// Empty string skipped
	if got := extractFirst(body, []string{"sender", "From"}); got != "+123" {
		t.Errorf("got %q, want %q", got, "+123")
	}

	// Float64 value
	if got := extractFirst(body, []string{"count"}); got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}

	// No match
	if got := extractFirst(body, []string{"missing"}); got != "" {
		t.Errorf("got %q, want empty", got)
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
		{"POST to /sms", "POST", "/sms", nil, true},
		{"POST to /sms/twilio", "POST", "/sms/twilio", nil, true},
		{"POST with detected type", "POST", "/", map[string]string{"X-Buggregator-Detected-Type": "sms"}, true},
		{"GET to /sms", "GET", "/sms", nil, false},
		{"POST to /other", "POST", "/other", nil, false},
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

func TestHandler_Handle(t *testing.T) {
	h := &handler{}

	t.Run("auto-detect twilio", func(t *testing.T) {
		body := `{"MessageSid":"SM123","Body":"Hello","From":"+111","To":"+222"}`
		r := httptest.NewRequest("POST", "/sms", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc == nil {
			t.Fatal("expected non-nil")
		}
		if inc.Type != "sms" {
			t.Errorf("Type = %q", inc.Type)
		}
		var msg SmsMessage
		json.Unmarshal(inc.Payload, &msg)
		if msg.Gateway != "twilio" {
			t.Errorf("Gateway = %q, want twilio", msg.Gateway)
		}
	})

	t.Run("explicit gateway", func(t *testing.T) {
		body := `{"From":"+111","To":"+222","Body":"Hi"}`
		r := httptest.NewRequest("POST", "/sms/twilio", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc == nil {
			t.Fatal("expected non-nil")
		}
		var msg SmsMessage
		json.Unmarshal(inc.Payload, &msg)
		if msg.Gateway != "twilio" {
			t.Errorf("Gateway = %q", msg.Gateway)
		}
	})

	t.Run("explicit gateway with project", func(t *testing.T) {
		body := `{"From":"+111","To":"+222","Body":"Hi"}`
		r := httptest.NewRequest("POST", "/sms/twilio/myproject", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc.Project != "myproject" {
			t.Errorf("Project = %q, want myproject", inc.Project)
		}
	})

	t.Run("unknown segment treated as project", func(t *testing.T) {
		body := `{"from":"+111","to":"+222","message":"Hi"}`
		r := httptest.NewRequest("POST", "/sms/myproject", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc == nil {
			t.Fatal("expected non-nil")
		}
		if inc.Project != "myproject" {
			t.Errorf("Project = %q, want myproject", inc.Project)
		}
	})

	t.Run("empty to and message returns nil", func(t *testing.T) {
		body := `{"random":"data"}`
		r := httptest.NewRequest("POST", "/sms", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		inc, err := h.Handle(r)
		if err != nil {
			t.Fatal(err)
		}
		if inc != nil {
			t.Error("expected nil for empty to+message")
		}
	})
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToSearchableText", func(t *testing.T) {
		msg := SmsMessage{From: "+111", To: "+222", Message: "Hello"}
		payload, _ := json.Marshal(msg)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "+111") || !strings.Contains(text, "+222") || !strings.Contains(text, "Hello") {
			t.Errorf("text = %q", text)
		}
	})
}
