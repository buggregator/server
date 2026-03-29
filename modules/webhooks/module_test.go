package webhooks

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

func TestMatchEvent(t *testing.T) {
	tests := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		{"*", "sentry", true},
		{"*", "ray", true},
		{"sentry", "sentry", true},
		{"Sentry", "sentry", true},
		{"sentry", "Sentry", true},
		{"sentry", "ray", false},
		{"ray", "sentry", false},
		{"smtp", "smtp", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.eventType, func(t *testing.T) {
			got := matchEvent(tt.pattern, tt.eventType)
			if got != tt.want {
				t.Errorf("matchEvent(%q, %q) = %v, want %v", tt.pattern, tt.eventType, got, tt.want)
			}
		})
	}
}

func TestModule_OnEventStored_DeliveresToMatchingWebhook(t *testing.T) {
	var mu sync.Mutex
	var received []webhookPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		body, _ := io.ReadAll(r.Body)
		var p webhookPayload
		json.Unmarshal(body, &p)

		mu.Lock()
		received = append(received, p)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{Event: "sentry", URL: srv.URL, Retry: false},
	}, nil)

	ev := event.Event{
		UUID:      "test-uuid",
		Type:      "sentry",
		Payload:   json.RawMessage(`{"message":"test error"}`),
		Timestamp: 1234567890.123,
		Project:   "default",
	}

	mod.OnEventStored(ev)

	// Wait for async delivery.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(received))
	}
	if received[0].UUID != "test-uuid" {
		t.Errorf("uuid = %q, want %q", received[0].UUID, "test-uuid")
	}
	if received[0].Type != "sentry" {
		t.Errorf("type = %q, want %q", received[0].Type, "sentry")
	}
	if received[0].Project != "default" {
		t.Errorf("project = %q, want %q", received[0].Project, "default")
	}
}

func TestModule_OnEventStored_WildcardMatchesAll(t *testing.T) {
	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{Event: "*", URL: srv.URL, Retry: false},
	}, nil)

	for _, evType := range []string{"sentry", "ray", "smtp", "monolog"} {
		mod.OnEventStored(event.Event{
			UUID:    "uuid-" + evType,
			Type:    evType,
			Payload: json.RawMessage(`{}`),
		})
	}

	time.Sleep(200 * time.Millisecond)

	if v := count.Load(); v != 4 {
		t.Errorf("expected 4 deliveries, got %d", v)
	}
}

func TestModule_OnEventStored_SkipsNonMatchingEvents(t *testing.T) {
	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{Event: "sentry", URL: srv.URL, Retry: false},
	}, nil)

	// Send a ray event — should NOT trigger the sentry-only webhook.
	mod.OnEventStored(event.Event{
		UUID:    "ray-uuid",
		Type:    "ray",
		Payload: json.RawMessage(`{}`),
	})

	time.Sleep(100 * time.Millisecond)

	if v := count.Load(); v != 0 {
		t.Errorf("expected 0 deliveries for non-matching event, got %d", v)
	}
}

func TestModule_OnEventStored_MultipleWebhooks(t *testing.T) {
	var countA, countB atomic.Int32

	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countA.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srvA.Close()

	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countB.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srvB.Close()

	mod := New([]WebhookConfig{
		{Event: "sentry", URL: srvA.URL, Retry: false},
		{Event: "*", URL: srvB.URL, Retry: false},
	}, nil)

	mod.OnEventStored(event.Event{
		UUID:    "test-uuid",
		Type:    "sentry",
		Payload: json.RawMessage(`{}`),
	})

	time.Sleep(200 * time.Millisecond)

	if v := countA.Load(); v != 1 {
		t.Errorf("sentry webhook: expected 1, got %d", v)
	}
	if v := countB.Load(); v != 1 {
		t.Errorf("wildcard webhook: expected 1, got %d", v)
	}
}

func TestModule_OnEventStored_CustomHeaders(t *testing.T) {
	var receivedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{
			Event:   "*",
			URL:     srv.URL,
			Headers: map[string]string{"Authorization": "Bearer secret-token"},
			Retry:   false,
		},
	}, nil)

	mod.OnEventStored(event.Event{
		UUID:    "test-uuid",
		Type:    "sentry",
		Payload: json.RawMessage(`{}`),
	})

	time.Sleep(100 * time.Millisecond)

	if receivedAuth != "Bearer secret-token" {
		t.Errorf("Authorization header = %q, want %q", receivedAuth, "Bearer secret-token")
	}
}

func TestModule_OnEventStored_RetriesOnServerError(t *testing.T) {
	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := count.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{Event: "*", URL: srv.URL, Retry: true},
	}, nil)

	mod.OnEventStored(event.Event{
		UUID:    "retry-uuid",
		Type:    "sentry",
		Payload: json.RawMessage(`{}`),
	})

	// Wait for retries (5s + 10s backoff, but test server is fast).
	time.Sleep(20 * time.Second)

	if v := count.Load(); v != 3 {
		t.Errorf("expected 3 attempts (2 failures + 1 success), got %d", v)
	}
}

func TestModule_OnEventStored_NoRetryWhenDisabled(t *testing.T) {
	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	mod := New([]WebhookConfig{
		{Event: "*", URL: srv.URL, Retry: false},
	}, nil)

	mod.OnEventStored(event.Event{
		UUID:    "no-retry-uuid",
		Type:    "sentry",
		Payload: json.RawMessage(`{}`),
	})

	time.Sleep(200 * time.Millisecond)

	if v := count.Load(); v != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", v)
	}
}

func TestModule_NameAndType(t *testing.T) {
	mod := New(nil, nil)
	if mod.Name() != "Webhooks" {
		t.Errorf("Name() = %q, want %q", mod.Name(), "Webhooks")
	}
	if mod.Type() != "webhooks" {
		t.Errorf("Type() = %q, want %q", mod.Type(), "webhooks")
	}
}
