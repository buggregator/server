package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// With cors.origins "*" other pages may call the API, but not /api/update:
// no CORS headers there and its preflight is not answered.
func TestCORS_UpdateIsSameOriginOnly(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	h := CORSMiddleware(next, []string{"*"})
	tests := []struct {
		method, path string
		code         int
		acao         string
	}{
		{"OPTIONS", "/api/events", http.StatusNoContent, "*"},
		{"OPTIONS", "/api/update", http.StatusMethodNotAllowed, ""},
		{"OPTIONS", "/api/update/check", http.StatusMethodNotAllowed, ""},
		{"POST", "/api/update", http.StatusMethodNotAllowed, ""},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "http://127.0.0.1:8000"+tt.path, nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tt.code || rec.Header().Get("Access-Control-Allow-Origin") != tt.acao {
			t.Errorf("%s %s = %d, ACAO %q; want %d, %q", tt.method, tt.path, rec.Code, rec.Header().Get("Access-Control-Allow-Origin"), tt.code, tt.acao)
		}
	}
}
