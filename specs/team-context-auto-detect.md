# Plan: Team Context Auto-Detection and Flexible Team Identifier Support

## Task Description
Fix two issues in the CloudStation CLI related to team context handling:
1. **Auto-detect team context** - When using service tokens (USER_TOKEN), automatically extract team_slug from JWT claims instead of requiring explicit --team flag
2. **Support both team ID and team slug** - Allow users to pass either team_abc123 (ID) or my-team-slug (slug) to the --team flag

## Objective
Make the CLI seamlessly work with service tokens by auto-detecting team context, and support flexible team identifier formats to improve LLM and automation usability.

## Problem Statement
Currently:
- Service tokens contain `team_slug` in JWT claims, but commands ignore this and require explicit `--team` flag
- The `--team` flag is defined in 9+ commands but only actually used in 2 (project create, template deploy)
- Users must know to use team slugs, not team IDs, even though the backend supports both
- This creates friction for LLMs and automated workflows using service tokens

## Solution Approach
1. Create a shared helper function `getTeamContext()` that:
   - First checks explicit `--team` flag value
   - Falls back to extracting `team_slug` from USER_TOKEN JWT claims (CS_TOKEN also supported but deprecated)
   - Returns empty string if neither available (personal context)
2. Move `decodeJWTClaims()` from auth_commands.go to pkg/auth/ for reuse
3. Update all commands that define `--team` to actually use team context
4. Update flag descriptions to document both ID and slug support (backend already handles dual lookup)

## Relevant Files
Use these files to complete the task:

### Core Auth Package
- `pkg/auth/credentials.go` - Add `GetTeamFromToken()` helper function
- `pkg/auth/storage.go` - Already handles USER_TOKEN env var loading (with CS_TOKEN fallback)

### Command Files to Update
- `cmd/cloudstation/auth_commands.go` - Move `decodeJWTClaims()` to auth package, update imports
- `cmd/cloudstation/service_commands.go` - Lines 145-147, 244-246, 338-340, 432-434, 531-533 - Fix 5 commands that define but don't use --team
- `cmd/cloudstation/deployment_commands.go` - Lines 129-131, 353-355 - Fix 2 commands that define but don't use --team
- `cmd/cloudstation/template_commands.go` - Already working, update flag description
- `cmd/cloudstation/project_commands.go` - Already working, update flag description

### Client Files to Update
- `cmd/cloudstation/service_commands.go` - Update `ServiceClient` methods to accept team parameter

## Implementation Phases

### Phase 1: Foundation - Add JWT Decode to Auth Package
Create reusable JWT decode function and team context helper in the auth package.

### Phase 2: Core Implementation - Update Commands
Update all commands that define --team to actually use team context via the helper function.

### Phase 3: Integration & Polish
Update ServiceClient to pass team parameter to API calls, update flag descriptions.

## Step by Step Tasks
IMPORTANT: Execute every step in order, top to bottom.

### 1. Add JWT Decode Function to Auth Package
- Create new function `DecodeJWTClaims(token string) (map[string]interface{}, error)` in `pkg/auth/credentials.go`
- Move the existing implementation from `cmd/cloudstation/auth_commands.go`
- Add proper imports (encoding/base64, strings)

```go
// DecodeJWTClaims decodes claims from a JWT without signature verification
// Used for extracting service token info like team_slug
func DecodeJWTClaims(token string) (map[string]interface{}, error) {
    parts := strings.Split(token, ".")
    if len(parts) != 3 {
        return nil, fmt.Errorf("invalid JWT format")
    }
    // ... (existing implementation)
}
```

### 2. Add GetTeamFromToken Helper Function
- Add `GetTeamFromToken() string` function to `pkg/auth/credentials.go`
- This extracts team_slug from USER_TOKEN environment variable (with CS_TOKEN fallback for backward compatibility)

```go
// GetTeamFromToken extracts team_slug from USER_TOKEN JWT claims (CS_TOKEN also supported but deprecated)
// Returns empty string if no token or no team_slug claim
func GetTeamFromToken() string {
    token := os.Getenv("USER_TOKEN")
    if token == "" {
        return ""
    }
    claims, err := DecodeJWTClaims(token)
    if err != nil {
        return ""
    }
    if teamSlug, ok := claims["team_slug"].(string); ok {
        return teamSlug
    }
    return ""
}
```

### 3. Add GetTeamContext Helper Function
- Add `GetTeamContext(explicitTeam string) string` to `pkg/auth/credentials.go`
- Prioritizes explicit flag, falls back to token

```go
// GetTeamContext returns team context from explicit flag or token
// Priority: explicit flag > token claim > empty (personal)
func GetTeamContext(explicitTeam string) string {
    if explicitTeam != "" {
        return explicitTeam
    }
    return GetTeamFromToken()
}
```

### 4. Update auth_commands.go to Use Auth Package
- Replace local `decodeJWTClaims()` with `auth.DecodeJWTClaims()`
- Remove the local function definition
- Update imports

### 5. Update ServiceClient to Accept Team Parameter
- In `cmd/cloudstation/service_commands.go`, update the ServiceClient methods:
  - `List(projectID, team string)`
  - `Stop(serviceID, team string)`
  - `Start(serviceID, team string)`
  - `Restart(serviceID, team string)`
  - `Delete(serviceID, team string)`
- Add team as query parameter: `?team=<team>` if non-empty

```go
func (c *ServiceClient) List(projectID, team string) (*ServicesResponse, error) {
    path := fmt.Sprintf("/services/list/%s", projectID)
    if team != "" {
        path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
    }
    // ... rest of implementation
}
```

### 6. Update Service Commands to Use Team Context
- Update `serviceListCommand()` action:
  ```go
  team := auth.GetTeamContext(c.String("team"))
  resp, err := client.List(projectID, team)
  ```
- Update `serviceStopCommand()` action
- Update `serviceStartCommand()` action
- Update `serviceRestartCommand()` action
- Update `serviceDeleteCommand()` action

### 7. Update Deployment Commands to Use Team Context
- In `cmd/cloudstation/deployment_commands.go`:
- Update `deploymentListCommand()` to use `auth.GetTeamContext(c.String("team"))`
- Update `deploymentCancelCommand()` to use team context
- Pass team to relevant API calls

### 8. Update Flag Descriptions
- Update all `--team` flag descriptions to indicate both formats work:
  ```go
  &cli.StringFlag{
      Name:  "team",
      Usage: "Team slug or ID (auto-detected from service token if not provided)",
  }
  ```

### 9. Validate All Changes Compile and Work
- Run `go build ./cmd/cloudstation`
- Test with USER_TOKEN env var set (or CS_TOKEN for backward compatibility)
- Verify `cs whoami` shows team context
- Verify `cs service list <project> --json` works with auto-detected team

## Testing Strategy
1. **Unit Tests**: Add tests for `DecodeJWTClaims`, `GetTeamFromToken`, `GetTeamContext`
2. **Integration Tests**:
   - Test with explicit `--team=my-slug` flag
   - Test with explicit `--team=team_abc123` ID format
   - Test with USER_TOKEN containing team_slug (auto-detect)
   - Test without team context (personal)
3. **Edge Cases**:
   - Invalid JWT format in USER_TOKEN
   - Token without team_slug claim
   - Empty team value
   - Both --team flag AND USER_TOKEN set (flag should win)

## Acceptance Criteria
- [ ] Service tokens automatically provide team context without explicit --team flag
- [ ] Both team slug (`my-team`) and team ID (`team_abc123`) formats work in --team flag
- [ ] All 9 commands with --team flag actually use team context in API calls
- [ ] Flag descriptions updated to reflect auto-detection and format flexibility
- [ ] `cs whoami` shows team from token
- [ ] All existing functionality continues to work (no regressions)
- [ ] Code compiles without errors

## Validation Commands
Execute these commands to validate the task is complete:

- `go build -o cs ./cmd/cloudstation` - Verify code compiles
- `./cs whoami --help` - Verify command exists
- `USER_TOKEN="<test-token>" ./cs whoami` - Verify team extraction from token
- `./cs service list --help` - Verify updated flag description
- `USER_TOKEN="<test-token>" ./cs service list proj_xxx --dry-run --json` - Verify auto team detection
- `./cs service list proj_xxx --team=my-team --dry-run --json` - Verify explicit flag works
- `go test ./pkg/auth/... -v` - Run auth package tests

## Notes
- The backend already supports dual lookup (FindTeamBySlug then FindTeamByID) so no backend changes needed
- Team parameter should be URL-encoded when passed as query parameter
- Consider adding CS_TEAM environment variable support in a future enhancement
- The `url` package needs to be imported where query escaping is used
