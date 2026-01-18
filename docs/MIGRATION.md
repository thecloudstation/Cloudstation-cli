## Migration Guide: Waypoint to CloudStation Orchestrator

## Overview

This guide helps you migrate from Waypoint to CloudStation Orchestrator.

## Key Differences

| Feature | Waypoint | CloudStation Orchestrator |
|---------|----------|--------------------------|
| Binary | `waypoint` | `cs` |
| Config File | `waypoint.hcl` | `cloudstation.hcl` |
| Plugin Type | External (RPC) | Builtin |
| Binary Size | 120MB | ~20MB |
| State Storage | BoltDB | In-memory (optional DB) |
| UI | Web UI included | CLI only |
| Commands | 40+ commands | 5 core commands |

## Configuration Migration

### Before (waypoint.hcl)

```hcl
project = "my-project"

app "web" {
  build {
    use "docker" {}
  }

  deploy {
    use "nomad" {}
  }
}
```

### After (cloudstation.hcl)

```hcl
project = "my-project"

app "web" {
  build {
    use = "docker"
  }

  deploy {
    use = "nomadpack"
  }
}
```

### Syntax Changes

1. **Plugin Usage**
   - Old: `use "plugin" {}`
   - New: `use = "plugin"`

2. **Plugin Configuration**
   - Old: Nested block
   - New: Inline key-value pairs

## Plugin Mapping

### Build Plugins

| Waypoint Plugin | CloudStation Plugin | Notes |
|----------------|---------------------|-------|
| `docker` | `docker` | Compatible |
| Custom Docker | `csdocker` | Docker + Vault |
| Custom Nixpacks | `nixpacks` | Nixpacks + Vault |
| - | `noop` | Testing only |

### Deploy Plugins

| Waypoint Plugin | CloudStation Plugin | Notes |
|----------------|---------------------|-------|
| `nomad` | `nomadpack` | Uses nomad-pack CLI |
| `docker` | `docker` | Platform component |

### Registry Plugins

| Waypoint Plugin | CloudStation Plugin | Notes |
|----------------|---------------------|-------|
| `docker` | `docker` | ACR support |

## Command Migration

### Build Commands

```bash
# Waypoint
waypoint build

# CloudStation
cs build --app myapp
```

### Deploy Commands

```bash
# Waypoint
waypoint deploy

# CloudStation
cs deploy --app myapp
```

### Full Lifecycle

```bash
# Waypoint
waypoint up

# CloudStation
cs up --app myapp
```

### Runner Agent

```bash
# Waypoint
waypoint runner agent -server-addr=... -server-token=...

# CloudStation
cs runner agent --server-addr=... --token=...
```

## Environment Variables

### Waypoint

```bash
export WAYPOINT_SERVER_ADDR=localhost:9701
export WAYPOINT_SERVER_TOKEN=xxxxx
```

### CloudStation

```bash
export CS_CONFIG=cloudstation.hcl
export CS_LOG_LEVEL=info
```

## Environment Variables Migration

### CS_TOKEN to USER_TOKEN (v2.0+)

**Status**: `CS_TOKEN` deprecated, `USER_TOKEN` recommended

The environment variable for service token authentication has been renamed from `CS_TOKEN` to `USER_TOKEN` for improved semantic clarity.

#### Before (deprecated)

```bash
export CS_TOKEN=eyJhbGc...
cs whoami
# Output: Authentication: Service Token (CS_TOKEN - deprecated, use USER_TOKEN)
```

#### After (recommended)

```bash
export USER_TOKEN=eyJhbGc...
cs whoami
# Output: Authentication: Service Token (USER_TOKEN)
```

#### Transition Period (both supported)

During the v2.x lifecycle, both variables are supported for backward compatibility:

```bash
# USER_TOKEN takes priority if both are set
export USER_TOKEN=eyJhbGc...  # Primary (used)
export CS_TOKEN=eyJhbGc...    # Fallback (ignored when USER_TOKEN is set)

cs whoami
# Uses USER_TOKEN, no deprecation warning
```

#### Priority Order

Credentials are loaded in this order:

1. `USER_TOKEN` environment variable (primary)
2. `CS_TOKEN` environment variable (deprecated, shows warning)
3. `~/.cloudstation/credentials.json` file (interactive login)

#### Migration Steps

**CI/CD Pipelines:**

1. Update your pipeline configuration to use `USER_TOKEN` instead of `CS_TOKEN`
2. Update secret names if necessary (e.g., `CLOUDSTATION_USER_TOKEN`)
3. Test the pipeline to verify authentication works
4. Remove `CS_TOKEN` once verified

**Docker/Kubernetes:**

1. Update environment variable names in deployment configs
2. Update ConfigMaps or Secrets if using those
3. Deploy and verify the application authenticates correctly
4. Clean up old `CS_TOKEN` references

**Shell Scripts:**

1. Search for `CS_TOKEN` references: `grep -r "CS_TOKEN" .`
2. Replace with `USER_TOKEN`
3. Test scripts locally
4. Deploy changes

#### Deprecation Timeline

| Version | Status |
|---------|--------|
| v2.0+ | `USER_TOKEN` introduced, `CS_TOKEN` deprecated (dual support) |
| v2.x | Both variables supported, deprecation warnings for `CS_TOKEN` |
| v3.0 | `CS_TOKEN` removed (breaking change) |

#### Additional Resources

For comprehensive migration details including troubleshooting and FAQ, see the complete release notes in the project documentation.

## Configuration Examples

### Docker Build + Nomad Deploy

#### Waypoint
```hcl
app "web" {
  build {
    use "docker" {
      dockerfile = "Dockerfile"
    }
  }

  deploy {
    use "nomad" {
      datacenter = "dc1"
    }
  }
}
```

#### CloudStation
```hcl
app "web" {
  build {
    use = "docker"
    dockerfile = "Dockerfile"
  }

  deploy {
    use = "nomadpack"
    deployment_name = "web"
    pack = "cloud_service"
  }
}
```

### With Vault Integration

#### Waypoint (custom plugin)
```hcl
app "web" {
  build {
    use "custom-docker-vault" {
      vault_addr = "https://vault.example.com"
      role_id = "${var.role_id}"
    }
  }
}
```

#### CloudStation
```hcl
app "web" {
  build {
    use = "csdocker"
    vault_address = "https://vault.example.com"
    role_id = env("VAULT_ROLE_ID")
    secret_id = env("VAULT_SECRET_ID")
  }
}
```

## Migration Steps

### Step 1: Install CloudStation Orchestrator

```bash
# Build from source
cd cloudstation-orchestrator
make build
sudo cp bin/cs /usr/local/bin/

# Verify
cs --version
```

### Step 2: Convert Configuration

```bash
# Create new config
cp waypoint.hcl cloudstation.hcl

# Edit cloudstation.hcl to match new syntax
# See examples above
```

### Step 3: Test Locally

```bash
# Initialize
cs init

# Test build
cs build --app myapp

# Test deploy (if applicable)
cs deploy --app myapp
```

### Step 4: Update CI/CD

Update your CI/CD pipelines to use `cs` instead of `waypoint`:

```yaml
# Before
- name: Deploy
  run: waypoint up

# After
- name: Deploy
  run: cs up --app myapp
```

### Step 5: Update cs-runner Integration

If using cs-runner, update the waypoint.ts file:

```typescript
// Before
const binary = '/kaniko/waypoint'

// After
const binary = '/kaniko/cs'
```

## Breaking Changes

### Removed Features

- **Web UI** - CLI only (can be added later)
- **Artifact Storage** - No built-in storage (use external registry)
- **BoltDB State** - In-memory only (optional DB later)
- **Plugin Marketplace** - Builtin plugins only
- **Server Auth** - Simplified token-based (no cookies)

### Changed Behavior

- **App Flag Required** - Must specify `--app` for all commands
- **No Auto-Install** - Plugins are builtin, no auto-installation
- **Simplified Config** - Fewer configuration options
- **No Variable Interpolation** - Use `env()` for environment variables

## Backward Compatibility

CloudStation Orchestrator aims to be **mostly compatible** with Waypoint HCL configs:

✅ **Compatible:**
- Project structure
- App blocks
- Build/deploy/registry sections
- Environment variable usage

❌ **Incompatible:**
- External plugins
- UI-specific configuration
- Advanced variable interpolation
- Pipeline definitions

## Rollback Plan

If you need to rollback:

1. Keep Waypoint binary installed during migration
2. Maintain both `waypoint.hcl` and `cloudstation.hcl`
3. Test thoroughly before decommissioning Waypoint
4. Keep Waypoint Docker images for 90 days

## Support

For migration assistance:

- Check documentation: `docs/`
- Review examples: `examples/`
- Report issues: GitHub Issues

## Performance Improvements

After migration, you should see:

- **70-80% smaller binary** (120MB → 20MB)
- **90% faster build time** (2min → 10s)
- **50% lower memory usage** (200MB → <100MB)
- **Simpler debugging** (5K lines vs 327K lines)

## Next Steps

1. Migrate staging environment first
2. Run parallel for 1-2 weeks
3. Monitor for issues
4. Migrate production
5. Decommission Waypoint after 30 days
