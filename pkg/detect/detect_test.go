package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBuilder(t *testing.T) {
	tests := []struct {
		name            string
		files           []string
		expectedBuilder string
		expectedDocker  bool
		expectedSignals int
	}{
		{
			name:            "Dockerfile present - should use csdocker",
			files:           []string{"Dockerfile", "package.json"},
			expectedBuilder: "csdocker",
			expectedDocker:  true,
			expectedSignals: 1,
		},
		{
			name:            "dockerfile (lowercase) present - should use csdocker",
			files:           []string{"dockerfile", "go.mod"},
			expectedBuilder: "csdocker",
			expectedDocker:  true,
			expectedSignals: 1,
		},
		{
			name:            "Dockerfile.prod present - should use csdocker",
			files:           []string{"Dockerfile.prod", "requirements.txt"},
			expectedBuilder: "csdocker",
			expectedDocker:  true,
			expectedSignals: 1,
		},
		{
			name:            "No Dockerfile, Node.js project - should use railpack",
			files:           []string{"package.json", "package-lock.json"},
			expectedBuilder: "railpack",
			expectedDocker:  false,
			expectedSignals: 1,
		},
		{
			name:            "No Dockerfile, Go project - should use railpack",
			files:           []string{"go.mod", "go.sum"},
			expectedBuilder: "railpack",
			expectedDocker:  false,
			expectedSignals: 1,
		},
		{
			name:            "No Dockerfile, Python project - should use railpack",
			files:           []string{"requirements.txt", "app.py"},
			expectedBuilder: "railpack",
			expectedDocker:  false,
			expectedSignals: 1,
		},
		{
			name:            "Empty directory - should use railpack",
			files:           []string{},
			expectedBuilder: "railpack",
			expectedDocker:  false,
			expectedSignals: 0,
		},
		{
			name:            "Multiple project types, no Docker - should use railpack",
			files:           []string{"package.json", "requirements.txt", "go.mod"},
			expectedBuilder: "railpack",
			expectedDocker:  false,
			expectedSignals: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "detect-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Test DetectBuilder
			result := DetectBuilder(tmpDir)

			if result.Builder != tt.expectedBuilder {
				t.Errorf("Expected builder %s, got %s", tt.expectedBuilder, result.Builder)
			}

			if result.HasDocker != tt.expectedDocker {
				t.Errorf("Expected HasDocker %v, got %v", tt.expectedDocker, result.HasDocker)
			}

			if len(result.Signals) != tt.expectedSignals {
				t.Errorf("Expected %d signals, got %d: %v", tt.expectedSignals, len(result.Signals), result.Signals)
			}

			// Test convenience functions
			if HasDockerfile(tmpDir) != tt.expectedDocker {
				t.Errorf("HasDockerfile expected %v, got %v", tt.expectedDocker, HasDockerfile(tmpDir))
			}

			if GetDefaultBuilder(tmpDir) != tt.expectedBuilder {
				t.Errorf("GetDefaultBuilder expected %s, got %s", tt.expectedBuilder, GetDefaultBuilder(tmpDir))
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "fileexists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !fileExists(testFile) {
		t.Error("fileExists returned false for existing file")
	}

	// Test non-existing file
	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists returned true for non-existing file")
	}

	// Test directory (should return false)
	if fileExists(tmpDir) {
		t.Error("fileExists returned true for directory")
	}
}

func TestDetectProjectSignals(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "signals-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create various project files
	projectFiles := []string{
		"go.mod",
		"package.json",
		"requirements.txt",
		"Cargo.toml",
	}

	for _, file := range projectFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", file, err)
		}
	}

	signals := detectProjectSignals(tmpDir)

	if len(signals) != len(projectFiles) {
		t.Errorf("Expected %d signals, got %d", len(projectFiles), len(signals))
	}

	// Check that signals contain expected descriptions
	expectedDescriptions := map[string]bool{
		"Go project (go.mod)":               false,
		"Node.js project (package.json)":    false,
		"Python project (requirements.txt)": false,
		"Rust project (Cargo.toml)":         false,
	}

	for _, signal := range signals {
		if _, exists := expectedDescriptions[signal]; exists {
			expectedDescriptions[signal] = true
		}
	}

	for desc, found := range expectedDescriptions {
		if !found {
			t.Errorf("Expected signal not found: %s", desc)
		}
	}
}

func TestDetectBuilder_BuilderChain(t *testing.T) {
	tests := []struct {
		name             string
		files            []string
		expectedBuilder  string
		expectedBuilders []string
	}{
		{
			name:             "Dockerfile present - chain starts with csdocker",
			files:            []string{"Dockerfile"},
			expectedBuilder:  "csdocker",
			expectedBuilders: []string{"csdocker", "railpack", "nixpacks"},
		},
		{
			name:             "dockerfile lowercase - chain starts with csdocker",
			files:            []string{"dockerfile"},
			expectedBuilder:  "csdocker",
			expectedBuilders: []string{"csdocker", "railpack", "nixpacks"},
		},
		{
			name:             "Dockerfile.prod - chain starts with csdocker",
			files:            []string{"Dockerfile.prod"},
			expectedBuilder:  "csdocker",
			expectedBuilders: []string{"csdocker", "railpack", "nixpacks"},
		},
		{
			name:             "No Dockerfile - chain starts with railpack",
			files:            []string{"package.json"},
			expectedBuilder:  "railpack",
			expectedBuilders: []string{"railpack", "nixpacks"},
		},
		{
			name:             "Empty directory - chain starts with railpack",
			files:            []string{},
			expectedBuilder:  "railpack",
			expectedBuilders: []string{"railpack", "nixpacks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "detect-chain-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Test DetectBuilder
			result := DetectBuilder(tmpDir)

			// Verify primary builder (backward compat)
			if result.Builder != tt.expectedBuilder {
				t.Errorf("Expected builder %s, got %s", tt.expectedBuilder, result.Builder)
			}

			// Verify builder chain
			if len(result.Builders) != len(tt.expectedBuilders) {
				t.Errorf("Expected %d builders, got %d: %v", len(tt.expectedBuilders), len(result.Builders), result.Builders)
			}

			for i, expected := range tt.expectedBuilders {
				if i >= len(result.Builders) {
					t.Errorf("Missing builder at index %d: expected %s", i, expected)
					continue
				}
				if result.Builders[i] != expected {
					t.Errorf("Builder at index %d: expected %s, got %s", i, expected, result.Builders[i])
				}
			}

			// Verify backward compat: Builder should equal Builders[0]
			if len(result.Builders) > 0 && result.Builder != result.Builders[0] {
				t.Errorf("Backward compat broken: Builder=%s but Builders[0]=%s", result.Builder, result.Builders[0])
			}
		})
	}
}

func TestGetBuilderChain(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		userBuilder   string
		expectedChain []string
	}{
		{
			name:          "No user override, Dockerfile present",
			files:         []string{"Dockerfile"},
			userBuilder:   "",
			expectedChain: []string{"csdocker", "railpack", "nixpacks"},
		},
		{
			name:          "No user override, no Dockerfile",
			files:         []string{"package.json"},
			userBuilder:   "",
			expectedChain: []string{"railpack", "nixpacks"},
		},
		{
			name:          "User specifies railpack, Dockerfile present",
			files:         []string{"Dockerfile"},
			userBuilder:   "railpack",
			expectedChain: []string{"railpack", "csdocker", "nixpacks"},
		},
		{
			name:          "User specifies nixpacks, Dockerfile present",
			files:         []string{"Dockerfile"},
			userBuilder:   "nixpacks",
			expectedChain: []string{"nixpacks", "csdocker", "railpack"},
		},
		{
			name:          "User specifies nixpacks, no Dockerfile",
			files:         []string{"package.json"},
			userBuilder:   "nixpacks",
			expectedChain: []string{"nixpacks", "railpack"},
		},
		{
			name:          "User specifies csdocker (already first), Dockerfile present",
			files:         []string{"Dockerfile"},
			userBuilder:   "csdocker",
			expectedChain: []string{"csdocker", "railpack", "nixpacks"},
		},
		{
			name:          "User specifies railpack (already first), no Dockerfile",
			files:         []string{"go.mod"},
			userBuilder:   "railpack",
			expectedChain: []string{"railpack", "nixpacks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "builder-chain-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
			}

			// Test GetBuilderChain
			chain := GetBuilderChain(tmpDir, tt.userBuilder)

			// Verify chain length
			if len(chain) != len(tt.expectedChain) {
				t.Errorf("Expected %d builders in chain, got %d: %v", len(tt.expectedChain), len(chain), chain)
			}

			// Verify chain order
			for i, expected := range tt.expectedChain {
				if i >= len(chain) {
					t.Errorf("Missing builder at index %d: expected %s", i, expected)
					continue
				}
				if chain[i] != expected {
					t.Errorf("Builder at index %d: expected %s, got %s", i, expected, chain[i])
				}
			}

			// Verify no duplicates in chain
			seen := make(map[string]bool)
			for _, builder := range chain {
				if seen[builder] {
					t.Errorf("Duplicate builder in chain: %s", builder)
				}
				seen[builder] = true
			}

			// Verify user builder is first if specified
			if tt.userBuilder != "" && len(chain) > 0 && chain[0] != tt.userBuilder {
				t.Errorf("User builder should be first: expected %s, got %s", tt.userBuilder, chain[0])
			}
		})
	}
}
