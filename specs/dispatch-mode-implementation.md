# Feature: Dispatch Mode for Nomad Job Execution

## Feature Description
Implement a dispatch mode in the CloudStation Orchestrator that enables the `cs` binary to replace the existing TypeScript cs-runner for executing Nomad-dispatched deployment jobs. This mode reads task metadata from Nomad environment variables (`NOMAD_META_TASK` and `NOMAD_META_PARAMS`), routes to appropriate handlers (deploy-repository, deploy-image, destroy-job-pack), executes the deployment lifecycle, and publishes status events to NATS JetStream for backend monitoring. This enables a complete migration from the Node.js-based cs-runner (~80MB with dependencies) to a single Go binary (~14-20MB) with significantly improved performance and maintainability.

## User Story
As a platform engineer maintaining the CloudStation deployment system
I want to replace the TypeScript cs-runner with a Go-based dispatch mode in the orchestrator
So that I can eliminate Node.js runtime overhead, reduce image size by 75%, improve startup time by 20-30x, and maintain a single unified codebase in Go instead of managing separate TypeScript and Go codebases

## Problem Statement
The current cs-runner implementation is a TypeScript application that wraps the Waypoint binary and handles Nomad-dispatched jobs. This creates several issues:

1. **Dual Runtime Overhead**: Requires both Node.js runtime and the Waypoint binary, resulting in ~80MB Docker images
2. **Slow Startup**: TypeScript compilation and Node.js initialization takes 2-3 seconds before job execution
3. **Split Codebase**: Deployment logic is split between TypeScript (cs-runner) and Go (orchestrator), making it harder to maintain
4. **Complex Dependencies**: Requires npm packages (simple-git, nats, axios, etc.) increasing attack surface and maintenance burden
5. **Waypoint Dependency**: Still depends on the unmaintained Waypoint binary for HCL generation and lifecycle execution
6. **Harder to Test**: Integration testing requires Node.js + Docker + NATS + Nomad stack

The cs-runner currently handles three task types:
- `deploy-repository`: Clone Git repo, generate HCL, run build+deploy lifecycle
- `deploy-image`: Deploy pre-built Docker image without building
- `destroy-job-pack`: Destroy Nomad Pack deployments

Each task publishes NATS events (started, succeeded, failed) back to the backend for status tracking.

## Solution Statement
Add a `dispatch` command to the CloudStation Orchestrator that:

1. **Reads Nomad Metadata**: Extracts `NOMAD_META_TASK` and `NOMAD_META_PARAMS` environment variables
2. **Routes to Handlers**: Implements three dispatch handlers (deploy-repository, deploy-image, destroy-job-pack)
3. **Executes Lifecycle**: Uses the existing orchestrator lifecycle executor instead of calling external Waypoint binary
4. **Publishes NATS Events**: Sends deployment status events to NATS JetStream for backend monitoring
5. **Clones Repositories**: Implements Git clone functionality for repository deployments
6. **Generates HCL Dynamically**: Creates cloudstation.hcl configurations from deployment parameters
7. **Handles Errors**: Catches all errors and publishes failure events with proper exit codes

This allows the Nomad job template to simply run `cs dispatch` instead of starting the TypeScript cs-runner container, reducing image size from ~80MB to ~20MB and eliminating Node.js runtime overhead entirely.

The dispatch mode will be a hidden command (not shown in CLI help) since it's only used internally by Nomad batch jobs, not by end users.

## Relevant Files
Use these files to implement the feature:

**Existing Files to Reference:**
- `cs-runner/src/index.ts:11-98` - Current dispatch logic with task routing, NATS events, and timeout handling
- `cs-runner/src/deployRepository.ts:23-99` - Repository deployment workflow (clone, HCL generation, lifecycle execution)
- `cs-runner/src/deployImage.ts:8-69` - Image deployment workflow (HCL generation for pre-built images)
- `cs-runner/src/destroyJob.ts:17-31` - Job destruction workflow (nomad-pack destroy)
- `cs-runner/src/nats.ts:15-117` - NATS client implementation with JetStream publishing (events: deployment.status.changed, deployment.succeeded, deployment.failed, job.destroyed)
- `cs-runner/src/lib/git.ts:23-46` - Git repository cloning logic with provider support (GitHub, GitLab, Bitbucket)
- `cs-runner/src/lib/hcl/hcl.ts:7-91` - Dynamic HCL generation for Waypoint configurations
- `cloudstation-orchestrator/cmd/cloudstation/commands.go:1-171` - Existing CLI commands (init, build, deploy, up, runner agent)
- `cloudstation-orchestrator/cmd/cloudstation/main.go:26-71` - CLI application setup and command registration
- `cloudstation-orchestrator/internal/lifecycle/executor.go:1-213` - Lifecycle executor with build, registry, deploy phases
- `cloudstation-orchestrator/internal/config/types.go` - Configuration type definitions for parsing HCL
- `cloudstation-orchestrator/internal/config/parser.go` - HCL configuration parser
- `cloudstation-orchestrator/go.mod:1-30` - Current Go dependencies

### New Files

**Dispatch Command:**
- `cloudstation-orchestrator/cmd/cloudstation/dispatch.go` - Dispatch command implementation with task routing and NATS integration

**NATS Client:**
- `cloudstation-orchestrator/pkg/nats/client.go` - NATS JetStream client for publishing deployment events
- `cloudstation-orchestrator/pkg/nats/types.go` - Event payload types matching cs-runner format
- `cloudstation-orchestrator/pkg/nats/client_test.go` - Unit tests for NATS client

**Git Operations:**
- `cloudstation-orchestrator/pkg/git/clone.go` - Git repository cloning with multi-provider support
- `cloudstation-orchestrator/pkg/git/types.go` - Git provider types and URL building
- `cloudstation-orchestrator/pkg/git/clone_test.go` - Unit tests for Git operations

**HCL Generation:**
- `cloudstation-orchestrator/internal/hclgen/generator.go` - Dynamic HCL generation from deployment parameters
- `cloudstation-orchestrator/internal/hclgen/types.go` - Deployment parameter types
- `cloudstation-orchestrator/internal/hclgen/generator_test.go` - Unit tests for HCL generation

**Dispatch Handlers:**
- `cloudstation-orchestrator/internal/dispatch/handlers.go` - Implementation of three dispatch handlers
- `cloudstation-orchestrator/internal/dispatch/params.go` - Parameter parsing from NOMAD_META_PARAMS JSON
- `cloudstation-orchestrator/internal/dispatch/types.go` - Type definitions for dispatch parameters
- `cloudstation-orchestrator/internal/dispatch/handlers_test.go` - Unit tests for dispatch handlers

**Integration Tests:**
- `cloudstation-orchestrator/test/integration/dispatch_test.go` - End-to-end dispatch mode tests

## Implementation Plan

### Phase 1: Foundation
Set up the foundational components needed for dispatch mode. This includes adding the NATS client library dependency, implementing the Git clone functionality, and creating the HCL generation logic that will dynamically create cloudstation.hcl files from deployment parameters. These are core capabilities that all dispatch handlers will depend on, so they must be implemented first and thoroughly tested before moving to handler implementation.

### Phase 2: Core Implementation
Implement the three dispatch handlers (deploy-repository, deploy-image, destroy-job-pack) and the dispatch command that routes to them. Integrate with the existing lifecycle executor for build+deploy operations, add NATS event publishing at all lifecycle stages, and implement proper error handling with exit codes. This phase brings all the foundational components together into working dispatch functionality.

### Phase 3: Integration
Create the dispatch command CLI entry point, add comprehensive integration tests that simulate Nomad job execution, update documentation with dispatch mode usage, and create a migration guide for transitioning from cs-runner to the new dispatch mode. This phase ensures the feature is production-ready and documented for deployment.

## Step by Step Tasks

### 1. Add NATS Client Library Dependency

**Install NATS Go client:**
- Run `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator`
- Run `go get github.com/nats-io/nats.go@latest` to add NATS client library
- Run `go get github.com/nats-io/nkeys@latest` to add NKeys authentication support
- Run `go mod tidy` to clean up dependencies
- Verify installation by checking `go.mod` contains `github.com/nats-io/nats.go` and `github.com/nats-io/nkeys`

### 2. Implement NATS Client Package

**Create NATS types:**
- Create `pkg/nats/types.go`
- Define `DeploymentEventPayload` struct with fields: `JobID`, `Type` (git_repo or image), `DeploymentID`, `ServiceID`, `TeamID`, `UserID`, `OwnerID` (matching cs-runner/src/nats.ts:5-13)
- Define `JobDestroyedPayload` struct with fields: `ID`, `Reason` (delete, upgrade_volume, migrate_volume, suspend_account, pause_service)
- Define `DeploymentStatus` enum: `IN_PROGRESS`, `SUCCEEDED`, `FAILED`
- Add JSON struct tags for serialization

**Create NATS client:**
- Create `pkg/nats/client.go`
- Implement `Client` struct with fields: `conn *nats.Conn`, `logger hclog.Logger`
- Implement `NewClient(servers []string, nkeySeed string, logger hclog.Logger) (*Client, error)` function
  - Parse NKey seed using `nkeys.FromSeed()`
  - Create NKey authenticator using `nats.Nkey()`
  - Connect to NATS with `nats.Connect(servers, nats.UserJWT(...))`
  - Return Client instance
- Implement `PublishDeploymentStarted(jobID int) error` - publishes to `deployment.status.changed` subject with status `IN_PROGRESS`
- Implement `PublishDeploymentSucceeded(payload DeploymentEventPayload) error` - publishes to `deployment.succeeded` subject
- Implement `PublishDeploymentFailed(payload DeploymentEventPayload) error` - publishes to `deployment.failed` subject
- Implement `PublishJobDestroyed(payload JobDestroyedPayload) error` - publishes to `job.destroyed` subject
- Implement `Close() error` to drain and close connection
- Use JetStream for all publishes: `js, _ := client.conn.JetStream()` then `js.Publish(subject, data)`
- Add structured logging for all publish operations

**Create NATS client tests:**
- Create `pkg/nats/client_test.go`
- Test `NewClient` with valid and invalid NKey seeds
- Test `NewClient` with unreachable NATS servers (should return error)
- Test all publish methods with mock NATS server or test container
- Test `Close` properly drains connection
- Use table-driven tests for different event types

### 3. Implement Git Clone Package

**Create Git types:**
- Create `pkg/git/types.go`
- Define `Provider` enum type with constants: `GitHub`, `GitLab`, `Bitbucket`
- Implement `GetBaseURL(provider Provider) string` function returning provider base URLs
- Implement `BuildAuthURL(repository, token string, provider Provider) string` function
  - For GitHub/GitLab: `https://x-access-token:{token}@{baseURL}/{repository}.git`
  - For Bitbucket: `https://x-token-auth:{token}@bitbucket.org/{repository}.git`
  - For no token: `https://{baseURL}/{repository}.git`

**Create Git clone functionality:**
- Create `pkg/git/clone.go`
- Import `os/exec` package for running git commands
- Implement `CloneOptions` struct with fields: `Repository`, `Branch`, `Token`, `Provider`, `DestinationDir`
- Implement `Clone(opts CloneOptions) error` function
  - Build Git URL using `BuildAuthURL()`
  - Execute `git clone --branch {branch} {url} {destination}` using `exec.Command()`
  - Capture stdout/stderr for logging
  - Return error if clone fails with context about repository and branch
- Add debug logging for clone operations (redact token from logs)

**Create Git clone tests:**
- Create `pkg/git/clone_test.go`
- Test `BuildAuthURL` with all providers (GitHub, GitLab, Bitbucket)
- Test `BuildAuthURL` without token (public repos)
- Test `Clone` with mock Git command execution (use test doubles)
- Test error handling for invalid repositories
- Test error handling for invalid branches
- Verify token is never logged in error messages

### 4. Implement HCL Generation Package

**Create HCL generator types:**
- Create `internal/hclgen/types.go`
- Define `DeploymentParams` struct with fields matching cs-runner deployment options:
  - `JobID`, `ImageName`, `ImageTag`, `BuilderType` (csdocker, nixpacks, noop), `DeployType` (nomadpack, noop)
  - `ReplicaCount`, `PrivateRegistry` bool, `VaultConfig` map, `NomadPackConfig` map
- Define `BuildConfig` struct for build stanza parameters
- Define `DeployConfig` struct for deploy stanza parameters

**Create HCL generator:**
- Create `internal/hclgen/generator.go`
- Implement `GenerateConfig(params DeploymentParams) (string, error)` function
  - Generate project name from `JobID`
  - Generate app block with name from `JobID`
  - Generate build stanza based on `BuilderType` (csdocker, nixpacks, or noop)
  - Generate deploy stanza based on `DeployType` (nomadpack or noop)
  - Include variable definitions for registry credentials
  - Return HCL string matching waypoint.hcl format from cs-runner
- Implement `WriteConfigFile(config string, directory string) (string, error)` function
  - Write config to `{directory}/cloudstation.hcl`
  - Return full file path
- Add template-based generation using text/template for maintainability
- Validate generated HCL can be parsed by config parser

**Create HCL generator tests:**
- Create `internal/hclgen/generator_test.go`
- Test `GenerateConfig` with csdocker builder + nomadpack deploy
- Test `GenerateConfig` with nixpacks builder + nomadpack deploy
- Test `GenerateConfig` with noop builder + noop deploy (redeploy scenario)
- Test `WriteConfigFile` creates file with correct content
- Parse generated HCL with config parser to validate syntax
- Test variable substitution works correctly

### 5. Implement Dispatch Parameter Parsing

**Create dispatch parameter types:**
- Create `internal/dispatch/types.go`
- Define `TaskType` string type with constants: `TaskDeployRepository`, `TaskDeployImage`, `TaskDestroyJob`
- Define `DeployRepositoryParams` struct matching cs-runner DeployRepositoryOptions
  - Include all fields from cs-runner/src/deployRepository.ts:12-21
- Define `DeployImageParams` struct matching cs-runner DeploymentJobOptions
- Define `DestroyJobParams` struct matching cs-runner DestroyJobOptionsNomadPack
  - Include `Jobs` slice and `Reason` field from cs-runner/src/destroyJob.ts:5-15

**Create parameter parsing:**
- Create `internal/dispatch/params.go`
- Implement `ParseTaskType() (TaskType, error)` - reads `NOMAD_META_TASK` env var
- Implement `ParseParams(taskType TaskType) (interface{}, error)` - reads `NOMAD_META_PARAMS` env var
  - Decode JSON string to appropriate struct based on task type
  - Validate required fields are present
  - Return typed struct (DeployRepositoryParams, DeployImageParams, or DestroyJobParams)
  - Return error if JSON parsing fails or required fields missing

### 6. Implement Deploy Repository Handler

**Create dispatch handlers file:**
- Create `internal/dispatch/handlers.go`
- Import packages: `context`, `fmt`, `os`, `path/filepath`, `github.com/hashicorp/go-hclog`
- Import internal packages: `config`, `lifecycle`, `hclgen`, `git`, `nats`

**Implement deploy repository handler:**
- Implement `HandleDeployRepository(params DeployRepositoryParams, natsClient *nats.Client, logger hclog.Logger) error` function
- Create temporary work directory using `os.MkdirTemp("", "cs-deploy-*")`
- Publish deployment started event via NATS
- Clone Git repository using `git.Clone()` with params.Repository, params.Branch, params.GitPass
- Check for Dockerfile existence with `os.Stat(filepath.Join(workDir, "Dockerfile"))`
- Generate HCL config using `hclgen.GenerateConfig()` with deployment parameters
- Write HCL to work directory using `hclgen.WriteConfigFile()`
- Load config using `config.LoadConfigFile()`
- Create lifecycle executor with `lifecycle.NewExecutor(cfg, logger)`
- Execute full lifecycle with `executor.Execute(ctx, params.JobID)`
- On success: publish deployment succeeded event
- On error: publish deployment failed event
- Clean up work directory with `defer os.RemoveAll(workDir)`
- Return error if any step fails

### 7. Implement Deploy Image Handler

**Implement deploy image handler:**
- Implement `HandleDeployImage(params DeployImageParams, natsClient *nats.Client, logger hclog.Logger) error` function
- Create temporary work directory
- Publish deployment started event via NATS
- Generate HCL config with `BuilderType: "noop"` since image is pre-built
- Set image name from params.ImageName and params.ImageTag in deploy config
- Write HCL to work directory
- Load config and create lifecycle executor
- Execute full lifecycle (build will be skipped due to noop, only deploy runs)
- On success: publish deployment succeeded event
- On error: publish deployment failed event
- Clean up work directory
- Return error if any step fails

### 8. Implement Destroy Job Handler

**Implement destroy job handler:**
- Implement `HandleDestroyJob(params DestroyJobParams, natsClient *nats.Client, logger hclog.Logger) error` function
- Import nomadpack plugin functionality for destroy operations
- Iterate over params.Jobs slice (limit concurrency to 1 job at a time matching cs-runner behavior)
- For each job:
  - Execute nomad-pack destroy command with job.JobID, job.NomadAddress, job.NomadToken
  - On success: publish job destroyed event with job.ServiceID and params.Reason
  - On error: log error but continue to next job (don't fail entire operation)
- Return nil (always succeeds even if individual destroys fail, matching cs-runner behavior)

### 9. Create Dispatch Command

**Implement dispatch command:**
- Create `cmd/cloudstation/dispatch.go`
- Import `os`, `fmt`, `time`, `context`, `github.com/urfave/cli/v2`, `github.com/hashicorp/go-hclog`
- Import internal packages: `dispatch`, `nats`
- Implement `dispatchCommand() *cli.Command` function
- Set `Name: "dispatch"`, `Usage: "Execute as Nomad dispatched job (internal use)"`, `Hidden: true`
- In Action function:
  - Get logger from context (use hclog.Default())
  - Parse task type with `dispatch.ParseTaskType()`
  - Parse parameters with `dispatch.ParseParams(taskType)`
  - Initialize NATS client from env vars `NATS_SERVERS` and `NATS_CLIENT_PRIVATE_KEY`
  - Set up 15-minute timeout matching cs-runner (src/index.ts:19)
  - Create context with timeout: `ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)`
  - Start timeout goroutine that publishes failure event if timeout occurs
  - Route to appropriate handler based on task type
  - Close NATS connection with `defer natsClient.Close()`
  - Return error if handler fails (triggers exit code 1)
- Add comprehensive error logging for debugging

**Register dispatch command:**
- Edit `cmd/cloudstation/main.go`
- Add `dispatchCommand()` to Commands slice after `upCommand()`
- Verify command registration with `cs --help` (should not show dispatch since it's hidden)
- Verify command works with `cs dispatch --help`

### 10. Add Dispatch Handler Tests

**Create dispatch handler tests:**
- Create `internal/dispatch/handlers_test.go`
- Mock NATS client using interface or test doubles
- Test `HandleDeployRepository` with valid parameters
  - Mock git clone operation
  - Mock lifecycle execution
  - Verify NATS events published (started, succeeded)
  - Verify HCL file generated correctly
- Test `HandleDeployRepository` with clone failure
  - Verify NATS failure event published
  - Verify error returned
- Test `HandleDeployImage` with valid parameters
  - Verify noop builder used in generated HCL
  - Verify NATS events published
- Test `HandleDestroyJob` with multiple jobs
  - Mock nomad-pack destroy calls
  - Verify job destroyed events published for each job
  - Verify continues on individual job failures
- Use table-driven tests for different parameter combinations

### 11. Create Integration Tests

**Create integration test file:**
- Create `test/integration/dispatch_test.go`
- Set up test environment with:
  - Mock NATS server or NATS test container
  - Temporary work directories
  - Environment variables (NOMAD_META_TASK, NOMAD_META_PARAMS, NATS_SERVERS, NATS_CLIENT_PRIVATE_KEY)
- Test end-to-end dispatch flow:
  - Set `NOMAD_META_TASK=deploy-repository`
  - Set `NOMAD_META_PARAMS` with valid JSON deployment parameters
  - Execute dispatch command programmatically
  - Verify NATS events received
  - Verify deployment succeeded/failed appropriately
- Test timeout handling:
  - Set very short timeout (1 second)
  - Trigger long-running operation
  - Verify timeout failure event published
- Test all three task types (deploy-repository, deploy-image, destroy-job-pack)
- Clean up test resources in defer statements

### 12. Update Documentation

**Update README.md:**
- Add section "Dispatch Mode" under "Basic Usage"
- Explain dispatch mode is for Nomad job execution (internal use)
- Document environment variables required:
  - `NOMAD_META_TASK` - Task type to execute
  - `NOMAD_META_PARAMS` - JSON-encoded task parameters
  - `NATS_SERVERS` - Comma-separated NATS server addresses
  - `NATS_CLIENT_PRIVATE_KEY` - NKey seed for authentication
- Add example Nomad job template using `cs dispatch`
- Link to migration guide from cs-runner

**Create migration guide:**
- Create `docs/MIGRATION_CS_RUNNER.md`
- Document differences between cs-runner and dispatch mode
- Provide step-by-step migration instructions:
  1. Build new Docker image with `cs` binary
  2. Update Nomad job template to use `cs dispatch`
  3. Update environment variable names if needed
  4. Test in staging environment
  5. Deploy to production
- Document rollback procedure
- Include troubleshooting section

### 13. Build and Test

**Build binary:**
- Run `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator`
- Run `make build` to compile binary
- Verify binary exists: `ls -lh bin/cs`
- Verify binary size < 25MB (with NATS client added)
- Test help output: `./bin/cs --help`
- Test dispatch command exists: `./bin/cs dispatch --help`

**Run unit tests:**
- Run `go test ./pkg/nats/... -v` - Test NATS client
- Run `go test ./pkg/git/... -v` - Test Git operations
- Run `go test ./internal/hclgen/... -v` - Test HCL generation
- Run `go test ./internal/dispatch/... -v` - Test dispatch handlers
- Run `go test ./... -cover` - Run all tests with coverage
- Verify coverage > 70% for new packages

**Run integration tests:**
- Set up test environment (NATS server, environment variables)
- Run `go test ./test/integration/... -v`
- Verify all dispatch scenarios work end-to-end
- Clean up test resources

### 14. Create Docker Image

**Create Dockerfile:**
- Create `Dockerfile.dispatch` in cloudstation-orchestrator root
- Use multi-stage build:
  - Stage 1: Build `cs` binary with `golang:1.21-alpine`
  - Stage 2: Create minimal runtime image with `alpine:latest`
  - Install git and docker CLI in runtime image
  - Copy `cs` binary to `/usr/bin/cs`
  - Set entrypoint to `/usr/bin/cs`
- Keep image size minimal (target < 30MB)

**Build and test image:**
- Run `docker build -f Dockerfile.dispatch -t cloudstation:dispatch .`
- Run `docker images` and verify size < 30MB
- Test image runs: `docker run --rm cloudstation:dispatch --version`
- Test dispatch command: `docker run --rm cloudstation:dispatch dispatch --help`

**Test with mock Nomad environment:**
- Run container with environment variables:
  ```bash
  docker run --rm \
    -e NOMAD_META_TASK=deploy-repository \
    -e NOMAD_META_PARAMS='{"jobId":"test","repository":"user/repo","branch":"main",...}' \
    -e NATS_SERVERS=nats://test:4222 \
    -e NATS_CLIENT_PRIVATE_KEY=... \
    cloudstation:dispatch dispatch
  ```
- Verify container executes and exits with proper status code

### 15. Validation Testing

**Run all validation commands** from the Validation Commands section below to ensure:
- All unit tests pass
- All integration tests pass
- Binary builds successfully
- Docker image builds and runs
- Dispatch mode works with all three task types
- NATS events are published correctly
- Error handling works as expected
- Documentation is complete and accurate

## Testing Strategy

### Unit Tests

**NATS Client Tests:**
- Test client initialization with valid NKey seed
- Test client initialization with invalid NKey seed (should fail)
- Test client initialization with unreachable NATS servers (should return error with retry)
- Test PublishDeploymentStarted creates correct message on correct subject
- Test PublishDeploymentSucceeded creates correct payload
- Test PublishDeploymentFailed creates correct payload
- Test PublishJobDestroyed creates correct payload
- Test Close properly drains and closes connection
- Mock NATS server using test containers or in-memory mock

**Git Clone Tests:**
- Test BuildAuthURL with GitHub provider and token
- Test BuildAuthURL with GitLab provider and token
- Test BuildAuthURL with Bitbucket provider and token (uses x-token-auth)
- Test BuildAuthURL without token (public repos)
- Test Clone with valid repository (mock git command execution)
- Test Clone with invalid repository returns error
- Test Clone with invalid branch returns error
- Test error messages never contain tokens (security)

**HCL Generator Tests:**
- Test GenerateConfig with csdocker builder produces valid HCL
- Test GenerateConfig with nixpacks builder produces valid HCL
- Test GenerateConfig with noop builder (redeploy scenario)
- Test GenerateConfig with nomadpack deployer
- Test GenerateConfig with noop deployer
- Test generated HCL can be parsed by config.LoadConfigFile()
- Test WriteConfigFile creates file at correct path
- Test variable definitions included in generated HCL

**Dispatch Handler Tests:**
- Test HandleDeployRepository with valid params (mock all I/O)
- Test HandleDeployRepository publishes started event before execution
- Test HandleDeployRepository publishes succeeded event on success
- Test HandleDeployRepository publishes failed event on error
- Test HandleDeployImage generates config with noop builder
- Test HandleDestroyJob iterates all jobs and publishes events
- Test HandleDestroyJob continues on individual job failures
- Mock NATS, git, and lifecycle executor for isolation

### Integration Tests

**End-to-End Dispatch Flow:**
- Start real NATS server (or use test container)
- Set NOMAD_META_TASK and NOMAD_META_PARAMS environment variables
- Execute dispatch command
- Verify NATS events received in correct order (started → succeeded/failed)
- Verify work directory cleaned up
- Test with all three task types

**Timeout Handling:**
- Set 1-second timeout
- Trigger operation that takes > 1 second
- Verify timeout failure event published to NATS
- Verify process exits with error code

**Error Recovery:**
- Test with invalid NOMAD_META_PARAMS JSON
- Test with missing required parameters
- Test with NATS connection failure
- Test with Git clone failure
- Test with lifecycle execution failure
- Verify appropriate error messages and NATS failure events

### Edge Cases

**Environment Variables:**
- Missing NOMAD_META_TASK - should return error with helpful message
- Missing NOMAD_META_PARAMS - should return error
- Missing NATS_SERVERS - should return error or skip NATS publishing
- Missing NATS_CLIENT_PRIVATE_KEY - should return authentication error
- Invalid NATS_CLIENT_PRIVATE_KEY format - should return parse error
- Empty NOMAD_META_PARAMS - should return JSON parse error

**Git Operations:**
- Repository doesn't exist - should fail with clear error
- Branch doesn't exist - should fail with clear error
- Authentication failure (invalid token) - should fail
- Network timeout during clone - should fail with timeout error
- Disk space full during clone - should fail with I/O error

**HCL Generation:**
- Missing required deployment parameters - should use defaults or return error
- Invalid builder type - should return error
- Invalid deploy type - should return error
- Empty jobID - should return validation error

**Lifecycle Execution:**
- Build fails - should publish failed event with build error details
- Deploy fails - should publish failed event with deploy error details
- Registry push fails - should publish failed event
- Config validation fails - should publish failed event with validation errors

**NATS Communication:**
- NATS server unreachable - should retry with backoff
- NATS connection drops mid-operation - should attempt reconnect
- JetStream publish fails - should retry or return error
- Message acknowledgment timeout - should retry

**Concurrency:**
- Multiple dispatch commands running simultaneously - should not interfere
- Concurrent NATS publishes - should be thread-safe
- Concurrent Git clones to different directories - should work independently

## Acceptance Criteria

- [ ] NATS client package publishes events to correct subjects (deployment.status.changed, deployment.succeeded, deployment.failed, job.destroyed)
- [ ] NATS client authenticates using NKey seed from environment variable
- [ ] Git clone package supports GitHub, GitLab, and Bitbucket providers
- [ ] Git clone package handles authentication tokens correctly and never logs them
- [ ] HCL generator creates valid cloudstation.hcl files parseable by config.LoadConfigFile()
- [ ] HCL generator supports csdocker, nixpacks, and noop builders
- [ ] HCL generator supports nomadpack and noop deployers
- [ ] Dispatch command reads NOMAD_META_TASK and NOMAD_META_PARAMS environment variables
- [ ] Dispatch command routes to correct handler based on task type (deploy-repository, deploy-image, destroy-job-pack)
- [ ] HandleDeployRepository clones Git repository, generates HCL, and executes lifecycle
- [ ] HandleDeployImage generates HCL with noop builder and executes deploy-only lifecycle
- [ ] HandleDestroyJob destroys all jobs and publishes job.destroyed events
- [ ] All handlers publish deployment.started event before execution
- [ ] All handlers publish deployment.succeeded event on success
- [ ] All handlers publish deployment.failed event on error
- [ ] Dispatch command implements 15-minute timeout matching cs-runner behavior
- [ ] Dispatch command exits with code 0 on success, 1 on failure
- [ ] Dispatch command is hidden from CLI help output
- [ ] Unit tests cover >70% of new code in pkg/nats, pkg/git, internal/hclgen, internal/dispatch
- [ ] Integration tests verify end-to-end dispatch flow with real NATS server
- [ ] Docker image with cs binary is <30MB
- [ ] Docker image includes git and docker CLI tools
- [ ] Documentation includes dispatch mode usage and environment variables
- [ ] Migration guide explains transition from cs-runner to dispatch mode
- [ ] All validation commands execute without errors

## Validation Commands
Execute every command to validate the feature works correctly with zero regressions.

```bash
# 1. Dependency Verification
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go mod verify
go mod tidy
grep "nats-io/nats.go" go.mod  # Verify NATS dependency added
grep "nats-io/nkeys" go.mod    # Verify NKeys dependency added

# 2. Build Verification
make clean
make build
ls -lh bin/cs                  # Verify binary exists and size < 30MB
./bin/cs --version             # Verify version output
./bin/cs --help                # Verify help shows commands (dispatch should be hidden)
./bin/cs dispatch --help       # Verify dispatch command exists and shows help

# 3. Unit Test Execution
go test ./pkg/nats/... -v -cover              # Test NATS client
go test ./pkg/git/... -v -cover               # Test Git operations
go test ./internal/hclgen/... -v -cover       # Test HCL generation
go test ./internal/dispatch/... -v -cover     # Test dispatch handlers
go test ./cmd/cloudstation/... -v             # Test CLI commands

# 4. All Tests with Coverage
go test ./... -race                           # Race condition detection
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out              # Show coverage summary (>70% target)

# 5. Integration Test Execution
# (Requires NATS server running - start with docker-compose or skip if unavailable)
# docker run -d --name nats-test -p 4222:4222 nats:latest
export NATS_SERVERS=nats://localhost:4222
export NATS_CLIENT_PRIVATE_KEY=SUAOYQ6M... # Test NKey seed
go test ./test/integration/... -v             # Run integration tests
# docker stop nats-test && docker rm nats-test

# 6. Code Quality
go fmt ./...                                  # Format all code
go vet ./...                                  # Static analysis
# golangci-lint run                          # Linting (if available)

# 7. Docker Image Build
docker build -f Dockerfile.dispatch -t cloudstation:dispatch .
docker images cloudstation:dispatch           # Verify size < 30MB
docker run --rm cloudstation:dispatch --version
docker run --rm cloudstation:dispatch dispatch --help

# 8. Dispatch Mode Manual Test (Mock Environment)
export NOMAD_META_TASK=deploy-repository
export NOMAD_META_PARAMS='{"jobId":"test-job","repository":"test/repo","branch":"main","deploymentId":"dep-123","serviceId":"svc-123","teamId":"team-123","userId":1,"ownerId":"owner-123","imageName":"test-image","imageTag":"v1.0.0","deploymentJobId":456,"build":{"builder":"noop"},"deploy":"nomadpack","replicaCount":1}'
# Note: This will fail without NATS but should parse parameters correctly
./bin/cs dispatch 2>&1 | grep "failed to initialize NATS" || echo "Test passed"

# 9. Verify Environment Variable Parsing
export NOMAD_META_TASK=invalid-task
./bin/cs dispatch 2>&1 | grep "unknown task"  # Should show error

export NOMAD_META_TASK=deploy-repository
export NOMAD_META_PARAMS='invalid-json'
./bin/cs dispatch 2>&1 | grep "failed to parse"  # Should show parse error

# 10. Git Clone Test (Requires git installed)
# This tests git package can be imported and compiled
go test ./pkg/git/... -run TestBuildAuthURL -v

# 11. HCL Generation Test
# This tests HCL generation produces valid output
go test ./internal/hclgen/... -run TestGenerateConfig -v

# 12. Documentation Verification
cat README.md | grep -A 10 "Dispatch Mode"    # Verify dispatch mode documented
ls -l docs/MIGRATION_CS_RUNNER.md             # Verify migration guide exists
cat docs/MIGRATION_CS_RUNNER.md | head -20    # Preview migration guide

# 13. Regression Testing - Existing Commands Still Work
./bin/cs init --config test-dispatch.hcl
cat test-dispatch.hcl                          # Verify file created
./bin/cs build --help                          # Verify build command works
./bin/cs up --help                             # Verify up command works
rm test-dispatch.hcl                           # Clean up

# 14. Final Build and Size Check
make build
ls -lh bin/cs
du -h bin/cs | awk '{if ($1 ~ /M/) {split($1,a,"M"); if (a[1] < 30) print "PASS: Binary size OK"; else print "FAIL: Binary too large"} else print "PASS: Binary size OK"}'

# 15. Cleanup
unset NOMAD_META_TASK NOMAD_META_PARAMS NATS_SERVERS NATS_CLIENT_PRIVATE_KEY
```

## Notes

### Dependencies Added
- `github.com/nats-io/nats.go` - NATS client library for JetStream event publishing
- `github.com/nats-io/nkeys` - NKeys authentication for secure NATS connections
- Consider adding `github.com/go-git/go-git/v5` if pure Go implementation preferred over shelling out to git CLI

### Migration Strategy
**Phase 1 - Build & Test (Week 1):**
- Implement all dispatch mode functionality
- Test thoroughly with integration tests
- Build Docker image with cs binary

**Phase 2 - Staging Deployment (Week 2):**
- Deploy new cloudstation:dispatch image to staging
- Update staging Nomad job template to use `cs dispatch`
- Run parallel testing: cs-runner (control) vs dispatch mode (test)
- Monitor NATS events match between both systems
- Verify deployment success rates identical

**Phase 3 - Production Rollout (Week 3):**
- Canary deployment: Route 10% of jobs to dispatch mode
- Monitor for 48 hours (error rates, performance, NATS events)
- Increase to 50% if stable
- Full rollout to 100% if no issues
- Keep cs-runner image available for 30-day rollback period

**Phase 4 - Cleanup (Week 4):**
- Remove cs-runner TypeScript codebase
- Update all documentation to reference dispatch mode only
- Archive cs-runner Docker images

### Performance Expectations
Based on analysis, dispatch mode should achieve:
- **Binary size**: 20MB (vs 80MB cs-runner) - 75% reduction
- **Image size**: 30MB (vs 100MB+ cs-runner image) - 70% reduction
- **Startup time**: 100ms (vs 2-3s cs-runner) - 20-30x faster
- **Memory usage**: 50-100MB (vs 200-500MB cs-runner) - 60% reduction
- **Build time**: 10-15s (vs 45-60s cs-runner npm install + tsc) - 75% faster

### Security Considerations
- **Never log tokens**: Ensure Git tokens and NATS keys never appear in logs
- **Redact secrets**: Use hclog redaction for sensitive config values
- **NKey authentication**: More secure than username/password for NATS
- **Minimal image**: Alpine-based image reduces attack surface
- **No npm vulnerabilities**: Eliminating Node.js removes npm supply chain risks

### Backward Compatibility
- NATS event payloads match cs-runner format exactly (same JSON structure)
- Environment variable names match cs-runner expectations
- Backend requires no changes (still dispatches via Nomad with same metadata)
- NOMAD_META_PARAMS JSON format remains identical

### Future Enhancements (Post-MVP)
- **Prometheus metrics**: Add metrics for dispatch task execution (duration, success rate)
- **Structured logging**: Add traceID to correlate logs across lifecycle
- **Retry logic**: Add automatic retry for transient failures (network, NATS)
- **Parallel job destruction**: Increase concurrency for destroy operations
- **Git cache**: Cache cloned repositories to speed up redeployments
- **HCL validation**: Add pre-execution validation of generated HCL
- **Dry-run mode**: Add --dry-run flag to preview actions without executing
