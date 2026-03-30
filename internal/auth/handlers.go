package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// Handlers provides HTTP handlers for the auth flow.
type Handlers struct {
	oauth  *OAuthClient
	tokens *TokenStore
}

// NewHandlers creates auth HTTP handlers.
func NewHandlers(oauth *OAuthClient, tokens *TokenStore) *Handlers {
	return &Handlers{oauth: oauth, tokens: tokens}
}

// RegisterRoutes registers auth endpoints on the mux.
// These routes are public (no auth middleware).
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/sso/login", h.handleLogin)
	mux.HandleFunc("GET /auth/sso/callback", h.handleCallback)
}

// RegisterProtectedRoutes registers routes that need auth middleware.
func (h *Handlers) RegisterProtectedRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/me", authMiddleware(http.HandlerFunc(h.handleMe)))
}

// handleLogin redirects the user to the OAuth2 provider's login page.
func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	// In a production system you'd store state in a cookie/session for CSRF validation.
	// For simplicity, we skip state validation here (the PHP backend also doesn't validate it).
	url := h.oauth.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// handleCallback handles the OAuth2 callback, exchanges the code for tokens,
// creates an internal JWT, and redirects to the frontend with the token.
func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, `{"message":"missing code parameter","code":400}`, http.StatusBadRequest)
		return
	}

	user, err := h.oauth.Exchange(code)
	if err != nil {
		slog.Error("OAuth exchange failed", "err", err)
		http.Error(w, `{"message":"authentication failed","code":500}`, http.StatusInternalServerError)
		return
	}

	token, err := h.tokens.Create(*user)
	if err != nil {
		slog.Error("failed to create JWT", "err", err)
		http.Error(w, `{"message":"token creation failed","code":500}`, http.StatusInternalServerError)
		return
	}

	// Redirect to frontend login page with token in query parameter.
	// The frontend router middleware picks up ?token=... and stores it.
	redirectURL := "/login?token=" + token
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleMe returns the authenticated user's profile.
func (h *Handlers) handleMe(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		guest := GuestUser()
		user = &guest
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"username":   user.Username,
			"email":      user.Email,
			"avatar":     user.Avatar,
			"logout_url": "",
		},
	})
}

// generateState creates a random state string for OAuth2 CSRF protection.
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// LoginURL returns the SSO login URL path.
// The frontend getter prepends REST_API_URL + "/" before this value.
func LoginURL() string {
	return "auth/sso/login"
}

// SplitScopes splits a comma-separated scopes string.
func SplitScopes(scopes string) []string {
	parts := strings.Split(scopes, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
