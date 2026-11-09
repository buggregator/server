package sentry

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
)

// Requests recorded from real Sentry SDKs (JS browser + Node, Python, Go);
// see testdata/sdk/README.md. Bodies are kept as the SDK sent them, so a
// compressed one is still compressed.
type sdkFixture struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	BodyB64 string            `json:"body_b64"`
}

func (f sdkFixture) request(t *testing.T) *http.Request {
	t.Helper()
	body := []byte(f.Body)
	if f.BodyB64 != "" {
		var err error
		if body, err = base64.StdEncoding.DecodeString(f.BodyB64); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(f.Method, f.Path, bytes.NewReader(body))
	for k, v := range f.Headers {
		r.Header.Set(k, v)
	}
	return r
}

// sdkFixtures loads testdata/sdk/<sdk>/*.json in name order.
func sdkFixtures(t *testing.T, sdk string) map[string]sdkFixture {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "sdk", sdk, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures for %s: %v", sdk, err)
	}
	sort.Strings(files)
	out := map[string]sdkFixture{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var fx sdkFixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[filepath.Base(f)] = fx
	}
	return out
}

// ingestSDK sends every recorded request of one SDK through the handler.
func ingestSDK(t *testing.T, sdk string) (*sql.DB, []*event.Incoming) {
	t.Helper()
	db := setupTestDB(t)
	h := &handler{db: db}
	fx := sdkFixtures(t, sdk)
	names := make([]string, 0, len(fx))
	for n := range fx {
		names = append(names, n)
	}
	sort.Strings(names)
	var feed []*event.Incoming
	for _, n := range names {
		inc, err := h.Handle(fx[n].request(t))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if inc != nil {
			feed = append(feed, inc)
		}
	}
	return db, feed
}

// TestSDKFixtures checks what the UI needs from each SDK: the error and the
// message in the feed, their structured rows, a trace that opens, and logs.
func TestSDKFixtures(t *testing.T) {
	for _, tc := range []struct {
		sdk, platform string
		feed          int    // error + message (+ an uncaught one in the browser)
		txName, txOp  string // the root of the recorded trace
		minSpans      int
	}{
		{"node", "node", 2, "GET /orders", "http.server", 2},     // @sentry/node 11: spans streamed as "span" items
		{"python", "python", 2, "GET /orders", "http.server", 2}, // sentry-python 2.74: Brotli bodies
		{"go", "go", 2, "GET /orders", "http.server", 2},         // sentry-go 0.50: "exception" as an array
		{"browser", "javascript", 3, "Pageload", "pageload", 5},  // @sentry/browser 11: text/plain, streamed spans
	} {
		t.Run(tc.sdk, func(t *testing.T) {
			db, feed := ingestSDK(t, tc.sdk)

			if len(feed) != tc.feed {
				t.Fatalf("feed events = %d, want %d", len(feed), tc.feed)
			}
			var texts []string
			for _, inc := range feed {
				var p struct {
					EventID   string          `json:"event_id"`
					Platform  string          `json:"platform"`
					Exception json.RawMessage `json:"exception"`
				}
				if err := json.Unmarshal(inc.Payload, &p); err != nil {
					t.Fatalf("payload: %v", err)
				}
				if inc.Type != "sentry" || p.EventID != inc.UUID || p.Platform != tc.platform {
					t.Errorf("event %s: type %q, payload event_id %q, platform %q", inc.UUID, inc.Type, p.EventID, p.Platform)
				}
				// The frontend reads exception.values only.
				if len(p.Exception) > 0 && p.Exception[0] != '{' {
					t.Errorf("event %s: exception is not an object: %.40s", inc.UUID, p.Exception)
				}
				texts = append(texts, strings.ToLower(string(inc.Payload)))
			}
			joined := strings.Join(texts, "\n")
			if !strings.Contains(joined, "order 15 not found") || !strings.Contains(joined, "payment gateway slow") {
				t.Errorf("feed misses the error or the message")
			}

			var errors, exceptions int
			db.QueryRow(`SELECT COUNT(*) FROM sentry_error_events WHERE platform = ?`, tc.platform).Scan(&errors)
			db.QueryRow(`SELECT COUNT(*) FROM sentry_exceptions WHERE lower(exception_value) LIKE '%order 15 not found%'`).Scan(&exceptions)
			if errors != tc.feed || exceptions == 0 {
				t.Errorf("structured: %d error rows (want %d), %d exception rows", errors, tc.feed, exceptions)
			}

			var traceID, op string
			if err := db.QueryRow(`SELECT trace_id, op FROM sentry_transactions WHERE transaction_name = ?`, tc.txName).Scan(&traceID, &op); err != nil {
				t.Fatalf("transaction %q: %v", tc.txName, err)
			}
			if op != tc.txOp {
				t.Errorf("transaction op = %q, want %q", op, tc.txOp)
			}
			// The trace page opens (it 404s without a transaction row).
			r := httptest.NewRequest("GET", "/api/sentry/traces/"+traceID, nil)
			r.SetPathValue("traceId", traceID)
			w := httptest.NewRecorder()
			handleTraceDetail(db)(w, r)
			var detail struct {
				Spans []struct {
					Op string `json:"op"`
				} `json:"spans"`
			}
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &detail) != nil || len(detail.Spans) < tc.minSpans {
				t.Errorf("trace detail: %d, %d spans (want >= %d)", w.Code, len(detail.Spans), tc.minSpans)
			}
			for _, s := range detail.Spans {
				if s.Op == tc.txOp {
					t.Errorf("the root %q is listed as a span too", s.Op)
				}
			}

			var logs int
			db.QueryRow(`SELECT COUNT(*) FROM sentry_logs WHERE body IN ('User 7 opened cart', 'Stock sync failed')`).Scan(&logs)
			if logs != 2 {
				t.Errorf("logs = %d, want 2", logs)
			}
		})
	}
}

// A segment span sent twice (an SDK retry) is one transaction.
func TestSDKFixtures_SegmentStoredOnce(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}
	fx := sdkFixtures(t, "node")["05-span.json"]
	for range 2 {
		if _, err := h.Handle(fx.request(t)); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions WHERE transaction_name = 'GET /orders'`).Scan(&n)
	if n != 1 {
		t.Errorf("transactions = %d, want 1", n)
	}
}
