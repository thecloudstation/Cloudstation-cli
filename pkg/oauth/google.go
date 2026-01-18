// Package oauth provides OAuth 2.0 authentication support for CloudStation CLI.
// It implements PKCE (Proof Key for Code Exchange) flow for secure authentication
// without requiring a client secret.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// GoogleClientID is CloudStation's Google OAuth client ID for Desktop apps.
	// Set via environment variable CS_GOOGLE_CLIENT_ID
	GoogleClientID = ""

	// GoogleClientSecret is required for token exchange.
	// Set via environment variable CS_GOOGLE_CLIENT_SECRET
	GoogleClientSecret = ""
)

// GoogleOAuth manages the Google OAuth 2.0 PKCE authentication flow.
// It handles authorization URL generation and code exchange for tokens.
type GoogleOAuth struct {
	config       *oauth2.Config
	codeVerifier string
	state        string
	port         int
}

// NewGoogleOAuth creates a new GoogleOAuth instance configured for PKCE flow.
// Uses dynamic port allocation for the callback server (like gcloud, firebase CLI).
// Returns an error if cryptographic random generation fails.
func NewGoogleOAuth(port int) (*GoogleOAuth, error) {
	// Generate PKCE code verifier (32 random bytes, base64 URL encoded)
	// Per RFC 7636, the code verifier should be a high-entropy cryptographic random string
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)

	// Generate state parameter (16 random bytes, base64 URL encoded)
	// State is used to prevent CSRF attacks by correlating requests and responses
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Create OAuth2 config for Google
	// Use localhost (not 127.0.0.1) - Google allows any port for Desktop OAuth clients
	config := &oauth2.Config{
		ClientID:     GoogleClientID,
		ClientSecret: GoogleClientSecret,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
		RedirectURL:  fmt.Sprintf("http://localhost:%d/callback", port),
	}

	return &GoogleOAuth{
		config:       config,
		codeVerifier: codeVerifier,
		state:        state,
		port:         port,
	}, nil
}

// GetPort returns the port used for the callback server.
func (g *GoogleOAuth) GetPort() int {
	return g.port
}

// GetAuthURL generates the OAuth authorization URL with PKCE challenge.
// The returned URL should be opened in the user's browser to initiate the OAuth flow.
// It uses S256 (SHA256) code challenge method as per RFC 7636.
func (g *GoogleOAuth) GetAuthURL() string {
	// Generate S256 code challenge from verifier
	// code_challenge = BASE64URL(SHA256(code_verifier))
	hash := sha256.Sum256([]byte(g.codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return g.config.AuthCodeURL(g.state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// ExchangeCode exchanges the authorization code for an OAuth token.
// The code is received from the OAuth callback after user authorization.
// The PKCE code verifier is automatically included in the token request.
func (g *GoogleOAuth) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code cannot be empty")
	}

	// Exchange the authorization code for a token
	// Include the code_verifier for PKCE verification
	token, err := g.config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", g.codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	return token, nil
}

// GetState returns the state parameter for verification during callback.
// The callback handler should verify that the state in the response matches
// this value to prevent CSRF attacks.
func (g *GoogleOAuth) GetState() string {
	return g.state
}
