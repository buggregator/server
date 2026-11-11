package http

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// DetectedEvent holds the event type and project extracted from the request.
type DetectedEvent struct {
	Type    string
	Project string
}

// detectEventType extracts event type and project from the request using
// three methods (matching PHP's DetectEventTypeMiddleware):
//
//  1. URI userinfo: http://type@host or http://type:project@host
//  2. Headers: X-Buggregator-Event / X-Buggregator-Project
//  3. Basic Auth: Authorization: Basic base64(type:project)
func detectEventType(r *http.Request) *DetectedEvent {
	// Method 1: URI userinfo (e.g., http://sentry@host, http://profiler@host:8000)
	if r.URL.User != nil {
		username := r.URL.User.Username()
		password, _ := r.URL.User.Password()
		if username != "" {
			return &DetectedEvent{Type: username, Project: password}
		}
	}

	// Method 2: X-Buggregator-Event header
	if eventType := r.Header.Get("X-Buggregator-Event"); eventType != "" {
		return &DetectedEvent{
			Type:    eventType,
			Project: r.Header.Get("X-Buggregator-Project"),
		}
	}

	// Method 3: SDK-specific headers that identify the event type.
	path := strings.TrimRight(r.URL.Path, "/")
	isSentryStore := strings.HasSuffix(path, "/store") && !strings.Contains(path, "/profiler/")
	if r.Header.Get("X-Sentry-Auth") != "" || strings.HasSuffix(path, "/envelope") || isSentryStore {
		// A DSN like http://sentry:shop@host/1 reaches us as Basic Auth on some
		// SDKs: keep its project.
		project := ""
		if typ, p := basicAuth(r); typ == "sentry" {
			project = p
		}
		return &DetectedEvent{Type: "sentry", Project: project}
	}
	if r.Header.Get("X-Inspector-Key") != "" || r.Header.Get("X-Inspector-Version") != "" {
		return &DetectedEvent{Type: "inspector"}
	}
	// Twilio's send API (/2010-04-01/Accounts/{sid}/Messages.json) carries
	// Basic Auth SID:TOKEN, which is no type:project pair.
	if strings.HasPrefix(path, "/2010-04-01/Accounts/") && strings.HasSuffix(path, "/Messages.json") {
		return &DetectedEvent{Type: "sms"}
	}

	// Method 4: Basic Auth (Authorization: Basic base64(type:project))
	if typ, project := basicAuth(r); typ != "" {
		return &DetectedEvent{Type: typ, Project: project}
	}

	return nil
}

// basicAuth reads Authorization: Basic base64(type:project); empty type when
// there is none.
func basicAuth(r *http.Request) (typ, project string) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return "", ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return "", ""
	}
	typ, project, _ = strings.Cut(string(decoded), ":")
	return typ, project
}
