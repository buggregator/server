package sentry

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct{}

func (h *handler) Priority() int { return 10 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	// Sentry sends X-Sentry-Auth header or path ends with /store or /envelope.
	if r.Header.Get("X-Sentry-Auth") != "" {
		return true
	}
	path := r.URL.Path
	return strings.HasSuffix(path, "/store") || strings.HasSuffix(path, "/envelope")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Try to extract event_id from payload for UUID.
	var parsed map[string]any
	uuid := ""
	if json.Unmarshal(body, &parsed) == nil {
		if id, ok := parsed["event_id"].(string); ok {
			uuid = id
		}
	}

	// Extract project from path: /api/{project}/store
	project := extractProject(r.URL.Path)

	return &event.Incoming{
		UUID:    uuid,
		Type:    "sentry",
		Payload: json.RawMessage(body),
		Project: project,
	}, nil
}

func extractProject(path string) string {
	// Path format: /api/{project}/store or /api/{project}/envelope
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
