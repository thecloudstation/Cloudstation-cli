package lifecycle

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets"
)

// MockSecretProvider for testing
type MockSecretProvider struct {
	name        string
	secrets     map[string]string
	err         error
	validateErr error
}

func (m *MockSecretProvider) Name() string {
	return m.name
}

func (m *MockSecretProvider) FetchSecrets(ctx context.Context, config secrets.ProviderConfig) (map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.secrets, nil
}

func (m *MockSecretProvider) ValidateConfig(config secrets.ProviderConfig) error {
	if m.validateErr != nil {
		return m.validateErr
	}
	return nil
}

func TestDetectSecretProvider(t *testing.T) {
	tests := []struct {
		name             string
		config           map[string]interface{}
		expectedProvider string
		expectedFound    bool
	}{
		{
			name: "vault config detected",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
				"secrets_path":  "secret/data/app",
			},
			expectedProvider: "vault",
			expectedFound:    true,
		},
		{
			name: "no secret provider",
			config: map[string]interface{}{
				"name": "my-app",
				"tag":  "latest",
			},
			expectedProvider: "",
			expectedFound:    false,
		},
		{
			name: "partial vault config",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				// missing secret_id and secrets_path
			},
			expectedProvider: "",
			expectedFound:    false,
		},
		{
			name:             "empty config",
			config:           map[string]interface{}{},
			expectedProvider: "",
			expectedFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, found := detectSecretProvider(tt.config)

			if found != tt.expectedFound {
				t.Errorf("expected found=%v, got %v", tt.expectedFound, found)
			}

			if provider != tt.expectedProvider {
				t.Errorf("expected provider=%q, got %q", tt.expectedProvider, provider)
			}
		})
	}
}

func TestEnrichConfigWithSecrets(t *testing.T) {
	logger := hclog.NewNullLogger()
	ctx := context.Background()

	tests := []struct {
		name           string
		provider       *MockSecretProvider
		config         map[string]interface{}
		expectedEnv    map[string]interface{}
		expectError    bool
		validateResult func(*testing.T, map[string]interface{})
	}{
		{
			name: "secrets added to env map",
			provider: &MockSecretProvider{
				name: "test",
				secrets: map[string]string{
					"DB_PASSWORD": "secret123",
					"API_KEY":     "key456",
				},
			},
			config: map[string]interface{}{
				"name": "my-app",
			},
			expectError: false,
			validateResult: func(t *testing.T, config map[string]interface{}) {
				env, ok := config["env"].(map[string]interface{})
				if !ok {
					t.Fatal("env map not found in config")
				}

				if env["DB_PASSWORD"] != "secret123" {
					t.Errorf("expected DB_PASSWORD=secret123, got %v", env["DB_PASSWORD"])
				}

				if env["API_KEY"] != "key456" {
					t.Errorf("expected API_KEY=key456, got %v", env["API_KEY"])
				}
			},
		},
		{
			name: "existing env vars preserved",
			provider: &MockSecretProvider{
				name: "test",
				secrets: map[string]string{
					"DB_PASSWORD": "secret123",
				},
			},
			config: map[string]interface{}{
				"env": map[string]interface{}{
					"PORT":     "3000",
					"NODE_ENV": "production",
				},
			},
			expectError: false,
			validateResult: func(t *testing.T, config map[string]interface{}) {
				env, ok := config["env"].(map[string]interface{})
				if !ok {
					t.Fatal("env map not found in config")
				}

				// Existing vars should be preserved
				if env["PORT"] != "3000" {
					t.Errorf("expected PORT=3000, got %v", env["PORT"])
				}
				if env["NODE_ENV"] != "production" {
					t.Errorf("expected NODE_ENV=production, got %v", env["NODE_ENV"])
				}

				// New secret should be added
				if env["DB_PASSWORD"] != "secret123" {
					t.Errorf("expected DB_PASSWORD=secret123, got %v", env["DB_PASSWORD"])
				}
			},
		},
		{
			name: "secrets not overwrite existing env vars",
			provider: &MockSecretProvider{
				name: "test",
				secrets: map[string]string{
					"PORT": "5432", // tries to override existing PORT
				},
			},
			config: map[string]interface{}{
				"env": map[string]interface{}{
					"PORT": "3000", // existing value
				},
			},
			expectError: false,
			validateResult: func(t *testing.T, config map[string]interface{}) {
				env, ok := config["env"].(map[string]interface{})
				if !ok {
					t.Fatal("env map not found in config")
				}

				// Existing value should not be overwritten
				if env["PORT"] != "3000" {
					t.Errorf("expected PORT=3000 (not overwritten), got %v", env["PORT"])
				}
			},
		},
		{
			name: "empty secrets from provider",
			provider: &MockSecretProvider{
				name:    "test",
				secrets: map[string]string{},
			},
			config: map[string]interface{}{
				"name": "my-app",
			},
			expectError: false,
			validateResult: func(t *testing.T, config map[string]interface{}) {
				// When no secrets are fetched, env map should not be created
				// (function returns early)
				if _, exists := config["env"]; exists {
					t.Error("env map should not be created when no secrets are fetched")
				}
			},
		},
		{
			name: "provider validation error",
			provider: &MockSecretProvider{
				name:        "test",
				validateErr: secrets.ErrInvalidConfig,
			},
			config: map[string]interface{}{
				"name": "my-app",
			},
			expectError: true,
		},
		{
			name: "provider fetch error",
			provider: &MockSecretProvider{
				name: "test",
				err:  secrets.ErrAuthenticationFailed,
			},
			config: map[string]interface{}{
				"name": "my-app",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of config for the test
			configCopy := make(map[string]interface{})
			for k, v := range tt.config {
				configCopy[k] = v
			}

			err := enrichConfigWithSecrets(ctx, tt.provider, configCopy, logger)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, configCopy)
			}
		})
	}
}

func TestRedactSecretsFromLog(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		validate func(*testing.T, map[string]interface{})
	}{
		{
			name: "redact secret_id",
			config: map[string]interface{}{
				"vault_address": "https://vault.example.com",
				"role_id":       "test-role",
				"secret_id":     "test-secret",
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				if redacted["vault_address"] != "https://vault.example.com" {
					t.Error("vault_address should not be redacted")
				}
				if redacted["role_id"] != "test-role" {
					t.Error("role_id should not be redacted")
				}
				if redacted["secret_id"] != "[REDACTED]" {
					t.Errorf("secret_id should be redacted, got %v", redacted["secret_id"])
				}
			},
		},
		{
			name: "redact env values",
			config: map[string]interface{}{
				"env": map[string]interface{}{
					"DB_PASSWORD": "secret123",
					"API_KEY":     "key456",
					"PORT":        "3000",
				},
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				env, ok := redacted["env"].(map[string]interface{})
				if !ok {
					t.Fatal("env should still be a map")
				}

				// All env values should be redacted
				for key, value := range env {
					if value != "[REDACTED]" {
						t.Errorf("env[%q] should be redacted, got %v", key, value)
					}
				}
			},
		},
		{
			name: "redact password field",
			config: map[string]interface{}{
				"username": "admin",
				"password": "secretpass",
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				if redacted["username"] != "admin" {
					t.Error("username should not be redacted")
				}
				if redacted["password"] != "[REDACTED]" {
					t.Errorf("password should be redacted, got %v", redacted["password"])
				}
			},
		},
		{
			name: "redact api_key",
			config: map[string]interface{}{
				"service": "myservice",
				"api_key": "sk-1234567890",
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				if redacted["service"] != "myservice" {
					t.Error("service should not be redacted")
				}
				if redacted["api_key"] != "[REDACTED]" {
					t.Errorf("api_key should be redacted, got %v", redacted["api_key"])
				}
			},
		},
		{
			name: "preserve non-secret values",
			config: map[string]interface{}{
				"name":    "my-app",
				"tag":     "latest",
				"context": ".",
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				if redacted["name"] != "my-app" {
					t.Error("name should not be redacted")
				}
				if redacted["tag"] != "latest" {
					t.Error("tag should not be redacted")
				}
				if redacted["context"] != "." {
					t.Error("context should not be redacted")
				}
			},
		},
		{
			name: "nested config redaction",
			config: map[string]interface{}{
				"database": map[string]interface{}{
					"host":     "localhost",
					"password": "dbpass",
				},
			},
			validate: func(t *testing.T, redacted map[string]interface{}) {
				db, ok := redacted["database"].(map[string]interface{})
				if !ok {
					t.Fatal("database should still be a map")
				}

				if db["host"] != "localhost" {
					t.Error("host should not be redacted")
				}
				if db["password"] != "[REDACTED]" {
					t.Errorf("nested password should be redacted, got %v", db["password"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted := redactSecretsFromLog(tt.config)

			// Ensure original config is not modified
			if &redacted == &tt.config {
				t.Error("redaction should return a copy, not modify original")
			}

			if tt.validate != nil {
				tt.validate(t, redacted)
			}
		})
	}
}
