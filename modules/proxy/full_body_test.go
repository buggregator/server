package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

const bigSize = 2 << 20 // 2 MB, four times the capture limit

func bigBody() []byte { return bytes.Repeat([]byte("0123456789abcdef"), bigSize/16) }

func proxyGet(t *testing.T, s *proxyServer, rawURL string, out *bytes.Buffer) *http.Request {
	t.Helper()
	u, _ := url.Parse(rawURL)
	req := &http.Request{Method: "GET", URL: u, Host: u.Host, Header: http.Header{}, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}
	s.proxyAndRecord(context.Background(), out, req, "http")
	return req
}

func storedBody(t *testing.T, inc []byte, side string) string {
	t.Helper()
	var p map[string]json.RawMessage
	if err := json.Unmarshal(inc, &p); err != nil {
		t.Fatal(err)
	}
	var part struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(p[side], &part); err != nil {
		t.Fatal(err)
	}
	return part.Body
}

func TestProxyAndRecord_ClientGetsWholeBody(t *testing.T) {
	body := bigBody()

	t.Run("with Content-Length", func(t *testing.T) {
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.Write(body)
		}))
		defer up.Close()
		store := &spyStorer{}
		var out bytes.Buffer
		req := proxyGet(t, newProxyServer("", store), up.URL+"/big", &out)

		resp, err := http.ReadResponse(bufio.NewReader(&out), req)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(resp.Body)
		if !bytes.Equal(got, body) || resp.ContentLength != int64(len(body)) {
			t.Errorf("client got %d bytes (Content-Length %d), want %d", len(got), resp.ContentLength, len(body))
		}
		stored := storedBody(t, store.got[0].Payload, "response")
		if !strings.HasSuffix(stored, "[truncated]") || len(stored) > maxBodySize+len("\n[truncated]") {
			t.Errorf("stored body: %d bytes, want the capped copy", len(stored))
		}
	})

	t.Run("chunked, two answers on one connection", func(t *testing.T) {
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i := 0; i < len(body); i += 64 << 10 {
				w.Write(body[i : i+64<<10])
				w.(http.Flusher).Flush()
			}
		}))
		defer up.Close()
		s := newProxyServer("", &spyStorer{})
		var out bytes.Buffer
		req1 := proxyGet(t, s, up.URL+"/one", &out)
		proxyGet(t, s, up.URL+"/two", &out)

		rd := bufio.NewReader(&out)
		for i := 0; i < 2; i++ {
			resp, err := http.ReadResponse(rd, req1)
			if err != nil {
				t.Fatalf("answer %d: %v", i+1, err)
			}
			got, _ := io.ReadAll(resp.Body)
			if !bytes.Equal(got, body) {
				t.Errorf("answer %d: client got %d bytes, want %d", i+1, len(got), len(body))
			}
		}
	})
}

func TestProxyAndRecord_UpstreamGetsWholeRequestBody(t *testing.T) {
	body := bigBody()
	var received int
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		received = len(b)
	}))
	defer up.Close()

	u, _ := url.Parse(up.URL + "/upload")
	req := &http.Request{Method: "POST", URL: u, Host: u.Host, Header: http.Header{}, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
		Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}
	store := &spyStorer{}
	var out bytes.Buffer
	newProxyServer("", store).proxyAndRecord(context.Background(), &out, req, "http")

	if received != len(body) {
		t.Errorf("upstream got %d bytes, want %d", received, len(body))
	}
	if stored := storedBody(t, store.got[0].Payload, "request"); !strings.HasSuffix(stored, "[truncated]") {
		t.Errorf("stored request body: %d bytes, want the capped copy", len(stored))
	}
}
