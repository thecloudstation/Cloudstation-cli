# Chore: Complete Noop Plugin Implementation with Registry and ReleaseManager

## Chore Description
The noop plugin currently only implements Builder and Platform components, but is missing Registry and ReleaseManager components. Based on the previous Waypoint-based noop plugin implementation (found in `cloudstation-server/noop-plugin/`), we need to add these missing components to provide a complete no-op plugin that can be used for testing the full lifecycle: Build → Registry → Deploy → Release.

The noop plugin should provide stub implementations for all plugin component interfaces to allow testing the complete deployment pipeline without actually executing any real operations. This is essential for:
- Testing the orchestrator's lifecycle execution flow
- Validating configuration parsing for all component types
- Debugging integration issues without side effects
- Providing a reference implementation for plugin developers

## Relevant Files
Use these files to resolve the chore:

**Core Implementation:**
- `builtin/noop/builder.go` - Main noop plugin file that needs Registry and ReleaseManager additions
  - Currently implements: Builder (Build, Config, ConfigSet) and Platform (Deploy, Destroy, Status)
  - Needs: Registry struct with Push/Pull/Config/ConfigSet methods
  - Needs: ReleaseManager struct with Release/Config/ConfigSet methods
  - Needs: Updated init() function to register new components

**Interface Definitions:**
- `pkg/component/interfaces.go` - Defines the Registry and ReleaseManager interfaces
  - Registry interface: Push(), Pull(), Config(), ConfigSet()
  - ReleaseManager interface: Release(), Config(), ConfigSet()
  - Used to ensure correct method signatures

**Type Definitions:**
- `pkg/artifact/types.go` - Contains Artifact and RegistryRef types
  - RegistryRef struct used by Registry.Push() return value
  - Artifact struct used by Registry.Pull() return value

**Reference Implementations:**
- `builtin/docker/plugin.go` - Shows existing Registry implementation pattern
  - Registry struct with config
  - Push() method returning RegistryRef
  - Pull() method (stub)
  - Config management methods

**Previous Implementation Reference:**
- `cloudstation-server/noop-plugin/release/release.go` - Shows Waypoint-based Release implementation
  - ReleaseManager struct pattern
  - Release() method that returns success
  - Config management

### New Files
- `builtin/noop/noop_test.go` - Unit tests for Registry and ReleaseManager components

## Step by Step Tasks

### 1. Add Registry Component to Noop Plugin
- Add `Registry` struct to `builtin/noop/builder.go`
- Add `RegistryConfig` struct with no fields (noop has no config)
- Implement `Push()` method that returns a fake RegistryRef with:
  - Registry: "noop-registry"
  - Repository: "noop"
  - Tag: "latest"
  - FullImage: "noop-registry/noop:latest"
  - PushedAt: time.Now()
- Implement `Pull()` method that returns nil artifact and error "noop registry: pull not implemented"
- Implement `Config()` method that returns the registry config
- Implement `ConfigSet()` method that:
  - Handles nil config by creating empty RegistryConfig
  - Handles map[string]interface{} config by creating empty RegistryConfig (noop ignores config)
  - Handles typed *RegistryConfig by setting it directly
  - Returns nil error

### 2. Add ReleaseManager Component to Noop Plugin
- Add `ReleaseManager` struct to `builtin/noop/builder.go`
- Add `ReleaseConfig` struct with optional fields:
  - Message string (optional message to include in logs)
- Implement `Release()` method that:
  - Accepts ctx context.Context and deployment *deployment.Deployment parameters
  - Logs "noop release: release completed" if logger available
  - Returns nil (success)
- Implement `Config()` method that returns the release manager config
- Implement `ConfigSet()` method that:
  - Handles nil config by creating empty ReleaseConfig
  - Handles map[string]interface{} config:
    - Parses optional "message" field from config map
  - Handles typed *ReleaseConfig by setting it directly
  - Returns nil error

### 3. Update Plugin Registration
- Locate the `init()` function in `builtin/noop/builder.go`
- Update the `plugin.Register()` call to include:
  - Registry: &Registry{config: &RegistryConfig{}}
  - ReleaseManager: &ReleaseManager{config: &ReleaseConfig{}}
- Ensure all components (Builder, Registry, Platform, ReleaseManager) are registered

### 4. Add Constructor Functions for Components
- Add `NewRegistry()` function that returns a new Registry with empty config
- Add `NewReleaseManager()` function that returns a new ReleaseManager with empty config
- Update `init()` to use these constructors for consistency with Builder pattern

### 5. Create Comprehensive Unit Tests
- Create `builtin/noop/noop_test.go` file
- Test Registry component:
  - Test Registry.ConfigSet() with nil config
  - Test Registry.ConfigSet() with map config
  - Test Registry.ConfigSet() with typed config
  - Test Registry.Config() returns correct config
  - Test Registry.Push() returns valid RegistryRef
  - Test Registry.Pull() returns error
- Test ReleaseManager component:
  - Test ReleaseManager.ConfigSet() with nil config
  - Test ReleaseManager.ConfigSet() with map config (with message)
  - Test ReleaseManager.ConfigSet() with typed config
  - Test ReleaseManager.Config() returns correct config
  - Test ReleaseManager.Release() succeeds
- Test existing Builder component (if not already tested):
  - Test Builder.ConfigSet() variations
  - Test Builder.Build() returns valid artifact
- Test existing Platform component (if not already tested):
  - Test Platform.Deploy() returns valid deployment
  - Test Platform.Destroy() succeeds
  - Test Platform.Status() returns valid status

### 6. Add Example Configuration
- Update `examples/` directory to include noop plugin example showing full lifecycle
- Create or update example HCL file demonstrating:
  - build { use = "noop" }
  - registry { use = "noop" }
  - deploy { use = "noop" }
  - release { use = "noop" }
- Include optional message configuration for components

### 7. Update Documentation
- Update `docs/PLUGINS.md` to include Registry and ReleaseManager in noop plugin description
- Add note that noop plugin is now complete for full lifecycle testing
- Include example usage of noop plugin for all phases

### 8. Run All Validation Commands
- Execute all validation commands listed below
- Fix any issues that arise
- Ensure all tests pass
- Verify binary builds successfully

## Validation Commands
Execute every command to validate the chore is complete with zero regressions.

```bash
# 1. Build the binary
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go build -o bin/cs ./cmd/cloudstation

# 2. Run unit tests for noop plugin
go test ./builtin/noop/... -v

# 3. Run all builtin plugin tests (ensure no regressions)
go test ./builtin/... -v

# 4. Run all component interface tests
go test ./pkg/component/... -v

# 5. Run plugin registry tests (ensure registration works)
go test ./internal/plugin/... -v

# 6. Run all tests
go test ./... -v

# 7. Format code
go fmt ./builtin/noop/...

# 8. Run go vet
go vet ./builtin/noop/...

# 9. Verify binary builds and runs
./bin/cs --version

# 10. Test with example config (if created)
# ./bin/cs --config examples/noop-example.hcl build --app test
```

## Notes

### Key Implementation Details

1. **No Actual Operations**: All noop components should return success immediately without performing any real operations (no Docker, no Nomad, no registry pushes, etc.)

2. **Consistent Pattern**: Follow the same pattern established by Builder and Platform:
   - Struct with config field
   - Separate config struct
   - Config(), ConfigSet() methods
   - Main operation method (Push/Pull for Registry, Release for ReleaseManager)

3. **Return Valid Data**: Even though it's a no-op, return valid data structures:
   - Registry.Push() returns valid RegistryRef (not nil)
   - ReleaseManager.Release() returns nil error (success)
   - Registry.Pull() returns error (not implemented)

4. **Config Flexibility**: Support three config types like other components:
   - nil config → create empty config
   - map[string]interface{} → parse fields
   - typed config → use directly

5. **Testing Focus**: Tests should verify:
   - Config parsing works correctly
   - Methods return expected data types
   - No panics or nil pointer errors
   - Backward compatibility maintained

### Reference to Previous Implementation

The previous Waypoint-based noop plugin structure:
```
noop-plugin/
├── main.go              # Registered all 3 components
├── builder/builder.go   # Builder implementation
├── platform/
│   ├── deploy.go        # Deploy implementation
│   └── destroy.go       # Destroy implementation
└── release/
    ├── release.go       # Release implementation ✅
    └── destroy.go       # Destroy implementation
```

Our new implementation consolidates everything into a single file for simplicity while maintaining the same functionality.

### Testing the Full Lifecycle

After implementation, you can test the full lifecycle with:

```hcl
project = "test"

app "noop-test" {
  build {
    use = "noop"
    message = "Building with noop"
  }

  registry {
    use = "noop"
  }

  deploy {
    use = "noop"
  }

  release {
    use = "noop"
    message = "Releasing with noop"
  }
}
```

Then run: `cs up --app noop-test` to execute the full Build → Registry → Deploy → Release pipeline.
