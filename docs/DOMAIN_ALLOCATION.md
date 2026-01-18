# Domain Allocation Logic

## Overview

The CloudStation Orchestrator intelligently allocates domains for services based on detected ports and user preferences. This document explains the domain allocation priority system, code flow, and the critical fix that ensures user-specified domains are never overridden by auto-allocated domains.

## Problem Statement (Historical Context)

Prior to the fix, a critical bug existed where user-specified domains were being overwritten by backend-allocated domains. The root cause was:

1. `AskDomain` was called unconditionally for ALL detected ports, even when users provided domains
2. `UpdateService` sent `allocatedDomains` instead of the complete `params.Networks`
3. This caused user domains to be lost during the database synchronization

**Old Buggy Flow:**
```
User specifies domain "api" for port 3000
    |
    v
AskDomain called for ALL ports --> allocates "cst-abc123"
    |
    v
params.Networks[0].Domain = "api" (preserved in memory)
    |
    v
UpdateService sends allocatedDomains[3000] = "cst-abc123"  [WRONG]
    |
    v
Database stores "cst-abc123" instead of "api"  [USER DOMAIN LOST]
```

## Allocation Rules (Priority System)

The domain allocation follows a strict priority system:

### 1. User-Specified Domains (Highest Priority)

When a user explicitly provides a domain value in `params.Networks[i].Domain`:
- The domain is **preserved as-is**
- `AskDomain` is **NOT called** for this port
- The user domain is sent directly to the backend via `UpdateService`

### 2. Auto-Allocated Domains (For Empty Domains Only)

When `params.Networks[i].Domain` is empty (or the port is detected but not in Networks):
- `AskDomain` API is called to allocate a subdomain
- If `ClusterDomain` is set, the subdomain is combined: `{subdomain}.{cluster_domain}`
- The allocated domain is stored in `params.Networks[i].Domain`

### 3. UpdateService Synchronization

After domain allocation completes:
- `UpdateService` sends the complete `params.Networks` array to cs-backend
- This includes **both** user-specified and auto-allocated domains
- The backend receives a single source of truth

## Code Flow

### Correct Flow (After Fix)

```
                              START
                                |
                                v
                    +------------------------+
                    | For each detected port |
                    +------------------------+
                                |
                                v
               +--------------------------------+
               | hasUserProvidedDomainForPort? |
               +--------------------------------+
                       |              |
                      YES             NO
                       |              |
                       v              v
              +-------------+   +-----------------+
              | Skip        |   | Call AskDomain  |
              | AskDomain   |   | Allocate domain |
              | Log: user   |   +-----------------+
              | domain      |           |
              | preserved   |           v
              +-------------+   +------------------+
                       |        | Append cluster   |
                       |        | domain if needed |
                       |        +------------------+
                       |              |
                       v              v
               +---------------------------+
               | Update params.Networks    |
               | with allocated domains    |
               | (only for empty domains)  |
               +---------------------------+
                            |
                            v
               +----------------------------+
               | UpdateService sends        |
               | complete params.Networks   |
               | to cs-backend              |
               +----------------------------+
                            |
                            v
                          END
```

### Handler-Specific Flows

Both `HandleDeployRepository` and `HandleDeployImage` follow the same pattern:

```
HandleDeployRepository                  HandleDeployImage
        |                                      |
        v                                      v
  Build artifact                         Detect ports from image
  (exposes ports)                              |
        |                                      v
        v                               +------+------+
  +-----+-----+                         | Zero-config?|
  |           |                         +------+------+
  v           v                           YES    NO
Port from  Port in                         |     |
build      params.Networks                 v     v
  |           |                      Create    Use existing
  v           v                      Networks  Networks
Check user domain                          |     |
for each port                              v     v
        |                            Domain allocation
        v                            (same logic)
Domain allocation
(skip if user domain)
        |
        v
UpdateService sync
(params.Networks)
```

## Implementation Details

### Helper Function

**Location:** `handlers.go:101-109`

```go
// hasUserProvidedDomainForPort checks if the user provided a domain for the given port
func hasUserProvidedDomainForPort(networks []NetworkPortSettings, port int) (bool, string) {
    for _, network := range networks {
        if int(network.PortNumber) == port && network.Domain != "" {
            return true, network.Domain
        }
    }
    return false, ""
}
```

### Domain Allocation Logic

**Location (Repository):** `handlers.go:338-371`
**Location (Image):** `handlers.go:670-703`

Key behavior:
- Iterates over detected/exposed ports
- Checks if user provided a domain for each port
- Only calls `AskDomain` when user domain is empty
- Combines subdomain with cluster domain when needed
- Updates `params.Networks` with allocated domains

### UpdateService Synchronization

**Location (Repository):** `handlers.go:411-465`
**Location (Image):** `handlers.go:732-778`

Key behavior:
- Builds `networkConfigs` from `params.Networks` (not `allocatedDomains`)
- Sends complete network configuration to backend
- Includes all domain information (user + allocated)
- Logs each network config for visibility

## Testing

Comprehensive tests are located in `handlers_test.go`:

### TestHandleDeployRepository_PreservesUserDomains
Verifies that user-provided domain values are NOT overwritten by allocated domains.

### TestHandleDeployRepository_MixedDomainScenario
Tests a realistic scenario where some ports have user-provided domains and others need allocation.

### TestHandleDeployRepository_AllocatesEmptyDomains
Verifies that when users don't provide a domain, allocation works correctly.

### TestDomainPreservation_*
Additional focused tests for domain preservation:
- `TestDomainPreservation_UserProvidedDomainsNotOverwritten`
- `TestDomainPreservation_EmptyDomainsGetAllocated`
- `TestDomainPreservation_MixedScenario`

## Examples

### Example 1: Zero-Config Deployment (All Auto-Allocated)

User deploys without specifying any network configuration:

```json
{
  "networks": []
}
```

Result:
- Ports are detected from build artifact or image
- `AskDomain` is called for each detected port
- All domains are auto-allocated
- `params.Networks` is populated with allocated domains

```
Port 3000 --> AskDomain --> "cst-xyz123.cluster.io"
Port 8080 --> AskDomain --> "cst-abc456.cluster.io"
```

### Example 2: Explicit User Domains (No Allocation Needed)

User specifies domains for all ports:

```json
{
  "networks": [
    {"port_number": 3000, "domain": "api"},
    {"port_number": 8080, "domain": "admin"}
  ]
}
```

Result:
- `AskDomain` is **never called**
- User domains are preserved exactly as specified
- `UpdateService` sends: `api` for port 3000, `admin` for port 8080

### Example 3: Mixed Scenario (User + Auto-Allocated)

User specifies domain for one port but not another:

```json
{
  "networks": [
    {"port_number": 3000, "domain": "api"},
    {"port_number": 9090, "domain": ""}
  ]
}
```

Result:
- Port 3000: User domain `api` is preserved, `AskDomain` NOT called
- Port 9090: `AskDomain` is called, allocates `cst-def789.cluster.io`
- `UpdateService` sends both domains correctly

```
Port 3000 --> User domain "api" --> Preserved
Port 9090 --> AskDomain --> "cst-def789.cluster.io"
```

## Edge Cases

### Empty params.Networks (Zero-Config)

When `params.Networks` is empty:
- Network entries are created for each detected port
- All domains are auto-allocated
- Default health check settings are applied

### All Ports Have User Domains

When all ports in `params.Networks` have non-empty domains:
- `AskDomain` is **never called** (no unnecessary API calls)
- All user domains are preserved
- Performance optimization: reduces backend calls

### User Domain Contains Dots (e.g., "api.custom.com")

User can specify fully qualified domains:
- Domain is preserved as-is
- Cluster domain is **not** appended
- Allows custom domain routing

### Cluster Domain Already in Subdomain

If `AskDomain` returns a subdomain that already includes the cluster domain:
- Cluster domain is **not** appended again (prevents double-suffix)
- Uses `strings.HasSuffix` check

### AskDomain Failure

If `AskDomain` API call fails:
- Warning is logged
- Port is skipped (graceful degradation)
- Deployment continues with remaining ports
- Does not block the entire deployment

## Logging

Domain allocation decisions are logged for operational visibility:

```
level=INFO msg="Checking which ports need domain allocation" port_count=2
level=INFO msg="User provided domain for port, skipping allocation" port=3000 domain=api
level=INFO msg="Domain allocated for port" port=9090 domain=cst-xyz.cluster.io
level=INFO msg="Syncing service configuration to backend" networkCount=2
level=INFO msg="Network config for backend sync" port=3000 domain=api type=http
level=INFO msg="Network config for backend sync" port=9090 domain=cst-xyz.cluster.io type=http
level=INFO msg="Service configuration synced successfully" serviceId=svc-123 networksSynced=2
```

## Related Files

| File | Purpose |
|------|---------|
| `internal/dispatch/handlers.go` | Domain allocation implementation |
| `internal/dispatch/handlers_test.go` | Test cases for domain preservation |
| `pkg/backend/client.go` | `AskDomain` and `UpdateService` methods |
| `internal/dispatch/types.go` | `NetworkPortSettings` struct definition |
| `pkg/backend/types.go` | `UpdateServiceRequest` and `NetworkConfig` structs |

## Acceptance Criteria

The domain allocation fix meets these criteria:

- [x] `AskDomain` is only called for ports without user-specified domains
- [x] `UpdateService` sends complete `params.Networks` (not `allocatedDomains`)
- [x] User-specified domain "api" for port 3000 reaches the database as "api"
- [x] Mixed scenario: user domain + allocated domain both reach database correctly
- [x] Zero-config deployment still works (all domains auto-allocated)
- [x] All existing tests pass
- [x] New tests cover mixed domain scenarios
- [x] Logs show domain allocation decisions clearly
- [x] Both `HandleDeployRepository` and `HandleDeployImage` are fixed
- [x] No regression in HCL generation or Nomad job specs
