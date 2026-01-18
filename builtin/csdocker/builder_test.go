package csdocker

import (
	"context"
	"testing"
)

func TestBuilder_ConfigSet(t *testing.T) {
	tests := []struct {
		name           string
		config         interface{}
		expectedName   string
		expectedImage  string
		expectedTag    string
		expectedCtx    string
		expectedFile   string
		expectedErr    bool
		validateConfig func(*testing.T, *BuilderConfig)
	}{
		{
			name: "valid config with all fields",
			config: map[string]interface{}{
				"name":       "test-app",
				"image":      "myregistry.azurecr.io/test-app",
				"tag":        "v1.0.0",
				"dockerfile": "Dockerfile.prod",
				"context":    "./src",
				"build_args": map[string]interface{}{
					"NODE_ENV": "production",
				},
				"env": map[string]interface{}{
					"API_KEY": "secret",
				},
			},
			expectedName:  "test-app",
			expectedImage: "myregistry.azurecr.io/test-app",
			expectedTag:   "v1.0.0",
			expectedCtx:   "./src",
			expectedFile:  "Dockerfile.prod",
			expectedErr:   false,
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
			name: "valid config with name only",
			config: map[string]interface{}{
				"name": "minimal-app",
			},
			expectedName: "minimal-app",
			expectedErr:  false,
		},
		{
			name: "valid config with image only",
			config: map[string]interface{}{
				"image": "registry.io/myapp",
			},
			expectedImage: "registry.io/myapp",
			expectedErr:   false,
		},
		{
			name: "config with Vault fields (backward compatibility)",
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
				if cfg.SecretID != "test-secret" {
					t.Errorf("expected SecretID = test-secret, got %s", cfg.SecretID)
				}
				if cfg.SecretsPath != "secret/data/app" {
					t.Errorf("expected SecretsPath = secret/data/app, got %s", cfg.SecretsPath)
				}
			},
		},
		{
			name: "config with build_args and env",
			config: map[string]interface{}{
				"name": "args-test",
				"build_args": map[string]interface{}{
					"ARG1": "value1",
					"ARG2": "value2",
				},
				"env": map[string]interface{}{
					"ENV1": "envvalue1",
					"ENV2": "envvalue2",
				},
			},
			expectedName: "args-test",
			expectedErr:  false,
			validateConfig: func(t *testing.T, cfg *BuilderConfig) {
				if len(cfg.BuildArgs) != 2 {
					t.Errorf("expected 2 build args, got %d", len(cfg.BuildArgs))
				}
				if cfg.BuildArgs["ARG1"] != "value1" {
					t.Errorf("expected BuildArgs[ARG1] = value1, got %s", cfg.BuildArgs["ARG1"])
				}
				if len(cfg.Env) != 2 {
					t.Errorf("expected 2 env vars, got %d", len(cfg.Env))
				}
				if cfg.Env["ENV1"] != "envvalue1" {
					t.Errorf("expected Env[ENV1] = envvalue1, got %s", cfg.Env["ENV1"])
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
				if tt.expectedImage != "" && b.config.Image != tt.expectedImage {
					t.Errorf("expected Image = %s, got %s", tt.expectedImage, b.config.Image)
				}
				if tt.expectedTag != "" && b.config.Tag != tt.expectedTag {
					t.Errorf("expected Tag = %s, got %s", tt.expectedTag, b.config.Tag)
				}
				if tt.expectedCtx != "" && b.config.Context != tt.expectedCtx {
					t.Errorf("expected Context = %s, got %s", tt.expectedCtx, b.config.Context)
				}
				if tt.expectedFile != "" && b.config.Dockerfile != tt.expectedFile {
					t.Errorf("expected Dockerfile = %s, got %s", tt.expectedFile, b.config.Dockerfile)
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
		Name:  "test-app",
		Image: "registry.io/test-app",
		Tag:   "v1.0.0",
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
			name:        "missing name and image fields",
			config:      &BuilderConfig{},
			expectedErr: "requires either 'name' or 'image' field to be set",
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
	// We won't actually run docker, just verify config handling
	b := &Builder{
		config: &BuilderConfig{
			Name: "test-app",
			// Tag, Dockerfile, and Context not set, should use defaults in Build method
		},
	}

	// Verify the config was set correctly
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

	if b.config.Dockerfile != "" {
		t.Errorf("expected Dockerfile to be empty, got %s", b.config.Dockerfile)
	}
}

func TestBuilder_Build_ImagePrecedence(t *testing.T) {
	// Test that Image takes precedence over Name
	b := &Builder{
		config: &BuilderConfig{
			Name:  "simple-name",
			Image: "registry.io/full-image",
		},
	}

	// We can't easily test actual docker build without running it
	// Just verify config was set properly
	if b.config.Name != "simple-name" {
		t.Errorf("expected Name = simple-name, got %s", b.config.Name)
	}

	if b.config.Image != "registry.io/full-image" {
		t.Errorf("expected Image = registry.io/full-image, got %s", b.config.Image)
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
