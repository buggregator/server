package http

import (
	"encoding/base64"
	nethttp "net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDetectEventType(t *testing.T) {
	tests := []struct {
		name        string
		makeRequest func() *nethttp.Request
		wantNil     bool
		wantType    string
		wantProject string
	}{
		{
			name: "URI userinfo type only",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/api/store", nil)
				r.URL.User = url.User("sentry")
				return r
			},
			wantType:    "sentry",
			wantProject: "",
		},
		{
			name: "URI userinfo type and project",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/api/store", nil)
				r.URL.User = url.UserPassword("profiler", "myproject")
				return r
			},
			wantType:    "profiler",
			wantProject: "myproject",
		},
		{
			name: "X-Buggregator-Event header",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("X-Buggregator-Event", "ray")
				r.Header.Set("X-Buggregator-Project", "my-proj")
				return r
			},
			wantType:    "ray",
			wantProject: "my-proj",
		},
		{
			name: "X-Buggregator-Event header without project",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("X-Buggregator-Event", "inspector")
				return r
			},
			wantType:    "inspector",
			wantProject: "",
		},
		{
			name: "Basic Auth with type and project",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("sms:production")))
				return r
			},
			wantType:    "sms",
			wantProject: "production",
		},
		{
			name: "Basic Auth with type only",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("monolog")))
				return r
			},
			wantType:    "monolog",
			wantProject: "",
		},
		{
			name: "Basic Auth invalid base64",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("Authorization", "Basic !!!invalid!!!")
				return r
			},
			wantNil: true,
		},
		{
			name: "no detection method",
			makeRequest: func() *nethttp.Request {
				return httptest.NewRequest("POST", "http://localhost/", nil)
			},
			wantNil: true,
		},
		{
			name: "X-Sentry-Auth header detects sentry",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/api/1/envelope/", nil)
				r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=abc")
				return r
			},
			wantType: "sentry",
		},
		{
			name: "envelope path suffix detects sentry",
			makeRequest: func() *nethttp.Request {
				return httptest.NewRequest("POST", "http://localhost/api/1/envelope", nil)
			},
			wantType: "sentry",
		},
		{
			name: "store path suffix detects sentry",
			makeRequest: func() *nethttp.Request {
				return httptest.NewRequest("POST", "http://localhost/api/1/store", nil)
			},
			wantType: "sentry",
		},
		{
			name: "X-Inspector-Key header detects inspector",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("X-Inspector-Key", "test-key")
				return r
			},
			wantType: "inspector",
		},
		{
			name: "URI userinfo takes priority over headers",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.URL.User = url.User("sentry")
				r.Header.Set("X-Buggregator-Event", "ray")
				return r
			},
			wantType:    "sentry",
			wantProject: "",
		},
		{
			name: "headers take priority over basic auth",
			makeRequest: func() *nethttp.Request {
				r := httptest.NewRequest("POST", "http://localhost/", nil)
				r.Header.Set("X-Buggregator-Event", "ray")
				r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("sentry:proj")))
				return r
			},
			wantType:    "ray",
			wantProject: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.makeRequest()
			got := detectEventType(r)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Project != tt.wantProject {
				t.Errorf("Project = %q, want %q", got.Project, tt.wantProject)
			}
		})
	}
}
