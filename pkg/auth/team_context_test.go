package auth

import (
	"os"
	"testing"
)

func TestDecodeJWTClaims(t *testing.T) {
	// JWT payload: {"team_slug":"test-team","exp":1999999999}
	token := "header.eyJ0ZWFtX3NsdWciOiJ0ZXN0LXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"

	claims, err := DecodeJWTClaims(token)
	if err != nil {
		t.Fatalf("DecodeJWTClaims failed: %v", err)
	}

	teamSlug, ok := claims["team_slug"].(string)
	if !ok || teamSlug != "test-team" {
		t.Errorf("Expected team_slug='test-team', got '%v'", claims["team_slug"])
	}
}

func TestDecodeJWTClaims_InvalidFormat(t *testing.T) {
	_, err := DecodeJWTClaims("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid JWT format")
	}
}

func TestGetTeamContext_ExplicitTeam(t *testing.T) {
	// Clear any existing CS_TOKEN
	os.Unsetenv("CS_TOKEN")

	team := GetTeamContext("explicit-team")
	if team != "explicit-team" {
		t.Errorf("Expected 'explicit-team', got '%s'", team)
	}
}

func TestGetTeamContext_EmptyNoToken(t *testing.T) {
	// Clear any existing CS_TOKEN
	os.Unsetenv("CS_TOKEN")

	team := GetTeamContext("")
	if team != "" {
		t.Errorf("Expected empty string, got '%s'", team)
	}
}

func TestGetTeamContext_AutoDetectFromToken(t *testing.T) {
	// Set CS_TOKEN with team_slug claim
	token := "header.eyJ0ZWFtX3NsdWciOiJhdXRvLXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"
	os.Setenv("CS_TOKEN", token)
	defer os.Unsetenv("CS_TOKEN")

	team := GetTeamContext("")
	if team != "auto-team" {
		t.Errorf("Expected 'auto-team' (auto-detected), got '%s'", team)
	}
}

func TestGetTeamContext_ExplicitOverridesToken(t *testing.T) {
	// Set CS_TOKEN with team_slug claim
	token := "header.eyJ0ZWFtX3NsdWciOiJ0b2tlbi10ZWFtIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"
	os.Setenv("CS_TOKEN", token)
	defer os.Unsetenv("CS_TOKEN")

	team := GetTeamContext("explicit-team")
	if team != "explicit-team" {
		t.Errorf("Expected 'explicit-team' (explicit wins over token), got '%s'", team)
	}
}

func TestGetTeamFromToken_NoToken(t *testing.T) {
	os.Unsetenv("CS_TOKEN")

	team := GetTeamFromToken()
	if team != "" {
		t.Errorf("Expected empty string when no token, got '%s'", team)
	}
}

func TestGetTeamFromToken_WithTeamSlug(t *testing.T) {
	// JWT payload: {"team_slug":"my-team","exp":1999999999}
	token := "header.eyJ0ZWFtX3NsdWciOiJteS10ZWFtIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"
	os.Setenv("CS_TOKEN", token)
	defer os.Unsetenv("CS_TOKEN")

	team := GetTeamFromToken()
	if team != "my-team" {
		t.Errorf("Expected 'my-team', got '%s'", team)
	}
}

func TestGetTeamFromToken_NoTeamSlugClaim(t *testing.T) {
	// JWT payload: {"user_id":"123","exp":1999999999} - no team_slug
	token := "header.eyJ1c2VyX2lkIjoiMTIzIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"
	os.Setenv("CS_TOKEN", token)
	defer os.Unsetenv("CS_TOKEN")

	team := GetTeamFromToken()
	if team != "" {
		t.Errorf("Expected empty string when no team_slug claim, got '%s'", team)
	}
}

// TestGetTeamContext_UserToken verifies team extraction from USER_TOKEN
func TestGetTeamContext_UserToken(t *testing.T) {
	// Clear any existing tokens
	os.Unsetenv("USER_TOKEN")
	os.Unsetenv("CS_TOKEN")

	// Set USER_TOKEN with team_slug claim
	token := "header.eyJ0ZWFtX3NsdWciOiJ1c2VyLXRva2VuLXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"
	os.Setenv("USER_TOKEN", token)
	defer os.Unsetenv("USER_TOKEN")

	team := GetTeamContext("")
	if team != "user-token-team" {
		t.Errorf("Expected 'user-token-team' from USER_TOKEN, got '%s'", team)
	}
}

// TestGetTeamFromToken_UserToken verifies team extraction specifically from USER_TOKEN
func TestGetTeamFromToken_UserToken(t *testing.T) {
	// Clear any existing tokens
	os.Unsetenv("USER_TOKEN")
	os.Unsetenv("CS_TOKEN")

	// JWT payload: {"team_slug":"my-user-team","exp":1999999999}
	token := "header.eyJ0ZWFtX3NsdWciOiJteS11c2VyLXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"
	os.Setenv("USER_TOKEN", token)
	defer os.Unsetenv("USER_TOKEN")

	team := GetTeamFromToken()
	if team != "my-user-team" {
		t.Errorf("Expected 'my-user-team' from USER_TOKEN, got '%s'", team)
	}
}

// TestGetTeamFromToken_UserTokenPriority verifies USER_TOKEN takes priority over CS_TOKEN
func TestGetTeamFromToken_UserTokenPriority(t *testing.T) {
	// Set both tokens with different team slugs
	userToken := "header.eyJ0ZWFtX3NsdWciOiJ1c2VyLXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"
	csToken := "header.eyJ0ZWFtX3NsdWciOiJjcy10ZWFtIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"

	os.Setenv("USER_TOKEN", userToken)
	os.Setenv("CS_TOKEN", csToken)
	defer func() {
		os.Unsetenv("USER_TOKEN")
		os.Unsetenv("CS_TOKEN")
	}()

	team := GetTeamFromToken()
	if team != "user-team" {
		t.Errorf("Expected USER_TOKEN to take priority, got '%s' (should be 'user-team', not 'cs-team')", team)
	}
}

// TestGetTeamContext_UserTokenPriority verifies priority in GetTeamContext
func TestGetTeamContext_UserTokenPriority(t *testing.T) {
	// Set both tokens with different team slugs
	userToken := "header.eyJ0ZWFtX3NsdWciOiJ1c2VyLXRlYW0iLCJleHAiOjE5OTk5OTk5OTl9.signature"
	csToken := "header.eyJ0ZWFtX3NsdWciOiJjcy10ZWFtIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"

	os.Setenv("USER_TOKEN", userToken)
	os.Setenv("CS_TOKEN", csToken)
	defer func() {
		os.Unsetenv("USER_TOKEN")
		os.Unsetenv("CS_TOKEN")
	}()

	// Test without explicit team (should use USER_TOKEN)
	team := GetTeamContext("")
	if team != "user-team" {
		t.Errorf("Expected USER_TOKEN to take priority in GetTeamContext, got '%s'", team)
	}

	// Test that explicit team still overrides everything
	team = GetTeamContext("explicit-override")
	if team != "explicit-override" {
		t.Errorf("Expected explicit team to override tokens, got '%s'", team)
	}
}

// TestGetTeamFromToken_NoUserTokenFallbackToCS verifies fallback behavior
func TestGetTeamFromToken_NoUserTokenFallbackToCS(t *testing.T) {
	// Clear USER_TOKEN, only set CS_TOKEN
	os.Unsetenv("USER_TOKEN")

	csToken := "header.eyJ0ZWFtX3NsdWciOiJmYWxsYmFjay10ZWFtIiwiZXhwIjoxOTk5OTk5OTk5fQ.signature"
	os.Setenv("CS_TOKEN", csToken)
	defer os.Unsetenv("CS_TOKEN")

	team := GetTeamFromToken()
	if team != "fallback-team" {
		t.Errorf("Expected fallback to CS_TOKEN when USER_TOKEN not set, got '%s'", team)
	}
}

// TestGetTeamFromToken_NoTokens verifies empty return when neither token is set
func TestGetTeamFromToken_NoTokens(t *testing.T) {
	// Clear both tokens
	os.Unsetenv("USER_TOKEN")
	os.Unsetenv("CS_TOKEN")

	team := GetTeamFromToken()
	if team != "" {
		t.Errorf("Expected empty string when no tokens set, got '%s'", team)
	}
}
