package http

import (
	"net/http"
	"strings"
)

// CORSMiddleware adds CORS headers based on allowed origins.
// If origins contains "*", all origins are allowed.
// A preflight (OPTIONS) is answered here with 204, except under the
// passPreflight path prefixes: there it gets the CORS headers and goes on to
// the handler (the mock server, whose OPTIONS rules must see it).
func CORSMiddleware(next http.Handler, origins []string, passPreflight ...string) http.Handler {
	allowAll := false
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && (allowAll || allowed[origin]) {
			allow := origin
			if allowAll {
				allow = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", allow)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Auth-Token, X-Sentry-Auth, Authorization")
			if !allowAll {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions && !hasPrefix(r.URL.Path, passPreflight) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func hasPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
