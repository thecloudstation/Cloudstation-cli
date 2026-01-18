# Feature: Complete Nomad Pack Platform Plugin Implementation

## Feature Description
Implement a complete, production-ready Nomad Pack platform plugin for CloudStation Orchestrator that provides full lifecycle management (deploy, destroy, status) of Nomad Pack deployments. This plugin will replace the current stub implementation with actual `nomad-pack` CLI integration, following the same patterns established in the old Waypoint-based implementation but adapted to the new Waypoint-free builtin plugin architecture.

The plugin will execute real `nomad-pack` commands to:
- Add and configure Nomad Pack registries with authentication
- Deploy packs with variables and configuration
- Check deployment status
- Destroy pack deployments cleanly

This is a critical plugin for the CloudStation Orchestrator as it enables deploying applications to Nomad clusters using Nomad Pack's templated job specifications.

## User Story
As a DevOps engineer
I want to deploy applications to Nomad using Nomad Pack templates via CloudStation Orchestrator
So that I can leverage existing pack configurations and maintain consistent deployment patterns without depending on the unmaintained Waypoint fork

## Problem Statement
The current nomad-pack plugin in `builtin/nomadpack/plugin.go` is a stub implementation with TODO comments. It returns fake deployment data without actually executing any `nomad-pack` commands. The old Waypoint-based implementation in `cloudstation-server/base-plugins/platform/` contains a complete working implementation that needs to be migrated to the new Waypoint-free architecture.

**Key Missing Features:**
1. No actual `nomad-pack` CLI command execution
2. No registry management (adding registries with authentication tokens)
3. No variable handling (variables and variable files)
4. No real deployment/destruction logic
5. No status checking from actual Nomad Pack deployments
6. No environment variable setup (NOMAD_TOKEN, NOMAD_ADDR)
7. No proper error handling for CLI failures

## Solution Statement
Migrate the complete Nomad Pack functionality from the old Waypoint plugin to the new builtin plugin architecture by:

1. Implementing real `nomad-pack` CLI command execution using Go's `exec.Command`
2. Adding registry management methods that configure pack registries with authentication
3. Implementing Deploy, Destroy, and Status methods that execute actual `nomad-pack` commands
4. Adding variable and variable file handling
5. Implementing proper error handling and logging using hclog
6. Following the established patterns from csdocker and nixpacks plugins
7. Adding comprehensive unit tests to ensure correctness

The implementation will be Waypoint-free, using only the standard Go library and HashiCorp's hclog for logging, with no dependency on Waypoint SDK.

## Relevant Files
Use these files to implement the feature:

**Core Implementation:**
- `builtin/nomadpack/plugin.go` - Main plugin file that needs complete rewrite
  - Currently contains only stub Platform implementation
  - Needs: Complete Deploy(), Destroy(), Status() implementations
  - Needs: Registry management helper methods
  - Needs: Variable handling helper methods
  - Needs: Expanded PlatformConfig with all necessary fields

**Reference Implementations:**
- `cloudstation-server/base-plugins/platform/deploy.go` - Old Waypoint implementation of deploy
  - Shows deployPack() logic with nomad-pack run
  - Shows addRegistry() for registry configuration
  - Shows setVarArgs() for variable handling
  - Shows packStatus() for status checking
  - Shows generation() for deployment ID generation

- `cloudstation-server/base-plugins/platform/destroy.go` - Old Waypoint implementation of destroy
  - Shows destroyPack() logic with nomad-pack destroy
  - Shows status checking before destruction
  - Shows proper error handling

**Pattern References:**
- `builtin/csdocker/plugin.go` - Shows command execution pattern
  - ConfigSet handling with map[string]interface{}
  - exec.CommandContext usage
  - Error handling and logging patterns
  - Environment variable setup

- `builtin/nixpacks/plugin.go` - Shows similar CLI tool integration
  - Command argument building
  - Context cancellation support
  - Output capture and error reporting

**Interface Definitions:**
- `pkg/component/interfaces.go` - Platform interface definition
  - Deploy(ctx, artifact) (*deployment.Deployment, error)
  - Destroy(ctx, deploymentID) error
  - Status(ctx, deploymentID) (*deployment.DeploymentStatus, error)
  - Config() and ConfigSet() methods

**Type Definitions:**
- `pkg/deployment/types.go` - Deployment and DeploymentStatus types
  - Deployment struct with ID, Name, Platform, Status fields
  - DeploymentStatus with State and Health enums
  - Used for return values

**Documentation:**
- `docs/PLUGINS.md` - Plugin documentation that needs updating
  - Currently shows nomadpack in minimal form
  - Needs: Complete configuration options documented
  - Needs: Example configurations with all features

### New Files
- `builtin/nomadpack/nomadpack_test.go` - Comprehensive unit tests for all methods
- `examples/nomadpack-example.hcl` - Complete example configuration showing all features

## Implementation Plan

### Phase 1: Foundation
Set up the complete configuration structure and helper methods needed for Nomad Pack operations. This includes expanding the config struct to match the old implementation's capabilities and creating reusable helper functions for registry management and variable handling.

### Phase 2: Core Implementation
Implement the three main Platform interface methods (Deploy, Destroy, Status) with actual `nomad-pack` CLI command execution. Each method will construct appropriate command arguments, execute the command, and parse the output to return proper deployment data.

### Phase 3: Integration
Add comprehensive tests, example configuration, update documentation, and validate the complete implementation works end-to-end with zero regressions.

## Step by Step Tasks

### 1. Expand PlatformConfig Structure
- Add all configuration fields from old implementation to PlatformConfig:
  - `RegistryName` string - Name for the Nomad Pack registry
  - `RegistrySource` string - Git URL for the pack registry
  - `RegistryRef` string (optional) - Specific git ref/tag/branch
  - `RegistryTarget` string (optional) - Specific pack within registry
  - `RegistryToken` string (optional) - Personal access token for private registries
  - `Variables` map[string]string (optional) - Variable overrides for pack
  - `VariableFiles` []string (optional) - Paths to variable files
  - Keep existing: `DeploymentName`, `Pack`, `NomadAddr`, `NomadToken`, `Registry` (deprecated field)
- Add struct tags for proper HCL parsing
- Document each field with clear comments

### 2. Implement ConfigSet Method
- Handle nil config by creating empty PlatformConfig
- Handle map[string]interface{} config by parsing all fields:
  - Parse string fields: deployment_name, pack, nomad_addr, nomad_token, registry_name, registry_source, registry_ref, registry_target, registry_token
  - Parse map fields: variables (convert to map[string]string)
  - Parse slice fields: variable_files (convert to []string)
- Handle typed *PlatformConfig by setting directly
- Return nil error on success
- Follow the pattern from csdocker and nixpacks plugins

### 3. Implement addRegistry Helper Method
- Create private method: `addRegistry(ctx context.Context) (string, error)`
- Build `nomad-pack registry add` command with arguments:
  - Registry name from config
  - Registry URL with embedded token: `https://{token}@{source}`
  - Optional --target flag if RegistryTarget is set
  - Optional --ref flag if RegistryRef is set
- Execute command using exec.CommandContext
- Capture output and check for errors
- Return registry ref argument for reuse in other commands
- Log operation with hclog

### 4. Implement setVarArgs Helper Method
- Create private method: `setVarArgs(args []string) []string`
- Iterate over Variables map and append `--var=key=value` for each
- Iterate over VariableFiles slice and append `--var-file=path` for each
- Return modified args slice
- Follow exact pattern from old implementation

### 5. Implement Deploy Method
- Validate configuration (check required fields)
- Get logger from context
- Call addRegistry to ensure registry is configured
- Build `nomad-pack run` command arguments:
  - Pack name from config
  - --name flag with DeploymentName
  - --registry flag with RegistryName
  - --ref flag if returned from addRegistry
  - Add variables via setVarArgs helper
- Set environment variables:
  - NOMAD_TOKEN from config
  - NOMAD_ADDR from config
- Execute command using exec.CommandContext
- Capture stdout and stderr
- Parse output for deployment information
- Return Deployment struct with:
  - ID: generated from pack name and deployment name
  - Name: deployment name from config
  - Platform: "nomad"
  - ArtifactID: from input artifact
  - Status: State=StateRunning, Health=HealthHealthy
  - Metadata: pack name, nomad addr, registry name
  - DeployedAt: time.Now()
- Log success/failure with hclog
- Return proper errors with context

### 6. Implement Destroy Method
- Get logger from context
- Call addRegistry to ensure registry is configured
- Check if deployment exists by running `nomad-pack status`:
  - Build status command with pack, registry, deployment name
  - Execute and parse output
  - If no deployment found (output has <=3 fields), log and return nil (nothing to destroy)
- Build `nomad-pack destroy` command arguments:
  - Pack name from config
  - --name flag with DeploymentName
  - --registry flag with RegistryName
  - --ref flag if returned from addRegistry
  - Add variables via setVarArgs helper
- Set environment variables (NOMAD_TOKEN, NOMAD_ADDR)
- Execute command using exec.CommandContext
- Capture output and log
- Return error if command fails
- Log successful destruction

### 7. Implement Status Method
- Get logger from context
- Call addRegistry to ensure registry is configured
- Build `nomad-pack status` command arguments:
  - Pack name
  - --registry flag
  - --name flag
  - --ref flag if applicable
- Set environment variables (NOMAD_TOKEN, NOMAD_ADDR)
- Execute command and capture output
- Parse output to extract status information:
  - Split output by newlines
  - Parse line 3 (contains pack info)
  - Split by "|" delimiter
  - Extract: PackName, RegistryName, DeploymentName, JobName, Status
- Map status string to DeploymentStatus:
  - "running" → State=StateRunning, Health=HealthHealthy
  - "pending" → State=StatePending, Health=HealthUnknown
  - Other → State=StateUnknown, Health=HealthUnknown
- Return DeploymentStatus struct
- Handle errors appropriately

### 8. Update Plugin Registration
- Locate init() function in plugin.go
- Ensure NewPlatform() constructor exists that returns &Platform{config: &PlatformConfig{}}
- Update plugin.Register call if needed
- Ensure plugin is imported in cmd/cloudstation/main.go

### 9. Create Comprehensive Unit Tests
- Create `builtin/nomadpack/nomadpack_test.go`
- Test ConfigSet method:
  - Test with nil config
  - Test with map config containing all fields
  - Test with typed *PlatformConfig
  - Test variables and variable_files parsing
- Test Config method returns correct config
- Test helper methods:
  - Test setVarArgs with variables and files
  - Mock addRegistry behavior (if possible without actual nomad-pack)
- Test Deploy method:
  - Test with valid configuration
  - Test validation errors (missing required fields)
  - Test command construction (verify args are correct)
- Test Destroy method:
  - Test successful destruction
  - Test when no deployment exists
- Test Status method:
  - Test status parsing for different states
  - Test error handling
- Use table-driven tests where appropriate
- Follow test patterns from csdocker_test.go and nixpacks_test.go

### 10. Create Example Configuration File
- Create `examples/nomadpack-example.hcl`
- Show complete deployment configuration with:
  - Project and app definition
  - Build phase (using another builder)
  - Deploy phase using nomadpack with:
    - All configuration options documented
    - Example variables
    - Example variable files
    - Registry configuration with token
    - Nomad authentication
- Add comments explaining each option
- Show minimal and full configurations
- Add usage instructions

### 11. Update Documentation
- Update `docs/PLUGINS.md` nomadpack section
- Change from minimal stub description to complete feature list
- Document all configuration options:
  - Required: deployment_name, pack, registry_name, registry_source
  - Optional: registry_ref, registry_target, registry_token, nomad_token, nomad_addr, variables, variable_files
- Add complete example configurations
- Document authentication requirements
- Add notes about nomad-pack CLI prerequisites
- Include troubleshooting section

### 12. Run All Validation Commands
- Execute every validation command listed below
- Fix any issues that arise
- Ensure all tests pass
- Verify code formatting and linting
- Validate example configuration is valid HCL

## Testing Strategy

### Unit Tests
- **ConfigSet Tests**: Verify all configuration parsing paths (nil, map, typed)
- **Helper Method Tests**: Test setVarArgs and addRegistry logic
- **Deploy Tests**: Verify command construction and error handling
- **Destroy Tests**: Verify status check and destroy command logic
- **Status Tests**: Verify status parsing and state mapping
- **Integration Tests**: Test complete deploy → status → destroy flow

### Integration Tests
Since this requires actual `nomad-pack` CLI and a running Nomad cluster, integration tests should:
- Document prerequisites (nomad-pack installed, Nomad cluster available)
- Provide optional integration test that can be skipped if dependencies not available
- Test against a real pack registry if possible
- Verify complete lifecycle works end-to-end

### Edge Cases
- Missing nomad-pack binary (command not found)
- Invalid registry credentials
- Pack doesn't exist in registry
- Nomad cluster unreachable
- Deployment already exists (idempotency)
- Destroying non-existent deployment
- Malformed nomad-pack output
- Context cancellation during long-running operations
- Empty or missing environment variables
- Invalid variable file paths

## Acceptance Criteria
- [ ] All Platform interface methods are fully implemented (Deploy, Destroy, Status)
- [ ] Configuration supports all fields from old Waypoint implementation
- [ ] Registry management works with authentication tokens
- [ ] Variables and variable files are properly passed to nomad-pack
- [ ] Environment variables (NOMAD_TOKEN, NOMAD_ADDR) are set correctly
- [ ] All unit tests pass with >80% code coverage
- [ ] Example configuration file works end-to-end (manual verification)
- [ ] Documentation is complete and accurate
- [ ] Code follows patterns from csdocker and nixpacks plugins
- [ ] No Waypoint dependencies (Waypoint-free implementation)
- [ ] Proper error handling and logging throughout
- [ ] Binary builds successfully with no errors
- [ ] All existing tests continue to pass (zero regressions)

## Validation Commands
Execute every command to validate the feature works correctly with zero regressions.

```bash
# 1. Build the binary
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go build -o bin/cs ./cmd/cloudstation

# 2. Run unit tests for nomadpack plugin
go test ./builtin/nomadpack/... -v

# 3. Run all builtin plugin tests (ensure no regressions)
go test ./builtin/... -v

# 4. Run all component interface tests
go test ./pkg/component/... -v

# 5. Run all deployment tests
go test ./pkg/deployment/... -v

# 6. Run all tests with coverage
go test ./... -v -cover

# 7. Format code
go fmt ./builtin/nomadpack/...

# 8. Run go vet
go vet ./builtin/nomadpack/...

# 9. Verify binary builds and runs
./bin/cs --version

# 10. Validate example HCL syntax
./bin/cs init --help  # Verify init command works

# 11. Test configuration parsing (if nomad-pack available)
# ./bin/cs --config examples/nomadpack-example.hcl deploy --app test-app

# 12. Run full test suite one more time
go test ./... -v
```

## Notes

### Implementation Notes

1. **No Waypoint Dependencies**: Unlike the old implementation, this version must NOT use any Waypoint SDK components. Use only:
   - Standard Go library (`os/exec`, `context`, `strings`, `fmt`)
   - HashiCorp's hclog for logging
   - Existing CloudStation types from `pkg/`

2. **Command Execution Pattern**: Follow the exact pattern from csdocker and nixpacks:
   ```go
   cmd := exec.CommandContext(ctx, "nomad-pack", args...)
   cmd.Env = append(os.Environ(), "NOMAD_TOKEN="+token, "NOMAD_ADDR="+addr)
   output, err := cmd.CombinedOutput()
   ```

3. **Error Handling**: Capture both stdout and stderr, log the output, and return descriptive errors that include the command output.

4. **Registry Token Security**: The registry token is embedded in the URL as `https://token@github.com/org/repo`. This is how nomad-pack expects authentication. Never log the full URL with token.

5. **Status Parsing**: The old implementation parses line 3 of `nomad-pack status` output. This is brittle but matches nomad-pack's output format. Add error handling for parsing failures.

6. **Idempotency**: nomad-pack commands should be idempotent where possible. The destroy method checks if deployment exists before attempting destruction.

7. **Context Support**: All methods accept context.Context and should respect cancellation via exec.CommandContext.

### Future Considerations

1. **Better Status Parsing**: Consider using nomad-pack's JSON output format if available instead of parsing table output.

2. **Registry Caching**: The addRegistry method is called for every operation. Consider caching registry addition to avoid redundant calls.

3. **Variable Validation**: Add validation for variable file paths before passing to nomad-pack.

4. **Deployment State Tracking**: Consider storing pack deployment state for better status reporting.

5. **Multiple Pack Support**: Allow deploying multiple packs in a single configuration.

6. **Pack Upgrades**: Support upgrading existing pack deployments with new variables or pack versions.

### Prerequisites

- **nomad-pack CLI**: Must be installed on the system running cloudstation-orchestrator
  - Installation: https://github.com/hashicorp/nomad-pack
  - macOS: `brew tap hashicorp/tap && brew install nomad-pack`
  - Linux: Download from GitHub releases

- **Nomad Cluster**: Must have access to a running Nomad cluster
  - NOMAD_ADDR should point to Nomad API
  - NOMAD_TOKEN should have sufficient permissions

- **Pack Registry**: Must have access to a pack registry (GitHub, etc.)
  - Public registries work without tokens
  - Private registries require RegistryToken
