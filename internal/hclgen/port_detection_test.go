package hclgen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

func TestGenerateNetworking_UserSpecifiedPorts(t *testing.T) {
	// Tier 1: User-specified networks should take precedence
	networks := []NetworkPort{
		{
			PortNumber: 8080,
			PortType:   "http",
			Public:     true,
		},
	}

	art := &artifact.Artifact{
		ExposedPorts: []int{3000}, // Should be ignored
	}

	result := generateNetworking(networks, art, "nixpacks")

	if !strings.Contains(result, "port=8080") {
		t.Errorf("Expected user-specified port 8080, got: %s", result)
	}

	if strings.Contains(result, "port=3000") {
		t.Errorf("Should not contain detected port 3000 when user specified port 8080")
	}
}

func TestGenerateNetworking_ArtifactDetectedPorts(t *testing.T) {
	// Tier 2: Use detected ports from artifact when networks array is empty
	networks := []NetworkPort{}

	art := &artifact.Artifact{
		ExposedPorts: []int{4000},
	}

	result := generateNetworking(networks, art, "nixpacks")

	if !strings.Contains(result, "port=4000") {
		t.Errorf("Expected detected port 4000, got: %s", result)
	}

	if !strings.Contains(result, "network = [") {
		t.Errorf("Expected network configuration to be generated")
	}
}

func TestGenerateNetworking_FrameworkDefault(t *testing.T) {
	// Tier 3: Use framework default when no networks and no detected ports
	networks := []NetworkPort{}
	art := &artifact.Artifact{
		ExposedPorts: []int{}, // Empty
	}

	result := generateNetworking(networks, art, "nixpacks")

	if !strings.Contains(result, "port=3000") {
		t.Errorf("Expected default port 3000 for nixpacks, got: %s", result)
	}
}

func TestGenerateNetworking_NilArtifact(t *testing.T) {
	// Tier 3: Use framework default when artifact is nil
	networks := []NetworkPort{}

	result := generateNetworking(networks, nil, "nixpacks")

	if !strings.Contains(result, "port=3000") {
		t.Errorf("Expected default port 3000 when artifact is nil, got: %s", result)
	}
}

func TestGenerateNetworking_CsDockerDefault(t *testing.T) {
	// Tier 3: CsDocker should default to port 8000
	networks := []NetworkPort{}

	result := generateNetworking(networks, nil, "csdocker")

	if !strings.Contains(result, "port=8000") {
		t.Errorf("Expected default port 8000 for csdocker, got: %s", result)
	}
}

func TestGenerateNetworking_MultipleExposedPorts(t *testing.T) {
	// Should use the first exposed port
	networks := []NetworkPort{}

	art := &artifact.Artifact{
		ExposedPorts: []int{5000, 6000, 7000},
	}

	result := generateNetworking(networks, art, "nixpacks")

	if !strings.Contains(result, "port=5000") {
		t.Errorf("Expected first exposed port 5000, got: %s", result)
	}

	if strings.Contains(result, "port=6000") {
		t.Errorf("Should only use first exposed port, not 6000")
	}
}

func TestGenerateVarsFile_WithDetectedPorts(t *testing.T) {
	// Test end-to-end: GenerateVarsFile with artifact containing detected ports
	params := DeploymentParams{
		JobID:        "test-app",
		BuilderType:  "nixpacks",
		ReplicaCount: 1,
		Networks:     []NetworkPort{}, // Empty - should use detected ports
	}

	art := &artifact.Artifact{
		ExposedPorts: []int{3001},
	}

	varsContent := GenerateVarsFile(params, art)

	if !strings.Contains(varsContent, "network = [") {
		t.Errorf("Expected network configuration to be generated")
	}

	if !strings.Contains(varsContent, "port=3001") {
		t.Errorf("Expected detected port 3001 in vars file, got: %s", varsContent)
	}
}

func TestGenerateVarsFile_WithUserNetworks(t *testing.T) {
	// Test that user-specified networks are preserved
	params := DeploymentParams{
		JobID:        "test-app",
		BuilderType:  "nixpacks",
		ReplicaCount: 1,
		Networks: []NetworkPort{
			{
				PortNumber: 9000,
				PortType:   "tcp",
			},
		},
	}

	art := &artifact.Artifact{
		ExposedPorts: []int{3000}, // Should be ignored
	}

	varsContent := GenerateVarsFile(params, art)

	if !strings.Contains(varsContent, "port=9000") {
		t.Errorf("Expected user-specified port 9000, got: %s", varsContent)
	}

	if strings.Contains(varsContent, "port=3000") {
		t.Errorf("Should not contain detected port when user specified networks")
	}
}

func TestGenerateVarsFile_WithNilArtifact(t *testing.T) {
	// Test backward compatibility with nil artifact
	params := DeploymentParams{
		JobID:        "test-app",
		BuilderType:  "nixpacks",
		ReplicaCount: 1,
		Networks:     []NetworkPort{}, // Empty
	}

	varsContent := GenerateVarsFile(params, nil)

	if !strings.Contains(varsContent, "network = [") {
		t.Errorf("Expected network configuration with default port")
	}

	if !strings.Contains(varsContent, "port=3000") {
		t.Errorf("Expected default port 3000 when artifact is nil, got: %s", varsContent)
	}
}

func TestGenerateNetworking_PublicFalseWithHTTP(t *testing.T) {
	// Test that Public=false is preserved even for HTTP port types
	// This verifies the fix for the bug where HTTP ports were forced to public=true
	networks := []NetworkPort{
		{
			PortNumber: 8080,
			PortType:   "http",
			Public:     false, // Explicitly set to false
			HealthCheck: HealthCheckConfig{
				Type:     "http",
				Path:     "/health",
				Interval: "10s",
				Timeout:  "5s",
			},
		},
	}

	result := generateNetworking(networks, nil, "nixpacks")

	// Verify Public=false is preserved (not overridden to true)
	if !strings.Contains(result, "public=false") {
		t.Errorf("Expected public=false to be preserved for HTTP port, got: %s", result)
	}

	// Should NOT contain public=true
	if strings.Contains(result, "public=true") {
		t.Errorf("Public field should be false, not true. Result: %s", result)
	}

	// Verify port configuration is otherwise correct
	if !strings.Contains(result, "port=8080") {
		t.Errorf("Expected port=8080, got: %s", result)
	}

	if !strings.Contains(result, "type=\"http\"") {
		t.Errorf("Expected type=\"http\", got: %s", result)
	}
}

func TestGenerateNetworking_CustomHealthCheckPath(t *testing.T) {
	// Test that custom health check paths are preserved
	networks := []NetworkPort{
		{
			PortNumber: 3000,
			PortType:   "http",
			Public:     true,
			HealthCheck: HealthCheckConfig{
				Type:     "http",
				Path:     "/api/health/ready", // Custom path
				Interval: "15s",
				Timeout:  "10s",
			},
		},
	}

	result := generateNetworking(networks, nil, "nixpacks")

	// Verify custom path is preserved
	if !strings.Contains(result, "path=\"/api/health/ready\"") {
		t.Errorf("Expected custom health check path to be preserved, got: %s", result)
	}

	// Should NOT contain default path
	if strings.Contains(result, "path=\"/\"") {
		t.Errorf("Should not use default path when custom path is provided. Result: %s", result)
	}

	// Should NOT contain the buggy "30s" path value
	if strings.Contains(result, "path=\"30s\"") {
		t.Errorf("Health check path should not be '30s' (time interval), got: %s", result)
	}
}

func TestGenerateNetworking_DefaultHealthCheckPath(t *testing.T) {
	// Test that the default health check path is "/" and not the buggy "30s"
	// This verifies the fix for the critical health check path initialization bug
	networks := []NetworkPort{
		{
			PortNumber: 8000,
			PortType:   "http",
			Public:     true,
			HealthCheck: HealthCheckConfig{
				Type:     "http",
				Path:     "", // Empty - should default to "/"
				Interval: "30s",
				Timeout:  "30s",
			},
		},
	}

	result := generateNetworking(networks, nil, "csdocker")

	// Verify default path is "/"
	if !strings.Contains(result, "path=\"/\"") {
		t.Errorf("Expected default health check path to be \"/\", got: %s", result)
	}

	// CRITICAL: Should NOT contain the buggy "30s" value for path
	if strings.Contains(result, "path=\"30s\"") {
		t.Errorf("CRITICAL BUG: Health check path should not be '30s', this is a time interval not a path. Result: %s", result)
	}

	// Verify interval is correctly "30s" (not confused with path)
	if !strings.Contains(result, "interval=\"30s\"") {
		t.Errorf("Expected interval=\"30s\", got: %s", result)
	}
}

func TestGenerateNetworking_InvalidHealthCheckType(t *testing.T) {
	// Test that invalid health check types are normalized to "tcp"
	testCases := []struct {
		name         string
		inputType    string
		expectedType string
	}{
		{
			name:         "empty type defaults to tcp",
			inputType:    "",
			expectedType: "tcp",
		},
		{
			name:         "no type converts to tcp",
			inputType:    "no",
			expectedType: "tcp",
		},
		{
			name:         "none type converts to tcp",
			inputType:    "none",
			expectedType: "tcp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			networks := []NetworkPort{
				{
					PortNumber: 5000,
					PortType:   "tcp",
					Public:     false,
					HealthCheck: HealthCheckConfig{
						Type:     tc.inputType, // Invalid type
						Interval: "20s",
						Timeout:  "15s",
					},
				},
			}

			result := generateNetworking(networks, nil, "nixpacks")

			// Verify type is converted to tcp
			expectedStr := fmt.Sprintf("type=\"%s\"", tc.expectedType)
			if !strings.Contains(result, expectedStr) {
				t.Errorf("Expected health check %s, got: %s", expectedStr, result)
			}

			// Should NOT contain the invalid type in output
			if tc.inputType != "" && strings.Contains(result, fmt.Sprintf("type=\"%s\"", tc.inputType)) {
				t.Errorf("Invalid type '%s' should have been converted to tcp. Result: %s", tc.inputType, result)
			}
		})
	}
}
