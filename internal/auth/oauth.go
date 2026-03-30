package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// ProviderEndpoints holds OAuth2 endpoint URLs and user info URL.
type ProviderEndpoints struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}

// OAuthClient wraps an oauth2.Config and provider-specific user mapping.
type OAuthClient struct {
	config      oauth2.Config
	userInfoURL string
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
	return c.fetchUser(client)
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

	return c.mapUser(body)
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
		u.Username, _ = raw["login"].(string)
		u.Email, _ = raw["email"].(string)
		u.Avatar, _ = raw["avatar_url"].(string)
	default:
		// OIDC-compatible providers (Auth0, Google, Keycloak, Okta, Azure AD, GitLab).
		u.Username = strOr(raw, "nickname", "preferred_username", "name", "login")
		u.Email, _ = raw["email"].(string)
		u.Avatar, _ = raw["picture"].(string)
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
		}, nil
	default:
		// Try OIDC discovery for all other providers.
		return discoverOIDC(providerURL)
	}
}

// oidcDiscoveryDoc is the subset of fields we need from .well-known/openid-configuration.
type oidcDiscoveryDoc struct {
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
