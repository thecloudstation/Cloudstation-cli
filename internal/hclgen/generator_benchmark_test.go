package hclgen

import (
	"testing"
)

// BenchmarkGenerateNetworking_SinglePort measures performance of network generation with one port
func BenchmarkGenerateNetworking_SinglePort(b *testing.B) {
	params := DeploymentParams{
		JobID:        "benchmark-app",
		ImageName:    "app",
		ImageTag:     "v1.0.0",
		BuilderType:  "csdocker",
		ReplicaCount: 1,
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         false,
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "30s",
					Timeout:  "10s",
					Port:     8080,
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateVarsFile(params, nil)
	}
}

// BenchmarkGenerateNetworking_MultiplePorts measures performance with multiple ports
func BenchmarkGenerateNetworking_MultiplePorts(b *testing.B) {
	params := DeploymentParams{
		JobID:        "benchmark-app-multi",
		ImageName:    "app",
		ImageTag:     "v1.0.0",
		BuilderType:  "csdocker",
		ReplicaCount: 3,
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         false,
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "30s",
					Timeout:  "10s",
				},
			},
			{
				PortNumber:     9090,
				PortType:       "grpc",
				Public:         false,
				HasHealthCheck: "grpc",
				HealthCheck: HealthCheckConfig{
					Type:     "grpc",
					Interval: "20s",
					Timeout:  "15s",
				},
			},
			{
				PortNumber:     3000,
				PortType:       "tcp",
				Public:         true,
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Interval: "15s",
					Timeout:  "5s",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateVarsFile(params, nil)
	}
}

// BenchmarkValidationHelpers measures overhead of validation helper functions
func BenchmarkValidationHelpers(b *testing.B) {
	// Test data
	testStrings := []string{"", " ", "  ", "value", "   value   "}
	testTypes := []string{"", "http", "tcp", "grpc", "script", "invalid", "no", "none"}

	b.Run("isEmptyString", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				_ = isEmptyString(s)
			}
		}
	})

	b.Run("isValidHealthCheckType", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, t := range testTypes {
				_ = isValidHealthCheckType(t)
			}
		}
	})

	b.Run("normalizeHealthCheckType", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, t := range testTypes {
				_ = normalizeHealthCheckType(t)
			}
		}
	})
}

// BenchmarkGenerateConfig_Complete measures full HCL generation pipeline
func BenchmarkGenerateConfig_Complete(b *testing.B) {
	params := DeploymentParams{
		JobID:        "benchmark-complete",
		ImageName:    "registry.example.com/app",
		ImageTag:     "v2.1.0",
		BuilderType:  "csdocker",
		DeployType:   "nomad-pack",
		ReplicaCount: 5,
		CPU:          1000,
		RAM:          2048,
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         false,
				CustomDomain:   "internal.example.com",
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/api/health",
					Interval: "30s",
					Timeout:  "10s",
					Port:     8080,
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateConfig(params)
	}
}
