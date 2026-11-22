package event

import (
	"net/http"
	"testing"
)

func TestTraceIDFromHeaders(t *testing.T) {
	const id = "771a43a4192642f0b136d5159a501700"
	for _, tc := range []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"sentry-trace sampled", map[string]string{"sentry-trace": id + "-1e57b752bc6e4544-1"}, id},
		{"sentry-trace no flag", map[string]string{"sentry-trace": id + "-1e57b752bc6e4544"}, id},
		{"traceparent", map[string]string{"traceparent": "00-" + id + "-00f067aa0ba902b7-01"}, id},
		{"x-trace-id upper case", map[string]string{"X-Trace-Id": "771A43A4192642F0B136D5159A501700"}, id},
		{"sentry-trace wins", map[string]string{"sentry-trace": id + "-1e57b752bc6e4544-1", "X-Trace-Id": "4bf92f3577b34da6a3ce929d0e0e4736"}, id},
		{"bad sentry-trace falls back", map[string]string{"sentry-trace": "junk", "X-Trace-Id": id}, id},
		{"all zero traceparent", map[string]string{"traceparent": "00-00000000000000000000000000000000-00f067aa0ba902b7-01"}, ""},
		{"short id", map[string]string{"X-Trace-Id": "abc123"}, ""},
		{"not hex", map[string]string{"X-Trace-Id": "zz1a43a4192642f0b136d5159a501700"}, ""},
		{"none", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			for k, v := range tc.headers {
				h.Set(k, v)
			}
			if got := TraceIDFromHeaders(h.Get); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
