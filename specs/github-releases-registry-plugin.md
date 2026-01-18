# Plan: GitHub Releases Registry Plugin

## Task Description

Add a new registry plugin to cloudstation-orchestrator that uploads binary artifacts to GitHub Releases. This plugin will implement the `component.Registry` interface and use the GitHub CLI (`gh`) for release management, following the established patterns from the docker registry plugin and the release scripts in charlie-cli and orbit.

## Objective

Create a `github` builtin plugin that:
1. Creates GitHub releases with proper tagging and release notes
2. Uploads binary artifacts (from builders) as release assets
3. Generates SHA256 checksums for all uploaded files
4. Provides real-time upload progress via WebSocket streaming
5. Returns a `RegistryRef` with the release URL and metadata

## Problem Statement

Currently, cloudstation-orchestrator can build container images and push them to Docker registries, but there's no mechanism to release native binaries (Go, Rust, etc.) to GitHub Releases. Projects like charlie-cli and orbit have standalone release scripts, but this functionality should be integrated into the orchestrator's plugin system for consistent deployment pipelines.

## Solution Approach

Implement a new `github` registry plugin that:
- Follows the existing plugin architecture (interfaces, registration, configuration)
- Uses the GitHub CLI (`gh`) for reliable, authenticated release operations
- Extracts binary paths from `artifact.Metadata["binaries"]` or `artifact.Metadata["release_assets"]`
- Supports checksum generation and automatic release notes
- Integrates with the WebSocket streaming system for progress feedback

## Relevant Files

### Existing Files to Reference

- `/apps/cloudstation-orchestrator/pkg/component/interfaces.go` (lines 23-37)
  - Defines the `Registry` interface with `Push()`, `Pull()`, `Config()`, `ConfigSet()` methods

- `/apps/cloudstation-orchestrator/internal/plugin/registry.go` (lines 39-42)
  - Global plugin registration via `plugin.Register(name, *Plugin)`

- `/apps/cloudstation-orchestrator/builtin/docker/plugin.go` (lines 54-235)
  - Reference implementation of a complete Registry plugin with Push(), ConfigSet(), error handling

- `/apps/cloudstation-orchestrator/builtin/railpack/plugin.go` (lines 194-271)
  - Reference for ConfigSet() pattern with getString() helper and init() registration

- `/apps/cloudstation-orchestrator/pkg/artifact/types.go` (lines 6-62)
  - Artifact and RegistryRef struct definitions

- `/apps/cloudstation-orchestrator/pkg/websocket/writer.go` (lines 8-71)
  - StreamWriter for real-time log streaming

- `/apps/cloudstation-orchestrator/cmd/cloudstation/main.go` (lines 10-20)
  - Blank import pattern for plugin registration

- `/apps/charlie-cli/release-directly.sh` (lines 150-300)
  - GitHub CLI patterns: `gh release view`, `gh release create`, `gh release upload`

- `/apps/orbit/scripts/release.sh` (lines 288-472)
  - Checksum generation with `shasum -a 256`, release notes template, gh CLI usage

### New Files to Create

- `/apps/cloudstation-orchestrator/builtin/github/plugin.go`
  - Main plugin implementation with Registry struct, RegistryConfig, Push(), ConfigSet()

- `/apps/cloudstation-orchestrator/builtin/github/plugin_test.go`
  - Unit tests for configuration parsing and validation

## Implementation Phases

### Phase 1: Foundation
- Create the `builtin/github/` directory structure
- Define `RegistryConfig` struct with all configuration fields
- Implement `ConfigSet()` following the railpack pattern
- Add `init()` function for plugin registration

### Phase 2: Core Implementation
- Implement `Push()` method with GitHub CLI integration
- Add release existence checking and creation
- Implement multi-file upload with checksum generation
- Add WebSocket progress streaming support

### Phase 3: Integration & Polish
- Add blank import to `cmd/cloudstation/main.go`
- Create comprehensive unit tests
- Test with real GitHub repository
- Document HCL configuration examples

## Step by Step Tasks

### 1. Create Plugin Directory and Base Structure

- Create directory: `builtin/github/`
- Create `plugin.go` with package declaration and imports
- Define `Registry` struct with `config *RegistryConfig` and `logger hclog.Logger`
- Define `RegistryConfig` struct with fields:
  ```go
  type RegistryConfig struct {
      // Required
      Repository string // format: "owner/repo"
      Token      string // GitHub token (can use env())
      TagName    string // release tag (e.g., "v1.0.0")

      // Optional
      ReleaseName    string            // defaults to TagName
      ReleaseNotes   string            // markdown content
      Draft          bool              // create as draft
      Prerelease     bool              // mark as prerelease
      GenerateNotes  bool              // auto-generate from commits
      Checksums      bool              // generate SHA256 checksums
      CreateRelease  bool              // create if doesn't exist (default: true)
      TargetCommit   string            // target commitish
      DiscussionCategory string        // discussion category name
  }
  ```

### 2. Implement ConfigSet() Method

- Handle `nil` config by initializing empty `RegistryConfig`
- Handle `map[string]interface{}` from HCL parsing
- Create `getString()` helper function (handles both `string` and `*string`)
- Create `getBool()` helper function with default value support
- Parse all configuration fields with proper type assertions
- Support nested `auth { token = "..." }` block pattern
- Handle typed `*RegistryConfig` for direct programmatic use
- Return `nil` error (validation happens in Push())

### 3. Implement Config() Method

- Return `r.config, nil`

### 4. Implement Push() Method - Validation Phase

- Initialize logger from context or use default: `hclog.FromContext(ctx)`
- Validate required fields:
  - `r.config.Repository` must not be empty
  - `r.config.Token` must not be empty
  - `r.config.TagName` must not be empty
  - `art` (artifact) must not be nil
- Extract binary paths from artifact:
  ```go
  var assets []string
  if binaries, ok := art.Metadata["binaries"].([]string); ok {
      assets = binaries
  } else if binaries, ok := art.Metadata["release_assets"].([]interface{}); ok {
      for _, b := range binaries {
          if s, ok := b.(string); ok {
              assets = append(assets, s)
          }
      }
  } else if binaryPath, ok := art.Metadata["binary_path"].(string); ok {
      assets = []string{binaryPath}
  }
  ```
- Validate at least one asset exists

### 5. Implement Push() Method - Release Check/Create Phase

- Set up environment with token: `cmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)`
- Check if release exists:
  ```go
  checkCmd := exec.CommandContext(ctx, "gh", "release", "view", r.config.TagName,
      "--repo", r.config.Repository)
  checkCmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)
  releaseExists := checkCmd.Run() == nil
  ```
- If release doesn't exist and `CreateRelease` is true (default), create it:
  ```go
  createArgs := []string{"release", "create", r.config.TagName,
      "--repo", r.config.Repository,
      "--title", releaseName,
  }
  if r.config.Draft { createArgs = append(createArgs, "--draft") }
  if r.config.Prerelease { createArgs = append(createArgs, "--prerelease") }
  if r.config.ReleaseNotes != "" {
      createArgs = append(createArgs, "--notes", r.config.ReleaseNotes)
  } else if r.config.GenerateNotes {
      createArgs = append(createArgs, "--generate-notes")
  }
  ```

### 6. Implement Push() Method - Checksum Generation Phase

- If `r.config.Checksums` is true:
  ```go
  checksumFile := filepath.Join(filepath.Dir(assets[0]), "checksums.txt")
  var checksumContent strings.Builder
  for _, asset := range assets {
      hash, err := computeSHA256(asset)
      if err != nil { return nil, fmt.Errorf("checksum failed for %s: %w", asset, err) }
      checksumContent.WriteString(fmt.Sprintf("%s  %s\n", hash, filepath.Base(asset)))
  }
  os.WriteFile(checksumFile, []byte(checksumContent.String()), 0644)
  assets = append(assets, checksumFile)
  ```
- Implement helper function `computeSHA256(filepath string) (string, error)`

### 7. Implement Push() Method - Upload Phase

- Extract WebSocket client for progress streaming:
  ```go
  wsClient, _ := ctx.Value("wsClient").(*websocket.Client)
  ```
- Upload each asset using `gh release upload`:
  ```go
  for _, asset := range assets {
      uploadArgs := []string{"release", "upload", r.config.TagName,
          asset, "--repo", r.config.Repository, "--clobber"}
      uploadCmd := exec.CommandContext(ctx, "gh", uploadArgs...)
      uploadCmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)

      if wsClient != nil {
          stdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
          stderrWriter := websocket.NewStreamWriter(wsClient, "stderr")
          uploadCmd.Stdout = stdoutWriter
          uploadCmd.Stderr = stderrWriter
          err := uploadCmd.Run()
          stdoutWriter.Flush()
          stderrWriter.Flush()
      } else {
          output, err := uploadCmd.CombinedOutput()
          logger.Debug("upload output", "file", asset, "output", string(output))
      }
  }
  ```

### 8. Implement Push() Method - Return RegistryRef

- Construct and return `*artifact.RegistryRef`:
  ```go
  return &artifact.RegistryRef{
      Registry:   "github.com",
      Repository: r.config.Repository,
      Tag:        r.config.TagName,
      FullImage:  fmt.Sprintf("https://github.com/%s/releases/tag/%s",
                              r.config.Repository, r.config.TagName),
      PushedAt:   time.Now(),
  }, nil
  ```

### 9. Implement Pull() Method

- Return `nil, fmt.Errorf("github releases pull not implemented")`
- Note: Could implement `gh release download` in future

### 10. Implement init() Registration

```go
func init() {
    plugin.Register("github", &plugin.Plugin{
        Registry: &Registry{config: &RegistryConfig{CreateRelease: true}},
    })
}
```

### 11. Add Blank Import to main.go

- Edit `/apps/cloudstation-orchestrator/cmd/cloudstation/main.go`
- Add import: `_ "github.com/thecloudstation/cloudstation-orchestrator/builtin/github"`
- Place after other builtin imports (alphabetically)

### 12. Create Unit Tests

- Create `builtin/github/plugin_test.go`
- Test `ConfigSet()` with various input formats:
  - nil config
  - map[string]interface{} with all fields
  - nested auth block
  - typed *RegistryConfig
- Test validation in Push() with missing required fields
- Test checksum generation helper function

### 13. Validate Implementation

- Build the project: `make build`
- Run tests: `go test ./builtin/github/...`
- Verify plugin registration: check that "github" appears in plugin list

## Testing Strategy

### Unit Tests
- ConfigSet() parsing with various input formats
- Validation of required fields in Push()
- Checksum computation helper function
- RegistryRef construction

### Integration Tests (Manual)
- Create a test release on a GitHub repository
- Upload multiple binary files
- Verify checksums are generated and uploaded
- Verify release notes appear correctly
- Test draft and prerelease flags
- Test --clobber for re-uploading assets

### Edge Cases
- Empty asset list (should error)
- Missing token (should error)
- Release already exists without CreateRelease flag
- Network failures during upload
- Invalid repository format

## Acceptance Criteria

- [ ] Plugin registers as "github" in the plugin registry
- [ ] ConfigSet() correctly parses HCL configuration
- [ ] Push() creates releases when they don't exist
- [ ] Push() uploads all binary assets from artifact metadata
- [ ] SHA256 checksums are generated when `checksums = true`
- [ ] WebSocket streaming shows upload progress when available
- [ ] RegistryRef contains valid GitHub release URL
- [ ] All unit tests pass
- [ ] Project builds without errors

## Validation Commands

Execute these commands to validate the task is complete:

- `cd apps/cloudstation-orchestrator && go build ./...` - Verify code compiles
- `cd apps/cloudstation-orchestrator && go test ./builtin/github/...` - Run unit tests
- `cd apps/cloudstation-orchestrator && go test -v ./builtin/github/... -run TestConfigSet` - Test configuration
- `cd apps/cloudstation-orchestrator && make build` - Build full binary

## Notes

### HCL Configuration Example

```hcl
app "my-cli" {
  path = "./cmd/mycli"

  build {
    use = "goreleaser"  # or any builder that produces binaries
    config {
      name    = "mycli"
      targets = ["linux/amd64", "darwin/arm64"]
    }
  }

  registry {
    use = "github"
    config {
      repository     = "myorg/mycli"
      token          = env("GITHUB_TOKEN")
      tag_name       = "v1.0.0"
      release_name   = "My CLI v1.0.0"
      checksums      = true
      generate_notes = true
      draft          = false
      prerelease     = false
    }
  }
}
```

### Dependencies

- Requires GitHub CLI (`gh`) to be installed and in PATH
- Token must have `repo` scope for private repos, `public_repo` for public repos

### Metadata Convention for Builders

Builders producing binary artifacts should populate one of:
- `artifact.Metadata["binaries"] = []string{"/path/to/bin1", "/path/to/bin2"}`
- `artifact.Metadata["release_assets"] = []string{...}`
- `artifact.Metadata["binary_path"] = "/path/to/single/binary"`

### Future Enhancements

- Implement `Pull()` using `gh release download`
- Add support for release notes from file (`--notes-file`)
- Add parallel upload support for large releases
- Add progress percentage tracking for individual file uploads
- Support for discussion category linking
