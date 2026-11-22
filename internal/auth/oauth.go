package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

// ProviderEndpoints holds OAuth2 endpoint URLs and user info URL.
type ProviderEndpoints struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	// EmailsURL lists the user's emails with their verified flag (GitHub).
	EmailsURL string
	// Issuer is the OIDC issuer from discovery; empty for GitHub.
	Issuer string
}

// OAuthClient wraps an oauth2.Config and provider-specific user mapping.
type OAuthClient struct {
	config      oauth2.Config
	userInfoURL string
	emailsURL   string
	issuer      string
	provider    string
}

// NewOAuthClient builds an OAuthClient from auth configuration.
// For OIDC-compatible providers it attempts discovery via .well-known/openid-configuration.
// For GitHub it uses hardcoded endpoints.
func NewOAuthClient(provider, providerURL, clientID, clientSecret, callbackURL, scopes string) (*OAuthClient, error) {
	scopeList := strings.Split(scopes, ",")
	for i := range scopeList {
		scopeList[i] = strings.TrimSpace(scopeList[i])
	}

	endpoints, err := resolveEndpoints(provider, providerURL)
	if err != nil {
		return nil, fmt.Errorf("resolve oauth endpoints: %w", err)
	}

	cfg := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  callbackURL,
		Scopes:       scopeList,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthURL,
			TokenURL: endpoints.TokenURL,
		},
	}

	return &OAuthClient{
		config:      cfg,
		userInfoURL: endpoints.UserInfoURL,
		emailsURL:   endpoints.EmailsURL,
		issuer:      endpoints.Issuer,
		provider:    provider,
	}, nil
}

// AuthCodeURL returns the URL to redirect the user to for authorization.
func (c *OAuthClient) AuthCodeURL(state string) string {
	return c.config.AuthCodeURL(state)
}

// Exchange trades an authorization code for tokens and fetches user info.
func (c *OAuthClient) Exchange(code string) (*User, error) {
	token, err := c.config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	client := c.config.Client(oauth2.NoContext, token)
	u, err := c.fetchUser(client)
	if err != nil {
		return nil, err
	}
	if raw, ok := token.Extra("id_token").(string); ok && raw != "" {
		c.applyIDToken(u, raw)
	}
	return u, nil
}

// applyIDToken fills what the userinfo answer lacks from the ID token:
// email_verified, hd, and Microsoft Entra's xms_edov. The token came straight
// from the token endpoint over TLS, which OIDC Core 3.1.3.7 accepts in place
// of its signature; issuer, audience and subject are still checked, and a
// token that fails them is ignored.
func (c *OAuthClient) applyIDToken(u *User, raw string) {
	claims, ok := decodeJWTClaims(raw)
	if !ok {
		return
	}
	if !audienceHas(claims["aud"], c.config.ClientID) {
		return
	}
	if iss, _ := claims["iss"].(string); c.issuer != "" && strings.TrimRight(iss, "/") != strings.TrimRight(c.issuer, "/") {
		return
	}
	if sub, _ := claims["sub"].(string); u.Subject != "" && sub != u.Subject {
		return
	}
	if u.HostedDomain == "" {
		u.HostedDomain, _ = claims["hd"].(string)
	}
	// The token's verdict applies only to the same address, and never
	// overrides an explicit email_verified from userinfo.
	tokenEmail, _ := claims["email"].(string)
	sameEmail := tokenEmail == "" || u.Email == "" || strings.EqualFold(tokenEmail, u.Email)
	if !u.EmailVerified && !u.verifiedSet && sameEmail {
		// Microsoft Entra sends no email_verified; its optional claim
		// xms_edov says the email's domain is verified by the tenant.
		u.EmailVerified = claimTrue(claims["email_verified"]) || claimTrue(claims["xms_edov"])
	}
	if u.Email == "" && u.EmailVerified {
		u.Email, _ = claims["email"].(string)
	}
}

// decodeJWTClaims returns the payload of a compact JWT without checking it.
func decodeJWTClaims(raw string) (map[string]any, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	if json.Unmarshal(body, &claims) != nil {
		return nil, false
	}
	return claims, true
}

// audienceHas reports whether the aud claim (a string or a list) names id.
func audienceHas(aud any, id string) bool {
	switch a := aud.(type) {
	case string:
		return a == id
	case []any:
		for _, v := range a {
			if v == id {
				return true
			}
		}
	}
	return false
}

// fetchUser retrieves the user profile from the provider's userinfo endpoint.
func (c *OAuthClient) fetchUser(client *http.Client) (*User, error) {
	resp, err := client.Get(c.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read user info: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info returned %d: %s", resp.StatusCode, body)
	}

	u, err := c.mapUser(body)
	if err != nil {
		return nil, err
	}
	if c.emailsURL != "" {
		c.fillVerifiedEmail(client, u)
	}
	return u, nil
}

// fillVerifiedEmail sets the user's primary verified email from the
// provider's emails list (GitHub /user/emails, scope user:email). Without
// that scope the list is refused: the public profile email is kept, which
// GitHub only lets a user pick from their verified emails.
func (c *OAuthClient) fillVerifiedEmail(client *http.Client, u *User) {
	resp, err := client.Get(c.emailsURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			u.Email = e.Email
			u.EmailVerified = true
			return
		}
	}
	u.EmailVerified = false
}

// mapUser parses provider-specific JSON into a User.
func (c *OAuthClient) mapUser(body []byte) (*User, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}

	u := &User{Provider: c.provider}

	switch c.provider {
	case "github":
		u.Username = strOr(raw, "name", "login")
		// The numeric account id: stable when the login or email changes.
		if id, ok := raw["id"].(float64); ok {
			u.Subject = strconv.FormatFloat(id, 'f', -1, 64)
		}
		u.Email, _ = raw["email"].(string)
		u.Avatar, _ = raw["avatar_url"].(string)
		u.EmailVerified = u.Email != ""
	default:
		// OIDC-compatible providers (Auth0, Google, Keycloak, Okta, Azure AD, GitLab).
		u.Subject, _ = raw["sub"].(string)
		u.Username = strOr(raw, "name", "preferred_username", "nickname", "login")
		u.Email, _ = raw["email"].(string)
		u.Avatar, _ = raw["picture"].(string)
		u.EmailVerified = claimTrue(raw["email_verified"])
		_, u.verifiedSet = raw["email_verified"]
		u.HostedDomain, _ = raw["hd"].(string)
	}

	return u, nil
}

// resolveEndpoints determines OAuth2 endpoints for the given provider.
func resolveEndpoints(provider, providerURL string) (*ProviderEndpoints, error) {
	providerURL = strings.TrimRight(providerURL, "/")

	switch provider {
	case "github":
		return &ProviderEndpoints{
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			UserInfoURL: "https://api.github.com/user",
			EmailsURL:   "https://api.github.com/user/emails",
		}, nil
	default:
		// Try OIDC discovery for all other providers.
		return discoverOIDC(providerURL)
	}
}

// oidcDiscoveryDoc is the subset of fields we need from .well-known/openid-configuration.
type oidcDiscoveryDoc struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// discoverOIDC fetches the OIDC discovery document.
func discoverOIDC(issuer string) (*ProviderEndpoints, error) {
	if issuer == "" {
		return nil, fmt.Errorf("provider_url is required for OIDC discovery")
	}

	url := issuer + "/.well-known/openid-configuration"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned status %d", resp.StatusCode)
	}

	var doc oidcDiscoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse OIDC discovery: %w", err)
	}

	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC discovery document missing required endpoints")
	}

	userInfoURL := doc.UserinfoEndpoint
	if userInfoURL == "" {
		userInfoURL = issuer + "/userinfo"
	}

	return &ProviderEndpoints{
		AuthURL:     doc.AuthorizationEndpoint,
		TokenURL:    doc.TokenEndpoint,
		UserInfoURL: userInfoURL,
		Issuer:      doc.Issuer,
	}, nil
}

// strOr returns the first non-empty string value for the given keys.
func strOr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// claimTrue reads a boolean claim; some providers send it as "true", "1" or 1.
func claimTrue(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true") || b == "1"
	case float64:
		return b == 1
	}
	return false
}

// DefaultScopes returns the scopes to ask for when none are configured.
// GitHub is not OIDC: its scopes for the profile and the email list are
// read:user and user:email.
func DefaultScopes(provider string) string {
	if provider == "github" {
		return "read:user,user:email"
	}
	return "openid,email,profile"
}
