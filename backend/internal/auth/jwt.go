package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload.
type Claims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
}

// JWTManager creates and validates JWT tokens.
type JWTManager struct {
	secret     []byte
	expiration time.Duration
}

// NewJWTManager creates a JWTManager with the given secret and expiration.
func NewJWTManager(secret string, expiration time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), expiration: expiration}
}

// Generate creates a signed JWT for the given user.
func (m *JWTManager) Generate(userID int64, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "rakutao",
		},
		UserID: userID,
		Email:  email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Validate parses and validates the token string, returning the claims.
func (m *JWTManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	return claims, nil
}
