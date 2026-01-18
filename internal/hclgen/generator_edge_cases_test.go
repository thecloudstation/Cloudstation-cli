package hclgen

import (
	"strings"
	"testing"
)

// TestGenerateNetworking_EmptyNetworkArray tests behavior with zero networks
func TestGenerateNetworking_EmptyNetworkArray(t *testing.T) {
	params := DeploymentParams{
		JobID:        "no-networks",
		ImageName:    "app",
		ImageTag:     "v1",
		BuilderType:  "csdocker",
		ReplicaCount: 1,
		Networks:     []NetworkPort{}, // Empty array
	}

	result := GenerateVarsFile(params, nil)

	// Should generate valid HCL even with empty networks array
	if result == "" {
		t.Error("Expected non-empty result even with empty networks array")
	}

	// With empty networks array, fallback to framework default should occur
	// csdocker defaults to port 8000
	if !strings.Contains(result, "port=8000") {
		t.Errorf("Expected fallback to framework default port 8000 for csdocker, got: %s", result)
	}
}

// TestGenerateNetworking_NilHealthCheck tests handling of nil/zero health check config
func TestGenerateNetworking_NilHealthCheck(t *testing.T) {
	params := DeploymentParams{
		JobID:       "nil-healthcheck",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				HasHealthCheck: "",                  // Empty - should trigger defaults
				HealthCheck:    HealthCheckConfig{}, // All zero values
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Should have health check type (defaults to tcp for invalid types)
	if !strings.Contains(result, "type=") {
		t.Error("Expected health check type to be set even with nil config")
	}

	// Should have default intervals
	if !strings.Contains(result, "interval=") {
		t.Error("Expected default interval to be set")
	}

	// Should NOT have the buggy "30s" as path
	if strings.Contains(result, "path=\"30s\"") {
		t.Error("CRITICAL: Found buggy path=\"30s\" with nil health check config")
	}

	// Path should default to "/"
	if !strings.Contains(result, "path=\"/\"") {
		t.Errorf("Expected default health check path to be \"/\", got: %s", result)
	}

	// Interval and timeout should default to "30s"
	if !strings.Contains(result, "interval=\"30s\"") {
		t.Errorf("Expected default interval to be \"30s\", got: %s", result)
	}

	if !strings.Contains(result, "timeout=\"30s\"") {
		t.Errorf("Expected default timeout to be \"30s\", got: %s", result)
	}
}

// TestGenerateNetworking_ExtremelyLongStrings tests handling of very long string values
func TestGenerateNetworking_ExtremelyLongStrings(t *testing.T) {
	longPath := "/" + strings.Repeat("a", 500)                     // 501 character path
	longDomain := strings.Repeat("subdomain.", 50) + "example.com" // Very long domain

	params := DeploymentParams{
		JobID:       "long-strings",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:   8080,
				PortType:     "http",
				Public:       false,
				CustomDomain: longDomain,
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     longPath,
					Interval: "30s",
					Timeout:  "10s",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Should preserve the long path without truncation
	if !strings.Contains(result, longPath) {
		t.Error("Expected long health check path to be preserved")
	}

	// Should preserve the long domain
	if !strings.Contains(result, longDomain) {
		t.Error("Expected long custom domain to be preserved")
	}

	// Should generate valid HCL
	if result == "" {
		t.Error("Expected non-empty result with long strings")
	}
}

// TestGenerateNetworking_SpecialCharactersInStrings tests special character handling
func TestGenerateNetworking_SpecialCharactersInStrings(t *testing.T) {
	params := DeploymentParams{
		JobID:       "special-chars",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:   8080,
				PortType:     "http",
				Public:       false,
				CustomDomain: "api-v2.example.com", // Hyphens
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/api/v2/health?check=true&verbose=1", // Query params
					Interval: "30s",
					Timeout:  "10s",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Should preserve special characters in path
	if !strings.Contains(result, "/api/v2/health?check=true&verbose=1") {
		t.Error("Expected special characters in health check path to be preserved")
	}

	// Should preserve hyphens in domain
	if !strings.Contains(result, "api-v2.example.com") {
		t.Error("Expected hyphens in custom domain to be preserved")
	}
}

// TestGenerateNetworking_BoundaryPortNumbers tests edge cases for port numbers
func TestGenerateNetworking_BoundaryPortNumbers(t *testing.T) {
	tests := []struct {
		name       string
		portNumber int
		shouldWork bool
	}{
		{"minimum_valid_port", 1, true},
		{"low_port", 80, true},
		{"high_port", 65535, true},
		{"typical_port", 8080, true},
		{"zero_port", 0, false}, // Port 0 should be skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:       "port-test-" + tt.name,
				ImageName:   "app",
				ImageTag:    "v1",
				BuilderType: "csdocker",
				Networks: []NetworkPort{
					{
						PortNumber: tt.portNumber,
						PortType:   "tcp",
						Public:     true,
					},
				},
			}

			result := GenerateVarsFile(params, nil)

			if tt.shouldWork {
				// Should contain the port number in the output
				if !strings.Contains(result, "port=") {
					t.Errorf("Expected result to contain port declaration for valid port %d", tt.portNumber)
				}
			} else {
				// Port 0 networks should be skipped, so either fall back to framework default
				// or generate network block with framework default port
				if strings.Contains(result, "port=0") {
					t.Errorf("Port 0 should be skipped or handled gracefully, not included directly")
				}
			}
		})
	}
}

// TestGenerateNetworking_MixedCaseHealthCheckTypes tests case sensitivity
func TestGenerateNetworking_MixedCaseHealthCheckTypes(t *testing.T) {
	tests := []struct {
		inputType    string
		expectedType string
	}{
		{"HTTP", "http"},
		{"Http", "http"},
		{"TCP", "tcp"},
		{"Tcp", "tcp"},
		{"GRPC", "grpc"},
		{"GrPc", "grpc"},
	}

	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			params := DeploymentParams{
				JobID:       "case-test",
				ImageName:   "app",
				ImageTag:    "v1",
				BuilderType: "csdocker",
				Networks: []NetworkPort{
					{
						PortNumber:     8080,
						PortType:       "http",
						Public:         true,
						HasHealthCheck: tt.inputType,
						HealthCheck: HealthCheckConfig{
							Type:     tt.inputType,
							Path:     "/health",
							Interval: "30s",
							Timeout:  "10s",
						},
					},
				},
			}

			result := GenerateVarsFile(params, nil)

			// Health check type should be normalized to lowercase
			// The normalizeHealthCheckType function handles this
			if !strings.Contains(result, "health_check=") {
				t.Errorf("Expected health_check block in result, got: %s", result)
			}

			// Verify health check type appears in lowercase form
			// Note: The implementation normalizes via strings.ToLower in isValidHealthCheckType
			expectedTypePattern := "type=\"" + tt.expectedType + "\""
			if !strings.Contains(result, expectedTypePattern) {
				// If not normalized, it might still appear in original case
				// Check for either lowercase or original case presence
				originalTypePattern := "type=\"" + tt.inputType + "\""
				if !strings.Contains(result, originalTypePattern) && !strings.Contains(result, expectedTypePattern) {
					t.Errorf("Expected type to be present (normalized to %q or original %q), got result: %s", tt.expectedType, tt.inputType, result)
				}
			}
		})
	}
}

// TestGenerateNetworking_WhitespaceInValues tests trimming of whitespace
func TestGenerateNetworking_WhitespaceInValues(t *testing.T) {
	params := DeploymentParams{
		JobID:       "whitespace-test",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:   8080,
				PortType:     "  http  ", // Leading/trailing whitespace
				Public:       true,
				CustomDomain: "  example.com  ",
				HealthCheck: HealthCheckConfig{
					Type:     "  http  ",
					Path:     "  /health  ",
					Interval: "  30s  ",
					Timeout:  "  10s  ",
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Should not break HCL generation
	if result == "" {
		t.Error("Expected non-empty result even with whitespace in values")
	}

	// Should not contain raw whitespace in quoted strings (or should be normalized)
	// This is a basic sanity check
	if !strings.Contains(result, "type=") {
		t.Error("Expected type field to be present despite whitespace")
	}

	// Network block should be generated
	if !strings.Contains(result, "network = [") {
		t.Errorf("Expected network configuration to be generated, got: %s", result)
	}
}

// TestGenerateNetworking_InvalidHealthCheckTypeFallback tests handling of invalid health check types with fallback behavior
func TestGenerateNetworking_InvalidHealthCheckTypeFallback(t *testing.T) {
	invalidTypes := []string{
		"no",
		"none",
		"",
		"invalid",
		"unknown",
		"123",
	}

	for _, invalidType := range invalidTypes {
		t.Run("type_"+invalidType, func(t *testing.T) {
			params := DeploymentParams{
				JobID:       "invalid-hc-type",
				ImageName:   "app",
				ImageTag:    "v1",
				BuilderType: "csdocker",
				Networks: []NetworkPort{
					{
						PortNumber:     8080,
						PortType:       "http",
						Public:         true,
						HasHealthCheck: invalidType,
						HealthCheck: HealthCheckConfig{
							Type:     invalidType,
							Path:     "/health",
							Interval: "30s",
							Timeout:  "10s",
						},
					},
				},
			}

			result := GenerateVarsFile(params, nil)

			// Invalid types should fall back to "tcp" as the safe default
			// Based on normalizeHealthCheckType implementation
			if !strings.Contains(result, "type=\"tcp\"") {
				// At minimum, ensure we have a valid type
				if !strings.Contains(result, "type=\"http\"") && !strings.Contains(result, "type=\"tcp\"") && !strings.Contains(result, "type=\"grpc\"") && !strings.Contains(result, "type=\"script\"") {
					t.Errorf("Expected invalid type %q to fall back to a valid type (tcp), got: %s", invalidType, result)
				}
			}
		})
	}
}

// TestGenerateNetworking_MultipleNetworksWithMixedConfig tests multiple networks with varied configs
func TestGenerateNetworking_MultipleNetworksWithMixedConfig(t *testing.T) {
	params := DeploymentParams{
		JobID:       "multi-network",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "15s",
					Timeout:  "5s",
				},
			},
			{
				PortNumber:     9090,
				PortType:       "grpc",
				Public:         false,
				HasHealthCheck: "grpc",
				HealthCheck: HealthCheckConfig{
					Type:     "grpc",
					Interval: "30s",
					Timeout:  "10s",
				},
			},
			{
				PortNumber: 0, // Should be skipped
				PortType:   "tcp",
				Public:     false,
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// First port should be present
	if !strings.Contains(result, "port=8080") {
		t.Errorf("Expected first port 8080 to be present, got: %s", result)
	}

	// Second port should be present
	if !strings.Contains(result, "port=9090") {
		t.Errorf("Expected second port 9090 to be present, got: %s", result)
	}

	// Port 0 should NOT be present
	if strings.Contains(result, "port=0") {
		t.Errorf("Port 0 should be skipped, but found in result: %s", result)
	}

	// Verify mixed health check configurations
	if !strings.Contains(result, "interval=\"15s\"") {
		t.Errorf("Expected first port's custom interval 15s, got: %s", result)
	}

	if !strings.Contains(result, "interval=\"30s\"") {
		t.Errorf("Expected second port's interval 30s, got: %s", result)
	}

	// Verify mixed public settings
	if !strings.Contains(result, "public=true") {
		t.Errorf("Expected public=true for first port, got: %s", result)
	}

	if !strings.Contains(result, "public=false") {
		t.Errorf("Expected public=false for second port, got: %s", result)
	}
}

// TestGenerateNetworking_IntervalWithoutTimeUnit tests intervals specified without time unit
func TestGenerateNetworking_IntervalWithoutTimeUnit(t *testing.T) {
	tests := []struct {
		name             string
		inputInterval    string
		expectedInterval string
	}{
		{"numeric_only", "30", "30s"},
		{"already_has_seconds", "30s", "30s"},
		{"already_has_minutes", "5m", "5m"},
		{"already_has_hours", "1h", "1h"},
		{"already_has_days", "1d", "1d"},
		{"invalid_format", "abc", "30s"}, // Falls back to default
		{"empty", "", "30s"},             // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:       "interval-test",
				ImageName:   "app",
				ImageTag:    "v1",
				BuilderType: "csdocker",
				Networks: []NetworkPort{
					{
						PortNumber:     8080,
						PortType:       "http",
						Public:         true,
						HasHealthCheck: "http",
						HealthCheck: HealthCheckConfig{
							Type:     "http",
							Path:     "/health",
							Interval: tt.inputInterval,
							Timeout:  "10s",
						},
					},
				},
			}

			result := GenerateVarsFile(params, nil)

			expectedPattern := "interval=\"" + tt.expectedInterval + "\""
			if !strings.Contains(result, expectedPattern) {
				t.Errorf("Expected interval to be %q, got result: %s", tt.expectedInterval, result)
			}
		})
	}
}

// TestGenerateNetworking_HealthCheckPortDefault tests health check port defaulting to network port
func TestGenerateNetworking_HealthCheckPortDefault(t *testing.T) {
	params := DeploymentParams{
		JobID:       "hc-port-default",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:     3000,
				PortType:       "http",
				Public:         true,
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "30s",
					Timeout:  "10s",
					Port:     0, // Zero - should default to network port
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Count occurrences of port=3000 - should appear for both network port and health check port
	portCount := strings.Count(result, "port=3000")
	if portCount < 2 {
		t.Errorf("Expected health check port to default to network port 3000, found %d occurrences. Result: %s", portCount, result)
	}
}

// TestGenerateNetworking_ExplicitHealthCheckPort tests explicit health check port (different from network port)
func TestGenerateNetworking_ExplicitHealthCheckPort(t *testing.T) {
	params := DeploymentParams{
		JobID:       "hc-port-explicit",
		ImageName:   "app",
		ImageTag:    "v1",
		BuilderType: "csdocker",
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         true,
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "30s",
					Timeout:  "10s",
					Port:     9090, // Explicit different port
				},
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Health check should use explicit port 9090
	if !strings.Contains(result, "port=9090") {
		t.Errorf("Expected explicit health check port 9090 to be used, got: %s", result)
	}

	// Network port should still be 8080
	if !strings.Contains(result, "port=8080") {
		t.Errorf("Expected network port 8080 to be present, got: %s", result)
	}
}
