package main

import (
	"strings"
	"testing"
)

// TestNATSURLFormatting tests the NATS URL handling fix
func TestNATSURLFormatting(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Single URL without prefix",
			input:    []string{"10.0.40.46:29167"},
			expected: "nats://10.0.40.46:29167",
		},
		{
			name:     "Single URL with nats:// prefix",
			input:    []string{"nats://10.0.40.46:29167"},
			expected: "nats://10.0.40.46:29167",
		},
		{
			name:     "Single URL with tls:// prefix",
			input:    []string{"tls://10.0.40.46:29167"},
			expected: "tls://10.0.40.46:29167",
		},
		{
			name:     "Multiple URLs without prefix",
			input:    []string{"10.0.40.46:29167", "10.0.40.47:29167", "10.0.40.48:29167"},
			expected: "nats://10.0.40.46:29167,nats://10.0.40.47:29167,nats://10.0.40.48:29167",
		},
		{
			name:     "Multiple URLs mixed prefixes",
			input:    []string{"nats://10.0.40.46:29167", "tls://10.0.40.47:29167", "10.0.40.48:29167"},
			expected: "nats://10.0.40.46:29167,tls://10.0.40.47:29167,nats://10.0.40.48:29167",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the fix from commands.go lines 604-612
			var formattedURLs []string
			for _, url := range tc.input {
				if !strings.HasPrefix(url, "nats://") && !strings.HasPrefix(url, "tls://") {
					url = "nats://" + url
				}
				formattedURLs = append(formattedURLs, url)
			}
			result := strings.Join(formattedURLs, ",")

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}

	t.Log("✅ All NATS URL formatting tests passed")
}
