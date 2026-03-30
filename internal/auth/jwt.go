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
}

// TokenStore creates and validates internal JWT tokens.
type TokenStore struct {
	secret []byte
}

// NewTokenStore creates a new TokenStore with the given HMAC secret.
func NewTokenStore(secret string) *TokenStore {
	return &TokenStore{secret: []byte(secret)}
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
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Validate parses and validates a JWT token string, returning the user.
func (s *TokenStore) Validate(tokenStr string) (*User, error) {
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

	return &User{
		Provider: claims.Provider,
		Username: claims.Username,
		Email:    claims.Email,
		Avatar:   claims.Avatar,
	}, nil
}
