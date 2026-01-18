package portdetector

import (
	"testing"
)

func TestExtractExposedPorts(t *testing.T) {
	tests := []struct {
		name     string
		input    *ImageInspection
		expected []int
	}{
		{
			name: "single exposed port",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					ExposedPorts: map[string]struct{}{
						"3000/tcp": {},
					},
				},
			},
			expected: []int{3000},
		},
		{
			name: "multiple exposed ports",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					ExposedPorts: map[string]struct{}{
						"3000/tcp": {},
						"8080/tcp": {},
					},
				},
			},
			expected: []int{3000, 8080}, // Order may vary, but should contain both
		},
		{
			name: "no exposed ports",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					ExposedPorts: nil,
				},
			},
			expected: nil,
		},
		{
			name: "invalid port format",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					ExposedPorts: map[string]struct{}{
						"invalid/tcp": {},
					},
				},
			},
			expected: nil,
		},
		{
			name: "port out of range",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					ExposedPorts: map[string]struct{}{
						"99999/tcp": {},
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExposedPorts(tt.input)

			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
				return
			}

			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d ports, got %d", len(tt.expected), len(result))
				return
			}

			// Check if all expected ports are in result
			for _, expectedPort := range tt.expected {
				found := false
				for _, port := range result {
					if port == expectedPort {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected port %d not found in result %v", expectedPort, result)
				}
			}
		})
	}
}

func TestExtractEnvPorts(t *testing.T) {
	tests := []struct {
		name     string
		input    *ImageInspection
		expected []int
	}{
		{
			name: "PORT environment variable",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					Env: []string{
						"PATH=/usr/bin",
						"PORT=3000",
						"NODE_ENV=production",
					},
				},
			},
			expected: []int{3000},
		},
		{
			name: "no PORT environment variable",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					Env: []string{
						"PATH=/usr/bin",
						"NODE_ENV=production",
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid PORT value",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					Env: []string{
						"PORT=invalid",
					},
				},
			},
			expected: nil,
		},
		{
			name: "PORT out of range",
			input: &ImageInspection{
				Config: struct {
					ExposedPorts map[string]struct{} `json:"ExposedPorts"`
					Env          []string            `json:"Env"`
				}{
					Env: []string{
						"PORT=99999",
					},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEnvPorts(tt.input)

			if tt.expected == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
				return
			}

			if tt.expected != nil && result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d ports, got %d", len(tt.expected), len(result))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected port %d, got %d", tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		port     int
		expected bool
	}{
		{3000, true},
		{8080, true},
		{1, true},
		{65535, true},
		{0, false},
		{-1, false},
		{65536, false},
		{99999, false},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.port)), func(t *testing.T) {
			result := isValidPort(tt.port)
			if result != tt.expected {
				t.Errorf("isValidPort(%d) = %v, expected %v", tt.port, result, tt.expected)
			}
		})
	}
}
