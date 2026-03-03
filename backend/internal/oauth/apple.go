package oauth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const appleKeysURL = "https://appleid.apple.com/auth/keys"
const appleIssuer = "https://appleid.apple.com"

// appleJWKS caches Apple's public keys.
var (
	appleKeysMu    sync.RWMutex
	appleKeysCache map[string]*rsa.PublicKey
	appleKeysAt    time.Time
)

// appleJWKSResponse is the JSON structure returned by Apple's JWKS endpoint.
type appleJWKSResponse struct {
	Keys []appleJWK `json:"keys"`
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// appleClaims holds the JWT claims from an Apple ID token.
type appleClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// VerifyAppleToken verifies an Apple ID token using Apple's JWKS public keys.
func VerifyAppleToken(ctx context.Context, idToken, bundleID string) (*OAuthUser, error) {
	keys, err := getAppleKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("oauth/apple: fetch keys: %w", err)
	}

	token, err := jwt.ParseWithClaims(idToken, &appleClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		key, ok := keys[kid]
		if !ok {
			return nil, fmt.Errorf("unknown key id: %s", kid)
		}
		return key, nil
	},
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(bundleID),
	)
	if err != nil {
		return nil, fmt.Errorf("oauth/apple: verify token: %w", err)
	}

	claims, ok := token.Claims.(*appleClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("oauth/apple: invalid claims")
	}

	sub, _ := claims.GetSubject()
	if sub == "" {
		return nil, fmt.Errorf("oauth/apple: missing subject")
	}

	return &OAuthUser{
		ProviderID: sub,
		Email:      claims.Email,
	}, nil
}

// getAppleKeys returns Apple's cached JWKS public keys, refreshing if stale (>1h).
func getAppleKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	appleKeysMu.RLock()
	if appleKeysCache != nil && time.Since(appleKeysAt) < time.Hour {
		defer appleKeysMu.RUnlock()
		return appleKeysCache, nil
	}
	appleKeysMu.RUnlock()

	appleKeysMu.Lock()
	defer appleKeysMu.Unlock()

	// Double-check after acquiring write lock.
	if appleKeysCache != nil && time.Since(appleKeysAt) < time.Hour {
		return appleKeysCache, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apple JWKS returned status %d", resp.StatusCode)
	}

	var jwks appleJWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := parseRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}

	appleKeysCache = keys
	appleKeysAt = time.Now()
	return keys, nil
}

// parseRSAPublicKey builds an RSA public key from base64url-encoded N and E.
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
