# CloudStation Orchestrator

A minimal, purpose-built deployment orchestrator designed to replace the unmaintained Waypoint fork. CloudStation Orchestrator provides a lightweight (~5K lines) alternative to the bloated 327K line Waypoint codebase, with complete control over the deployment pipeline.

## Features

- **HCL Configuration** - Parse deployment configurations using HashiCorp HCL
- **Builtin Plugins** - 5 integrated plugins with direct function calls (no RPC overhead)
- **Lifecycle Execution** - Build → Registry → Deploy → Release pipeline
- **Server-Runner Architecture** - Distributed job execution via gRPC
- **Simple CLI** - Familiar commands: `init`, `up`, `build`, `deploy`, `runner agent`

## Builtin Plugins

- **csdocker** - Docker builds with Vault integration
- **nixpacks** - Nixpacks builds with Vault secrets
- **railpack** - Railway's next-gen zero-config builder with BuildKit
- **noop** - No-op builder for testing
- **nomadpack** - Nomad Pack deployments
- **docker** - Standard Docker + Azure Container Registry

## Quick Start

### Installation

```bash
# Build from source
make build

# Install to $GOPATH/bin
make install

# Verify installation
cs --version
```

### Basic Usage

```bash
# Initialize a new project
cs init

# Build an application
cs build --app myapp

# Deploy an application
cs deploy --app myapp

# Build and deploy in one command
cs up --app myapp

# Start a runner agent
cs runner agent --server-addr localhost:9701 --token <token>
```

### Example Configuration

Create a `cloudstation.hcl` file:

```hcl
project = "my-project"

app "web" {
  build {
    use "railpack" {
      name = "web"
      tag  = "latest"
    }
  }

  registry {
    use "docker" {
      image = "myregistry.azurecr.io/web"
      tag   = "latest"
      auth {
        username = env("ACR_USERNAME")
        password = env("ACR_PASSWORD")
      }
    }
  }

  deploy {
    use "nomadpack" {
      deployment_name = "web-app"
      pack            = "cloud_service"
      nomad_addr      = "https://nomad.example.com"
      nomad_token     = env("NOMAD_TOKEN")
    }
  }
}
```

## Network Configuration

CloudStation Orchestrator provides flexible network configuration through a 3-tier fallback system that balances user control with intelligent defaults.

### Configuration Priority (Highest to Lowest)

1. **User-Specified Networks** - Explicit network configuration in deployment payload
2. **Detected Ports** - Ports discovered from Docker EXPOSE directives or buildpack detection
3. **Framework Defaults** - Port defaults based on builder type (nixpacks=3000, csdocker=8000, railpack=3000)

### Network Port Configuration

Each network port supports the following configuration:

```hcl
app "myapp" {
  deploy {
    use "nomadpack" {
      networks = [
        {
          port_number     = 8080
          port_type       = "http"        # "http" or "tcp"
          public          = false         # true = public internet, false = internal only
          domain          = ""            # Auto-assigned domain
          custom_domain   = "api.internal.company.local"
          has_health_check = "http"       # "http", "tcp", "grpc", or "script"
          health_check = {
            type     = "http"
            path     = "/api/health"      # Health check endpoint path
            interval = "15s"               # How often to check
            timeout  = "10s"               # Max time to wait for response
            port     = 8080                # Port to check (defaults to port_number)
          }
        }
      ]
    }
  }
}
```

### Key Features

#### Private HTTP Services
Users can create internal-only HTTP services by setting `public = false`. This prevents unintended public exposure:

```hcl
networks = [
  {
    port_number = 8080
    port_type   = "http"
    public      = false  # Internal HTTP API - not exposed to internet
    health_check = {
      type = "http"
      path = "/health"
    }
  }
]
```

#### Custom Health Check Paths
Health check paths can be customized to match your application's endpoints:

```hcl
health_check = {
  type     = "http"
  path     = "/api/v2/health/ready"  # Custom health endpoint
  interval = "10s"
  timeout  = "5s"
}
```

#### Multiple Network Ports
Applications can expose multiple ports with different configurations:

```hcl
networks = [
  {
    port_number = 8080
    port_type   = "http"
    public      = true        # Public web interface
  },
  {
    port_number = 9090
    port_type   = "tcp"
    public      = false       # Internal metrics/admin port
  }
]
```

### Default Behavior

When network configuration is not explicitly provided:

1. **Port Detection**: CloudStation attempts to detect exposed ports from:
   - Docker `EXPOSE` directives in Dockerfile
   - Buildpack-detected ports (nixpacks, railpack)
   - Framework conventions (e.g., Next.js on 3000, Rails on 3000)

2. **Health Check Defaults**:
   - **Path**: `"/"` for HTTP health checks
   - **Interval**: `"30s"`
   - **Timeout**: `"30s"`
   - **Type**: `"tcp"` for invalid or unspecified types

3. **Public Access**: Defaults to `false` (internal only)
   - **Important**: HTTP ports are NOT automatically made public
   - Users must explicitly set `public = true` for internet-facing services

### Examples

#### Minimal Configuration (Auto-Detection)
```hcl
# CloudStation detects port 3000 from buildpack, uses defaults
app "web" {
  build {
    use "railpack" {}
  }
  deploy {
    use "nomadpack" {}
  }
}
```

#### Explicit Internal API
```hcl
# Internal-only HTTP API with custom health checks
networks = [
  {
    port_number = 8080
    port_type   = "http"
    public      = false
    custom_domain = "api.internal.company.local"
    health_check = {
      type     = "http"
      path     = "/health/ready"
      interval = "15s"
      timeout  = "10s"
    }
  }
]
```

#### Public Web Application
```hcl
# Public-facing web app with custom domain
networks = [
  {
    port_number = 3000
    port_type   = "http"
    public      = true
    custom_domain = "www.example.com"
    health_check = {
      type     = "http"
      path     = "/"
      interval = "30s"
      timeout  = "30s"
    }
  }
]
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `port_number` | int | Yes | - | Port number to expose |
| `port_type` | string | No | "http" | Port protocol: "http" or "tcp" |
| `public` | bool | No | false | Expose to public internet |
| `domain` | string | No | "" | Auto-assigned domain (managed by cluster) |
| `custom_domain` | string | No | "" | Custom domain name |
| `has_health_check` | string | No | port_type | Enable health checking: "http", "tcp", "grpc", "script" |
| `health_check.type` | string | No | "tcp" | Health check type |
| `health_check.path` | string | No | "/" | HTTP health check path |
| `health_check.interval` | string | No | "30s" | Check interval (requires time unit: s, m, h) |
| `health_check.timeout` | string | No | "30s" | Check timeout (requires time unit: s, m, h) |
| `health_check.port` | int | No | port_number | Port to perform health check on |

### Best Practices

1. **Always specify `public` explicitly** for production deployments to avoid unintended exposure
2. **Use custom health check paths** that accurately reflect application readiness
3. **Set appropriate intervals** - too frequent checks waste resources, too infrequent delays failure detection
4. **Use internal-only services** (`public = false`) for databases, caches, and internal APIs
5. **Configure custom domains** for user-facing services to provide stable endpoints

### Troubleshooting

**Issue**: Health checks failing with "connection refused"
- **Solution**: Verify `health_check.port` matches the port your application listens on
- **Solution**: Check that `health_check.path` exists in your application

**Issue**: Service not accessible from internet
- **Solution**: Ensure `public = true` is set for the network port
- **Solution**: Verify custom domain DNS configuration if using `custom_domain`

**Issue**: Health check path returns wrong URL
- **Solution**: Do NOT use time intervals (e.g., "30s") as health check paths
- **Solution**: Health check paths should be URL paths (e.g., "/health", "/api/health/ready")

## Environment Variables

### Authentication

#### USER_TOKEN (Recommended)

Primary environment variable for service token authentication in CI/CD and automated environments.

```bash
export USER_TOKEN="your_jwt_token_here"
cs whoami
```

**Use cases:**
- CI/CD pipelines (GitHub Actions, GitLab CI, CircleCI, etc.)
- Docker containers
- Kubernetes deployments
- Automated scripts
- Service-to-service authentication

#### CS_TOKEN (Deprecated)

> **Warning:** Deprecated in v2.0 - Use `USER_TOKEN` instead.

`CS_TOKEN` is maintained for backward compatibility but will be removed in v3.0.

```bash
export CS_TOKEN="your_jwt_token_here"  # Deprecated
cs whoami
# Warning: CS_TOKEN is deprecated, please use USER_TOKEN instead
```

**Migration:** See [Migration Guide](docs/RELEASE_NOTES_CS_TOKEN_TO_USER_TOKEN.md) for detailed instructions.

#### Priority Order

When multiple credential sources are available, CloudStation Orchestrator uses this priority:

1. `USER_TOKEN` environment variable (recommended)
2. `CS_TOKEN` environment variable (deprecated)
3. `~/.cloudstation/credentials.json` (from `cs login`)

### API Configuration

#### CS_API_URL

CloudStation API endpoint URL.

```bash
export CS_API_URL="https://your-api.cloud-station.io"
```

**Default:** `https://cst-cs-backend-gmlyovvq.cloud-station.io`

### Backend Integration

For CloudStation Backend API integration (optional):

| Variable | Description |
|----------|-------------|
| `BACKEND_URL` | CloudStation backend API URL (e.g., `https://api.cloudstation.io`) |
| `ACCESS_TOKEN` | API authentication token for backend communication |

When these variables are provided, the orchestrator will automatically allocate subdomains, sync service configuration, and track deployment progress. If not provided, the orchestrator operates normally without backend integration.

### Azure Container Registry

For Azure Container Registry authentication:

| Variable | Description |
|----------|-------------|
| `AZURE_CLIENT_ID` | Azure service principal client ID |
| `AZURE_CLIENT_SECRET` | Azure service principal password |
| `AZURE_TENANT_ID` | Azure tenant ID |

## Architecture

CloudStation Orchestrator uses a simple, straightforward architecture:

```
┌─────────────┐
│     CLI     │
└──────┬──────┘
       │
       ▼
┌─────────────┐      ┌──────────────┐
│   Config    │─────▶│   Plugin     │
│   Parser    │      │   Registry   │
└──────┬──────┘      └──────┬───────┘
       │                    │
       ▼                    ▼
┌─────────────┐      ┌──────────────┐
│  Lifecycle  │─────▶│   Builtin    │
│  Executor   │      │   Plugins    │
└─────────────┘      └──────────────┘

Server-Runner Model:
┌────────┐  gRPC   ┌────────┐
│ Server │◀───────▶│ Runner │
└────────┘  Jobs   └────────┘
```

## Performance

- **Binary Size**: <20MB (vs 120MB Waypoint)
- **Build Time**: <15s (vs 2min Waypoint)
- **Memory Usage**: <100MB (vs 200MB+ Waypoint)
- **Code Size**: ~5K lines (vs 327K Waypoint)

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [Plugin Development](docs/PLUGINS.md) - Creating custom plugins
- [Migration Guide](docs/MIGRATION.md) - Migrating from Waypoint
- [CS_TOKEN to USER_TOKEN Migration](docs/RELEASE_NOTES_CS_TOKEN_TO_USER_TOKEN.md) - Environment variable migration guide

## Docker Deployment

CloudStation Orchestrator can be deployed as a lightweight Docker container (~250-300MB) that includes all required external dependencies.

### Building the Docker Image

```bash
# Build the Docker image
cd cloudstation-orchestrator
make docker-build

# Build for specific architectures
make docker-build-linux-amd64
make docker-build-linux-arm64

# Check image size
docker images cloudstation-orchestrator:latest
```

### Running the Container

```bash
# Basic usage
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/workspace \
  cloudstation-orchestrator:latest \
  cs --version

# With Azure authentication
docker run --rm \
  -e AZURE_CLIENT_ID="your-client-id" \
  -e AZURE_CLIENT_SECRET="your-client-secret" \
  -e AZURE_TENANT_ID="your-tenant-id" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/workspace \
  cloudstation-orchestrator:latest \
  cs up --app myapp
```

### Required Environment Variables

See the [Environment Variables](#environment-variables) section for complete documentation.

**Quick Reference for Docker:**
- `USER_TOKEN` - Authentication token for CI/CD (recommended over deprecated `CS_TOKEN`)
- `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID` - Azure Container Registry authentication
- `BACKEND_URL`, `ACCESS_TOKEN` - Optional backend API integration

### Testing the Image

```bash
# Verify all required binaries are available
make docker-test

# Start an interactive shell
make docker-shell
```

### Pushing to Azure Container Registry

```bash
# Tag and push to ACR
make docker-push

# This will push to: acrbc001.azurecr.io/cloudstation-orchestrator:latest
# And also: acrbc001.azurecr.io/cloudstation-orchestrator:<version>
```

### Performance Metrics

Compared to the legacy cs-runner (Node.js/TypeScript):

| Metric | CloudStation Orchestrator | cs-runner | Improvement |
|--------|---------------------------|-----------|-------------|
| Image Size | ~250-300MB | ~600MB | 50% reduction |
| Memory Usage (idle) | ~80-100MB | ~200MB | 50% reduction |
| Startup Time | <5 seconds | ~10 seconds | 50% faster |
| Build Time | <2 minutes | ~5 minutes | 60% faster |

### Included Dependencies

The Docker image includes all external tools required by builtin plugins:
- `cs` - CloudStation Orchestrator binary
- `nixpacks` - Nixpacks builder
- `railpack` - Railway builder
- `nomad-pack` - Nomad Pack deployment tool
- `nomad` - Nomad CLI
- `docker` - Docker CLI
- `git` - Version control
- `az` - Azure CLI

### Troubleshooting

**Issue: "docker: command not found" inside container**
- Ensure `/var/run/docker.sock` is mounted: `-v /var/run/docker.sock:/var/run/docker.sock`

**Issue: "nixpacks: not found" or similar**
- Verify the image was built successfully: `make docker-build`
- Check that binaries are in PATH: `docker run --rm IMAGE which nixpacks`

**Issue: Azure authentication fails**
- Verify environment variables are set correctly
- Test credentials outside container: `az login --service-principal -u $AZURE_CLIENT_ID -p $AZURE_CLIENT_SECRET --tenant $AZURE_TENANT_ID`

**Issue: Image size is too large**
- Review installed packages
- Use `docker history cloudstation-orchestrator:latest` to identify large layers

## Development

```bash
# Run tests
make test

# Run tests with coverage
make coverage

# Format code
make fmt

# Run linter
make lint

# Build for multiple platforms
make build-all
```
End.
## License

Copyright © 2025 CloudStation. All rights reserved.
