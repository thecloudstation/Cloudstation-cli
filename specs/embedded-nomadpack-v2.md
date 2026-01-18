# Feature: Embedded Nomad Pack with V2 Syntax Support

## Feature Description
Embed the CloudStation Nomad Pack directly into the orchestrator binary using Go's `embed` package, eliminating the need for external Git registry fetches during deployment. The embedded pack will use the new Nomad Pack v2 syntax (from `cloudstation-packs/packs/cloudstation/`), removing the dependency on the `--parser-v1` flag and enabling offline deployments with faster execution.

## User Story
As a DevOps engineer deploying CloudStation services
I want the Nomad Pack templates embedded in the orchestrator binary
So that deployments are faster, work offline, and don't depend on external Git registry availability

## Problem Statement
The current `nomadpack` plugin architecture has several issues:

1. **Remote dependency**: Every deployment executes `nomad-pack registry add <git-url>` which:
   - Requires network access to GitHub
   - Adds 2-5 seconds latency per deployment
   - Can fail if the registry is unavailable

2. **Legacy parser dependency**: Using `--parser-v1` flag due to v1 syntax in templates

3. **No offline capability**: Cannot deploy without internet access to fetch packs

4. **Inconsistent pack versions**: Different deployments might pull different pack versions

## Solution Statement
Embed the `cloudstation` pack (v2 syntax) directly into the orchestrator binary:

1. Copy pack files from `cloudstation-packs/packs/cloudstation/` to `builtin/nomadpack/packs/`
2. Use Go's `//go:embed` directive to include pack files in the binary
3. Modify the `Platform.Deploy()` method to extract embedded packs to temp directory
4. Use `nomad-pack run --path <local-path>` instead of `--registry` for embedded packs
5. Support fallback to remote registries for custom packs via `UseEmbedded` config flag

## Relevant Files
Use these files to implement the feature:

**Core Plugin Files:**
- `builtin/nomadpack/plugin.go:64-169` - Main Deploy() method that needs modification to support embedded packs
- `builtin/nomadpack/plugin.go:408-471` - addRegistry() method that can be bypassed for embedded packs
- `builtin/nomadpack/plugin.go:22-62` - PlatformConfig struct needs new fields

**Plugin System:**
- `internal/plugin/loader.go:74-93` - LoadPlatform() for configuration flow
- `internal/plugin/registry.go:69-80` - Plugin registration

**Source Pack (v2 syntax):**
- `../cloudstation-packs/packs/cloudstation/metadata.hcl` - Pack metadata
- `../cloudstation-packs/packs/cloudstation/variables.hcl` - 442 lines of variable definitions
- `../cloudstation-packs/packs/cloudstation/outputs.tpl` - Output template
- `../cloudstation-packs/packs/cloudstation/templates/_helpers.tpl` - Helper templates
- `../cloudstation-packs/packs/cloudstation/templates/cloudstation.nomad.tpl` - 466 line main job template

**Build Configuration:**
- `Makefile:17-21` - Build target needs update for pack copying
- `Dockerfile:103-105` - Docker build needs pack embedding
- `go.mod` - Module configuration (no changes needed)

**Tests:**
- `builtin/nomadpack/nomadpack_test.go` - Existing tests to extend

### New Files
- `builtin/nomadpack/packs/cloudstation/metadata.hcl` - Embedded pack metadata
- `builtin/nomadpack/packs/cloudstation/variables.hcl` - Embedded variables
- `builtin/nomadpack/packs/cloudstation/outputs.tpl` - Embedded outputs
- `builtin/nomadpack/packs/cloudstation/templates/_helpers.tpl` - Embedded helpers
- `builtin/nomadpack/packs/cloudstation/templates/cloudstation.nomad.tpl` - Embedded job template
- `builtin/nomadpack/embedded.go` - Go embed directive and extraction utilities

## Implementation Plan

### Phase 1: Foundation
Set up the embedded pack infrastructure:
1. Create directory structure for embedded packs
2. Copy v2 pack files from cloudstation-packs repository
3. Create embed.go with `//go:embed` directive
4. Add pack extraction utilities

### Phase 2: Core Implementation
Modify the nomadpack plugin to support embedded packs:
1. Add `UseEmbedded` and `EmbeddedPack` fields to PlatformConfig
2. Create `extractEmbeddedPack()` method to write pack to temp directory
3. Modify `Deploy()` to use `--path` flag for embedded packs
4. Implement hybrid mode: embedded for known packs, remote for custom packs

### Phase 3: Integration
Update build pipeline and documentation:
1. Add Makefile target to sync packs from cloudstation-packs
2. Update Dockerfile to include packs at build time
3. Add comprehensive tests for embedded pack deployment
4. Update documentation with new configuration options

## Step by Step Tasks

### Step 1: Create Pack Directory Structure
- Create `builtin/nomadpack/packs/` directory
- Create `builtin/nomadpack/packs/cloudstation/` directory
- Create `builtin/nomadpack/packs/cloudstation/templates/` directory

### Step 2: Copy V2 Pack Files
Copy from `../cloudstation-packs/packs/cloudstation/`:
- `metadata.hcl` - Pack metadata with name="cloudstation", version="1.0.0"
- `variables.hcl` - All variable definitions (442 lines)
- `outputs.tpl` - Output template
- `templates/_helpers.tpl` - Helper templates with v2 syntax
- `templates/cloudstation.nomad.tpl` - Main job template with v2 syntax (466 lines)

### Step 3: Create embedded.go
Create `builtin/nomadpack/embedded.go`:
```go
package nomadpack

import (
    "embed"
    "io/fs"
    "os"
    "path/filepath"
)

//go:embed packs/*
var EmbeddedPacks embed.FS

// AvailableEmbeddedPacks returns list of embedded pack names
func AvailableEmbeddedPacks() []string {
    return []string{"cloudstation"}
}

// HasEmbeddedPack checks if a pack is embedded
func HasEmbeddedPack(name string) bool {
    for _, p := range AvailableEmbeddedPacks() {
        if p == name {
            return true
        }
    }
    return false
}

// ExtractEmbeddedPack extracts an embedded pack to a temp directory
func ExtractEmbeddedPack(packName string) (string, error) {
    // Implementation details in Step 4
}
```

### Step 4: Implement ExtractEmbeddedPack Function
Implement the extraction logic in `embedded.go`:
- Create temp directory with prefix `cs-pack-`
- Walk embedded FS for `packs/<packName>/`
- Copy all files preserving directory structure
- Return path to extracted pack directory
- Handle cleanup via caller (defer os.RemoveAll)

```go
func ExtractEmbeddedPack(packName string) (string, error) {
    if !HasEmbeddedPack(packName) {
        return "", fmt.Errorf("embedded pack %q not found", packName)
    }

    tempDir, err := os.MkdirTemp("", "cs-pack-")
    if err != nil {
        return "", fmt.Errorf("failed to create temp dir: %w", err)
    }

    packPath := filepath.Join("packs", packName)
    err = fs.WalkDir(EmbeddedPacks, packPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        // Calculate destination path
        relPath, _ := filepath.Rel(packPath, path)
        destPath := filepath.Join(tempDir, packName, relPath)

        if d.IsDir() {
            return os.MkdirAll(destPath, 0755)
        }

        // Read and write file
        content, err := EmbeddedPacks.ReadFile(path)
        if err != nil {
            return err
        }
        return os.WriteFile(destPath, content, 0644)
    })

    if err != nil {
        os.RemoveAll(tempDir)
        return "", fmt.Errorf("failed to extract pack: %w", err)
    }

    return filepath.Join(tempDir, packName), nil
}
```

### Step 5: Update PlatformConfig
Modify `builtin/nomadpack/plugin.go` PlatformConfig struct (lines 22-62):

```go
type PlatformConfig struct {
    // Existing fields...

    // UseEmbedded uses built-in embedded pack instead of remote registry
    // When true, ignores RegistryName and RegistrySource
    UseEmbedded bool

    // EmbeddedPack overrides Pack name when using embedded packs
    // If empty, uses Pack field. Available: "cloudstation"
    EmbeddedPack string
}
```

### Step 6: Update ConfigSet Method
Modify ConfigSet in `plugin.go` (lines 338-406) to parse new fields:

```go
// Add to getString parsing section:
if val, ok := configMap["use_embedded"]; ok {
    if boolVal, ok := val.(bool); ok {
        p.config.UseEmbedded = boolVal
    }
}

if val, ok := configMap["embedded_pack"]; ok {
    if strVal, ok := val.(string); ok {
        p.config.EmbeddedPack = strVal
    }
}
```

### Step 7: Modify Deploy Method
Update `Deploy()` in `plugin.go` (lines 64-169):

```go
func (p *Platform) Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error) {
    // Existing validation...

    // Determine pack source
    var packPath string
    var cleanupFunc func()

    if p.config.UseEmbedded {
        packName := p.config.EmbeddedPack
        if packName == "" {
            packName = p.config.Pack
        }

        if !HasEmbeddedPack(packName) {
            return nil, fmt.Errorf("embedded pack %q not found, available: %v",
                packName, AvailableEmbeddedPacks())
        }

        var err error
        packPath, err = ExtractEmbeddedPack(packName)
        if err != nil {
            return nil, fmt.Errorf("failed to extract embedded pack: %w", err)
        }
        cleanupFunc = func() { os.RemoveAll(filepath.Dir(packPath)) }

        logger.Info("using embedded pack", "pack", packName, "path", packPath)
    } else {
        // Existing registry add logic
        if err := p.addRegistry(ctx); err != nil {
            return nil, fmt.Errorf("failed to add registry: %w", err)
        }
    }

    // Cleanup embedded pack on exit
    if cleanupFunc != nil {
        defer cleanupFunc()
    }

    // Build command args based on source
    args := []string{"run", p.config.Pack}
    args = append(args, "--name", p.config.DeploymentName)

    if p.config.UseEmbedded {
        args = append(args, "--path", packPath)
    } else {
        args = append(args, "--registry", p.config.RegistryName)
        if p.config.RegistryRef != "" {
            args = append(args, "--ref", p.config.RegistryRef)
        }
        // Legacy parser flag only for remote registries (v1 syntax)
        args = append(args, "--parser-v1")
    }

    // Add variables (existing logic)
    args = p.setVarArgs(args)

    // Execute command (existing logic)
    // ...
}
```

### Step 8: Add Makefile Target for Pack Sync
Add to `Makefile`:

```makefile
# Sync embedded packs from cloudstation-packs repository
sync-packs:
	@echo "Syncing embedded packs..."
	@mkdir -p builtin/nomadpack/packs/cloudstation/templates
	@cp -r ../cloudstation-packs/packs/cloudstation/* builtin/nomadpack/packs/cloudstation/
	@echo "Packs synced successfully"

# Build with embedded packs
build: sync-packs
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/cloudstation
	@echo "Binary built: $(BUILD_DIR)/$(BINARY)"
```

### Step 9: Update Dockerfile
Modify `Dockerfile` to include pack sync:

```dockerfile
# After COPY . .
# Sync packs from cloudstation-packs (if building in monorepo context)
# For standalone builds, packs should already be in builtin/nomadpack/packs/
COPY builtin/nomadpack/packs builtin/nomadpack/packs
```

### Step 10: Create Unit Tests
Create/update `builtin/nomadpack/embedded_test.go`:

```go
package nomadpack

import (
    "os"
    "path/filepath"
    "testing"
)

func TestAvailableEmbeddedPacks(t *testing.T) {
    packs := AvailableEmbeddedPacks()
    if len(packs) == 0 {
        t.Error("Expected at least one embedded pack")
    }

    found := false
    for _, p := range packs {
        if p == "cloudstation" {
            found = true
            break
        }
    }
    if !found {
        t.Error("Expected 'cloudstation' in embedded packs")
    }
}

func TestHasEmbeddedPack(t *testing.T) {
    if !HasEmbeddedPack("cloudstation") {
        t.Error("Expected cloudstation to be embedded")
    }
    if HasEmbeddedPack("nonexistent") {
        t.Error("Expected nonexistent to not be embedded")
    }
}

func TestExtractEmbeddedPack(t *testing.T) {
    path, err := ExtractEmbeddedPack("cloudstation")
    if err != nil {
        t.Fatalf("Failed to extract pack: %v", err)
    }
    defer os.RemoveAll(filepath.Dir(path))

    // Verify key files exist
    files := []string{
        "metadata.hcl",
        "variables.hcl",
        "outputs.tpl",
        "templates/_helpers.tpl",
        "templates/cloudstation.nomad.tpl",
    }

    for _, f := range files {
        fullPath := filepath.Join(path, f)
        if _, err := os.Stat(fullPath); os.IsNotExist(err) {
            t.Errorf("Expected file %s to exist", f)
        }
    }
}

func TestExtractEmbeddedPack_NotFound(t *testing.T) {
    _, err := ExtractEmbeddedPack("nonexistent")
    if err == nil {
        t.Error("Expected error for nonexistent pack")
    }
}
```

### Step 11: Update Plugin Tests
Add embedded pack tests to `builtin/nomadpack/nomadpack_test.go`:

```go
func TestDeploy_EmbeddedPack(t *testing.T) {
    config := &PlatformConfig{
        DeploymentName: "test-deployment",
        Pack:           "cloudstation",
        UseEmbedded:    true,
        NomadAddr:      "http://localhost:4646",
    }

    p := &Platform{config: config}

    // Verify embedded pack is available
    if !HasEmbeddedPack(config.Pack) {
        t.Fatalf("Embedded pack %q not found", config.Pack)
    }
}

func TestConfigSet_EmbeddedFields(t *testing.T) {
    config := map[string]interface{}{
        "deployment_name": "test",
        "pack":            "cloudstation",
        "use_embedded":    true,
        "embedded_pack":   "cloudstation",
    }

    p := &Platform{}
    err := p.ConfigSet(config)
    if err != nil {
        t.Fatalf("ConfigSet failed: %v", err)
    }

    if !p.config.UseEmbedded {
        t.Error("Expected UseEmbedded to be true")
    }
    if p.config.EmbeddedPack != "cloudstation" {
        t.Errorf("Expected EmbeddedPack='cloudstation', got %q", p.config.EmbeddedPack)
    }
}
```

### Step 12: Update HCL Generator
Update `internal/hclgen/generator.go` to support embedded pack configuration:
- When generating deploy stanza, add `use_embedded = true` for cloudstation pack
- Default to embedded for known packs

### Step 13: Run Validation Commands

## Testing Strategy

### Unit Tests
- `TestAvailableEmbeddedPacks` - Verify pack list
- `TestHasEmbeddedPack` - Verify pack detection
- `TestExtractEmbeddedPack` - Verify extraction works
- `TestExtractEmbeddedPack_NotFound` - Verify error handling
- `TestDeploy_EmbeddedPack` - Verify deployment config
- `TestConfigSet_EmbeddedFields` - Verify config parsing

### Integration Tests
- Deploy with embedded pack to local Nomad cluster
- Deploy with remote registry (fallback mode)
- Verify extracted pack renders correctly with nomad-pack
- Verify v2 syntax works without --parser-v1 flag

### Edge Cases
- Pack extraction with concurrent deployments
- Temp directory cleanup on failure
- Large pack files (>1MB)
- Invalid embedded pack structure
- Mixed embedded + remote pack usage
- Docker container builds with embedded packs

## Acceptance Criteria
- [ ] `cloudstation` pack is embedded in the binary
- [ ] Binary size increases by <500KB
- [ ] `UseEmbedded: true` deploys without network access
- [ ] No `--parser-v1` flag needed for embedded v2 packs
- [ ] Fallback to remote registry works when `UseEmbedded: false`
- [ ] All existing tests pass
- [ ] New unit tests for embedded functionality pass
- [ ] `make build` syncs packs and builds successfully
- [ ] Docker image includes embedded packs
- [ ] Deployment latency reduced by eliminating registry fetch

## Validation Commands
Execute every command to validate the feature works correctly with zero regressions.

- `cd apps/cloudstation-orchestrator && make sync-packs` - Sync packs from cloudstation-packs repository
- `cd apps/cloudstation-orchestrator && go build ./...` - Verify code compiles
- `cd apps/cloudstation-orchestrator && go test -v ./builtin/nomadpack/...` - Run nomadpack plugin tests
- `cd apps/cloudstation-orchestrator && go test -v -race ./...` - Run all tests with race detector
- `cd apps/cloudstation-orchestrator && make build` - Build binary with embedded packs
- `ls -la apps/cloudstation-orchestrator/bin/cs` - Verify binary exists
- `apps/cloudstation-orchestrator/bin/cs --version` - Verify binary runs
- `cd apps/cloudstation-orchestrator && make docker-build && make docker-test` - Build and test Docker image

## Notes

### Go Embed Limitations
- Cannot embed files from parent directories (`../cloudstation-packs`)
- Must copy pack files into `builtin/nomadpack/packs/` before build
- Embedded files are read-only; must extract to temp dir for nomad-pack

### Binary Size Impact
- Estimated pack size: ~50KB (templates + variables + metadata)
- Minimal impact on binary size (<1% increase)

### Fallback Strategy
When `UseEmbedded: false` or pack is not embedded:
1. Continue using existing `addRegistry()` + `--registry` flow
2. Use `--parser-v1` for v1 syntax packs
3. No code changes to existing remote registry path

### Migration Path
1. Default to `UseEmbedded: true` for new deployments using "cloudstation" pack
2. Existing deployments continue working unchanged
3. Optional migration: set `use_embedded = true` in HCL config

### Future Enhancements
- Embed multiple packs (cloud_service v1, cloudstation v2)
- Pack versioning with checksums
- Automatic pack updates via config
- Pack caching to avoid repeated extraction

### V2 Syntax Benefits
- No `--parser-v1` flag required
- Better error messages from nomad-pack
- Cleaner template syntax: `var "name" .` instead of `.cloud_service.name`
- Future-proof for nomad-pack updates
