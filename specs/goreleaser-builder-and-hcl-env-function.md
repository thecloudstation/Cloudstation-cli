# Plan: GoReleaser Builder Plugin and HCL env() Function

## Task Description

Implement two critical features for the cloudstation-orchestrator:
1. **GoReleaser Builder Plugin** - A new builder that compiles Go binaries with cross-compilation support and populates artifact metadata for the GitHub registry plugin
2. **HCL env() Function** - Fix the HCL parser to properly evaluate `env("VAR_NAME")` expressions during configuration parsing

## Objective

When complete:
- Users can build Go binaries using `use = "goreleaser"` in their HCL config
- Built binaries flow to the GitHub registry plugin via artifact metadata
- `env("VAR_NAME")` expressions in HCL configs are properly evaluated to environment variable values
- The full lifecycle works: `cs up` → build binaries → upload to GitHub Releases

## Problem Statement

### Issue 1: No Binary Builder
Current builders (docker, nixpacks, railpack) produce Docker images, not standalone binaries. The GitHub registry plugin expects binary paths in artifact metadata (`binaries`, `release_assets`, or `binary_path`), but no builder populates these fields.

### Issue 2: env() Function Not Working
The HCL parser calls `gohcl.DecodeBody(file.Body, nil, &config)` with a `nil` EvalContext. This means:
- `env("VAR_NAME")` expressions cannot be evaluated
- Users must hardcode tokens or use workarounds
- The `expandEnvVars()` function only handles `${VAR}` syntax via `os.ExpandEnv()`

## Solution Approach

### GoReleaser Builder
- Create a new builder plugin at `builtin/goreleaser/`
- Use Go's native cross-compilation (`GOOS`/`GOARCH`)
- Support multiple target platforms in a single build
- Populate `artifact.Metadata["binaries"]` with built binary paths
- Follow existing builder patterns (ConfigSet, Build, init registration)

### HCL env() Function
- Create an HCL function package at `internal/hclfunc/`
- Implement `env()` function using `github.com/zclconf/go-cty/cty/function`
- Create `hcl.EvalContext` with custom functions
- Pass EvalContext to `gohcl.DecodeBody()` calls in parser
- Remove/simplify post-processing `expandEnvVars()` function

## Relevant Files

### Existing Files to Reference

**Builder Patterns:**
- `/apps/cloudstation-orchestrator/builtin/csdocker/plugin.go` (lines 55-204) - Full builder implementation with env vars, port detection, artifact construction
- `/apps/cloudstation-orchestrator/builtin/railpack/plugin.go` (lines 194-271) - ConfigSet pattern with getString() helper
- `/apps/cloudstation-orchestrator/builtin/noop/builder.go` (lines 26-43) - Simple artifact creation with metadata

**Interface Definitions:**
- `/apps/cloudstation-orchestrator/pkg/component/interfaces.go` (lines 10-21) - Builder interface
- `/apps/cloudstation-orchestrator/pkg/artifact/types.go` (lines 5-62) - Artifact and RegistryRef structs

**Plugin Registration:**
- `/apps/cloudstation-orchestrator/internal/plugin/registry.go` (lines 39-42) - Register() function
- `/apps/cloudstation-orchestrator/internal/plugin/loader.go` (lines 32-51) - Plugin loading pattern
- `/apps/cloudstation-orchestrator/cmd/cloudstation/main.go` (lines 10-17) - Blank imports for registration

**HCL Parser:**
- `/apps/cloudstation-orchestrator/internal/config/parser.go` (lines 37, 59) - DecodeBody with nil context
- `/apps/cloudstation-orchestrator/internal/config/parser.go` (lines 162-174) - expandEnvVars placeholder

**GitHub Registry (Consumer):**
- `/apps/cloudstation-orchestrator/builtin/github/plugin.go` (lines 70-90) - Asset extraction from metadata

**Build System:**
- `/apps/cloudstation-orchestrator/Makefile` (lines 73-81) - Cross-compilation pattern with GOOS/GOARCH

### New Files to Create

```
builtin/goreleaser/
├── plugin.go          # Main builder implementation
└── plugin_test.go     # Unit tests

internal/hclfunc/
├── functions.go       # HCL function definitions (env, etc.)
├── context.go         # EvalContext builder
└── functions_test.go  # Unit tests
```

## Implementation Phases

### Phase 1: Foundation - HCL Functions Package
Create the HCL function infrastructure first since it's simpler and enables testing of the goreleaser builder with env() in configs.

### Phase 2: Core Implementation - GoReleaser Builder
Implement the full builder plugin with cross-compilation support and proper artifact metadata population.

### Phase 3: Integration & Polish
- Update parser to use new EvalContext
- Add blank import for goreleaser
- Create example HCL configs
- Comprehensive testing

## Step by Step Tasks

### 1. Create HCL Functions Package

Create `/apps/cloudstation-orchestrator/internal/hclfunc/functions.go`:

- Import `github.com/zclconf/go-cty/cty/function`
- Implement `EnvFunc()` that returns environment variable values:
  ```go
  func EnvFunc() function.Function {
      return function.New(&function.Spec{
          Params: []function.Parameter{
              {Name: "varname", Type: cty.String},
          },
          Type: function.StaticReturnType(cty.String),
          Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
              return cty.StringVal(os.Getenv(args[0].AsString())), nil
          },
      })
  }
  ```
- Create `Functions()` map returning all available functions
- Add optional functions: `lower()`, `upper()`, `concat()`

### 2. Create EvalContext Builder

Create `/apps/cloudstation-orchestrator/internal/hclfunc/context.go`:

- Implement `NewEvalContext(variables map[string]string) *hcl.EvalContext`
- Build context with Functions() map
- Convert string variables to cty.Value map
- Create convenience `NewEvalContextWithEnv()` for simple use

### 3. Update HCL Parser

Modify `/apps/cloudstation-orchestrator/internal/config/parser.go`:

- Add import: `hclfunc "github.com/thecloudstation/cloudstation-orchestrator/internal/hclfunc"`
- In `ParseFile()` at line 37, replace:
  ```go
  // Before
  diags = gohcl.DecodeBody(file.Body, nil, &config)
  // After
  evalCtx := hclfunc.NewEvalContext(nil)
  diags = gohcl.DecodeBody(file.Body, evalCtx, &config)
  ```
- Do the same in `ParseBytes()` at line 59
- Simplify `expandEnvVars()` - keep `${VAR}` support but remove env() placeholder comment

### 4. Create GoReleaser Builder - Config Struct

Create `/apps/cloudstation-orchestrator/builtin/goreleaser/plugin.go`:

- Define `BuilderConfig` struct:
  ```go
  type BuilderConfig struct {
      Name       string            // Binary name (e.g., "myapp")
      Path       string            // Path to main package (e.g., "./cmd/myapp")
      Version    string            // Version string for ldflags
      Targets    []string          // Target platforms (e.g., ["linux/amd64", "darwin/arm64"])
      LdFlags    string            // Additional ldflags
      BuildArgs  map[string]string // Additional build arguments
      OutputDir  string            // Output directory (default: "./dist")
  }
  ```
- Define `Builder` struct with config and logger fields

### 5. Implement GoReleaser ConfigSet

In `builtin/goreleaser/plugin.go`:

- Handle nil config with defaults
- Handle `map[string]interface{}` from HCL:
  - Use `getString()` helper for string fields
  - Use `getStringSlice()` for targets array
  - Use `getStringMap()` for build_args
- Handle typed `*BuilderConfig`
- Set defaults: OutputDir="./dist", Targets=["linux/amd64", "darwin/arm64"]

### 6. Implement GoReleaser Build Method

In `builtin/goreleaser/plugin.go`:

- Validate config: Name required, Path defaults to "."
- Create output directory if needed
- For each target in Targets:
  - Parse GOOS/GOARCH from target string (split on "/")
  - Construct binary name: `{name}-{goos}-{goarch}` (add .exe for windows)
  - Build ldflags string with version injection:
    ```go
    ldflags := fmt.Sprintf("-s -w -X main.Version=%s", b.config.Version)
    if b.config.LdFlags != "" {
        ldflags += " " + b.config.LdFlags
    }
    ```
  - Execute `go build` with CGO_ENABLED=0:
    ```go
    cmd := exec.CommandContext(ctx, "go", "build",
        "-ldflags", ldflags,
        "-o", outputPath,
        b.config.Path)
    cmd.Env = append(os.Environ(),
        "CGO_ENABLED=0",
        "GOOS="+goos,
        "GOARCH="+goarch)
    ```
  - Collect built binary paths
- Create artifact with binaries metadata:
  ```go
  return &artifact.Artifact{
      ID:        fmt.Sprintf("goreleaser-%s-%d", b.config.Name, time.Now().Unix()),
      Image:     b.config.Name,
      Tag:       b.config.Version,
      Labels:    map[string]string{"builder": "goreleaser"},
      Metadata: map[string]interface{}{
          "builder":  "goreleaser",
          "binaries": binaryPaths,  // []string - consumed by github registry
          "targets":  b.config.Targets,
          "version":  b.config.Version,
      },
      BuildTime: time.Now(),
  }, nil
  ```

### 7. Implement Plugin Registration

In `builtin/goreleaser/plugin.go`:

- Implement `Config()` method: `return b.config, nil`
- Add `init()` function:
  ```go
  func init() {
      plugin.Register("goreleaser", &plugin.Plugin{
          Builder: &Builder{config: &BuilderConfig{
              OutputDir: "./dist",
              Targets:   []string{"linux/amd64", "darwin/arm64"},
          }},
      })
  }
  ```

### 8. Add Blank Import

Modify `/apps/cloudstation-orchestrator/cmd/cloudstation/main.go`:

- Add import between "github" and "nixpacks" (alphabetical):
  ```go
  _ "github.com/thecloudstation/cloudstation-orchestrator/builtin/goreleaser"
  ```

### 9. Create Unit Tests for HCL Functions

Create `/apps/cloudstation-orchestrator/internal/hclfunc/functions_test.go`:

- Test `EnvFunc()` with set and unset variables
- Test `NewEvalContext()` with variables
- Test integration with HCL parsing
- Test error cases

### 10. Create Unit Tests for GoReleaser Builder

Create `/apps/cloudstation-orchestrator/builtin/goreleaser/plugin_test.go`:

- Test `ConfigSet()` with nil, map, and typed configs
- Test target parsing (linux/amd64, darwin/arm64, windows/amd64)
- Test default values
- Test validation errors (missing name)
- Skip actual build tests with `t.Skip("requires Go installation")`

### 11. Create Example HCL Configuration

Create `/apps/cloudstation-orchestrator/examples/goreleaser-example.hcl`:

```hcl
project = "cli-release"

app "my-cli" {
  build {
    use = "goreleaser"
    name    = "mycli"
    path    = "./cmd/mycli"
    version = "v1.0.0"
    targets = ["linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64"]
    ldflags = "-X main.BuildTime=${BUILD_TIME}"
  }

  registry {
    use = "github"
    repository = "myorg/mycli"
    token      = env("GITHUB_TOKEN")
    tag_name   = "v1.0.0"
    checksums  = true
    draft      = true
  }

  deploy {
    use = "noop"
  }
}
```

### 12. Validate Implementation

- Run `go build ./...` to verify compilation
- Run `go test ./internal/hclfunc/...` to test HCL functions
- Run `go test ./builtin/goreleaser/...` to test builder
- Run `go test ./builtin/github/...` to ensure no regressions
- Test full lifecycle with example config

## Testing Strategy

### Unit Tests

**HCL Functions:**
- Test env() returns correct environment variable values
- Test env() returns empty string for unset variables
- Test EvalContext properly exposes functions
- Test variable injection into context

**GoReleaser Builder:**
- Test ConfigSet with all input types
- Test target parsing (GOOS/GOARCH extraction)
- Test default values applied correctly
- Test validation of required fields
- Test binary naming convention

### Integration Tests

- Test HCL parsing with env() expressions
- Test full build → registry flow with mock binaries
- Test artifact metadata is correctly populated
- Test GitHub registry can consume goreleaser artifacts

### Edge Cases

- Empty targets list (should use defaults)
- Invalid target format (should error gracefully)
- Missing required fields
- Windows target (.exe extension)
- Version with "v" prefix

## Acceptance Criteria

- [ ] `env("VAR_NAME")` expressions in HCL configs resolve to environment variable values
- [ ] GoReleaser builder registers as "goreleaser" plugin
- [ ] Builder produces binaries for all specified targets
- [ ] Binaries follow naming convention: `{name}-{goos}-{goarch}`
- [ ] Windows binaries have `.exe` extension
- [ ] Artifact metadata contains `binaries` key with paths
- [ ] GitHub registry can consume goreleaser artifacts
- [ ] Full lifecycle works: build → upload to GitHub Releases
- [ ] All unit tests pass
- [ ] Project compiles without errors

## Validation Commands

Execute these commands to validate the task is complete:

- `cd apps/cloudstation-orchestrator && go build ./...` - Verify compilation
- `cd apps/cloudstation-orchestrator && go test ./internal/hclfunc/...` - Test HCL functions
- `cd apps/cloudstation-orchestrator && go test ./builtin/goreleaser/...` - Test builder
- `cd apps/cloudstation-orchestrator && go test ./builtin/github/...` - Verify no regressions
- `cd apps/cloudstation-orchestrator && go vet ./...` - Static analysis

## Notes

### HCL Function Implementation Details

The `github.com/zclconf/go-cty/cty/function` package provides:
- `function.New(&function.Spec{...})` for creating functions
- `function.Parameter` for defining parameters with types
- `function.StaticReturnType()` for fixed return types
- Support for variadic functions with `VarLength: true`

### Binary Naming Convention

Following the existing Makefile pattern:
- `{name}-linux-amd64`
- `{name}-linux-arm64`
- `{name}-darwin-amd64`
- `{name}-darwin-arm64`
- `{name}-windows-amd64.exe`

### Version Injection

Use ldflags to inject version at build time:
```go
-ldflags "-s -w -X main.Version=v1.0.0"
```

The `-s -w` flags strip debug info for smaller binaries.

### Dependencies

No new external dependencies required - uses existing:
- `github.com/zclconf/go-cty v1.16.3` (already in go.mod)
- `github.com/hashicorp/hcl/v2 v2.24.0` (already in go.mod)

### Future Enhancements

- Add `checksums` config option to auto-generate checksums file
- Support GoReleaser YAML config files (`.goreleaser.yml`)
- Add parallel builds for faster multi-target compilation
- Support for CGO-enabled builds with specific target configurations
