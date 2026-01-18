package hclgen

import "testing"

func TestGetFrameworkDefault(t *testing.T) {
	tests := []struct {
		name        string
		builderType string
		expected    int
	}{
		{
			name:        "nixpacks builder",
			builderType: "nixpacks",
			expected:    DefaultWebPort,
		},
		{
			name:        "railpack builder",
			builderType: "railpack",
			expected:    DefaultWebPort,
		},
		{
			name:        "csdocker builder",
			builderType: "csdocker",
			expected:    PythonDefaultPort,
		},
		{
			name:        "unknown builder",
			builderType: "unknown",
			expected:    DefaultWebPort,
		},
		{
			name:        "empty builder",
			builderType: "",
			expected:    DefaultWebPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFrameworkDefault(tt.builderType)
			if result != tt.expected {
				t.Errorf("GetFrameworkDefault(%q) = %d, expected %d", tt.builderType, result, tt.expected)
			}
		})
	}
}

func TestDetectFrameworkFromMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: "",
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFrameworkFromMetadata(tt.metadata)
			if result != tt.expected {
				t.Errorf("DetectFrameworkFromMetadata() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
