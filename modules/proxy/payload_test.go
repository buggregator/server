package proxy

import (
	"encoding/json"
	"testing"
)

func TestBuildHTTPDumpEvent_KeepsScheme(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		t.Run(scheme, func(t *testing.T) {
			req := &capturedRequest{Method: "GET", URI: "/orders", Host: "api.example.com", Scheme: scheme}
			inc := buildHTTPDumpEvent(req, &capturedResponse{StatusCode: 200}, 1, "")

			var payload map[string]any
			if err := json.Unmarshal(inc.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["scheme"] != scheme {
				t.Errorf("scheme = %v, want %q", payload["scheme"], scheme)
			}
			if payload["host"] != "api.example.com" {
				t.Errorf("host = %v", payload["host"])
			}
		})
	}
}
