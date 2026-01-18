# Secret Provider Architecture

CloudStation Orchestrator provides a clean, composable secret provider architecture that decouples secret management from build plugins. This allows you to inject secrets from various backends (HashiCorp Vault, AWS Secrets Manager, Azure Key Vault, etc.) into your build environment without builders needing to know about the secret source.

## Overview

The secret provider architecture follows these principles:

1. **Separation of Concerns**: Builders focus on building; secret providers handle secret retrieval
2. **Composition over Inheritance**: Secret providers are composed at the lifecycle layer
3. **Provider Agnostic**: Builders receive secrets via environment variables, regardless of source
4. **Security First**: Secrets are never logged, TLS verification is enabled by default
5. **Extensible**: Easy to add new secret providers without modifying existing code

## How It Works

```
┌─────────────────┐
│  HCL Config     │
│  (with Vault)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Lifecycle     │
│   Executor      │──────► Detect Secret Provider Config
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Secret Provider │
│   (Vault)       │──────► Authenticate & Fetch Secrets
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Enrich Config  │──────► Merge secrets into env map
│   (env map)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Builder      │──────► Build with secrets in environment
│ (nixpacks/      │
│  csdocker)      │
└─────────────────┘
```

Secret fetching and injection happens at the **lifecycle orchestration layer** before plugin configuration, keeping builders completely agnostic to secret sources.

## Supported Providers

### HashiCorp Vault

The Vault provider supports:
- **Authentication**: AppRole authentication method
- **Secret Engines**: KV v1 and KV v2
- **TLS**: Configurable TLS verification with custom CA support
- **Retry Logic**: Automatic retry with exponential backoff
- **Timeout**: Configurable request timeout

## Using Vault Provider

### Prerequisites

1. A running Vault instance
2. AppRole authentication enabled
3. A role with permissions to read secrets
4. Secrets stored in Vault KV engine

### Basic Configuration with Nixpacks

Add Vault configuration to your build block in `cloudstation.hcl`:

```hcl
project = "my-project"

app "web" {
  build {
    use = "nixpacks"
    name = "my-app"
    tag = "latest"
    context = "."

    # Vault configuration
    vault_address = "https://vault.example.com:8200"
    role_id = env("VAULT_ROLE_ID")
    secret_id = env("VAULT_SECRET_ID")
    secrets_path = "secret/data/myapp"

    # Secrets from Vault will be merged into env
    env = {
      PORT = "3000"
      NODE_ENV = "production"
    }
  }

  deploy {
    use = "noop"
  }
}
```

### Basic Configuration with Docker (csdocker)

The csdocker plugin uses Vault secrets the same way as nixpacks. Secrets are automatically passed as Docker build arguments:

```hcl
project = "my-project"

app "docker-app" {
  build {
    use = "csdocker"
    name = "my-docker-app"
    tag = "latest"
    dockerfile = "Dockerfile"
    context = "."

    # Vault configuration
    vault_address = "https://vault.example.com:8200"
    role_id = env("VAULT_ROLE_ID")
    secret_id = env("VAULT_SECRET_ID")
    secrets_path = "secret/data/docker-app"

    # Static build arguments
    build_args = {
      NODE_ENV = "production"
    }

    # Static environment variables
    # Vault secrets will be merged here and passed as --build-arg
    env = {
      PORT = "3000"
    }
  }

  deploy {
    use = "noop"
  }
}
```

**How it works with Docker:**
- Secrets from Vault are fetched by the lifecycle layer
- Secrets are merged into the `env` map
- The csdocker plugin passes all `env` values as `--build-arg` to Docker
- Your Dockerfile can access these secrets during build:

```dockerfile
# In your Dockerfile
ARG DATABASE_URL
ARG API_KEY

# Use the secrets during build
RUN echo "Building with DATABASE_URL=${DATABASE_URL}"

# Important: Don't store secrets in final image layers!
# Use multi-stage builds to ensure secrets aren't in final image
```

### Configuration Reference

| Field | Required | Description | Default |
|-------|----------|-------------|---------|
| `vault_address` | Yes | Vault server URL | - |
| `role_id` | Yes | AppRole role ID | - |
| `secret_id` | Yes | AppRole secret ID | - |
| `secrets_path` | Yes | Path to secrets in Vault | - |
| `vault_tls_skip_verify` | No | Skip TLS verification (dev only) | `false` |
| `vault_ca_cert` | No | Path to CA certificate | - |
| `vault_timeout` | No | Request timeout (seconds or duration) | `30s` |
| `vault_max_retries` | No | Maximum retry attempts | `3` |

### Environment Variables

You can use environment variables for sensitive values:

```hcl
vault_address = env("VAULT_ADDR")
role_id = env("VAULT_ROLE_ID")
secret_id = env("VAULT_SECRET_ID")
```

Then set them before running:

```bash
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_ROLE_ID="your-role-id"
export VAULT_SECRET_ID="your-secret-id"

cs build --app web
```

### TLS Configuration

#### Production (Verify TLS)

```hcl
vault_address = "https://vault.example.com:8200"
vault_tls_skip_verify = false  # default, TLS verification enabled
```

#### Development (Skip Verification)

```hcl
vault_address = "https://vault-dev.local:8200"
vault_tls_skip_verify = true  # skip TLS verification (logs warning)
```

#### Custom CA Certificate

```hcl
vault_address = "https://vault.internal:8200"
vault_ca_cert = "/path/to/ca.pem"
```

### Vault Secret Paths

#### KV Version 2 (Default)

For KV v2, use the `/data/` path segment:

```hcl
secrets_path = "secret/data/myapp"
```

This reads from `secret/myapp` in Vault's KV v2 engine.

#### KV Version 1

For KV v1, omit the `/data/` segment:

```hcl
secrets_path = "secret/myapp"
```

The provider automatically handles both KV v1 and v2 formats.

## Setting Up Vault

### 1. Enable AppRole Authentication

```bash
vault auth enable approle
```

### 2. Create a Policy

Create a policy file `myapp-policy.hcl`:

```hcl
path "secret/data/myapp" {
  capabilities = ["read"]
}
```

Apply the policy:

```bash
vault policy write myapp-policy myapp-policy.hcl
```

### 3. Create an AppRole

```bash
vault write auth/approle/role/myapp \
    secret_id_ttl=24h \
    token_ttl=1h \
    token_max_ttl=4h \
    policies="myapp-policy"
```

### 4. Get Role ID

```bash
vault read auth/approle/role/myapp/role-id
```

Output:
```
Key        Value
---        -----
role_id    a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### 5. Generate Secret ID

```bash
vault write -f auth/approle/role/myapp/secret-id
```

Output:
```
Key                   Value
---                   -----
secret_id             x9y8z7w6-v5u4-3210-zyxw-vu9876543210
secret_id_accessor    ...
secret_id_ttl         24h
```

### 6. Store Secrets

For KV v2:

```bash
vault kv put secret/myapp \
    DATABASE_URL="postgres://localhost:5432/mydb" \
    API_KEY="sk-1234567890abcdef" \
    STRIPE_SECRET="sk_test_1234567890"
```

For KV v1:

```bash
vault kv put -mount=secret myapp \
    DATABASE_URL="postgres://localhost:5432/mydb" \
    API_KEY="sk-1234567890abcdef"
```

## Security Best Practices

### 1. Never Hardcode Secrets

❌ **Bad**:
```hcl
secret_id = "x9y8z7w6-v5u4-3210-zyxw-vu9876543210"
```

✅ **Good**:
```hcl
secret_id = env("VAULT_SECRET_ID")
```

### 2. Enable TLS Verification

❌ **Bad** (production):
```hcl
vault_tls_skip_verify = true
```

✅ **Good**:
```hcl
vault_tls_skip_verify = false  # or omit entirely
```

### 3. Use Least Privilege

Only grant read access to specific secret paths:

```hcl
# Good - specific path
path "secret/data/myapp" {
  capabilities = ["read"]
}

# Bad - too permissive
path "secret/*" {
  capabilities = ["read", "list"]
}
```

### 4. Rotate Secrets Regularly

- Rotate AppRole secret IDs regularly
- Set appropriate TTLs for tokens
- Use short-lived secrets when possible

### 5. Monitor and Audit

- Enable Vault audit logging
- Monitor secret access patterns
- Alert on unusual access

## Troubleshooting

### Authentication Failed

**Error**: `vault provider: authenticate failed: ...`

**Possible Causes**:
- Invalid role ID or secret ID
- AppRole not configured correctly
- Vault server unreachable

**Solution**:
1. Verify role ID: `vault read auth/approle/role/myapp/role-id`
2. Generate new secret ID: `vault write -f auth/approle/role/myapp/secret-id`
3. Check Vault server is accessible: `curl $VAULT_ADDR/v1/sys/health`

### Secret Not Found

**Error**: `vault provider: read_secret failed: secret not found at path: ...`

**Possible Causes**:
- Incorrect secret path
- KV v1 vs v2 path mismatch
- Secret doesn't exist

**Solution**:
1. List secrets: `vault kv list secret/`
2. Read secret directly: `vault kv get secret/myapp`
3. Check KV version: `vault secrets list -detailed`
4. Use correct path format (with `/data/` for KV v2)

### TLS Verification Failed

**Error**: `request failed: x509: certificate signed by unknown authority`

**Possible Causes**:
- Self-signed certificate
- Custom CA not configured

**Solution**:
1. For development: Set `vault_tls_skip_verify = true`
2. For production: Provide CA certificate with `vault_ca_cert = "/path/to/ca.pem"`

### Connection Timeout

**Error**: `request failed: context deadline exceeded`

**Possible Causes**:
- Vault server unreachable
- Network issues
- Firewall blocking connection

**Solution**:
1. Check network connectivity: `ping vault.example.com`
2. Verify Vault is running: `vault status`
3. Increase timeout: `vault_timeout = 60`
4. Check firewall rules

## Advanced Usage

### Multiple Secret Paths

Currently, only one `secrets_path` is supported per build. To use secrets from multiple paths, you can:

1. **Flatten secrets in Vault**: Store all secrets for an app in one path
2. **Use Vault policy templates**: Create a role that can read multiple paths
3. **Wait for multi-path support**: This feature is planned for a future release

### Secret Overrides

Secrets from Vault are **additive** and will **not override** existing environment variables:

```hcl
env = {
  PORT = "3000"           # This will NOT be overridden by Vault
  NODE_ENV = "production" # This will NOT be overridden by Vault
}
```

If both `env` and Vault contain the same key, the `env` value takes precedence.

### Conditional Secrets

Use HCL conditionals to enable Vault only in certain environments:

```hcl
locals {
  use_vault = env("ENVIRONMENT") == "production"
}

app "web" {
  build {
    use = "nixpacks"

    vault_address = local.use_vault ? "https://vault.example.com:8200" : ""
    role_id = local.use_vault ? env("VAULT_ROLE_ID") : ""
    secret_id = local.use_vault ? env("VAULT_SECRET_ID") : ""
    secrets_path = local.use_vault ? "secret/data/myapp" : ""
  }
}
```

## Future Enhancements

The following features are planned for future releases:

- [ ] AWS Secrets Manager provider
- [ ] Azure Key Vault provider
- [ ] Google Cloud Secret Manager provider
- [ ] File-based secret provider
- [ ] Multi-provider support (merge secrets from multiple providers)
- [ ] Secret caching for repeated builds
- [ ] Secret versioning support
- [ ] Secret rotation during long builds
- [ ] Secret filtering (fetch only specific keys)
- [ ] Secret transformation (rename keys, transform values)

## Builder-Specific Notes

### Nixpacks Builder

The nixpacks builder passes secrets as environment variables using the `--env` flag:

```bash
nixpacks build . --env DATABASE_URL=... --env API_KEY=...
```

Secrets are available as environment variables during the build process.

### Docker Builder (csdocker)

The csdocker builder passes secrets as Docker build arguments using the `--build-arg` flag:

```bash
docker build . --build-arg DATABASE_URL=... --build-arg API_KEY=...
```

**Important Docker Security Notes:**

1. **Build Arguments are Not Secret**: Docker build args are visible in image metadata and build logs. Use them only for build-time configuration, not runtime secrets.

2. **Use Multi-Stage Builds**: Don't let secrets persist in final image layers:

```dockerfile
# Build stage - secrets are OK here
FROM node:20 AS builder
ARG DATABASE_URL
ARG API_KEY

RUN echo "Building with API_KEY=${API_KEY}"
RUN npm install
RUN npm run build

# Final stage - no secrets here!
FROM node:20-slim
COPY --from=builder /app/dist /app/dist
# Secrets are NOT in this layer
```

3. **Docker BuildKit Secrets**: For production, consider using Docker BuildKit's secret mounting feature instead of build args:

```dockerfile
# Mount secrets without exposing them
RUN --mount=type=secret,id=api_key \
    API_KEY=$(cat /run/secrets/api_key) npm run build
```

Then build with:
```bash
docker buildx build --secret id=api_key,env=API_KEY .
```

## Examples

See the `examples/` directory for complete examples:

- `examples/vault-example.hcl` - Basic Vault integration with nixpacks
- `examples/csdocker-example.hcl` - Docker builds with Vault secrets
- `examples/vault-tls.hcl` - Vault with TLS configuration
- `examples/multi-app-vault.hcl` - Multiple apps with different Vault configs

## Migration Guide

If you're migrating from the old approach (Vault logic in builders):

1. **Update HCL config**: Vault fields stay in the build block (no changes needed)
2. **Remove vaultlib dependency**: No longer needed in your codebase
3. **Rebuild**: Use the new cloudstation-orchestrator binary
4. **Verify**: Secrets should be injected automatically via env map

The migration is **backward compatible** - existing configurations will continue to work.
