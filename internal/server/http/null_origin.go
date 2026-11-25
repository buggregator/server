package http

import (
	"net/http"
	"strings"
)

// NullOriginGuard refuses pages with Origin: null — a sandboxed frame (the
// html event's view), file:// and data: pages — on everything but ingestion.
// Such a page could otherwise read events through CORS * and change or
// delete them over a request that needs no preflight or over the
// WebSocket's RPC bridge. It gets 403 with no CORS headers on the UI and
// management API (/api/…), the WebSocket and /auth/…; ingestion (the
// pipeline's catch-all, Sentry cron check-ins) keeps today's CORS, so a
// page opened from file:// still sends events.
//
// It wraps the CORS middleware; mux is the router it guards, used to tell
// ingestion routes from the rest.
func NullOriginGuard(next http.Handler, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "null" || !guardedPath(r.URL.Path) || isIngestion(mux, r) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"origin_null","error":"not available to pages without an origin (sandboxed frames, file://)"}` + "\n"))
	})
}

func guardedPath(p string) bool {
	return strings.HasPrefix(p, "/api/") || p == "/api" ||
		p == "/connection/websocket" || p == "/ws" ||
		strings.HasPrefix(p, "/auth/")
}

// isIngestion: the route the mux picks for the request (for a preflight,
// for the method it asks about) is the ingestion pipeline's catch-all or a
// Sentry cron check-in.
func isIngestion(mux *http.ServeMux, r *http.Request) bool {
	probe := r
	if r.Method == http.MethodOptions {
		if m := r.Header.Get("Access-Control-Request-Method"); m != "" {
			probe = r.Clone(r.Context())
			probe.Method = m
		}
	}
	_, pattern := mux.Handler(probe)
	return pattern == "/" || strings.Contains(pattern, "/cron/{slug}/{key}")
}
