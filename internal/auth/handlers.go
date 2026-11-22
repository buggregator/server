package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// stateCookie carries the OAuth state from login to callback; a callback
// whose state differs from it is refused (login CSRF).
const stateCookie = "bg_oauth_state"

// returnCookie carries the page to open after sign-in (?return_to= on login).
const returnCookie = "bg_oauth_return"

// oauthProvider is what the handlers need from the OAuth client.
type oauthProvider interface {
	AuthCodeURL(state string) string
	Exchange(code string) (*User, error)
}

// Handlers provides HTTP handlers for the auth flow.
type Handlers struct {
	oauth  oauthProvider
	tokens *TokenStore
	access AccessPolicy
	// store keeps users, sessions, settings and the audit log; nil keeps the
	// stateless behaviour (token only).
	store *UserStore
	// admins may read the audit log; empty: every signed-in user.
	admins map[string]bool
}

// NewHandlers creates auth HTTP handlers. access limits who may sign in;
// an empty policy lets in everyone the provider lets in.
func NewHandlers(oauth oauthProvider, tokens *TokenStore, access AccessPolicy) *Handlers {
	return &Handlers{oauth: oauth, tokens: tokens, access: access}
}

// SetStore turns on users, server-side sessions (real Logout), per-user
// settings and the audit log. admins are the emails that may read the audit
// log; none: every signed-in user.
func (h *Handlers) SetStore(store *UserStore, admins []string) {
	h.store = store
	h.admins = map[string]bool{}
	for _, e := range admins {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			h.admins[e] = true
		}
	}
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
	mux.Handle("POST /api/auth/logout", authMiddleware(http.HandlerFunc(h.handleLogout)))
	if h.store != nil {
		mux.Handle("GET /api/me/settings", authMiddleware(http.HandlerFunc(h.handleGetSettings)))
		mux.Handle("PUT /api/me/settings", authMiddleware(http.HandlerFunc(h.handlePutSettings)))
		mux.Handle("GET /api/audit", authMiddleware(http.HandlerFunc(h.handleAudit)))
	}
}

// handleLogin redirects the user to the OAuth2 provider's login page.
func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     "/auth/sso",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
	ret := &http.Cookie{Name: returnCookie, Path: "/auth/sso", MaxAge: -1, HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode}
	if p, ok := safeReturnTo(r.URL.Query().Get("return_to")); ok {
		ret.Value, ret.MaxAge = url.QueryEscape(p), 600
	}
	http.SetCookie(w, ret)
	loc := h.oauth.AuthCodeURL(state)
	http.Redirect(w, r, loc, http.StatusFound)
}

// handleCallback handles the OAuth2 callback, exchanges the code for tokens,
// creates an internal JWT, and redirects to the frontend with the token.
func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
	// The state cookie is single-use: clear it whatever the outcome.
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Path: "/auth/sso", MaxAge: -1, HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: returnCookie, Path: "/auth/sso", MaxAge: -1, HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode})
	if !validState(r) {
		slog.Warn("SSO callback refused: state mismatch", "remote", r.RemoteAddr)
		authPage(w, http.StatusBadRequest, "Sign-in expired", "This sign-in link is not valid any more. Start the sign-in again.")
		return
	}

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

	if !h.access.Allows(*user) {
		slog.Warn("SSO sign-in refused: not on the allow-list", "email", user.Email, "verified", user.EmailVerified, "provider", user.Provider)
		h.audit(r, AuditEntry{UserLabel: labelOf(user), Action: "login_denied", Status: http.StatusForbidden})
		who := user.Email
		if who == "" {
			who = "This account"
		}
		authPage(w, http.StatusForbidden, "Access denied", who+" is not allowed to sign in to this Buggregator. Ask its administrator to add you, or sign in with another account.")
		return
	}

	if h.store != nil {
		if err := h.openSession(r, user); err != nil {
			slog.Error("SSO: save user and session", "err", err)
			http.Error(w, `{"message":"sign-in failed","code":500}`, http.StatusInternalServerError)
			return
		}
		h.audit(r, AuditEntry{UserID: user.ID, UserLabel: labelOf(user), Action: "login", Status: http.StatusOK})
	}

	token, err := h.tokens.Create(*user)
	if err != nil {
		slog.Error("failed to create JWT", "err", err)
		http.Error(w, `{"message":"token creation failed","code":500}`, http.StatusInternalServerError)
		return
	}

	// Redirect to frontend login page with token in query parameter.
	// The frontend router middleware picks up ?token=... and stores it, then
	// opens return_to when given.
	redirectURL := "/login?token=" + url.QueryEscape(token)
	if c, err := r.Cookie(returnCookie); err == nil {
		if raw, err := url.QueryUnescape(c.Value); err == nil {
			if p, ok := safeReturnTo(raw); ok {
				redirectURL += "&return_to=" + url.QueryEscape(p)
			}
		}
	}
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
			"id":         user.ID,
			"provider":   user.Provider,
			"username":   user.Username,
			"email":      user.Email,
			"avatar":     user.Avatar,
			"logout_url": "",
			"is_admin":   h.isAdmin(user),
		},
	})
}

// validState checks the callback's state against the login's cookie.
func validState(r *http.Request) bool {
	c, err := r.Cookie(stateCookie)
	if err != nil || c.Value == "" {
		return false
	}
	got := r.URL.Query().Get("state")
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(got)) == 1
}

// isHTTPS reports whether the browser reached us over HTTPS (directly or
// through a proxy that says so).
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// authPage writes a small HTML page for a refused sign-in.
func authPage(w http.ResponseWriter, status int, title, text string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>` + html.EscapeString(title) +
		`</title></head><body style="font-family:sans-serif;max-width:32rem;margin:4rem auto;padding:0 1rem">` +
		`<h1>` + html.EscapeString(title) + `</h1><p>` + html.EscapeString(text) +
		`</p><p><a href="/auth/sso/login">Sign in again</a></p></body></html>`))
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
