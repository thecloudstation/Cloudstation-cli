#!/usr/bin/env bash
# Do not use set -e as we want to collect all test results before exiting
set -uo pipefail

# validate-network-migration.sh
# Validates that network configuration bug fixes are working correctly
# Usage: ./scripts/validate-network-migration.sh

echo "========================================="
echo "Network Configuration Migration Validator"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Function to print test result
print_result() {
    local test_name="$1"
    local result="$2"
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}[PASS]${NC} $test_name"
        ((PASSED++)) || true
    else
        echo -e "${RED}[FAIL]${NC} $test_name"
        ((FAILED++)) || true
    fi
}

# Function to run a go test and capture result
run_go_test() {
    local test_pattern="$1"
    local test_desc="$2"
    local output

    output=$(go test ./internal/hclgen/... -run "$test_pattern" -v 2>&1)
    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        print_result "$test_desc" "PASS"
    else
        print_result "$test_desc" "FAIL"
        # Check for specific error types
        if echo "$output" | grep -q "build failed"; then
            echo -e "  ${YELLOW}Note: Test build failed - check for duplicate declarations${NC}"
        fi
    fi
}

# Change to script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "Step 1: Validating build..."
if go build ./cmd/cloudstation > /dev/null 2>&1; then
    print_result "Build compiles successfully" "PASS"
else
    print_result "Build compiles successfully" "FAIL"
    echo -e "${RED}FATAL: Cannot proceed without successful build${NC}"
    exit 1
fi

echo ""
echo "Step 2: Pre-checking test compilation..."
# Check if tests compile before running them
TEST_BUILD_OUTPUT=$(go test -c ./internal/hclgen/... 2>&1 || true)
if echo "$TEST_BUILD_OUTPUT" | grep -q "build failed\|redeclared"; then
    print_result "Test compilation check" "FAIL"
    echo -e "  ${YELLOW}Warning: Tests have compilation issues${NC}"
    echo "$TEST_BUILD_OUTPUT" | grep -E "redeclared|duplicate" | head -5 | sed 's/^/  /'
else
    print_result "Test compilation check" "PASS"
fi

echo ""
echo "Step 3: Running critical bug fix tests..."

# Test 1: Health check path default
run_go_test "TestGenerateNetworking_DefaultHealthCheckPath" "Health check path defaults to '/' not '30s'"

# Test 2: Public field override removed
run_go_test "TestGenerateNetworking_PublicFalseWithHTTP" "Public=false preserved for HTTP ports"

# Test 3: Custom health check paths preserved
run_go_test "TestGenerateNetworking_CustomHealthCheckPath" "Custom health check paths preserved"

# Test 4: Invalid health check types normalized
run_go_test "TestGenerateNetworking_InvalidHealthCheckType" "Invalid health check types normalized"

echo ""
echo "Step 4: Running integration tests..."

# Test 5: Full payload preservation
run_go_test "TestGenerateVarsFile_PreservesExplicitNetworkConfig" "Explicit network config preserved end-to-end"

# Test 6: Production payload validation
run_go_test "TestGenerateVarsFile_ProductionPayloadValidation" "Production payload validates correctly"

echo ""
echo "Step 5: Running edge case tests..."

# Test 7: Empty network array (uses framework default)
run_go_test "TestGenerateVarsFile_EmptyNetworkFieldDefaults" "Empty network array handled correctly"

# Test 8: Nil artifact config
run_go_test "TestGenerateNetworking_NilArtifact" "Nil artifact config handled correctly"

echo ""
echo "Step 6: Checking for regression..."

# Run all existing network tests to ensure no regressions
run_go_test "TestGenerateNetworking" "All existing network tests still pass"

echo ""
echo "Step 7: Validating no buggy patterns in code..."

# Check for the buggy "30s" path pattern (the old bug assigned interval to path)
# Exclude comments (// ...) and test files documenting the bug
BUGGY_PATH_MATCH=$(grep -r 'hcPath := "30s"' ./internal/hclgen/*.go 2>/dev/null | grep -v '//' | grep -v '_test.go' || true)
if [ -n "$BUGGY_PATH_MATCH" ]; then
    print_result "No buggy path initialization found" "FAIL"
    echo "  ERROR: Found 'hcPath := \"30s\"' pattern in code"
    echo "  $BUGGY_PATH_MATCH"
else
    print_result "No buggy path initialization found" "PASS"
fi

# Check for the buggy public override pattern (old code forced public=true for HTTP)
if grep -r 'network.Public || network.PortType == "http"' ./internal/hclgen/*.go > /dev/null 2>&1; then
    print_result "No public field override found" "FAIL"
    echo "  ERROR: Found public field override pattern in code"
else
    print_result "No public field override found" "PASS"
fi

# Check that correct helper functions exist
if grep -q 'func normalizeHealthCheckType' ./internal/hclgen/generator.go > /dev/null 2>&1; then
    print_result "Health check type normalizer exists" "PASS"
else
    print_result "Health check type normalizer exists" "FAIL"
fi

# Check that correct default path handling exists
if grep -q 'hcPath := "/"' ./internal/hclgen/generator.go > /dev/null 2>&1; then
    print_result "Correct default path '/' initialization" "PASS"
else
    print_result "Correct default path '/' initialization" "FAIL"
fi

echo ""
echo "========================================="
echo "Validation Summary"
echo "========================================="
echo -e "Tests Passed: ${GREEN}${PASSED}${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "Tests Failed: ${RED}${FAILED}${NC}"
else
    echo -e "Tests Failed: ${FAILED}"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} All validations passed!"
    echo ""
    echo "The network configuration bug fixes are working correctly."
    echo "Safe to deploy to production."
    exit 0
else
    echo -e "${RED}[FAILURE]${NC} Some validations failed!"
    echo ""
    echo "Please review the failed tests before deploying."
    echo "Run individual tests with: go test ./internal/hclgen/... -v -run <TestName>"
    exit 1
fi
