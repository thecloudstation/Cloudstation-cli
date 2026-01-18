#!/bin/bash
# Real E2E test for cloudstation-orchestrator dispatch
# This tests the NATS log streaming with a real repository clone

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== CloudStation Orchestrator Real E2E Test ==="
echo "Project dir: $PROJECT_DIR"
echo ""

# Configuration
NATS_SERVERS="${NATS_SERVERS:-nats://localhost:4222}"
NATS_STREAM_PREFIX="${NATS_STREAM_PREFIX:-test}"
DEPLOYMENT_ID="${DEPLOYMENT_ID:-e2e-test-$(date +%s)}"

# Test repo - use a simple public repo
TEST_REPO="${TEST_REPO:-https://github.com/expressjs/express}"
TEST_BRANCH="${TEST_BRANCH:-master}"

echo "Configuration:"
echo "  NATS_SERVERS: $NATS_SERVERS"
echo "  NATS_STREAM_PREFIX: $NATS_STREAM_PREFIX"
echo "  DEPLOYMENT_ID: $DEPLOYMENT_ID"
echo "  TEST_REPO: $TEST_REPO"
echo "  TEST_BRANCH: $TEST_BRANCH"
echo ""

# Build minimal params JSON - only what's required for clone phase
# This will test: param parsing, NATS connection, clone phase logging
PARAMS_JSON=$(cat <<EOF
{
  "jobId": "999",
  "deploymentId": "$DEPLOYMENT_ID",
  "serviceId": "svc-test-001",
  "ownerId": "owner-test",
  "userId": 1,
  "deploymentJobId": 999,
  "repository": "$TEST_REPO",
  "branch": "$TEST_BRANCH",
  "build": {
    "builder": "noop"
  },
  "deploy": "simple",
  "imageName": "test-image",
  "imageTag": "latest"
}
EOF
)

echo "Params JSON:"
echo "$PARAMS_JSON" | jq .
echo ""

# Base64 encode
PARAMS_B64=$(echo "$PARAMS_JSON" | base64 -w0)
echo "Base64 params (first 100 chars): ${PARAMS_B64:0:100}..."
echo ""

# Export environment variables
export NOMAD_META_TASK="deploy-repository"
export NOMAD_META_PARAMS="$PARAMS_B64"
export NATS_SERVERS="$NATS_SERVERS"
export NATS_STREAM_PREFIX="$NATS_STREAM_PREFIX"
# No NATS key for unauthenticated local NATS
export NATS_CLIENT_PRIVATE_KEY=""

echo "Environment set:"
echo "  NOMAD_META_TASK=$NOMAD_META_TASK"
echo "  NOMAD_META_PARAMS length: ${#NOMAD_META_PARAMS}"
echo "  NATS_SERVERS=$NATS_SERVERS"
echo "  NATS_STREAM_PREFIX=$NATS_STREAM_PREFIX"
echo ""

# Check binary exists
BINARY="$PROJECT_DIR/bin/cs"
if [ ! -f "$BINARY" ]; then
    echo "ERROR: Binary not found at $BINARY"
    echo "Run: make build"
    exit 1
fi

echo "Binary: $BINARY"
$BINARY --version
echo ""

echo "=== Starting dispatch (will clone repo and stream logs to NATS) ==="
echo "Subscribe to NATS in another terminal to see logs:"
echo "  nats sub \"$NATS_STREAM_PREFIX.build.log.$DEPLOYMENT_ID\""
echo ""

# Run dispatch
$BINARY dispatch 2>&1

echo ""
echo "=== Test completed ==="
