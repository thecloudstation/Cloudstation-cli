package config

import (
	"strings"
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH format with .git",
			url:      "git@github.com:user/my-repo.git",
			expected: "my-repo",
		},
		{
			name:     "SSH format without .git",
			url:      "git@github.com:user/my-repo",
			expected: "my-repo",
		},
		{
			name:     "HTTPS format with .git",
			url:      "https://github.com/user/my-repo.git",
			expected: "my-repo",
		},
		{
			name:     "HTTPS format without .git",
			url:      "https://github.com/user/my-repo",
			expected: "my-repo",
		},
		{
			name:     "GitLab SSH format",
			url:      "git@gitlab.com:group/subgroup/project-name.git",
			expected: "project-name",
		},
		{
			name:     "BitBucket HTTPS format",
			url:      "https://bitbucket.org/team/REPO_NAME.git",
			expected: "repo_name",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: "not-a-url",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoName(tt.url)
			if result != tt.expected {
				t.Errorf("extractRepoName(%q) = %q; expected %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Already valid name",
			input:    "my-project",
			expected: "my-project",
		},
		{
			name:     "Uppercase to lowercase",
			input:    "MyProject",
			expected: "myproject",
		},
		{
			name:     "Mixed case with hyphens",
			input:    "My-Cool-Project",
			expected: "my-cool-project",
		},
		{
			name:     "Special characters replaced",
			input:    "my.project@2024",
			expected: "my-project-2024",
		},
		{
			name:     "Leading and trailing hyphens",
			input:    "---project---",
			expected: "project",
		},
		{
			name:     "Underscores preserved",
			input:    "my_project_123",
			expected: "my_project_123",
		},
		{
			name:     "Spaces replaced",
			input:    "my cool project",
			expected: "my-cool-project",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "my-project",
		},
		{
			name:     "Only special chars",
			input:    "@#$%",
			expected: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectProjectName(t *testing.T) {
	// This test just verifies that DetectProjectName returns something valid
	name := DetectProjectName()
	if name == "" {
		t.Error("DetectProjectName() returned empty string")
	}

	// Verify the returned name is sanitized
	if strings.Contains(name, " ") || strings.Contains(name, "@") {
		t.Errorf("DetectProjectName() returned unsanitized name: %q", name)
	}
}
