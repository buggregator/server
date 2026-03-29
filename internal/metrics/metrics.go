package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "buggregator"

// Collector holds all Prometheus metrics for the application.
type Collector struct {
	// Ingestion.
	EventsReceivedTotal     *prometheus.CounterVec
	EventsIngestionDuration *prometheus.HistogramVec
	EventsIngestionErrors   *prometheus.CounterVec
	EventsPayloadBytes      *prometheus.HistogramVec

	// Storage.
	EventsStoredTotal    *prometheus.GaugeVec
	StorageQueryDuration *prometheus.HistogramVec

	// WebSocket.
	WSConnectionsActive prometheus.Gauge
	WSMessagesSentTotal *prometheus.CounterVec

	// TCP.
	TCPConnectionsActive    *prometheus.GaugeVec
	TCPMessagesReceivedTotal *prometheus.CounterVec

	// HTTP.
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Webhooks.
	WebhooksSentTotal *prometheus.CounterVec
}

// NewCollector creates and registers all Prometheus metrics.
func NewCollector() *Collector {
	return &Collector{
		// Ingestion.
		EventsReceivedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_received_total",
			Help:      "Total number of events received.",
		}, []string{"type", "project"}),

		EventsIngestionDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "events_ingestion_duration_seconds",
			Help:      "Time spent processing incoming events.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"type"}),

		EventsIngestionErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_ingestion_errors_total",
			Help:      "Total number of event ingestion errors.",
		}, []string{"type", "error"}),

		EventsPayloadBytes: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "events_payload_bytes",
			Help:      "Size of event payloads in bytes.",
			Buckets:   prometheus.ExponentialBuckets(256, 4, 8), // 256B, 1KB, 4KB, 16KB, 64KB, 256KB, 1MB, 4MB
		}, []string{"type"}),

		// Storage.
		EventsStoredTotal: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "events_stored_total",
			Help:      "Current number of events in storage.",
		}, []string{"type"}),

		StorageQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "storage_query_duration_seconds",
			Help:      "Duration of storage operations.",
			Buckets:   []float64{0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		}, []string{"operation"}),

		// WebSocket.
		WSConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ws_connections_active",
			Help:      "Number of active WebSocket connections.",
		}),

		WSMessagesSentTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ws_messages_sent_total",
			Help:      "Total number of WebSocket messages sent.",
		}, []string{"channel"}),

		// TCP.
		TCPConnectionsActive: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tcp_connections_active",
			Help:      "Number of active TCP connections.",
		}, []string{"server"}),

		TCPMessagesReceivedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tcp_messages_received_total",
			Help:      "Total number of TCP messages received.",
		}, []string{"server"}),

		// HTTP.
		HTTPRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		}, []string{"method", "path", "status"}),

		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"method", "path"}),

		// Webhooks.
		WebhooksSentTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "webhooks_sent_total",
			Help:      "Total number of webhooks sent.",
		}, []string{"event", "url", "success"}),
	}
}
