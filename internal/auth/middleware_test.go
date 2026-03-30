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
