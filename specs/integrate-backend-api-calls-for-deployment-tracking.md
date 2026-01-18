# Chore: Integrate Backend API Calls for Deployment Tracking

## Chore Description

Integrate CloudStation backend API calls into the deployment handlers to enable real-time deployment progress tracking, domain allocation, and service configuration synchronization. This implementation follows the same flow as cs-runner but uses clean Go code instead of bash scripts and Waypoint hooks.

The integration must support both deployment types (repository and image) and handle the complete deployment lifecycle: build, push/registry, image_download, deploy, and healthCheck steps. The backend uses the deployment ID (e.g., "gdeployment_...") to track progress, which is already available in the deployment parameters.

## Relevant Files

Use these files to resolve the chore:

- **cloudstation-orchestrator/internal/dispatch/handlers.go** - Contains HandleDeployRepository() and HandleDeployImage() functions where backend API calls need to be integrated at each lifecycle phase (build, registry, deploy, healthCheck).

- **cloudstation-orchestrator/internal/dispatch/types.go** - Contains DeployRepositoryParams and DeployImageParams structs. Need to add BackendURL and AccessToken fields to enable backend API integration.

- **cloudstation-orchestrator/pkg/backend/client.go** - Already implements the HTTP client with AskDomain(), UpdateService(), and UpdateDeploymentStep() methods. Ready to use, no changes needed.

- **cloudstation-orchestrator/pkg/backend/types.go** - Defines DeploymentStep constants (clone, build, registry, deploy, release) and DeploymentStatus constants (in_progress, completed, failed). Backend expects "push" and "healthCheck" steps which need to be added.

- **cloudstation-orchestrator/internal/lifecycle/executor.go** - Contains the Build(), Registry(), and Deploy() execution logic. Backend API tracking calls should be added before/after these operations.

- **cloudstation-orchestrator/pkg/portdetector/detector.go** - Detects exposed ports from Docker images. Integration point for ask-domain and service-update API calls after port detection.

- **cloudstation-orchestrator/internal/dispatch/params.go** - Parses deployment parameters from base64-encoded NOMAD_META_PARAMS. Needs to extract backendUrl and accessToken from environment.

### New Files

None required - all necessary HTTP client code already exists in pkg/backend/.

## Step by Step Tasks

IMPORTANT: Execute every step in order, top to bottom.

### Step 1: Add Missing Deployment Steps to Backend Types

- Add missing step constants to pkg/backend/types.go to match what cs-runner sends and what the backend expects
- Add `StepPush DeploymentStep = "push"` (for registry/push phase tracking)
- Add `StepImageDownload DeploymentStep = "image_download"` (for image pull tracking in deploy-image flow)
- Add `StepHealthCheck DeploymentStep = "healthCheck"` (for health check phase tracking)
- These match the steps that deployment-step.sh sends in cs-runner and what the backend's updateGitDeploymentStep() expects

### Step 2: Add Backend Configuration to Deployment Parameter Types

- Add BackendURL and AccessToken fields to DeployRepositoryParams struct in internal/dispatch/types.go
- Add BackendURL and AccessToken fields to DeployImageParams struct in internal/dispatch/types.go
- Use `json:"backendUrl,omitempty"` and `json:"accessToken,omitempty"` tags to match the parameter naming convention
- These will be populated from BACKEND_URL and ACCESS_TOKEN environment variables via params.go

### Step 3: Update Parameter Parsing to Extract Backend Configuration

- Modify internal/dispatch/params.go to read BACKEND_URL and ACCESS_TOKEN from environment
- Populate params.BackendURL and params.AccessToken when parsing deployment parameters
- Handle missing values gracefully (backend integration is optional - if not provided, deployment continues without tracking)

### Step 4: Create Helper Function for Deployment Step Tracking

- Add a helper function `trackDeploymentStep()` in internal/dispatch/handlers.go that wraps backend client calls
- Function signature: `trackDeploymentStep(client *backend.Client, deploymentID, deploymentType string, step backend.DeploymentStep, status backend.DeploymentStatus, logger hclog.Logger)`
- Automatically converts params.DeploymentID (e.g., "gdeployment_...") to the correct format
- Handles nil client gracefully (no-op if backend integration disabled)
- Logs warnings if API calls fail but doesn't fail the deployment (graceful degradation)

### Step 5: Integrate Backend Tracking in HandleDeployRepository

- Create backend client at the start of HandleDeployRepository() using params.BackendURL and params.AccessToken
- Track "build" step:
  - Call `trackDeploymentStep(..., "build", StatusInProgress)` before build phase
  - Call `trackDeploymentStep(..., "build", StatusCompleted)` after successful build
  - Call `trackDeploymentStep(..., "build", StatusFailed)` on build error
- Track "push" step (registry phase):
  - Call `trackDeploymentStep(..., "push", StatusInProgress)` before registry push
  - Call `trackDeploymentStep(..., "push", StatusCompleted)` after successful push
  - Call `trackDeploymentStep(..., "push", StatusFailed)` on push error
- Track "deploy" step:
  - Call `trackDeploymentStep(..., "deploy", StatusInProgress)` before Nomad deployment
  - Call `trackDeploymentStep(..., "deploy", StatusCompleted)` after successful deployment
  - Call `trackDeploymentStep(..., "deploy", StatusFailed)` on deployment error
- Track "healthCheck" step:
  - Call `trackDeploymentStep(..., "healthCheck", StatusInProgress)` before health checks
  - Call `trackDeploymentStep(..., "healthCheck", StatusCompleted)` after successful health checks
  - Call `trackDeploymentStep(..., "healthCheck", StatusFailed)` on health check failure
- Use deploymentType = "git" for repository deployments

### Step 6: Integrate Backend Tracking in HandleDeployImage

- Create backend client at the start of HandleDeployImage() using params.BackendURL and params.AccessToken
- Track "image_download" step:
  - Call `trackDeploymentStep(..., "image_download", StatusInProgress)` before image inspection
  - Call `trackDeploymentStep(..., "image_download", StatusCompleted)` after successful inspection
  - Call `trackDeploymentStep(..., "image_download", StatusFailed)` on inspection error
- Track "deploy" step (same as HandleDeployRepository deploy phase)
- Track "healthCheck" step (same as HandleDeployRepository healthCheck phase)
- Use deploymentType = "app" for image deployments (matching backend's deployment_type enum)

### Step 7: Integrate Domain Allocation (AskDomain)

- After port detection in both handlers, call backendClient.AskDomain(params.ServiceID) for each detected port
- Collect allocated subdomains in network configuration
- Use allocated domains when building Nomad job specification
- Handle API failures gracefully - if domain allocation fails, log warning and continue with default domain generation
- This matches cs-runner's vars.sh:78 pattern where it calls ask-domain for each detected port

### Step 8: Integrate Service Configuration Sync (UpdateService)

- After port detection and domain allocation, call backendClient.UpdateService() with detected metadata
- Include detected ports with allocated domains in Network field
- Include detected DockerUser if available
- Include detected CMD and Entrypoint if available
- This should be called before Nomad deployment, after all service metadata is collected
- Handle API failures gracefully - log warning but don't fail deployment
- This matches cs-runner's vars.sh pattern where it calls service-update after inspecting the image

### Step 9: Add Error Handling with Graceful Degradation

- Ensure all backend API calls use non-blocking error handling
- If backend client creation fails, log warning and continue deployment without tracking
- If individual API calls fail, log warning and continue to next deployment phase
- Never fail a deployment due to backend API failures (backend is for tracking only, not critical path)
- Match cs-runner's pattern: `curl ... || true` (don't fail on API errors)

### Step 10: Run Validation Commands

- Execute all validation commands listed below to ensure the integration works correctly
- Verify zero regressions in existing functionality
- Confirm backend API calls are made at correct lifecycle points

## Validation Commands

Execute every command to validate the chore is complete with zero regressions.

- `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator && go build ./...` - Build all packages to verify no compilation errors
- `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator && go test ./pkg/backend/...` - Run backend package tests to verify HTTP client works correctly
- `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator && go test ./internal/dispatch/...` - Run dispatch package tests to verify handlers work correctly
- `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator && go test ./...` - Run all tests to ensure zero regressions across the codebase
- `cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator && go vet ./...` - Run go vet to catch potential issues

## Notes

- The backend expects deployment_type as "git" for repository deployments and "app" for image deployments (matching the backend's DeploymentUpdateDto type)
- The backend's updateGitDeploymentStep() looks up deployments by EITHER `id` OR `n_deployment_id`, so using params.DeploymentID (the "gdeployment_..." primary key) will work immediately
- cs-runner uses deployment-step.sh script called via Waypoint hooks - we're implementing the same functionality natively in Go with direct HTTP calls
- cs-runner tracks steps: build, push, healthCheck - we must match these exact step names for backend compatibility
- The backend step mapping (from events.service.ts): "Downloading image" → "image_download", "Task started by client" → "healthCheck"
- Backend API integration is optional - if BACKEND_URL or ACCESS_TOKEN not provided, deployment continues normally without tracking
- All backend API calls should use graceful degradation - log warnings on failure but never fail the deployment
- The UpdateDeploymentStep API uses PUT /api/local/deployment-step/update with accessToken query parameter
- The AskDomain API uses GET /api/local/ask-domain with serviceId and accessToken query parameters
- The UpdateService API uses PUT /api/local/service-update with accessToken query parameter
- Status values must match backend expectations: "start" and "end" in cs-runner map to "in_progress" and "completed" in our implementation
