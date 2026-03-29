package webhooks

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/metrics"
	"github.com/buggregator/go-buggregator/internal/module"
)

// WebhookConfig defines a single webhook from the configuration.
type WebhookConfig struct {
	Event     string            `yaml:"event"`      // Event type to match (e.g., "sentry", "*" for all).
	URL       string            `yaml:"url"`        // Target URL (http/https).
	Headers   map[string]string `yaml:"headers"`    // Custom HTTP headers.
	VerifySSL bool              `yaml:"verify_ssl"` // Verify SSL certificates.
	Retry     bool              `yaml:"retry"`      // Retry on failure (default: true).
}

// Module implements the webhooks module.
type Module struct {
	module.BaseModule
	webhooks []WebhookConfig
	metrics  *metrics.Collector
	client   *http.Client
}

// New creates a new webhooks module with the given webhook configurations.
func New(webhooks []WebhookConfig, m *metrics.Collector) *Module {
	// Set defaults.
	for i := range webhooks {
		if webhooks[i].Headers == nil {
			webhooks[i].Headers = make(map[string]string)
		}
	}

	return &Module{
		webhooks: webhooks,
		metrics:  m,
		client:   newHTTPClient(webhooks),
	}
}

func (m *Module) Name() string { return "Webhooks" }
func (m *Module) Type() string { return "webhooks" }

// OnEventStored is called after any event is persisted.
// It finds matching webhooks and delivers the event asynchronously.
func (m *Module) OnEventStored(ev event.Event) {
	for _, wh := range m.webhooks {
		if !matchEvent(wh.Event, ev.Type) {
			continue
		}
		go m.deliver(wh, ev)
	}
}

// matchEvent checks if a webhook event pattern matches the event type.
// Supports exact match ("sentry") and wildcard ("*").
func matchEvent(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(pattern, eventType)
}

// webhookPayload is the JSON payload sent to webhook endpoints.
type webhookPayload struct {
	UUID      string          `json:"uuid"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp float64         `json:"timestamp"`
	Project   string          `json:"project,omitempty"`
}

// deliver sends the event to the webhook URL with retries.
func (m *Module) deliver(wh WebhookConfig, ev event.Event) {
	payload := webhookPayload{
		UUID:      ev.UUID,
		Type:      ev.Type,
		Payload:   ev.Payload,
		Timestamp: ev.Timestamp,
		Project:   ev.Project,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: failed to marshal payload", "err", err)
		return
	}

	maxAttempts := 1
	if wh.Retry {
		maxAttempts = 3
	}

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			// Exponential backoff: 5s, 10s.
			delay := time.Duration(5<<uint(attempt-1)) * time.Second
			time.Sleep(delay)
		}

		lastErr = m.doRequest(wh, body)
		if lastErr == nil {
			m.recordMetric(ev.Type, wh.URL, true)
			return
		}

		slog.Warn("webhook delivery failed",
			"url", wh.URL, "event", ev.Type, "attempt", attempt+1,
			"max_attempts", maxAttempts, "err", lastErr,
		)
	}

	m.recordMetric(ev.Type, wh.URL, false)
	slog.Error("webhook delivery exhausted all retries",
		"url", wh.URL, "event", ev.Type, "err", lastErr,
	)
}

// doRequest performs a single HTTP POST to the webhook URL.
func (m *Module) doRequest(wh WebhookConfig, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Buggregator/Webhooks")
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: status %d", resp.StatusCode)
	}

	return nil
}

func (m *Module) recordMetric(eventType, url string, success bool) {
	if m.metrics == nil {
		return
	}
	s := "false"
	if success {
		s = "true"
	}
	m.metrics.WebhooksSentTotal.WithLabelValues(eventType, url, s).Inc()
}

// newHTTPClient creates an HTTP client. If any webhook disables SSL verification,
// a custom transport is used.
func newHTTPClient(webhooks []WebhookConfig) *http.Client {
	skipVerify := false
	for _, wh := range webhooks {
		if !wh.VerifySSL {
			skipVerify = true
			break
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	if skipVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client
}
