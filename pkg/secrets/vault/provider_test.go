package vault

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets"
)

func TestVaultProvider_Name(t *testing.T) {
	provider := NewProvider()
	if provider.Name() != "vault" {
		t.Errorf("expected name 'vault', got %q", provider.Name())
	}
}

func TestVaultProvider_ValidateConfig(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name        string
		config      secrets.ProviderConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: secrets.ProviderConfig{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectError: false,
		},
		{
			name: "missing vault_address",
			config: secrets.ProviderConfig{
				"role_id":      "test-role",
				"secret_id":    "test-secret",
				"secrets_path": "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "missing role_id",
			config: secrets.ProviderConfig{
				"vault_address": "https://vault.example.com",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "missing secret_id",
			config: secrets.ProviderConfig{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secrets_path":  "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "missing secrets_path",
			config: secrets.ProviderConfig{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
			},
			expectError: true,
		},
		{
			name: "empty vault_address",
			config: secrets.ProviderConfig{
				"vault_address": "",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "with optional TLS config",
			config: secrets.ProviderConfig{
				"vault_address":         "https://vault.example.com",
				"role_id":               "test-role",
				"secret_id":             "test-secret",
				"secrets_path":          "secret/data/app",
				"vault_tls_skip_verify": true,
			},
			expectError: false,
		},
		{
			name: "with optional timeout",
			config: secrets.ProviderConfig{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
				"vault_timeout": 60,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		validate    func(*testing.T, *VaultConfig)
	}{
		{
			name: "valid minimal config",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.Address != "https://vault.example.com" {
					t.Errorf("unexpected address: %q", cfg.Address)
				}
				if cfg.RoleID != "test-role" {
					t.Errorf("unexpected role_id: %q", cfg.RoleID)
				}
				if cfg.SecretID != "test-secret" {
					t.Errorf("unexpected secret_id: %q", cfg.SecretID)
				}
				if cfg.SecretsPath != "secret/data/app" {
					t.Errorf("unexpected secrets_path: %q", cfg.SecretsPath)
				}
			},
		},
		{
			name: "config with TLS skip verify",
			config: map[string]interface{}{
				"vault_address":         "https://vault.example.com",
				"role_id":               "test-role",
				"secret_id":             "test-secret",
				"secrets_path":          "secret/data/app",
				"vault_tls_skip_verify": true,
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.TLS == nil {
					t.Fatal("expected TLS config to be set")
				}
				if !cfg.TLS.InsecureSkipVerify {
					t.Error("expected InsecureSkipVerify to be true")
				}
			},
		},
		{
			name: "config with CA cert",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
				"vault_ca_cert": "/path/to/ca.pem",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.TLS == nil {
					t.Fatal("expected TLS config to be set")
				}
				if cfg.TLS.CACert != "/path/to/ca.pem" {
					t.Errorf("unexpected CA cert path: %q", cfg.TLS.CACert)
				}
			},
		},
		{
			name: "config with timeout as int",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
				"vault_timeout": 60,
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.Timeout.Seconds() != 60 {
					t.Errorf("unexpected timeout: %v", cfg.Timeout)
				}
			},
		},
		{
			name: "config with timeout as string",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
				"vault_timeout": "2m",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.Timeout.Minutes() != 2 {
					t.Errorf("unexpected timeout: %v", cfg.Timeout)
				}
			},
		},
		{
			name: "config with max retries",
			config: map[string]interface{}{
				"vault_address":     "https://vault.example.com",
				"role_id":           "test-role",
				"secret_id":         "test-secret",
				"secrets_path":      "secret/data/app",
				"vault_max_retries": 5,
			},
			expectError: false,
			validate: func(t *testing.T, cfg *VaultConfig) {
				if cfg.MaxRetries != 5 {
					t.Errorf("unexpected max retries: %d", cfg.MaxRetries)
				}
			},
		},
		{
			name: "missing required field",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				// missing secret_id and secrets_path
			},
			expectError: true,
		},
		{
			name: "invalid timeout string",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
				"vault_timeout": "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestHasVaultConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected bool
	}{
		{
			name: "has vault config",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expected: true,
		},
		{
			name: "missing vault_address",
			config: map[string]interface{}{
				"role_id":      "test-role",
				"secret_id":    "test-secret",
				"secrets_path": "secret/data/app",
			},
			expected: false,
		},
		{
			name: "missing role_id",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expected: false,
		},
		{
			name:     "empty config",
			config:   map[string]interface{}{},
			expected: false,
		},
		{
			name: "has some vault fields but not all",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasVaultConfig(tt.config)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestVaultConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *VaultConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &VaultConfig{
				Address:     "https://vault.example.com",
				RoleID:      "test-role",
				SecretID:    "test-secret",
				SecretsPath: "secret/data/app",
				Timeout:     30 * time.Second,
				MaxRetries:  3,
			},
			expectError: false,
		},
		{
			name: "empty address",
			config: &VaultConfig{
				Address:     "",
				RoleID:      "test-role",
				SecretID:    "test-secret",
				SecretsPath: "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "empty role_id",
			config: &VaultConfig{
				Address:     "https://vault.example.com",
				RoleID:      "",
				SecretID:    "test-secret",
				SecretsPath: "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "empty secret_id",
			config: &VaultConfig{
				Address:     "https://vault.example.com",
				RoleID:      "test-role",
				SecretID:    "",
				SecretsPath: "secret/data/app",
			},
			expectError: true,
		},
		{
			name: "empty secrets_path",
			config: &VaultConfig{
				Address:     "https://vault.example.com",
				RoleID:      "test-role",
				SecretID:    "test-secret",
				SecretsPath: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "string value",
			input:       "test-value",
			expected:    "test-value",
			expectError: false,
		},
		{
			name:        "int value",
			input:       42,
			expected:    "42",
			expectError: false,
		},
		{
			name:        "float value",
			input:       3.14,
			expected:    "3.140000",
			expectError: false,
		},
		{
			name:        "bool true",
			input:       true,
			expected:    "true",
			expectError: false,
		},
		{
			name:        "bool false",
			input:       false,
			expected:    "false",
			expectError: false,
		},
		{
			name:        "nil value",
			input:       nil,
			expected:    "",
			expectError: false,
		},
		{
			name:        "map value (unsupported)",
			input:       map[string]string{"key": "value"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "slice value (unsupported)",
			input:       []string{"one", "two"},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestVaultProvider_Integration(t *testing.T) {
	// This is a basic integration test that doesn't require a real Vault server
	// It tests the provider initialization and config validation flow

	provider := NewProviderWithLogger(hclog.NewNullLogger())

	config := secrets.ProviderConfig{
		"vault_address": "https://vault.example.com",
		"role_id":       "test-role",
		"secret_id":     "test-secret",
		"secrets_path":  "secret/data/app",
	}

	// Test validation
	if err := provider.ValidateConfig(config); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	// Test that FetchSecrets would fail with invalid Vault server
	// (we expect this to fail because there's no real Vault server)
	ctx := context.Background()
	_, err := provider.FetchSecrets(ctx, config)

	// We expect an error since we're not connecting to a real Vault
	if err == nil {
		t.Error("expected error when connecting to fake Vault server")
	}
}
