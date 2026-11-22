package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_Disabled(t *testing.T) {
	tokens := NewTokenStore("secret")
	mw := Middleware(tokens, false)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestMiddleware_Enabled_NoToken(t *testing.T) {
	tokens := NewTokenStore("secret")
	mw := Middleware(tokens, true)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMiddleware_Enabled_ValidToken(t *testing.T) {
	tokens := NewTokenStore("secret")
	mw := Middleware(tokens, true)

	user := User{Provider: "test", Username: "john", Email: "john@test.com"}
	token, _ := tokens.Create(user)

	var gotUser *User
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/events", nil)
	r.Header.Set("X-Auth-Token", token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if gotUser == nil {
		t.Fatal("user not set in context")
	}
	if gotUser.Username != "john" {
		t.Errorf("Username = %q, want %q", gotUser.Username, "john")
	}
}

func TestMiddleware_Enabled_InvalidToken(t *testing.T) {
	tokens := NewTokenStore("secret")
	mw := Middleware(tokens, true)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/api/events", nil)
	r.Header.Set("X-Auth-Token", "bad-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// A token issued before an allow-list was set is cut once it is set:
// both the HTTP middleware and the WebSocket check (ExpiresAt) refuse it.
func TestMiddleware_AllowListCutsOldTokens(t *testing.T) {
	tokens := NewTokenStore("secret")
	stranger, _ := tokens.Create(User{Provider: "google", Username: "bob", Email: "bob@gmail.com", EmailVerified: true})
	member, _ := tokens.Create(User{Provider: "google", Username: "ann", Email: "ann@acme.com", EmailVerified: true, HostedDomain: "acme.com"})
	// A token from before this change has no email_verified claim.
	legacy, _ := NewTokenStore("secret").Create(User{Provider: "google", Username: "ann", Email: "ann@acme.com"})

	tokens.SetAccess(NewAccessPolicy(nil, []string{"acme.com"}))
	mw := Middleware(tokens, true)
	ok := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

	for name, tc := range map[string]struct {
		token string
		want  int
	}{
		"stranger": {stranger, http.StatusUnauthorized},
		"legacy":   {legacy, http.StatusUnauthorized},
		"member":   {member, http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
			req.Header.Set("X-Auth-Token", tc.token)
			rec := httptest.NewRecorder()
			ok.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("HTTP status %d, want %d", rec.Code, tc.want)
			}
			_, err := tokens.ExpiresAt(tc.token)
			if (err == nil) != (tc.want == http.StatusOK) {
				t.Fatalf("ExpiresAt err = %v, want valid=%v", err, tc.want == http.StatusOK)
			}
		})
	}
}
