package httpdumps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// EventStorer stores a captured request (the core EventService).
type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

// mockServer answers requests under MockPrefix and captures each one.
type mockServer struct {
	rules   *mockStore
	capture *handler
	events  EventStorer
}

// mockProject reads the project of a mock request: the X-Buggregator-Project
// header, or Basic Auth "http-dump:<project>" / "mock:<project>". Other Basic
// Auth credentials belong to the mocked API and are left alone.
func mockProject(r *http.Request) string {
	if p := strings.TrimSpace(r.Header.Get("X-Buggregator-Project")); p != "" {
		return p
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Basic ") {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic ")); err == nil {
			user, pass, _ := strings.Cut(string(decoded), ":")
			if (user == "http-dump" || user == "mock") && pass != "" {
				return pass
			}
		}
	}
	return "default"
}

func (m *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, MockPrefix)
	if path == "" {
		path = "/"
	}
	project := mockProject(r)
	rule := m.rules.match(r.Method, path, project)

	m.store(r, project, rule)

	// Mock bodies are user content served from the UI's origin: run them in an
	// opaque origin, whatever headers the rule sets.
	h := w.Header()
	if rule == nil {
		h.Set("Content-Type", "application/json")
		setMockSecurityHeaders(h)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "no mock rule matched " + r.Method + " " + path,
		})
		return
	}

	if rule.DelayMs > 0 {
		select {
		case <-time.After(time.Duration(rule.DelayMs) * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}

	for k, v := range rule.Headers {
		h.Set(k, v)
	}
	setMockSecurityHeaders(h)
	w.WriteHeader(rule.Status)
	if r.Method != http.MethodHead {
		w.Write([]byte(rule.Body))
	}
}

func setMockSecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("X-Content-Type-Options", "nosniff")
}

// store captures the request as an HTTP dump with a "mock" field: the rule
// that answered, or null.
func (m *mockServer) store(r *http.Request, project string, rule *MockRule) {
	if m.events == nil {
		return
	}
	inc, err := m.capture.Handle(r)
	if err != nil {
		slog.Warn("mock: failed to capture request", "err", err)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(inc.Payload, &payload); err != nil {
		slog.Warn("mock: failed to capture request", "err", err)
		return
	}
	var info any // null when no rule matched
	if rule != nil {
		info = map[string]any{"rule_id": rule.ID, "rule_name": rule.Name, "status": rule.Status}
	}
	payload["mock"] = info
	inc.Payload, _ = json.Marshal(payload)
	inc.Project = project

	if err := m.events.HandleIncoming(r.Context(), inc); err != nil {
		slog.Warn("mock: failed to store request", "err", err)
	}
}
