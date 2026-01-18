# Chore: Implement csdocker Docker Builder Plugin

## Chore Description
Implement the full Docker build functionality for the csdocker plugin to make it a working builder that:
- Executes actual Docker builds (currently just a stub)
- Supports Vault secret injection via the existing secret provider architecture
- Passes environment variables (including secrets) as Docker build args
- Handles Dockerfile and build context configuration
- Returns proper build artifacts
- Follows the same patterns as the nixpacks plugin

The csdocker plugin is already registered and has the infrastructure in place, but the `Build()` method is a stub that doesn't execute actual Docker builds. This chore implements the full Docker build logic similar to how nixpacks works.

## Relevant Files
Use these files to implement the chore:

**Core Implementation:**
- `builtin/csdocker/plugin.go` - Main plugin file that needs Build() implementation and ConfigSet() updates
  - Currently has stub Build() method that returns fake artifact
  - ConfigSet() only parses Vault fields, needs to parse image, tag, dockerfile, context, and env
  - BuilderConfig needs additional fields (Image, Tag, Name, BuildArgs)

**Reference Implementation:**
- `builtin/nixpacks/plugin.go` - Reference for how to implement Build() method properly
  - Shows proper context handling, logger usage, command execution
  - Demonstrates env var and build arg passing
  - Shows artifact creation with proper metadata

**Testing Reference:**
- `builtin/nixpacks/builder_test.go` - Test patterns to follow for csdocker tests
  - Shows ConfigSet() testing approach
  - Shows Build() validation error testing
  - Demonstrates config validation patterns

**Secret Provider Integration (Already Working):**
- `internal/lifecycle/executor.go` - Lines 86-105 show how secrets are injected before builder is called
  - No changes needed here - secret enrichment already works for all plugins
- `internal/lifecycle/secrets.go` - Secret enrichment logic (already implemented)
  - Merges Vault secrets into config["env"] map before passing to plugin

### New Files
- `builtin/csdocker/builder_test.go` - Unit tests for csdocker plugin

## Step by Step Tasks

### 1. Update BuilderConfig struct
- Add `Name` field (string) - Docker image name (e.g., "myapp")
- Add `Image` field (string) - Full image path including registry (e.g., "myregistry.azurecr.io/myapp")
- Add `Tag` field (string) - Image tag (defaults to "latest")
- Add `BuildArgs` field (map[string]string) - Docker build arguments
- Keep existing `Dockerfile` field (defaults to "Dockerfile")
- Keep existing `Context` field (defaults to ".")
- Keep existing `Env` field (map[string]string) - Environment variables including secrets from Vault
- Keep deprecated Vault fields for backward compatibility (VaultAddress, RoleID, SecretID, SecretsPath)

### 2. Update ConfigSet() method
- Parse `name` field from config map to BuilderConfig.Name
- Parse `image` field from config map to BuilderConfig.Image
- Parse `tag` field from config map to BuilderConfig.Tag
- Parse `dockerfile` field from config map to BuilderConfig.Dockerfile
- Parse `context` field from config map to BuilderConfig.Context
- Parse `build_args` map from config to BuilderConfig.BuildArgs
- Parse `env` map from config to BuilderConfig.Env (THIS IS WHERE VAULT SECRETS ARRIVE)
  - Create helper function to handle type assertions properly
  - Support both string values and *string pointers (like nixpacks does)
- Keep existing Vault field parsing for backward compatibility
- Follow the same pattern as nixpacks ConfigSet() implementation

### 3. Implement Build() method
- Validate required configuration:
  - Check if config is nil, return error if so
  - Require either Name or Image to be set
  - If only Name is set, use it as the image name
  - If Image is set, use it (allows full registry paths)
- Apply defaults:
  - Dockerfile defaults to "Dockerfile"
  - Context defaults to "."
  - Tag defaults to "latest"
- Get logger from context (same pattern as nixpacks)
- Build Docker command:
  - Start with `docker build`
  - Add context path as argument
  - Add `-f` flag with Dockerfile path
  - Add `-t` flag with full image:tag
  - Add `--build-arg` for each BuildArgs entry
  - Add `--build-arg` for each Env entry (including Vault secrets!)
- Execute command:
  - Use `exec.CommandContext(ctx, "docker", args...)`
  - Capture stdout and stderr in buffers
  - Set working directory if context != "."
  - Log command being executed at debug level
  - Run command with cmd.Run()
  - Log stdout at debug level if present
  - Log stderr and return error if command fails
- Create and return artifact:
  - Generate unique artifact ID: `fmt.Sprintf("csdocker-%s-%d", imageName, time.Now().Unix())`
  - Set Image field to the built image name
  - Set Tag field to the tag used
  - Add labels: `{"builder": "csdocker"}`
  - Add metadata: builder type, context, docker command args
  - Set BuildTime to time.Now()
  - Log success message with image name

### 4. Create comprehensive unit tests
- Create `builtin/csdocker/builder_test.go`
- Test ConfigSet() with various scenarios:
  - Valid config with all fields (name, image, tag, dockerfile, context, build_args, env)
  - Valid config with only required fields
  - Config with Vault fields (backward compatibility)
  - Nil config
  - Typed config (*BuilderConfig)
  - Validate BuildArgs parsing
  - Validate Env parsing (important for Vault secret injection)
- Test Config() method returns correct config
- Test Build() validation errors:
  - Nil config should error
  - Missing both name and image should error
- Test Build() default handling:
  - Verify defaults are applied correctly
  - Don't actually run docker, just verify config
- Follow same test structure as nixpacks tests

### 5. Create integration test example
- Create test HCL configuration file: `examples/csdocker-example.hcl`
- Include example with Vault configuration
- Include example with standard Docker build
- Add comments explaining each field
- Show both simple and advanced usage patterns

### 6. Update documentation
- Update `docs/SECRETS.md` to include csdocker example
- Add section showing how csdocker uses Vault secrets
- Include example HCL configuration
- Explain that secrets are passed as build args

### 7. Run all validation commands
- Execute validation commands listed below
- Fix any issues that arise
- Ensure all tests pass
- Verify binary builds successfully
- Test with actual Vault integration (if available)

## Validation Commands
Execute every command to validate the chore is complete with zero regressions.

```bash
# 1. Build the binary
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go build -o bin/cs ./cmd/cloudstation

# 2. Run unit tests for csdocker plugin
go test ./builtin/csdocker/... -v

# 3. Run all builtin plugin tests
go test ./builtin/... -v

# 4. Run secret provider tests (ensure no regressions)
go test ./pkg/secrets/... -v

# 5. Run lifecycle tests (ensure no regressions)
go test ./internal/lifecycle/... -v

# 6. Run all tests
go test ./... -v

# 7. Format code
go fmt ./builtin/csdocker/...

# 8. Run linter
go vet ./builtin/csdocker/...

# 9. Test with example config (requires Docker installed)
./bin/cs --config examples/csdocker-example.hcl build --app docker-test

# 10. Test with Vault integration (if Vault credentials available)
./bin/cs --config vault-test.hcl --log-level debug build --app docker-app

# 11. Verify binary size and build
ls -lh bin/cs
file bin/cs
```

## Notes

### Key Implementation Details

1. **No Vault Code in Plugin**: The csdocker plugin should NOT contain any Vault-specific logic. The secret provider architecture at the lifecycle layer handles all Vault interaction.

2. **Environment Variables as Build Args**: Docker build args are how we pass secrets into the build. The pattern is:
   ```bash
   docker build --build-arg KEY=value --build-arg SECRET=from_vault ...
   ```

3. **Secret Flow**:
   ```
   Vault → Lifecycle (enrichment) → config["env"] map → ConfigSet() → Build() → docker --build-arg
   ```

4. **Image vs Name**:
   - `name` is simple: "myapp"
   - `image` is full path: "myregistry.azurecr.io/myapp"
   - If both provided, `image` takes precedence
   - This allows flexibility for different registry scenarios

5. **Error Handling**: Follow nixpacks pattern:
   - Clear error messages with context
   - Include stderr output in error messages
   - Log errors before returning

6. **Testing Without Docker**: Tests should validate config parsing and command building without actually running Docker (like nixpacks tests do)

7. **Backward Compatibility**: Keep deprecated Vault fields in BuilderConfig and ConfigSet() even though they're not used - this ensures existing HCL configs don't break

### Reference: Docker Build Command Structure
```bash
docker build \
  <context-path> \
  -f <dockerfile-path> \
  -t <image:tag> \
  --build-arg ENV_VAR=value \
  --build-arg SECRET=vault_value
```

### Example Usage in HCL
```hcl
app "web" {
  build {
    use = "csdocker"
    name = "my-web-app"
    tag = "v1.0.0"
    dockerfile = "Dockerfile"
    context = "."

    # Vault secrets (handled by lifecycle layer)
    vault_address = "https://vault.example.com"
    role_id = env("VAULT_ROLE_ID")
    secret_id = env("VAULT_SECRET_ID")
    secrets_path = "secret/data/app"

    # Static build args
    build_args = {
      NODE_ENV = "production"
    }

    # Env vars (secrets from Vault will be merged here)
    env = {
      PORT = "3000"
    }
  }
}
```

### Testing Strategy
- Unit tests verify config parsing and validation
- Integration tests verify actual Docker builds (if Docker available)
- Vault integration is tested via the existing vault-test.hcl configuration
- Follow test-driven development: write tests first, then implement
