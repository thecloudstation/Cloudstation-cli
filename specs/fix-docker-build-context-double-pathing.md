# Plan: Fix Docker Build Context Double-Pathing Bug

## Task Description
Fix the bug in csdocker, nixpacks, and docker builder plugins where setting `cmd.Dir` to the context path AND passing the same context as an argument to the build command causes double-pathing. For example, when `context = "app/server"`, Docker looks for `app/server/app/server` which doesn't exist.

## Objective
When this plan is complete, all three builder plugins (csdocker, nixpacks, docker) will correctly handle non-default context paths by using "." as the build context argument when `cmd.Dir` is set, following the pattern already implemented in the railpack builder.

## Problem Statement
When a user specifies `rootDirectory: "app/server"` in their deployment configuration:
1. The orchestrator correctly maps this to `context = "app/server"` in the HCL config
2. The builder plugin sets `cmd.Dir = "app/server"` to change the working directory
3. **BUG**: The builder ALSO passes `"app/server"` as the first argument to the build command
4. This causes the build tool to look for `app/server/app/server` (relative to the new working directory)
5. The path doesn't exist, and the build fails with "context not found"

```
CURRENT (BUGGY):
┌────────────────────────────────────────────────────────────────┐
│ cmd.Dir = "app/server"                                         │
│ Command: docker build app/server -f Dockerfile -t myapp:latest │
│                                                                │
│ Docker resolves: app/server/app/server (DOES NOT EXIST)        │
└────────────────────────────────────────────────────────────────┘

CORRECT (FIXED):
┌────────────────────────────────────────────────────────────────┐
│ cmd.Dir = "app/server"                                         │
│ Command: docker build . -f Dockerfile -t myapp:latest          │
│                                                                │
│ Docker resolves: app/server/. = app/server (CORRECT)           │
└────────────────────────────────────────────────────────────────┘
```

## Solution Approach
Apply the same fix pattern from railpack (which already handles this correctly) to csdocker, nixpacks, and docker builders. When `cmd.Dir` is set to a non-default context, use "." as the build context argument instead of the full path.

## Relevant Files

### Files to Modify

- `builtin/csdocker/plugin.go` (lines 81-129) - Main csdocker builder with the bug
- `builtin/nixpacks/plugin.go` (lines 67-107) - Nixpacks builder with the same bug
- `builtin/docker/plugin.go` (lines 72-115) - Docker builder with the same bug

### Reference Implementation (Already Fixed)

- `builtin/railpack/plugin.go` (lines 84-92) - Shows the correct pattern to follow
- `builtin/railpack/context_test.go` - Test file validating the correct behavior

### Test Files to Update/Create

- `builtin/csdocker/plugin_test.go` - Add context handling tests
- `builtin/nixpacks/builder_test.go` - Add context handling tests
- `builtin/docker/plugin_test.go` - Add context handling tests (if exists)

## Implementation Phases

### Phase 1: Foundation
Understand the railpack fix pattern and prepare test cases.

### Phase 2: Core Implementation
Apply the fix to all three builder plugins following the railpack pattern.

### Phase 3: Integration & Polish
Add tests, verify the fix works end-to-end, and ensure no regressions.

## Step by Step Tasks

### 1. Fix csdocker Builder (Primary Target)
- Read `builtin/csdocker/plugin.go` completely
- At line 107, before building the args array, add logic to use "." when cmd.Dir will be set:

```go
// Current (buggy) - line 107:
args := []string{"build", context}

// Fixed:
// Determine the build context argument
// If we're setting cmd.Dir, use "." as context to avoid double-pathing
buildContext := context
if context != "." && context != "" {
    buildContext = "." // Will be relative to cmd.Dir
}
args := []string{"build", buildContext}
```

- The `cmd.Dir` setting at lines 127-129 remains unchanged
- Verify dockerfile path handling still works correctly (it's relative to cmd.Dir)

### 2. Fix nixpacks Builder
- Read `builtin/nixpacks/plugin.go` completely
- At line 82, apply the same fix pattern:

```go
// Current (buggy) - lines 81-82:
args := []string{"build", context}

// Fixed:
// Determine the build context argument
// If we're setting cmd.Dir, use "." as context to avoid double-pathing
buildContext := context
if context != "." && context != "" {
    buildContext = "." // Will be relative to cmd.Dir
}
args := []string{"build", buildContext}
```

- The `cmd.Dir` setting at lines 104-107 remains unchanged

### 3. Fix docker Builder
- Read `builtin/docker/plugin.go` completely
- At line 93, apply the same fix pattern:

```go
// Current (buggy) - line 93:
args := []string{"build", buildContext}

// Fixed:
// Determine the build context argument for docker command
// If we're setting cmd.Dir, use "." as context to avoid double-pathing
buildArg := buildContext
if buildContext != "." && buildContext != "" {
    buildArg = "." // Will be relative to cmd.Dir
}
args := []string{"build", buildArg}
```

- The `cmd.Dir` setting at lines 113-115 remains unchanged

### 4. Add Test for csdocker Context Handling
- Add test to `builtin/csdocker/plugin_test.go` that validates:
  - When context is "app/server", build command uses "." as argument
  - cmd.Dir is still set to "app/server"
  - Default context "." works unchanged

### 5. Add Test for nixpacks Context Handling
- Add test to `builtin/nixpacks/builder_test.go` similar to csdocker

### 6. Validate the Fix
- Build the orchestrator: `make build`
- Run existing tests: `go test ./builtin/... -v`
- Verify no regressions in existing functionality

## Testing Strategy

### Unit Tests
1. **Context Path Resolution Test**
   - Input: `context = "app/server"`
   - Expected: Build command uses "." as argument, cmd.Dir = "app/server"

2. **Default Context Test**
   - Input: `context = ""` or `context = "."`
   - Expected: Build command uses "." as argument, cmd.Dir not set

3. **Relative Path Test**
   - Input: `context = "./src"`
   - Expected: Build command uses "." as argument, cmd.Dir = "./src"

### Integration Tests
- Test actual Docker build with subdirectory context (requires Docker)
- Verify Dockerfile is found in the correct location

### Edge Cases
- Empty context (should default to ".")
- Context with trailing slash
- Absolute paths (e.g., "/tmp/upload-xyz")

## Acceptance Criteria

- [ ] csdocker builder uses "." as build argument when cmd.Dir is set to non-default context
- [ ] nixpacks builder uses "." as build argument when cmd.Dir is set to non-default context
- [ ] docker builder uses "." as build argument when cmd.Dir is set to non-default context
- [ ] All existing tests pass
- [ ] New tests validate the context handling fix
- [ ] Build with `rootDirectory: "app/server"` completes successfully on admin-api repo

## Validation Commands

Execute these commands to validate the task is complete:

```bash
# Build the orchestrator
cd /root/code/cs-monorepo/apps/cloudstation-orchestrator
make build

# Run all builder tests
go test ./builtin/csdocker/... -v
go test ./builtin/nixpacks/... -v
go test ./builtin/docker/... -v

# Run full test suite
go test ./... -v

# Manual validation: Check the generated docker command includes "." not the context path
# (Can be verified by adding debug logging or reviewing test output)
```

## Notes

- The railpack builder at `builtin/railpack/plugin.go` lines 84-92 shows the correct implementation pattern
- The fix is minimal and localized - only affects the argument passed to the build command
- The `cmd.Dir` logic remains unchanged in all builders
- Metadata storage (artifact.Metadata) should still record the original context path, not "."
- This fix has been production-tested in railpack and is the established pattern in the codebase
