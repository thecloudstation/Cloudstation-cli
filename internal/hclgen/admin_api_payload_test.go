package hclgen

import (
	"strings"
	"testing"
)

// TestAdminAPIPayload_ProductionValidation tests with the actual admin-api production payload
// provided by the user to validate bug fixes work with real data.
// This test helps diagnose issues with HTTP port configuration, domain preservation,
// and health check settings.
func TestAdminAPIPayload_ProductionValidation(t *testing.T) {
	// User's actual admin-api production payload
	// Issue: User reported there's still an issue with this payload
	// This test will help diagnose what's wrong

	params := DeploymentParams{
		JobID:           "csi-adminapi-main-dmjmyddoamyo",
		ImageName:       "acrbc001.azurecr.io/csi-adminapi-main-dmjmyddoamyo",
		ImageTag:        "latest",
		BuilderType:     "docker",
		DeployType:      "nomad-pack",
		ClusterDomain:   "frparis.cloud-station.io",
		CPU:             100,
		RAM:             500,
		ReplicaCount:    1,
		RestartMode:     "fail",
		RestartAttempts: 0,
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http", // HTTP not TCP!
				Public:         true,
				Domain:         "admin-api", // User's domain
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/", // Explicitly set to "/"
					Port:     8080,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
	}

	// Generate HCL
	result := GenerateVarsFile(params, nil)

	// Log the full output for debugging
	t.Logf("Generated HCL for admin-api:\n%s", result)

	// === CRITICAL CHECKS ===

	// 1. Bug #1: Health check path must be "/"
	if !strings.Contains(result, `path="/"`) {
		t.Error("FAIL: Health check path not found or incorrect")
		t.Logf("Expected: path=\"/\"")
	} else {
		t.Log("PASS: Health check path is correct: path=\"/\"")
	}

	// Check for buggy "30s" path
	if strings.Contains(result, `path="30s"`) {
		t.Error("CRITICAL BUG: Found path=\"30s\" (should be \"/\")")
	} else {
		t.Log("PASS: No buggy path=\"30s\" found")
	}

	// 2. Bug #2: Public field preserved
	if !strings.Contains(result, "public=true") {
		t.Error("FAIL: public=true not preserved")
	} else {
		t.Log("PASS: Public field preserved: public=true")
	}

	// 3. Bug #3: Domain must be preserved
	if !strings.Contains(result, `domain="admin-api"`) {
		t.Error("CRITICAL: domain=\"admin-api\" NOT FOUND!")
		t.Error("User's domain was overwritten or lost!")

		// Check what domain we got instead
		if strings.Contains(result, "domain=") {
			// Extract domain value
			domainIdx := strings.Index(result, "domain=")
			if domainIdx != -1 {
				endIdx := domainIdx + 50
				if endIdx > len(result) {
					endIdx = len(result)
				}
				snippet := result[domainIdx:endIdx]
				t.Errorf("Found domain value: %s", snippet)
			}
		}
	} else {
		t.Log("PASS: Domain preserved: domain=\"admin-api\"")
	}

	// 4. Cluster domain
	if !strings.Contains(result, "frparis.cloud-station.io") {
		t.Error("FAIL: ClusterDomain not found")
	} else {
		t.Log("PASS: ClusterDomain preserved")
	}

	// 5. Port type should be HTTP
	httpTypeCount := strings.Count(result, `type="http"`)
	if httpTypeCount < 2 {
		t.Errorf("Expected at least 2 occurrences of type=\"http\" (port type + health check), found %d", httpTypeCount)
	} else {
		t.Logf("PASS: HTTP type found %d times (port + health check)", httpTypeCount)
	}

	// 6. Health check configuration
	if !strings.Contains(result, `interval="30s"`) {
		t.Error("FAIL: Health check interval missing")
	}
	if !strings.Contains(result, `timeout="30s"`) {
		t.Error("FAIL: Health check timeout missing")
	}

	// 7. Port number
	if !strings.Contains(result, "port=8080") {
		t.Error("FAIL: Port 8080 not found")
	}

	// === SUMMARY ===
	t.Log("\n=== ADMIN-API PAYLOAD TEST SUMMARY ===")

	// Check for any allocated/generic domains that would indicate override bug
	if strings.Contains(result, "subdomain-") || strings.Contains(result, "allocated-") {
		t.Error("CRITICAL: Found allocated domain pattern - domain was overwritten!")
	}

	// Final FQDN
	t.Log("Expected FQDN: admin-api.frparis.cloud-station.io")

	if strings.Contains(result, `domain="admin-api"`) && strings.Contains(result, "frparis.cloud-station.io") {
		t.Log("PASS: Domain components present for FQDN construction")
	} else {
		t.Error("FAIL: Cannot construct correct FQDN - domain or cluster domain missing")
	}
}

// TestAdminAPIPayload_HTTPHealthCheckType validates that HTTP health check type is preserved
// (not normalized to TCP) when explicitly set by the user
func TestAdminAPIPayload_HTTPHealthCheckType(t *testing.T) {
	params := DeploymentParams{
		JobID:         "admin-api-http-test",
		ImageName:     "admin-api-image",
		ImageTag:      "v1.0.0",
		BuilderType:   "docker",
		DeployType:    "nomad-pack",
		ClusterDomain: "frparis.cloud-station.io",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				Domain:         "admin-api",
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http", // Explicitly HTTP
					Path:     "/health",
					Port:     8080,
					Interval: "10s",
					Timeout:  "5s",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)
	t.Logf("Generated HCL:\n%s", result)

	// HTTP health check type should be preserved
	// The normalizeHealthCheckType function should NOT convert "http" to "tcp"
	if !strings.Contains(result, `health_check={type="http"`) {
		t.Error("FAIL: HTTP health check type not preserved in health_check block")
		if strings.Contains(result, `health_check={type="tcp"`) {
			t.Error("BUG: HTTP health check type was incorrectly normalized to TCP")
		}
	} else {
		t.Log("PASS: HTTP health check type preserved correctly")
	}

	// Custom path should be preserved
	if !strings.Contains(result, `path="/health"`) {
		t.Error("FAIL: Custom health check path /health not preserved")
	} else {
		t.Log("PASS: Custom health check path preserved")
	}

	// Custom intervals should be preserved
	if !strings.Contains(result, `interval="10s"`) {
		t.Error("FAIL: Custom interval 10s not preserved")
	}
	if !strings.Contains(result, `timeout="5s"`) {
		t.Error("FAIL: Custom timeout 5s not preserved")
	}
}

// TestAdminAPIPayload_FullNetworkBlock validates the complete network block format
// for the admin-api payload to ensure all fields are correctly positioned
func TestAdminAPIPayload_FullNetworkBlock(t *testing.T) {
	params := DeploymentParams{
		JobID:         "csi-adminapi-main-dmjmyddoamyo",
		ImageName:     "acrbc001.azurecr.io/csi-adminapi-main-dmjmyddoamyo",
		ImageTag:      "latest",
		BuilderType:   "docker",
		DeployType:    "nomad-pack",
		ClusterDomain: "frparis.cloud-station.io",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				Domain:         "admin-api",
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/",
					Port:     8080,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Extract the network block for detailed inspection
	networkIdx := strings.Index(result, "network = [")
	if networkIdx == -1 {
		t.Fatal("network block not found in output")
	}

	// Find the end of the network block
	endIdx := strings.Index(result[networkIdx:], "]\n")
	if endIdx == -1 {
		endIdx = len(result) - networkIdx
	}
	networkBlock := result[networkIdx : networkIdx+endIdx+1]
	t.Logf("Network block:\n%s", networkBlock)

	// Verify expected structure
	expectedParts := []string{
		`name="8080"`,
		`port=8080`,
		`type="http"`,
		`public=true`,
		`domain="admin-api"`,
		`has_health_check="http"`,
		`health_check={`,
		`type="http"`,
		`interval="30s"`,
		`path="/"`,
		`timeout="30s"`,
		`port=8080`,
	}

	for _, expected := range expectedParts {
		if !strings.Contains(networkBlock, expected) {
			t.Errorf("Missing expected part in network block: %s", expected)
		}
	}
}

// TestAdminAPIPayload_EmptyDomainBehavior validates that when domain is empty,
// the system still generates valid HCL (for zero-config deployments)
func TestAdminAPIPayload_EmptyDomainBehavior(t *testing.T) {
	params := DeploymentParams{
		JobID:         "admin-api-no-domain",
		ImageName:     "admin-api-image",
		ImageTag:      "latest",
		BuilderType:   "docker",
		DeployType:    "nomad-pack",
		ClusterDomain: "frparis.cloud-station.io",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				Domain:         "", // Empty domain
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/",
					Port:     8080,
					Interval: "30s",
					Timeout:  "30s",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)
	t.Logf("Generated HCL with empty domain:\n%s", result)

	// Should contain domain field (even if empty)
	if !strings.Contains(result, `domain=""`) {
		t.Error("Expected domain=\"\" for empty domain input")
	}

	// Should still have valid port configuration
	if !strings.Contains(result, "port=8080") {
		t.Error("Port 8080 missing from output")
	}

	// ClusterDomain should still be present
	if !strings.Contains(result, "frparis.cloud-station.io") {
		t.Error("ClusterDomain missing from output")
	}
}

// TestAdminAPIPayload_DomainNotOverwritten verifies that user-provided domains
// are never overwritten with generated values
func TestAdminAPIPayload_DomainNotOverwritten(t *testing.T) {
	testCases := []struct {
		name           string
		inputDomain    string
		expectedDomain string
	}{
		{"admin-api domain", "admin-api", "admin-api"},
		{"custom subdomain", "my-custom-api", "my-custom-api"},
		{"hyphenated domain", "admin-api-v2", "admin-api-v2"},
		{"short domain", "api", "api"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:         "test-job",
				ImageName:     "test-image",
				ImageTag:      "latest",
				BuilderType:   "docker",
				DeployType:    "nomad-pack",
				ClusterDomain: "frparis.cloud-station.io",
				Networks: []NetworkPort{
					{
						PortNumber:     8080,
						PortType:       "http",
						Public:         true,
						Domain:         tc.inputDomain,
						HasHealthCheck: "http",
						HealthCheck: HealthCheckConfig{
							Type:     "http",
							Path:     "/",
							Port:     8080,
							Interval: "30s",
							Timeout:  "30s",
						},
					},
				},
			}

			result := GenerateVarsFile(params, nil)
			expectedStr := `domain="` + tc.expectedDomain + `"`

			if !strings.Contains(result, expectedStr) {
				t.Errorf("Domain was not preserved. Expected %s, got:\n%s", expectedStr, result)
			}
		})
	}
}
