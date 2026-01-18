# CloudStation Orchestrator Architecture

## Overview

CloudStation Orchestrator is a minimal, purpose-built deployment orchestrator designed to replace the unmaintained Waypoint fork. It provides a lightweight alternative (~5K lines vs 327K lines) with complete control over the deployment pipeline.

## System Components

### 1. Configuration Parser

**Location:** `internal/config/`

The configuration parser reads HCL files and converts them into Go structs. It uses HashiCorp's HCL library for parsing.

**Key Features:**
- Parse `cloudstation.hcl` files
- Support for environment variable expansion
- Validation of required fields
- Plugin existence verification

**Files:**
- `parser.go` - HCL parsing logic
- `types.go` - Configuration struct definitions
- `validator.go` - Configuration validation

### 2. Plugin System

**Location:** `internal/plugin/`

The plugin system manages builtin plugins with direct function calls (no RPC overhead).

**Components:**
- **Registry** - Central registry of all builtin plugins
- **Loader** - Loads and configures plugin instances
- **Plugin Interface** - Defines Builder, Registry, Platform, and ReleaseManager components

**Architecture:**
```
┌─────────────────────────────────────┐
│        Plugin Registry              │
│  (Global map of all plugins)        │
└──────────┬──────────────────────────┘
           │
           ├──> csdocker (Builder)
           ├──> nixpacks (Builder)
           ├──> noop (Builder)
           ├──> docker (Builder + Registry + Platform)
           └──> nomadpack (Platform)
```

### 3. Component Interfaces

**Location:** `pkg/component/`

Defines the core interfaces that all plugins must implement:

- **Builder** - Builds artifacts from source code
- **Registry** - Pushes/pulls artifacts to/from registries
- **Platform** - Deploys artifacts to platforms
- **ReleaseManager** - Manages releases (future)

### 4. Lifecycle Executor

**Location:** `internal/lifecycle/`

The lifecycle executor orchestrates the deployment pipeline:

```
Build → Registry → Deploy → Release
  ↓         ↓         ↓         ↓
Plugin   Plugin   Plugin   Plugin
```

**Phases:**
1. **Build** - Execute builder plugin to create artifact
2. **Registry** - Push artifact to container registry (optional)
3. **Deploy** - Deploy artifact to platform
4. **Release** - Manage traffic/release (optional)

**Files:**
- `executor.go` - Main lifecycle orchestration
- `context.go` - Execution context management

### 5. CLI

**Location:** `cmd/cloudstation/`

Command-line interface built with urfave/cli:

**Commands:**
- `cs init` - Create new configuration file
- `cs build` - Build an application
- `cs deploy` - Deploy an application
- `cs up` - Full build → deploy lifecycle
- `cs runner agent` - Start runner agent (future)

### 6. Builtin Plugins

**Location:** `builtin/`

All plugins are builtin (compiled into the binary):

#### csdocker
- **Type:** Builder
- **Purpose:** Docker builds with Vault secret integration
- **Key Features:** Fetches secrets from Vault, injects into build

#### nixpacks
- **Type:** Builder
- **Purpose:** Nixpacks builds with Vault integration
- **Key Features:** Uses nixpacks CLI, Vault secret injection

#### noop
- **Type:** Builder
- **Purpose:** No-op for testing
- **Key Features:** Returns empty artifact immediately

#### docker
- **Type:** Builder + Registry + Platform
- **Purpose:** Standard Docker operations
- **Key Features:** Docker build, push to ACR, deploy containers

#### nomadpack
- **Type:** Platform
- **Purpose:** Nomad Pack deployments
- **Key Features:** Uses nomad-pack CLI to deploy to Nomad

## Data Flow

### Build Flow

```
User runs: cs build --app myapp
    ↓
Load cloudstation.hcl
    ↓
Parse and validate configuration
    ↓
Get app "myapp" config
    ↓
Load builder plugin (e.g., "csdocker")
    ↓
Configure plugin with app.build.config
    ↓
Execute plugin.Build(ctx)
    ↓
Return Artifact
    ↓
Display artifact info to user
```

### Full Lifecycle Flow

```
User runs: cs up --app myapp
    ↓
Load cloudstation.hcl
    ↓
Parse and validate configuration
    ↓
Create Lifecycle Executor
    ↓
┌──────────────────┐
│   Build Phase    │ → Load builder plugin → Execute Build() → Artifact
└────────┬─────────┘
         ↓
┌──────────────────┐
│  Registry Phase  │ → Load registry plugin → Execute Push() → RegistryRef
└────────┬─────────┘
         ↓
┌──────────────────┐
│   Deploy Phase   │ → Load platform plugin → Execute Deploy() → Deployment
└────────┬─────────┘
         ↓
┌──────────────────┐
│  Release Phase   │ → Load release plugin → Execute Release() → Success
└──────────────────┘
```

## Plugin Registration

Plugins self-register in their `init()` functions:

```go
func init() {
    plugin.Register("csdocker", &plugin.Plugin{
        Builder: &Builder{},
    })
}
```

The global registry is populated when the binary starts, before main() runs.

## Configuration Structure

```hcl
project = "my-project"

app "web" {
  build {
    use = "csdocker"
    # Plugin-specific config
    vault_address = "https://vault.example.com"
  }

  registry {
    use = "docker"
    image = "myregistry.azurecr.io/web"
  }

  deploy {
    use = "nomadpack"
    deployment_name = "web-app"
    pack = "cloud_service"
  }
}
```

## Design Decisions

### Why Builtin Plugins?

- **No RPC overhead** - Direct function calls are faster
- **Type safety** - Compile-time checking of interfaces
- **Simplicity** - No plugin discovery or versioning complexity
- **Security** - All code is reviewed and compiled together
- **Size** - Single small binary

### Why No State Persistence?

- **Simplicity** - Reduces complexity and dependencies
- **Ephemeral runners** - Designed for short-lived execution
- **Stateless** - Each execution is independent
- **Future enhancement** - Can add BoltDB later if needed

### Why HCL?

- **Familiar** - Same format as Waypoint, Terraform, Nomad
- **Expressive** - Supports variables, functions, blocks
- **Maintained** - Active development by HashiCorp
- **Type-safe** - Can decode into Go structs

## Performance Characteristics

- **Binary Size:** ~15-20MB (vs 120MB Waypoint)
- **Build Time:** ~10-15 seconds (vs 2 minutes Waypoint)
- **Memory Usage:** <100MB baseline (vs 200MB+ Waypoint)
- **Plugin Loading:** <100ms per plugin (direct calls vs RPC)

## Future Enhancements

1. **Server-Runner Architecture** - gRPC-based job distribution
2. **State Persistence** - Optional BoltDB for job history
3. **Web UI** - Simple monitoring interface
4. **Metrics** - Prometheus metrics
5. **Audit Logging** - Track all deployments
6. **Plugin Hot Reload** - Reload plugins without restart

## Comparison with Waypoint

| Feature | Waypoint | CloudStation Orchestrator |
|---------|----------|--------------------------|
| Lines of Code | 327,000 | ~5,000 |
| Binary Size | 120MB | ~20MB |
| Build Time | 2 minutes | 10 seconds |
| Plugins | External (RPC) | Builtin (direct call) |
| State | BoltDB | In-memory (optional DB) |
| UI | Full web UI | CLI only |
| Maintenance | Unmaintained | Active |
| Complexity | High | Low |

## Security Considerations

- **Secrets** - Never log secrets, redact in errors
- **Vault Integration** - Short-lived tokens, auto-renewal
- **Input Validation** - Sanitize all HCL inputs
- **Dependency Scanning** - Regular security audits
- **Minimal Dependencies** - Reduce attack surface
