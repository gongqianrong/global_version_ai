package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// googleTokenInfo is the response from Google's tokeninfo endpoint.
type googleTokenInfo struct {
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
}

// VerifyGoogleToken verifies a Google ID token via the tokeninfo endpoint
// and returns the extracted user info.
func VerifyGoogleToken(ctx context.Context, idToken, clientID string) (*OAuthUser, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth/google: build request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth/google: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth/google: invalid token (status %d)", resp.StatusCode)
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("oauth/google: decode response: %w", err)
	}

	if info.Aud != clientID {
		return nil, fmt.Errorf("oauth/google: audience mismatch: got %q, want %q", info.Aud, clientID)
	}

	if info.Sub == "" {
		return nil, fmt.Errorf("oauth/google: missing subject")
	}

	return &OAuthUser{
		ProviderID: info.Sub,
		Email:      info.Email,
		Name:       info.Name,
	}, nil
}
