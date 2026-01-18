package vault

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets"
)

// Register the Vault provider on package initialization
func init() {
	secrets.RegisterProvider("vault", NewProvider())
}

// Provider implements the secrets.Provider interface for HashiCorp Vault.
type Provider struct {
	logger hclog.Logger
}

// NewProvider creates a new Vault provider instance.
func NewProvider() *Provider {
	return &Provider{
		logger: hclog.Default(),
	}
}

// NewProviderWithLogger creates a new Vault provider with a custom logger.
func NewProviderWithLogger(logger hclog.Logger) *Provider {
	return &Provider{
		logger: logger,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "vault"
}

// ValidateConfig validates the Vault provider configuration.
func (p *Provider) ValidateConfig(config secrets.ProviderConfig) error {
	vaultConfig, err := ParseConfig(config)
	if err != nil {
		return secrets.NewProviderError(p.Name(), "validate", err)
	}

	if err := vaultConfig.Validate(); err != nil {
		return secrets.NewProviderError(p.Name(), "validate", err)
	}

	return nil
}

// FetchSecrets retrieves secrets from Vault using AppRole authentication.
func (p *Provider) FetchSecrets(ctx context.Context, config secrets.ProviderConfig) (map[string]string, error) {
	// Parse Vault-specific configuration
	vaultConfig, err := ParseConfig(config)
	if err != nil {
		return nil, secrets.NewProviderError(p.Name(), "parse_config", err)
	}

	// Validate configuration
	if err := vaultConfig.Validate(); err != nil {
		return nil, secrets.NewProviderError(p.Name(), "validate_config", err)
	}

	p.logger.Info("fetching secrets from Vault", "address", vaultConfig.Address, "path", vaultConfig.SecretsPath)

	// Create Vault client
	client, err := NewClient(&ClientConfig{
		Address:    vaultConfig.Address,
		TLS:        vaultConfig.TLS,
		Timeout:    vaultConfig.Timeout,
		MaxRetries: vaultConfig.MaxRetries,
		Logger:     p.logger,
	})
	if err != nil {
		return nil, secrets.NewProviderError(p.Name(), "create_client", err)
	}
	defer client.Close()

	// Authenticate using AppRole
	p.logger.Debug("authenticating to Vault using AppRole")
	if err := client.AuthenticateAppRole(ctx, vaultConfig.RoleID, vaultConfig.SecretID); err != nil {
		return nil, secrets.NewProviderError(p.Name(), "authenticate", err)
	}

	// Fetch secrets from the configured path
	p.logger.Debug("reading secrets from Vault", "path", vaultConfig.SecretsPath)
	secretData, err := client.ReadSecret(ctx, vaultConfig.SecretsPath)
	if err != nil {
		return nil, secrets.NewProviderError(p.Name(), "read_secret", err)
	}

	// Convert map[string]interface{} to map[string]string
	result := make(map[string]string)
	for key, value := range secretData {
		// Convert value to string
		strValue, err := convertToString(value)
		if err != nil {
			p.logger.Warn("skipping secret key with unconvertible value", "key", key, "error", err)
			continue
		}
		result[key] = strValue
	}

	secretKeys := make([]string, 0, len(result))
	for key := range result {
		secretKeys = append(secretKeys, key)
	}

	p.logger.Info("successfully fetched secrets from Vault", "count", len(result), "keys", secretKeys)

	return result, nil
}

// convertToString converts various types to string representation.
// Handles strings, numbers, booleans, and attempts JSON marshaling for complex types.
func convertToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case nil:
		return "", nil
	default:
		// For complex types (maps, arrays, etc.), we could JSON marshal them
		// but for now we'll return an error as they should be strings in env vars
		return "", fmt.Errorf("unsupported type %T for environment variable", value)
	}
}
