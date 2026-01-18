package detect

import (
	"os"
	"path/filepath"
)

// DetectionResult contains the result of project detection
type DetectionResult struct {
	Builder   string   // Primary builder (backward compat) - always equals Builders[0]
	Builders  []string // Ordered builder chain to try (with fallback)
	Reason    string   // Why this builder was selected
	Signals   []string // Files that triggered detection
	HasDocker bool     // Whether Dockerfile was found
}

// DetectBuilder analyzes the source directory and returns the recommended builder
// Priority:
// 1. Dockerfile present → csdocker (user has explicit Docker config, use Vault integration)
// 2. Otherwise → railpack (zero-config auto-detection)
func DetectBuilder(rootDir string) *DetectionResult {
	result := &DetectionResult{
		Builder: "railpack",
		Reason:  "default zero-config builder",
		Signals: []string{},
	}

	// Check for Dockerfile variants
	dockerfiles := []string{
		"Dockerfile",
		"dockerfile",
		"Dockerfile.prod",
		"Dockerfile.production",
		"Dockerfile.dev",
		"Dockerfile.development",
	}

	for _, df := range dockerfiles {
		dockerPath := filepath.Join(rootDir, df)
		if fileExists(dockerPath) {
			result.Builder = "csdocker"
			result.Builders = []string{"csdocker", "railpack", "nixpacks"}
			result.Reason = "Dockerfile found - will try csdocker, fallback to railpack/nixpacks"
			result.Signals = append(result.Signals, df)
			result.HasDocker = true
			return result
		}
	}

	// No Dockerfile - use railpack for auto-detection
	// Railpack will detect: Go, Python, Node, Ruby, Java, PHP, Rust, etc.
	result.Builders = []string{"railpack", "nixpacks"}
	result.Reason = "no Dockerfile found - will try railpack, fallback to nixpacks"

	// Optionally detect what railpack will find (for logging)
	result.Signals = detectProjectSignals(rootDir)

	return result
}

// detectProjectSignals returns files that indicate project type (for logging)
func detectProjectSignals(rootDir string) []string {
	var signals []string

	// Check for common project files
	projectFiles := map[string]string{
		"go.mod":           "Go project",
		"go.work":          "Go workspace",
		"package.json":     "Node.js project",
		"requirements.txt": "Python project",
		"pyproject.toml":   "Python project",
		"Pipfile":          "Python project",
		"Gemfile":          "Ruby project",
		"Cargo.toml":       "Rust project",
		"pom.xml":          "Java/Maven project",
		"build.gradle":     "Java/Gradle project",
		"composer.json":    "PHP project",
		"mix.exs":          "Elixir project",
		"Procfile":         "Procfile-based app",
	}

	for file, desc := range projectFiles {
		if fileExists(filepath.Join(rootDir, file)) {
			signals = append(signals, desc+" ("+file+")")
		}
	}

	return signals
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
}

// HasDockerfile is a convenience function to check if project has Dockerfile
func HasDockerfile(rootDir string) bool {
	result := DetectBuilder(rootDir)
	return result.HasDocker
}

// GetDefaultBuilder returns the default builder for a directory
func GetDefaultBuilder(rootDir string) string {
	result := DetectBuilder(rootDir)
	return result.Builder
}

// GetBuilderChain returns the ordered list of builders to try
// If userBuilder is specified, it becomes first in chain, followed by auto-detected fallbacks
func GetBuilderChain(rootDir string, userBuilder string) []string {
	detection := DetectBuilder(rootDir)

	if userBuilder == "" {
		return detection.Builders
	}

	// User specified a builder - put it first, then add others (avoiding duplicates)
	chain := []string{userBuilder}
	for _, b := range detection.Builders {
		if b != userBuilder {
			chain = append(chain, b)
		}
	}
	return chain
}
