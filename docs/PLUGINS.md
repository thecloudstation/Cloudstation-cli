# Plugin Development Guide

## Overview

CloudStation Orchestrator uses a builtin plugin system where all plugins are compiled directly into the binary. This document explains how plugins work and how to create new ones.

## Plugin Architecture

### Component Types

There are four types of components a plugin can provide:

1. **Builder** - Builds artifacts from source code
2. **Registry** - Pushes/pulls artifacts to/from registries
3. **Platform** - Deploys artifacts to platforms
4. **ReleaseManager** - Manages releases and traffic

A plugin can provide one or more of these components.

### Component Interfaces

All components must implement the `Configurable` interface:

```go
type Configurable interface {
    Config() (interface{}, error)
    ConfigSet(config interface{}) error
}
```

#### Builder Interface

```go
type Builder interface {
    Build(ctx context.Context) (*artifact.Artifact, error)
    Config() (interface{}, error)
    ConfigSet(config interface{}) error
}
```

#### Registry Interface

```go
type Registry interface {
    Push(ctx context.Context, artifact *artifact.Artifact) (*artifact.RegistryRef, error)
    Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error)
    Config() (interface{}, error)
    ConfigSet(config interface{}) error
}
```

#### Platform Interface

```go
type Platform interface {
    Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error)
    Destroy(ctx context.Context, deploymentID string) error
    Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error)
    Config() (interface{}, error)
    ConfigSet(config interface{}) error
}
```

## Creating a New Plugin

### Step 1: Create Package Structure

Create a new directory under `builtin/`:

```bash
mkdir -p builtin/myplugin
```

### Step 2: Implement Components

Create your plugin file (e.g., `builtin/myplugin/plugin.go`):

```go
package myplugin

import (
    "context"
    "github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/component"
)

// Builder implementation
type Builder struct {
    config *BuilderConfig
}

type BuilderConfig struct {
    // Your configuration fields
    SomeOption string
    AnotherOption int
}

func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    // Implement your build logic here
    return &artifact.Artifact{
        ID: "my-artifact",
        Image: "myimage:latest",
        // ...
    }, nil
}

func (b *Builder) Config() (interface{}, error) {
    return b.config, nil
}

func (b *Builder) ConfigSet(config interface{}) error {
    // Handle configuration from HCL
    b.config = &BuilderConfig{}

    if configMap, ok := config.(map[string]interface{}); ok {
        if option, ok := configMap["some_option"].(string); ok {
            b.config.SomeOption = option
        }
    }

    return nil
}
```

### Step 3: Register the Plugin

Add an `init()` function to register your plugin:

```go
func init() {
    plugin.Register("myplugin", &plugin.Plugin{
        Builder: &Builder{config: &BuilderConfig{}},
    })
}
```

### Step 4: Import in Main

Add your plugin to `cmd/cloudstation/main.go`:

```go
import (
    // ...
    _ "github.com/thecloudstation/cloudstation-orchestrator/builtin/myplugin"
)
```

### Step 5: Use in Configuration

```hcl
app "myapp" {
  build {
    use = "myplugin"
    some_option = "value"
    another_option = 42
  }
}
```

## Example: Simple Builder Plugin

```go
package simple

import (
    "context"
    "fmt"
    "time"

    "github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

type Builder struct {
    config *Config
}

type Config struct {
    Message string
}

func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    fmt.Printf("Building with message: %s\n", b.config.Message)

    return &artifact.Artifact{
        ID: "simple-artifact",
        Image: "simple:latest",
        Tag: "latest",
        BuildTime: time.Now(),
        Metadata: map[string]interface{}{
            "message": b.config.Message,
        },
    }, nil
}

func (b *Builder) Config() (interface{}, error) {
    return b.config, nil
}

func (b *Builder) ConfigSet(config interface{}) error {
    b.config = &Config{}

    if configMap, ok := config.(map[string]interface{}); ok {
        if msg, ok := configMap["message"].(string); ok {
            b.config.Message = msg
        }
    }

    return nil
}

func init() {
    plugin.Register("simple", &plugin.Plugin{
        Builder: &Builder{config: &Config{}},
    })
}
```

## Builtin Plugins Reference

### csdocker

**Type:** Builder
**Purpose:** Docker builds with Vault integration

**Configuration:**
```hcl
build {
  use = "csdocker"
  vault_address = "https://vault.example.com:8200"
  role_id = env("VAULT_ROLE_ID")
  secret_id = env("VAULT_SECRET_ID")
  secrets_path = "secret/data/app"
}
```

### nixpacks

**Type:** Builder
**Purpose:** Builds Docker images from source code using Nixpacks CLI

**Prerequisites:**
- Nixpacks must be installed on the system running cloudstation-orchestrator
- macOS: `brew install nixpacks`
- Linux/macOS with Cargo: `cargo install nixpacks`

**Configuration:**

Required fields:
- `name` - Docker image name (required)

Optional fields:
- `tag` - Docker image tag (default: "latest")
- `context` - Build directory path (default: ".")
- `build_args` - Map of build arguments to pass to nixpacks
- `env` - Map of environment variables to pass to the build

**Examples:**

Basic usage:
```hcl
build {
  use = "nixpacks"
  name = "my-app"
  tag = "latest"
  context = "."
}
```

With build arguments and environment variables:
```hcl
build {
  use = "nixpacks"
  name = "my-app"
  tag = "v1.2.3"
  context = "./services/api"

  build_args = {
    NODE_ENV = "production"
  }

  env = {
    PORT = "3000"
    API_URL = "https://api.example.com"
  }
}
```

Future (Phase 2) - Vault integration:
```hcl
build {
  use = "nixpacks"
  name = "my-app"
  vault_address = "https://vault.example.com:8200"
  role_id = env("VAULT_ROLE_ID")
  secret_id = env("VAULT_SECRET_ID")
  secrets_path = "secret/data/app"
}
```

### noop

**Type:** Builder + Registry + Platform + ReleaseManager
**Purpose:** Complete no-op plugin for testing the full deployment lifecycle

The noop plugin provides stub implementations for all component types, allowing you to test the complete Build → Registry → Deploy → Release pipeline without performing any actual operations. This is ideal for:
- Testing the orchestrator's lifecycle execution flow
- Validating configuration parsing for all component types
- Debugging integration issues without side effects
- Providing a reference implementation for plugin developers

**Configuration:**

Full lifecycle example:
```hcl
app "noop-test" {
  build {
    use = "noop"
    message = "Building with noop"  # Optional
  }

  registry {
    use = "noop"
    # Returns fake registry reference: noop-registry/noop:latest
  }

  deploy {
    use = "noop"
    # Returns deployment in "running" state with "healthy" health
  }

  release {
    use = "noop"
    message = "Releasing with noop"  # Optional
  }
}
```

Individual phase usage:
```hcl
build {
  use = "noop"
  message = "Testing"  # Optional
}
```

Run the full lifecycle with: `cs up --app noop-test`

### docker

**Type:** Builder + Registry + Platform
**Purpose:** Standard Docker operations including building images and pushing to container registries

#### Builder Component

The Docker builder is currently a stub implementation. For actual Docker builds, use the `csdocker` plugin.

**Configuration:**
```hcl
build {
  use = "docker"
  dockerfile = "Dockerfile"
  context = "."
}
```

#### Registry Component

The Docker registry component enables pushing Docker images to container registries such as Azure Container Registry (ACR), Docker Hub, or private registries. This restores functionality from the legacy cs-runner system where images were automatically pushed to ACR after building.

**Prerequisites:**
- Docker must be installed and running on the system
- Valid credentials for the target registry

**Configuration:**

Required fields:
- `image` - Full image name with registry prefix (e.g., "acrbc001.azurecr.io/myapp")
- `tag` - Image tag to push (e.g., "latest", "v1.0.0")
- `username` - Registry username or service principal ID
- `password` - Registry password, token, or service principal secret

Optional fields:
- `registry` - Explicit registry URL (extracted from image if not provided)

**Examples:**

Basic Azure Container Registry push:
```hcl
registry {
  use = "docker"
  image = "acrbc001.azurecr.io/myapp"
  tag = "latest"
  username = var.registry_username
  password = var.registry_password
}
```

Docker Hub push:
```hcl
registry {
  use = "docker"
  image = "docker.io/myusername/myapp"
  tag = "v1.0.0"
  username = var.registry_username
  password = var.registry_password
}
```

Private registry with explicit registry URL:
```hcl
registry {
  use = "docker"
  image = "myapp"
  tag = "latest"
  registry = "registry.company.com:5000"
  username = env("REGISTRY_USER")
  password = env("REGISTRY_PASS")
}
```

Alternative nested auth syntax:
```hcl
registry {
  use = "docker"
  image = "myregistry.azurecr.io/myapp"
  tag = "latest"
  auth {
    username = env("ACR_USERNAME")
    password = env("ACR_PASSWORD")
  }
}
```

**Authentication Flow:**
1. Dispatcher loads credentials from Vault → environment variables
2. HCL generator references credentials as var.registry_username, var.registry_password
3. Config parser loads these from environment when parsing the HCL
4. Registry plugin receives credentials in ConfigSet()
5. Registry plugin executes `docker login` with credentials
6. Registry plugin executes `docker push` after successful login

**Security Best Practices:**
- NEVER hardcode credentials in HCL files
- Use variable references (var.xxx) or environment variables (env())
- Mark credential variables as `sensitive = true`
- Credentials should be loaded from Vault via dispatcher.hcl
- The dispatcher Vault template automatically injects registry_username and registry_password from Vault path `/acr/data/registry`

**Error Handling:**

The registry plugin provides clear error messages for common issues:
- Authentication failures: Check credentials and registry URL
- Network errors: Verify network connectivity to registry
- Permission errors: Ensure credentials have push permissions
- Missing credentials: Ensure username and password are provided

**Registry URL Formats:**
- Azure Container Registry: `<name>.azurecr.io`
- Docker Hub: `docker.io` or omit (defaults to Docker Hub)
- Private registry: `<hostname>:<port>` (port optional)
- Local registry: `localhost:<port>`

**Features:**
- Supports Azure Container Registry (ACR)
- Supports Docker Hub (public and private)
- Supports private/self-hosted registries
- Automatic docker login with credentials
- Image tagging with custom tags
- Digest extraction from push output
- Context cancellation for long-running operations
- Structured logging with credential redaction

**Troubleshooting:**

*Docker not available:*
```bash
# Verify Docker is installed and running
docker version
```

*Authentication failures:*
- Verify credentials are correct
- For ACR, ensure service principal has AcrPush role
- Check registry URL is correct and accessible

*Push failures:*
- Ensure image was built successfully before pushing
- Verify network connectivity to registry
- Check disk space for image export

*Missing credentials:*
- Ensure REGISTRY_USERNAME and REGISTRY_PASSWORD environment variables are set
- Verify dispatcher.hcl Vault template is loading credentials correctly

#### Platform Component

The Docker platform component is currently a stub implementation. For actual deployments, use the `nomadpack` plugin.

### nomadpack

**Type:** Platform
**Purpose:** Deploy applications to Nomad clusters using Nomad Pack templates

**Prerequisites:**
- Nomad Pack CLI must be installed on the system running cloudstation-orchestrator
  - macOS: `brew tap hashicorp/tap && brew install nomad-pack`
  - Linux: Download from [GitHub releases](https://github.com/hashicorp/nomad-pack)
- Access to a running Nomad cluster
- Access to a Nomad Pack registry (public or private)

**Configuration:**

Required fields:
- `deployment_name` - Name for the Nomad Pack deployment
- `pack` - Name of the pack to deploy (e.g., "nginx", "redis", "webapp")
- `registry_name` - Name for the Nomad Pack registry
- `registry_source` - Git URL for the pack registry

Optional fields:
- `registry_ref` - Specific git ref/tag/branch (e.g., "v0.0.1", "main")
- `registry_target` - Specific pack within the registry (e.g., "packs/nginx")
- `registry_token` - Personal access token for private registries
- `nomad_addr` - Nomad API address (e.g., "http://localhost:4646")
- `nomad_token` - Nomad ACL token for authentication
- `variables` - Map of variable overrides for the pack
- `variable_files` - List of paths to variable files

**Examples:**

Basic deployment:
```hcl
deploy {
  use = "nomadpack"
  deployment_name = "nginx-app"
  pack = "nginx"
  registry_name = "community"
  registry_source = "https://github.com/hashicorp/nomad-pack-community-registry"
  nomad_addr = "http://localhost:4646"
}
```

With variables:
```hcl
deploy {
  use = "nomadpack"
  deployment_name = "webapp-prod"
  pack = "webapp"
  registry_name = "community"
  registry_source = "https://github.com/hashicorp/nomad-pack-community-registry"
  nomad_addr = env("NOMAD_ADDR")
  nomad_token = env("NOMAD_TOKEN")

  variables = {
    port = "8080"
    replicas = "3"
    memory = "512"
    cpu = "500"
  }
}
```

With variable files:
```hcl
deploy {
  use = "nomadpack"
  deployment_name = "api-prod"
  pack = "webapp"
  registry_name = "myorg"
  registry_source = "https://github.com/myorg/nomad-packs"
  nomad_addr = env("NOMAD_ADDR")
  nomad_token = env("NOMAD_TOKEN")

  variable_files = [
    "./config/base.hcl",
    "./config/production.hcl"
  ]
}
```

Private registry with authentication:
```hcl
deploy {
  use = "nomadpack"
  deployment_name = "secure-app"
  pack = "webapp"
  registry_name = "private"
  registry_source = "https://github.com/myorg/private-packs"
  registry_token = env("GITHUB_TOKEN")  # Personal access token
  registry_ref = "v1.2.3"               # Specific version
  nomad_addr = env("NOMAD_ADDR")
  nomad_token = env("NOMAD_TOKEN")

  variables = {
    environment = "production"
    replicas = "10"
    domain = "app.example.com"
  }
}
```

**Features:**
- Full lifecycle management (deploy, destroy, status)
- Support for public and private pack registries
- Variable overrides via inline variables or variable files
- Authentication for private registries using personal access tokens
- Nomad ACL token support for secure clusters
- Version pinning with registry refs
- Context cancellation support for long-running operations

**Troubleshooting:**

*Command not found:*
```bash
# Verify nomad-pack is installed
nomad-pack version

# Install if missing (macOS)
brew tap hashicorp/tap && brew install nomad-pack
```

*Authentication failures:*
- For private registries, ensure `registry_token` is set to a valid GitHub personal access token
- For Nomad clusters with ACLs enabled, ensure `nomad_token` has appropriate permissions

*Pack not found:*
- Verify the pack name exists in the registry
- Check `registry_source` URL is correct
- Try specifying `registry_ref` to use a specific version

*Deployment failures:*
- Check Nomad cluster logs: `nomad monitor`
- Verify Nomad cluster is reachable at `nomad_addr`
- Ensure pack variables match the pack's expected schema

## Best Practices

### 1. Configuration Handling

Always provide defaults and validate configuration:

```go
func (b *Builder) ConfigSet(config interface{}) error {
    // Initialize with defaults
    b.config = &BuilderConfig{
        DefaultValue: "default",
    }

    configMap, ok := config.(map[string]interface{})
    if !ok {
        return nil  // Use defaults
    }

    // Parse and validate
    if value, ok := configMap["required_field"].(string); ok {
        if value == "" {
            return fmt.Errorf("required_field cannot be empty")
        }
        b.config.RequiredField = value
    } else {
        return fmt.Errorf("required_field is required")
    }

    return nil
}
```

### 2. Error Handling

Provide clear, actionable error messages:

```go
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    if err := b.validate(); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }

    result, err := b.executeBuild(ctx)
    if err != nil {
        return nil, fmt.Errorf("build execution failed: %w", err)
    }

    return result, nil
}
```

### 3. Context Usage

Always respect context cancellation:

```go
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Perform build...

    return artifact, nil
}
```

### 4. Logging

Use structured logging with the context logger:

```go
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    logger := hclog.FromContext(ctx)
    logger.Debug("starting build", "config", b.config)

    // Build logic...

    logger.Info("build completed", "artifact_id", artifact.ID)
    return artifact, nil
}
```

### 5. Secrets Management

Never log secrets:

```go
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
    // DON'T DO THIS:
    // logger.Debug("config", "password", b.config.Password)

    // DO THIS:
    logger.Debug("config", "vault_address", b.config.VaultAddress)

    return artifact, nil
}
```

## Testing Plugins

Create tests for your plugin:

```go
package myplugin

import (
    "context"
    "testing"
)

func TestBuilder_Build(t *testing.T) {
    builder := &Builder{
        config: &BuilderConfig{
            SomeOption: "test",
        },
    }

    artifact, err := builder.Build(context.Background())
    if err != nil {
        t.Fatalf("Build failed: %v", err)
    }

    if artifact.ID == "" {
        t.Error("Expected artifact ID to be set")
    }
}

func TestBuilder_ConfigSet(t *testing.T) {
    builder := &Builder{}

    config := map[string]interface{}{
        "some_option": "value",
    }

    err := builder.ConfigSet(config)
    if err != nil {
        t.Fatalf("ConfigSet failed: %v", err)
    }

    if builder.config.SomeOption != "value" {
        t.Errorf("Expected SomeOption to be 'value', got %s", builder.config.SomeOption)
    }
}
```

## Debugging

Enable debug logging:

```bash
cs --log-level debug build --app myapp
```

This will show detailed logs from the plugin execution.
