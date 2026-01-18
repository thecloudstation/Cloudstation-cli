# Plan: Smart Builder Fallback System

## Task Description

Implement an intelligent builder fallback system that ensures builds "never fail" by automatically trying alternative builders when the primary builder fails. The system should:
- Try the user's specified builder first (if provided)
- Fall back to alternative builders in priority order
- Support both auto-detected and user-specified builder scenarios
- Provide clear logging of fallback attempts for user visibility

## Objective

When this plan is complete, the cloudstation-orchestrator will:
1. Never fail a deployment due to builder selection mismatch
2. Automatically try `csdocker` → `railpack` → `nixpacks` (or configured priority)
3. Log each builder attempt and fallback reason to NATS for user visibility
4. Respect user preferences while still providing fallback safety net

## Problem Statement

Currently, the build system selects a single builder based on Dockerfile presence:
- Dockerfile present → `csdocker`
- No Dockerfile → `railpack`

**Issues:**
1. If `railpack` fails on a project (e.g., unsupported language), the build fails completely
2. If user specifies wrong builder, build fails without trying alternatives
3. Projects with Dockerfile that could work with `railpack` fail if Dockerfile has issues

**User Impact:** Deployments fail unnecessarily when alternative builders could succeed.

## Solution Approach

Implement a **builder chain** system where:
1. `DetectBuilder()` returns an ordered list of builders to try
2. `HandleDeployRepository()` iterates through builders until one succeeds
3. User-specified builders become the first item in the chain (priority), not the only option
4. Each attempt is logged to NATS with clear status messages

```
┌─────────────────────────────────────────────────────────────────┐
│                 BUILD FLOW (With Fallback)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   User Choice: "railpack"    OR    Auto-detect (no choice)      │
│         │                              │                         │
│         ▼                              ▼                         │
│   ┌─────────────────┐      ┌─────────────────────────────┐      │
│   │ Builder Chain:  │      │ DetectBuilder() returns:    │      │
│   │ 1. railpack     │      │ - Dockerfile? [csdocker,    │      │
│   │ 2. csdocker     │      │   railpack, nixpacks]       │      │
│   │ 3. nixpacks     │      │ - No Dockerfile? [railpack, │      │
│   └────────┬────────┘      │   nixpacks]                 │      │
│            │               └─────────────┬───────────────┘      │
│            └──────────────┬──────────────┘                      │
│                           ▼                                      │
│            ┌──────────────────────────────┐                     │
│            │    Try Builder 1             │                     │
│            └──────────────┬───────────────┘                     │
│                    SUCCESS│FAIL                                  │
│               ┌───────────┴───────────┐                         │
│               ▼                       ▼                          │
│        ┌──────────┐         ┌──────────────────┐                │
│        │ Continue │         │ Log: "Builder X  │                │
│        │ Pipeline │         │ failed, trying Y"│                │
│        └──────────┘         └────────┬─────────┘                │
│                                      │                           │
│                                      ▼                           │
│                           ┌──────────────────────────────┐      │
│                           │    Try Builder 2             │      │
│                           └──────────────────────────────┘      │
│                                      │                           │
│                              (repeat until success               │
│                               or all builders exhausted)         │
└─────────────────────────────────────────────────────────────────┘
```

## Relevant Files

### Core Detection Logic
- `pkg/detect/detect.go` (Lines 8-107)
  - `DetectionResult` struct needs `Builders []string` field
  - `DetectBuilder()` returns ordered builder chain
  - `GetDefaultBuilder()` backward compat wrapper

### Dispatch Handler
- `internal/dispatch/handlers.go` (Lines 231-340)
  - Add retry loop around `executor.ExecuteBuild()`
  - Update NATS logging for each attempt
  - Handle builder switching between attempts

### Dispatch Types
- `internal/dispatch/types.go` (Lines 80-95)
  - `BuildOptions` struct - consider adding `FallbackBuilders []string`

### Lifecycle Executor
- `internal/lifecycle/executor.go` (Lines 82-120)
  - `ExecuteBuild()` stays unchanged (single attempt)
  - Handler manages retry, not executor

### HCL Generation
- `internal/hclgen/generator.go` (Lines 20-27, 103-155)
  - `GenerateConfig()` may need to be called per builder attempt
  - Builder-specific config generation

### Builder Plugins (Reference)
- `builtin/railpack/plugin.go` - Railpack builder
- `builtin/csdocker/plugin.go` - CSDocker builder
- `builtin/nixpacks/plugin.go` - Nixpacks builder

### Tests
- `pkg/detect/detect_test.go` - Add tests for builder chain
- `internal/dispatch/handlers_test.go` - Add fallback scenario tests
- `internal/config/zeroconfig_integration_test.go` - Integration tests

### New Files
- `pkg/detect/chain.go` - Builder chain utilities (optional, can be in detect.go)

## Implementation Phases

### Phase 1: Foundation - Extend Detection System
Modify `DetectionResult` to support ordered builder chains while maintaining backward compatibility.

### Phase 2: Core Implementation - Add Retry Logic to Handler
Implement the builder fallback loop in `HandleDeployRepository()` with proper logging and error aggregation.

### Phase 3: Integration & Polish
Add comprehensive tests, update documentation, and ensure NATS logging provides clear visibility into fallback behavior.

## Step by Step Tasks

IMPORTANT: Execute every step in order, top to bottom.

### 1. Extend DetectionResult Struct

- In `pkg/detect/detect.go`, modify `DetectionResult` struct (line 8-14):
  ```go
  type DetectionResult struct {
      Builder   string   // Primary builder (backward compat)
      Builders  []string // Ordered builder chain to try
      Reason    string
      Signals   []string
      HasDocker bool
  }
  ```
- Ensure `Builder` field always equals `Builders[0]` for backward compatibility

### 2. Update DetectBuilder Function

- In `pkg/detect/detect.go`, modify `DetectBuilder()` function (lines 20-56):
  ```go
  func DetectBuilder(rootDir string) *DetectionResult {
      result := &DetectionResult{
          Signals: []string{},
      }

      if hasDockerfileInDir(rootDir) {
          result.HasDocker = true
          result.Builders = []string{"csdocker", "railpack", "nixpacks"}
          result.Builder = "csdocker"
          result.Reason = "Dockerfile found - will try csdocker, fallback to railpack/nixpacks"
      } else {
          result.Builders = []string{"railpack", "nixpacks"}
          result.Builder = "railpack"
          result.Reason = "No Dockerfile - will try railpack, fallback to nixpacks"
      }

      result.Signals = detectProjectSignals(rootDir)
      return result
  }
  ```

### 3. Add GetBuilderChain Helper Function

- In `pkg/detect/detect.go`, add new function after `GetDefaultBuilder()`:
  ```go
  // GetBuilderChain returns the ordered list of builders to try
  // If userBuilder is specified, it becomes first in chain
  func GetBuilderChain(rootDir string, userBuilder string) []string {
      detection := DetectBuilder(rootDir)

      if userBuilder == "" {
          return detection.Builders
      }

      // User specified a builder - put it first, then add others
      chain := []string{userBuilder}
      for _, b := range detection.Builders {
          if b != userBuilder {
              chain = append(chain, b)
          }
      }
      return chain
  }
  ```

### 4. Update Detection Tests

- In `pkg/detect/detect_test.go`, add test cases for `Builders` field:
  ```go
  func TestDetectBuilder_BuilderChain(t *testing.T) {
      tests := []struct {
          name             string
          files            []string
          expectedBuilders []string
      }{
          {
              name:             "Dockerfile present - chain starts with csdocker",
              files:            []string{"Dockerfile"},
              expectedBuilders: []string{"csdocker", "railpack", "nixpacks"},
          },
          {
              name:             "No Dockerfile - chain starts with railpack",
              files:            []string{"package.json"},
              expectedBuilders: []string{"railpack", "nixpacks"},
          },
      }
      // ... implement test
  }

  func TestGetBuilderChain(t *testing.T) {
      // Test that user-specified builder comes first
      // Test that duplicate builders are not added
  }
  ```

### 5. Implement Fallback Loop in Handler

- In `internal/dispatch/handlers.go`, replace lines 231-340 with fallback logic:
  ```go
  // Get builder chain (user choice first, then auto-detected fallbacks)
  builderChain := detect.GetBuilderChain(workDir, params.Build.Builder)

  writeLog(stdoutWriter, fmt.Sprintf("Builder chain: %v\n", builderChain))

  var lastErr error
  var artifact *artifact.Artifact

  for attemptNum, builder := range builderChain {
      if attemptNum > 0 {
          writeLog(stdoutWriter, fmt.Sprintf(
              "⚠️  Builder '%s' failed, trying '%s' (attempt %d/%d)...\n",
              builderChain[attemptNum-1], builder, attemptNum+1, len(builderChain),
          ))
      }

      // Update params with current builder
      params.Build.Builder = builder

      // Regenerate HCL config for this builder
      hclParams := mapDeployRepositoryToHCLParams(params)
      hclConfig, err := hclgen.GenerateConfig(hclParams)
      if err != nil {
          lastErr = err
          continue // Try next builder
      }

      // Write and load config
      configPath, err := hclgen.WriteConfigFile(hclConfig, workDir)
      if err != nil {
          lastErr = err
          continue
      }

      cfg, err := config.LoadConfigFile(configPath)
      if err != nil {
          lastErr = err
          continue
      }

      // Create executor and attempt build
      executor := lifecycle.NewExecutor(cfg, logger)
      app := cfg.GetApp(params.JobID)

      // Set build phase on log writers
      setPhaseForWriters(ctx, "build")
      writeLog(stdoutWriter, fmt.Sprintf("Building with %s...\n", builder))

      artifact, err = executor.ExecuteBuild(buildCtx, app)
      if err == nil {
          // SUCCESS! Log and continue pipeline
          writeLog(stdoutWriter, fmt.Sprintf("✅ Build succeeded with %s\n", builder))
          break
      }

      // Build failed, log error and try next
      lastErr = err
      logger.Warn("Builder failed, trying next",
          "builder", builder,
          "error", err,
          "attempt", attemptNum+1,
          "remaining", len(builderChain)-attemptNum-1,
      )
  }

  // Check if all builders failed
  if artifact == nil {
      updateDeploymentStep(backendClient, params.DeploymentID, "repository",
          backend.StepBuild, backend.StatusFailed,
          fmt.Sprintf("All builders failed. Last error: %v", lastErr), logger)
      writeLog(stderrWriter, fmt.Sprintf("ERROR: All builders failed. Last error: %v\n", lastErr))
      publishFailure(natsClient, params, logger, lastErr)
      return fmt.Errorf("build failed with all builders: %w", lastErr)
  }
  ```

### 6. Extract Build Attempt Logic to Helper Function

- In `internal/dispatch/handlers.go`, create helper function for cleaner code:
  ```go
  // attemptBuild tries to build with a specific builder
  // Returns artifact on success, error on failure
  func attemptBuild(
      ctx context.Context,
      workDir string,
      params *DeployRepositoryParams,
      builder string,
      logger hclog.Logger,
  ) (*artifact.Artifact, error) {
      params.Build.Builder = builder

      hclParams := mapDeployRepositoryToHCLParams(*params)
      hclConfig, err := hclgen.GenerateConfig(hclParams)
      if err != nil {
          return nil, fmt.Errorf("failed to generate HCL for %s: %w", builder, err)
      }

      configPath, err := hclgen.WriteConfigFile(hclConfig, workDir)
      if err != nil {
          return nil, fmt.Errorf("failed to write config for %s: %w", builder, err)
      }

      cfg, err := config.LoadConfigFile(configPath)
      if err != nil {
          return nil, fmt.Errorf("failed to load config for %s: %w", builder, err)
      }

      executor := lifecycle.NewExecutor(cfg, logger)
      app := cfg.GetApp(params.JobID)
      if app == nil {
          return nil, fmt.Errorf("app not found for %s", builder)
      }

      return executor.ExecuteBuild(ctx, app)
  }
  ```

### 7. Update CLI Commands for Consistency

- In `cmd/cloudstation/commands.go`, update `buildCmd` (line 295) and `upCmd` (line 633):
  - Use `GetBuilderChain()` instead of `DetectBuilder().Builder`
  - Log the builder chain being used
  - For CLI local builds, consider simpler single-builder behavior (optional)

### 8. Add Handler Integration Tests

- In `internal/dispatch/handlers_test.go`, add fallback scenario tests:
  ```go
  func TestHandleDeployRepository_BuilderFallback(t *testing.T) {
      // Test: Primary builder fails, fallback succeeds
      // Test: All builders fail
      // Test: User-specified builder tried first
      // Test: Logging shows correct attempt numbers
  }
  ```

### 9. Update NATS Log Phase Tracking

- Ensure each builder attempt shows clearly in NATS logs:
  - Phase: `build` (unchanged)
  - Content includes: `[Attempt 1/3] Building with railpack...`
  - On failure: `[Attempt 1/3] railpack failed: <error>`
  - On fallback: `[Attempt 2/3] Trying csdocker...`

### 10. Validate and Test End-to-End

- Run unit tests: `go test ./pkg/detect/... ./internal/dispatch/...`
- Run integration tests: `go test ./internal/config/...`
- Manual test scenarios:
  - Project with Dockerfile → should use csdocker
  - Project without Dockerfile → should use railpack
  - railpack failure → should fallback to nixpacks
  - User specifies "nixpacks" → should try nixpacks first, then others

## Testing Strategy

### Unit Tests
1. **Detection Tests** (`pkg/detect/detect_test.go`)
   - `TestDetectBuilder_BuilderChain` - Verify correct chain for Dockerfile/non-Dockerfile
   - `TestGetBuilderChain_UserOverride` - Verify user choice becomes first in chain
   - `TestGetBuilderChain_NoDuplicates` - Verify no duplicate builders in chain

2. **Handler Tests** (`internal/dispatch/handlers_test.go`)
   - `TestHandleDeployRepository_FallbackOnFailure` - Mock first builder to fail
   - `TestHandleDeployRepository_AllBuildersFail` - All builders fail scenario
   - `TestHandleDeployRepository_FirstBuilderSucceeds` - No fallback needed

### Integration Tests
1. **Zero-Config Integration** (`internal/config/zeroconfig_integration_test.go`)
   - Add test for builder chain in generated config
   - Test that fallback works end-to-end

### Edge Cases
- Empty builder chain (should never happen, but handle gracefully)
- Single builder in chain (no fallback possible)
- Builder that doesn't exist in registry
- Context cancellation during fallback loop

## Acceptance Criteria

- [ ] `DetectionResult.Builders` contains ordered builder chain
- [ ] `DetectBuilder()` returns appropriate chain based on Dockerfile presence
- [ ] `GetBuilderChain()` puts user-specified builder first
- [ ] `HandleDeployRepository()` tries builders in order until success
- [ ] NATS logs clearly show each attempt: `[Attempt X/Y] Building with Z...`
- [ ] Failed attempts log: `Builder X failed: <error>, trying Y...`
- [ ] All existing tests pass (backward compatibility)
- [ ] New tests cover fallback scenarios
- [ ] CLI commands (`build`, `up`) work with new detection

## Validation Commands

Execute these commands to validate the task is complete:

```bash
# Run detection unit tests
cd /root/code/cs-monorepo/apps/cloudstation-orchestrator
go test -v ./pkg/detect/... -run TestDetectBuilder
go test -v ./pkg/detect/... -run TestGetBuilderChain

# Run handler tests
go test -v ./internal/dispatch/... -run TestHandleDeployRepository

# Run integration tests
go test -v ./internal/config/... -run TestZeroConfig

# Run all tests
go test ./...

# Build the binary to verify compilation
go build -o cs ./cmd/cloudstation/

# Manual validation (requires test project)
./cs build --path /path/to/test/project
```

## Notes

### Backward Compatibility
- `DetectionResult.Builder` field is preserved and always equals `Builders[0]`
- `GetDefaultBuilder()` continues to work unchanged
- Existing callers of `DetectBuilder()` that only use `.Builder` are unaffected

### Performance Considerations
- Each builder attempt regenerates HCL config (minimal overhead)
- Failed builder attempts are logged but don't persist state
- Workdir is reused between attempts (no re-clone)

### Future Enhancements
- Add `StrictBuilder bool` to `BuildOptions` to disable fallback
- Add retry metrics/telemetry for monitoring
- Consider caching builder success/failure patterns per project type
- Add configurable builder chain via environment variable

### Dependencies
- No new dependencies required
- Uses existing `hclgen`, `lifecycle`, and `config` packages
