# Plan: Zero-Config Build UX

## Task Description
Implement zero-config support for the `cs build` command so users can run builds without creating a `cloudstation.hcl` file first. The CLI should auto-detect project type, select the appropriate builder (railpack/csdocker), and generate an in-memory configuration automatically.

## Objective
When a user runs `cs build` in a directory without a `cloudstation.hcl` file:
1. Auto-detect project name from git remote or directory name
2. Auto-detect builder type (railpack for zero-config, csdocker if Dockerfile present)
3. Generate a synthetic config in-memory
4. Execute the build successfully
5. Provide clear feedback about what was auto-detected

## Problem Statement
Currently, `cs build` fails immediately if `cloudstation.hcl` doesn't exist:
```
Error: failed to load configuration: configuration file not found: /path/to/cloudstation.hcl
```

This creates friction for new users who just want to quickly build their project. The codebase already has all the detection capabilities in `pkg/detect/` and `internal/config/detector.go`, but they're not used by the build command.

## Solution Approach
Create a fallback path in `buildCommand()` that generates a synthetic `*config.Config` when no config file exists, using the existing detection modules. This preserves backward compatibility (existing configs still work) while enabling zero-config for new users.

## Relevant Files

### Files to Modify

1. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/cmd/cloudstation/commands.go`**
   - Lines 234-275: `buildCommand()` function
   - Add zero-config fallback when `LoadConfigFile()` fails with "not found"

2. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/internal/config/parser.go`**
   - Lines 192-204: Add new `GenerateDefaultConfig()` function
   - Bridge detection modules to config generation

### Files to Reference (Read-Only)

3. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/internal/config/types.go`**
   - Lines 4-70: Config, AppConfig, PluginConfig struct definitions
   - Use these types for synthetic config creation

4. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/internal/config/detector.go`**
   - Lines 11-21: `DetectProjectName()` function
   - Already implemented, just need to call it

5. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/pkg/detect/detect.go`**
   - Lines 20-56: `DetectBuilder()` function
   - Returns railpack or csdocker based on Dockerfile presence

6. **`/root/code/cs-monorepo/apps/cloudstation-orchestrator/internal/lifecycle/executor.go`**
   - Lines 178-194: `BuildOnly()` method
   - Verify it works with synthetic config

## Implementation Phases

### Phase 1: Foundation
- Add `GenerateDefaultConfig()` function to config package
- Import `pkg/detect` into config package
- Write unit tests for the new function

### Phase 2: Core Implementation
- Modify `buildCommand()` to catch config-not-found errors
- Call `GenerateDefaultConfig()` as fallback
- Add user feedback for zero-config mode

### Phase 3: Integration & Polish
- Test with various project types (Node.js, Go, Python, Rust)
- Test with and without Dockerfile
- Ensure backward compatibility with existing configs

## Step by Step Tasks

### 1. Add GenerateDefaultConfig Function
- Open `/root/code/cs-monorepo/apps/cloudstation-orchestrator/internal/config/parser.go`
- Add import for `"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"`
- Add new function after `LoadConfigFile()` (after line 204):

```go
// GenerateDefaultConfig creates a default configuration for zero-config builds
// when no cloudstation.hcl file exists. It uses auto-detection for project name
// and builder type.
func GenerateDefaultConfig(rootDir string) (*Config, error) {
    // Detect project name from git remote or directory
    projectName := DetectProjectName()
    if projectName == "" {
        projectName = "my-app"
    }

    // Detect builder type (railpack or csdocker)
    detection := detect.DetectBuilder(rootDir)

    // Create synthetic config
    cfg := &Config{
        Project: projectName,
        Apps: []*AppConfig{
            {
                Name: projectName,
                Build: &PluginConfig{
                    Use: detection.Builder,
                    Config: map[string]interface{}{
                        "name":    projectName,
                        "tag":     "latest",
                        "context": ".",
                    },
                },
                // Deploy block is optional for build-only operations
                Deploy: &PluginConfig{
                    Use: "nomad-pack",
                    Config: map[string]interface{}{
                        "pack": "cloudstation",
                    },
                },
            },
        },
    }

    return cfg, nil
}
```

### 2. Modify buildCommand to Support Zero-Config
- Open `/root/code/cs-monorepo/apps/cloudstation-orchestrator/cmd/cloudstation/commands.go`
- Modify the `buildCommand()` Action function (lines 234-275)
- Replace the config loading logic with zero-config fallback:

```go
Action: func(c *cli.Context) error {
    appName := c.String("app")
    apiURL := c.String("api-url")
    logger := hclog.Default()

    // Remote build
    if c.Bool("remote") {
        return executeRemoteBuild(c, appName, apiURL, logger)
    }

    // Local build
    configPath := c.String("config")

    // Try to load config file, fall back to zero-config if not found
    cfg, err := config.LoadConfigFile(configPath)
    if err != nil {
        // Check if it's a "not found" error
        if strings.Contains(err.Error(), "not found") {
            // Zero-config mode
            fmt.Println("⚡ No cloudstation.hcl found, using zero-config mode")

            cfg, err = config.GenerateDefaultConfig(".")
            if err != nil {
                return fmt.Errorf("failed to generate default config: %w", err)
            }

            // Use detected project name as app name
            appName = cfg.Apps[0].Name

            // Get detection info for user feedback
            detection := detect.DetectBuilder(".")
            fmt.Printf("  Project: %s\n", appName)
            fmt.Printf("  Builder: %s (%s)\n", detection.Builder, detection.Reason)
            if len(detection.Signals) > 0 {
                fmt.Printf("  Detected: %v\n", detection.Signals)
            }
            fmt.Println()
        } else {
            return fmt.Errorf("failed to load configuration: %w", err)
        }
    } else {
        // Config file exists, auto-detect app name if not provided
        if appName == "" && len(cfg.Apps) > 0 {
            appName = cfg.Apps[0].Name
        }
    }

    if appName == "" {
        return fmt.Errorf("--app flag required when config has no apps defined")
    }

    executor := lifecycle.NewExecutor(cfg, logger)

    ctx := context.Background()
    artifact, err := executor.BuildOnly(ctx, appName)
    if err != nil {
        return fmt.Errorf("build failed: %w", err)
    }

    fmt.Printf("Build completed successfully\n")
    fmt.Printf("  Artifact ID: %s\n", artifact.ID)
    fmt.Printf("  Image: %s\n", artifact.Image)
    return nil
}
```

### 3. Add Required Import
- In `commands.go`, add import for detect package:
```go
import (
    // ... existing imports
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
)
```

### 4. Update Validator for Build-Only Mode (Optional Enhancement)
- The current validator requires both build and deploy blocks
- For build-only operations, deploy should be optional
- Consider adding `ValidateForBuildOnly()` function if validation fails

### 5. Test Zero-Config Flow
- Create test directory without cloudstation.hcl
- Run `cs build` and verify:
  - Project name detected correctly
  - Builder selected correctly (railpack vs csdocker)
  - Build completes successfully
  - Image created with correct name

## Testing Strategy

### Unit Tests
- Test `GenerateDefaultConfig()` returns valid config
- Test with different project types (Go, Node.js, Python)
- Test with and without Dockerfile present

### Integration Tests
- Test `cs build` in directory with no config
- Test `cs build` in directory with existing config (backward compatibility)
- Test builder detection (railpack vs csdocker)

### Edge Cases
- Empty directory (no project files)
- Directory with only Dockerfile
- Git repo vs non-git directory
- Special characters in directory name

## Acceptance Criteria
- [ ] `cs build` works without `cloudstation.hcl` file
- [ ] Project name auto-detected from git remote or directory
- [ ] Builder auto-selected (railpack default, csdocker if Dockerfile exists)
- [ ] Clear feedback shown in zero-config mode
- [ ] Existing configs still work (backward compatibility)
- [ ] Build produces working Docker image
- [ ] Port detection works (detected from built image)

## Validation Commands

Execute these commands to validate the task is complete:

```bash
# 1. Build the orchestrator
cd /root/code/cs-monorepo/apps/cloudstation-orchestrator
make build

# 2. Create test directory without config
mkdir -p /tmp/zero-config-test
cd /tmp/zero-config-test
echo '{"name":"test-app","scripts":{"start":"node index.js"}}' > package.json
echo 'console.log("Hello!")' > index.js

# 3. Test zero-config build (should work without cloudstation.hcl)
/root/code/cs-monorepo/apps/cloudstation-orchestrator/bin/cs build

# 4. Verify image was created
docker images | grep test-app

# 5. Test with Dockerfile (should use csdocker)
echo 'FROM node:20-alpine' > Dockerfile
/root/code/cs-monorepo/apps/cloudstation-orchestrator/bin/cs build

# 6. Verify backward compatibility (existing config still works)
cd /tmp/cs-local-test  # Directory with cloudstation.hcl from earlier
/root/code/cs-monorepo/apps/cloudstation-orchestrator/bin/cs build --app test-app
```

## Notes

- The `pkg/detect` package is already well-tested with 8 test cases
- The `config.DetectProjectName()` function already handles git remote parsing
- Railpack handles most language/framework detection internally
- Consider adding a `--zero-config` or `--auto` flag for explicit opt-in
- Future enhancement: persist generated config to disk with `cs init --auto`
