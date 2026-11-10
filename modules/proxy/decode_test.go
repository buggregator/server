package proxy

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/buggregator/server/v2/internal/event"
)

const jsonBody = `{"login":"buggregator","stars":1077}`

func gz(b []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func zl(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func rawDeflate(b []byte) []byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func br(b []byte) []byte {
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func TestDecodedBody(t *testing.T) {
	plain := []byte(jsonBody)
	tests := []struct {
		name, encoding string
		body           []byte
		want           []byte
	}{
		{"no encoding", "", plain, plain},
		{"gzip", "gzip", gz(plain), plain},
		{"x-gzip, upper case", "X-GZIP", gz(plain), plain},
		{"deflate (zlib)", "deflate", zl(plain), plain},
		{"deflate (raw)", "deflate", rawDeflate(plain), plain},
		{"br", "br", br(plain), plain},
		{"br then gzip", "br, gzip", gz(br(plain)), plain},
		{"identity", "identity", plain, plain},
		{"unknown kept as is", "zstd", []byte("raw"), []byte("raw")},
		{"broken gzip kept as is", "gzip", []byte("not gzip"), []byte("not gzip")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.encoding != "" {
				h.Set("Content-Encoding", tt.encoding)
			}
			if got := decodedBody(tt.body, h); !bytes.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("cut gzip decodes as far as it goes", func(t *testing.T) {
		long := bytes.Repeat([]byte("line of text\n"), 2000)
		z := gz(long)
		got := decodedBody(z[:len(z)/2], http.Header{"Content-Encoding": {"gzip"}})
		if len(got) == 0 || !bytes.HasPrefix(long, got) {
			t.Errorf("got %d bytes, want a prefix of the text", len(got))
		}
	})

	t.Run("decoded output is capped", func(t *testing.T) {
		big := gz(bytes.Repeat([]byte{'a'}, 4*maxBodySize))
		if got := decodedBody(big, http.Header{"Content-Encoding": {"gzip"}}); len(got) != maxBodySize+1 {
			t.Errorf("len = %d, want %d", len(got), maxBodySize+1)
		}
	})
}

type spyStorer struct {
	mu  sync.Mutex
	got []*event.Incoming
}

func (s *spyStorer) HandleIncoming(_ context.Context, inc *event.Incoming) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = append(s.got, inc)
	return nil
}

// A gzip answer reaches the client unchanged and is stored decoded.
func TestProxyAndRecord_GzipResponse(t *testing.T) {
	zipped := gz([]byte(jsonBody))
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") != "gzip, deflate" {
			t.Errorf("upstream Accept-Encoding = %q", r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(zipped)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL + "/orgs/buggregator")
	req := &http.Request{Method: "GET", URL: u, Host: u.Host, Header: http.Header{"Accept-Encoding": {"gzip, deflate"}}, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}

	store := &spyStorer{}
	s := newProxyServer("", store)
	var out bytes.Buffer
	s.proxyAndRecord(context.Background(), &out, req, "http")

	resp, err := http.ReadResponse(bufio.NewReader(&out), req)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if resp.Header.Get("Content-Encoding") != "gzip" || !bytes.Equal(got, zipped) {
		t.Errorf("client got encoding %q, %d bytes; want the original gzip bytes", resp.Header.Get("Content-Encoding"), len(got))
	}

	if len(store.got) != 1 {
		t.Fatalf("stored %d events, want 1", len(store.got))
	}
	var payload struct {
		Response struct {
			Body    string              `json:"body"`
			Headers map[string][]string `json:"headers"`
		} `json:"response"`
	}
	if err := json.Unmarshal(store.got[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Response.Body != jsonBody {
		t.Errorf("stored body = %q, want the decoded JSON", payload.Response.Body)
	}
	if !strings.Contains(strings.Join(payload.Response.Headers["Content-Encoding"], ","), "gzip") {
		t.Errorf("stored headers lost Content-Encoding: %v", payload.Response.Headers)
	}
}
