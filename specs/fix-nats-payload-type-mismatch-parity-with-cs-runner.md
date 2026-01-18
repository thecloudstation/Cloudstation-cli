# Chore: Fix NATS Payload Type Mismatch for Complete cs-runner Parity

## Chore Description

The cloudstation-orchestrator currently sends incorrect NATS payload types that don't match cs-runner's expected format, causing database errors in the backend. The backend expects `jobId` as an integer (the `deploymentJobId` field), but cloudstation-orchestrator is sending it as a string (the Nomad job name like "csi-stellaback-main-pbaalnxzegzy").

**Error Symptoms:**
```
QueryFailedError: invalid input syntax for type integer: "csi-stellaback-main-pbaalnxzegzy"
```

**Root Cause:**
- `pkg/nats/types.go` defines `DeploymentEventPayload.JobID` as `string` when it should be `int`
- `internal/dispatch/handlers.go` sends `params.JobID` (job name string) instead of `params.DeploymentJobID` (numeric ID)
- This breaks backend database inserts which expect an integer `deploymentJobId`

**Goal:**
Achieve 100% NATS payload parity with cs-runner by:
1. Changing `JobID` field type from `string` to `int` in NATS types
2. Sending `DeploymentJobID` numeric value instead of job name string
3. Applying the fix to all deployment event publishers (started, succeeded, failed)

## Relevant Files

Use these files to resolve the chore:

### Existing Files

- **`pkg/nats/types.go`** (lines 13-21) - NATS payload type definitions
  - Contains `DeploymentEventPayload` struct with incorrect `JobID` type (string instead of int)
  - Needs type change from `string` to `int` to match cs-runner exactly

- **`internal/dispatch/handlers.go`** (lines 593-627) - NATS event publishers
  - `publishFailure()` function (line 594) - Sends wrong field (`params.JobID` instead of `params.DeploymentJobID`)
  - `publishImageFailure()` function (line 612) - Same issue for image deployments
  - Lines 398-411 - Success event publisher for repository deployments
  - Lines 529-542 - Success event publisher for image deployments
  - All need to use `DeploymentJobID` instead of `JobID`

- **`internal/dispatch/types.go`** (lines 202-295, 298-382) - Parameter type definitions
  - `DeployRepositoryParams` struct has `DeploymentJobID FlexInt` field (line 217)
  - `DeployImageParams` struct - Need to verify if it has `DeploymentJobID` field
  - May need to add `DeploymentJobID` to `DeployImageParams` if missing

- **`pkg/nats/client.go`** (lines 66-74) - NATS client methods
  - `PublishDeploymentStarted()` already uses `int` correctly (line 67)
  - `PublishDeploymentSucceeded()` and `PublishDeploymentFailed()` use `DeploymentEventPayload`
  - No changes needed, but validate it works correctly after type change

- **`pkg/nats/client_test.go`** - NATS client unit tests
  - Must be updated to reflect `JobID` type change from string to int
  - Test data and assertions need to use integer values

## Step by Step Tasks

IMPORTANT: Execute every step in order, top to bottom.

### 1. Update NATS Payload Type Definition

- Edit `pkg/nats/types.go`
- Change line 14 from:
  ```go
  JobID        string `json:"jobId"`
  ```
  To:
  ```go
  JobID        int `json:"jobId"`
  ```
- This matches cs-runner's TypeScript definition: `jobId: number;`
- Verify the type change is consistent with `DeploymentStatusPayload.JobID` which is already `int`

### 2. Add DeploymentJobID to DeployImageParams (If Missing)

- Edit `internal/dispatch/types.go`
- Check if `DeployImageParams` struct (starting line 298) has `DeploymentJobID` field
- If missing, add after line 305 (after `OwnerID`):
  ```go
  DeploymentJobID FlexInt    `json:"deploymentJobId"`
  ```
- This ensures parity with `DeployRepositoryParams` which has this field

### 3. Fix Repository Deployment Failure Publisher

- Edit `internal/dispatch/handlers.go`
- In `publishFailure()` function (line 594), change line 597 from:
  ```go
  JobID:        params.JobID,
  ```
  To:
  ```go
  JobID:        int(params.DeploymentJobID),
  ```
- This sends the numeric deployment job ID instead of the Nomad job name string

### 4. Fix Image Deployment Failure Publisher

- Edit `internal/dispatch/handlers.go`
- In `publishImageFailure()` function (line 612), change line 615 from:
  ```go
  JobID:        params.JobID,
  ```
  To:
  ```go
  JobID:        int(params.DeploymentJobID),
  ```
- Ensures image deployments also send the correct numeric ID

### 5. Fix Repository Deployment Success Publisher

- Edit `internal/dispatch/handlers.go`
- Around line 398-411, locate the repository success event publisher
- Find the line setting `JobID: params.JobID` in the `DeploymentEventPayload`
- Change from:
  ```go
  JobID:        params.JobID,
  ```
  To:
  ```go
  JobID:        int(params.DeploymentJobID),
  ```

### 6. Fix Image Deployment Success Publisher

- Edit `internal/dispatch/handlers.go`
- Around line 529-542, locate the image success event publisher
- Find the line setting `JobID: params.JobID` in the `DeploymentEventPayload`
- Change from:
  ```go
  JobID:        params.JobID,
  ```
  To:
  ```go
  JobID:        int(params.DeploymentJobID),
  ```

### 7. Update NATS Client Unit Tests

- Edit `pkg/nats/client_test.go`
- Find all test cases that create `DeploymentEventPayload` structs
- Change `JobID` values from string literals to integer literals
- Example change:
  ```go
  // From:
  JobID: "test-job-123",

  // To:
  JobID: 123,
  ```
- Update all test assertions to expect integer values instead of strings

### 8. Run All Validation Commands

- Execute every command in the `Validation Commands` section below
- Verify all tests pass with zero failures
- Verify build completes successfully
- Verify no regressions in existing functionality

## Validation Commands

Execute every command to validate the chore is complete with zero regressions.

```bash
# 1. Verify code formatting
cd /Users/oumnyabenhassou/Code/runner/cloudstation-orchestrator
go fmt ./...

# 2. Verify no linting errors
go vet ./...

# 3. Build binary successfully
make clean
make build
ls -lh bin/cs

# 4. Run NATS package tests
go test ./pkg/nats/... -v -race

# 5. Run dispatch handler tests
go test ./internal/dispatch/... -v -race

# 6. Run all tests with race detection
go test ./... -race

# 7. Run tests with coverage
go test ./... -cover

# 8. Verify binary runs without errors
./bin/cs --version
./bin/cs --help

# 9. Test type compatibility (should compile without errors)
go build -o /dev/null ./...

# 10. Check for any remaining string usage of jobId in NATS code
grep -r "JobID.*string" pkg/nats/ internal/dispatch/
# Should return no results after fix
```

## Notes

### cs-runner Reference Implementation

From `cs-runner/src/nats.ts` (line 5-13):
```typescript
export type DeploymentEventPayload = {
  jobId: number;  // ← INTEGER type
  type: "git_repo" | "image";
  deploymentId: string;
  serviceId: string;
  teamId?: string;
  userId?: string | null;
  ownerId: string;
};
```

From `cs-runner/src/deployRepository.ts` (line 79-87):
```typescript
const payload: DeploymentEventPayload = {
  type: "git_repo",
  deploymentId: options.deploymentId,
  serviceId: options.serviceId,
  teamId: options.teamId,
  userId: options.userId ? options.userId.toString() : null,
  ownerId: options.ownerId,
  jobId: options.deploymentJobId!,  // ← Uses deploymentJobId (number)
};
```

### Backend Database Expectation

The backend database schema expects:
- Field name: `jobId` (camelCase in JSON)
- Field type: INTEGER (numeric deployment job ID, not job name string)
- Value source: `deploymentJobId` parameter from the deployment request

### Why This Matters

The job name (`params.JobID`) and deployment job ID (`params.DeploymentJobID`) are different:
- **JobID**: String like "csi-stellaback-main-pbaalnxzegzy" (Nomad job identifier)
- **DeploymentJobID**: Integer like `1` or `42` (database primary key for deployment_jobs table)

The backend uses `DeploymentJobID` to:
1. Track deployment status in the `deployment_jobs` table
2. Link deployments to services and projects
3. Query deployment history and metrics

Sending the wrong value causes database insert failures and breaks the deployment tracking system.

### Backward Compatibility

This change maintains backward compatibility because:
- The JSON field name (`jobId`) remains the same
- Only the type changes from string to int
- The backend has always expected an integer
- This fixes a bug, not a breaking change

### Testing Strategy

After implementing the fix:
1. Unit tests validate type correctness
2. Integration tests verify NATS message format
3. Manual testing with real deployments confirms backend accepts the payload
4. Monitor backend logs for absence of "invalid input syntax for type integer" errors
