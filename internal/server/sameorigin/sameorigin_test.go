package sameorigin

import (
	"net/http/httptest"
	"testing"
)

func TestCheck(t *testing.T) {
	for _, tc := range []struct {
		name string
		hdr  []string
		want bool
	}{
		{"UI, same origin", []string{"Sec-Fetch-Site", "same-origin", "Origin", "http://127.0.0.1:8000"}, true},
		{"curl, no headers", nil, true},
		{"typed by the user", []string{"Sec-Fetch-Site", "none"}, true},
		{"no Sec-Fetch, Origin of this host", []string{"Origin", "http://127.0.0.1:8000"}, true},
		{"other port of localhost", []string{"Sec-Fetch-Site", "same-site", "Origin", "http://localhost:5173"}, false},
		{"cross-site", []string{"Sec-Fetch-Site", "cross-site"}, false},
		{"no Sec-Fetch, other port", []string{"Origin", "http://127.0.0.1:5173"}, false},
		{"no Sec-Fetch, Origin null", []string{"Origin", "null"}, false},
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8000/api/x", nil)
		for i := 0; i+1 < len(tc.hdr); i += 2 {
			r.Header.Set(tc.hdr[i], tc.hdr[i+1])
		}
		if got := Check(r); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLocal(t *testing.T) {
	for _, tc := range []struct {
		name, peer, host string
		hdr              []string
		want             bool
	}{
		{"loopback v4", "127.0.0.1:41000", "127.0.0.1:8000", nil, true},
		{"loopback v6, localhost", "[::1]:41000", "localhost:8000", nil, true},
		{"LAN peer", "192.168.1.50:41000", "192.168.1.20:8000", nil, false},
		{"Docker bridge gateway", "172.17.0.1:41000", "localhost:8000", nil, false},
		{"DNS rebinding: foreign Host", "127.0.0.1:41000", "evil.example:8000", nil, false},
		{"local proxy that says so", "127.0.0.1:41000", "127.0.0.1:8000", []string{"X-Forwarded-For", "203.0.113.9"}, false},
		{"Forwarded", "127.0.0.1:41000", "127.0.0.1:8000", []string{"Forwarded", "for=203.0.113.9"}, false},
		{"X-Real-IP", "127.0.0.1:41000", "127.0.0.1:8000", []string{"X-Real-IP", "203.0.113.9"}, false},
	} {
		r := httptest.NewRequest("POST", "http://"+tc.host+"/api/x", nil)
		r.RemoteAddr = tc.peer
		for i := 0; i+1 < len(tc.hdr); i += 2 {
			r.Header.Set(tc.hdr[i], tc.hdr[i+1])
		}
		if got := Local(r); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}
