package proxy

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// Sequential requests through one proxy reuse one upstream connection
// instead of opening (and leaking) a new pool per request.
func TestProxyAndRecord_ReusesUpstreamConnection(t *testing.T) {
	var conns atomic.Int32
	up := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2")
		w.Write([]byte("ok"))
	}))
	up.Config.ConnState = func(_ net.Conn, st http.ConnState) {
		if st == http.StateNew {
			conns.Add(1)
		}
	}
	up.Start()
	defer up.Close()

	s := newProxyServer("", &spyStorer{})
	defer s.Stop()
	for i := 0; i < 10; i++ {
		var out bytes.Buffer
		proxyGet(t, s, up.URL+"/ping", &out)
		if !strings.HasSuffix(out.String(), "\r\n\r\nok") {
			t.Fatalf("request %d: unexpected answer %q", i, out.String())
		}
	}
	if n := conns.Load(); n != 1 {
		t.Fatalf("upstream connections opened: %d, want 1", n)
	}
}

// An answer of unknown length goes to an HTTP/1.0 client close-delimited,
// never chunked (RFC 9112 §7.1); an HTTP/1.1 client gets it chunked.
func TestProxyAndRecord_UnknownLengthFraming(t *testing.T) {
	body := bigBody()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body[:1024])
		w.(http.Flusher).Flush() // no Content-Length: upstream answers chunked
		w.Write(body[1024:])
	}))
	defer up.Close()

	for _, tc := range []struct {
		minor       int
		wantChunked bool
		wantKeep    bool
	}{
		{minor: 0, wantChunked: false, wantKeep: false},
		{minor: 1, wantChunked: true, wantKeep: true},
	} {
		t.Run("HTTP/1."+strconv.Itoa(tc.minor), func(t *testing.T) {
			u, _ := url.Parse(up.URL + "/big")
			req := &http.Request{Method: "GET", URL: u, Host: u.Host, Header: http.Header{},
				Proto: "HTTP/1." + strconv.Itoa(tc.minor), ProtoMajor: 1, ProtoMinor: tc.minor}
			var out bytes.Buffer
			keep := newProxyServer("", &spyStorer{}).proxyAndRecord(context.Background(), &out, req, "http")

			head, got, found := strings.Cut(out.String(), "\r\n\r\n")
			if !found {
				t.Fatalf("no header end in %q", out.String()[:min(out.Len(), 200)])
			}
			chunked := strings.Contains(strings.ToLower(head), "transfer-encoding: chunked")
			if chunked != tc.wantChunked {
				t.Fatalf("chunked = %v, want %v; header:\n%s", chunked, tc.wantChunked, head)
			}
			if keep != tc.wantKeep {
				t.Fatalf("keepAlive = %v, want %v", keep, tc.wantKeep)
			}
			if !tc.wantChunked {
				if !strings.Contains(head, "Connection: close") {
					t.Fatalf("no Connection: close in header:\n%s", head)
				}
				if got != string(body) {
					t.Fatalf("client got %d bytes, want %d", len(got), len(body))
				}
			}
		})
	}
}
