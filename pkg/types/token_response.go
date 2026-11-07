package types

// TokenResponse is returned by the registry on successful authentication
type TokenResponse struct {
	Token string `json:"token"`
	// AccessToken is the OAuth 2.0 compatible name for the token, which some registries use instead
	AccessToken string `json:"access_token"`
}
