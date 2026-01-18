# Plan: Fix CLI Services Listing and Path Sanitization

## Task Description
Fix two critical issues in the cloudstation-orchestrator CLI:
1. **Services not appearing**: The `cs link` command shows only cached services from login, not dynamically fetched. Users should see ALL services and optionally filter by team.
2. **Path sanitization**: Leading/trailing slashes in `rootDirectory` (e.g., `/app/client`) cause build failures. Paths should be sanitized to relative format.

## Objective
When this plan is complete:
- `cs link` will dynamically fetch services from the API instead of using cached credentials
- Users can filter services by team with `--team` flag
- All services across teams will be visible (not just those cached at login)
- `rootDirectory` paths will be sanitized (no leading/trailing slashes)
- Builds with paths like `/app/client` will work correctly as `app/client`

## Problem Statement

### Issue 1: Services Not Appearing
The `linkCommand()` reads services from `creds.Services` which is populated **only during login** and stored in `~/.cloudstation/credentials.json`. If a user creates new services or is added to new teams after logging in, they won't appear until re-login.

```
CURRENT (BUGGY):
┌─────────────────────────────────────────────────────────────┐
│ cs login → Services cached in credentials.json              │
│     ↓                                                       │
│ cs link → Reads cached services (STALE DATA)                │
│     ↓                                                       │
│ New services created after login → NOT VISIBLE              │
└─────────────────────────────────────────────────────────────┘

FIXED:
┌─────────────────────────────────────────────────────────────┐
│ cs link → Calls ListUserServices() API (FRESH DATA)         │
│     ↓                                                       │
│ All services visible, can filter by --team                  │
└─────────────────────────────────────────────────────────────┘
```

### Issue 2: Path Sanitization
The `rootDirectory` value flows through the pipeline without sanitization:

```
JSON Input: {"rootDirectory": "/app/client"}
     ↓ (no sanitization)
HCL Output: context = "/app/client"
     ↓
Builder: cmd.Dir = "/app/client"
     ↓
ERROR: chdir /app/client: no such file or directory
```

## Solution Approach

### For Services:
- Modify `linkCommand()` to call `ListUserServices()` dynamically (pattern already exists in `initCommand()`)
- Add `--team` flag to filter services by team slug
- Add `--api-url` flag for consistency with other commands
- Use `GetTeamContext()` for auto-detection from USER_TOKEN JWT (CS_TOKEN also supported but deprecated)

### For Path Sanitization:
- Add sanitization in `mapDeployRepositoryToHCLParams()` function
- Strip leading/trailing slashes
- Use `filepath.Clean()` for normalization
- Add defense-in-depth sanitization in HCL generator

## Relevant Files

### Files to Modify

1. **`cmd/cloudstation/auth_commands.go`** (lines 142-217)
   - `linkCommand()` function - change from cached services to dynamic fetch
   - Add `--team` and `--api-url` flags

2. **`internal/dispatch/handlers.go`** (line 956)
   - `mapDeployRepositoryToHCLParams()` function - add path sanitization for RootDirectory

3. **`internal/hclgen/generator.go`** (lines 129-133)
   - `generateBuildStanza()` function - add defensive path sanitization

### Reference Files (Read-Only)

4. **`pkg/auth/client.go`** (lines 22-31, 444-460)
   - `UserService` struct with `TeamSlug` field
   - `ListUserServices()` function to call

5. **`pkg/auth/credentials.go`** (lines 279-303)
   - `GetTeamContext()` and `GetTeamFromToken()` helpers

6. **`cmd/cloudstation/commands.go`** (lines 76-81)
   - `initCommand()` - reference implementation using `ListUserServices()`

7. **`internal/dispatch/types.go`** (line 86)
   - `BuildOptions.RootDirectory` field definition

8. **`internal/hclgen/types.go`** (line 186)
   - `DeploymentParams.RootDirectory` field definition

## Implementation Phases

### Phase 1: Foundation
- Create helper function for path sanitization
- Review existing patterns in codebase

### Phase 2: Core Implementation
- Update `linkCommand()` to use dynamic service fetching
- Add path sanitization to dispatch handlers
- Add defensive sanitization to HCL generator

### Phase 3: Integration & Polish
- Add unit tests
- Test with various path formats
- Verify backward compatibility

## Step by Step Tasks

### 1. Add Path Sanitization Helper Function
In `internal/dispatch/handlers.go`, add a helper function at the top of the file (after imports):

```go
// sanitizeRootDirectory ensures the path is safe and relative for use as a build context
func sanitizeRootDirectory(path string) string {
    if path == "" || path == "." {
        return path
    }
    // Remove leading slashes (prevents absolute path interpretation)
    path = strings.TrimPrefix(path, "/")
    // Remove trailing slashes
    path = strings.TrimSuffix(path, "/")
    // Clean the path (normalizes . and ..)
    path = filepath.Clean(path)
    // Handle edge case where Clean returns "."
    if path == "." {
        return ""
    }
    return path
}
```

- Add import for `"path/filepath"` if not already present

### 2. Apply Path Sanitization in Dispatch Handler
In `internal/dispatch/handlers.go` at line 956, modify the `mapDeployRepositoryToHCLParams()` function:

**Before:**
```go
RootDirectory:  params.Build.RootDirectory,
```

**After:**
```go
RootDirectory:  sanitizeRootDirectory(params.Build.RootDirectory),
```

### 3. Add Defensive Sanitization in HCL Generator
In `internal/hclgen/generator.go` at lines 129-133, add defensive sanitization:

**Before:**
```go
if params.RootDirectory != "" {
    build.WriteString(fmt.Sprintf("    context = \"%s\"\n", params.RootDirectory))
} else {
    build.WriteString("    context = \".\"\n")
}
```

**After:**
```go
if params.RootDirectory != "" {
    // Defensive sanitization - strip leading/trailing slashes
    context := strings.TrimPrefix(params.RootDirectory, "/")
    context = strings.TrimSuffix(context, "/")
    if context != "" && context != "." {
        build.WriteString(fmt.Sprintf("    context = \"%s\"\n", context))
    } else {
        build.WriteString("    context = \".\"\n")
    }
} else {
    build.WriteString("    context = \".\"\n")
}
```

### 4. Update Link Command Flags
In `cmd/cloudstation/auth_commands.go`, update the `linkCommand()` function flags (around line 146):

**Before:**
```go
Flags: []cli.Flag{
    &cli.StringFlag{
        Name:  "service",
        Usage: "Service ID to link (e.g., prj_integ_xxx)",
    },
    &cli.BoolFlag{
        Name:  "json",
        Usage: "Output as JSON",
    },
},
```

**After:**
```go
Flags: []cli.Flag{
    &cli.StringFlag{
        Name:  "service",
        Usage: "Service ID to link (e.g., prj_integ_xxx)",
    },
    &cli.StringFlag{
        Name:  "team",
        Usage: "Filter services by team slug",
    },
    &cli.StringFlag{
        Name:    "api-url",
        Usage:   "CloudStation API URL",
        Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
        EnvVars: []string{"CS_API_URL"},
    },
    &cli.BoolFlag{
        Name:  "json",
        Usage: "Output as JSON",
    },
},
```

### 5. Update Link Command to Use Dynamic Service Fetching
In `cmd/cloudstation/auth_commands.go`, replace the service listing logic in the Action function (lines 163-176):

**Before:**
```go
serviceID := c.String("service")
if serviceID == "" {
    // Interactive selection from available services
    if len(creds.Services) == 0 {
        return fmt.Errorf("no services available to link")
    }

    fmt.Println("Available services:")
    for i, svc := range creds.Services {
        fmt.Printf("  %d. %s (%s)\n", i+1, svc.ServiceName, svc.ServiceID)
    }

    // TODO: Interactive selection
    // For now, require explicit service flag
    return fmt.Errorf("interactive service selection not yet implemented. Please use --service flag")
}
```

**After:**
```go
serviceID := c.String("service")
apiURL := c.String("api-url")
teamFilter := auth.GetTeamContext(c.String("team"))

// Fetch services dynamically from API
authClient := auth.NewClient(apiURL)
services, err := authClient.ListUserServices(creds.SessionToken)
if err != nil {
    // Fall back to cached services if API fails
    fmt.Printf("Warning: Could not fetch services from API: %v\n", err)
    fmt.Println("Falling back to cached services...")
    services = make([]auth.UserService, 0, len(creds.Services))
    for _, svc := range creds.Services {
        services = append(services, auth.UserService{
            ID:   svc.ServiceID,
            Name: svc.ServiceName,
        })
    }
}

// Filter by team if specified
if teamFilter != "" {
    filtered := make([]auth.UserService, 0)
    for _, svc := range services {
        if svc.TeamSlug == teamFilter {
            filtered = append(filtered, svc)
        }
    }
    services = filtered
}

if serviceID == "" {
    // Interactive selection from available services
    if len(services) == 0 {
        if teamFilter != "" {
            return fmt.Errorf("no services found for team '%s'", teamFilter)
        }
        return fmt.Errorf("no services available to link")
    }

    fmt.Println("Available services:")
    for i, svc := range services {
        teamInfo := ""
        if svc.TeamSlug != "" {
            teamInfo = fmt.Sprintf(" [%s]", svc.TeamSlug)
        }
        fmt.Printf("  %d. %s (%s)%s\n", i+1, svc.Name, svc.ID, teamInfo)
    }

    // TODO: Interactive selection
    return fmt.Errorf("interactive service selection not yet implemented. Please use --service flag")
}
```

### 6. Update Service Validation to Use Fetched Services
Continue modifying `auth_commands.go` - update the service validation logic (around lines 179-189):

**Before:**
```go
// Validate that the service ID exists in the user's services
service := auth.FindServiceByID(creds, serviceID)
if service == nil {
    // Check if the user passed a service name instead of ID
    service = auth.FindServiceByName(creds, serviceID)
    if service != nil {
        serviceID = service.ServiceID
    } else {
        return fmt.Errorf("service '%s' not found in your available services", serviceID)
    }
}
```

**After:**
```go
// Validate that the service ID exists in the fetched services
var matchedService *auth.UserService
for i := range services {
    if services[i].ID == serviceID || services[i].Name == serviceID {
        matchedService = &services[i]
        if services[i].Name == serviceID {
            serviceID = services[i].ID // Use ID if name was provided
        }
        break
    }
}
if matchedService == nil {
    return fmt.Errorf("service '%s' not found in your available services", serviceID)
}
```

### 7. Update Success Output to Use Matched Service
Update the success output section (around lines 196-212):

**Before:**
```go
if c.Bool("json") {
    output := map[string]interface{}{
        "success":    true,
        "service_id": serviceID,
    }
    if service != nil && service.ServiceName != "" {
        output["service_name"] = service.ServiceName
    }
    data, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(data))
    return nil
}

fmt.Printf("✓ Linked to service %s\n", serviceID)
if service != nil && service.ServiceName != "" {
    fmt.Printf("  Service name: %s\n", service.ServiceName)
}
```

**After:**
```go
if c.Bool("json") {
    output := map[string]interface{}{
        "success":    true,
        "service_id": serviceID,
    }
    if matchedService != nil {
        output["service_name"] = matchedService.Name
        if matchedService.TeamSlug != "" {
            output["team"] = matchedService.TeamSlug
        }
    }
    data, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(data))
    return nil
}

fmt.Printf("✓ Linked to service %s\n", serviceID)
if matchedService != nil {
    fmt.Printf("  Service name: %s\n", matchedService.Name)
    if matchedService.TeamSlug != "" {
        fmt.Printf("  Team: %s\n", matchedService.TeamSlug)
    }
}
```

### 8. Add Unit Tests for Path Sanitization
Create or update `internal/dispatch/handlers_test.go` to add tests:

```go
func TestSanitizeRootDirectory(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty string", "", ""},
        {"current dir", ".", "."},
        {"leading slash", "/app/client", "app/client"},
        {"trailing slash", "app/client/", "app/client"},
        {"both slashes", "/app/client/", "app/client"},
        {"multiple leading", "///app/client", "app/client"},
        {"relative path", "app/server", "app/server"},
        {"nested path", "src/app/frontend", "src/app/frontend"},
        {"just slash", "/", ""},
        {"dot with slashes", "/./", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := sanitizeRootDirectory(tt.input)
            if result != tt.expected {
                t.Errorf("sanitizeRootDirectory(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### 9. Validate Implementation
Run tests and verify the changes work correctly.

## Testing Strategy

### Unit Tests
1. **Path sanitization tests**: Cover empty, ".", leading slash, trailing slash, both slashes, nested paths
2. **Service filtering tests**: Verify team filter works correctly

### Integration Tests
1. Test `cs link` with new service created after login (should appear)
2. Test `cs link --team=<team>` filtering
3. Test deployment with `rootDirectory: "/app/client"` (should work as `app/client`)

### Edge Cases
- Empty rootDirectory (should remain empty or default to ".")
- rootDirectory = "/" (should become "." or "")
- rootDirectory = "." (should remain unchanged)
- Multiple leading/trailing slashes
- Services with no team slug
- API failure fallback to cached services

## Acceptance Criteria

- [ ] `cs link` shows ALL services from API, not just cached services
- [ ] `cs link --team=<slug>` filters services by team
- [ ] `cs link` works for services created after login (without re-login)
- [ ] `rootDirectory: "/app/client"` is sanitized to `app/client`
- [ ] `rootDirectory: "app/client/"` is sanitized to `app/client`
- [ ] Existing tests pass
- [ ] New path sanitization tests pass
- [ ] Build with subdirectory rootDirectory completes successfully

## Validation Commands

Execute these commands to validate the task is complete:

```bash
# Build the orchestrator
cd /root/code/cs-monorepo/apps/cloudstation-orchestrator
go build -o cs ./cmd/cloudstation/

# Run all tests
go test ./... -v

# Run specific tests for path sanitization
go test ./internal/dispatch/... -v -run TestSanitizeRootDirectory

# Test the link command shows services
./cs link

# Test team filtering
./cs link --team=my-team

# Test path sanitization in actual build (requires service)
# This would be tested via deployment with rootDirectory set
```

## Notes

- The `initCommand()` in `commands.go` lines 76-81 provides the pattern for dynamic service fetching
- `GetTeamContext()` helper auto-detects team from USER_TOKEN JWT claims (CS_TOKEN also supported but deprecated)
- Fallback to cached services ensures backward compatibility if API fails
- Path sanitization uses `strings.TrimPrefix/TrimSuffix` and `filepath.Clean` - standard Go patterns
- Defense-in-depth: sanitization in both handlers.go and generator.go ensures paths are always clean
