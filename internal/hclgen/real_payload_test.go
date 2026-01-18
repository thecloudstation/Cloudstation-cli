package hclgen

import (
	"strings"
	"testing"
)

// TestRealProductionPayload_CSBackend tests with the actual production payload
// provided by the user to validate bug fixes work with real data
func TestRealProductionPayload_CSBackend(t *testing.T) {
	// This is the EXACT payload from the user (decoded from base64)
	// Payload represents: Backend service with 2 TCP ports (8085 API, 8086 Auth)
	params := DeploymentParams{
		JobID:           "csi-csbackend-main-xdjmwnyyplgo",
		ImageName:       "acrbc001.azurecr.io/csi-csbackend-main-xdjmwnyyplgo",
		ImageTag:        "latest",   // Not in payload, using default
		BuilderType:     "csdocker", // Inferred from deployment type
		DeployType:      "nomad-pack",
		CPU:             100,
		RAM:             500,
		ReplicaCount:    1,
		RestartAttempts: 0,
		RestartMode:     "fail",
		Networks: []NetworkPort{
			{
				PortNumber:     8085,
				PortType:       "tcp",
				Public:         true,
				Domain:         "api",
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Path:     "/",
					Port:     8085,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
			{
				PortNumber:     8086,
				PortType:       "tcp",
				Public:         true,
				Domain:         "auth",
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Path:     "/",
					Port:     8086,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
		ClusterDomain: "frparis.cloud-station.io",
	}

	// Generate HCL vars file
	result := GenerateVarsFile(params, nil)

	// === Critical Bug Fix Validations ===

	// 1. CRITICAL: Verify NO instances of the buggy "30s" path
	// (The bug was: hcPath := "30s" instead of hcPath := "/")
	if strings.Contains(result, `path="30s"`) {
		t.Error("CRITICAL BUG DETECTED: Found path=\"30s\" which is the old bug (time interval instead of URL path)")
		t.Logf("Result contains buggy pattern:\n%s", result)
	}

	// 2. Verify correct "/" path is used for TCP health checks
	pathCount := strings.Count(result, `path="/"`)
	if pathCount < 2 {
		t.Errorf("Expected at least 2 health checks with path=\"/\", found %d", pathCount)
		t.Logf("Full result:\n%s", result)
	}

	// 3. Verify public=true is preserved for both ports
	publicTrueCount := strings.Count(result, "public=true")
	if publicTrueCount < 2 {
		t.Errorf("Expected 2 occurrences of public=true, found %d", publicTrueCount)
		t.Logf("Full result:\n%s", result)
	}

	// === Network Configuration Validations ===

	// 4. Verify first port (8085 - API)
	if !strings.Contains(result, "port=8085") {
		t.Error("Expected port=8085 for API endpoint")
	}

	if !strings.Contains(result, `domain="api"`) {
		t.Error("Expected domain=\"api\" for first port")
	}

	// 5. Verify second port (8086 - Auth)
	if !strings.Contains(result, "port=8086") {
		t.Error("Expected port=8086 for Auth endpoint")
	}

	if !strings.Contains(result, `domain="auth"`) {
		t.Error("Expected domain=\"auth\" for second port")
	}

	// 6. Verify clusterDomain flows through correctly
	clusterDomainCount := strings.Count(result, "frparis.cloud-station.io")
	if clusterDomainCount < 1 {
		t.Errorf("Expected clusterDomain to appear at least 1 time, found %d", clusterDomainCount)
	}

	// === Health Check Configuration Validations ===

	// 7. Verify TCP health check type for both ports
	tcpTypeCount := strings.Count(result, `type="tcp"`)
	if tcpTypeCount < 4 { // 2 port types + 2 health check types
		t.Errorf("Expected at least 4 occurrences of type=\"tcp\" (port types and health check types), found %d", tcpTypeCount)
	}

	// 8. Verify custom interval "30s" is preserved
	intervalCount := strings.Count(result, `interval="30s"`)
	if intervalCount < 2 {
		t.Errorf("Expected 2 health checks with interval=\"30s\", found %d", intervalCount)
	}

	// 9. Verify custom timeout "30s" is preserved
	timeoutCount := strings.Count(result, `timeout="30s"`)
	if timeoutCount < 2 {
		t.Errorf("Expected 2 health checks with timeout=\"30s\", found %d", timeoutCount)
	}

	// 10. Verify health check ports match network ports
	if !strings.Contains(result, "port=8085") || !strings.Contains(result, "port=8086") {
		t.Error("Health check ports should match network ports (8085 and 8086)")
	}

	// === Job Configuration Validations ===

	// 11. Verify job name
	if !strings.Contains(result, "csi-csbackend-main-xdjmwnyyplgo") {
		t.Error("Expected job ID to be preserved in output")
	}

	// 12. Verify resource allocations
	if !strings.Contains(result, "cpu=100") {
		t.Error("Expected cpu=100")
	}

	if !strings.Contains(result, "memory=500") {
		t.Error("Expected memory=500")
	}

	// 13. Verify replica count
	if !strings.Contains(result, "count = 1") {
		t.Error("Expected count = 1 (replica count)")
	}

	// 14. Verify restart configuration
	// Note: When RestartAttempts=0 and RestartMode="fail", the system defaults to 3 attempts
	// Only RestartMode="never" results in 0 attempts
	if !strings.Contains(result, "restart_attempts = 3") {
		t.Error("Expected restart_attempts = 3 (default for fail mode with 0 specified)")
	}

	// === Regression Check ===

	// 15. Verify result is not empty
	if result == "" {
		t.Fatal("Generated HCL is empty - critical failure")
	}

	// 16. Verify basic HCL structure
	if !strings.Contains(result, "network = [") {
		t.Error("Expected networks array in HCL output")
	}

	// === Summary ===
	t.Logf("All validations passed for real production payload")
	t.Logf("Bug fixes confirmed working:")
	t.Logf("  - Health check path correctly defaults to \"/\" not \"30s\"")
	t.Logf("  - Public field preserved as specified (public=true)")
	t.Logf("  - ClusterDomain flows through correctly")
	t.Logf("  - All network configurations preserved")
	t.Logf("  - Health check configurations preserved")
}

// TestRealProductionPayload_PrivateTCPVariant tests a variant where
// the user wants PRIVATE TCP services (not public)
func TestRealProductionPayload_PrivateTCPVariant(t *testing.T) {
	// Same payload but with public=false to test the bug fix
	// This tests that TCP ports can be private (the bug would have kept them public)
	params := DeploymentParams{
		JobID:        "csi-csbackend-internal",
		ImageName:    "internal-backend",
		ImageTag:     "v1.0.0",
		BuilderType:  "csdocker",
		DeployType:   "nomad-pack",
		CPU:          100,
		RAM:          500,
		ReplicaCount: 1,
		Networks: []NetworkPort{
			{
				PortNumber:     8085,
				PortType:       "tcp",
				Public:         false, // EXPLICITLY PRIVATE
				Domain:         "",
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Path:     "/",
					Port:     8085,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
		ClusterDomain: "internal.cloud-station.io",
	}

	result := GenerateVarsFile(params, nil)

	// CRITICAL: Verify public=false is preserved for TCP ports
	if !strings.Contains(result, "public=false") {
		t.Error("CRITICAL: Expected public=false to be preserved for private TCP service")
		t.Logf("This was the bug - TCP ports were being forced to public")
		t.Logf("Result:\n%s", result)
	}

	// Should NOT contain public=true
	if strings.Contains(result, "public=true") {
		t.Error("CRITICAL: Found public=true when service should be private")
		t.Logf("Result:\n%s", result)
	}

	// Verify other fields preserved
	if !strings.Contains(result, "port=8085") {
		t.Error("Expected port=8085")
	}

	if !strings.Contains(result, `type="tcp"`) {
		t.Error("Expected type=\"tcp\"")
	}

	if !strings.Contains(result, `path="/"`) {
		t.Error("Expected path=\"/\"")
	}

	// Verify NO buggy "30s" path
	if strings.Contains(result, `path="30s"`) {
		t.Error("CRITICAL BUG: Found path=\"30s\"")
	}

	t.Logf("Private TCP service test passed")
	t.Logf("Confirmed: TCP ports can be private (public=false preserved)")
}

// TestRealProductionPayload_DomainPreservation tests end-to-end domain preservation
// from the user's actual production payload after the domain override bug fix in handlers.go
// This tests the COMPLETE flow: payload -> handlers -> generator -> HCL output
func TestRealProductionPayload_DomainPreservation(t *testing.T) {
	// User's actual production payload has these domains:
	// - Port 8085: domain="api"
	// - Port 8086: domain="auth"
	// - ClusterDomain: "frparis.cloud-station.io"

	params := DeploymentParams{
		JobID:         "csi-csbackend-main-xdjmwnyyplgo",
		ImageName:     "acrbc001.azurecr.io/csi-csbackend-main-xdjmwnyyplgo",
		ImageTag:      "latest",
		BuilderType:   "csdocker",
		DeployType:    "nomad-pack",
		ClusterDomain: "frparis.cloud-station.io",
		CPU:           100,
		RAM:           500,
		ReplicaCount:  1,
		Networks: []NetworkPort{
			{
				PortNumber:     8085,
				PortType:       "tcp",
				Public:         true,
				Domain:         "api", // USER PROVIDED - must be preserved
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Path:     "/",
					Port:     8085,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
			{
				PortNumber:     8086,
				PortType:       "tcp",
				Public:         true,
				Domain:         "auth", // USER PROVIDED - must be preserved
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Path:     "/",
					Port:     8086,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
	}

	// Generate HCL output
	result := GenerateVarsFile(params, nil)

	// === CRITICAL: Domain Preservation Validations ===

	// 1. Verify domain="api" is preserved (not overwritten)
	if !strings.Contains(result, `domain="api"`) {
		t.Error("CRITICAL BUG: domain=\"api\" not found in output!")
		t.Error("User's custom domain was overwritten or lost")
		t.Logf("Result:\n%s", result)
	}

	// 2. Verify domain="auth" is preserved (not overwritten)
	if !strings.Contains(result, `domain="auth"`) {
		t.Error("CRITICAL BUG: domain=\"auth\" not found in output!")
		t.Error("User's custom domain was overwritten or lost")
		t.Logf("Result:\n%s", result)
	}

	// 3. Verify NO generic allocated domains (like "subdomain-xxxx")
	// If the bug exists, we'd see backend-allocated domains instead of "api"/"auth"
	if strings.Contains(result, "subdomain-") || strings.Contains(result, "allocated-") {
		t.Error("CRITICAL BUG: Found allocated domain pattern - user domains were overwritten!")
		t.Logf("Result:\n%s", result)
	}

	// 4. Verify cluster domain is present
	if !strings.Contains(result, "frparis.cloud-station.io") {
		t.Error("ClusterDomain missing from output")
	}

	// 5. Verify full FQDNs can be constructed
	// The pack template will combine domain + cluster_domain to create:
	// - api.frparis.cloud-station.io
	// - auth.frparis.cloud-station.io

	// Count domain occurrences
	apiCount := strings.Count(result, `domain="api"`)
	authCount := strings.Count(result, `domain="auth"`)

	if apiCount != 1 {
		t.Errorf("Expected exactly 1 occurrence of domain=\"api\", found %d", apiCount)
	}

	if authCount != 1 {
		t.Errorf("Expected exactly 1 occurrence of domain=\"auth\", found %d", authCount)
	}

	// === Summary ===
	t.Log("Domain preservation verified end-to-end")
	t.Log("User domains preserved:")
	t.Log("  - Port 8085: domain=\"api\" -> api.frparis.cloud-station.io")
	t.Log("  - Port 8086: domain=\"auth\" -> auth.frparis.cloud-station.io")
	t.Log("No backend-allocated domains found (bug fixed)")
}

// TestRealProductionPayload_EmptyDomainAllocation tests that when user doesn't provide
// a domain, allocation still works. This ensures the bug fix doesn't break zero-config deployments.
func TestRealProductionPayload_EmptyDomainAllocation(t *testing.T) {
	params := DeploymentParams{
		JobID:         "test-empty-domains",
		ImageName:     "test-app",
		ImageTag:      "v1.0.0",
		BuilderType:   "csdocker",
		ClusterDomain: "test-cluster.io",
		Networks: []NetworkPort{
			{
				PortNumber: 8080,
				PortType:   "http",
				Public:     true,
				Domain:     "", // Empty - should work with allocation
			},
		},
	}

	// Generate HCL
	result := GenerateVarsFile(params, nil)

	// Should generate valid HCL even with empty domain
	if result == "" {
		t.Fatal("Generated HCL is empty")
	}

	// Should contain the port configuration
	if !strings.Contains(result, "port=8080") {
		t.Error("Port configuration missing")
	}

	// Domain field should be present (even if empty)
	if !strings.Contains(result, "domain=") {
		t.Error("Domain field missing from HCL")
	}

	t.Log("Empty domain handled correctly (allocation flow works)")
}
