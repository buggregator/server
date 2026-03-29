package metrics

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// MetricsResponse is the JSON response for GET /api/metrics.
type MetricsResponse struct {
	Events      EventsMetrics      `json:"events"`
	Connections ConnectionsMetrics `json:"connections"`
	HTTP        HTTPMetrics        `json:"http"`
	Webhooks    WebhooksMetrics    `json:"webhooks"`
}

// EventsMetrics shows event counters grouped by type.
type EventsMetrics struct {
	Received map[string]float64 `json:"received"` // by type
	Stored   map[string]float64 `json:"stored"`   // by type (current gauge)
	Errors   float64            `json:"errors"`
}

// ConnectionsMetrics shows active connections.
type ConnectionsMetrics struct {
	WebSocket float64            `json:"websocket"`
	TCP       map[string]float64 `json:"tcp"` // by server name
}

// HTTPMetrics shows HTTP request stats.
type HTTPMetrics struct {
	RequestsTotal float64 `json:"requests_total"`
}

// WebhooksMetrics shows webhook delivery stats.
type WebhooksMetrics struct {
	Sent   float64 `json:"sent"`
	Failed float64 `json:"failed"`
}

// HandleMetricsJSON returns an http.HandlerFunc that gathers metrics
// from the default Prometheus registry and returns a JSON summary.
func HandleMetricsJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		families, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		index := make(map[string]*dto.MetricFamily, len(families))
		for _, f := range families {
			index[f.GetName()] = f
		}

		resp := MetricsResponse{
			Events: EventsMetrics{
				Received: sumByLabel(index, "buggregator_events_received_total", "type"),
				Stored:   gaugeByLabel(index, "buggregator_events_stored_total", "type"),
				Errors:   sumAll(index, "buggregator_events_ingestion_errors_total"),
			},
			Connections: ConnectionsMetrics{
				WebSocket: gaugeScalar(index, "buggregator_ws_connections_active"),
				TCP:       gaugeByLabel(index, "buggregator_tcp_connections_active", "server"),
			},
			HTTP: HTTPMetrics{
				RequestsTotal: sumAll(index, "buggregator_http_requests_total"),
			},
			Webhooks: webhooksFromFamily(index),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// sumByLabel sums counter values grouped by a specific label.
func sumByLabel(index map[string]*dto.MetricFamily, name, labelName string) map[string]float64 {
	result := make(map[string]float64)
	f, ok := index[name]
	if !ok {
		return result
	}
	for _, m := range f.GetMetric() {
		lv := getLabelValue(m, labelName)
		if lv == "" {
			continue
		}
		result[lv] += m.GetCounter().GetValue()
	}
	return result
}

// gaugeByLabel reads gauge values grouped by a specific label.
func gaugeByLabel(index map[string]*dto.MetricFamily, name, labelName string) map[string]float64 {
	result := make(map[string]float64)
	f, ok := index[name]
	if !ok {
		return result
	}
	for _, m := range f.GetMetric() {
		lv := getLabelValue(m, labelName)
		if lv == "" {
			continue
		}
		result[lv] += m.GetGauge().GetValue()
	}
	return result
}

// gaugeScalar reads a single gauge value (no labels).
func gaugeScalar(index map[string]*dto.MetricFamily, name string) float64 {
	f, ok := index[name]
	if !ok {
		return 0
	}
	metrics := f.GetMetric()
	if len(metrics) == 0 {
		return 0
	}
	return metrics[0].GetGauge().GetValue()
}

// sumAll sums all metric values for a counter (across all label combinations).
func sumAll(index map[string]*dto.MetricFamily, name string) float64 {
	f, ok := index[name]
	if !ok {
		return 0
	}
	var total float64
	for _, m := range f.GetMetric() {
		total += m.GetCounter().GetValue()
	}
	return total
}

// webhooksFromFamily extracts sent/failed counts from the webhooks counter.
func webhooksFromFamily(index map[string]*dto.MetricFamily) WebhooksMetrics {
	var wm WebhooksMetrics
	f, ok := index["buggregator_webhooks_sent_total"]
	if !ok {
		return wm
	}
	for _, m := range f.GetMetric() {
		success := getLabelValue(m, "success")
		v := m.GetCounter().GetValue()
		if success == "true" {
			wm.Sent += v
		} else {
			wm.Failed += v
		}
	}
	return wm
}

// getLabelValue extracts the value of a specific label from a metric.
func getLabelValue(m *dto.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}
