# Chore: Ensure Perfect vars.hcl Parity with cs-runner for cloudstation-packs Compatibility

## Chore Description
The cloudstation-orchestrator's HCL generation (specifically vars.hcl) must produce output that is byte-for-byte compatible with cs-runner's output to work seamlessly with @cloudstation-packs/. Currently there are several discrepancies that cause deployment failures:

1. **Missing Consul Tags** - cs-runner generates `consul_tags` array with automatic "sticky" tag for multi-replica deployments plus user tags, but the new CLI omits this entirely
2. **Missing TLS Variables** - cs-runner generates `use_tls` and `tls` array variables, but new CLI has no TLS support
3. **Incorrect Empty Array Handling** - New CLI always outputs empty arrays for some fields (e.g., `consul_linked_services = []`), while cs-runner conditionally omits empty fields
4. **Missing config_files Conditional Logic** - cs-runner only outputs config_files if non-empty, new CLI always outputs it
5. **Missing template Conditional Logic** - cs-runner only outputs template if non-empty, new CLI always outputs it
6. **vault_linked_secrets Always Empty** - New CLI outputs placeholder empty array instead of actual vault linked secrets
7. **Field Ordering** - vars.hcl field order must match cs-runner exactly for pack compatibility

This chore ensures 100% parity by comparing every field, every conditional, and every output format between cs-runner's `getDeployStanzaVars()` and the new CLI's `GenerateVarsFile()`.

## Relevant Files
Use these files to resolve the chore:

**Existing Files to Update:**

- `internal/hclgen/types.go` - Add missing TLS types (TLSConfig struct with cert_path, key_path, common_name, pka_path, ttl fields)
- `internal/hclgen/generator.go` - Fix all vars.hcl generation issues:
  - Add `generateConsulTags()` function to generate consul_tags with "sticky" logic
  - Add `generateTLSVariables()` function to generate use_tls and tls array
  - Fix `generateConsulVariables()` to include tags and conditionally output linked_services
  - Fix config_files to only output when non-empty (remove always-empty-array)
  - Fix template to only output when non-empty (remove always-empty-array)
  - Implement vault_linked_secrets generation (currently placeholder)
  - Reorder vars.hcl output to match cs-runner field ordering exactly
- `internal/dispatch/types.go` - Add TLS field to DeployRepositoryParams and DeployImageParams
- `internal/dispatch/handlers.go` - Map TLS from params to hclgen.DeploymentParams

**Reference Files (cs-runner - DO NOT MODIFY):**

- `/Users/oumnyabenhassou/Code/runner/cs-runner/src/lib/hcl/deploy.ts:268-362` - Complete vars.hcl generation with exact field ordering
- `/Users/oumnyabenhassou/Code/runner/cs-runner/src/lib/hcl/consul.ts:1-42` - Consul variables with tags and sticky logic
- `/Users/oumnyabenhassou/Code/runner/cs-runner/src/lib/hcl/tls.ts:1-12` - TLS variables generation
- `/Users/oumnyabenhassou/Code/runner/cs-runner/src/lib/hcl/types.ts:60-65` - ConsulSettings type with tags field
- `/Users/oumnyabenhassou/Code/runner/cs-runner/src/lib/hcl/types.ts:84-90` - TLSOptions type definition

**Test Files:**

- `internal/hclgen/generator_test.go` - Update tests to validate consul_tags, TLS, and conditional field output

### New Files

None - all changes are to existing files

## Step by Step Tasks
IMPORTANT: Execute every step in order, top to bottom.

### 1. Add TLS Type Definitions

- Edit `internal/hclgen/types.go`
- Add `TLSConfig` struct after `ConsulConfig` struct:
  ```go
  // TLSConfig represents TLS certificate configuration
  type TLSConfig struct {
      CertPath   string
      KeyPath    string
      CommonName string
      PkaPath    string
      TTL        string
  }
  ```
- Add `TLS *TLSConfig` field to `DeploymentParams` struct (around line 187, after Regions field)

### 2. Update Dispatch Types for TLS

- Edit `internal/dispatch/types.go`
- Add `TLSSettings` struct after `VaultLinkedSecret` struct (around line 147):
  ```go
  // TLSSettings represents TLS certificate configuration
  type TLSSettings struct {
      CertPath   string `json:"cert_path"`
      KeyPath    string `json:"key_path"`
      CommonName string `json:"common_name"`
      PkaPath    string `json:"pka_path"`
      TTL        string `json:"ttl"`
  }
  ```
- Add `TLS *TLSSettings` field to `DeployRepositoryParams` struct after VaultLinkedSecrets field (around line 201)
- Add `TLS *TLSSettings` field to `DeployImageParams` struct after VaultLinkedSecrets field (around line 283)

### 3. Fix Consul Tags Generation

- Edit `internal/hclgen/generator.go`
- Update `generateConsulVariables()` function (line 412-433) to match cs-runner logic:
  ```go
  func generateConsulVariables(consul ConsulConfig, replicaCount int) string {
      if consul.ServiceName == "" {
          return ""
      }

      var vars strings.Builder

      // Generate consul tags array
      consulTags := make([]string, 0)

      // Add "sticky" tag for multi-replica deployments
      if replicaCount > 1 {
          consulTags = append(consulTags, "\"sticky\"")
      }

      // Add user-provided tags
      for _, tag := range consul.Tags {
          consulTags = append(consulTags, fmt.Sprintf("\"%s\"", tag))
      }

      vars.WriteString(fmt.Sprintf("consul_service_name = \"%s\"\n", consul.ServiceName))

      // Output consul_tags array
      if len(consulTags) > 0 {
          vars.WriteString(fmt.Sprintf("consul_tags = [%s]\n", strings.Join(consulTags, ", ")))
      }

      // Linked services - only output if non-empty (match cs-runner behavior)
      if len(consul.LinkedServices) > 0 {
          linkedServices := make([]string, len(consul.LinkedServices))
          for i, ls := range consul.LinkedServices {
              linkedServices[i] = fmt.Sprintf("{key=\"%s\", value=\"%s\"}", ls.VariableName, ls.ConsulServiceName)
          }
          vars.WriteString(fmt.Sprintf("consul_linked_services = [%s]\n", strings.Join(linkedServices, ", ")))
      }
      // DO NOT output empty array if no linked services - match cs-runner

      return vars.String()
  }
  ```

### 4. Add TLS Variables Generation

- Edit `internal/hclgen/generator.go`
- Add new function `generateTLSVariables()` before `generateConsulVariables()` function (around line 410):
  ```go
  func generateTLSVariables(tls *TLSConfig) string {
      if tls == nil {
          return "use_tls = false\n"
      }

      var vars strings.Builder
      vars.WriteString("use_tls = true\n")
      vars.WriteString(fmt.Sprintf("tls = [{ cert_path=\"%s\", key_path=\"%s\", common_name=\"%s\", pka_path=\"%s\", ttl=\"%s\" }]\n",
          tls.CertPath, tls.KeyPath, tls.CommonName, tls.PkaPath, tls.TTL))

      return vars.String()
  }
  ```

### 5. Fix config_files Conditional Output

- Edit `internal/hclgen/generator.go`
- Update config_files generation in `GenerateVarsFile()` function (around line 296-305):
  ```go
  // Config files - only output if non-empty (match cs-runner)
  if len(params.ConfigFiles) > 0 {
      configFiles := make([]string, len(params.ConfigFiles))
      for i, cf := range params.ConfigFiles {
          configFiles[i] = fmt.Sprintf("{ path=\"%s\", content=\"%s\" }", cf.Path, cf.Content)
      }
      vars.WriteString(fmt.Sprintf("config_files = [%s]\n", strings.Join(configFiles, ", ")))
  }
  // DO NOT output "config_files = []" if empty - removed to match cs-runner
  ```

### 6. Fix template Conditional Output

- Edit `internal/hclgen/generator.go`
- Update template generation in `GenerateVarsFile()` function (around line 310-324):
  ```go
  // Template string variables - only output if non-empty (match cs-runner)
  if len(params.TemplateStringVariables) > 0 {
      templateVars := make([]string, len(params.TemplateStringVariables))
      for i, tv := range params.TemplateStringVariables {
          linkedVars := make([]string, len(tv.LinkedVars))
          for j, lv := range tv.LinkedVars {
              linkedVars[j] = fmt.Sprintf("\"%s\"", lv)
          }
          templateVars[i] = fmt.Sprintf("{name=\"%s\",pattern=\"%s\",service_name=\"%s\",service_secret_path=\"%s\",linked_vars=[%s]}",
              tv.Name, tv.Pattern, tv.ServiceName, tv.ServiceSecretPath, strings.Join(linkedVars, ","))
      }
      vars.WriteString(fmt.Sprintf("template = [%s]\n", strings.Join(templateVars, ", ")))
  }
  // DO NOT output "template = []" if empty - removed to match cs-runner
  ```

### 7. Remove vault_linked_secrets Placeholder

- Edit `internal/hclgen/generator.go`
- Remove vault_linked_secrets generation (lines 326-333):
  ```go
  // DELETE THESE LINES - cs-runner does NOT output vault_linked_secrets
  // Vault linked secrets (always include - required by pack)
  // if len(params.VaultLinkedSecrets) > 0 {
  //     // Generate vault_linked_secrets content if needed
  //     // For now just include empty array
  //     vars.WriteString("vault_linked_secrets = []\n")
  // } else {
  //     vars.WriteString("vault_linked_secrets = []\n")
  // }
  ```
- Note: cs-runner handles vault linked secrets through the `args` variable, NOT a separate vault_linked_secrets field

### 8. Remove regions Always-Output Logic

- Edit `internal/hclgen/generator.go`
- Update regions generation (around line 335-340):
  ```go
  // Regions - only output if non-empty AND gpu == 0 (match cs-runner)
  if params.Regions != "" && params.GPU == 0 {
      vars.WriteString(fmt.Sprintf("regions = \"%s\"\n", params.Regions))
  }
  // DO NOT output "regions = \"\"" if empty - removed to match cs-runner
  ```

### 9. Reorder vars.hcl Fields to Match cs-runner Exactly

- Edit `internal/hclgen/generator.go`
- Reorder the `GenerateVarsFile()` function to match cs-runner's `getDeployStanzaVars()` field order (lines 268-361):

**Exact Field Order from cs-runner:**
1. job_name
2. count
3. secret_path
4. restart_attempts
5. restart_mode
6. resources
7. gpu_type (if gpu > 0)
8. node_pool (if present)
9. user_id
10. alloc_id
11. project_id
12. service_id
13. shared_secret_path (if present)
14. uses_kv_engine (if defined)
15. owner_uses_kv_engine (if defined)
16. regions (if present AND gpu == 0)
17. private_registry (if present)
18. private_registry_provider (if present)
19. user (docker_user)
20. command (if present)
21. image
22. use_csi_volume, volume_name, volume_mount_destination
23. config_files (if non-empty)
24. consul_service_name, consul_tags, consul_linked_services (if consul configured)
25. entrypoint (if present)
26. template (if non-empty)
27. network (if present)
28. use_tls, tls (TLS variables)
29. "update" (if present)
30. job_config (if present)
31. args (if vault linked secrets OR start command)

- Restructure `GenerateVarsFile()` to output fields in this exact order
- Ensure all conditional fields match cs-runner logic (only output when non-empty/defined)

### 10. Update Dispatch Handlers to Pass TLS

- Edit `internal/dispatch/handlers.go`
- In `HandleDeployRepository()` function, update hclParams construction (around line 77):
  ```go
  hclParams := mapDeployRepositoryToHCLParams(params)
  ```
- In `mapDeployRepositoryToHCLParams()` helper function (create if doesn't exist), add TLS mapping:
  ```go
  if params.TLS != nil {
      hclParams.TLS = &hclgen.TLSConfig{
          CertPath:   params.TLS.CertPath,
          KeyPath:    params.TLS.KeyPath,
          CommonName: params.TLS.CommonName,
          PkaPath:    params.TLS.PkaPath,
          TTL:        params.TLS.TTL,
      }
  }
  ```
- Similarly update `HandleDeployImage()` with TLS mapping

### 11. Update Tests to Validate Perfect Parity

- Edit `internal/hclgen/generator_test.go`
- Add test `TestGenerateVarsFile_ConsulTags`:
  ```go
  func TestGenerateVarsFile_ConsulTags(t *testing.T) {
      params := hclgen.DeploymentParams{
          JobID: "test-job",
          ReplicaCount: 3, // Multi-replica for sticky tag
          Consul: hclgen.ConsulConfig{
              ServiceName: "my-service",
              Tags: []string{"web", "api"},
          },
      }

      varsContent := hclgen.GenerateVarsFile(params)

      // Verify consul_tags includes sticky + user tags
      if !strings.Contains(varsContent, `consul_tags = ["sticky", "web", "api"]`) {
          t.Errorf("Expected consul_tags with sticky tag for multi-replica")
      }
  }
  ```
- Add test `TestGenerateVarsFile_TLS`:
  ```go
  func TestGenerateVarsFile_TLS(t *testing.T) {
      params := hclgen.DeploymentParams{
          JobID: "test-job",
          TLS: &hclgen.TLSConfig{
              CertPath:   "/path/to/cert",
              KeyPath:    "/path/to/key",
              CommonName: "example.com",
              PkaPath:    "/path/to/pka",
              TTL:        "24h",
          },
      }

      varsContent := hclgen.GenerateVarsFile(params)

      // Verify TLS output
      if !strings.Contains(varsContent, "use_tls = true") {
          t.Errorf("Expected use_tls = true")
      }
      if !strings.Contains(varsContent, `tls = [{ cert_path="/path/to/cert"`) {
          t.Errorf("Expected tls array with cert_path")
      }
  }
  ```
- Add test `TestGenerateVarsFile_ConditionalFields` to verify empty fields are NOT output:
  ```go
  func TestGenerateVarsFile_ConditionalFields(t *testing.T) {
      params := hclgen.DeploymentParams{
          JobID: "test-job",
          // No ConfigFiles, TemplateStringVariables, or Consul.LinkedServices
      }

      varsContent := hclgen.GenerateVarsFile(params)

      // Verify empty arrays are NOT output
      if strings.Contains(varsContent, "config_files = []") {
          t.Errorf("Should NOT output empty config_files array")
      }
      if strings.Contains(varsContent, "template = []") {
          t.Errorf("Should NOT output empty template array")
      }
      if strings.Contains(varsContent, "consul_linked_services = []") {
          t.Errorf("Should NOT output empty consul_linked_services array")
      }
      if strings.Contains(varsContent, "vault_linked_secrets") {
          t.Errorf("Should NOT output vault_linked_secrets at all")
      }
  }
  ```
- Update existing test `TestGenerateVarsFile_AllFields` to include TLS and verify field ordering matches cs-runner

### 12. Run All Validation Commands

- Execute all commands from Validation Commands section below
- Fix any test failures or discrepancies
- Manually compare generated vars.hcl with cs-runner output for same payload
- Ensure 100% byte-for-byte parity

## Validation Commands
Execute every command to validate the chore is complete with zero regressions.

```bash
# Verify working directory
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator

# Verify Go modules are clean
go mod verify
go mod tidy

# Build the binary
make clean
make build
ls -lh bin/cs

# Run hclgen package tests with verbose output
go test ./internal/hclgen/... -v -cover

# Run all tests for dispatch package
go test ./internal/dispatch/... -v -cover

# Run specific new tests
go test -run TestGenerateVarsFile_ConsulTags ./internal/hclgen/... -v
go test -run TestGenerateVarsFile_TLS ./internal/hclgen/... -v
go test -run TestGenerateVarsFile_ConditionalFields ./internal/hclgen/... -v

# Run all tests with race detection
go test ./... -race

# Run all tests with coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Code quality checks
go fmt ./...
go vet ./...

# Test with real payload (if test-real.sh exists)
./test-real.sh

# Verify generated vars.hcl matches cs-runner output
# Manual step: Compare vars.hcl from work directory with cs-runner generated vars.hcl
# They should be identical except for dynamic values (IDs, timestamps, etc.)
```

## Notes

### Critical Differences Found Between cs-runner and Current Implementation

1. **Consul Tags Missing** - cs-runner adds "sticky" tag automatically for replicaCount > 1, plus user tags from `consul.tags[]`. New CLI completely omits consul_tags.

2. **TLS Variables Missing** - cs-runner outputs `use_tls = false` or `use_tls = true` plus `tls = [{ cert_path=..., key_path=..., ... }]`. New CLI has zero TLS support.

3. **Empty Array Philosophy** - cs-runner conditionally outputs fields only when non-empty. New CLI outputs empty arrays `[]` for config_files, template, vault_linked_secrets, consul_linked_services, and regions even when empty. This must be fixed.

4. **vault_linked_secrets** - cs-runner does NOT output a vault_linked_secrets variable. Instead, it includes vault linked secrets in the `args` variable as shell export statements. New CLI has placeholder empty array that should be removed.

5. **Field Ordering Matters** - Nomad packs may rely on specific field ordering for template parsing. vars.hcl field order must match cs-runner exactly.

### cs-runner vars.hcl Structure Reference

From `cs-runner/src/lib/hcl/deploy.ts:268-362`, the complete vars.hcl structure is:

```hcl
job_name                = "..."
count                   = 1
secret_path             = "..."
restart_attempts        = 3
restart_mode            = "fail"
resources               = {cpu=100, memory=512, gpu=0}
gpu_type                = "L4"  # only if gpu > 0
node_pool               = "..."  # only if defined
user_id                 = "..."
alloc_id                = "..."
project_id              = "..."
service_id              = "..."
shared_secret_path      = "..."  # only if defined
uses_kv_engine          = true  # only if defined
owner_uses_kv_engine    = false  # only if defined
regions                 = "..."  # only if defined AND gpu == 0
private_registry        = "..."  # only if defined
private_registry_provider = "..."  # only if defined
user                    = "0"  # docker_user, always output
command                 = "..."  # only if defined
image                   = "registry.io/image:tag"
use_csi_volume          = true
volume_name             = "vol_xxx"
volume_mount_destination = ["/data"]
config_files            = [{ path="...", content="..." }]  # only if non-empty
consul_service_name     = "..."
consul_tags             = ["sticky", "web", "api"]
consul_linked_services  = [{key="...", value="..."}]  # only if non-empty
entrypoint              = "[\"sh\", \"-c\"]"  # only if defined
template                = [{name="...",pattern="...",service_name="...",service_secret_path="...",linked_vars=[...]}]  # only if non-empty
network                 = [{name="3000", port=3000, type="http", public=true, domain="...", custom_domain="...", has_health_check="http", health_check={...}}]
use_tls                 = false
tls                     = [{ cert_path="...", key_path="...", common_name="...", pka_path="...", ttl="..." }]  # only if use_tls=true
"update"                = "{min_healthy_time=..., healthy_deadline=..., ...}"  # only if defined
job_config              = {type="service", cron="", prohibit_overlap="true", payload="", meta_required=[]}  # only if defined
args                    = ["export SECRET=value;start-command"]  # only if vault linked secrets OR start command
```

### Testing Strategy

1. **Unit Tests** - Test each generation function individually (consul tags, TLS, conditional fields)
2. **Integration Test** - Use real cs-runner payload, generate vars.hcl with new CLI, compare byte-for-byte
3. **Pack Deployment Test** - Actually deploy to Nomad using generated HCL to verify pack compatibility

### Backward Compatibility

- Existing deployments without TLS should continue working (use_tls = false by default)
- Existing deployments without Consul tags should work (tags array can be empty)
- Only breaking change: Removed empty arrays for config_files, template, vault_linked_secrets to match cs-runner
