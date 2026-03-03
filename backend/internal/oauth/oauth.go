package oauth

// OAuthUser holds the user information extracted from an OAuth provider's ID token.
type OAuthUser struct {
	ProviderID string
	Email      string
	Name       string
}
