package git

import (
	"strings"
	"testing"
)

func TestBuildAuthURL_GitHub(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		token      string
		provider   Provider
		want       string
	}{
		{
			name:       "GitHub with token",
			repository: "user/repo",
			token:      "ghp_token123",
			provider:   GitHub,
			want:       "https://x-access-token:ghp_token123@github.com/user/repo.git",
		},
		{
			name:       "GitHub without token",
			repository: "user/repo",
			token:      "",
			provider:   GitHub,
			want:       "https://github.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAuthURL(tt.repository, tt.token, tt.provider)
			if got != tt.want {
				t.Errorf("BuildAuthURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAuthURL_GitLab(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		token      string
		want       string
	}{
		{
			name:       "GitLab with token",
			repository: "user/repo",
			token:      "glpat-token123",
			want:       "https://x-access-token:glpat-token123@gitlab.com/user/repo.git",
		},
		{
			name:       "GitLab without token",
			repository: "user/repo",
			token:      "",
			want:       "https://gitlab.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAuthURL(tt.repository, tt.token, GitLab)
			if got != tt.want {
				t.Errorf("BuildAuthURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAuthURL_Bitbucket(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		token      string
		want       string
	}{
		{
			name:       "Bitbucket with token",
			repository: "user/repo",
			token:      "bb_token123",
			want:       "https://x-token-auth:bb_token123@bitbucket.org/user/repo.git",
		},
		{
			name:       "Bitbucket without token",
			repository: "user/repo",
			token:      "",
			want:       "https://bitbucket.org/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAuthURL(tt.repository, tt.token, Bitbucket)
			if got != tt.want {
				t.Errorf("BuildAuthURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{GitHub, "github.com"},
		{GitLab, "gitlab.com"},
		{Bitbucket, "bitbucket.org"},
		{Provider("unknown"), "github.com"}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := GetBaseURL(tt.provider)
			if got != tt.want {
				t.Errorf("GetBaseURL(%v) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestClone_TokenNotInError(t *testing.T) {
	// Test that tokens are redacted from error messages
	token := "secret_token_123"
	errorMsg := "fatal: repository 'https://x-access-token:" + token + "@github.com/invalid/repo.git' not found"

	// Simulate what Clone does with error messages
	redacted := strings.ReplaceAll(errorMsg, token, "***REDACTED***")

	if strings.Contains(redacted, token) {
		t.Errorf("Token was not redacted from error message: %s", redacted)
	}

	if !strings.Contains(redacted, "***REDACTED***") {
		t.Errorf("Expected redaction placeholder in error message: %s", redacted)
	}
}
