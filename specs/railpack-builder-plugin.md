# Feature: Railpack Builder Plugin

## Feature Description
Add a new `railpack` builder plugin to CloudStation Orchestrator that provides zero-configuration container image building using Railway's Railpack tool. This plugin will follow the same architecture and patterns as the existing `nixpacks` and `csdocker` builders, with full support for Vault secret integration, port detection, and artifact metadata enrichment. Railpack is the successor to Nixpacks and leverages BuildKit for more efficient builds, making it an attractive option for modern containerized deployments.

## User Story
As a CloudStation user deploying applications
I want to use Railpack as a builder option
So that I can leverage Railway's latest zero-config builder with BuildKit optimizations and benefit from automatic language detection and containerization

## Problem Statement
CloudStation Orchestrator currently supports two main builders (nixpacks and csdocker), but does not support Railpack - Railway's next-generation builder that improves upon Nixpacks with:
- BuildKit-based architecture for better caching and performance
- More comprehensive language support and detection
- Improved build plans with detailed step information
- Better integration with modern container workflows

Users who want to leverage Railpack's features cannot use it within CloudStation Orchestrator without manually creating Docker builds, losing the benefits of zero-configuration deployment.

## Solution Statement
Implement a new `railpack` builder plugin that:
1. Follows the same structure and patterns as existing `nixpacks` and `csdocker` builders
2. Executes `railpack build` commands with appropriate configuration
3. Integrates with Vault for secret injection via environment variables
4. Implements port detection from built images (falling back to framework defaults)
5. Generates artifact metadata for the deployment pipeline
6. Supports all standard builder configuration options (name, tag, context, build args, env vars)
7. Provides comprehensive logging and error handling
8. Includes unit tests following existing test patterns

## Relevant Files
Use these files to implement the feature:

### Core Builder Files
- **`builtin/nixpacks/plugin.go`** - Template for builder structure, shows configuration parsing, build execution, port detection integration, and artifact creation patterns
- **`builtin/csdocker/plugin.go`** - Alternative builder example showing Vault integration and environment variable handling
- **`pkg/artifact/types.go`** - Artifact structure with ExposedPorts field and metadata
- **`pkg/portdetector/detector.go`** - Port detection logic that inspects Docker images
- **`internal/plugin/registry.go`** - Plugin registration system for adding new builders

### Integration Points
- **`internal/lifecycle/executor.go`** - Lifecycle executor that calls builder plugins and handles secret injection
- **`internal/lifecycle/secrets.go`** - Secret provider detection and environment variable enrichment
- **`internal/hclgen/port_defaults.go`** - Framework default ports (needs railpack case added)
- **`internal/hclgen/generator.go`** - HCL/vars generation that uses artifact metadata
- **`internal/dispatch/handlers.go`** - Handlers that map deployment types to builders

### Configuration Files
- **`README.md`** - Documentation listing available builders (needs railpack added)
- **`go.mod`** - Go module dependencies

### New Files
- **`builtin/railpack/plugin.go`** - New railpack builder implementation
- **`builtin/railpack/plugin_test.go`** - Unit tests for railpack builder
- **`builtin/railpack/builder_test.go`** - Additional builder-specific tests

## Implementation Plan

### Phase 1: Foundation
Create the basic railpack builder structure with configuration parsing and command execution. This phase establishes the core builder without Vault integration or port detection, ensuring the basic build flow works correctly.

### Phase 2: Core Implementation
Integrate Vault secret injection, port detection, and artifact metadata enrichment. This phase adds the advanced features that make the builder production-ready and consistent with other builders.

### Phase 3: Integration
Update framework defaults, documentation, and test the complete end-to-end deployment flow. This phase ensures the railpack builder integrates seamlessly with the existing CloudStation Orchestrator ecosystem.

## Step by Step Tasks

### 1. Create Railpack Builder Package Structure
- Create new directory `builtin/railpack/`
- Create `plugin.go` file with package declaration and imports
- Copy import structure from `nixpacks/plugin.go`:
  - bytes, context, fmt, os/exec, strings, time
  - github.com/hashicorp/go-hclog
  - internal/plugin, pkg/artifact, pkg/portdetector
- Define `Builder` struct with `config *BuilderConfig` field
- Define `BuilderConfig` struct with fields:
  - Name string (Docker image name, required)
  - Tag string (image tag, defaults to "latest")
  - Context string (build directory, defaults to ".")
  - BuildArgs map[string]string (additional railpack build arguments)
  - Env map[string]string (environment variables for build, includes Vault secrets)
  - VaultAddress, RoleID, SecretID, SecretsPath (deprecated, for backward compatibility)
- Add documentation comments for all structs and fields

### 2. Implement Build Method
- Implement `Build(ctx context.Context) (*artifact.Artifact, error)` method
- Add configuration validation:
  - Check if config is nil, return error if not set
  - Validate Name field is not empty (required)
- Get logger from context with fallback to default logger
- Set default values:
  - context defaults to "."
  - tag defaults to "latest"
- Construct full image name: `fmt.Sprintf("%s:%s", b.config.Name, tag)`
- Log build start with image name and context
- Build railpack command arguments:
  - Base command: `railpack build <context>`
  - Add `--name <name>` flag
  - Add `--tag <tag>` flag if not empty
  - Iterate BuildArgs map and add `--build-arg KEY=VALUE` for each
  - Iterate Env map and add `--env KEY=VALUE` for each (includes Vault secrets)
- Create `exec.CommandContext(ctx, "railpack", args...)` for cancellation support
- Set working directory if context is not "."
- Capture stdout and stderr using bytes.Buffer
- Execute command with `cmd.Run()`
- Log stdout if present (debug level)
- Handle errors:
  - Log stderr if present (error level)
  - Return formatted error with stderr included
- Log successful build completion

### 3. Implement Port Detection
- After successful build, call `portdetector.DetectPorts(imageName)`
- Handle port detection errors gracefully:
  - Log warning if detection fails
  - Continue with empty ports (don't fail the build)
- Log detected ports if successful
- Create artifact ID: `fmt.Sprintf("railpack-%s-%d", b.config.Name, time.Now().Unix())`
- Create artifact struct with:
  - ID, Image, Tag, ExposedPorts
  - Labels: map with "builder": "railpack"
  - Metadata: map with "builder", "context", "railpack_args"
  - BuildTime: time.Now()
- Add detected_ports to metadata if ports were detected
- Return artifact

### 4. Implement Config Methods
- Implement `Config() (interface{}, error)` method:
  - Return `b.config, nil`
- Implement `ConfigSet(config interface{}) error` method:
  - Handle nil config: create empty BuilderConfig
  - Handle map[string]interface{} config (most common):
    - Create helper function `getString(key string) string` to safely extract strings
    - Parse required field: Name
    - Parse optional fields: Tag, Context
    - Parse BuildArgs map[string]interface{} to map[string]string
    - Parse Env map[string]interface{} to map[string]string (includes Vault secrets)
    - Parse deprecated Vault fields for backward compatibility
  - Handle typed *BuilderConfig directly
  - Default to empty config if type doesn't match
  - Return nil on success

### 5. Implement Plugin Registration
- Add `init()` function at end of file
- Call `plugin.Register("railpack", &plugin.Plugin{...})`
- Set Builder field to `&Builder{config: &BuilderConfig{}}`
- Verify registration follows same pattern as nixpacks and csdocker

### 6. Update Framework Port Defaults
- Open `internal/hclgen/port_defaults.go`
- Add case for "railpack" in `GetFrameworkDefault()` switch statement:
  - Return DefaultWebPort (3000) - same as nixpacks since railpack is Node.js-focused
- Add comment explaining railpack uses same default as nixpacks

### 7. Update README Documentation
- Open `README.md`
- Add "railpack" to the Builtin Plugins list with description:
  - "**railpack** - Railway's next-gen zero-config builder with BuildKit"
- Add example usage in the Example Configuration section:
  ```hcl
  build {
    use "railpack" {
      vault_address = "https://vault.example.com:8200"
      role_id       = env("VAULT_ROLE_ID")
      secret_id     = env("VAULT_SECRET_ID")
      secrets_path  = "secret/data/app"
    }
  }
  ```

### 8. Create Unit Tests for ConfigSet
- Create `builtin/railpack/plugin_test.go`
- Import testing, reflect packages
- Test ConfigSet with nil config:
  - Verify config is initialized to empty BuilderConfig
- Test ConfigSet with map config containing all fields:
  - Create map with name, tag, context, build_args, env
  - Verify all fields parsed correctly
- Test ConfigSet with map config containing minimal fields:
  - Create map with only name field
  - Verify required field parsed, optional fields use defaults
- Test ConfigSet with typed *BuilderConfig:
  - Create typed config
  - Verify config set correctly

### 9. Create Unit Tests for Build Validation
- Add tests in `plugin_test.go` for Build method
- Test Build with nil config:
  - Verify returns error
- Test Build with missing name:
  - Create config with empty name
  - Verify returns error "railpack builder requires 'name' field to be set"
- Test Build with valid minimal config:
  - Mock railpack command execution (if possible)
  - Verify no errors

### 10. Create Integration Test for Port Detection
- Create `builtin/railpack/builder_test.go`
- Test that ExposedPorts field is populated after build
- Test that metadata includes detected_ports key
- Test graceful handling when port detection fails
- Verify artifact metadata contains railpack-specific information

### 11. Run All Validation Commands
- Execute `make test` to run all unit tests
- Execute `make build` to verify binary builds successfully
- Execute `make fmt` to format code
- Execute `make vet` to check for issues
- Verify no existing tests are broken
- Verify railpack builder tests pass

## Testing Strategy

### Unit Tests

**ConfigSet Tests (`plugin_test.go`):**
- `TestConfigSet_NilConfig` - Verify nil config creates empty BuilderConfig
- `TestConfigSet_MapConfigAllFields` - Verify full map config parses correctly
- `TestConfigSet_MapConfigMinimal` - Verify minimal config works with defaults
- `TestConfigSet_TypedConfig` - Verify typed BuilderConfig assignment works

**Build Validation Tests (`plugin_test.go`):**
- `TestBuild_NilConfig` - Verify error when config not set
- `TestBuild_MissingName` - Verify error when name field empty
- `TestBuild_ValidConfig` - Verify build executes with valid config (may need mocking)

**Port Detection Tests (`builder_test.go`):**
- `TestBuild_PopulatesExposedPorts` - Verify ExposedPorts field populated
- `TestBuild_PortDetectionFailure` - Verify graceful handling of port detection errors
- `TestBuild_MetadataIncludes DetectedPorts` - Verify metadata contains detected ports

### Integration Tests

**End-to-End Build Flow:**
- Create test Node.js application
- Configure railpack builder with test image name
- Execute build
- Verify artifact created with correct metadata
- Verify image exists in local Docker
- Verify detected ports match expectations

**Vault Integration:**
- Configure railpack with Vault settings
- Mock Vault secret retrieval
- Verify secrets injected into env map
- Verify secrets passed as environment variables to railpack build

**Port Detection Integration:**
- Build image with EXPOSE directive
- Verify port detected from image
- Build image without EXPOSE
- Verify defaults to 3000 (framework default)

### Edge Cases

1. **Railpack Not Installed** - Verify clear error message when railpack binary not found
2. **Invalid Image Name** - Verify error handling for malformed image names
3. **Build Context Not Exists** - Verify error when context directory doesn't exist
4. **Empty BuildArgs Map** - Verify no flags added when BuildArgs is empty
5. **Empty Env Map** - Verify build succeeds with no environment variables
6. **Build Timeout** - Verify cancellation via context works correctly
7. **Port Detection Image Not Found** - Verify graceful fallback to defaults
8. **Multiple Exposed Ports** - Verify first port used, others logged
9. **Concurrent Builds** - Verify multiple railpack builds can run simultaneously
10. **Large Build Output** - Verify stdout/stderr buffers handle large outputs

## Acceptance Criteria

1. ✅ Railpack builder plugin is registered and discoverable via plugin registry
2. ✅ Builder executes `railpack build` command with correct arguments
3. ✅ Configuration parsing handles both map[string]interface{} and typed configs
4. ✅ Name field is required and validated (returns error if missing)
5. ✅ Tag defaults to "latest" when not specified
6. ✅ Context defaults to "." when not specified
7. ✅ BuildArgs map correctly translates to `--build-arg KEY=VALUE` flags
8. ✅ Env map correctly translates to `--env KEY=VALUE` flags
9. ✅ Vault secrets are injected into Env map by lifecycle layer (verified via integration)
10. ✅ Port detection executes after successful build
11. ✅ Detected ports populate artifact.ExposedPorts field
12. ✅ Port detection failures are logged but don't fail the build
13. ✅ Framework default port (3000) is configured for railpack builder
14. ✅ Artifact metadata includes builder type, context, and railpack args
15. ✅ All existing tests pass (zero regressions)
16. ✅ New unit tests achieve >80% code coverage for railpack builder
17. ✅ README documentation includes railpack in builder list
18. ✅ Example configuration shows railpack usage with Vault
19. ✅ Build logs include clear information about build progress
20. ✅ Error messages are descriptive and actionable

## Validation Commands
Execute every command to validate the feature works correctly with zero regressions.

```bash
# Navigate to orchestrator directory
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator

# Verify railpack is installed (required dependency)
railpack --version

# Run all unit tests with verbose output
make test

# Run tests with coverage report
make coverage

# Verify coverage for new railpack package
go tool cover -func=coverage.txt | grep railpack

# Build the binary to ensure no compilation errors
make build

# Verify binary was created
ls -lh bin/cs

# Format code according to Go standards
make fmt

# Run vet to check for common issues
make vet

# Test railpack builder in isolation
go test -v ./builtin/railpack

# Test port detection integration
go test -v ./pkg/portdetector

# Test HCL generation with railpack builder type
go test -v ./internal/hclgen -run TestGenerateNetworking

# Manual end-to-end test: Create test app and build with railpack
mkdir -p /tmp/railpack-test-app
cd /tmp/railpack-test-app
echo '{"name":"test","scripts":{"start":"node index.js"}}' > package.json
echo 'console.log("hello from railpack")' > index.js

# Create cloudstation.hcl for railpack
cat > cloudstation.hcl << 'EOF'
project = "railpack-test"

app "test-app" {
  build {
    use "railpack" {
      name = "railpack-test"
      tag  = "latest"
    }
  }
}
EOF

# Execute build with railpack (from orchestrator directory)
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
./bin/cs build --app test-app --config /tmp/railpack-test-app/cloudstation.hcl

# Verify image was built
docker images | grep railpack-test

# Inspect image for port detection
docker inspect railpack-test:latest | jq '.[0].Config.ExposedPorts'

# Cleanup test
rm -rf /tmp/railpack-test-app
docker rmi railpack-test:latest
```

## Notes

### Dependencies
- **Railpack Installation Required**: Users must have `railpack` installed on their system. The builder will fail with a clear error if railpack is not found.
- **Railpack Version**: Tested with railpack v0.0.64. The builder should work with future versions, but version-specific features may require updates.
- **No New Go Dependencies**: The implementation uses only existing dependencies (exec, hclog, etc.). No `go get` required.

### Design Decisions
1. **Same Structure as Nixpacks**: Followed nixpacks builder pattern exactly to maintain consistency and make the codebase predictable
2. **Port Detection Strategy**: Uses same 3-tier fallback as other builders (user-specified → detected → framework default 3000)
3. **Vault Integration**: Secrets injected via Env map at lifecycle layer, not in builder itself (follows established pattern)
4. **Error Handling**: Port detection failures are warnings, not errors (build succeeds even if port detection fails)
5. **Default Port**: Set to 3000 (same as nixpacks) since railpack is primarily used for Node.js applications

### Future Enhancements
1. **Railpack Plan Integration**: Could run `railpack plan` before build to extract metadata (port info, detected languages, etc.)
2. **BuildKit Cache Configuration**: Expose railpack's BuildKit caching options for faster rebuilds
3. **Multi-Platform Builds**: Support railpack's platform targeting for ARM/AMD builds
4. **Custom Railpack Config**: Support `railpack.toml` or `.railpack` directory configurations
5. **Build Output Parsing**: Parse railpack's structured output to extract additional metadata
6. **Health Check Auto-Detection**: Use railpack's framework detection to configure health check paths

### Migration from Nixpacks
Users can easily migrate from nixpacks to railpack by changing one line in their `cloudstation.hcl`:
```hcl
# Before
build { use "nixpacks" { ... } }

# After
build { use "railpack" { ... } }
```

All other configuration (Vault settings, environment variables, etc.) remains identical.

### Backward Compatibility
- Deprecated Vault fields (vault_address, role_id, secret_id, secrets_path) are kept in BuilderConfig for backward compatibility
- These fields have no effect (secrets are injected via lifecycle layer) but won't break existing configurations
- Future versions may remove these deprecated fields with a deprecation warning cycle

### Performance Considerations
- Railpack uses BuildKit which provides better caching than Docker's default builder
- Build times may be faster than nixpacks for projects with good caching strategies
- Port detection adds <100ms overhead (same as other builders)
- No significant memory overhead compared to nixpacks or csdocker builders
