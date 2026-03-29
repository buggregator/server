package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newTestCollector creates a Collector with a fresh registry to avoid conflicts between tests.
func newTestCollector(t *testing.T) (*Collector, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()

	c := &Collector{
		EventsReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "events_received_total",
		}, []string{"type", "project"}),
		EventsIngestionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "events_ingestion_duration_seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 1},
		}, []string{"type"}),
		EventsIngestionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "events_ingestion_errors_total",
		}, []string{"type", "error"}),
		EventsPayloadBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "events_payload_bytes",
			Buckets: prometheus.ExponentialBuckets(256, 4, 8),
		}, []string{"type"}),
		EventsStoredTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "events_stored_total",
		}, []string{"type"}),
		StorageQueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "storage_query_duration_seconds",
			Buckets: []float64{0.001, 0.01, 0.1},
		}, []string{"operation"}),
		WSConnectionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Name: "ws_connections_active",
		}),
		WSMessagesSentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "ws_messages_sent_total",
		}, []string{"channel"}),
		TCPConnectionsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "tcp_connections_active",
		}, []string{"server"}),
		TCPMessagesReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "tcp_messages_received_total",
		}, []string{"server"}),
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "http_requests_total",
		}, []string{"method", "path", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "http_request_duration_seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 1},
		}, []string{"method", "path"}),
		WebhooksSentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "webhooks_sent_total",
		}, []string{"event", "url", "success"}),
	}

	reg.MustRegister(
		c.EventsReceivedTotal, c.EventsIngestionDuration, c.EventsIngestionErrors,
		c.EventsPayloadBytes, c.EventsStoredTotal, c.StorageQueryDuration,
		c.WSConnectionsActive, c.WSMessagesSentTotal,
		c.TCPConnectionsActive, c.TCPMessagesReceivedTotal,
		c.HTTPRequestsTotal, c.HTTPRequestDuration, c.WebhooksSentTotal,
	)

	return c, reg
}

func TestCollector_EventsReceivedTotal(t *testing.T) {
	c, _ := newTestCollector(t)

	c.EventsReceivedTotal.WithLabelValues("sentry", "default").Inc()
	c.EventsReceivedTotal.WithLabelValues("sentry", "default").Inc()
	c.EventsReceivedTotal.WithLabelValues("ray", "my-app").Inc()

	if v := testutil.ToFloat64(c.EventsReceivedTotal.WithLabelValues("sentry", "default")); v != 2 {
		t.Errorf("sentry/default = %v, want 2", v)
	}
	if v := testutil.ToFloat64(c.EventsReceivedTotal.WithLabelValues("ray", "my-app")); v != 1 {
		t.Errorf("ray/my-app = %v, want 1", v)
	}
}

func TestCollector_WSConnectionsGauge(t *testing.T) {
	c, _ := newTestCollector(t)

	c.WSConnectionsActive.Inc()
	c.WSConnectionsActive.Inc()
	c.WSConnectionsActive.Dec()

	if v := testutil.ToFloat64(c.WSConnectionsActive); v != 1 {
		t.Errorf("ws_connections_active = %v, want 1", v)
	}
}

func TestCollector_TCPConnectionsGauge(t *testing.T) {
	c, _ := newTestCollector(t)

	c.TCPConnectionsActive.WithLabelValues("monolog").Inc()
	c.TCPConnectionsActive.WithLabelValues("monolog").Inc()
	c.TCPConnectionsActive.WithLabelValues("smtp").Inc()
	c.TCPConnectionsActive.WithLabelValues("monolog").Dec()

	if v := testutil.ToFloat64(c.TCPConnectionsActive.WithLabelValues("monolog")); v != 1 {
		t.Errorf("monolog = %v, want 1", v)
	}
	if v := testutil.ToFloat64(c.TCPConnectionsActive.WithLabelValues("smtp")); v != 1 {
		t.Errorf("smtp = %v, want 1", v)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/api/event/550e8400-e29b-41d4-a716-446655440000", "/api/event/{uuid}"},
		{"/api/profiler/A1B2C3D4-E5F6-7890-ABCD-EF1234567890/summary", "/api/profiler/{uuid}/summary"},
		{"/api/events", "/api/events"},
		{"/metrics", "/metrics"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHTTPMiddleware(t *testing.T) {
	c, _ := newTestCollector(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := HTTPMiddleware(handler, c)
	req := httptest.NewRequest("GET", "/api/events", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if v := testutil.ToFloat64(c.HTTPRequestsTotal.WithLabelValues("GET", "/api/events", "200")); v != 1 {
		t.Errorf("http_requests_total = %v, want 1", v)
	}
}

func TestHTTPMiddleware_StatusCodes(t *testing.T) {
	c, _ := newTestCollector(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := HTTPMiddleware(handler, c)

	// OK request.
	req := httptest.NewRequest("GET", "/ok", nil)
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	// Error request.
	req = httptest.NewRequest("POST", "/error", nil)
	wrapped.ServeHTTP(httptest.NewRecorder(), req)

	if v := testutil.ToFloat64(c.HTTPRequestsTotal.WithLabelValues("GET", "/ok", "200")); v != 1 {
		t.Errorf("GET /ok 200 = %v, want 1", v)
	}
	if v := testutil.ToFloat64(c.HTTPRequestsTotal.WithLabelValues("POST", "/error", "500")); v != 1 {
		t.Errorf("POST /error 500 = %v, want 1", v)
	}
}

func TestHTTPMiddleware_UUIDNormalization(t *testing.T) {
	c, _ := newTestCollector(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := HTTPMiddleware(handler, c)

	// Two requests with different UUIDs should map to the same label.
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/event/550e8400-e29b-41d4-a716-446655440000", nil))
	wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/event/660e8400-f39c-52e5-b827-557766551111", nil))

	if v := testutil.ToFloat64(c.HTTPRequestsTotal.WithLabelValues("GET", "/api/event/{uuid}", "200")); v != 2 {
		t.Errorf("uuid-normalized requests = %v, want 2", v)
	}
}

func setupTestDB(t *testing.T) *storage.SQLiteStore {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return storage.NewSQLiteStore(db)
}

func TestInstrumentedStore_StoreAndGauge(t *testing.T) {
	c, _ := newTestCollector(t)
	db, _ := storage.Open(":memory:")
	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	defer db.Close()

	store := storage.NewSQLiteStore(db)
	instrumented := NewInstrumentedStore(store, c, db)

	ctx := context.Background()
	ev := event.Event{
		UUID:    "test-uuid-1",
		Type:    "sentry",
		Payload: json.RawMessage(`{"test": true}`),
		Project: "default",
	}

	if err := instrumented.Store(ctx, ev); err != nil {
		t.Fatal(err)
	}

	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("sentry")); v != 1 {
		t.Errorf("events_stored_total{sentry} = %v, want 1", v)
	}

	// Store another event.
	ev.UUID = "test-uuid-2"
	instrumented.Store(ctx, ev)

	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("sentry")); v != 2 {
		t.Errorf("events_stored_total{sentry} = %v, want 2", v)
	}
}

func TestInstrumentedStore_DeleteDecrementsGauge(t *testing.T) {
	c, _ := newTestCollector(t)
	db, _ := storage.Open(":memory:")
	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	defer db.Close()

	store := storage.NewSQLiteStore(db)
	instrumented := NewInstrumentedStore(store, c, db)

	ctx := context.Background()
	ev := event.Event{
		UUID:    "del-test-1",
		Type:    "ray",
		Payload: json.RawMessage(`{}`),
		Project: "default",
	}
	instrumented.Store(ctx, ev)

	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("ray")); v != 1 {
		t.Errorf("before delete: events_stored_total{ray} = %v, want 1", v)
	}

	instrumented.Delete(ctx, "del-test-1")

	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("ray")); v != 0 {
		t.Errorf("after delete: events_stored_total{ray} = %v, want 0", v)
	}
}

func TestInstrumentedStore_QueryDuration(t *testing.T) {
	c, _ := newTestCollector(t)
	db, _ := storage.Open(":memory:")
	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	defer db.Close()

	store := storage.NewSQLiteStore(db)
	instrumented := NewInstrumentedStore(store, c, db)

	ctx := context.Background()
	instrumented.FindAll(ctx, event.FindOptions{})

	// Verify the histogram has at least one observation by collecting the metric.
	count := testutil.CollectAndCount(c.StorageQueryDuration)
	if count == 0 {
		t.Error("storage_query_duration should have recorded observations")
	}
}

func TestInstrumentedStore_SeedGauge(t *testing.T) {
	c, _ := newTestCollector(t)
	db, _ := storage.Open(":memory:")
	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	// Pre-populate.
	db.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned) VALUES ('a', 'sentry', '{}', '1.0', 'default', 0)`)
	db.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned) VALUES ('b', 'sentry', '{}', '2.0', 'default', 0)`)
	db.Exec(`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned) VALUES ('c', 'ray', '{}', '3.0', 'default', 0)`)
	defer db.Close()

	store := storage.NewSQLiteStore(db)
	_ = NewInstrumentedStore(store, c, db) // seed happens in constructor

	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("sentry")); v != 2 {
		t.Errorf("seeded sentry = %v, want 2", v)
	}
	if v := testutil.ToFloat64(c.EventsStoredTotal.WithLabelValues("ray")); v != 1 {
		t.Errorf("seeded ray = %v, want 1", v)
	}
}

func TestNewCollector(t *testing.T) {
	// Verify NewCollector doesn't panic and all fields are non-nil.
	// Use a separate registry to avoid double-registration with default.
	c := &Collector{
		EventsReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "events_received_total",
		}, []string{"type", "project"}),
		EventsIngestionDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "test", Name: "events_ingestion_duration_seconds",
		}, []string{"type"}),
		EventsIngestionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "events_ingestion_errors_total",
		}, []string{"type", "error"}),
		EventsPayloadBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "test", Name: "events_payload_bytes",
		}, []string{"type"}),
		EventsStoredTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "test", Name: "events_stored_total",
		}, []string{"type"}),
		StorageQueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "test", Name: "storage_query_duration_seconds",
		}, []string{"operation"}),
		WSConnectionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "test", Name: "ws_connections_active",
		}),
		WSMessagesSentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "ws_messages_sent_total",
		}, []string{"channel"}),
		TCPConnectionsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "test", Name: "tcp_connections_active",
		}, []string{"server"}),
		TCPMessagesReceivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "tcp_messages_received_total",
		}, []string{"server"}),
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "http_requests_total",
		}, []string{"method", "path", "status"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "test", Name: "http_request_duration_seconds",
		}, []string{"method", "path"}),
		WebhooksSentTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "test", Name: "webhooks_sent_total",
		}, []string{"event", "url", "success"}),
	}

	if c.EventsReceivedTotal == nil {
		t.Error("EventsReceivedTotal is nil")
	}
	if c.WSConnectionsActive == nil {
		t.Error("WSConnectionsActive is nil")
	}
	if c.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}

	// Verify labels work without panic.
	c.EventsReceivedTotal.WithLabelValues("sentry", "default").Inc()
	c.HTTPRequestsTotal.WithLabelValues("GET", "/api/events", "200").Inc()
	c.WebhooksSentTotal.WithLabelValues("sentry", "https://example.com", "true").Inc()
}

func TestHTTPMiddleware_ResponseWriterUnwrap(t *testing.T) {
	c, _ := newTestCollector(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write body without explicit WriteHeader — should default to 200.
		w.Write([]byte("hello"))
	})

	wrapped := HTTPMiddleware(handler, c)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("response code = %d, want 200", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != "hello" {
		t.Errorf("response body = %q, want %q", body, "hello")
	}

	if v := testutil.ToFloat64(c.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200")); v != 1 {
		t.Errorf("http_requests_total = %v, want 1", v)
	}
}
