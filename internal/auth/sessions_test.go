package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/storage"
)

// newUserStore opens a fresh database file with the core migrations.
func newUserStore(t *testing.T) *UserStore {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "bg.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	return NewUserStore(db)
}

func TestUserStore(t *testing.T) {
	s := newUserStore(t)
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	id1, err := s.Upsert(User{Provider: "keycloak", Subject: "s1", Email: "ann@acme.com", Username: "Ann"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.Upsert(User{Provider: "keycloak", Subject: "s1", Email: "ann@new.com", Username: "Ann B"})
	if err != nil || id2 != id1 {
		t.Fatalf("same subject got id %q (err %v), want %q", id2, err, id1)
	}
	var email string
	s.db.QueryRow(`SELECT email FROM users WHERE id = ?`, id1).Scan(&email)
	if email != "ann@new.com" {
		t.Errorf("email not updated: %q", email)
	}
	if id3, _ := s.Upsert(User{Provider: "github", Subject: "s1"}); id3 == id1 {
		t.Error("another provider with the same subject must be another user")
	}

	sid, err := s.CreateSession(id1, "ua", "10.0.0.1", now.Add(time.Hour))
	if err != nil || !s.SessionValid(sid) {
		t.Fatalf("new session not valid (err %v)", err)
	}
	if s.SessionValid("") || s.SessionValid("nope") {
		t.Error("unknown session valid")
	}
	now = now.Add(2 * time.Hour)
	if s.SessionValid(sid) {
		t.Error("expired session valid")
	}
	sid2, _ := s.CreateSession(id1, "", "", now.Add(time.Hour))
	if err := s.Revoke(sid2); err != nil || s.SessionValid(sid2) {
		t.Fatalf("revoked session valid (err %v)", err)
	}

	if err := s.PutSettings(id1, map[string]json.RawMessage{"theme": json.RawMessage(`"dark"`), "cols": json.RawMessage(`[1,2]`)}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.PutSettings(id1, map[string]json.RawMessage{"theme": json.RawMessage(`null`), "x": json.RawMessage(`{"a":1}`)}, 0); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Settings(id1)
	if len(got) != 2 || string(got["cols"]) != `[1,2]` || string(got["x"]) != `{"a":1}` {
		t.Errorf("settings %v", got)
	}
	if other, _ := s.Settings("someone-else"); len(other) != 0 {
		t.Errorf("settings leak to another user: %v", other)
	}

	for i, a := range []string{"login", "events.clear", "logout"} {
		s.Record(AuditEntry{UserID: id1, UserLabel: "ann@acme.com", Action: a, Status: 200 + i})
	}
	page, _ := s.ListAudit(2, 0)
	if len(page) != 2 || page[0].Action != "logout" || page[1].Action != "events.clear" {
		t.Fatalf("first page %+v", page)
	}
	rest, _ := s.ListAudit(2, page[1].ID)
	if len(rest) != 1 || rest[0].Action != "login" {
		t.Fatalf("second page %+v", rest)
	}

	// Prune: audit older than the cut and closed sessions go.
	now = now.Add(100 * 24 * time.Hour)
	s.Record(AuditEntry{Action: "fresh"})
	if err := s.Prune(now.Add(-auditRetention), now); err != nil {
		t.Fatal(err)
	}
	if all, _ := s.ListAudit(10, 0); len(all) != 1 || all[0].Action != "fresh" || all[0].UserLabel != "guest" {
		t.Errorf("after prune %+v", all)
	}
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Errorf("%d closed sessions left", n)
	}
}

// ssoServer wires the auth handlers as app.go does, with one protected
// mutating route standing in for the API.
func ssoServer(t *testing.T, claims *map[string]any, admins []string) (*http.ServeMux, *TokenStore, *UserStore) {
	t.Helper()
	idp := fakeOIDC(t, claims)
	oauth, err := NewOAuthClient("keycloak", idp.URL, "id", "secret", "http://bg.test/auth/sso/callback", "openid,email")
	if err != nil {
		t.Fatal(err)
	}
	store := newUserStore(t)
	tokens := NewTokenStore("test-secret")
	tokens.SetSessions(store)
	mw := AuditMiddleware(store, Middleware(tokens, true))
	mux := http.NewServeMux()
	h := NewHandlers(oauth, tokens, AccessPolicy{})
	h.SetStore(store, admins)
	h.RegisterRoutes(mux)
	h.RegisterProtectedRoutes(mux, mw)
	mux.Handle("DELETE /api/events", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	mux.Handle("GET /api/events", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	return mux, tokens, store
}

// signIn runs login (with return_to) and callback, and returns the redirect.
func signIn(t *testing.T, mux http.Handler, returnTo string) *url.URL {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/sso/login?return_to="+url.QueryEscape(returnTo), nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge > 0 {
			req.AddCookie(c)
		}
	}
	state, _ := req.Cookie(stateCookie)
	req.URL, _ = url.Parse("/auth/sso/callback?code=c&state=" + url.QueryEscape(state.Value))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("callback %d: %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || loc.Path != "/login" {
		t.Fatalf("redirect %q", rec.Header().Get("Location"))
	}
	return loc
}

func call(mux http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("X-Auth-Token", token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSSO_SessionsSettingsAuditLogout(t *testing.T) {
	claims := map[string]any{"sub": "u-1", "email": "ann@acme.com", "email_verified": true, "name": "Ann", "picture": "https://x/a.png"}
	mux, tokens, store := ssoServer(t, &claims, nil)

	loc := signIn(t, mux, "/#/events/sentry/abc")
	token := loc.Query().Get("token")
	if got := loc.Query().Get("return_to"); got != "/#/events/sentry/abc" {
		t.Errorf("return_to %q", got)
	}

	// /api/me carries the users row id.
	rec := call(mux, "GET", "/api/me", token, "")
	var me struct {
		Data map[string]any `json:"data"`
	}
	json.Unmarshal(rec.Body.Bytes(), &me)
	if rec.Code != 200 || me.Data["id"] == "" || me.Data["id"] == nil || me.Data["username"] != "Ann" || me.Data["is_admin"] != true {
		t.Fatalf("me %d %s", rec.Code, rec.Body.String())
	}

	// Settings: merge, null removes, survive a new sign-in of the same user.
	if rec := call(mux, "PUT", "/api/me/settings", token, `{"theme":"dark","layout":{"cols":2}}`); rec.Code != 200 {
		t.Fatalf("put settings %d %s", rec.Code, rec.Body.String())
	}
	call(mux, "PUT", "/api/me/settings", token, `{"theme":null}`)
	token2 := signIn(t, mux, "").Query().Get("token")
	rec = call(mux, "GET", "/api/me/settings", token2, "")
	if strings.TrimSpace(rec.Body.String()) != `{"data":{"layout":{"cols":2}}}` {
		t.Fatalf("settings after re-login: %s", rec.Body.String())
	}
	for _, bad := range []string{`[1]`, `nope`, `{"":1}`, `{"big":"` + strings.Repeat("x", maxSettingValue) + `"}`} {
		if rec := call(mux, "PUT", "/api/me/settings", token, bad); rec.Code < 400 {
			t.Errorf("bad settings %.30q accepted: %d", bad, rec.Code)
		}
	}

	// A mutating API call is audited; a GET is not.
	if rec := call(mux, "DELETE", "/api/events?type=sentry", token, ""); rec.Code != 204 {
		t.Fatalf("delete %d", rec.Code)
	}
	call(mux, "GET", "/api/events", token, "")

	// Logout revokes this session only.
	if rec := call(mux, "POST", "/api/auth/logout", token, ""); rec.Code != 204 {
		t.Fatalf("logout %d", rec.Code)
	}
	if _, err := tokens.Validate(token); err == nil {
		t.Error("token still valid after logout")
	}
	if rec := call(mux, "GET", "/api/me", token, ""); rec.Code != 401 {
		t.Errorf("me after logout %d", rec.Code)
	}
	if _, err := tokens.ExpiresAt(token); err == nil {
		t.Error("WebSocket check still accepts the token after logout")
	}
	if _, err := tokens.Validate(token2); err != nil {
		t.Errorf("other session cut by logout: %v", err)
	}

	// A token from before sessions (no sid) is refused.
	old, _ := NewTokenStore("test-secret").Create(User{Provider: "keycloak", Email: "ann@acme.com"})
	if _, err := tokens.Validate(old); err == nil {
		t.Error("token without session accepted")
	}

	entries, _ := store.ListAudit(50, 0)
	var actions []string
	for _, e := range entries {
		actions = append(actions, e.Action+":"+e.UserLabel)
	}
	want := "logout:ann@acme.com,events.clear:ann@acme.com,login:ann@acme.com,settings.update:ann@acme.com,settings.update:ann@acme.com,login:ann@acme.com"
	if strings.Join(actions, ",") != want {
		t.Errorf("audit\n got %s\nwant %s", strings.Join(actions, ","), want)
	}
	if entries[1].Target != "/api/events?type=sentry" || entries[1].Status != 204 {
		t.Errorf("clear entry %+v", entries[1])
	}

	rec = call(mux, "GET", "/api/audit?limit=2", token2, "")
	var page struct {
		Data       []AuditEntry `json:"data"`
		NextBefore int64        `json:"next_before"`
	}
	json.Unmarshal(rec.Body.Bytes(), &page)
	if rec.Code != 200 || len(page.Data) != 2 || page.NextBefore != page.Data[1].ID {
		t.Fatalf("audit api %d %s", rec.Code, rec.Body.String())
	}
}

func TestSSO_AuditAdminsAndDenied(t *testing.T) {
	claims := map[string]any{"sub": "u-2", "email": "bob@acme.com", "email_verified": true, "name": "Bob"}
	mux, _, store := ssoServer(t, &claims, []string{"Ann@Acme.com"})
	token := signIn(t, mux, "").Query().Get("token")
	if rec := call(mux, "GET", "/api/audit", token, ""); rec.Code != http.StatusForbidden {
		t.Errorf("non-admin audit %d", rec.Code)
	}
	rec := call(mux, "GET", "/api/me", token, "")
	if !strings.Contains(rec.Body.String(), `"is_admin":false`) {
		t.Errorf("me %s", rec.Body.String())
	}
	claims["email"] = "ann@acme.com"
	claims["sub"] = "u-1"
	token = signIn(t, mux, "").Query().Get("token")
	if rec := call(mux, "GET", "/api/audit", token, ""); rec.Code != 200 {
		t.Errorf("admin audit %d", rec.Code)
	}

	// A refused sign-in is recorded too.
	h := NewHandlers(nil, NewTokenStore("x"), NewAccessPolicy([]string{"x@y.z"}, nil))
	h.SetStore(store, nil)
	h.audit(httptest.NewRequest("GET", "/", nil), AuditEntry{UserLabel: "eve@evil.com", Action: "login_denied", Status: 403})
	if e, _ := store.ListAudit(1, 0); len(e) != 1 || e[0].Action != "login_denied" || e[0].IP == "" {
		t.Errorf("denied entry %+v", e)
	}
}

func TestSafeReturnTo(t *testing.T) {
	ok := []string{"/", "/#/events/sentry/123", "/events/sentry/123?x=1", "/smtp/abc#part"}
	bad := []string{"", "events", "//evil.com", "/\\evil.com", "\\\\evil.com", "https://evil.com", "/%2F%2Fevil.com",
		"/%5Cevil.com", "javascript:alert(1)", "/\r\nLocation: x", "/" + strings.Repeat("a", 2048)}
	for _, p := range ok {
		if got, okk := safeReturnTo(p); !okk || got != p {
			t.Errorf("%q refused", p)
		}
	}
	for _, p := range bad {
		if _, okk := safeReturnTo(p); okk {
			t.Errorf("%q accepted", p)
		}
	}
}

// An unsafe return_to is dropped: the sign-in goes on to /login without it.
func TestSSO_ReturnToUnsafeDropped(t *testing.T) {
	claims := map[string]any{"sub": "u-1", "email": "ann@acme.com", "email_verified": true}
	mux, _, _ := ssoServer(t, &claims, nil)
	for _, p := range []string{"//evil.com", "https://evil.com/x", "/\\evil.com"} {
		loc := signIn(t, mux, p)
		if loc.Query().Has("return_to") || loc.Host != "" {
			t.Errorf("%q: redirect %s", p, loc)
		}
	}
}

func TestSkipAudit(t *testing.T) {
	cases := map[string]bool{
		"GET /api/events":                          true,
		"POST /api/mocks/preview":                  true,
		"POST /api/xdebug/sessions/{id}/step-over": true,
		"POST /api/xdebug/sessions/{id}/eval":      false,
		"DELETE /api/events":                       false,
		"POST /api/mocks":                          false,
		"PUT /api/crons/{slug}":                    false,
	}
	for pattern, want := range cases {
		method, _, _ := strings.Cut(pattern, " ")
		r := httptest.NewRequest(method, "/", nil)
		r.Pattern = pattern
		if got := skipAudit(r); got != want {
			t.Errorf("%s: skip %v, want %v", pattern, got, want)
		}
	}
}

// The key cap is checked in the write's transaction: a change over it
// writes nothing.
func TestUserStore_SettingsCap(t *testing.T) {
	s := newUserStore(t)
	v := json.RawMessage(`1`)
	if err := s.PutSettings("u", map[string]json.RawMessage{"a": v, "b": v}, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.PutSettings("u", map[string]json.RawMessage{"c": v}, 2); err != ErrTooManySettings {
		t.Fatalf("err %v, want ErrTooManySettings", err)
	}
	if err := s.PutSettings("u", map[string]json.RawMessage{"a": json.RawMessage(`null`), "c": v}, 2); err != nil {
		t.Fatalf("swap within the cap: %v", err)
	}
	if got, _ := s.Settings("u"); len(got) != 2 || got["c"] == nil {
		t.Errorf("settings %v", got)
	}
}

// ?limit= of GET /api/audit: over the cap clamps to 500, not the default.
func TestAuditLimit(t *testing.T) {
	s := newUserStore(t)
	tx, _ := s.db.Begin()
	for i := 0; i < 600; i++ {
		tx.Exec(`INSERT INTO audit_log (at, user_label, action) VALUES ('2026-11-22T00:00:00.000000Z', 'guest', 'x')`)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	h := NewHandlers(nil, NewTokenStore("x"), AccessPolicy{})
	h.SetStore(s, nil)
	for q, want := range map[string]int{"": 100, "0": 100, "1": 1, "500": 500, "501": 500, "600": 500, "-5": 100, "abc": 100} {
		req := httptest.NewRequest("GET", "/api/audit?limit="+q, nil)
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, &User{Provider: "test", Email: "a@b.c"}))
		rec := httptest.NewRecorder()
		h.handleAudit(rec, req)
		var page struct {
			Data []AuditEntry `json:"data"`
		}
		json.Unmarshal(rec.Body.Bytes(), &page)
		if rec.Code != 200 || len(page.Data) != want {
			t.Errorf("limit=%q: %d rows (status %d), want %d", q, len(page.Data), rec.Code, want)
		}
	}
}
