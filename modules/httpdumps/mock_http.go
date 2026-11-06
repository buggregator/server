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

	// Mock bodies are user content served from the UI's origin: run them in an
	// opaque origin, whatever headers the rule sets.
	h := w.Header()
	status, delayMs := http.StatusNotFound, 0
	var body []byte
	if rule == nil && isPreflight(r) {
		// No OPTIONS rule: answer the CORS preflight as the CORS middleware
		// would (it set the Access-Control headers when the origin is allowed).
		status = http.StatusNoContent
	} else if rule == nil {
		h.Set("Content-Type", "application/json")
		body, _ = json.Marshal(map[string]string{
			"error": "no mock rule matched " + r.Method + " " + path,
		})
		body = append(body, '\n')
		if r.Method == http.MethodHead {
			body = nil
		}
	} else {
		for k, v := range rule.Headers {
			if http.CanonicalHeaderKey(k) == "Set-Cookie" {
				if v = scopeMockCookie(v, path); v == "" {
					continue
				}
			}
			h.Set(k, v)
		}
		status, delayMs = rule.Status, rule.DelayMs
		if r.Method != http.MethodHead {
			body = []byte(rule.Body)
		}
	}
	setMockSecurityHeaders(h)
	setSniffedContentType(h, status, body)

	sent := mockSent{Status: status, Headers: h, Body: body, DelayMs: delayMs}
	if rule == nil {
		sent.Near = m.rules.near(r.Method, path, project)
	}
	m.store(r, project, rule, sent)

	if delayMs > 0 {
		select {
		case <-time.After(time.Duration(delayMs) * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}
	w.WriteHeader(status)
	w.Write(body)
}

// setSniffedContentType sets the Content-Type net/http would sniff at write
// time, so the stored copy of the headers shows it too.
func setSniffedContentType(h http.Header, status int, body []byte) {
	if len(body) == 0 || status < 200 || status == http.StatusNoContent || status == http.StatusNotModified {
		return
	}
	if _, ok := h["Content-Type"]; ok {
		return
	}
	if h.Get("Content-Encoding") != "" || h.Get("Transfer-Encoding") != "" {
		return
	}
	h.Set("Content-Type", http.DetectContentType(body))
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

// scopeMockCookie keeps a rule's cookie inside the mock server: no Domain, and
// a Path under MockPrefix (the rule's own path, or the request path's when it
// has none), so a mock cannot set or shadow cookies of the Buggregator UI.
// An unreadable cookie is dropped ("").
func scopeMockCookie(v, requestPath string) string {
	c, err := http.ParseSetCookie(v)
	if err != nil {
		return ""
	}
	c.Domain = ""
	p := c.Path
	if !strings.HasPrefix(p, "/") {
		p = "/"
	}
	c.Path = MockPrefix + p
	return c.String()
}

func setMockSecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("X-Content-Type-Options", "nosniff")
}

// MaxMockSentBody caps the response body kept with a captured mock request.
const MaxMockSentBody = 64 << 10 // 64 KB

// mockSent is the response the mock server sent for one request.
type mockSent struct {
	Status  int
	Headers http.Header
	Body    []byte
	DelayMs int
	Near    []MockNear
}

// mockInfo is the "mock" field of a captured request: the rule that answered
// (null ids when none did) and a copy of what was sent (headers except the
// Date and Content-Length that net/http adds), so the request detail stays
// true after the rule changes. Near lists rules that almost matched an
// unmatched request.
func mockInfo(rule *MockRule, sent mockSent) map[string]any {
	headers := make(map[string]string, len(sent.Headers))
	for k, v := range sent.Headers {
		headers[k] = strings.Join(v, ", ")
	}
	body := sent.Body
	truncated := len(body) > MaxMockSentBody
	if truncated {
		body = body[:MaxMockSentBody]
	}
	info := map[string]any{
		"matched":        rule != nil,
		"rule_id":        nil,
		"rule_name":      nil,
		"status":         sent.Status,
		"headers":        headers,
		"body":           string(body),
		"body_size":      len(sent.Body),
		"body_truncated": truncated,
		"delay_ms":       sent.DelayMs,
	}
	if rule != nil {
		info["rule_id"], info["rule_name"] = rule.ID, rule.Name
	} else {
		near := sent.Near
		if near == nil {
			near = []MockNear{}
		}
		info["near"] = near
	}
	return info
}

// store captures the request as an HTTP dump with a "mock" field (see mockInfo).
func (m *mockServer) store(r *http.Request, project string, rule *MockRule, sent mockSent) {
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
	payload["mock"] = mockInfo(rule, sent)
	inc.Payload, _ = json.Marshal(payload)
	inc.Project = project

	if err := m.events.HandleIncoming(r.Context(), inc); err != nil {
		slog.Warn("mock: failed to store request", "err", err)
	}
}
