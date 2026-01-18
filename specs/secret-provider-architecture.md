# Feature: Secret Provider Architecture with Vault Integration

## Feature Description
Implement a clean, composable secret provider architecture that decouples secret management from build plugins. This feature introduces a provider interface that allows multiple secret backends (Vault, AWS Secrets Manager, Azure Key Vault, etc.) to inject secrets into the build environment without builders needing to know about secret sources. The initial implementation will support HashiCorp Vault with AppRole authentication, but the architecture is designed to be extensible for future providers.

The key innovation is that secret fetching and injection happens at the **lifecycle orchestration layer** before plugin configuration, keeping builders completely agnostic to secret sources. This follows the composition-over-inheritance pattern and promotes separation of concerns.

## User Story
As a **DevOps engineer**
I want to **securely inject secrets from Vault (or other providers) into my build process**
So that **my builds can access sensitive credentials without hardcoding them in configuration files or builder code**

## Problem Statement
Currently, the nixpacks and csdocker builders have Vault configuration fields defined, but the implementation is either:
1. **Not implemented** (cloudstation-orchestrator) - Vault fields are placeholders
2. **Tightly coupled** (cloudstation-server/base-plugins) - Vault logic is embedded in the builder's Build() method, violating separation of concerns

The tight coupling approach has several issues:
- **Code duplication** - Every builder that needs Vault must implement the same authentication and secret fetching logic
- **Security risks** - Secrets are logged, TLS verification is disabled, no secret lifecycle management
- **No abstraction** - Cannot swap Vault for other providers without modifying every builder
- **Difficult to test** - Vault logic is intertwined with build logic
- **Single Responsibility Principle violation** - Builders should build, not manage secrets

## Solution Statement
Implement a **Secret Provider Interface** with a Vault provider implementation, and integrate it into the **lifecycle executor** layer. The executor will:

1. Check if the build configuration contains secret provider config (e.g., Vault fields)
2. Initialize the appropriate secret provider
3. Fetch secrets from the provider
4. Enrich the build configuration's `env` map with the fetched secrets
5. Pass the enriched configuration to the builder

This approach provides:
- ✅ **Clean separation of concerns** - Builders know nothing about secret sources
- ✅ **Reusability** - Works for any builder (nixpacks, docker, pack, etc.)
- ✅ **Testability** - Secret providers can be mocked/stubbed
- ✅ **Extensibility** - Easy to add new providers (AWS, Azure, etc.)
- ✅ **Security** - Centralized secret handling with proper redaction and lifecycle management
- ✅ **Configurability** - TLS settings, retry logic, caching can be configured globally

## Relevant Files
Use these files to implement the feature:

**Core Infrastructure:**
- `pkg/secrets/provider.go` - Secret provider interface definition
  - Defines the `Provider` interface that all secret providers must implement
  - Provides common types like `ProviderConfig` and `SecretData`

- `pkg/secrets/vault/client.go` - Vault client implementation
  - Port from `cloudstation-server/base-plugins/vaultlib/vault.go`
  - Fix security issues (TLS verification, logging)
  - Add retry logic and better error handling

- `pkg/secrets/vault/provider.go` - Vault provider implementation
  - Implements the `Provider` interface
  - Handles Vault-specific configuration and authentication
  - Maps Vault KV v1/v2 to the common secret format

**Integration Layer:**
- `internal/lifecycle/executor.go` - Lifecycle executor (MODIFY)
  - Add secret provider initialization
  - Enrich build config with secrets before calling builder.ConfigSet()
  - Lines 82-97 in ExecuteBuild() method need modification

- `internal/lifecycle/secrets.go` - Secret enrichment logic (NEW)
  - Helper functions to detect secret provider config
  - Config enrichment logic
  - Secret redaction for logging

**Configuration:**
- `internal/config/types.go` - Configuration types (MODIFY)
  - Already has Vault fields in PluginConfig via the `Config` map
  - May need to add SecretProviderConfig type for clarity

**Builder Updates:**
- `builtin/nixpacks/plugin.go` - Nixpacks builder (MODIFY)
  - Remove Vault-specific logic (it's just placeholder fields currently)
  - Vault fields can stay for backward compatibility but won't be used directly
  - Secrets will come through the `env` map

- `builtin/csdocker/plugin.go` - Docker builder (MODIFY)
  - Same as nixpacks - keep fields for compatibility but use enriched `env` map

**Testing:**
- `pkg/secrets/vault/provider_test.go` - Unit tests for Vault provider
- `pkg/secrets/provider_test.go` - Interface contract tests
- `internal/lifecycle/secrets_test.go` - Secret enrichment tests
- `internal/lifecycle/executor_test.go` - Integration tests with mock provider

**Documentation:**
- `docs/SECRETS.md` - Secret provider documentation (NEW)
- `docs/PLUGINS.md` - Update with secret injection pattern (MODIFY)
- `examples/vault-example.hcl` - Example configuration (NEW)

### New Files
- `pkg/secrets/provider.go` - Secret provider interface
- `pkg/secrets/vault/client.go` - Vault client
- `pkg/secrets/vault/provider.go` - Vault provider implementation
- `pkg/secrets/vault/config.go` - Vault-specific configuration
- `internal/lifecycle/secrets.go` - Secret enrichment helpers
- `pkg/secrets/vault/provider_test.go` - Vault provider tests
- `pkg/secrets/provider_test.go` - Interface tests
- `internal/lifecycle/secrets_test.go` - Enrichment tests
- `docs/SECRETS.md` - Secret provider documentation
- `examples/vault-example.hcl` - Vault example config

## Implementation Plan

### Phase 1: Foundation - Secret Provider Interface
Create the foundational secret provider interface and types that all providers will implement. This establishes the contract for secret providers without implementing any specific provider yet.

**Goal:** Define clean abstractions for secret management
**Dependencies:** None
**Outcome:** A well-defined interface that any secret provider can implement

### Phase 2: Core Implementation - Vault Provider
Implement the Vault provider by porting and improving the existing vaultlib code. This includes fixing security issues, adding proper error handling, retry logic, and implementing the Provider interface.

**Goal:** Working Vault provider with security best practices
**Dependencies:** Phase 1 complete
**Outcome:** Vault provider that can authenticate and fetch secrets securely

### Phase 3: Integration - Lifecycle Layer Secret Enrichment
Integrate the secret provider into the lifecycle executor. This is where secrets are fetched and injected into the build configuration before builders are invoked.

**Goal:** Automatic secret injection at the orchestration layer
**Dependencies:** Phases 1-2 complete
**Outcome:** Builders receive enriched configuration with secrets, completely agnostic to the source

## Step by Step Tasks

### 1. Create Secret Provider Interface
- Define `pkg/secrets/provider.go` with the `Provider` interface
  - `FetchSecrets(ctx context.Context, config ProviderConfig) (map[string]string, error)` method
  - `Name() string` method for provider identification
  - `ValidateConfig(config ProviderConfig) error` method for config validation
- Define `ProviderConfig` type as `map[string]interface{}` for flexibility
- Define `SecretData` type for structured secret representation
- Add documentation comments explaining the interface contract
- Add helper types for common error cases (authentication failed, secret not found, etc.)

### 2. Create Vault Client
- Create `pkg/secrets/vault/client.go` by porting from `cloudstation-server/base-plugins/vaultlib/vault.go`
- **Fix security issues:**
  - Make TLS verification configurable instead of hardcoded `InsecureSkipVerify: true`
  - Add `TLSConfig` struct with `InsecureSkipVerify` and `CACert` fields
  - Default to secure (verify TLS) with opt-in to skip verification
- Add retry logic using exponential backoff
  - Use `github.com/hashicorp/go-retryablehttp` library (already in dependencies)
  - Configurable max retries (default 3)
  - Configurable timeout (default 30s)
- Improve error messages with context (which address failed, what operation, etc.)
- Add context support to replace `context.Background()` hardcoding
- Add `Close()` method for cleanup
- Add structured logging with hclog
- **Do not log secret values** - only log keys and metadata

### 3. Create Vault Provider Implementation
- Create `pkg/secrets/vault/provider.go` implementing `secrets.Provider` interface
- Define `VaultConfig` struct with fields:
  - `Address` (string, required) - Vault server URL
  - `RoleID` (string, required) - AppRole role ID
  - `SecretID` (string, required) - AppRole secret ID
  - `SecretsPath` (string, required) - Path to secrets (e.g., "secret/data/app")
  - `TLSConfig` (*TLSConfig, optional) - TLS configuration
  - `Timeout` (time.Duration, optional) - Request timeout
  - `MaxRetries` (int, optional) - Max retry attempts
- Implement `FetchSecrets()` method:
  - Initialize Vault client with config
  - Authenticate using AppRole
  - Fetch secrets from SecretsPath
  - Convert `map[string]interface{}` to `map[string]string` (all values as strings)
  - Handle both KV v1 and KV v2 paths
- Implement `Name()` returning "vault"
- Implement `ValidateConfig()` to check required fields
- Add error wrapping with clear messages
- Add logging (info for success, error for failures)

### 4. Create Vault Configuration Helpers
- Create `pkg/secrets/vault/config.go` for configuration parsing
- Implement `ParseConfig(raw map[string]interface{}) (*VaultConfig, error)` function
  - Extract and validate all Vault-specific fields
  - Handle type assertions gracefully
  - Apply defaults for optional fields
  - Return clear validation errors
- Implement `TLSConfig` parsing from config map
- Add helper to detect Vault config in generic config map: `HasVaultConfig(config map[string]interface{}) bool`

### 5. Create Secret Enrichment Logic
- Create `internal/lifecycle/secrets.go` with helper functions
- Implement `detectSecretProvider(config map[string]interface{}) (string, bool)` function
  - Checks for vault_address, role_id, secret_id, secrets_path fields
  - Returns ("vault", true) if found, ("", false) otherwise
  - Extensible for future providers (aws, azure, etc.)
- Implement `enrichConfigWithSecrets(ctx context.Context, provider secrets.Provider, config map[string]interface{}) error`
  - Extract provider-specific config
  - Call provider.FetchSecrets()
  - Merge secrets into config["env"] map
  - Create env map if it doesn't exist
  - **Do not overwrite** existing env vars - secrets are additive
  - Log secret keys (not values) being injected
- Implement `redactSecretsFromLog(config map[string]interface{}) map[string]interface{}`
  - Returns a copy of config with secret values replaced by "[REDACTED]"
  - Used for safe logging of configuration
- Add error handling for provider failures (auth failed, secret not found, network errors)

### 6. Integrate into Lifecycle Executor
- Modify `internal/lifecycle/executor.go` in the `ExecuteBuild()` method
- **Before** calling `e.pluginLoader.LoadBuilder()` (line 86):
  - Call `detectSecretProvider(app.Build.Config)` to check for secret provider config
  - If found, initialize the appropriate provider (Vault for now)
  - Call `enrichConfigWithSecrets()` to fetch and inject secrets
  - Log success/failure of secret injection (with redacted config)
- Pass the enriched config to `LoadBuilder()`
- Add error handling - if secret fetching fails, the build should fail fast with clear error
- Add debug logging showing which provider is being used
- Add metrics/timing for secret fetching operations (optional but recommended)

### 7. Add Vault Provider to Plugin Registry
- Create `pkg/secrets/registry.go` for provider registration (similar to plugin registry)
- Implement provider registry pattern:
  ```go
  var providers = make(map[string]Provider)

  func RegisterProvider(name string, provider Provider)
  func GetProvider(name string) (Provider, error)
  ```
- In `pkg/secrets/vault/provider.go` add `init()` function:
  ```go
  func init() {
      secrets.RegisterProvider("vault", &VaultProvider{})
  }
  ```
- Import vault provider in `cmd/cloudstation/main.go`:
  ```go
  import _ "github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets/vault"
  ```

### 8. Write Unit Tests for Secret Provider Interface
- Create `pkg/secrets/provider_test.go`
- Create a mock provider for testing:
  ```go
  type MockProvider struct {
      secrets map[string]string
      err     error
  }
  ```
- Test provider interface contract:
  - Mock provider returns expected secrets
  - Mock provider handles errors correctly
  - Provider validation works
- Test error scenarios:
  - Provider returns error
  - Provider returns nil secrets
  - Provider validation fails

### 9. Write Unit Tests for Vault Provider
- Create `pkg/secrets/vault/provider_test.go`
- Test configuration parsing:
  - Valid config is parsed correctly
  - Missing required fields return validation errors
  - Optional fields have correct defaults
  - Invalid types are handled gracefully
- Test config detection:
  - HasVaultConfig() detects Vault config correctly
  - Returns false for non-Vault config
- Mock the Vault client to test provider logic:
  - Successful secret fetching
  - Authentication failures
  - Network errors
  - Secret not found
  - KV v1 vs v2 path handling
- Test retry logic:
  - Retries on transient failures
  - Gives up after max retries
  - Succeeds on retry

### 10. Write Integration Tests for Secret Enrichment
- Create `internal/lifecycle/secrets_test.go`
- Test detectSecretProvider():
  - Detects Vault config correctly
  - Returns false for configs without secret provider
  - Handles malformed configs
- Test enrichConfigWithSecrets():
  - Secrets are added to env map
  - Existing env vars are preserved
  - New env map is created if it doesn't exist
  - Provider errors are propagated
  - Secrets are not logged
- Test redactSecretsFromLog():
  - Secret values are replaced with [REDACTED]
  - Other config values are preserved
  - Handles nested config correctly

### 11. Write Integration Tests for Lifecycle Executor
- Modify `internal/lifecycle/executor_test.go` or create new test file
- Test ExecuteBuild() with mock secret provider:
  - Secrets are injected before builder is loaded
  - Builder receives enriched config
  - Build succeeds with injected secrets
  - Build fails gracefully if secret fetching fails
- Test with real Vault provider (optional, requires Vault instance):
  - Set up test Vault server
  - Create test AppRole and secrets
  - Run full build with Vault integration
  - Verify secrets are available to builder
- Test error scenarios:
  - Invalid Vault config
  - Vault authentication failure
  - Secret path not found
  - Network timeout

### 12. Update Builder Implementations
- Modify `builtin/nixpacks/plugin.go`:
  - Keep Vault fields in BuilderConfig for backward compatibility
  - Add comment: `// Deprecated: Vault fields are no longer used directly. Secrets are injected via env map by the secret provider.`
  - Ensure secrets from env map are passed to nixpacks CLI
  - Remove any Vault-specific logic if it existed
- Modify `builtin/csdocker/plugin.go`:
  - Same changes as nixpacks
  - Ensure env map is used for environment variables
- Test that builders still work with enriched env map

### 13. Create Documentation
- Create `docs/SECRETS.md` with:
  - Overview of secret provider architecture
  - How to use Vault provider
  - Configuration reference for Vault
  - Security best practices
  - Example HCL configurations
  - Troubleshooting guide
  - How to implement custom secret providers
- Update `docs/PLUGINS.md`:
  - Add section on secret injection
  - Explain that builders receive secrets via env map
  - Show example of accessing secrets in custom builders
- Update `docs/ARCHITECTURE.md` (if exists):
  - Add secret provider to architecture diagram
  - Explain where secret enrichment happens in the lifecycle

### 14. Create Example Configurations
- Create `examples/vault-example.hcl`:
  ```hcl
  project = "my-project"

  app "web" {
    build {
      use = "nixpacks"
      name = "my-app"
      tag = "latest"
      context = "."

      # Vault configuration for secret injection
      vault_address = "https://vault.example.com:8200"
      role_id = env("VAULT_ROLE_ID")
      secret_id = env("VAULT_SECRET_ID")
      secrets_path = "secret/data/myapp"

      # Optional: TLS configuration
      # vault_tls_skip_verify = false
      # vault_ca_cert = "/path/to/ca.pem"

      # Secrets from Vault will be merged into env
      env = {
        PORT = "3000"
        # Additional non-secret env vars
      }
    }

    deploy {
      use = "noop"
    }
  }
  ```
- Add comments explaining each field
- Show different scenarios (dev vs prod, multiple apps, etc.)

### 15. Add Command-line Flags for Secret Provider
- Modify `cmd/cloudstation/commands.go` to add flags:
  - `--vault-addr` - Override Vault address
  - `--vault-role-id` - Override role ID
  - `--vault-secret-id` - Override secret ID
  - `--vault-tls-skip-verify` - Skip TLS verification (for dev/testing)
- Allow CLI flags to override config file values
- Add flag documentation in help text

### 16. Run All Validation Commands
- Run all commands in the Validation Commands section
- Fix any failures
- Ensure zero regressions
- Verify secret injection works end-to-end
- Validate that secrets are not logged
- Check that TLS verification is enabled by default

## Testing Strategy

### Unit Tests

**Provider Interface Tests (`pkg/secrets/provider_test.go`):**
- Mock provider implements interface correctly
- Provider validation works
- Error handling is correct
- Provider registry works (register, get, duplicate names)

**Vault Client Tests (`pkg/secrets/vault/client_test.go`):**
- Vault client initialization with valid config
- Vault client initialization with invalid config (bad URL, etc.)
- TLS configuration (verify enabled, skip verify, custom CA)
- Retry logic on transient failures
- Timeout handling
- Context cancellation

**Vault Provider Tests (`pkg/secrets/vault/provider_test.go`):**
- Configuration parsing (valid, invalid, missing fields, defaults)
- FetchSecrets() with mock Vault client (success, auth failure, not found)
- KV v1 vs KV v2 path handling
- Type conversion (interface{} to string)
- Error wrapping and messages
- Config validation

**Secret Enrichment Tests (`internal/lifecycle/secrets_test.go`):**
- detectSecretProvider() detects Vault config
- detectSecretProvider() returns false for non-secret configs
- enrichConfigWithSecrets() adds secrets to env map
- enrichConfigWithSecrets() preserves existing env vars
- enrichConfigWithSecrets() creates env map if missing
- enrichConfigWithSecrets() handles provider errors
- redactSecretsFromLog() redacts secret values
- redactSecretsFromLog() preserves non-secret values

### Integration Tests

**Lifecycle Executor Tests (`internal/lifecycle/executor_test.go`):**
- ExecuteBuild() with mock secret provider injects secrets
- Builder receives enriched config with secrets in env map
- Build fails if secret provider fails
- Build succeeds without secret provider (backward compatibility)
- Multiple builds with different secret configs
- Secret provider errors are logged correctly

**End-to-End Tests (manual or automated):**
- Full build with Vault integration (requires test Vault server)
- Verify secrets are available in nixpacks build
- Verify secrets are not logged in output
- Verify TLS verification works
- Test with invalid Vault config (should fail gracefully)
- Test with missing secrets (should fail with clear error)

### Edge Cases

**Configuration Edge Cases:**
- Empty config map
- Config with only some Vault fields (missing required)
- Config with invalid Vault address (malformed URL)
- Config with expired Vault token
- Config with Vault fields but empty values
- Config with both Vault and env vars (should merge)
- Config with env var that conflicts with Vault secret (Vault should not override)

**Vault Edge Cases:**
- Vault server is unreachable (network timeout)
- Vault server returns 404 (secret not found)
- Vault server returns 403 (permission denied)
- Vault server returns 500 (server error, should retry)
- Vault AppRole authentication fails (invalid role/secret ID)
- Vault secret path is for KV v1 vs v2 (both should work)
- Vault secret contains non-string values (should convert to string)
- Vault secret contains nested objects (should flatten or error)

**Lifecycle Edge Cases:**
- Build canceled mid-secret-fetch (context cancellation)
- Multiple apps with different secret providers
- Build without secret provider (should work normally)
- Build with secret provider but fetch returns empty map
- Secret provider returns nil map (should error)
- Secret enrichment fails partway through (should rollback or fail atomic)

**Security Edge Cases:**
- Secrets logged in debug mode (should be redacted)
- Secrets in error messages (should be redacted)
- Secrets in config displayed to user (should be redacted)
- TLS verification disabled (should log warning)
- Secrets in environment variables of spawned processes (acceptable, this is the goal)

## Acceptance Criteria

1. **Secret Provider Interface**
   - ✅ `Provider` interface is defined with `FetchSecrets()`, `Name()`, `ValidateConfig()` methods
   - ✅ Interface is well-documented with examples
   - ✅ Provider registry allows registering and retrieving providers by name

2. **Vault Provider Implementation**
   - ✅ Vault provider implements the `Provider` interface
   - ✅ Supports HashiCorp Vault with AppRole authentication
   - ✅ Supports both KV v1 and KV v2 secret engines
   - ✅ TLS verification is **enabled by default** with opt-in to skip
   - ✅ Configurable timeout and retry logic
   - ✅ Proper error handling with clear, actionable error messages
   - ✅ Structured logging with no secret value exposure

3. **Lifecycle Integration**
   - ✅ Lifecycle executor detects secret provider config before building
   - ✅ Secrets are fetched and injected into `env` map before builder is configured
   - ✅ Builders receive enriched config without knowing about secret source
   - ✅ Build fails gracefully if secret fetching fails
   - ✅ Build works normally without secret provider (backward compatible)

4. **Security Requirements**
   - ✅ Secrets are **never logged** in plain text (keys logged, values redacted)
   - ✅ TLS verification is enabled by default
   - ✅ Secret values in error messages are redacted
   - ✅ Config displayed to users has secrets redacted
   - ✅ Warning logged if TLS verification is disabled

5. **Builder Updates**
   - ✅ Nixpacks builder uses secrets from `env` map
   - ✅ Csdocker builder uses secrets from `env` map
   - ✅ Vault fields in builders are marked as deprecated but still present for compatibility
   - ✅ Builders are completely agnostic to secret source

6. **Testing**
   - ✅ All unit tests pass (provider, client, enrichment, executor)
   - ✅ Integration tests demonstrate end-to-end secret injection
   - ✅ Edge cases are covered (network errors, auth failures, missing secrets)
   - ✅ Zero regressions in existing functionality

7. **Documentation**
   - ✅ `docs/SECRETS.md` explains secret provider architecture
   - ✅ Vault provider configuration is documented with examples
   - ✅ Security best practices are documented
   - ✅ Example HCL configs show Vault usage
   - ✅ Plugin docs explain secret injection pattern

8. **User Experience**
   - ✅ Clear error messages when Vault config is invalid
   - ✅ Clear error messages when authentication fails
   - ✅ Clear error messages when secrets are not found
   - ✅ Helpful logs guide users through secret injection process
   - ✅ Configuration is intuitive and follows Vault conventions

## Validation Commands
Execute every command to validate the feature works correctly with zero regressions.

```bash
# 1. Install dependencies (Vault client library should already be present)
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go mod download

# 2. Run unit tests for secret provider interface
go test ./pkg/secrets/... -v -cover

# 3. Run unit tests for Vault provider
go test ./pkg/secrets/vault/... -v -cover

# 4. Run tests for secret enrichment logic
go test ./internal/lifecycle/... -v -cover -run TestSecrets

# 5. Run all lifecycle executor tests
go test ./internal/lifecycle/... -v -cover

# 6. Run all tests to ensure no regressions
go test ./... -v

# 7. Build the binary
go build -o bin/cs ./cmd/cloudstation

# 8. Verify binary was created
ls -lh bin/cs

# 9. Test with example Vault config (requires running Vault instance)
# Set up test Vault server with AppRole and secrets
# Then run:
export VAULT_ROLE_ID="test-role-id"
export VAULT_SECRET_ID="test-secret-id"
./bin/cs build --app web -c examples/vault-example.hcl

# 10. Verify secrets are injected (check logs for secret keys, not values)
./bin/cs --log-level debug build --app web -c examples/vault-example.hcl 2>&1 | grep -i "secret"

# 11. Verify secrets are redacted in logs
./bin/cs --log-level debug build --app web -c examples/vault-example.hcl 2>&1 | grep -i "REDACTED"

# 12. Test without Vault config (should work normally)
./bin/cs build --app api -c cloudstation.hcl

# 13. Test with invalid Vault config (should fail gracefully)
# Create config with invalid Vault address
./bin/cs build --app web -c examples/vault-invalid.hcl 2>&1

# 14. Verify TLS verification is enabled by default
# Should fail if Vault has self-signed cert and skip_verify is not set
./bin/cs build --app web -c examples/vault-tls.hcl

# 15. Verify TLS skip works when explicitly enabled
./bin/cs build --app web -c examples/vault-tls-skip.hcl

# 16. Run formatter
go fmt ./pkg/secrets/... ./internal/lifecycle/...

# 17. Run linter
go vet ./pkg/secrets/... ./internal/lifecycle/...

# 18. Check test coverage
go test ./pkg/secrets/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html and verify >80% coverage

# 19. Run existing nixpacks tests to ensure no regressions
go test ./builtin/nixpacks/... -v

# 20. Run all builtin plugin tests
go test ./builtin/... -v

# 21. Integration test: Build cs-runner with Vault secrets
cd /Users/oumnyabenhassou/Code/runner/cs-runner
# Add Vault config to cloudstation.hcl
/Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator/bin/cs build --app api

# 22. Verify Docker image was created with secrets available during build
docker images | grep cs-runner-api

# 23. Clean up test artifacts
docker rmi cs-runner-api:test
rm coverage.out coverage.html
```

## Notes

### Dependencies
- **hashicorp/vault/api** - Already in dependencies, used for Vault client
- **hashicorp/go-retryablehttp** - Already in dependencies, used for retry logic
- **hashicorp/go-hclog** - Already in dependencies, used for logging
- No new external dependencies needed

### Security Considerations
1. **TLS Verification**: Default to enabled, log warning if disabled
2. **Secret Logging**: Never log secret values, only keys
3. **Error Messages**: Redact secrets from error messages
4. **Token Storage**: Vault tokens are ephemeral, not persisted
5. **Secret Lifecycle**: Secrets exist only for the duration of the build
6. **Audit Trail**: Log when secrets are fetched and which keys are injected

### Future Enhancements (Not in this Feature)
1. **AWS Secrets Manager Provider** - Implement `Provider` interface for AWS
2. **Azure Key Vault Provider** - Implement `Provider` interface for Azure
3. **File-based Provider** - Read secrets from encrypted files
4. **Multi-provider Support** - Merge secrets from multiple providers
5. **Secret Caching** - Cache fetched secrets for repeated builds
6. **Secret Versioning** - Support specific secret versions
7. **Secret Rotation** - Handle secret rotation during long builds
8. **Secret Filtering** - Only fetch specific secrets, not all from path
9. **Secret Transformation** - Transform secret keys/values before injection
10. **Metrics** - Prometheus metrics for secret fetching latency, errors, etc.

### Migration from Old Approach
Users currently using the cloudstation-server/base-plugins approach:
1. Update HCL config to use new field names (if any changed)
2. No code changes needed - secrets injected automatically
3. Rebuild with new cloudstation-orchestrator binary
4. Vault config stays in build block, works transparently
5. Can remove vaultlib dependency from their codebase

### Backward Compatibility
- Vault fields in BuilderConfig are preserved for compatibility
- Builds without Vault config work exactly as before
- No breaking changes to existing HCL configurations
- Secret provider is opt-in based on config presence

### Development Workflow
1. Implement foundation (Provider interface)
2. Implement Vault provider with tests
3. Integrate into lifecycle executor
4. Test with real Vault instance
5. Update builders to use enriched env
6. Write documentation and examples
7. Run full validation suite
8. Deploy to staging for testing
9. Roll out to production

### Testing with Local Vault
For development and testing, use Vault in dev mode:
```bash
# Start Vault dev server
vault server -dev -dev-root-token-id=root

# Set up AppRole
vault auth enable approle
vault write auth/approle/role/test-role \
    secret_id_ttl=24h \
    token_ttl=1h \
    token_max_ttl=4h

# Get role ID
vault read auth/approle/role/test-role/role-id

# Generate secret ID
vault write -f auth/approle/role/test-role/secret-id

# Write test secrets
vault kv put secret/myapp \
    DATABASE_URL="postgres://localhost:5432/mydb" \
    API_KEY="test-api-key-123"
```

### Performance Considerations
- Secret fetching adds ~100-500ms to build time (one-time cost)
- Consider caching for repeated builds (future enhancement)
- Vault client reuses HTTP connections (connection pooling)
- Timeout prevents hanging builds
- Retry logic handles transient failures without user intervention
