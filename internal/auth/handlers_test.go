package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeOIDC is a minimal OIDC provider: discovery, token and userinfo.
// userinfo answers with the claims the test sets.
func fakeOIDC(t *testing.T, claims *map[string]any) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"userinfo_endpoint":      srv.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 3600})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(*claims)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// login runs GET /auth/sso/login and returns the state cookie and the
// state sent to the provider.
func login(t *testing.T, mux http.Handler) (*http.Cookie, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/sso/login", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("login status %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == stateCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("state cookie missing or weak: %+v", cookie)
	}
	if got := loc.Query().Get("state"); got != cookie.Value {
		t.Fatalf("state %q sent to provider, cookie %q", got, cookie.Value)
	}
	return cookie, cookie.Value
}

func TestCallback_AllowListAndState(t *testing.T) {
	googleWork := map[string]any{"email": "ann@acme.com", "email_verified": true, "hd": "acme.com", "name": "ann"}
	googlePersonal := map[string]any{"email": "bob@gmail.com", "email_verified": true, "name": "bob"}
	// A Google account with an acme.com address but no Workspace: the email
	// domain alone must not let it in for provider google.
	googleNoHD := map[string]any{"email": "eve@acme.com", "email_verified": true, "name": "eve"}
	unverified := map[string]any{"email": "ann@acme.com", "email_verified": false, "name": "ann"}
	verifiedString := map[string]any{"email": "ann@acme.com", "email_verified": "true", "name": "ann"}

	cases := []struct {
		name     string
		provider string
		emails   []string
		domains  []string
		claims   map[string]any
		state    string // "" = the login's state; "-" = no cookie
		want     int
	}{
		{"no list: anyone", "google", nil, nil, googlePersonal, "", http.StatusFound},
		{"google hd allowed", "google", nil, []string{"acme.com"}, googleWork, "", http.StatusFound},
		{"google other account denied", "google", nil, []string{"acme.com"}, googlePersonal, "", http.StatusForbidden},
		{"google email domain without hd denied", "google", nil, []string{"acme.com"}, googleNoHD, "", http.StatusForbidden},
		{"email listed", "google", []string{"Bob@Gmail.com"}, []string{"acme.com"}, googlePersonal, "", http.StatusFound},
		{"oidc email domain allowed", "oidc", nil, []string{"@acme.com"}, googleNoHD, "", http.StatusFound},
		{"oidc verified as string", "keycloak", nil, []string{"acme.com"}, verifiedString, "", http.StatusFound},
		{"unverified email denied", "oidc", []string{"ann@acme.com"}, nil, unverified, "", http.StatusForbidden},
		{"state mismatch", "google", nil, nil, googlePersonal, "forged", http.StatusBadRequest},
		{"no state cookie", "google", nil, nil, googlePersonal, "-", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := tc.claims
			idp := fakeOIDC(t, &claims)
			oauth, err := NewOAuthClient(tc.provider, idp.URL, "id", "secret", "http://bg.test/auth/sso/callback", "openid,email")
			if err != nil {
				t.Fatal(err)
			}
			tokens := NewTokenStore("test-secret")
			mux := http.NewServeMux()
			NewHandlers(oauth, tokens, NewAccessPolicy(tc.emails, tc.domains)).RegisterRoutes(mux)

			cookie, state := login(t, mux)
			if tc.state != "" && tc.state != "-" {
				state = tc.state
			}
			req := httptest.NewRequest(http.MethodGet, "/auth/sso/callback?code=c&state="+url.QueryEscape(state), nil)
			if tc.state != "-" {
				req.AddCookie(cookie)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d; body %s", rec.Code, tc.want, rec.Body.String())
			}
			// The state cookie is cleared on every callback.
			cleared := false
			for _, c := range rec.Result().Cookies() {
				if c.Name == stateCookie && c.MaxAge < 0 {
					cleared = true
				}
			}
			if !cleared {
				t.Error("state cookie not cleared")
			}
			switch tc.want {
			case http.StatusFound:
				loc := rec.Header().Get("Location")
				if !strings.HasPrefix(loc, "/login?token=") {
					t.Fatalf("redirect %q", loc)
				}
				if _, err := tokens.Validate(strings.TrimPrefix(loc, "/login?token=")); err != nil {
					t.Fatalf("token: %v", err)
				}
			default:
				if loc := rec.Header().Get("Location"); loc != "" {
					t.Fatalf("refused sign-in redirected to %q", loc)
				}
				if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
					t.Errorf("content type %q", rec.Header().Get("Content-Type"))
				}
			}
		})
	}
}

func TestStateCookieSecureBehindHTTPSProxy(t *testing.T) {
	claims := map[string]any{}
	idp := fakeOIDC(t, &claims)
	oauth, err := NewOAuthClient("oidc", idp.URL, "id", "secret", "https://bg.test/auth/sso/callback", "openid")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandlers(oauth, NewTokenStore("s"), AccessPolicy{}).RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/auth/sso/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == stateCookie && !c.Secure {
			t.Fatal("state cookie not Secure behind an HTTPS proxy")
		}
	}
}

func TestGitHubVerifiedEmail(t *testing.T) {
	emails := `[{"email":"old@acme.com","primary":false,"verified":true},{"email":"me@acme.com","primary":true,"verified":true}]`
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"login":"me","email":null}`))
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(emails))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &OAuthClient{provider: "github", userInfoURL: srv.URL + "/user", emailsURL: srv.URL + "/user/emails"}
	u, err := c.fetchUser(srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "me@acme.com" || !u.EmailVerified {
		t.Fatalf("got %+v", u)
	}

	emails = `[{"email":"me@acme.com","primary":true,"verified":false}]`
	u, err = c.fetchUser(srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if u.EmailVerified {
		t.Fatalf("unverified primary taken as verified: %+v", u)
	}
	if NewAccessPolicy(nil, []string{"acme.com"}).Allows(*u) {
		t.Fatal("unverified GitHub email let in")
	}
}
