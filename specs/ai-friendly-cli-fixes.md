# Plan: AI-Friendly CLI Fixes

## Task Description
Make the CloudStation CLI (cs) fully AI-friendly by fixing flag ordering issues, adding missing --json and --dry-run flags, improving error messages, and validating service links. These changes ensure AI agents like Claude can use the CLI naturally without workarounds.

## Objective
Transform the CLI from requiring strict flag-before-argument ordering to supporting flexible flag positioning, while adding comprehensive JSON output support, dry-run previews for destructive operations, and clear error messages with usage hints.

## Problem Statement
Current CLI issues preventing natural AI usage:

1. **Flag Position Requirement**: urfave/cli v2 stops parsing flags after first positional argument
   - `cs template deploy <id> --project <prj>` FAILS
   - `cs template deploy --project <prj> <id>` WORKS

2. **Missing JSON Output**: 14 commands lack `--json` flag for structured output
3. **Missing Dry-Run**: 6 destructive commands lack preview capability
4. **Invalid Service Links**: No validation of service ID format in `~/.cloudstation/links.json`
5. **Panic Statements**: 4 files use `panic()` instead of graceful error returns
6. **Poor Error Messages**: Generic errors without usage hints or actionable context

## Solution Approach
1. Enable `UseShortOptionHandling: true` at app level for flexible flag parsing
2. Add `--json` flag to all data-returning commands using consistent MarshalIndent pattern
3. Add `--dry-run` flag to all mutative/destructive commands using guard-clause pattern
4. Add service ID format validation with regex pattern
5. Replace all `panic()` with proper error returns
6. Standardize error messages with usage hints

## Relevant Files

### Core Files to Modify

- `cmd/cloudstation/main.go` (Lines 30-90)
  - Add `UseShortOptionHandling: true` to app config
  - Enables flexible flag positioning globally

- `cmd/cloudstation/auth_commands.go` (Lines 11-130)
  - Add `--json` to `whoami` command
  - Add `--json` to `link` command

- `cmd/cloudstation/commands.go` (Lines 26-681)
  - Add `--json` to `init`, `build`, `up` commands
  - Replace generic "deployment failed" errors with detailed messages

- `cmd/cloudstation/template_commands.go` (Lines 23-281)
  - Replace `panic()` at line 18 with error return
  - Add `--dry-run` to `deploy` command

- `cmd/cloudstation/image_commands.go` (Lines 25-833)
  - Improve parser error messages (lines 723-829)
  - Add flag context to validation errors

- `cmd/cloudstation/service_commands.go` (Lines 121-532)
  - Replace `panic()` at line 61 with error return
  - Add `--dry-run` to `stop`, `start`, `restart`, `delete` commands

- `cmd/cloudstation/deployment_commands.go` (Lines 105-493)
  - Replace `panic()` at line 56 with error return
  - Add `--dry-run` to `cancel`, `clear-queue` commands

- `cmd/cloudstation/billing_commands.go` (Lines 23-381)
  - Replace `panic()` at line 18 with error return

- `pkg/auth/credentials.go` (Lines 30-78)
  - Add `ValidateServiceID()` function
  - Add validation in `SaveServiceLink()` and `LoadServiceLink()`

### New Files

None required - all changes are modifications to existing files.

## Implementation Phases

### Phase 1: Foundation (Critical - Enables AI Usage)
- Enable flexible flag ordering in main.go
- Replace all panic() statements with error returns
- Add service ID format validation

### Phase 2: Core Implementation (High Priority)
- Add --json flags to 14 missing commands
- Add --dry-run flags to 6 destructive commands
- Standardize JSON output format

### Phase 3: Integration & Polish (Medium Priority)
- Improve all error messages with usage hints
- Add parser error improvements with flag context
- Update help text to document both flag positions work

## Step by Step Tasks
IMPORTANT: Execute every step in order, top to bottom.

### 1. Enable Flexible Flag Ordering (main.go)

- Open `cmd/cloudstation/main.go`
- At line 30, modify the `cli.App` struct:
```go
app := &cli.App{
    Name:                   "cs",
    Usage:                  "CloudStation Orchestrator - Minimal deployment orchestrator",
    Version:                Version,
    UseShortOptionHandling: true,  // ADD THIS LINE
    EnableBashCompletion:   true,  // ADD THIS LINE (optional but helpful)
    Flags: []cli.Flag{
        // ... existing flags ...
    },
    // ...
}
```

### 2. Replace Panic Statements with Error Returns

**File: cmd/cloudstation/deployment_commands.go (line 56)**
- Change from:
```go
panic("SessionToken is required - user must login with 'cs login'")
```
- Change to (create helper function):
```go
func newDeploymentClient(apiURL string, creds *auth.Credentials) (*backend.DeploymentClient, error) {
    if creds.SessionToken == "" {
        return nil, fmt.Errorf("not authenticated: run 'cs login' first")
    }
    return backend.NewDeploymentClient(apiURL, creds.SessionToken), nil
}
```
- Update all usages to handle returned error

**File: cmd/cloudstation/service_commands.go (line 61)**
- Same pattern as deployment_commands.go
- Create `newServiceClient()` function that returns error

**File: cmd/cloudstation/billing_commands.go (line 18)**
- Same pattern
- Create `newBillingClient()` function that returns error

**File: cmd/cloudstation/template_commands.go (line 18)**
- Same pattern
- Modify `newTemplateClient()` to return error instead of panic

### 3. Add Service ID Validation (pkg/auth/credentials.go)

- Add new function before `SaveServiceLink()` (around line 28):
```go
import "regexp"

// ValidateServiceID checks if a service ID has valid format
// Valid formats: svc_xxx, prj_integ_xxx, img_xxx
func ValidateServiceID(serviceID string) error {
    if serviceID == "" {
        return fmt.Errorf("service ID cannot be empty")
    }

    // Match: svc_, prj_integ_, img_ followed by alphanumeric/hyphens
    pattern := `^(svc|prj_integ|img)_[a-zA-Z0-9-]+$`
    matched, err := regexp.MatchString(pattern, serviceID)
    if err != nil {
        return fmt.Errorf("failed to validate service ID: %w", err)
    }
    if !matched {
        return fmt.Errorf("invalid service ID format '%s': must start with svc_, prj_integ_, or img_", serviceID)
    }
    return nil
}
```

- Modify `SaveServiceLink()` (line 31) to add validation:
```go
func SaveServiceLink(serviceID string) error {
    if err := ValidateServiceID(serviceID); err != nil {
        return err
    }
    // ... rest of existing code ...
}
```

- Modify `LoadServiceLink()` (line 77, before return):
```go
    if err := ValidateServiceID(serviceID); err != nil {
        return "", fmt.Errorf("corrupted service link: %w (run 'cs link' to fix)", err)
    }
    return serviceID, nil
```

### 4. Add --json Flag to Auth Commands (auth_commands.go)

**whoami command (lines 48-69)**
- Add flag:
```go
Flags: []cli.Flag{
    &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
},
```
- Add JSON output handling:
```go
if c.Bool("json") {
    output := map[string]interface{}{
        "email":          creds.Email,
        "user_id":        creds.UserID,
        "user_uuid":      creds.UserUUID,
        "services_count": len(creds.Services),
        "expires_at":     creds.ExpiresAt,
        "is_super_admin": creds.IsSuperAdmin,
    }
    data, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(data))
    return nil
}
```

**link command (lines 72-130)**
- Add `--json` flag
- Output: `{"success": true, "service_id": "xxx", "service_name": "xxx"}`

### 5. Add --json Flag to Lifecycle Commands (commands.go)

**init command (lines 26-188)**
- Add `--json` flag
- Output initialization result as JSON

**build command (lines 191-277)**
- Add `--json` flag
- Output: `{"success": true, "artifact_id": "xxx", "image": "xxx:tag"}`

**up command (lines 297-362)**
- Add `--json` flag
- Output: `{"success": true, "deployment_id": "xxx", "status": "xxx", "url": "xxx"}`

### 6. Add --dry-run Flag to Template Deploy (template_commands.go)

- At line 152, add flag:
```go
&cli.BoolFlag{Name: "dry-run", Usage: "Preview deployment without executing"},
```

- Before API call (around line 188), add guard:
```go
if c.Bool("dry-run") {
    fmt.Println("DRY RUN MODE - Would deploy:")
    fmt.Printf("  Template: %s\n", c.Args().First())
    fmt.Printf("  Project: %s\n", projectID)
    if c.String("env") != "" {
        fmt.Printf("  Environment: %s\n", c.String("env"))
    }
    if c.Bool("json") {
        output := map[string]interface{}{
            "dry_run":     true,
            "template_id": c.Args().First(),
            "project_id":  projectID,
            "environment": c.String("env"),
        }
        data, _ := json.MarshalIndent(output, "", "  ")
        fmt.Println(string(data))
    }
    return nil
}
```

### 7. Add --dry-run Flag to Service Commands (service_commands.go)

**For stop, start, restart commands:**
- Add `&cli.BoolFlag{Name: "dry-run", Usage: "Preview action without executing"}` to flags
- Add guard clause before API call:
```go
if c.Bool("dry-run") {
    fmt.Printf("DRY RUN: Would %s service %s\n", "stop|start|restart", serviceID)
    if c.Bool("json") {
        output := map[string]interface{}{
            "dry_run":    true,
            "action":     "stop|start|restart",
            "service_id": serviceID,
        }
        data, _ := json.MarshalIndent(output, "", "  ")
        fmt.Println(string(data))
    }
    return nil
}
```

**For delete command (line 440):**
- Add `--dry-run` flag
- Guard should show what would be deleted:
```go
if c.Bool("dry-run") {
    fmt.Printf("DRY RUN: Would permanently delete service %s\n", serviceID)
    fmt.Println("  This action cannot be undone.")
    fmt.Println("  All associated data will be removed.")
    return nil
}
```

### 8. Add --dry-run Flag to Deployment Commands (deployment_commands.go)

**cancel command (line 334)**
- Add `--dry-run` flag
- Show what deployment would be cancelled

**clear-queue command (line 404)**
- Add `--dry-run` flag
- Show what would be cleared (list queued deployments if possible)

### 9. Improve Error Messages

**Parser errors in image_commands.go (lines 723-829)**
- Change from: `"must be in KEY=value format"`
- Change to: `"invalid --env format: must be KEY=value (e.g., --env DEBUG=true)"`

**Generic errors in commands.go (line 462)**
- Change from: `fmt.Errorf("deployment failed")`
- Change to: `fmt.Errorf("deployment failed: status=%s, reason=%s (ID: %s)", deployment.Status, deployment.StatusReason, deployment.ID)`

**Validation errors - add usage hints:**
```go
// Pattern for all argument validation errors:
return fmt.Errorf("service ID required\n\nUsage: cs service stop <service-id>\n\nRun 'cs service stop --help' for more information")
```

### 10. Validate All Changes

- Build the CLI: `go build -o cs ./cmd/cloudstation/`
- Run test suite: `go test ./...`
- Manual testing of flag ordering:
  - `./cs template deploy --project prj_xxx apptpl_xxx` (should work)
  - `./cs template deploy apptpl_xxx --project prj_xxx` (should now also work)
- Test --json output on all modified commands
- Test --dry-run on all destructive commands
- Test service link validation with invalid IDs

## Testing Strategy

### Unit Tests
- Add tests for `ValidateServiceID()` in `pkg/auth/credentials_test.go`
- Add tests for error returns (no panics) in client creation functions
- Test JSON output format consistency

### Integration Tests
- Test flag ordering: `cs <cmd> <arg> --flag` vs `cs <cmd> --flag <arg>`
- Test --dry-run doesn't execute actions
- Test --json produces valid JSON

### Edge Cases
- Invalid service ID formats in links.json
- Empty flags/arguments
- Combined flags: `--json --dry-run`

## Acceptance Criteria

- [ ] `cs template deploy <id> --project <prj>` works (flags after args)
- [ ] `cs template deploy --project <prj> <id>` works (flags before args)
- [ ] All 14 commands have `--json` flag producing valid JSON
- [ ] All 6 destructive commands have `--dry-run` flag
- [ ] No `panic()` statements in cmd/cloudstation/*.go
- [ ] Invalid service IDs are rejected with clear error message
- [ ] All error messages include usage hints
- [ ] `go test ./...` passes
- [ ] `go build ./cmd/cloudstation/` succeeds

## Validation Commands

Execute these commands to validate the task is complete:

```bash
# Build the CLI
go build -o cs ./cmd/cloudstation/

# Run tests
go test ./... -v

# Test flag ordering (both should work)
./cs template deploy --project prj_test apptpl_test --dry-run
./cs template deploy apptpl_test --project prj_test --dry-run

# Test JSON output
./cs whoami --json
./cs project list --json
./cs billing status --json

# Test dry-run
./cs service delete svc_test --dry-run
./cs template deploy apptpl_test --project prj_test --dry-run

# Test service link validation
echo '{"service_id": "invalid"}' > ~/.cloudstation/links.json
./cs deployment list  # Should show clear validation error

# Verify no panics (grep for panic in cmd files)
grep -r "panic(" cmd/cloudstation/*.go  # Should return nothing
```

## Notes

### Dependencies
- No new dependencies required
- Uses existing `encoding/json` for JSON output
- Uses Go standard `regexp` for service ID validation

### Backward Compatibility
- All changes are additive (new flags)
- Existing flag-before-argument syntax continues to work
- No breaking changes to existing CLI interface

### Performance Impact
- Negligible - regex validation is O(n) on short strings
- JSON marshaling already used in many commands

### Documentation Updates Needed
- Update CLI help text to mention both flag orderings work
- Document --json output schemas
- Document --dry-run behavior for each command
