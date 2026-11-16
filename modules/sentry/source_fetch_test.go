package sentry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPrivateAddressOnly(t *testing.T) {
	for _, tc := range []struct {
		addr  string
		allow bool
	}{
		{"127.0.0.1:5173", true},
		{"127.10.0.1:80", true},
		{"[::1]:5173", true},
		{"10.1.2.3:80", true},
		{"172.18.0.5:3000", true}, // docker network
		{"192.168.65.254:80", true},
		{"[fd00::1]:80", true},
		{"8.8.8.8:80", false},
		{"[2606:4700::1111]:443", false},
		{"169.254.169.254:80", false}, // cloud metadata
		{"[fd00:ec2::254]:80", false}, // AWS IMDS over IPv6, inside fc00::/7
		{"[fd00:ec2::253]:80", true},
		{"[fe80::1]:80", false},
		{"0.0.0.0:80", false},
		{"[::ffff:8.8.8.8]:80", false},
	} {
		err := privateAddressOnly("tcp", tc.addr, nil)
		if (err == nil) != tc.allow {
			t.Errorf("%s: err = %v, want allow = %v", tc.addr, err, tc.allow)
		}
	}
}

// source serves a file and counts the requests.
func source(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case strings.HasPrefix(r.URL.Path, "/redirect-same"):
			http.Redirect(w, r, "/app.js", http.StatusFound)
		case strings.HasPrefix(r.URL.Path, "/redirect-other"):
			http.Redirect(w, r, r.URL.Query().Get("to"), http.StatusFound)
		default:
			_, _ = w.Write([]byte("line1\nline2"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestSourceFetchPrivate(t *testing.T) {
	srv, _ := source(t)
	fetch := newSourceFetcher(SourceFetchPrivate)

	if text, ok := fetch(srv.URL + "/app.js"); !ok || text != "line1\nline2" {
		t.Fatalf("loopback: ok = %v, text = %q", ok, text)
	}
	if _, ok := fetch(srv.URL + "/redirect-same"); !ok {
		t.Error("a redirect on the same host is followed")
	}
	// 127.0.0.1 → localhost is another host: not followed.
	other := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1) + "/app.js"
	if _, ok := fetch(srv.URL + "/redirect-other?to=" + other); ok {
		t.Error("a redirect to another host is not followed")
	}
	// Refused before connect: no packet leaves.
	for _, u := range []string{"http://8.8.8.8:9/app.js", "http://169.254.169.254/latest/meta-data/"} {
		if _, ok := fetch(u); ok {
			t.Errorf("%s fetched", u)
		}
	}
}

func TestSourceFetchAny(t *testing.T) {
	srv, _ := source(t)
	fetch := newSourceFetcher(SourceFetchAny)
	other := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1) + "/app.js"
	if _, ok := fetch(srv.URL + "/redirect-other?to=" + other); !ok {
		t.Error("any: a redirect to another host is followed")
	}
}

func TestSourceFetchOff(t *testing.T) {
	srv, hits := source(t)
	h := &handler{fetchSource: newSourceFetcher(SourceFetchOff)}
	payload := `{"event_id":"e1","platform":"javascript","exception":{"values":[{"type":"Error","value":"boom",
		"stacktrace":{"frames":[{"filename":"` + srv.URL + `/app.js","lineno":1,"colno":1}]}}]}}`
	req := httptest.NewRequest("POST", "/api/1/store/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if _, err := h.Handle(req); err != nil {
		t.Fatal(err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("off: %d requests to the source server, want 0", n)
	}

	// The same event with the private default fetches the loopback source.
	h = &handler{}
	req = httptest.NewRequest("POST", "/api/1/store/", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if _, err := h.Handle(req); err != nil {
		t.Fatal(err)
	}
	if hits.Load() == 0 {
		t.Fatal("default: the loopback source was not fetched")
	}
}
