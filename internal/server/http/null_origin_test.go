package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	serverhttp "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/server/ws"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/sentry"
)

// guardedHandler wires the core API, Sentry, the WebSocket and ingestion,
// then CORS "*" and the null-origin guard, as app.Run does with auth off.
func guardedHandler(t *testing.T) (http.Handler, event.Store) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewSQLiteStore(db)
	hub := ws.NewHub()
	registry := module.NewRegistry()
	registry.Register(sentry.New())
	mux := http.NewServeMux()
	if err := registry.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	noop := func(next http.Handler) http.Handler { return next }
	registry.RegisterProtectedRoutes(mux, noop)
	es := serverhttp.NewEventService(store, hub, registry, nil, db)
	serverhttp.RegisterAPI(mux, store, registry.Previews(), es, "test", db, []string{"sentry"}, serverhttp.AuthSettings{}, noop, serverhttp.ListLimits{}, nil, map[string]int{"http": 8000})
	mux.HandleFunc("GET /connection/websocket", hub.HandleUpgrade)
	mux.HandleFunc("GET /ws", hub.HandleUpgrade)
	mux.Handle("/", serverhttp.NewIngestionPipeline(registry.Handlers(), es, fstest.MapFS{"index.html": {Data: []byte("spa")}}))

	h := serverhttp.CORSMiddleware(mux, []string{"*"}, "/mock/")
	return serverhttp.NullOriginGuard(h, mux), store
}

func TestNullOriginGuard(t *testing.T) {
	h, store := guardedHandler(t)
	if err := store.Store(t.Context(), event.Event{UUID: "e1", Type: "sentry", Payload: []byte(`{}`), Project: "default", Timestamp: 1}); err != nil {
		t.Fatal(err)
	}
	envelope := "{\"event_id\":\"0123456789abcdef0123456789abcdef\"}\n{\"type\":\"event\"}\n{\"message\":\"boom\",\"level\":\"error\"}"

	tests := []struct {
		name, origin, method, path, preflight, body string
		code                                        int
		acao                                        string
	}{
		// A sandboxed html event's script (Origin: null): no UI API.
		{"read events", "null", "GET", "/api/events", "", "", 403, ""},
		{"read one", "null", "GET", "/api/event/e1", "", "", 403, ""},
		{"delete preflight", "null", "OPTIONS", "/api/event/e1", "DELETE", "", 403, ""},
		{"delete", "null", "DELETE", "/api/event/e1", "", "", 403, ""},
		{"clear all", "null", "DELETE", "/api/events", "", "", 403, ""},
		{"pin, no preflight", "null", "POST", "/api/event/e1/pin", "", "", 403, ""},
		{"settings", "null", "GET", "/api/settings", "", "", 403, ""},
		{"sentry ui", "null", "GET", "/api/sentry/exceptions", "", "", 403, ""},
		{"websocket", "null", "GET", "/connection/websocket", "", "", 403, ""},
		{"websocket /ws", "null", "GET", "/ws", "", "", 403, ""},
		// Ingestion keeps its CORS: a page from file:// still sends events.
		{"sentry envelope", "null", "POST", "/api/1/envelope/", "", envelope, 200, "*"},
		{"sentry envelope preflight", "null", "OPTIONS", "/api/1/envelope/", "POST", "", 204, "*"},
		{"sentry store preflight", "null", "OPTIONS", "/api/1/store/", "POST", "", 204, "*"},
		{"sentry cron check-in", "null", "GET", "/api/1/cron/nightly/public", "", "", 0, "*"},
		{"http dump", "null", "POST", "/anything", "", "hi", 200, "*"},
		// Other origins and no origin: as before.
		{"other page reads", "http://localhost:5173", "GET", "/api/events", "", "", 200, "*"},
		{"no origin reads", "", "GET", "/api/events", "", "", 200, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://127.0.0.1:8000"+tt.path, strings.NewReader(tt.body))
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.preflight != "" {
				r.Header.Set("Access-Control-Request-Method", tt.preflight)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if tt.code != 0 && w.Code != tt.code {
				t.Errorf("code %d, want %d (%s)", w.Code, tt.code, w.Body.String())
			}
			if tt.code == 0 && w.Code == http.StatusForbidden {
				t.Errorf("refused: %s", w.Body.String())
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.acao {
				t.Errorf("Access-Control-Allow-Origin %q, want %q", got, tt.acao)
			}
		})
	}
	// The refused delete did not happen.
	if ev, _ := store.FindByUUID(t.Context(), "e1"); ev == nil {
		t.Error("event e1 was deleted by an Origin: null request")
	}
}
