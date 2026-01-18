package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
)

// TestZeroConfigIntegration tests the complete zero-config workflow
func TestZeroConfigIntegration(t *testing.T) {
	tests := []struct {
		name            string
		setupDir        func(t *testing.T, dir string)
		setupGit        func(t *testing.T, dir string) // optional git setup
		wantBuilder     string
		wantProjectName string // empty = derived from dir
		wantSignals     bool   // expect project signals
	}{
		{
			name: "Node.js project without Dockerfile",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"my-node-app","version":"1.0.0"}`)
				writeFile(t, dir, "index.js", `console.log("hello")`)
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Go project without Dockerfile",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "go.mod", "module example.com/myapp\n\ngo 1.21")
				writeFile(t, dir, "main.go", "package main\n\nfunc main() {}")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Python project without Dockerfile",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "requirements.txt", "flask==2.0.0")
				writeFile(t, dir, "app.py", "print('hello')")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Project with Dockerfile uses csdocker",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"dockerized-app"}`)
				writeFile(t, dir, "Dockerfile", "FROM node:20-alpine\nCMD [\"node\"]")
			},
			wantBuilder: "csdocker",
			wantSignals: false, // Dockerfile overrides other signals
		},
		{
			name: "Empty directory uses railpack",
			setupDir: func(t *testing.T, dir string) {
				// Empty directory
			},
			wantBuilder: "railpack",
			wantSignals: false,
		},
		{
			name: "Rust project",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "Cargo.toml", "[package]\nname = \"myapp\"")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Git repo with HTTPS remote",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"test"}`)
			},
			setupGit: func(t *testing.T, dir string) {
				runGit(t, dir, "init")
				runGit(t, dir, "remote", "add", "origin", "https://github.com/testuser/git-detected-name.git")
			},
			wantBuilder:     "railpack",
			wantProjectName: "git-detected-name",
		},
		{
			name: "Git repo with SSH remote",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "go.mod", "module test")
			},
			setupGit: func(t *testing.T, dir string) {
				runGit(t, dir, "init")
				runGit(t, dir, "remote", "add", "origin", "git@github.com:testuser/ssh-detected-name.git")
			},
			wantBuilder:     "railpack",
			wantProjectName: "ssh-detected-name",
		},
		{
			name: "Ruby project with Gemfile",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'\ngem 'rails'")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Java Maven project",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "pom.xml", `<?xml version="1.0"?><project></project>`)
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "PHP project with composer",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "composer.json", `{"name":"my/app"}`)
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Elixir project",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "mix.exs", "defmodule MyApp do end")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Python project with pyproject.toml",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "pyproject.toml", "[project]\nname = \"myapp\"")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
		{
			name: "Go workspace",
			setupDir: func(t *testing.T, dir string) {
				writeFile(t, dir, "go.work", "go 1.21\nuse ./app")
			},
			wantBuilder: "railpack",
			wantSignals: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated temp directory
			dir := t.TempDir()

			// Setup test files
			tt.setupDir(t, dir)

			// Setup git if specified
			if tt.setupGit != nil {
				tt.setupGit(t, dir)
			}

			// Change working directory for project name detection
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Chdir(oldWd); err != nil {
					t.Logf("Warning: failed to restore working directory: %v", err)
				}
			}()

			// Test builder detection
			detection := detect.DetectBuilder(dir)
			if detection.Builder != tt.wantBuilder {
				t.Errorf("DetectBuilder() = %q, want %q", detection.Builder, tt.wantBuilder)
			}

			// Test config generation
			cfg, err := GenerateDefaultConfig(dir)
			if err != nil {
				t.Fatalf("GenerateDefaultConfig() error = %v", err)
			}

			// Verify builder in config
			if cfg.Apps[0].Build.Use != tt.wantBuilder {
				t.Errorf("Config builder = %q, want %q", cfg.Apps[0].Build.Use, tt.wantBuilder)
			}

			// Verify project name if specified
			if tt.wantProjectName != "" {
				if cfg.Project != tt.wantProjectName {
					t.Errorf("Config project = %q, want %q", cfg.Project, tt.wantProjectName)
				}
			}

			// Verify signals detection
			if tt.wantSignals && len(detection.Signals) == 0 {
				t.Error("Expected project signals to be detected")
			}
		})
	}
}

// TestZeroConfigEdgeCases tests edge cases and error handling
func TestZeroConfigEdgeCases(t *testing.T) {
	t.Run("special characters in directory name", func(t *testing.T) {
		dir := t.TempDir()

		// Create subdirectory with special characters
		specialDir := filepath.Join(dir, "My App (v2.0)")
		if err := os.MkdirAll(specialDir, 0755); err != nil {
			t.Fatal(err)
		}

		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(specialDir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				t.Logf("Warning: failed to restore working directory: %v", err)
			}
		}()

		cfg, err := GenerateDefaultConfig(specialDir)
		if err != nil {
			t.Fatalf("GenerateDefaultConfig() error = %v", err)
		}

		// Project name should be sanitized
		if cfg.Project == "" {
			t.Error("Project name should not be empty")
		}
		// Should not contain special characters
		for _, c := range cfg.Project {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				t.Errorf("Project name contains invalid character: %c", c)
			}
		}
	})

	t.Run("multiple Dockerfile variants", func(t *testing.T) {
		variants := []string{"Dockerfile", "dockerfile", "Dockerfile.prod", "Dockerfile.dev"}

		for _, variant := range variants {
			t.Run(variant, func(t *testing.T) {
				dir := t.TempDir()
				writeFile(t, dir, variant, "FROM alpine")

				detection := detect.DetectBuilder(dir)
				if detection.Builder != "csdocker" {
					t.Errorf("%s should trigger csdocker, got %s", variant, detection.Builder)
				}
			})
		}
	})

	t.Run("config has all required fields", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "package.json", `{"name":"test"}`)

		cfg, err := GenerateDefaultConfig(dir)
		if err != nil {
			t.Fatal(err)
		}

		// Verify structure
		if cfg.Project == "" {
			t.Error("Project should not be empty")
		}
		if len(cfg.Apps) != 1 {
			t.Fatalf("Expected 1 app, got %d", len(cfg.Apps))
		}

		app := cfg.Apps[0]
		if app.Name == "" {
			t.Error("App.Name should not be empty")
		}
		if app.Build == nil {
			t.Error("App.Build should not be nil")
		}
		if app.Build.Use == "" {
			t.Error("App.Build.Use should not be empty")
		}
		if app.Build.Config == nil {
			t.Error("App.Build.Config should not be nil")
		}
		if app.Deploy == nil {
			t.Error("App.Deploy should not be nil")
		}
		if app.Deploy.Use == "" {
			t.Error("App.Deploy.Use should not be empty")
		}
	})

	t.Run("Dockerfile.production variant", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Dockerfile.production", "FROM golang:1.21-alpine")

		detection := detect.DetectBuilder(dir)
		if detection.Builder != "csdocker" {
			t.Errorf("Dockerfile.production should trigger csdocker, got %s", detection.Builder)
		}
		if !detection.HasDocker {
			t.Error("HasDocker should be true")
		}
	})

	t.Run("Dockerfile.development variant", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Dockerfile.development", "FROM python:3.11")

		detection := detect.DetectBuilder(dir)
		if detection.Builder != "csdocker" {
			t.Errorf("Dockerfile.development should trigger csdocker, got %s", detection.Builder)
		}
	})

	t.Run("multiple project types detected", func(t *testing.T) {
		dir := t.TempDir()
		// Create a multi-language project
		writeFile(t, dir, "package.json", `{"name":"frontend"}`)
		writeFile(t, dir, "requirements.txt", "django==4.0")
		writeFile(t, dir, "go.mod", "module backend")

		detection := detect.DetectBuilder(dir)
		if detection.Builder != "railpack" {
			t.Errorf("Multi-language project should use railpack, got %s", detection.Builder)
		}

		// Should detect multiple signals
		if len(detection.Signals) < 3 {
			t.Errorf("Expected at least 3 signals for multi-language project, got %d: %v",
				len(detection.Signals), detection.Signals)
		}
	})

	t.Run("project name derived from directory", func(t *testing.T) {
		dir := t.TempDir()

		// Create a subdirectory with a known name
		projectDir := filepath.Join(dir, "my-test-project")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatal(err)
		}

		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(projectDir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				t.Logf("Warning: failed to restore working directory: %v", err)
			}
		}()

		cfg, err := GenerateDefaultConfig(projectDir)
		if err != nil {
			t.Fatalf("GenerateDefaultConfig() error = %v", err)
		}

		if cfg.Project != "my-test-project" {
			t.Errorf("Expected project name 'my-test-project', got %q", cfg.Project)
		}
	})

	t.Run("app name matches project name", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := GenerateDefaultConfig(dir)
		if err != nil {
			t.Fatalf("GenerateDefaultConfig() error = %v", err)
		}

		if cfg.Apps[0].Name != cfg.Project {
			t.Errorf("App name (%q) should match project name (%q)", cfg.Apps[0].Name, cfg.Project)
		}
	})

	t.Run("build config contains required fields", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module test")

		cfg, err := GenerateDefaultConfig(dir)
		if err != nil {
			t.Fatal(err)
		}

		buildConfig := cfg.Apps[0].Build.Config
		if _, ok := buildConfig["name"]; !ok {
			t.Error("Build config should have 'name' field")
		}
		if _, ok := buildConfig["tag"]; !ok {
			t.Error("Build config should have 'tag' field")
		}
		if _, ok := buildConfig["context"]; !ok {
			t.Error("Build config should have 'context' field")
		}
	})

	t.Run("deploy config is valid", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := GenerateDefaultConfig(dir)
		if err != nil {
			t.Fatalf("GenerateDefaultConfig() error = %v", err)
		}

		if cfg.Apps[0].Deploy.Use != "nomad-pack" {
			t.Errorf("Deploy.Use should be 'nomad-pack', got %q", cfg.Apps[0].Deploy.Use)
		}

		deployConfig := cfg.Apps[0].Deploy.Config
		if pack, ok := deployConfig["pack"]; !ok || pack != "cloudstation" {
			t.Errorf("Deploy config pack should be 'cloudstation', got %v", deployConfig["pack"])
		}
	})
}

// TestZeroConfigDetectionReason tests that detection reasons are appropriate
func TestZeroConfigDetectionReason(t *testing.T) {
	t.Run("Dockerfile reason includes Vault integration", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Dockerfile", "FROM alpine")

		detection := detect.DetectBuilder(dir)
		if detection.Reason == "" {
			t.Error("Detection reason should not be empty")
		}
		// The reason should mention csdocker or Vault
		if detection.Builder != "csdocker" {
			t.Errorf("Expected csdocker builder for Dockerfile")
		}
	})

	t.Run("no Dockerfile reason mentions zero-config", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "package.json", `{}`)

		detection := detect.DetectBuilder(dir)
		if detection.Reason == "" {
			t.Error("Detection reason should not be empty")
		}
	})
}

// TestZeroConfigConcurrentDirectories tests behavior with concurrent directory changes
func TestZeroConfigConcurrentDirectories(t *testing.T) {
	t.Run("different directories yield different results", func(t *testing.T) {
		// Create two different project types
		nodeDir := t.TempDir()
		writeFile(t, nodeDir, "package.json", `{"name":"node-app"}`)

		goDir := t.TempDir()
		writeFile(t, goDir, "go.mod", "module goapp")

		dockerDir := t.TempDir()
		writeFile(t, dockerDir, "Dockerfile", "FROM alpine")

		// Test all three independently
		nodeDetection := detect.DetectBuilder(nodeDir)
		goDetection := detect.DetectBuilder(goDir)
		dockerDetection := detect.DetectBuilder(dockerDir)

		if nodeDetection.Builder != "railpack" {
			t.Errorf("Node project should use railpack, got %s", nodeDetection.Builder)
		}
		if goDetection.Builder != "railpack" {
			t.Errorf("Go project should use railpack, got %s", goDetection.Builder)
		}
		if dockerDetection.Builder != "csdocker" {
			t.Errorf("Docker project should use csdocker, got %s", dockerDetection.Builder)
		}
	})
}

// TestZeroConfigHelperFunctions tests convenience functions
func TestZeroConfigHelperFunctions(t *testing.T) {
	t.Run("HasDockerfile returns correct value", func(t *testing.T) {
		withDocker := t.TempDir()
		writeFile(t, withDocker, "Dockerfile", "FROM alpine")

		withoutDocker := t.TempDir()
		writeFile(t, withoutDocker, "package.json", "{}")

		if !detect.HasDockerfile(withDocker) {
			t.Error("HasDockerfile should return true for directory with Dockerfile")
		}
		if detect.HasDockerfile(withoutDocker) {
			t.Error("HasDockerfile should return false for directory without Dockerfile")
		}
	})

	t.Run("GetDefaultBuilder returns correct builder", func(t *testing.T) {
		withDocker := t.TempDir()
		writeFile(t, withDocker, "Dockerfile", "FROM alpine")

		withoutDocker := t.TempDir()
		writeFile(t, withoutDocker, "go.mod", "module test")

		if detect.GetDefaultBuilder(withDocker) != "csdocker" {
			t.Errorf("GetDefaultBuilder should return csdocker for Dockerfile project")
		}
		if detect.GetDefaultBuilder(withoutDocker) != "railpack" {
			t.Errorf("GetDefaultBuilder should return railpack for non-Docker project")
		}
	})
}

// TestZeroConfigProcfileHandling tests Procfile detection
func TestZeroConfigProcfileHandling(t *testing.T) {
	t.Run("Procfile detected as signal", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Procfile", "web: node server.js")

		detection := detect.DetectBuilder(dir)
		if detection.Builder != "railpack" {
			t.Errorf("Procfile project should use railpack, got %s", detection.Builder)
		}

		// Check that Procfile is in signals
		found := false
		for _, signal := range detection.Signals {
			if signal != "" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected Procfile to be detected as a signal")
		}
	})
}

// Helper functions

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
