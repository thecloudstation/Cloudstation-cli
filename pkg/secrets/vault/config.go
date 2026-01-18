package vault

import (
	"fmt"
	"time"
)

// VaultConfig holds Vault-specific configuration.
type VaultConfig struct {
	// Address is the Vault server URL (e.g., "https://vault.example.com:8200")
	Address string

	// RoleID is the AppRole role ID for authentication
	RoleID string

	// SecretID is the AppRole secret ID for authentication
	SecretID string

	// SecretsPath is the path to secrets in Vault (e.g., "secret/data/myapp")
	SecretsPath string

	// TLS configuration
	TLS *TLSConfig

	// Timeout is the maximum time to wait for Vault requests
	// Default: 30 seconds
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests
	// Default: 3
	MaxRetries int
}

// ParseConfig extracts and validates Vault configuration from a generic config map.
// It handles type assertions gracefully and applies defaults for optional fields.
func ParseConfig(raw map[string]interface{}) (*VaultConfig, error) {
	config := &VaultConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	// Extract required fields
	address, ok := raw["vault_address"].(string)
	if !ok || address == "" {
		return nil, fmt.Errorf("vault_address is required and must be a string")
	}
	config.Address = address

	roleID, ok := raw["role_id"].(string)
	if !ok || roleID == "" {
		return nil, fmt.Errorf("role_id is required and must be a string")
	}
	config.RoleID = roleID

	secretID, ok := raw["secret_id"].(string)
	if !ok || secretID == "" {
		return nil, fmt.Errorf("secret_id is required and must be a string")
	}
	config.SecretID = secretID

	secretsPath, ok := raw["secrets_path"].(string)
	if !ok || secretsPath == "" {
		return nil, fmt.Errorf("secrets_path is required and must be a string")
	}
	config.SecretsPath = secretsPath

	// Extract optional TLS configuration
	if tlsSkipVerify, ok := raw["vault_tls_skip_verify"].(bool); ok {
		if config.TLS == nil {
			config.TLS = &TLSConfig{}
		}
		config.TLS.InsecureSkipVerify = tlsSkipVerify
	}

	if caCert, ok := raw["vault_ca_cert"].(string); ok && caCert != "" {
		if config.TLS == nil {
			config.TLS = &TLSConfig{}
		}
		config.TLS.CACert = caCert
	}

	// Extract optional timeout
	if timeout, ok := raw["vault_timeout"].(int); ok && timeout > 0 {
		config.Timeout = time.Duration(timeout) * time.Second
	} else if timeout, ok := raw["vault_timeout"].(string); ok && timeout != "" {
		// Try parsing as duration string (e.g., "30s", "1m")
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid vault_timeout: %w", err)
		}
		config.Timeout = d
	}

	// Extract optional max retries
	if maxRetries, ok := raw["vault_max_retries"].(int); ok && maxRetries > 0 {
		config.MaxRetries = maxRetries
	}

	return config, nil
}

// HasVaultConfig checks if the given config map contains Vault configuration.
// It looks for the presence of required Vault fields to determine if Vault is configured.
func HasVaultConfig(config map[string]interface{}) bool {
	_, hasAddress := config["vault_address"]
	_, hasRoleID := config["role_id"]
	_, hasSecretID := config["secret_id"]
	_, hasSecretsPath := config["secrets_path"]

	// All required fields must be present
	return hasAddress && hasRoleID && hasSecretID && hasSecretsPath
}

// Validate checks if the VaultConfig is valid.
func (c *VaultConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("vault address is required")
	}
	if c.RoleID == "" {
		return fmt.Errorf("role_id is required")
	}
	if c.SecretID == "" {
		return fmt.Errorf("secret_id is required")
	}
	if c.SecretsPath == "" {
		return fmt.Errorf("secrets_path is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	return nil
}
