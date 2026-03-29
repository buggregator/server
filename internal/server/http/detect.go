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

	// Method 3: Basic Auth (Authorization: Basic base64(type:project))
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) >= 1 && parts[0] != "" {
				project := ""
				if len(parts) >= 2 {
					project = parts[1]
				}
				return &DetectedEvent{Type: parts[0], Project: project}
			}
		}
	}

	return nil
}
