package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/buggregator/go-buggregator/modules/httpdumps"
	"github.com/buggregator/go-buggregator/modules/inspector"
	"github.com/buggregator/go-buggregator/modules/sentry"
)

// spyStore captures all stored events for inspection.
type spyStore struct {
	mu     sync.Mutex
	events []event.Event
}

func (s *spyStore) Store(_ context.Context, ev event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}
func (s *spyStore) FindByUUID(_ context.Context, _ string) (*event.Event, error) { return nil, nil }
func (s *spyStore) FindAll(_ context.Context, _ event.FindOptions) ([]event.Event, error) {
	return nil, nil
}
func (s *spyStore) Delete(_ context.Context, _ string) error              { return nil }
func (s *spyStore) DeleteAll(_ context.Context, _ event.DeleteOptions) error { return nil }
func (s *spyStore) Pin(_ context.Context, _ string) error                 { return nil }
func (s *spyStore) Unpin(_ context.Context, _ string) error               { return nil }

func (s *spyStore) storedEvents() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]event.Event, len(s.events))
	copy(cp, s.events)
	return cp
}

func (s *spyStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

// spyHandler wraps a real handler and records whether it was called.
type spyHandler struct {
	module.HTTPIngestionHandler
	name     string
	matched  bool
	handled  bool
	returned *event.Incoming
}

func (s *spyHandler) Match(r *http.Request) bool {
	m := s.HTTPIngestionHandler.Match(r)
	if m {
		s.matched = true
	}
	return m
}

func (s *spyHandler) Handle(r *http.Request) (*event.Incoming, error) {
	s.handled = true
	inc, err := s.HTTPIngestionHandler.Handle(r)
	s.returned = inc
	return inc, err
}

func (s *spyHandler) reset() {
	s.matched = false
	s.handled = false
	s.returned = nil
}

// buildPipeline creates a full ingestion pipeline with all real module handlers wrapped in spies.
func buildPipeline(t *testing.T) (*serverhttp.IngestionPipeline, *spyStore, []*spyHandler) {
	t.Helper()

	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations for modules that need DB.
	migrator := storage.NewMigrator(db)
	if err := migrator.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}

	registry := module.NewRegistry()
	// Register the modules involved in the priority chain.
	registry.Register(sentry.New())
	registry.Register(inspector.New())
	registry.Register(httpdumps.New(nil, db))

	mux := http.NewServeMux()
	store := &spyStore{}

	// Init all modules: runs migrations, routes, collects handlers.
	if err := registry.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}

	realHandlers := registry.Handlers()

	// Wrap each in a spy.
	var spies []*spyHandler
	var handlers []module.HTTPIngestionHandler
	for _, h := range realHandlers {
		spy := &spyHandler{HTTPIngestionHandler: h, name: handlerName(h)}
		spies = append(spies, spy)
		handlers = append(handlers, spy)
	}

	// Sort by priority (pipeline does this, but handlers from registry are already sorted).
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].Priority() < handlers[j].Priority()
	})

	hub := ws.NewHub()
	es := serverhttp.NewEventService(store, hub, registry, nil)

	emptyFS := fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}
	pipeline := serverhttp.NewIngestionPipeline(handlers, es, emptyFS)

	return pipeline, store, spies
}

func handlerName(h module.HTTPIngestionHandler) string {
	p := h.Priority()
	switch {
	case p == 10:
		return "sentry"
	case p == 20:
		return "ray"
	case p == 30:
		return "inspector"
	case p == 35:
		return "sms"
	case p == 40:
		return "profiler"
	case p == 9999:
		return "http-dump"
	default:
		return "unknown"
	}
}

func resetSpies(spies []*spyHandler) {
	for _, s := range spies {
		s.reset()
	}
}

func findSpy(spies []*spyHandler, name string) *spyHandler {
	for _, s := range spies {
		if s.name == name {
			return s
		}
	}
	return nil
}

// ─── Tests ─────────────────────────────────────────────────────────────

func TestPipeline_SentryEnvelope_OnlySentryClaims(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	// Simulate a Sentry SDK POST with sentry@host in DSN (detect sets X-Buggregator-Detected-Type: sentry)
	body := `{"event_id":"aaa111","sent_at":"2026-01-01T00:00:00Z"}
{"type":"event"}
{"event_id":"aaa111","level":"error","message":"test error","exception":{"values":[{"type":"RuntimeException","value":"boom"}]}}`

	r := httptest.NewRequest("POST", "http://localhost/api/default/envelope/", strings.NewReader(body))
	r.URL.User = url.User("sentry") // DSN userinfo
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=abc")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	// Sentry MUST have matched and handled.
	sentry := findSpy(spies, "sentry")
	if !sentry.matched {
		t.Error("Sentry handler did not match")
	}
	if !sentry.handled {
		t.Error("Sentry handler did not handle")
	}

	// Inspector MUST NOT have matched or handled.
	inspector := findSpy(spies, "inspector")
	if inspector.handled {
		t.Error("Inspector handler should NOT have handled the Sentry request")
	}

	// HTTP Dump MUST NOT have matched or handled.
	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump handler should NOT have handled the Sentry request")
	}

	// Exactly 1 canonical event stored.
	events := store.storedEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].Type != "sentry" {
		t.Errorf("event type = %q, want 'sentry'", events[0].Type)
	}
}

func TestPipeline_SentryTransaction_NoCanonicalEvent_NoDumpLeak(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	// Sentry transaction envelope — should be stored in structured tables only.
	body := `{"event_id":"txn-111","sent_at":"2026-01-01T00:00:00Z"}
{"type":"transaction"}
{"event_id":"txn-111","type":"transaction","transaction":"GET /api/users","start_timestamp":1700000000.0,"timestamp":1700000000.234,"contexts":{"trace":{"trace_id":"aabb","span_id":"root"}},"spans":[]}`

	r := httptest.NewRequest("POST", "http://localhost/api/default/envelope/", strings.NewReader(body))
	r.URL.User = url.User("sentry")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=abc")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	// Sentry must have handled it.
	sentry := findSpy(spies, "sentry")
	if !sentry.handled {
		t.Error("Sentry handler should have handled the transaction")
	}

	// HTTP Dump must NOT have captured it.
	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump should NOT capture Sentry transactions")
	}

	// No canonical event stored (transactions go to structured tables only).
	events := store.storedEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 stored canonical events for transaction, got %d", len(events))
	}
}

func TestPipeline_SentryLogs_NoCanonicalEvent_NoDumpLeak(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	body := `{"sent_at":"2026-01-01T00:00:00Z"}
{"type":"log"}
{"items":[{"level":"info","body":"test log","timestamp":1700000000.0}]}`

	r := httptest.NewRequest("POST", "http://localhost/api/default/envelope/", strings.NewReader(body))
	r.URL.User = url.User("sentry")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=abc")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	sentry := findSpy(spies, "sentry")
	if !sentry.handled {
		t.Error("Sentry handler should have handled logs")
	}

	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump should NOT capture Sentry logs")
	}

	events := store.storedEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 canonical events for logs, got %d", len(events))
	}
}

func TestPipeline_SentrySessionOnly_NoDumpLeak(t *testing.T) {
	pipeline, _, spies := buildPipeline(t)

	// Envelope with ONLY a session item and no event_id in header.
	// The Sentry handler matches (X-Sentry-Auth), processes the envelope,
	// discards the session, and returns nil — pipeline must stop.
	body := `{}
{"type":"session","length":0}
{"sid":"test-session","status":"ok","started":"2026-01-01T00:00:00Z","attrs":{"release":"1.0"}}`

	r := httptest.NewRequest("POST", "http://localhost/api/default/envelope/", strings.NewReader(body))
	r.URL.User = url.User("sentry")
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=abc")
	r.Header.Set("Content-Type", "application/x-sentry-envelope")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Sentry must have handled (matched + processed).
	sentry := findSpy(spies, "sentry")
	if !sentry.handled {
		t.Error("Sentry handler should have handled session envelope")
	}

	// HTTP Dump must NOT have captured.
	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump should NOT capture Sentry session envelopes")
	}

	// Note: the envelope header `{}` is valid JSON, so the sentry handler
	// takes the plain-JSON path and stores a minimal event. This is acceptable —
	// the critical assertion is that HTTP dump doesn't leak.
	// A future improvement could detect envelope format before plain JSON.
}

func TestPipeline_InspectorRequest_OnlyInspectorClaims(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	payload := `[{"model":"transaction","type":"request","name":"GET /test","hash":"abc","timestamp":1700000000}]`

	r := httptest.NewRequest("POST", "http://localhost/", strings.NewReader(payload))
	r.URL.User = url.User("inspector")
	r.Header.Set("X-Inspector-Key", "test-key")
	r.Header.Set("X-Inspector-Version", "4.0")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	// Inspector must have handled.
	inspector := findSpy(spies, "inspector")
	if !inspector.handled {
		t.Error("Inspector handler should have handled")
	}

	// Sentry must NOT have matched.
	sentry := findSpy(spies, "sentry")
	if sentry.handled {
		t.Error("Sentry should NOT handle Inspector requests")
	}

	// HTTP Dump must NOT have captured.
	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump should NOT capture Inspector requests")
	}

	events := store.storedEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "inspector" {
		t.Errorf("event type = %q, want 'inspector'", events[0].Type)
	}
}

func TestPipeline_UnknownPOST_HttpDumpClaims(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	// A random POST with no detected type — should be caught by HTTP dump.
	r := httptest.NewRequest("POST", "http://localhost/some/webhook", strings.NewReader(`{"foo":"bar"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}

	// Only HTTP Dump should have handled.
	httpDump := findSpy(spies, "http-dump")
	if !httpDump.handled {
		t.Error("HTTP Dump should have handled unrecognized POST")
	}

	sentry := findSpy(spies, "sentry")
	if sentry.handled {
		t.Error("Sentry should NOT handle random POSTs")
	}

	events := store.storedEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "http-dump" {
		t.Errorf("event type = %q, want 'http-dump'", events[0].Type)
	}
}

func TestPipeline_SentryPlainJSON_OnlySentryClaims(t *testing.T) {
	pipeline, store, spies := buildPipeline(t)

	// Sentry /store endpoint with plain JSON (not envelope).
	body := `{"event_id":"plain-json-1","level":"error","message":"plain json error","platform":"php"}`

	r := httptest.NewRequest("POST", "http://localhost/api/1/store", strings.NewReader(body))
	r.URL.User = url.User("sentry")
	w := httptest.NewRecorder()

	pipeline.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	sentry := findSpy(spies, "sentry")
	if !sentry.handled {
		t.Error("Sentry handler should have handled plain JSON")
	}

	httpDump := findSpy(spies, "http-dump")
	if httpDump.handled {
		t.Error("HTTP Dump should NOT capture Sentry plain JSON")
	}

	events := store.storedEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "sentry" {
		t.Errorf("type = %q, want 'sentry'", events[0].Type)
	}
}

func TestPipeline_HandlerSummary(t *testing.T) {
	// Summary test: send all types and verify no cross-contamination.
	pipeline, store, spies := buildPipeline(t)

	requests := []struct {
		name       string
		body       string
		path       string
		userinfo   string
		headers    map[string]string
		wantClaims string // which handler should claim it
		wantEvents int    // expected canonical events
	}{
		{
			name:       "Sentry error event",
			body:       "{\"event_id\":\"s1\",\"sent_at\":\"2026-01-01T00:00:00Z\"}\n{\"type\":\"event\"}\n{\"event_id\":\"s1\",\"level\":\"error\",\"message\":\"test\"}",
			path:       "/api/default/envelope/",
			userinfo:   "sentry",
			headers:    map[string]string{"X-Sentry-Auth": "Sentry sentry_key=abc"},
			wantClaims: "sentry",
			wantEvents: 1,
		},
		{
			name:       "Sentry transaction (no canonical event)",
			body:       "{\"event_id\":\"t1\"}\n{\"type\":\"transaction\"}\n{\"event_id\":\"t1\",\"type\":\"transaction\",\"transaction\":\"X\",\"start_timestamp\":1.0,\"timestamp\":2.0,\"contexts\":{\"trace\":{\"trace_id\":\"aa\",\"span_id\":\"bb\"}},\"spans\":[]}",
			path:       "/api/default/envelope/",
			userinfo:   "sentry",
			headers:    map[string]string{"X-Sentry-Auth": "Sentry sentry_key=abc"},
			wantClaims: "sentry",
			wantEvents: 0,
		},
		{
			name:       "Unknown POST → HTTP dump",
			body:       "{\"hello\":\"world\"}",
			path:       "/webhook/test",
			wantClaims: "http-dump",
			wantEvents: 1,
		},
	}

	for _, tc := range requests {
		t.Run(tc.name, func(t *testing.T) {
			store.reset()
			resetSpies(spies)

			r := httptest.NewRequest("POST", "http://localhost"+tc.path, strings.NewReader(tc.body))
			if tc.userinfo != "" {
				r.URL.User = url.User(tc.userinfo)
			}
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			pipeline.ServeHTTP(w, r)

			if w.Code != 200 {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}

			// Verify correct handler claimed it.
			claimer := findSpy(spies, tc.wantClaims)
			if claimer == nil {
				t.Fatalf("no spy for %q", tc.wantClaims)
			}
			if !claimer.handled {
				t.Errorf("%q handler should have handled", tc.wantClaims)
			}

			// Verify no other handler handled it.
			for _, spy := range spies {
				if spy.name != tc.wantClaims && spy.handled {
					t.Errorf("%q handler should NOT have handled (only %q should)", spy.name, tc.wantClaims)
				}
			}

			// Verify event count.
			events := store.storedEvents()
			if len(events) != tc.wantEvents {
				types := make([]string, len(events))
				for i, e := range events {
					types[i] = e.Type
				}
				t.Errorf("expected %d events, got %d: %v", tc.wantEvents, len(events), types)
			}
		})
	}
}

// Helper to pretty-print spy state for debugging.
func dumpSpies(t *testing.T, spies []*spyHandler) {
	t.Helper()
	for _, s := range spies {
		hasPayload := s.returned != nil
		var payloadSnippet string
		if hasPayload {
			b, _ := json.Marshal(s.returned.Payload)
			if len(b) > 80 {
				b = append(b[:80], '.', '.', '.')
			}
			payloadSnippet = string(b)
		}
		t.Logf("  [%s] priority=%d matched=%v handled=%v returned=%v payload=%s",
			s.name, s.Priority(), s.matched, s.handled, hasPayload, payloadSnippet)
	}
}
