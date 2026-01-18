# Backend API Integration Testing

This document explains how to test the CloudStation Backend API integration.

## Backend Configuration

The CloudStation Orchestrator integrates with the backend API using these environment variables:

```bash
BACKEND_URL="http://10.225.142.179:22593"
ACCESS_TOKEN="73e16d55f4fca8e76a608f1eda58f6f530b5b1a859d558a104cc722da0ac7d740727969dfa0523aaf77ef23832db9fcd9ee7"
```

## Testing Methods

### 1. Unit Tests (Already Passing ✅)

Unit tests use mock HTTP servers and verify the client implementation:

```bash
cd cloudstation-orchestrator
go test ./pkg/backend/... -v
```

**Status**: ✅ All 5 test suites passing (22 seconds)

### 2. Integration Tests

Integration tests require network access to the real backend server:

```bash
# Set environment variables
export BACKEND_URL="http://10.225.142.179:22593"
export ACCESS_TOKEN="73e16d55f4fca8e76a608f1eda58f6f530b5b1a859d558a104cc722da0ac7d740727969dfa0523aaf77ef23832db9fcd9ee7"

# Run integration tests (requires network access to backend)
go test -tags=integration ./pkg/backend/... -v
```

**Status**: ⚠️ Requires network access to backend server (currently unreachable from local machine)

### 3. Manual curl Testing

Test backend API endpoints directly:

```bash
# Test 1: Domain Allocation
curl -X GET "http://10.225.142.179:22593/api/local/ask-domain?serviceId=test-service-123&accessToken=${ACCESS_TOKEN}"

# Expected response:
# {"domain":"subdomain123"}

# Test 2: Service Update
curl -X PUT "http://10.225.142.179:22593/api/local/service-update/?accessToken=${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "serviceId": "test-service-123",
    "network": [
      {"port": 3000, "domain": "subdomain123.cluster.io"}
    ],
    "docker_user": "appuser",
    "cmd": "npm start",
    "entrypoint": "/bin/sh -c"
  }'

# Expected response:
# "updated!" or {"message": "updated!"}

# Test 3: Deployment Step Tracking
curl -X PUT "http://10.225.142.179:22593/api/local/deployment-step/update?accessToken=${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "deploymentId": "test-deployment-123",
    "deployment_type": "repository",
    "step": "build",
    "status": "completed"
  }'

# Expected response:
# {"success": true} or similar success message
```

**Status**: ⚠️ Backend server not reachable (connection timeout)

### 4. End-to-End Deployment Test

Test with a real deployment using dispatch mode:

```bash
# Create a test deployment config
export NOMAD_META_TASK="deploy-repository"
export NOMAD_META_PARAMS="<base64-encoded-json-with-backend-config>"

# The params JSON should include:
# {
#   "backendUrl": "http://10.225.142.179:22593",
#   "accessToken": "73e16d55f4fca8e76a608f1eda58f6f530b5b1a859d558a104cc722da0ac7d740727969dfa0523aaf77ef23832db9fcd9ee7",
#   ... other deployment params ...
# }

# Run orchestrator in dispatch mode
./bin/cs dispatch
```

## Expected Behavior

### With Backend Available

1. **Domain Allocation**: After build completes, orchestrator calls `/api/local/ask-domain` for each detected port
2. **Service Update**: After build, orchestrator syncs metadata to `/api/local/service-update/`
3. **Step Tracking**: Throughout deployment, orchestrator updates progress via `/api/local/deployment-step/update`

### With Backend Unavailable (Graceful Degradation)

1. **No Failures**: Deployment continues normally even if backend is down
2. **Warning Logs**: Backend errors logged as warnings, not errors
3. **No Domain Allocation**: Ports exposed without allocated domains
4. **No Progress Tracking**: UI won't show real-time deployment status

## Troubleshooting

### Connection Timeout

**Symptom**: `Failed to connect to 10.225.142.179 port 22593 after 5000 ms: Timeout was reached`

**Possible Causes**:
- Backend server is on a different network/VPN
- Firewall blocking port 22593
- Backend service not running
- IP/port configuration changed

**Solutions**:
1. Verify network connectivity: `ping 10.225.142.179`
2. Check if port is open: `nc -zv 10.225.142.179 22593`
3. Verify backend service status
4. Test from a machine on the same network as the backend

### Authentication Errors

**Symptom**: `401 Unauthorized` or `403 Forbidden`

**Solutions**:
1. Verify `ACCESS_TOKEN` is correct and not expired
2. Check token has proper permissions for all 3 endpoints
3. Verify token format (should be passed as query parameter)

### Invalid Responses

**Symptom**: JSON parsing errors or unexpected response format

**Solutions**:
1. Check backend API version matches expected format
2. Verify endpoint URLs are correct
3. Review backend logs for errors
4. Test endpoints manually with curl to see actual response format

## Test Results Summary

| Test Type | Status | Notes |
|-----------|--------|-------|
| Unit Tests | ✅ Passing | All 5 test suites pass (22s runtime) |
| Integration Tests | ⚠️ Skipped | Requires network access to backend |
| Manual curl Tests | ⚠️ Failed | Backend not reachable from local machine |
| Code Quality | ✅ Passing | go vet, formatting, race detector all pass |
| Build | ✅ Success | Binary compiles and runs correctly |

## Next Steps for Complete Testing

To fully validate the backend integration:

1. **Deploy to Environment with Backend Access**: Run the orchestrator in an environment that has network access to `10.225.142.179:22593`

2. **Run Integration Tests**: Execute the integration test suite with proper backend access

3. **Real Deployment Test**: Trigger an actual deployment with backend integration enabled and verify:
   - Domains are allocated for detected ports
   - Service metadata appears in backend database
   - Deployment progress is visible in CloudStation UI

4. **Error Scenario Testing**: Test behavior when:
   - Backend returns 500 errors
   - Backend is slow (>30s response time)
   - Network partitions during deployment
   - Invalid token provided
