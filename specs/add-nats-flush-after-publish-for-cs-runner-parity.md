# Bug: Missing NATS Flush() After Publish Causes Message Buffering

## Bug Description

The cloudstation-orchestrator NATS client does not call `Flush()` after publishing events to JetStream, which can cause messages to be buffered instead of sent immediately. This creates a parity gap with cs-runner and risks message loss if the connection closes before buffered messages are transmitted.

**Symptoms:**
- NATS messages may be buffered and not sent immediately
- Messages could be lost if the application terminates before buffer flush
- No server-side acknowledgment wait, reducing delivery confidence
- Inconsistent behavior compared to cs-runner reference implementation

**Expected Behavior:**
After publishing to JetStream, the NATS client should:
1. Publish the message and receive JetStream acknowledgment
2. Call `Flush()` to force immediate transmission of buffered messages
3. Wait for server confirmation before proceeding

**Actual Behavior:**
The NATS client currently:
1. Publishes the message and receives JetStream acknowledgment
2. Returns immediately without flushing
3. Leaves messages potentially buffered in the client

## Problem Statement

The `pkg/nats/client.go` publish helper function (lines 92-111) publishes events to JetStream but does not call `c.conn.Flush()` to ensure messages are immediately transmitted to the NATS server. This creates three problems:

1. **Message Loss Risk**: If the connection closes or the application terminates before the buffer flushes, unsent messages are lost
2. **Delivery Uncertainty**: Without flush, there's no guarantee the server received the message
3. **cs-runner Parity Gap**: cs-runner calls `await this.natsConnection.flush()` after every publish (lines 51, 67, 83 in `cs-runner/src/nats.ts`)

## Solution Statement

Add `c.conn.Flush()` call after JetStream publish operations in the `publish()` helper function. This will:
- Force immediate transmission of buffered messages
- Wait for server acknowledgment before returning
- Match cs-runner's behavior exactly
- Eliminate message loss risk during application shutdown

The fix is surgical: add 3-4 lines to the existing `publish()` function to flush the connection after successful publish.

## Steps to Reproduce

1. Start cloudstation-orchestrator with NATS configured
2. Trigger a deployment that publishes NATS events
3. Observe that `Flush()` is never called in NATS client code
4. Compare with cs-runner source which calls `flush()` after every publish
5. Note the parity gap in message delivery guarantees

**Evidence from cs-runner** (`/Users/oumnyabenhassou/Code/runner/cs-runner/src/nats.ts`):
```typescript
// Line 47-51
const ack = await js.publish(subject, sc.encode(JSON.stringify(data)));
console.log(ack);
await this.natsConnection.flush();  // ← cs-runner ALWAYS flushes
```

**Current cloudstation-orchestrator code** (`pkg/nats/client.go:103-107`):
```go
_, err = js.Publish(subject, data)
if err != nil {
    c.logger.Error("Failed to publish event", "subject", subject, "error", err)
    return fmt.Errorf("failed to publish to %s: %w", subject, err)
}
// ← Missing flush() call here
```

## Root Cause Analysis

The root cause is incomplete implementation during the initial NATS client development. When migrating from cs-runner (TypeScript/NATS.js) to cloudstation-orchestrator (Go/nats.go), the `flush()` call was not included in the publish helper function.

**Why this matters:**

1. **Buffering Behavior**: NATS clients buffer outgoing messages for performance. Without explicit flush, messages stay in the buffer until:
   - Buffer fills up (size threshold)
   - Connection flushes automatically (time threshold)
   - Connection closes (forces flush, but may fail if already closing)

2. **JetStream vs Core NATS**: While JetStream provides at-least-once delivery guarantees via publish acknowledgments, the message must first reach the server. If buffered and the connection closes, the message never arrives.

3. **Application Lifecycle**: Deployment orchestrators often start processes, publish events, and exit quickly. Without flush, messages sent just before exit risk being lost.

**cs-runner Reference**: Every publish in cs-runner includes flush:
- `publishDeploymentStartedEvent` (line 51): `await this.natsConnection.flush()`
- `publishDeploymentSucceededEvent` (line 67): `await this.natsConnection.flush()`
- `publishDeploymentFailedEvent` (line 83): `await this.natsConnection.flush()`

**Go nats.go Library**: The `Flush()` method exists and is documented:
> Flush will perform a round trip to the server and return when it receives the internal reply, or an error.

This guarantees the server received all published messages.

## Relevant Files

Use these files to fix the bug:

### Existing Files

- **`pkg/nats/client.go`** (lines 92-111) - NATS client publish helper function
  - Contains the `publish()` helper that publishes to JetStream
  - Missing `c.conn.Flush()` call after successful publish
  - Need to add flush with error handling after line 107
  - Need to add timeout context to prevent indefinite blocking

- **`pkg/nats/client_test.go`** - NATS client unit tests
  - Need to verify flush behavior is tested
  - May need to add mock expectations for Flush() calls
  - Validate error handling for flush failures

- **`pkg/nats/types.go`** - NATS payload type definitions
  - Review to ensure no other changes needed
  - No modifications required for this bug fix

## Step by Step Tasks

IMPORTANT: Execute every step in order, top to bottom.

### 1. Add Flush() Call to publish() Helper Function

- Edit `pkg/nats/client.go`
- Locate the `publish()` function (line 92)
- After the successful `js.Publish()` call (after line 107), add:
  ```go
  // Flush connection to ensure message is sent immediately
  if err := c.conn.Flush(); err != nil {
      c.logger.Warn("Failed to flush NATS connection", "subject", subject, "error", err)
      // Don't return error - message was published to JetStream successfully
      // Flush failure is a warning, not a fatal error
  }
  ```
- This ensures messages are transmitted immediately while avoiding false failures

### 2. Update Debug Logging

- Edit `pkg/nats/client.go`
- Modify the debug log statement on line 109 to:
  ```go
  c.logger.Debug("Published and flushed event", "subject", subject)
  ```
- This confirms both publish and flush completed successfully

### 3. Review Close() Method for Flush Behavior

- Edit `pkg/nats/client.go`
- Review the `Close()` method (lines 114-121)
- Verify `c.conn.Drain()` is called before `c.conn.Close()`
- `Drain()` already flushes, so no changes needed here
- Add comment documenting the relationship between Drain and Flush

### 4. Verify Test Coverage

- Read `pkg/nats/client_test.go`
- Ensure tests don't make assumptions about Flush() not being called
- If tests use mocks, verify Flush() calls won't break them
- Add comment documenting that real NATS connection is required for flush testing

### 5. Run All Validation Commands

- Execute every command in the `Validation Commands` section below
- Verify all tests pass with zero failures
- Verify code compiles without errors
- Verify no regressions in existing functionality

## Validation Commands

Execute every command to validate the bug is fixed with zero regressions.

```bash
# 1. Verify code formatting
go fmt ./pkg/nats/...

# 2. Verify no linting errors in NATS package
go vet ./pkg/nats/...

# 3. Build the binary to ensure compilation succeeds
make clean
make build
ls -lh bin/cs

# 4. Run NATS package tests with race detection
go test ./pkg/nats/... -v -race

# 5. Run all tests to ensure no regressions
go test ./... -race

# 6. Run tests with coverage
go test ./pkg/nats/... -cover

# 7. Verify the fix by inspecting the code
grep -A 3 "js.Publish" pkg/nats/client.go
# Should show Flush() call after Publish()

# 8. Compare with cs-runner implementation
grep -A 3 "js.publish" /Users/oumnyabenhassou/Code/runner/cs-runner/src/nats.ts
# Should show flush() call after publish()

# 9. Check for consistent flush usage across all publish methods
grep -c "Flush()" pkg/nats/client.go
# Should return at least 1 (from publish helper)

# 10. Verify no breaking changes to public API
go build -o /dev/null ./...
```

## Notes

### Why Flush() is Important

From NATS Go client documentation:
> `Flush()` performs a round trip to the server and returns when it receives the internal reply. This is useful to ensure that all published messages have been processed by the server.

### Error Handling Strategy

The implementation treats flush failures as **warnings, not errors**:
- **Rationale**: The message was already successfully published to JetStream and acknowledged
- **Risk Mitigation**: JetStream guarantees at-least-once delivery once acknowledged
- **Graceful Degradation**: Flush failure means "might be delayed" not "message lost"
- **Logging**: Warning-level log allows operators to detect connection issues without breaking deployments

### Performance Impact

Adding `Flush()` has minimal performance impact:
- **Latency**: Adds ~1-5ms per publish (one round trip to server)
- **Throughput**: No impact on message throughput
- **Trade-off**: Slight latency increase for guaranteed delivery is acceptable for deployment events
- **Context**: Deployment events are low-frequency (not high-throughput data streams)

### cs-runner Parity Achieved

After this fix, cloudstation-orchestrator will have 100% parity with cs-runner for NATS publishing:
1. ✅ Correct payload types (jobId: int)
2. ✅ Correct field values (deploymentJobId instead of job name)
3. ✅ Flush after publish (this fix)

### Alternative Considered: FlushTimeout()

The NATS Go client also provides `FlushTimeout(duration)`. We chose `Flush()` because:
- Matches cs-runner's behavior (no timeout specified)
- Simpler implementation
- Default timeout is reasonable for deployment events
- Can be added later if needed without breaking changes

### Future Enhancement

Consider adding a configurable flush timeout via environment variable:
```go
timeout := 5 * time.Second
if envTimeout := os.Getenv("NATS_FLUSH_TIMEOUT"); envTimeout != "" {
    timeout, _ = time.ParseDuration(envTimeout)
}
if err := c.conn.FlushTimeout(timeout); err != nil {
    // handle error
}
```

This is not required for the current bug fix but could be added as a future improvement.
