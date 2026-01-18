package lifecycle

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets/vault"
)

// detectSecretProvider checks if the config contains secret provider configuration
// and returns the provider name if found.
func detectSecretProvider(config map[string]interface{}) (string, bool) {
	// Check for Vault configuration
	if vault.HasVaultConfig(config) {
		return "vault", true
	}

	// Future: Check for AWS Secrets Manager, Azure Key Vault, etc.
	// if aws.HasAWSConfig(config) {
	//     return "aws", true
	// }

	return "", false
}

// enrichConfigWithSecrets fetches secrets from the provider and merges them into
// the config's env map. Existing env vars are preserved (not overwritten).
func enrichConfigWithSecrets(ctx context.Context, provider secrets.Provider, config map[string]interface{}, logger hclog.Logger) error {
	// Validate provider config
	if err := provider.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid secret provider config: %w", err)
	}

	// Fetch secrets from provider
	logger.Info("fetching secrets from provider", "provider", provider.Name())
	fetchedSecrets, err := provider.FetchSecrets(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to fetch secrets: %w", err)
	}

	if len(fetchedSecrets) == 0 {
		logger.Warn("no secrets fetched from provider", "provider", provider.Name())
		return nil
	}

	// Get or create the env map
	var envMap map[string]interface{}
	if envRaw, exists := config["env"]; exists {
		if envTyped, ok := envRaw.(map[string]interface{}); ok {
			envMap = envTyped
		} else {
			logger.Warn("env field exists but is not a map, creating new env map")
			envMap = make(map[string]interface{})
		}
	} else {
		envMap = make(map[string]interface{})
	}

	// Merge secrets into env map (without overwriting existing values)
	addedCount := 0
	skippedCount := 0
	for key, value := range fetchedSecrets {
		if _, exists := envMap[key]; exists {
			logger.Debug("skipping secret key (already exists in env)", "key", key)
			skippedCount++
			continue
		}
		envMap[key] = value
		addedCount++
	}

	// Update config with enriched env map
	config["env"] = envMap

	logger.Info("secrets enriched successfully",
		"provider", provider.Name(),
		"added", addedCount,
		"skipped", skippedCount,
		"total_env_vars", len(envMap))

	// Log the secret keys that were added (not values!)
	if addedCount > 0 {
		secretKeys := make([]string, 0, addedCount)
		for key := range fetchedSecrets {
			if _, exists := envMap[key]; exists {
				secretKeys = append(secretKeys, key)
			}
		}
		logger.Debug("secret keys injected", "keys", secretKeys)
	}

	return nil
}

// redactSecretsFromLog creates a copy of the config with secret values redacted.
// This is used for safe logging of configuration.
func redactSecretsFromLog(config map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{})

	for key, value := range config {
		switch key {
		// Redact known secret fields
		case "secret_id", "vault_token", "api_key", "password", "secret_key":
			redacted[key] = "[REDACTED]"
		case "env":
			// Redact environment variable values
			if envMap, ok := value.(map[string]interface{}); ok {
				redactedEnv := make(map[string]interface{})
				for envKey := range envMap {
					redactedEnv[envKey] = "[REDACTED]"
				}
				redacted[key] = redactedEnv
			} else {
				redacted[key] = value
			}
		default:
			// For nested maps, recursively redact
			if nestedMap, ok := value.(map[string]interface{}); ok {
				redacted[key] = redactSecretsFromLog(nestedMap)
			} else {
				redacted[key] = value
			}
		}
	}

	return redacted
}
