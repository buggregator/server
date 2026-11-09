package http_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/buggregator/server/v2/internal/auth"
	"github.com/buggregator/server/v2/internal/module"
	serverhttp "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/server/ws"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/profiler"
	"github.com/buggregator/server/v2/modules/sentry"
	smtpmod "github.com/buggregator/server/v2/modules/smtp"
)

// newRoutedMux wires the module UI APIs, attachment API and ingestion the way
// app.Run does, with the auth middleware on or off.
func newRoutedMux(t *testing.T, authOn bool) (*http.ServeMux, *auth.TokenStore) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	store := storage.NewSQLiteStore(db)
	attachments := storage.NewAttachmentStore("memory", "")
	registry := module.NewRegistry()
	registry.Register(sentry.New())
	registry.Register(profiler.New())
	registry.Register(smtpmod.New(":0", attachments, db))

	mux := http.NewServeMux()
	if err := registry.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	tokens := auth.NewTokenStore("secret")
	mw := auth.Middleware(tokens, authOn)
	registry.RegisterProtectedRoutes(mux, mw)
	serverhttp.RegisterAttachmentAPI(mux, db, attachments, mw)

	es := serverhttp.NewEventService(store, ws.NewHub(), registry, nil, db)
	mux.Handle("/", serverhttp.NewIngestionPipeline(registry.Handlers(), es, fstest.MapFS{}))
	return mux, tokens
}

// uiRoutes are the module UI API routes that must sit behind the API auth.
var uiRoutes = []struct{ method, path string }{
	{"GET", "/api/sentry/exceptions"},
	{"GET", "/api/sentry/exceptions/1"},
	{"GET", "/api/sentry/traces"},
	{"GET", "/api/sentry/traces/abc"},
	{"GET", "/api/sentry/logs"},
	{"GET", "/api/sentry/metrics"},
	{"GET", "/api/sentry/service-map"},
	{"GET", "/api/sentry/counts"},
	{"DELETE", "/api/sentry/all"},
	{"GET", "/api/profiler/p1/summary"},
	{"GET", "/api/profiler/p1/call-graph"},
	{"GET", "/api/profiler/p1/top"},
	{"GET", "/api/profiler/p1/flame-chart"},
	{"GET", "/api/smtp/cursor"},
	{"GET", "/api/smtp/stats"},
	{"DELETE", "/api/smtp/messages"},
	{"GET", "/api/smtp/messages"},
	{"GET", "/api/smtp/message/m1/raw"},
	{"GET", "/api/smtp/message/m1/links"},
	{"GET", "/api/smtp/message/m1/codes"},
	{"GET", "/api/smtp/attachments/m1"},
	{"GET", "/api/http-dump/attachments/m1"},
	{"GET", "/api/http-dumps/attachments/m1"},
}

func serve(mux *http.ServeMux, method, path, token string) int {
	r := httptest.NewRequest(method, path, nil)
	if token != "" {
		r.Header.Set("X-Auth-Token", token)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code
}

func sentryEnvelope(project string) *http.Request {
	payload := `{"message":"boom","level":"error"}`
	body := "{\"event_id\":\"0123456789abcdef0123456789abcdef\"}\n" +
		"{\"type\":\"event\",\"length\":" + strconv.Itoa(len(payload)) + "}\n" + payload
	r := httptest.NewRequest("POST", "/api/"+project+"/envelope/", strings.NewReader(body))
	r.Header.Set("X-Sentry-Auth", "Sentry sentry_key=public")
	return r
}

func TestModuleUIRoutes_AuthOn(t *testing.T) {
	mux, tokens := newRoutedMux(t, true)
	token, err := tokens.Create(auth.User{Provider: "test", Username: "dev"})
	if err != nil {
		t.Fatal(err)
	}

	for _, rt := range uiRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			if code := serve(mux, rt.method, rt.path, ""); code != http.StatusUnauthorized {
				t.Errorf("no token: status = %d, want 401", code)
			}
			if code := serve(mux, rt.method, rt.path, "forged"); code != http.StatusUnauthorized {
				t.Errorf("bad token: status = %d, want 401", code)
			}
			if code := serve(mux, rt.method, rt.path, token); code == http.StatusUnauthorized {
				t.Errorf("valid token: status = 401, want the handler's answer")
			}
		})
	}

	// SDK ingestion stays open: SDKs carry no session.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, sentryEnvelope("shop"))
	if w.Code != http.StatusOK {
		t.Fatalf("sentry envelope without token: status = %d, want 200", w.Code)
	}
	r := httptest.NewRequest("GET", "/api/sentry/exceptions?project=shop", nil)
	r.Header.Set("X-Auth-Token", token)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "0123456789abcdef0123456789abcdef") {
		t.Fatalf("stored exception not listed with token: %d %s", w.Code, w.Body.String())
	}
}

func TestModuleUIRoutes_AuthOff(t *testing.T) {
	mux, _ := newRoutedMux(t, false)

	for _, rt := range uiRoutes {
		if code := serve(mux, rt.method, rt.path, ""); code == http.StatusUnauthorized {
			t.Errorf("%s %s: status = 401 with auth off", rt.method, rt.path)
		}
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, sentryEnvelope("shop"))
	if w.Code != http.StatusOK {
		t.Fatalf("sentry envelope: status = %d, want 200", w.Code)
	}
	if code := serve(mux, "GET", "/api/sentry/exceptions?project=shop", ""); code != http.StatusOK {
		t.Fatalf("exceptions list with auth off: status = %d, want 200", code)
	}
}
