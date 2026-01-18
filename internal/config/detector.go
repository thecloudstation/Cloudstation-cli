package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectProjectName attempts to detect the project name from various sources
// Priority: 1. Git remote origin, 2. Current directory name
func DetectProjectName() string {
	// Try git remote first
	if name := detectFromGitRemote(); name != "" {
		return name
	}

	// Fall back to directory name
	return detectFromDirectory()
}

// detectFromGitRemote extracts repo name from git remote origin
func detectFromGitRemote() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))
	return extractRepoName(url)
}

// extractRepoName parses git URLs to extract repository name
// Supports: https://github.com/user/repo.git, git@github.com:user/repo.git
func extractRepoName(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH format: git@github.com:user/repo
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			pathParts := strings.Split(parts[1], "/")
			return sanitizeName(pathParts[len(pathParts)-1])
		}
	}

	// Handle HTTPS format: https://github.com/user/repo
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return sanitizeName(parts[len(parts)-1])
	}

	return ""
}

// detectFromDirectory returns the current directory name
func detectFromDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "my-project"
	}
	return sanitizeName(filepath.Base(cwd))
}

// sanitizeName ensures the name is valid for CloudStation (alphanumeric, hyphens, underscores)
func sanitizeName(name string) string {
	// Convert to lowercase first
	name = strings.ToLower(name)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9_-]+`)
	name = reg.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	if name == "" {
		return "my-project"
	}
	return name
}
