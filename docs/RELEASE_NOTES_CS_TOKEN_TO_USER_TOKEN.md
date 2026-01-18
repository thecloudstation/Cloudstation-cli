# Release Notes: CS_TOKEN to USER_TOKEN Migration

**Version:** v2.0.0
**Date:** TBD
**Type:** Deprecation / Breaking Change (Future)

## Executive Summary

The CloudStation CLI `cs` now uses `USER_TOKEN` as the primary environment variable for service authentication, replacing the deprecated `CS_TOKEN` variable. This change improves semantic clarity by explicitly indicating that the token represents user/service authentication credentials.

**Key Points:**
- **Backward Compatible**: `CS_TOKEN` continues to work in v2.x with deprecation warnings
- **Zero Downtime**: Existing CI/CD pipelines using `CS_TOKEN` remain functional
- **Clear Migration Path**: Simple one-line environment variable rename
- **Deprecation Timeline**: `CS_TOKEN` supported throughout v2.x, removed in v3.0

---

## What Changed?

### Environment Variable Naming

| Aspect | Old (Deprecated) | New (Recommended) |
|--------|------------------|-------------------|
| Variable Name | `CS_TOKEN` | `USER_TOKEN` |
| Status | Deprecated with warning | Primary/Recommended |
| Support Timeline | Until v3.0 | Long-term supported |

### Priority Order

Credentials are loaded in this order:

1. `USER_TOKEN` environment variable (primary)
2. `CS_TOKEN` environment variable (deprecated, with warning)
3. `~/.cloudstation/credentials.json` file (user login)

### Error Messages

Error messages now reference `USER_TOKEN`:

```bash
# Old message
not logged in: run 'cs login' or set CS_TOKEN env var

# New message
not logged in: run 'cs login' or set USER_TOKEN env var
```

### Display Output

The `cs whoami` command now reflects the token source:

```bash
# Using USER_TOKEN
Authentication: Service Token (USER_TOKEN)

# Using CS_TOKEN (deprecated)
Authentication: Service Token (CS_TOKEN - deprecated, use USER_TOKEN)
```

---

## Who Is Affected?

### NOT Affected (No Action Required)

- **Interactive users** who use `cs login` command
- **Personal workstations** with file-based credentials (`~/.cloudstation/credentials.json`)
- **Applications** that don't use environment variables for authentication
- **Deployments** using the credential file approach

### Affected (Migration Recommended)

- **CI/CD pipelines** using `CS_TOKEN` environment variable
- **Docker containers** with `CS_TOKEN` configured
- **Kubernetes deployments** with `CS_TOKEN` secrets
- **Shell scripts** referencing `CS_TOKEN`
- **Documentation/guides** showing `CS_TOKEN` examples
- **Automation tooling** that sets `CS_TOKEN` programmatically

---

## Migration Guide

### Quick Migration (One-Line Change)

#### CI/CD Platforms

**GitHub Actions:**
```yaml
# Before
- name: Deploy
  env:
    CS_TOKEN: ${{ secrets.CLOUDSTATION_TOKEN }}

# After
- name: Deploy
  env:
    USER_TOKEN: ${{ secrets.CLOUDSTATION_TOKEN }}
```

**GitLab CI:**
```yaml
# Before
deploy:
  variables:
    CS_TOKEN: $CI_CLOUDSTATION_TOKEN

# After
deploy:
  variables:
    USER_TOKEN: $CI_CLOUDSTATION_TOKEN
```

**CircleCI:**
```yaml
# Before
- run:
    environment:
      CS_TOKEN: $CLOUDSTATION_TOKEN

# After
- run:
    environment:
      USER_TOKEN: $CLOUDSTATION_TOKEN
```

**Jenkins Pipeline:**
```groovy
// Before
environment {
    CS_TOKEN = credentials('cloudstation-token')
}

// After
environment {
    USER_TOKEN = credentials('cloudstation-token')
}
```

#### Docker/Docker Compose

```yaml
# docker-compose.yml
services:
  app:
    environment:
      # Before
      # - CS_TOKEN=${CLOUDSTATION_TOKEN}

      # After
      - USER_TOKEN=${CLOUDSTATION_TOKEN}
```

**Dockerfile:**
```dockerfile
# Before
ENV CS_TOKEN=""

# After
ENV USER_TOKEN=""
```

#### Kubernetes

```yaml
# deployment.yaml
env:
  # Before
  # - name: CS_TOKEN
  #   valueFrom:
  #     secretKeyRef:
  #       name: cloudstation-credentials
  #       key: token

  # After
  - name: USER_TOKEN
    valueFrom:
      secretKeyRef:
        name: cloudstation-credentials
        key: token
```

#### Shell Scripts

```bash
# Before
export CS_TOKEN="eyJhbGc..."
cs whoami

# After
export USER_TOKEN="eyJhbGc..."
cs whoami
```

### Gradual Migration (Zero Risk)

During the transition period, you can set both variables to ensure compatibility:

```bash
# Set both for zero-risk migration
export USER_TOKEN="eyJhbGc..."
export CS_TOKEN="eyJhbGc..."  # Fallback during transition

# USER_TOKEN takes priority, CS_TOKEN serves as backup
cs whoami
```

Once you verify `USER_TOKEN` works, remove `CS_TOKEN`.

### Verification Steps

1. **Update environment variable name** from `CS_TOKEN` to `USER_TOKEN`

2. **Test authentication**:
   ```bash
   cs whoami
   ```

3. **Verify output** shows `"Service Token (USER_TOKEN)"` instead of deprecated message

4. **Confirm team context** extraction works (if using teams):
   ```bash
   cs link --team <team-slug>
   ```

5. **Test all commands** that require authentication:
   ```bash
   cs services
   cs deploy --app myapp
   ```

---

## Impact Assessment

### What Stays the Same

- Token format (JWT) unchanged
- Token generation process unchanged
- Token validation/security unchanged
- Authentication flow unchanged
- All CLI commands work identically
- Team context extraction from JWT claims
- Backend API interactions

### What Changes

- Environment variable name (`CS_TOKEN` -> `USER_TOKEN`)
- Error messages reference `USER_TOKEN`
- Deprecation warning appears when using `CS_TOKEN`
- Documentation updated to show `USER_TOKEN`
- `cs whoami` output reflects token source

---

## Backward Compatibility Guarantee

### v2.x Lifecycle (Current)

| Behavior | Status |
|----------|--------|
| `USER_TOKEN` works | Fully supported |
| `CS_TOKEN` works | Supported with deprecation warning |
| Both set simultaneously | `USER_TOKEN` takes priority |
| No breaking changes | Existing deployments continue working |

### v3.0 (Future)

| Behavior | Status |
|----------|--------|
| `CS_TOKEN` support | **Removed** |
| Applications using `CS_TOKEN` | **Will fail** |
| Migration required | **Before upgrading to v3.0** |

### Timeline

```
v2.0 (Current)     v2.x                           v3.0 (Future)
     │               │                                │
     ▼               ▼                                ▼
┌────────────────────────────────────────┐     ┌──────────────┐
│ USER_TOKEN introduced                  │     │ CS_TOKEN     │
│ CS_TOKEN deprecated with warning       │     │ removed      │
│ Full backward compatibility            │     │              │
└────────────────────────────────────────┘     └──────────────┘
     │                                               │
     └── Migration window (recommended) ─────────────┘
```

---

## Deprecation Warning

When using `CS_TOKEN`, you'll see this warning on stderr:

```
Warning: CS_TOKEN is deprecated, please use USER_TOKEN instead
```

**Important Notes:**

- Warning goes to **stderr**, not stdout (won't break output parsing)
- Warning doesn't affect functionality
- Warning won't appear when using `USER_TOKEN`
- Token value **never** appears in warnings (security-safe)
- Warning appears once per command execution

### Silencing the Warning

The only way to silence the deprecation warning is to migrate to `USER_TOKEN`:

```bash
# Shows warning
CS_TOKEN=xxx cs whoami 2>&1 | grep -i deprecated
# Output: Warning: CS_TOKEN is deprecated, please use USER_TOKEN instead

# No warning
USER_TOKEN=xxx cs whoami 2>&1 | grep -i deprecated
# (no output)
```

---

## Testing Recommendations

### Unit Testing

Ensure your tests work with `USER_TOKEN`:

```bash
# Before
CS_TOKEN=test_token go test ./...

# After
USER_TOKEN=test_token go test ./...
```

### Integration Testing

Verify your deployment pipelines:

```bash
# Test authentication with USER_TOKEN
USER_TOKEN=$YOUR_TOKEN cs whoami

# Verify expected output
cs whoami | grep "Service Token (USER_TOKEN)"

# Test team context extraction (if applicable)
USER_TOKEN=$YOUR_TOKEN cs link --team your-team

# Verify no deprecation warning appears
USER_TOKEN=$YOUR_TOKEN cs whoami 2>&1 | grep -c deprecated
# Expected: 0
```

### Smoke Testing Checklist

After migration, verify:

- [ ] Authentication succeeds with `cs whoami`
- [ ] Service commands execute correctly (`cs services`, `cs deploy`)
- [ ] Team context auto-detection works (if using teams)
- [ ] No unexpected errors in logs
- [ ] Deprecation warnings don't appear (stderr is clean)
- [ ] CI/CD pipeline runs complete successfully
- [ ] Docker containers start without authentication errors

### Priority Verification

Test that `USER_TOKEN` takes priority over `CS_TOKEN`:

```bash
# Create two different tokens (for testing)
# TOKEN_A has team_slug: "team-alpha"
# TOKEN_B has team_slug: "team-beta"

# Set both, USER_TOKEN should win
export USER_TOKEN=$TOKEN_A
export CS_TOKEN=$TOKEN_B

# Should show team-alpha, not team-beta
cs whoami
# Authentication: Service Token (USER_TOKEN)
```

---

## FAQ

### General Questions

**Q: Do I need to regenerate my tokens?**

A: No. The token format is unchanged. Simply rename the environment variable from `CS_TOKEN` to `USER_TOKEN`.

---

**Q: What happens if I set both USER_TOKEN and CS_TOKEN?**

A: `USER_TOKEN` takes priority. `CS_TOKEN` is ignored entirely (no warning shown).

---

**Q: Will my CI/CD pipeline break immediately?**

A: No. `CS_TOKEN` continues working with a deprecation warning. You have until v3.0 to migrate.

---

**Q: How do I silence the deprecation warning?**

A: Migrate to `USER_TOKEN`. The warning only appears when using `CS_TOKEN`.

---

### Authentication Questions

**Q: Does this affect interactive login (`cs login`)?**

A: No. Interactive login uses file-based credentials (`~/.cloudstation/credentials.json`), not environment variables.

---

**Q: Can I use both tokens during migration?**

A: Yes. Setting both provides zero-risk migration. `USER_TOKEN` takes priority when both are set.

---

**Q: Does the token from `cs login` get affected?**

A: No. The `cs login` command stores credentials in `~/.cloudstation/credentials.json`, which is separate from environment variables.

---

### Timeline Questions

**Q: When will CS_TOKEN stop working?**

A: `CS_TOKEN` will be removed in v3.0. The exact release date will be announced well in advance.

---

**Q: How long do I have to migrate?**

A: The entire v2.x release cycle. We recommend migrating early to avoid last-minute issues before v3.0.

---

**Q: Will there be any more warnings before v3.0?**

A: Yes. v3.0 release notes will clearly document the breaking change, and we'll provide advance notice.

---

### Security Questions

**Q: Does this change affect security?**

A: No. Token format, validation, and security properties remain identical. Only the environment variable name changed.

---

**Q: Is my token exposed in the deprecation warning?**

A: No. Token values are never logged or displayed. Only the variable name appears in messages.

---

**Q: Do I need to rotate tokens after migration?**

A: No. The same token works with both variable names.

---

### Troubleshooting

**Q: I'm seeing "not logged in" errors after migration. What's wrong?**

A: Verify that:
1. The environment variable is named exactly `USER_TOKEN` (not `user_token` or `User_Token`)
2. The token value is correct (no extra whitespace)
3. The token hasn't expired

```bash
# Check if variable is set
echo $USER_TOKEN | head -c 20
# Should show first 20 characters of your token
```

---

**Q: The deprecation warning is breaking my script. How do I fix it?**

A: The warning goes to stderr. If your script captures stderr, either:
1. Migrate to `USER_TOKEN` (recommended)
2. Redirect stderr: `cs whoami 2>/dev/null`

---

**Q: My team context isn't being detected after migration. What's wrong?**

A: The team context extraction works the same way with `USER_TOKEN`. Verify your token contains the `team_slug` claim:

```bash
# Decode JWT payload (base64)
echo $USER_TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .
# Look for "team_slug" field
```

---

## Technical Details

### Code Changes

The following files were modified:

| File | Change |
|------|--------|
| `pkg/auth/storage.go` | Added dual token support with priority |
| `pkg/auth/credentials.go` | Updated team extraction to check both tokens |
| `cmd/cloudstation/auth_commands.go` | Updated display messages |

### Constants Added

```go
const (
    userTokenEnvVar       = "USER_TOKEN"   // Primary
    deprecatedTokenEnvVar = "CS_TOKEN"     // Deprecated
)
```

### Load Priority Implementation

```go
// Priority: USER_TOKEN > CS_TOKEN (deprecated) > credentials file
func LoadCredentials() (*Credentials, error) {
    // 1. Check USER_TOKEN first (primary)
    if token := os.Getenv(userTokenEnvVar); token != "" {
        return &Credentials{SessionToken: token}, nil
    }

    // 2. Check CS_TOKEN (deprecated, with warning)
    if token := os.Getenv(deprecatedTokenEnvVar); token != "" {
        fmt.Fprintln(os.Stderr, "Warning: CS_TOKEN is deprecated, please use USER_TOKEN instead")
        return &Credentials{SessionToken: token}, nil
    }

    // 3. Fall back to credentials file
    return loadFromFile()
}
```

---

## Support

If you encounter issues during migration:

1. Check this migration guide for common scenarios
2. Verify your token is valid: `cs whoami`
3. Review your environment variable configuration
4. Check the [MIGRATION.md](MIGRATION.md) for additional guidance
5. Report issues at: https://github.com/thecloudstation/cloudstation-orchestrator/issues

---

## Related Documentation

- [MIGRATION.md](MIGRATION.md) - Complete migration documentation
- [README.md](../README.md) - Main documentation with environment variables
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture overview

---

## Summary

| Aspect | Details |
|--------|---------|
| Change | `CS_TOKEN` renamed to `USER_TOKEN` |
| Backward Compatible | Yes, until v3.0 |
| Action Required | Rename environment variable |
| Urgency | Low (deprecation, not breaking) |
| Migration Effort | Minimal (one-line change) |

**This is a deprecation notice with full backward compatibility. Migration is recommended but not urgent.**
