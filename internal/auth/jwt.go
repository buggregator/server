package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenExpiry = 30 * 24 * time.Hour // 30 days

// TokenClaims are the JWT claims stored in internal tokens.
type TokenClaims struct {
	jwt.RegisteredClaims
	Provider string `json:"provider"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	// EmailVerified and HostedDomain let the access policy re-check a token,
	// so an allow-list set later also cuts sessions issued before it.
	EmailVerified bool   `json:"email_verified,omitempty"`
	HostedDomain  string `json:"hd,omitempty"`
}

// TokenStore creates and validates internal JWT tokens.
type TokenStore struct {
	secret []byte
	access AccessPolicy
}

// NewTokenStore creates a new TokenStore with the given HMAC secret.
func NewTokenStore(secret string) *TokenStore {
	return &TokenStore{secret: []byte(secret)}
}

// SetAccess makes every token check also apply the sign-in policy: a token
// whose user the policy refuses is invalid (HTTP API and WebSocket alike).
func (s *TokenStore) SetAccess(p AccessPolicy) {
	s.access = p
}

// Create issues a new JWT token for the given user.
func (s *TokenStore) Create(u User) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiry)),
		},
		Provider: u.Provider,
		Username: u.Username,
		Email:    u.Email,
		Avatar:   u.Avatar,

		EmailVerified: u.EmailVerified,
		HostedDomain:  u.HostedDomain,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Validate parses and validates a JWT token string, returning the user.
func (s *TokenStore) Validate(tokenStr string) (*User, error) {
	claims, err := s.parse(tokenStr)
	if err != nil {
		return nil, err
	}
	return claims.user(), nil
}

func (c *TokenClaims) user() *User {
	return &User{
		Provider:      c.Provider,
		Username:      c.Username,
		Email:         c.Email,
		Avatar:        c.Avatar,
		EmailVerified: c.EmailVerified,
		HostedDomain:  c.HostedDomain,
	}
}

// ExpiresAt validates a token and returns when it expires (zero: never).
func (s *TokenStore) ExpiresAt(tokenStr string) (time.Time, error) {
	claims, err := s.parse(tokenStr)
	if err != nil {
		return time.Time{}, err
	}
	if claims.ExpiresAt == nil {
		return time.Time{}, nil
	}
	return claims.ExpiresAt.Time, nil
}

func (s *TokenStore) parse(tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if !s.access.Allows(*claims.user()) {
		return nil, fmt.Errorf("token user not allowed by the sign-in policy")
	}
	return claims, nil
}
