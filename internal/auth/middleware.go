package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "auth_user"

// Middleware checks the X-Auth-Token header and injects the user into context.
// When auth is enabled, requests without a valid token get 401.
// When auth is disabled, this middleware is a no-op.
func Middleware(tokens *TokenStore, enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !enabled {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.Header.Get("X-Auth-Token")
			if tokenStr == "" {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}

			user, err := tokens.Validate(tokenStr)
			if err != nil {
				http.Error(w, `{"message":"unauthorized","code":401}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extracts the authenticated user from the request context.
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userContextKey).(*User)
	return u
}
