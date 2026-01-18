package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
)

// TestBuildCommand_BuilderChainDetection verifies that the builder chain
// is correctly determined based on project structure and user input.
func TestBuildCommand_BuilderChainDetection(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		userBuilder   string
		expectedFirst string
		expectedLen   int
	}{
		{
			name:          "Dockerfile present - csdocker first",
			files:         []string{"Dockerfile"},
			userBuilder:   "",
			expectedFirst: "csdocker",
			expectedLen:   3,
		},
		{
			name:          "No Dockerfile - railpack first",
			files:         []string{"package.json"},
			userBuilder:   "",
			expectedFirst: "railpack",
			expectedLen:   2,
		},
		{
			name:          "User specifies nixpacks with Dockerfile",
			files:         []string{"Dockerfile"},
			userBuilder:   "nixpacks",
			expectedFirst: "nixpacks",
			expectedLen:   3,
		},
		{
			name:          "User specifies railpack without Dockerfile",
			files:         []string{"go.mod"},
			userBuilder:   "railpack",
			expectedFirst: "railpack",
			expectedLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "cli-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create %s: %v", file, err)
				}
			}

			// Get builder chain using the same logic as commands.go
			var chain []string
			if tt.userBuilder == "" {
				detection := detect.DetectBuilder(tmpDir)
				chain = detection.Builders
			} else {
				chain = detect.GetBuilderChain(tmpDir, tt.userBuilder)
			}

			// Verify chain
			if len(chain) != tt.expectedLen {
				t.Errorf("Expected chain length %d, got %d: %v", tt.expectedLen, len(chain), chain)
			}

			if len(chain) > 0 && chain[0] != tt.expectedFirst {
				t.Errorf("Expected first builder %s, got %s", tt.expectedFirst, chain[0])
			}
		})
	}
}

// TestUpCommand_DryRunOutput verifies that dry-run mode correctly
// includes builder chain information.
func TestUpCommand_DryRunOutput(t *testing.T) {
	// Create temporary directory with a project
	tmpDir, err := os.MkdirTemp("", "up-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create package.json (Node.js project)
	packageJSON := `{"name": "test-app", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Simulate what the up command does for builder detection
	detection := detect.DetectBuilder(tmpDir)

	// Verify detection result
	if detection.Builder != "railpack" {
		t.Errorf("Expected railpack for Node.js project, got %s", detection.Builder)
	}

	if len(detection.Builders) != 2 {
		t.Errorf("Expected 2 builders in chain, got %d", len(detection.Builders))
	}

	expectedChain := []string{"railpack", "nixpacks"}
	for i, expected := range expectedChain {
		if i < len(detection.Builders) && detection.Builders[i] != expected {
			t.Errorf("Builder at index %d: expected %s, got %s", i, expected, detection.Builders[i])
		}
	}
}

// TestBuilderChainConsistency verifies that the builder chain logic
// in CLI commands matches the handler behavior.
func TestBuilderChainConsistency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "consistency-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Dockerfile
	if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	// Test 1: Auto-detection should return same chain via both methods
	detection := detect.DetectBuilder(tmpDir)
	chain := detect.GetBuilderChain(tmpDir, "")

	if len(detection.Builders) != len(chain) {
		t.Errorf("Inconsistent chain lengths: DetectBuilder=%d, GetBuilderChain=%d",
			len(detection.Builders), len(chain))
	}

	for i := range detection.Builders {
		if i < len(chain) && detection.Builders[i] != chain[i] {
			t.Errorf("Inconsistent builder at %d: DetectBuilder=%s, GetBuilderChain=%s",
				i, detection.Builders[i], chain[i])
		}
	}

	// Test 2: User override should make specified builder first
	userChain := detect.GetBuilderChain(tmpDir, "nixpacks")
	if userChain[0] != "nixpacks" {
		t.Errorf("User-specified builder should be first, got %s", userChain[0])
	}

	// Verify the chain contains all expected builders (no missing)
	expectedBuilders := map[string]bool{"csdocker": false, "railpack": false, "nixpacks": false}
	for _, b := range userChain {
		expectedBuilders[b] = true
	}
	for builder, found := range expectedBuilders {
		if !found {
			t.Errorf("Builder %s missing from chain: %v", builder, userChain)
		}
	}
}
