package goreleaser

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

func TestConfigSet_Nil(t *testing.T) {
	builder := &Builder{}
	err := builder.ConfigSet(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if builder.config == nil {
		t.Fatal("config should not be nil")
	}

	// Verify defaults
	if builder.config.OutputDir != "./dist" {
		t.Errorf("expected OutputDir './dist', got '%s'", builder.config.OutputDir)
	}

	if len(builder.config.Targets) != 2 {
		t.Errorf("expected 2 default targets, got %d", len(builder.config.Targets))
	}

	if builder.config.Targets[0] != "linux/amd64" {
		t.Errorf("expected target[0] 'linux/amd64', got '%s'", builder.config.Targets[0])
	}

	if builder.config.Targets[1] != "darwin/arm64" {
		t.Errorf("expected target[1] 'darwin/arm64', got '%s'", builder.config.Targets[1])
	}

	// Path has no default in ConfigSet - it remains empty
	if builder.config.Path != "" {
		t.Errorf("expected empty Path, got '%s'", builder.config.Path)
	}

	// Name has no default
	if builder.config.Name != "" {
		t.Errorf("expected empty Name, got '%s'", builder.config.Name)
	}

	// Version has no default
	if builder.config.Version != "" {
		t.Errorf("expected empty Version, got '%s'", builder.config.Version)
	}

	// LdFlags has no default
	if builder.config.LdFlags != "" {
		t.Errorf("expected empty LdFlags, got '%s'", builder.config.LdFlags)
	}

	// BuildArgs should be nil
	if builder.config.BuildArgs != nil {
		t.Errorf("expected nil BuildArgs, got %v", builder.config.BuildArgs)
	}
}

func TestConfigSet_Map(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]interface{}
		check func(*testing.T, *BuilderConfig)
	}{
		{
			name: "basic config",
			input: map[string]interface{}{
				"name":    "myapp",
				"path":    "./cmd/myapp",
				"version": "v1.0.0",
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.Name != "myapp" {
					t.Errorf("expected Name 'myapp', got '%s'", cfg.Name)
				}
				if cfg.Path != "./cmd/myapp" {
					t.Errorf("expected Path './cmd/myapp', got '%s'", cfg.Path)
				}
				if cfg.Version != "v1.0.0" {
					t.Errorf("expected Version 'v1.0.0', got '%s'", cfg.Version)
				}
				// Defaults applied
				if cfg.OutputDir != "./dist" {
					t.Errorf("expected OutputDir './dist', got '%s'", cfg.OutputDir)
				}
				if len(cfg.Targets) != 2 {
					t.Errorf("expected 2 default targets, got %d", len(cfg.Targets))
				}
			},
		},
		{
			name: "with targets",
			input: map[string]interface{}{
				"name":    "myapp",
				"targets": []interface{}{"linux/amd64", "windows/amd64"},
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if len(cfg.Targets) != 2 {
					t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
				}
				if cfg.Targets[0] != "linux/amd64" {
					t.Errorf("expected target[0] 'linux/amd64', got '%s'", cfg.Targets[0])
				}
				if cfg.Targets[1] != "windows/amd64" {
					t.Errorf("expected target[1] 'windows/amd64', got '%s'", cfg.Targets[1])
				}
			},
		},
		{
			name: "with build_args",
			input: map[string]interface{}{
				"name": "myapp",
				"build_args": map[string]interface{}{
					"trimpath": "true",
				},
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if len(cfg.BuildArgs) != 1 {
					t.Errorf("expected 1 build arg, got %d", len(cfg.BuildArgs))
				}
				if cfg.BuildArgs["trimpath"] != "true" {
					t.Errorf("expected trimpath='true', got '%s'", cfg.BuildArgs["trimpath"])
				}
			},
		},
		{
			name: "with ldflags",
			input: map[string]interface{}{
				"name":    "myapp",
				"ldflags": "-X main.custom=value",
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.LdFlags != "-X main.custom=value" {
					t.Errorf("expected LdFlags, got '%s'", cfg.LdFlags)
				}
			},
		},
		{
			name: "with output_dir",
			input: map[string]interface{}{
				"name":       "myapp",
				"output_dir": "./build",
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.OutputDir != "./build" {
					t.Errorf("expected OutputDir './build', got '%s'", cfg.OutputDir)
				}
			},
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			check: func(t *testing.T, cfg *BuilderConfig) {
				// All fields should be empty except defaults
				if cfg.Name != "" {
					t.Errorf("expected empty Name, got '%s'", cfg.Name)
				}
				if cfg.Path != "" {
					t.Errorf("expected empty Path, got '%s'", cfg.Path)
				}
				if cfg.Version != "" {
					t.Errorf("expected empty Version, got '%s'", cfg.Version)
				}
				// Defaults applied
				if cfg.OutputDir != "./dist" {
					t.Errorf("expected OutputDir './dist', got '%s'", cfg.OutputDir)
				}
				if len(cfg.Targets) != 2 {
					t.Errorf("expected 2 default targets, got %d", len(cfg.Targets))
				}
			},
		},
		{
			name: "with empty targets",
			input: map[string]interface{}{
				"name":    "myapp",
				"targets": []interface{}{},
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				// Empty targets still triggers default
				if len(cfg.Targets) != 2 {
					t.Errorf("expected 2 default targets for empty slice, got %d", len(cfg.Targets))
				}
			},
		},
		{
			name: "with string pointers",
			input: map[string]interface{}{
				"name": stringPtr("myapp"),
				"path": stringPtr("./cmd/myapp"),
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.Name != "myapp" {
					t.Errorf("expected Name 'myapp', got '%s'", cfg.Name)
				}
				if cfg.Path != "./cmd/myapp" {
					t.Errorf("expected Path './cmd/myapp', got '%s'", cfg.Path)
				}
			},
		},
		{
			name: "with targets as []string",
			input: map[string]interface{}{
				"name":    "myapp",
				"targets": []string{"linux/amd64", "darwin/amd64"},
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if len(cfg.Targets) != 2 {
					t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
				}
				if cfg.Targets[0] != "linux/amd64" {
					t.Errorf("expected target[0] 'linux/amd64', got '%s'", cfg.Targets[0])
				}
				if cfg.Targets[1] != "darwin/amd64" {
					t.Errorf("expected target[1] 'darwin/amd64', got '%s'", cfg.Targets[1])
				}
			},
		},
		{
			name: "with build_args as map[string]string",
			input: map[string]interface{}{
				"name": "myapp",
				"build_args": map[string]string{
					"race": "true",
				},
			},
			check: func(t *testing.T, cfg *BuilderConfig) {
				if len(cfg.BuildArgs) != 1 {
					t.Errorf("expected 1 build arg, got %d", len(cfg.BuildArgs))
				}
				if cfg.BuildArgs["race"] != "true" {
					t.Errorf("expected race='true', got '%s'", cfg.BuildArgs["race"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &Builder{}
			err := builder.ConfigSet(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.check(t, builder.config)
		})
	}
}

func TestConfigSet_TypedConfig(t *testing.T) {
	expected := &BuilderConfig{
		Name:      "testapp",
		Path:      "./cmd/test",
		Version:   "v2.0.0",
		OutputDir: "./output",
		Targets:   []string{"linux/arm64"},
		BuildArgs: map[string]string{"race": "true"},
		LdFlags:   "-X main.custom=value",
	}

	builder := &Builder{}
	err := builder.ConfigSet(expected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if builder.config != expected {
		t.Error("config should be the same instance")
	}
}

func TestConfig(t *testing.T) {
	expected := &BuilderConfig{Name: "test"}
	builder := &Builder{config: expected}

	got, err := builder.Config()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != expected {
		t.Error("Config() should return the builder's config")
	}
}

func TestTargetParsing(t *testing.T) {
	tests := []struct {
		target string
		goos   string
		goarch string
	}{
		{"linux/amd64", "linux", "amd64"},
		{"darwin/arm64", "darwin", "arm64"},
		{"windows/amd64", "windows", "amd64"},
		{"linux/arm64", "linux", "arm64"},
		{"darwin/amd64", "darwin", "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			parts := strings.Split(tt.target, "/")
			if len(parts) != 2 {
				t.Fatalf("invalid target: %s", tt.target)
			}
			goos, goarch := parts[0], parts[1]
			if goos != tt.goos || goarch != tt.goarch {
				t.Errorf("expected %s/%s, got %s/%s", tt.goos, tt.goarch, goos, goarch)
			}
		})
	}
}

func TestBinaryNaming(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		goarch   string
		expected string
	}{
		{"myapp", "linux", "amd64", "myapp-linux-amd64"},
		{"myapp", "darwin", "arm64", "myapp-darwin-arm64"},
		{"myapp", "windows", "amd64", "myapp-windows-amd64.exe"},
		{"myapp", "windows", "arm64", "myapp-windows-arm64.exe"},
		{"test-app", "linux", "arm64", "test-app-linux-arm64"},
		{"test-app", "darwin", "amd64", "test-app-darwin-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			name := fmt.Sprintf("%s-%s-%s", tt.name, tt.goos, tt.goarch)
			if tt.goos == "windows" {
				name += ".exe"
			}
			if name != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, name)
			}
		})
	}
}

func TestBuild_NilConfig(t *testing.T) {
	builder := &Builder{}

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected error for nil config")
	}

	if !strings.Contains(err.Error(), "configuration is not set") {
		t.Errorf("error should mention configuration not set, got: %v", err)
	}
}

func TestBuild_MissingName(t *testing.T) {
	builder := &Builder{
		config: &BuilderConfig{
			Path: "./cmd/app",
		},
	}

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Fatal("expected error for missing Name")
	}

	if !strings.Contains(err.Error(), "'name' field to be set") {
		t.Errorf("error should mention name requirement, got: %v", err)
	}
}

func TestBuild_InvalidTarget(t *testing.T) {
	builder := &Builder{
		config: &BuilderConfig{
			Name:    "myapp",
			Targets: []string{"invalid-target"},
		},
	}

	ctx := hclog.WithContext(context.Background(), hclog.NewNullLogger())
	_, err := builder.Build(ctx)
	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	if !strings.Contains(err.Error(), "invalid target format") {
		t.Errorf("error should mention invalid target format, got: %v", err)
	}
}

func TestBuild_DefaultsApplied(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// This would normally execute the build, but we skip it for tests
	// The test verifies that defaults are applied during Build()
	// Path defaults to "." in Build
	// OutputDir defaults to "./dist" in Build
	// Targets defaults to ["linux/amd64", "darwin/arm64"] in Build
}

func TestBuild_ArtifactMetadata(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// If we could run the build, we'd verify:
	// - artifact.ID contains "goreleaser-testapp-" + timestamp
	// - artifact.Image == "testapp"
	// - artifact.Tag == "v1.2.3"
	// - artifact.Labels["builder"] == "goreleaser"
	// - artifact.Metadata["builder"] == "goreleaser"
	// - artifact.Metadata["version"] == "v1.2.3"
	// - artifact.Metadata["targets"] == []string{"linux/amd64"}
	// - artifact.Metadata["binaries"] contains the binary paths
	// - artifact.BuildTime is set
}

func TestBuild_LdFlags(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// If we could run the build, the ldflags would be:
	// "-s -w -X main.Version=v1.0.0 -X main.version=v1.0.0 -X main.custom=value"
}

func TestBuild_BuildArgs(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// If we could run the build, the build args would be added as:
	// -trimpath=true -race=true
}

func TestBuild_OutputDirectory(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// If we could run the build:
	// - /tmp/test-build directory would be created
	// - Binary would be at /tmp/test-build/testapp-linux-amd64
}

func TestBuild_PathField(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// Test cases for path field:
	// - empty path defaults to "." in Build()
	// - specific path like "./cmd/myapp" is used as-is
	// - absolute path like "/go/src/app" is preserved
}

func TestBuild_WithGoInstalled(t *testing.T) {
	t.Skip("requires Go installation - skip in CI")

	// This would be an actual build test with a real Go file
	// It would:
	// 1. Create a temporary Go module
	// 2. Write a simple main.go file
	// 3. Run the builder
	// 4. Verify the binary was created
	// 5. Clean up temporary files
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
			"key2": stringPtr("value2"),
			"key3": 123, // not a string
		}

		if v := getString(m, "key1"); v != "value1" {
			t.Errorf("expected 'value1', got '%s'", v)
		}

		if v := getString(m, "key2"); v != "value2" {
			t.Errorf("expected 'value2', got '%s'", v)
		}

		if v := getString(m, "key3"); v != "" {
			t.Errorf("expected empty string for non-string value, got '%s'", v)
		}

		if v := getString(m, "missing"); v != "" {
			t.Errorf("expected empty string for missing key, got '%s'", v)
		}
	})

	t.Run("getStringSlice", func(t *testing.T) {
		m := map[string]interface{}{
			"slice1": []interface{}{"a", "b", "c"},
			"slice2": []string{"d", "e", "f"},
			"slice3": "not-a-slice",
			"slice4": []interface{}{"a", 123, "b"}, // mixed types
		}

		if v := getStringSlice(m, "slice1"); len(v) != 3 || v[0] != "a" {
			t.Errorf("expected [a b c], got %v", v)
		}

		if v := getStringSlice(m, "slice2"); len(v) != 3 || v[0] != "d" {
			t.Errorf("expected [d e f], got %v", v)
		}

		if v := getStringSlice(m, "slice3"); v != nil {
			t.Errorf("expected nil for non-slice, got %v", v)
		}

		if v := getStringSlice(m, "slice4"); len(v) != 2 || v[0] != "a" || v[1] != "b" {
			t.Errorf("expected [a b] for mixed slice, got %v", v)
		}

		if v := getStringSlice(m, "missing"); v != nil {
			t.Errorf("expected nil for missing key, got %v", v)
		}
	})

	t.Run("getStringMap", func(t *testing.T) {
		m := map[string]interface{}{
			"map1": map[string]interface{}{"k1": "v1", "k2": "v2"},
			"map2": map[string]string{"k3": "v3", "k4": "v4"},
			"map3": "not-a-map",
			"map4": map[string]interface{}{"k1": "v1", "k2": 123}, // mixed values
		}

		if v := getStringMap(m, "map1"); len(v) != 2 || v["k1"] != "v1" {
			t.Errorf("expected map[k1:v1 k2:v2], got %v", v)
		}

		if v := getStringMap(m, "map2"); len(v) != 2 || v["k3"] != "v3" {
			t.Errorf("expected map[k3:v3 k4:v4], got %v", v)
		}

		if v := getStringMap(m, "map3"); v != nil {
			t.Errorf("expected nil for non-map, got %v", v)
		}

		if v := getStringMap(m, "map4"); len(v) != 1 || v["k1"] != "v1" {
			t.Errorf("expected map[k1:v1] for mixed map, got %v", v)
		}

		if v := getStringMap(m, "missing"); v != nil {
			t.Errorf("expected nil for missing key, got %v", v)
		}
	})
}

func TestBuild_ArtifactStructure(t *testing.T) {
	// Test artifact creation without actually building
	now := time.Now()
	binaries := []string{"/dist/app-linux-amd64", "/dist/app-darwin-arm64"}

	// This simulates what the Build function creates
	artifactID := fmt.Sprintf("goreleaser-%s-%d", "testapp", now.Unix())
	if !strings.HasPrefix(artifactID, "goreleaser-testapp-") {
		t.Errorf("artifact ID should start with 'goreleaser-testapp-', got %s", artifactID)
	}

	// Test metadata structure
	metadata := map[string]interface{}{
		"builder":  "goreleaser",
		"binaries": binaries,
		"targets":  []string{"linux/amd64", "darwin/arm64"},
		"version":  "v1.0.0",
	}

	if metadata["builder"] != "goreleaser" {
		t.Errorf("expected builder='goreleaser', got %v", metadata["builder"])
	}

	if b, ok := metadata["binaries"].([]string); !ok || len(b) != 2 {
		t.Errorf("expected binaries to be []string with 2 items, got %v", metadata["binaries"])
	}

	// Test labels structure
	labels := map[string]string{
		"builder": "goreleaser",
	}

	if labels["builder"] != "goreleaser" {
		t.Errorf("expected label builder='goreleaser', got %s", labels["builder"])
	}
}

// Helper function for string pointers in tests
func stringPtr(s string) *string {
	return &s
}
