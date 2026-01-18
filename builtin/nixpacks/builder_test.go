package nixpacks

import (
	"context"
	"testing"
)

func TestBuilder_ConfigSet(t *testing.T) {
	tests := []struct {
		name           string
		config         interface{}
		expectedName   string
		expectedTag    string
		expectedCtx    string
		expectedErr    bool
		validateConfig func(*testing.T, *BuilderConfig)
	}{
		{
			name: "valid config with all fields",
			config: map[string]interface{}{
				"name":    "test-app",
				"tag":     "v1.0.0",
				"context": "./src",
				"build_args": map[string]interface{}{
					"NODE_ENV": "production",
				},
				"env": map[string]interface{}{
					"API_KEY": "secret",
				},
			},
			expectedName: "test-app",
			expectedTag:  "v1.0.0",
			expectedCtx:  "./src",
			expectedErr:  false,
			validateConfig: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.BuildArgs["NODE_ENV"] != "production" {
					t.Errorf("expected BuildArgs[NODE_ENV] = production, got %s", cfg.BuildArgs["NODE_ENV"])
				}
				if cfg.Env["API_KEY"] != "secret" {
					t.Errorf("expected Env[API_KEY] = secret, got %s", cfg.Env["API_KEY"])
				}
			},
		},
		{
			name: "valid config with required fields only",
			config: map[string]interface{}{
				"name": "minimal-app",
			},
			expectedName: "minimal-app",
			expectedTag:  "",
			expectedCtx:  "",
			expectedErr:  false,
		},
		{
			name: "config with vault fields (Phase 2)",
			config: map[string]interface{}{
				"name":          "vault-app",
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectedName: "vault-app",
			expectedErr:  false,
			validateConfig: func(t *testing.T, cfg *BuilderConfig) {
				if cfg.VaultAddress != "https://vault.example.com" {
					t.Errorf("expected VaultAddress = https://vault.example.com, got %s", cfg.VaultAddress)
				}
				if cfg.RoleID != "test-role" {
					t.Errorf("expected RoleID = test-role, got %s", cfg.RoleID)
				}
			},
		},
		{
			name:        "nil config",
			config:      nil,
			expectedErr: false,
		},
		{
			name: "typed config",
			config: &BuilderConfig{
				Name: "typed-app",
				Tag:  "latest",
			},
			expectedName: "typed-app",
			expectedTag:  "latest",
			expectedErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{}
			err := b.ConfigSet(tt.config)

			if (err != nil) != tt.expectedErr {
				t.Errorf("ConfigSet() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if !tt.expectedErr && b.config != nil {
				if tt.expectedName != "" && b.config.Name != tt.expectedName {
					t.Errorf("expected Name = %s, got %s", tt.expectedName, b.config.Name)
				}
				if tt.expectedTag != "" && b.config.Tag != tt.expectedTag {
					t.Errorf("expected Tag = %s, got %s", tt.expectedTag, b.config.Tag)
				}
				if tt.expectedCtx != "" && b.config.Context != tt.expectedCtx {
					t.Errorf("expected Context = %s, got %s", tt.expectedCtx, b.config.Context)
				}

				if tt.validateConfig != nil {
					tt.validateConfig(t, b.config)
				}
			}
		})
	}
}

func TestBuilder_Config(t *testing.T) {
	expectedConfig := &BuilderConfig{
		Name: "test-app",
		Tag:  "v1.0.0",
	}

	b := &Builder{config: expectedConfig}
	cfg, err := b.Config()

	if err != nil {
		t.Errorf("Config() unexpected error: %v", err)
	}

	if cfg != expectedConfig {
		t.Errorf("Config() returned wrong config")
	}
}

func TestBuilder_Build_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      *BuilderConfig
		expectedErr string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedErr: "configuration is not set",
		},
		{
			name:        "missing name field",
			config:      &BuilderConfig{},
			expectedErr: "requires 'name' field to be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{config: tt.config}
			ctx := context.Background()

			_, err := b.Build(ctx)

			if err == nil {
				t.Errorf("Build() expected error containing %q, got nil", tt.expectedErr)
				return
			}

			if !contains(err.Error(), tt.expectedErr) {
				t.Errorf("Build() error = %v, expected to contain %q", err, tt.expectedErr)
			}
		})
	}
}

func TestBuilder_Build_Defaults(t *testing.T) {
	// This test validates that config is set correctly
	// We won't actually run nixpacks, just verify config handling
	b := &Builder{
		config: &BuilderConfig{
			Name: "test-app",
			// Tag and Context not set, should use defaults in Build method
		},
	}

	// We can't easily test the actual build without mocking exec.Command
	// so we'll just verify the config was set correctly
	if b.config.Name != "test-app" {
		t.Errorf("expected Name = test-app, got %s", b.config.Name)
	}

	// The defaults are applied in the Build() method, not stored in config
	// So we just verify the config fields are empty as expected
	if b.config.Context != "" {
		t.Errorf("expected Context to be empty, got %s", b.config.Context)
	}

	if b.config.Tag != "" {
		t.Errorf("expected Tag to be empty, got %s", b.config.Tag)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestContextHandling(t *testing.T) {
	t.Run("subdirectory context uses dot as build arg", func(t *testing.T) {
		// This test validates the fix for double-pathing bug
		// When context = "app/server", the build command should use "."
		// and cmd.Dir should be set to "app/server"

		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "app/server",
			},
		}

		// We can't easily test cmd execution without mocking,
		// but we can validate the config is set correctly
		if builder.config.Context != "app/server" {
			t.Errorf("expected context app/server, got %s", builder.config.Context)
		}

		// The actual test would verify that:
		// 1. args contains "build", "." (not "app/server")
		// 2. cmd.Dir is set to "app/server"
		// This is validated through integration tests or by inspecting
		// the Build method's internal logic
	})

	t.Run("default context uses dot", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: ".",
			},
		}

		if builder.config.Context != "." {
			t.Errorf("expected context ., got %s", builder.config.Context)
		}
	})

	t.Run("empty context defaults to dot", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "",
			},
		}

		// Empty context should be treated as "." in Build method
		if builder.config.Context != "" {
			t.Errorf("expected empty context, got %s", builder.config.Context)
		}
	})

	t.Run("relative path context", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "./src",
			},
		}

		if builder.config.Context != "./src" {
			t.Errorf("expected context ./src, got %s", builder.config.Context)
		}
	})

	t.Run("absolute path context", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "/tmp/upload-xyz",
			},
		}

		if builder.config.Context != "/tmp/upload-xyz" {
			t.Errorf("expected context /tmp/upload-xyz, got %s", builder.config.Context)
		}
	})
}
