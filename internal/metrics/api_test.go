package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHandleMetricsJSON_EmptyMetrics(t *testing.T) {
	// Use a custom registry to avoid pollution from other tests.
	reg := prometheus.NewRegistry()
	prometheus.DefaultGatherer = reg

	handler := HandleMetricsJSON()
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp MetricsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Events.Received) != 0 {
		t.Errorf("events.received = %v, want empty", resp.Events.Received)
	}
	if resp.Connections.WebSocket != 0 {
		t.Errorf("connections.websocket = %v, want 0", resp.Connections.WebSocket)
	}
	if resp.HTTP.RequestsTotal != 0 {
		t.Errorf("http.requests_total = %v, want 0", resp.HTTP.RequestsTotal)
	}
}

func TestHandleMetricsJSON_WithData(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultGatherer = reg

	// Create and register metrics manually.
	eventsReceived := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "events_received_total",
	}, []string{"type", "project"})
	eventsStored := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Name: "events_stored_total",
	}, []string{"type"})
	ingestionErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "events_ingestion_errors_total",
	}, []string{"type", "error"})
	wsActive := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace, Name: "ws_connections_active",
	})
	tcpActive := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace, Name: "tcp_connections_active",
	}, []string{"server"})
	httpTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "http_requests_total",
	}, []string{"method", "path", "status"})
	webhooksTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace, Name: "webhooks_sent_total",
	}, []string{"event", "url", "success"})

	reg.MustRegister(eventsReceived, eventsStored, ingestionErrors,
		wsActive, tcpActive, httpTotal, webhooksTotal)

	// Simulate activity.
	eventsReceived.WithLabelValues("sentry", "default").Add(10)
	eventsReceived.WithLabelValues("sentry", "my-app").Add(5)
	eventsReceived.WithLabelValues("ray", "default").Add(3)
	eventsStored.WithLabelValues("sentry").Set(12)
	eventsStored.WithLabelValues("ray").Set(3)
	ingestionErrors.WithLabelValues("sentry", "store_failed").Add(2)
	wsActive.Set(4)
	tcpActive.WithLabelValues("monolog").Set(1)
	tcpActive.WithLabelValues("smtp").Set(2)
	httpTotal.WithLabelValues("GET", "/api/events", "200").Add(50)
	httpTotal.WithLabelValues("POST", "/", "200").Add(20)
	webhooksTotal.WithLabelValues("sentry", "https://example.com", "true").Add(8)
	webhooksTotal.WithLabelValues("sentry", "https://example.com", "false").Add(1)

	handler := HandleMetricsJSON()
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp MetricsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Events received: sentry=15 (10+5 across projects), ray=3.
	if v := resp.Events.Received["sentry"]; v != 15 {
		t.Errorf("events.received[sentry] = %v, want 15", v)
	}
	if v := resp.Events.Received["ray"]; v != 3 {
		t.Errorf("events.received[ray] = %v, want 3", v)
	}

	// Events stored.
	if v := resp.Events.Stored["sentry"]; v != 12 {
		t.Errorf("events.stored[sentry] = %v, want 12", v)
	}
	if v := resp.Events.Stored["ray"]; v != 3 {
		t.Errorf("events.stored[ray] = %v, want 3", v)
	}

	// Errors.
	if resp.Events.Errors != 2 {
		t.Errorf("events.errors = %v, want 2", resp.Events.Errors)
	}

	// WebSocket.
	if resp.Connections.WebSocket != 4 {
		t.Errorf("connections.websocket = %v, want 4", resp.Connections.WebSocket)
	}

	// TCP.
	if v := resp.Connections.TCP["monolog"]; v != 1 {
		t.Errorf("connections.tcp[monolog] = %v, want 1", v)
	}
	if v := resp.Connections.TCP["smtp"]; v != 2 {
		t.Errorf("connections.tcp[smtp] = %v, want 2", v)
	}

	// HTTP.
	if resp.HTTP.RequestsTotal != 70 {
		t.Errorf("http.requests_total = %v, want 70", resp.HTTP.RequestsTotal)
	}

	// Webhooks.
	if resp.Webhooks.Sent != 8 {
		t.Errorf("webhooks.sent = %v, want 8", resp.Webhooks.Sent)
	}
	if resp.Webhooks.Failed != 1 {
		t.Errorf("webhooks.failed = %v, want 1", resp.Webhooks.Failed)
	}
}

func TestHandleMetricsJSON_ContentType(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultGatherer = reg

	handler := HandleMetricsJSON()
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}
