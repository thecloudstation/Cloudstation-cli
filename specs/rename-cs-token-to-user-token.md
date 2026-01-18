# Plan: Rename CS_TOKEN to USER_TOKEN

## Task Description
Rename the `CS_TOKEN` environment variable to `USER_TOKEN` throughout the cloudstation-orchestrator binary to better reflect its semantic purpose as a user authentication token. This change improves code clarity and aligns naming conventions with the token's actual function.

## Objective
Successfully rename all references of `CS_TOKEN` to `USER_TOKEN` across source code, tests, and documentation while maintaining backward compatibility during a deprecation period to prevent breaking existing CI/CD pipelines and automated workflows.

## Problem Statement
The current environment variable name `CS_TOKEN` (CloudStation Token) is ambiguous and doesn't clearly communicate that it represents user/service authentication credentials. The name could be confused with other CloudStation-related configuration tokens. Renaming to `USER_TOKEN` provides semantic clarity about the token's purpose while establishing a clearer naming convention for authentication-related environment variables.

## Solution Approach
Implement a phased migration strategy with dual support for both `CS_TOKEN` (deprecated) and `USER_TOKEN` (new) during a transition period. This approach:
1. Prioritizes `USER_TOKEN` when present
2. Falls back to `CS_TOKEN` with a deprecation warning
3. Maintains full backward compatibility until a future major version
4. Updates all documentation to promote `USER_TOKEN` usage
5. Provides clear migration guidance for users

## Relevant Files

### Production Code Files

- **pkg/auth/storage.go** (lines 54, 58, 76)
  - Primary credentials loading function that checks CS_TOKEN environment variable
  - Contains error message mentioning CS_TOKEN
  - Priority: CS_TOKEN env var > ~/.cloudstation/credentials.json

- **pkg/auth/credentials.go** (lines 20, 279-294)
  - Service token validation logic
  - GetTeamFromToken() function that extracts team_slug from CS_TOKEN JWT
  - GetTeamContext() helper for team auto-detection

- **cmd/cloudstation/auth_commands.go** (lines 63-65, 95)
  - Service token detection in whoami command
  - Display message showing "Service Token (CS_TOKEN)"
  - JWT claim decoding and presentation

### Test Files

- **pkg/auth/storage_test.go** (lines 211, 303, 494)
  - Error message assertions checking for exact string "CS_TOKEN"
  - Three test cases verifying credential loading behavior

- **pkg/auth/team_context_test.go** (lines 31-99)
  - Environment variable operations using os.Setenv("CS_TOKEN", ...)
  - Six test functions testing team context extraction from tokens

### Documentation Files

- **specs/fix-cli-services-and-path-sanitization.md** (lines 58, 250)
  - References CS_TOKEN for team context extraction in specifications

- **specs/team-context-auto-detect.md** (lines 82, 88, 196, 198)
  - Detailed documentation of CS_TOKEN JWT claim extraction mechanism

- **docs/MIGRATION.md** (lines 129-143)
  - Environment variable migration documentation

- **README.md**
  - Main documentation requiring environment variable reference updates

### New Files

- **docs/RELEASE_NOTES_CS_TOKEN_TO_USER_TOKEN.md**
  - Comprehensive breaking change documentation following project patterns
  - Migration guide with before/after examples
  - Impact assessment and testing recommendations

## Implementation Phases

### Phase 1: Foundation
- Add backward-compatible dual support in core authentication code
- Define constant for environment variable name
- Implement deprecation warning mechanism
- Update core credential loading with priority handling

### Phase 2: Core Implementation
- Update all source code references to use new constant
- Modify error messages and display text
- Update test files to test both old and new variable names
- Ensure all authentication paths support both tokens

### Phase 3: Integration & Polish
- Create comprehensive migration documentation
- Update README and migration guides
- Add changelog entries
- Verify all test suites pass
- Add integration tests for backward compatibility

## Step by Step Tasks

### 1. Add Dual Token Support to Storage Layer
- Add constant in `pkg/auth/storage.go`:
  ```go
  const (
      configDirName        = ".cloudstation"
      credentialsFile      = "credentials.json"
      serviceLinksFile     = "links.json"
      userTokenEnvVar      = "USER_TOKEN"      // New primary token
      deprecatedTokenEnvVar = "CS_TOKEN"       // Deprecated, for compatibility
  )
  ```
- Update `LoadCredentials()` function to check both variables with priority:
  ```go
  // Priority: USER_TOKEN > CS_TOKEN (deprecated) > ~/.cloudstation/credentials.json
  if token := os.Getenv(userTokenEnvVar); token != "" {
      return &Credentials{SessionToken: token}, nil
  }
  // Legacy support (deprecated)
  if token := os.Getenv(deprecatedTokenEnvVar); token != "" {
      fmt.Fprintln(os.Stderr, "⚠️  CS_TOKEN is deprecated, please use USER_TOKEN instead")
      return &Credentials{SessionToken: token}, nil
  }
  ```
- Update error message to mention both variables:
  ```go
  return nil, fmt.Errorf("not logged in: run 'cs login' or set USER_TOKEN env var")
  ```

### 2. Update Credentials Package
- Update `pkg/auth/credentials.go` line 20 comment to reference both variables
- Modify `GetTeamFromToken()` function (lines 279-294):
  ```go
  func GetTeamFromToken() string {
      // Check USER_TOKEN first, then fall back to deprecated CS_TOKEN
      token := os.Getenv(userTokenEnvVar)
      if token == "" {
          token = os.Getenv(deprecatedTokenEnvVar)
      }
      if token == "" {
          return ""
      }
      // ... rest of implementation
  }
  ```
- Update function comment to: `// GetTeamFromToken extracts team_slug from USER_TOKEN JWT claims`

### 3. Update Auth Commands
- Modify `cmd/cloudstation/auth_commands.go` service token detection (lines 63-65):
  ```go
  // Check if this is a service token (from USER_TOKEN or CS_TOKEN env)
  hasUserToken := os.Getenv(userTokenEnvVar) != ""
  hasDeprecatedToken := os.Getenv(deprecatedTokenEnvVar) != ""
  isServiceToken := creds.Email == "" && creds.SessionToken != "" && (hasUserToken || hasDeprecatedToken)
  ```
- Update display message (line 95):
  ```go
  if hasUserToken {
      fmt.Println("Authentication: Service Token (USER_TOKEN)")
  } else if hasDeprecatedToken {
      fmt.Println("Authentication: Service Token (CS_TOKEN - deprecated, use USER_TOKEN)")
  }
  ```

### 4. Update Test Files
- Update `pkg/auth/storage_test.go`:
  - Line 211: Change expected error to `"not logged in: run 'cs login' or set USER_TOKEN env var"`
  - Line 303: Same change
  - Line 494: Same change
  - Add new test cases for CS_TOKEN backward compatibility with deprecation warning

- Update `pkg/auth/team_context_test.go`:
  - Add new test cases using USER_TOKEN
  - Keep existing CS_TOKEN tests for backward compatibility verification
  - Add test verifying USER_TOKEN takes priority over CS_TOKEN
  - Example new test:
    ```go
    func TestGetTeamContext_PrioritizesUserToken(t *testing.T) {
        // Set both tokens, USER_TOKEN should win
        os.Setenv("USER_TOKEN", createJWT("user-team"))
        os.Setenv("CS_TOKEN", createJWT("cs-team"))
        defer os.Unsetenv("USER_TOKEN")
        defer os.Unsetenv("CS_TOKEN")

        team := GetTeamContext("")
        if team != "user-team" {
            t.Errorf("Expected USER_TOKEN to take priority")
        }
    }
    ```

### 5. Update Documentation
- Create `docs/RELEASE_NOTES_CS_TOKEN_TO_USER_TOKEN.md`:
  - Executive summary explaining the change
  - Impact assessment (who is affected vs. not affected)
  - Before/after code examples
  - Step-by-step migration guide
  - Backward compatibility guarantee
  - Testing recommendations
  - Timeline for CS_TOKEN deprecation and eventual removal

- Update `docs/MIGRATION.md`:
  - Add new section after line 143: "CS_TOKEN to USER_TOKEN Migration"
  - Document the change with clear examples:
    ```markdown
    ## Environment Variables Migration

    ### CS_TOKEN → USER_TOKEN (v2.0+)

    #### Before (deprecated)
    export CS_TOKEN=eyJhbGc...

    #### After (recommended)
    export USER_TOKEN=eyJhbGc...

    #### Transition Period (both supported)
    # USER_TOKEN takes priority if both are set
    export USER_TOKEN=eyJhbGc...  # Primary
    export CS_TOKEN=eyJhbGc...    # Fallback (deprecated)
    ```

- Update `README.md`:
  - Add "Environment Variables" section documenting all authentication variables
  - Clearly mark CS_TOKEN as deprecated with migration timeline
  - Document USER_TOKEN as the recommended approach
  - Link to comprehensive release notes

- Update specification files:
  - `specs/fix-cli-services-and-path-sanitization.md`: Update references to USER_TOKEN
  - `specs/team-context-auto-detect.md`: Update code examples to use USER_TOKEN

### 6. Add Changelog Entry
- Update `dist/CHANGELOG.md`:
  ```markdown
  ## [v2.0.0] - YYYY-MM-DD

  ### Breaking Changes
  * refactor(auth): rename CS_TOKEN to USER_TOKEN for semantic clarity
    - USER_TOKEN is now the primary environment variable for service authentication
    - CS_TOKEN is deprecated but still supported for backward compatibility
    - Migration guide: docs/RELEASE_NOTES_CS_TOKEN_TO_USER_TOKEN.md
  ```

### 7. Validate Implementation
- Run all test suites and verify passing
- Test backward compatibility with CS_TOKEN
- Test new behavior with USER_TOKEN
- Test priority handling when both are set
- Verify deprecation warning appears correctly
- Check all error messages reflect new variable name

## Testing Strategy

### Unit Tests
- Test USER_TOKEN authentication flow
- Test CS_TOKEN backward compatibility with deprecation warning
- Test priority when both USER_TOKEN and CS_TOKEN are set (USER_TOKEN wins)
- Test error messages mention USER_TOKEN
- Test team context extraction from both token types
- Test credential loading priority: USER_TOKEN > CS_TOKEN > file

### Integration Tests
- Test whoami command with USER_TOKEN
- Test whoami command with CS_TOKEN (verify deprecation warning)
- Test whoami command with both (verify USER_TOKEN takes priority)
- Test team auto-detection from USER_TOKEN JWT claims
- Test all service commands that use GetTeamContext()

### Edge Cases
- Test with malformed USER_TOKEN (should fail gracefully)
- Test with expired USER_TOKEN (backend should validate)
- Test with both tokens set to different values (USER_TOKEN should win)
- Test with only CS_TOKEN set (should work with warning)
- Test with empty/whitespace token values

### Backward Compatibility Tests
- Verify existing CI/CD pipelines using CS_TOKEN continue to work
- Verify deprecation warning appears on stderr, not stdout
- Verify error messages are helpful for both migration paths
- Verify documentation reflects transition period

## Acceptance Criteria

- [ ] USER_TOKEN environment variable is checked first in LoadCredentials()
- [ ] CS_TOKEN is still supported as fallback with deprecation warning
- [ ] When both are set, USER_TOKEN takes priority
- [ ] All error messages reference USER_TOKEN as primary variable
- [ ] Deprecation warning is displayed to stderr when CS_TOKEN is used
- [ ] All test suites pass (existing + new backward compatibility tests)
- [ ] Documentation is updated with migration guide
- [ ] Changelog entry is added for breaking change
- [ ] Release notes document the migration path
- [ ] Code uses constants instead of hardcoded strings
- [ ] Specifications are updated to reflect new naming

## Validation Commands

Execute these commands to validate the task is complete:

- `go test ./pkg/auth/... -v` - Verify all authentication tests pass
- `go test ./cmd/cloudstation/... -v` - Verify all command tests pass
- `go build -o cs ./cmd/cloudstation/` - Verify binary builds successfully
- `USER_TOKEN=test_token ./cs whoami` - Test new variable works
- `CS_TOKEN=test_token ./cs whoami 2>&1 | grep deprecated` - Verify deprecation warning
- `USER_TOKEN=new CS_TOKEN=old ./cs whoami` - Verify USER_TOKEN takes priority
- `grep -r "CS_TOKEN" pkg/ cmd/ | grep -v "test\|deprecated"` - Find any hardcoded references
- `git grep "cs_token\|CS_token" .` - Case-insensitive check for variants

## Notes

### Migration Timeline
- **v2.0**: Introduce USER_TOKEN, deprecate CS_TOKEN (dual support)
- **v2.x**: Maintain dual support throughout v2 lifecycle with warnings
- **v3.0**: Remove CS_TOKEN support (breaking change)

### Security Considerations
- Token values should never appear in logs (use redaction)
- Deprecation warning should not include token value
- Both tokens have identical security properties (JWT-based)
- Migration does not change token format or validation

### Communication Strategy
- Announce deprecation in release notes before code changes
- Post migration guide to documentation site
- Add warning to CLI output when CS_TOKEN is detected
- Update examples in all tutorials and guides

### Alternative Naming Considered
- `CLOUDSTATION_USER_TOKEN` - Too verbose
- `AUTH_TOKEN` - Too generic
- `SERVICE_TOKEN` - Confusing (implies service-to-service only)
- `USER_TOKEN` - Selected for brevity and clarity

### Backward Compatibility Guarantee
During the v2.x lifecycle, CS_TOKEN will remain fully functional to prevent breaking existing deployments. The deprecation period allows users to migrate at their own pace before the v3.0 hard cutover.
