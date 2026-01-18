package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateDefaultConfig(t *testing.T) {
	tests := []struct {
		name            string
		setupDir        func(t *testing.T, dir string)
		expectedBuilder string
		expectedProject string // empty means use directory name
	}{
		{
			name: "empty directory uses railpack",
			setupDir: func(t *testing.T, dir string) {
				// Empty directory - no setup needed
			},
			expectedBuilder: "railpack",
		},
		{
			name: "directory with Dockerfile uses csdocker",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "csdocker",
		},
		{
			name: "directory with package.json uses railpack",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "railpack",
		},
		{
			name: "directory with go.mod uses railpack",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "railpack",
		},
		{
			name: "Dockerfile takes priority over other files",
			setupDir: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
				os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM node:20"), 0644)
			},
			expectedBuilder: "csdocker",
		},
		{
			name: "lowercase dockerfile detected",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "dockerfile"), []byte("FROM alpine"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "csdocker",
		},
		{
			name: "Dockerfile.prod detected",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "Dockerfile.prod"), []byte("FROM alpine"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "csdocker",
		},
		{
			name: "Dockerfile.production detected",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "Dockerfile.production"), []byte("FROM alpine"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "csdocker",
		},
		{
			name: "directory with requirements.txt uses railpack",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0.0"), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "railpack",
		},
		{
			name: "directory with Cargo.toml uses railpack",
			setupDir: func(t *testing.T, dir string) {
				err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
				if err != nil {
					t.Fatal(err)
				}
			},
			expectedBuilder: "railpack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			dir := t.TempDir()

			// Setup test directory
			tt.setupDir(t, dir)

			// Change to test directory for project name detection
			oldWd, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(oldWd)

			// Generate config
			cfg, err := GenerateDefaultConfig(dir)
			if err != nil {
				t.Fatalf("GenerateDefaultConfig() error = %v", err)
			}

			// Verify config is not nil
			if cfg == nil {
				t.Fatal("GenerateDefaultConfig() returned nil config")
			}

			// Verify project name is set
			if cfg.Project == "" {
				t.Error("Config.Project should not be empty")
			}

			// Verify apps array has one entry
			if len(cfg.Apps) != 1 {
				t.Fatalf("Expected 1 app, got %d", len(cfg.Apps))
			}

			app := cfg.Apps[0]

			// Verify app name matches project
			if app.Name != cfg.Project {
				t.Errorf("App.Name = %q, want %q", app.Name, cfg.Project)
			}

			// Verify build block
			if app.Build == nil {
				t.Fatal("App.Build should not be nil")
			}
			if app.Build.Use != tt.expectedBuilder {
				t.Errorf("App.Build.Use = %q, want %q", app.Build.Use, tt.expectedBuilder)
			}

			// Verify build config has required fields
			if app.Build.Config == nil {
				t.Fatal("App.Build.Config should not be nil")
			}
			if _, ok := app.Build.Config["name"]; !ok {
				t.Error("App.Build.Config should have 'name' field")
			}
			if _, ok := app.Build.Config["tag"]; !ok {
				t.Error("App.Build.Config should have 'tag' field")
			}
			if _, ok := app.Build.Config["context"]; !ok {
				t.Error("App.Build.Config should have 'context' field")
			}

			// Verify deploy block
			if app.Deploy == nil {
				t.Fatal("App.Deploy should not be nil")
			}
			if app.Deploy.Use != "nomad-pack" {
				t.Errorf("App.Deploy.Use = %q, want %q", app.Deploy.Use, "nomad-pack")
			}
		})
	}
}

func TestGenerateDefaultConfigDefaults(t *testing.T) {
	dir := t.TempDir()

	cfg, err := GenerateDefaultConfig(dir)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	app := cfg.Apps[0]

	// Verify default tag is "latest"
	if tag, ok := app.Build.Config["tag"].(string); !ok || tag != "latest" {
		t.Errorf("Default tag should be 'latest', got %v", app.Build.Config["tag"])
	}

	// Verify default context is "."
	if ctx, ok := app.Build.Config["context"].(string); !ok || ctx != "." {
		t.Errorf("Default context should be '.', got %v", app.Build.Config["context"])
	}

	// Verify deploy config has pack field
	if app.Deploy.Config == nil {
		t.Fatal("App.Deploy.Config should not be nil")
	}
	if pack, ok := app.Deploy.Config["pack"].(string); !ok || pack != "cloudstation" {
		t.Errorf("Default deploy pack should be 'cloudstation', got %v", app.Deploy.Config["pack"])
	}
}

func TestGenerateDefaultConfigProjectName(t *testing.T) {
	// Test that the project name in build config matches the detected project name
	dir := t.TempDir()

	// Change to test directory for project name detection
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cfg, err := GenerateDefaultConfig(dir)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	app := cfg.Apps[0]

	// Verify build config name matches project name
	if buildName, ok := app.Build.Config["name"].(string); !ok {
		t.Error("Build config 'name' should be a string")
	} else if buildName != cfg.Project {
		t.Errorf("Build config name = %q, want %q (project name)", buildName, cfg.Project)
	}
}

func TestGenerateDefaultConfigStructure(t *testing.T) {
	dir := t.TempDir()

	cfg, err := GenerateDefaultConfig(dir)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	// Verify the overall structure
	if cfg.Project == "" {
		t.Error("Project name should be set")
	}

	if cfg.Apps == nil {
		t.Fatal("Apps should not be nil")
	}

	if len(cfg.Apps) == 0 {
		t.Fatal("Apps should have at least one entry")
	}

	app := cfg.Apps[0]

	// App name should be set
	if app.Name == "" {
		t.Error("App name should not be empty")
	}

	// Build block required
	if app.Build == nil {
		t.Fatal("Build block should be present")
	}

	// Build.Use should be either railpack or csdocker
	validBuilders := map[string]bool{"railpack": true, "csdocker": true}
	if !validBuilders[app.Build.Use] {
		t.Errorf("Build.Use should be 'railpack' or 'csdocker', got %q", app.Build.Use)
	}

	// Deploy block required
	if app.Deploy == nil {
		t.Fatal("Deploy block should be present")
	}

	// Deploy.Use should be nomad-pack
	if app.Deploy.Use != "nomad-pack" {
		t.Errorf("Deploy.Use should be 'nomad-pack', got %q", app.Deploy.Use)
	}
}

func TestGenerateDefaultConfigNoError(t *testing.T) {
	// Test that GenerateDefaultConfig doesn't error on various directory states
	tests := []struct {
		name     string
		setupDir func(t *testing.T, dir string)
	}{
		{
			name: "empty directory",
			setupDir: func(t *testing.T, dir string) {
				// No setup needed
			},
		},
		{
			name: "directory with multiple project files",
			setupDir: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0644)
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
				os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644)
			},
		},
		{
			name: "directory with subdirectories",
			setupDir: func(t *testing.T, dir string) {
				os.MkdirAll(filepath.Join(dir, "src"), 0755)
				os.MkdirAll(filepath.Join(dir, "test"), 0755)
				os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main"), 0644)
			},
		},
		{
			name: "directory with hidden files",
			setupDir: func(t *testing.T, dir string) {
				os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/"), 0644)
				os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=test"), 0644)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setupDir(t, dir)

			cfg, err := GenerateDefaultConfig(dir)
			if err != nil {
				t.Errorf("GenerateDefaultConfig() unexpected error = %v", err)
			}
			if cfg == nil {
				t.Error("GenerateDefaultConfig() returned nil config")
			}
		})
	}
}

func TestGenerateDefaultConfigDockerfileVariants(t *testing.T) {
	// Test that various Dockerfile variants are properly detected
	dockerfileVariants := []string{
		"Dockerfile",
		"dockerfile",
		"Dockerfile.prod",
		"Dockerfile.production",
		"Dockerfile.dev",
		"Dockerfile.development",
	}

	for _, variant := range dockerfileVariants {
		t.Run(variant, func(t *testing.T) {
			dir := t.TempDir()

			// Create the Dockerfile variant
			err := os.WriteFile(filepath.Join(dir, variant), []byte("FROM alpine:latest"), 0644)
			if err != nil {
				t.Fatal(err)
			}

			cfg, err := GenerateDefaultConfig(dir)
			if err != nil {
				t.Fatalf("GenerateDefaultConfig() error = %v", err)
			}

			if cfg.Apps[0].Build.Use != "csdocker" {
				t.Errorf("With %s present, expected builder 'csdocker', got %q", variant, cfg.Apps[0].Build.Use)
			}
		})
	}
}

func TestGenerateDefaultConfigBuildConfigConsistency(t *testing.T) {
	dir := t.TempDir()

	cfg, err := GenerateDefaultConfig(dir)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	app := cfg.Apps[0]
	buildConfig := app.Build.Config

	// Verify all required build config fields are present and have correct types
	requiredFields := []struct {
		key          string
		expectedType string
	}{
		{"name", "string"},
		{"tag", "string"},
		{"context", "string"},
	}

	for _, field := range requiredFields {
		val, ok := buildConfig[field.key]
		if !ok {
			t.Errorf("Build config missing required field %q", field.key)
			continue
		}

		switch field.expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				t.Errorf("Build config field %q should be string, got %T", field.key, val)
			}
		}
	}
}

func TestGenerateDefaultConfigFallbackProjectName(t *testing.T) {
	// Test when DetectProjectName would return empty (fallback to "my-app")
	dir := t.TempDir()

	// Change to a directory without git (temp dir won't have git)
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	cfg, err := GenerateDefaultConfig(dir)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	// Project name should not be empty - it should have some fallback value
	if cfg.Project == "" {
		t.Error("Project name should have a fallback value, got empty string")
	}

	// The project name should be a valid identifier (lowercase, no spaces, etc.)
	// This is already tested in detector_test.go, so we just verify non-empty
	t.Logf("Detected project name: %q", cfg.Project)
}
