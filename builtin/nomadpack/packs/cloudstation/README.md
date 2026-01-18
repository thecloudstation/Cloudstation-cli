# CloudStation Pack (v2 Syntax)

A modern Nomad Pack for deploying standalone service instances using Nomad Pack v2 syntax. This pack provides a streamlined way to deploy containerized applications with full support for service discovery, health checks, and resource management.

## Key Features

- **Modern Nomad Pack v2 Syntax** - No `--parser-v1` flag required
- **Drop-in Replacement** - Functionally equivalent to the legacy `cloud_service` pack
- **Better Error Messages** - Enhanced validation and clearer error reporting
- **Improved Maintainability** - Cleaner template syntax and better variable handling
- **Full Feature Support** - Supports all CloudStation deployment patterns including services, batch jobs, and periodic tasks

## Usage

### Basic Commands

```bash
# Render the pack to preview the generated Nomad job
nomad-pack render packs/cloudstation -f vars.hcl

# Deploy the pack to your Nomad cluster
nomad-pack run packs/cloudstation -f vars.hcl

# Update an existing deployment
nomad-pack run packs/cloudstation -f vars.hcl --name my-service

# Stop and remove a deployment
nomad-pack destroy packs/cloudstation --name my-service
```

### With Environment-Specific Variables

```bash
# Development environment
nomad-pack run packs/cloudstation \
  -f environments/dev/vars.hcl \
  --var image="myapp:dev"

# Production environment with overrides
nomad-pack run packs/cloudstation \
  -f environments/prod/vars.hcl \
  --var count=3 \
  --var resources.cpu=2000
```

## Important Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `job_name` | string | Pack name | Override the default job name |
| `image` | string | `mnomitch/hello_world_server` | Docker image to deploy |
| `count` | number | `1` | Number of instances to run |
| `datacenters` | list(string) | `["*"]` | Target datacenters for deployment |
| `region` | string | `""` | Nomad region for deployment |
| `node_pool` | string | `minions` | Node pool for task placement |
| `resources` | object | See example | CPU, memory, and network resources |
| `consul_service_name` | string | Job name | Service name for Consul registration |
| `consul_service_tags` | list(string) | `[]` | Tags for service discovery |
| `ports` | object | `{}` | Port configurations for the service |
| `env_vars` | map(string) | `{}` | Environment variables for the container |
| `volumes` | list(object) | `[]` | Volume mounts for persistent storage |

## Consul Service Tags and Domain Configuration

CloudStation automatically generates Consul service tags for load balancer routing and service discovery. The tag format depends on the service type and network configuration.

### Public Services

Public services receive tags with full FQDN (Fully Qualified Domain Name) for proper routing:

- **`urlprefix-<domain>.<cluster_domain>`** - Routes HTTP/HTTPS traffic to the service
- **`custom-<custom_domain>`** - Routes to user-provided custom domain
- **`tcp-lb`** - Marks TCP services for TCP load balancing

#### Example: HTTP Service with Domain

For a service with `network.domain = "myapp"` and `cluster_domain = "cloud-station.io"`:

```hcl
tags = [
  "blue",
  "urlprefix-myapp.cloud-station.io"
]

canary_tags = [
  "green",
  "urlprefix-canary-myapp.cloud-station.io"
]
```

The load balancer (Fabio/Traefik) uses these tags to route requests from `https://myapp.cloud-station.io` to your service.

### Custom Domains

When using custom domains, tags preserve the user-provided domain exactly:

```hcl
network = [{
  name          = "http"
  port          = 8080
  type          = "http"
  public        = true
  custom_domain = "app.example.com"
}]

# Generated tags:
tags = [
  "blue",
  "custom-app.example.com"
]
```

### Internal Services

Non-public services receive internal-only tags:

```hcl
tags = ["urlprefix-myservice.internal.cloud-station.io"]
```

### TCP Services

TCP services receive both FQDN and tcp-lb tags:

```hcl
network = [{
  name   = "tcp"
  port   = 5432
  type   = "tcp"
  public = true
  domain = "postgres"
}]

# Generated tags:
tags = [
  "blue",
  "urlprefix-postgres.cloud-station.io",
  "tcp-lb"
]
```

### Cluster Domain Configuration

The `cluster_domain` variable defaults to `"cloud-station.app"` but can be overridden in your vars.hcl:

```hcl
cluster_domain = "cloud-station.io"

network = [{
  name   = "http"
  port   = 8080
  type   = "http"
  public = true
  domain = "api"
}]
```

This generates the tag `"urlprefix-api.cloud-station.io"` for proper SSL/TLS certificate validation and multi-tenancy support.

## Testing

### 1. Validate Your Configuration

```bash
# Render the pack locally
nomad-pack render packs/cloudstation -f vars.hcl -o rendered-job.nomad

# Validate the rendered job
nomad job validate rendered-job.nomad

# Plan the deployment (dry-run)
nomad job plan rendered-job.nomad
```

### 2. Test Deployment

```bash
# Deploy with test configuration
nomad-pack run packs/cloudstation -f test-vars.hcl --name test-deployment

# Check deployment status
nomad job status test-deployment

# View logs
nomad alloc logs -f $(nomad job allocs test-deployment -json | jq -r '.[0].ID')

# Clean up test deployment
nomad-pack destroy packs/cloudstation --name test-deployment
```

## Migration from cloud_service Pack

The `cloudstation` pack is a modern replacement for the legacy `cloud_service` pack with identical functionality but using Nomad Pack v2 syntax.

### Key Differences

| Aspect | cloud_service (v1) | cloudstation (v2) |
|--------|-------------------|------------------|
| **Syntax** | Requires `--parser-v1` flag | Native v2 syntax |
| **Template Engine** | Legacy HCL1 templates | Modern HCL2 templates |
| **Error Messages** | Basic validation | Enhanced error reporting |
| **Variable Handling** | Limited type support | Full HCL2 type system |

### Migration Steps

1. **Update your commands** - Remove the `--parser-v1` flag:
   ```bash
   # Old (cloud_service)
   nomad-pack render packs/cloud_service --var-file="vars.hcl" --parser-v1

   # New (cloudstation)
   nomad-pack render packs/cloudstation -f vars.hcl
   ```

2. **Review your variables file** - The variable format remains the same, but v2 provides better validation:
   ```hcl
   # vars.hcl works with both packs
   job_name = "my-service"
   image    = "mycompany/myapp:v1.2.3"
   count    = 2
   ```

3. **Test the migration**:
   ```bash
   # Render both and compare
   nomad-pack render packs/cloud_service --var-file="vars.hcl" --parser-v1 -o old.nomad
   nomad-pack render packs/cloudstation -f vars.hcl -o new.nomad
   diff old.nomad new.nomad
   ```

4. **Deploy with the new pack**:
   ```bash
   nomad-pack run packs/cloudstation -f vars.hcl --name my-service
   ```

## Example Variables File

### Minimal Configuration

```hcl
# vars.hcl - Minimal example
job_name = "hello-world"
image    = "nginx:alpine"
```

### Full Service Configuration

```hcl
# vars.hcl - Complete service example
job_name = "web-api"
image    = "mycompany/api:v2.1.0"
count    = 3

datacenters = ["dc1", "dc2"]
region      = "us-west"
node_pool   = "production"

resources = {
  cpu    = 500  # MHz
  memory = 512  # MB
  network = {
    mbits = 10
  }
}

consul_service_name = "api"
consul_service_tags = ["http", "api", "v2"]

ports = {
  http = {
    static = 8080
    to     = 8080
  }
}

env_vars = {
  LOG_LEVEL     = "info"
  DATABASE_URL  = "postgresql://db:5432/myapp"
  ENVIRONMENT   = "production"
}

volumes = [
  {
    name        = "data"
    type        = "host"
    source      = "/opt/data"
    destination = "/app/data"
    read_only   = false
  }
]

health_check = {
  type     = "http"
  path     = "/health"
  interval = "10s"
  timeout  = "2s"
}
```

### Batch Job Configuration

```hcl
# batch-job.hcl - Batch/periodic job example
job_name = "data-processor"
image    = "mycompany/processor:latest"

job_config = {
  type             = "batch"
  cron             = "0 */6 * * *"  # Every 6 hours
  prohibit_overlap = true
  payload          = ""
  meta_required    = ["tenant_id", "job_id"]
}

resources = {
  cpu    = 2000
  memory = 4096
}
```

### Parameterized Job Configuration (Dispatcher)

Parameterized jobs allow you to create job templates that can be dispatched on-demand with runtime parameters. This is ideal for:
- AI agent workloads that receive task-specific parameters
- Development environments provisioned per-user
- Ephemeral sandboxes with custom configurations
- Data processing jobs with per-run settings

The cloudstation pack supports parameterized dispatcher jobs with these variables:

| Variable | Type | Description |
|----------|------|-------------|
| `job_config.meta_optional` | list(string) | Optional metadata parameters that can be provided at dispatch time |
| `restart.attempts` | number | Number of restart attempts (set to 0 for batch jobs) |
| `restart.interval` | string | Time window for restart attempts |
| `restart.delay` | string | Delay before restarting |
| `restart.mode` | string | Restart mode: "fail", "delay", or "forbid" |
| `ephemeral_disk.enabled` | bool | Enable ephemeral disk for temporary storage |
| `ephemeral_disk.migrate` | bool | Migrate disk data on reschedule |
| `ephemeral_disk.size` | number | Disk size in MB |
| `ephemeral_disk.sticky` | bool | Keep disk on the same client |
| `privileged` | bool | Enable Docker privileged mode |
| `user` | string | User to run the task as (default: "0" for root) |

#### Example Dispatcher Job

```hcl
# dispatcher-job.hcl - On-demand AI agent dispatcher
job_name = "ai-agent-dispatcher"
image    = "mycompany/ai-agent:latest"

job_config = {
  type             = "batch"
  cron             = ""
  prohibit_overlap = false
  payload          = "optional"
  meta_required    = ["project_id", "user_id"]
  meta_optional    = ["agent_type", "model", "temperature"]
}

# Batch jobs should fail fast, not retry
restart = {
  attempts = 0
  interval = "1m"
  delay    = "5s"
  mode     = "fail"
}

# Enable ephemeral disk for job artifacts
ephemeral_disk = {
  enabled = true
  migrate = false
  size    = 1000
  sticky  = false
}

# Enable privileged mode if needed for Docker-in-Docker
privileged = false

# Run as non-root for security
user = "1000"

resources = {
  cpu        = 2000
  memory     = 4096
  gpu        = 0
  memory_max = 8192
}
```

#### Deploying and Dispatching

```bash
# 1. Render and deploy the dispatcher job template
nomad-pack render packs/cloudstation -f dispatcher-job.hcl -o /tmp/dispatcher.nomad
nomad job run /tmp/dispatcher.nomad

# 2. Verify the job is registered as parameterized
nomad job status ai-agent-dispatcher
# Should show: Type = batch (parameterized)

# 3. Dispatch a job with runtime parameters
nomad job dispatch ai-agent-dispatcher \
  project_id=prj-12345 \
  user_id=usr-67890 \
  agent_type=research \
  model=gpt-4 \
  temperature=0.7

# 4. Monitor the dispatched job
nomad job status -verbose ai-agent-dispatcher
nomad alloc logs <alloc-id>

# 5. Dispatch more jobs as needed - each gets unique parameters
nomad job dispatch ai-agent-dispatcher \
  project_id=prj-99999 \
  user_id=usr-11111 \
  agent_type=coding
```

#### Accessing Parameters in Your Application

Nomad provides dispatched parameters as environment variables with the `NOMAD_META_` prefix:

```bash
# Inside your container
echo $NOMAD_META_PROJECT_ID      # prj-12345
echo $NOMAD_META_USER_ID         # usr-67890
echo $NOMAD_META_AGENT_TYPE      # research
echo $NOMAD_META_MODEL           # gpt-4
echo $NOMAD_META_TEMPERATURE     # 0.7
```

These variables are automatically available to your application and can be used to configure runtime behavior.

## Troubleshooting

### Common Issues

1. **Variable validation errors** - V2 provides stricter type checking. Ensure your variables match the expected types.

2. **Template rendering errors** - Check for HCL2 syntax in your variables file:
   ```bash
   nomad-pack render packs/cloudstation -f vars.hcl --debug
   ```

3. **Missing dependencies** - Ensure Nomad Pack is updated:
   ```bash
   nomad-pack version  # Should be 0.1.0 or higher for v2 support
   ```

## Support

For issues or questions:
- Check the [CloudStation documentation](https://github.com/cloudstation/cloudstation)
- Review the [pack repository](https://github.com/thecloudstation/cloudstation-packs)
- Contact the CloudStation team

## License

Copyright (c) CloudStation, Inc. All rights reserved.