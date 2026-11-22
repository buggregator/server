package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// unsignedJWT builds a compact JWT with the given claims (signature unused:
// the ID token is taken from the token endpoint directly).
func unsignedJWT(claims map[string]any) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	b, _ := json.Marshal(claims)
	return h + "." + base64.RawURLEncoding.EncodeToString(b) + ".sig"
}

// idTokenProvider is an OIDC provider whose token endpoint also returns an
// ID token, like Microsoft Entra. idClaims gets "iss" set to the server URL
// unless it has one.
func idTokenProvider(t *testing.T, userinfo, idClaims map[string]any) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"userinfo_endpoint":      srv.URL + "/userinfo",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 3600}
		if idClaims != nil {
			if _, ok := idClaims["iss"]; !ok {
				idClaims["iss"] = srv.URL
			}
			resp["id_token"] = unsignedJWT(idClaims)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(userinfo)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestExchange_IDTokenClaims(t *testing.T) {
	// Entra's userinfo: sub, name, email, no email_verified.
	entra := map[string]any{"sub": "s1", "name": "Ann Acme", "email": "ann@acme.com"}
	cases := []struct {
		name         string
		userinfo     map[string]any
		idClaims     map[string]any
		wantVerified bool
		wantHD       string
	}{
		{"entra with xms_edov", entra, map[string]any{"aud": "id", "sub": "s1", "xms_edov": true}, true, ""},
		{"entra xms_edov as string", entra, map[string]any{"aud": []any{"other", "id"}, "sub": "s1", "xms_edov": "1"}, true, ""},
		{"entra without xms_edov", entra, map[string]any{"aud": "id", "sub": "s1"}, false, ""},
		{"no id token", entra, nil, false, ""},
		{"wrong audience ignored", entra, map[string]any{"aud": "someone-else", "sub": "s1", "xms_edov": true}, false, ""},
		{"wrong issuer ignored", entra, map[string]any{"iss": "https://evil.test", "aud": "id", "sub": "s1", "xms_edov": true}, false, ""},
		{"other subject ignored", entra, map[string]any{"aud": "id", "sub": "s2", "xms_edov": true}, false, ""},
		{"email_verified and hd from id token", map[string]any{"sub": "s1", "email": "ann@acme.com"},
			map[string]any{"aud": "id", "sub": "s1", "email_verified": true, "hd": "acme.com"}, true, "acme.com"},
		{"userinfo false is not overridden by a missing claim", map[string]any{"sub": "s1", "email": "ann@acme.com", "email_verified": false},
			map[string]any{"aud": "id", "sub": "s1"}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idp := idTokenProvider(t, tc.userinfo, tc.idClaims)
			c, err := NewOAuthClient("oidc", idp.URL, "id", "secret", "http://bg.test/cb", "openid,email,profile")
			if err != nil {
				t.Fatal(err)
			}
			u, err := c.Exchange("code")
			if err != nil {
				t.Fatal(err)
			}
			if u.EmailVerified != tc.wantVerified || u.HostedDomain != tc.wantHD {
				t.Fatalf("verified=%v hd=%q, want %v %q", u.EmailVerified, u.HostedDomain, tc.wantVerified, tc.wantHD)
			}
			allowed := NewAccessPolicy(nil, []string{"acme.com"}).Allows(*u)
			if allowed != tc.wantVerified {
				t.Fatalf("allow-list let in = %v, want %v", allowed, tc.wantVerified)
			}
		})
	}
}

func TestMapUser_NameBeforeLogin(t *testing.T) {
	cases := []struct {
		provider string
		raw      string
		want     string
	}{
		{"keycloak", `{"name":"Ann Acme","preferred_username":"ann","nickname":"annie"}`, "Ann Acme"},
		{"keycloak", `{"preferred_username":"ann","nickname":"annie"}`, "ann"},
		{"auth0", `{"nickname":"annie"}`, "annie"},
		{"github", `{"login":"ann","name":"Ann Acme"}`, "Ann Acme"},
		{"github", `{"login":"ann","name":null}`, "ann"},
	}
	for _, tc := range cases {
		u, err := (&OAuthClient{provider: tc.provider}).mapUser([]byte(tc.raw))
		if err != nil {
			t.Fatal(err)
		}
		if u.Username != tc.want {
			t.Errorf("%s %s: username %q, want %q", tc.provider, tc.raw, u.Username, tc.want)
		}
	}
}

func TestDefaultScopes(t *testing.T) {
	if got := DefaultScopes("github"); got != "read:user,user:email" {
		t.Errorf("github: %q", got)
	}
	for _, p := range []string{"google", "oidc", "keycloak", ""} {
		if got := DefaultScopes(p); got != "openid,email,profile" {
			t.Errorf("%s: %q", p, got)
		}
	}
}
