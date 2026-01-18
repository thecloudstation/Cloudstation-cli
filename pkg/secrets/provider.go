// Package secrets provides a composable interface for secret providers.
// It allows decoupling secret management from build plugins, enabling
// secrets to be injected from various backends (Vault, AWS, Azure, etc.)
// without builders needing to know about the secret source.
package secrets

import (
	"context"
	"errors"
	"fmt"
)

// Provider defines the interface that all secret providers must implement.
// Secret providers are responsible for fetching secrets from external sources
// (like HashiCorp Vault, AWS Secrets Manager, Azure Key Vault, etc.) and
// returning them as a simple key-value map that can be injected into the
// build environment.
//
// Example usage:
//
//	provider := vault.NewProvider()
//	secrets, err := provider.FetchSecrets(ctx, config)
//	if err != nil {
//	    return fmt.Errorf("failed to fetch secrets: %w", err)
//	}
//	// secrets can now be merged into the build env map
type Provider interface {
	// FetchSecrets retrieves secrets from the provider using the given configuration.
	// The config parameter is a flexible map that contains provider-specific settings.
	// Returns a map of secret keys to values, or an error if fetching fails.
	//
	// The context should be used for cancellation and timeouts.
	// Implementations should respect context cancellation and return promptly.
	//
	// All secret values are returned as strings. Providers are responsible for
	// converting their native types (e.g., JSON objects, numbers) to strings.
	FetchSecrets(ctx context.Context, config ProviderConfig) (map[string]string, error)

	// Name returns the unique identifier for this provider (e.g., "vault", "aws", "azure").
	// This name is used for provider registration and selection.
	Name() string

	// ValidateConfig checks if the provided configuration is valid for this provider.
	// It should verify that all required fields are present and have valid values.
	// Returns nil if config is valid, or a descriptive error if validation fails.
	ValidateConfig(config ProviderConfig) error
}

// ProviderConfig represents the configuration for a secret provider.
// It's a flexible map that can contain any provider-specific settings.
// Each provider implementation should define its own expected fields.
type ProviderConfig map[string]interface{}

// SecretData represents structured secret information.
// This can be used for more complex secret handling in the future,
// such as versioning, metadata, or secret rotation.
type SecretData struct {
	// Key is the secret identifier
	Key string

	// Value is the secret value as a string
	Value string

	// Metadata contains optional provider-specific metadata
	// (e.g., version, created_at, rotation_info)
	Metadata map[string]interface{}
}

// Common error types for secret providers
var (
	// ErrProviderNotFound is returned when a requested provider is not registered
	ErrProviderNotFound = errors.New("secret provider not found")

	// ErrAuthenticationFailed is returned when authentication to the secret backend fails
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrSecretNotFound is returned when a requested secret path doesn't exist
	ErrSecretNotFound = errors.New("secret not found")

	// ErrInvalidConfig is returned when provider configuration is invalid
	ErrInvalidConfig = errors.New("invalid provider configuration")

	// ErrNetworkError is returned when network communication with the provider fails
	ErrNetworkError = errors.New("network error communicating with secret provider")

	// ErrPermissionDenied is returned when the provider denies access to a secret
	ErrPermissionDenied = errors.New("permission denied")
)

// ProviderError wraps provider-specific errors with additional context.
type ProviderError struct {
	Provider string
	Op       string // Operation that failed (e.g., "authenticate", "fetch")
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s provider: %s failed: %v", e.Provider, e.Op, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError with context.
func NewProviderError(provider, op string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Op:       op,
		Err:      err,
	}
}
