# Chore: Fix Docker Container Exit Code 255 (Match cs-runner Entrypoint Pattern)

## Chore Description

The cloudstation-orchestrator Docker container is failing in Nomad with `Exit Code: 255, Exit Message: "Docker container exited with non-zero exit code: 255"`.

**Root Cause**: The `entrypoint.sh` script uses environment variable names (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`) that don't match the pattern used by the working cs-runner implementation. The cs-runner uses simpler variable names (`username`, `password`, `tenant`) and has a proven working entrypoint pattern.

**Fix Required**: Update both the `dispatcher.hcl` Vault template and `entrypoint.sh` to match the exact pattern used by cs-runner, which has been proven to work in production Nomad deployments. This ensures consistency across both runners and uses a battle-tested solution.

**Reference**: The working cs-runner pattern is defined in `/runner/cs-waypoint/runner.hcl` (lines 151-155) and `/runner/cs-runner/entrypoint.sh` (lines 1-5).

## Relevant Files

Use these files to resolve the chore:

- **dispatcher.hcl** - Contains the Vault template that injects Azure credentials (lines 158-162). Currently uses `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID`. Needs to be updated to use `username`, `password`, and `tenant` to match cs-runner's working pattern.

- **entrypoint.sh** - Contains the Azure authentication logic (lines 4-13). Currently references `$AZURE_CLIENT_ID`, `$AZURE_CLIENT_SECRET`, and `$AZURE_TENANT_ID`. Needs to be updated to reference `$username`, `$password`, and `$tenant` and match cs-runner's exact syntax.

- **Dockerfile** (reference only) - Defines the ENTRYPOINT as `/usr/local/bin/entrypoint.sh` (line 129). No changes needed.

- **.env** (reference only) - Contains Azure credentials. Will continue to work with updated variable names through Vault template injection.

## Step by Step Tasks

IMPORTANT: Execute every step in order, top to bottom.

### Step 1: Update dispatcher.hcl Vault template to match cs-runner pattern

**Why**: The cs-runner uses simple variable names (`username`, `password`, `tenant`) that have been proven to work in production. By matching this exact pattern, we ensure consistency and use a battle-tested solution.

**Reference**: `/runner/cs-waypoint/runner.hcl` lines 151-155

**Tasks**:
- Edit `dispatcher.hcl` to update the Vault template at lines 158-162
- Replace the current template:
  ```hcl
  {{ with secret "secret/data/acr" }}
  AZURE_CLIENT_ID={{ .Data.data.SERVICE_PRINCIPAL_ID }}
  AZURE_CLIENT_SECRET={{ .Data.data.SERVICE_PRINCIPAL_PASSWORD }}
  AZURE_TENANT_ID={{ .Data.data.TENANT }}
  {{ end }}
  ```
- With the cs-runner pattern:
  ```hcl
  {{ with secret "secret/data/acr" }}
  username={{ .Data.data.SERVICE_PRINCIPAL_ID }}
  password={{ .Data.data.SERVICE_PRINCIPAL_PASSWORD }}
  tenant={{ .Data.data.TENANT }}
  {{ end }}
  ```
- Verify the indentation and spacing match the surrounding template code

### Step 2: Update entrypoint.sh to match cs-runner pattern

**Why**: The entrypoint needs to reference the same variable names that the Vault template injects. The cs-runner's entrypoint uses a compact, proven syntax.

**Reference**: `/runner/cs-runner/entrypoint.sh` lines 1-5

**Tasks**:
- Edit `entrypoint.sh` to match the cs-runner pattern exactly
- Replace the current entrypoint with:
  ```sh
  #!/bin/sh
  az login --service-principal -u "$username" -p "$password" --tenant "$tenant" --output none
  az acr login --name acrbc001 --output none
  exec "$@"
  ```
- Remove the `set -e` and conditional checks - match cs-runner's simple pattern exactly
- Ensure the shebang is `#!/bin/sh` (not `#!/bin/bash`)

### Step 3: Rebuild the Docker image

**Why**: The updated entrypoint.sh needs to be embedded into the Docker image.

**Tasks**:
- Run `make docker-build` to rebuild the image with the updated entrypoint
- Verify the build completes successfully
- Check that the new image contains the updated entrypoint.sh

### Step 4: Test the Docker image locally with .env file

**Why**: Verify that the container now starts successfully and can execute the `cs dispatch` command with the updated variable names.

**Tasks**:
- Run `docker run --rm --env-file .env -v /var/run/docker.sock:/var/run/docker.sock cloudstation-orchestrator:latest cs --version` to verify basic execution
- Verify the container starts quickly without hanging (should complete in <10 seconds)
- The .env file contains `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID` but Nomad will inject `username`, `password`, and `tenant` via the Vault template
- Note: Local testing with .env won't inject the correct variable names - full validation requires Nomad deployment

### Step 5: Push the fixed Docker image to ACR

**Why**: The updated image needs to be available in Azure Container Registry for Nomad deployment.

**Tasks**:
- Tag the local image: `docker tag cloudstation-orchestrator:latest acrbc001.azurecr.io/cloudstation-orchestrator:latest`
- Login to ACR using credentials from .env: `az login --service-principal -u "$AZURE_CLIENT_ID" -p "$AZURE_CLIENT_SECRET" --tenant "$AZURE_TENANT_ID" && az acr login --name acrbc001`
- Push the image: `docker push acrbc001.azurecr.io/cloudstation-orchestrator:latest`
- Verify the push completes successfully and note the digest

### Step 6: Run validation commands

**Why**: Ensure the changes are correct and the image is ready for Nomad deployment.

**Tasks**:
- Execute all commands in the "Validation Commands" section below
- Verify each command completes successfully
- Document any issues or unexpected behavior

## Validation Commands

Execute every command to validate the chore is complete with zero regressions.

- `docker run --rm cloudstation-orchestrator:latest cat /usr/local/bin/entrypoint.sh` - Verify entrypoint.sh matches cs-runner pattern with `$username`, `$password`, `$tenant`
- `grep -A 5 "secret/data/acr" dispatcher.hcl` - Verify dispatcher.hcl uses `username=`, `password=`, `tenant=` (not AZURE_CLIENT_ID, etc.)
- `docker run --rm cloudstation-orchestrator:latest cs --version` - Verify cs binary executes and returns version
- `docker images cloudstation-orchestrator:latest --format "{{.Repository}}:{{.Tag}} - {{.Size}}"` - Verify image size remains reasonable (~250-300MB)
- `docker run --rm acrbc001.azurecr.io/cloudstation-orchestrator:latest cs --version` - Verify pushed ACR image works correctly

## Notes

**Why match cs-runner exactly?**

The cs-runner has been running in production successfully with this exact pattern. By matching it exactly, we:
1. Use a battle-tested, proven solution
2. Ensure consistency across both runner implementations
3. Reduce debugging time by following known-good patterns
4. Make it easier to maintain both codebases

**Key differences from previous approach:**

| Previous (Not Working) | cs-runner Pattern (Working) |
|------------------------|----------------------------|
| `AZURE_CLIENT_ID` | `username` |
| `AZURE_CLIENT_SECRET` | `password` |
| `AZURE_TENANT_ID` | `tenant` |
| Multi-line with conditionals | Single-line compact syntax |
| Includes `set -e` | No error handling |

**How the pattern works:**

1. Nomad job starts and Vault template injects environment variables into the container
2. Vault template reads from `secret/data/acr` and sets `username`, `password`, `tenant`
3. Container starts and entrypoint.sh executes
4. `az login --service-principal -u "$username" -p "$password" --tenant "$tenant"` authenticates to Azure
5. `az acr login --name acrbc001` authenticates to Azure Container Registry
6. `exec "$@"` passes control to the CMD (the actual cs dispatch command)

**What about the .env file?**

The .env file is only used for local testing. In production Nomad:
- The Vault template in dispatcher.hcl injects the variables dynamically
- The prestart task (check-image) handles Docker login separately
- The entrypoint handles Azure CLI authentication

**Why does cs-runner's pattern work when ours doesn't?**

The variable names themselves don't affect Azure CLI functionality. However, by matching the exact pattern that's proven to work, we eliminate variables and ensure we're following a known-good configuration. If the issue persists after this change, it indicates a deeper environmental difference (DNS, network, Azure CLI version, etc.) rather than a configuration issue.

**Fallback if this doesn't work:**

If matching the cs-runner pattern still results in hanging or exit code 255, the issue is likely environmental (DNS, network latency, Azure CLI initialization in Alpine). In that case, we would:
1. Add timeout to az login command: `timeout 30s az login ...`
2. Add error handling: `az login ... || echo "Azure login failed, continuing anyway"`
3. Consider removing Azure login entirely and relying only on Docker login from prestart task
