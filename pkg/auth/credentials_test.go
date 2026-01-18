package auth

import (
	"testing"
	"time"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		creds    *Credentials
		expected bool
	}{
		{
			name:     "nil credentials",
			creds:    nil,
			expected: false,
		},
		{
			name: "valid credentials with future expiry",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Vault: VaultCreds{
					ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
				},
			},
			expected: true,
		},
		{
			name: "expired main credentials",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired vault token",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Vault: VaultCreds{
					ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
				},
			},
			expected: false,
		},
		{
			name: "no expiration set",
			creds: &Credentials{
				Email: "test@example.com",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValid(tt.creds)
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsRefresh(t *testing.T) {
	tests := []struct {
		name     string
		creds    *Credentials
		expected bool
	}{
		{
			name:     "nil credentials",
			creds:    nil,
			expected: false,
		},
		{
			name: "expires in 2 minutes",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(2 * time.Minute),
			},
			expected: true,
		},
		{
			name: "expires in 10 minutes",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(10 * time.Minute),
			},
			expected: false,
		},
		{
			name: "vault expires in 3 minutes",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Vault: VaultCreds{
					ExpiresAt: time.Now().Add(3 * time.Minute).Unix(),
				},
			},
			expected: true,
		},
		{
			name: "already expired",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsRefresh(tt.creds)
			if result != tt.expected {
				t.Errorf("NeedsRefresh() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindServiceByID(t *testing.T) {
	creds := &Credentials{
		Services: []ServiceLink{
			{ServiceID: "svc_123", ServiceName: "api"},
			{ServiceID: "svc_456", ServiceName: "web"},
		},
	}

	tests := []struct {
		name      string
		creds     *Credentials
		serviceID string
		found     bool
	}{
		{
			name:      "find existing service",
			creds:     creds,
			serviceID: "svc_123",
			found:     true,
		},
		{
			name:      "service not found",
			creds:     creds,
			serviceID: "svc_999",
			found:     false,
		},
		{
			name:      "empty service ID",
			creds:     creds,
			serviceID: "",
			found:     false,
		},
		{
			name:      "nil credentials",
			creds:     nil,
			serviceID: "svc_123",
			found:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindServiceByID(tt.creds, tt.serviceID)
			if (result != nil) != tt.found {
				t.Errorf("FindServiceByID() found = %v, want %v", result != nil, tt.found)
			}
			if result != nil && result.ServiceID != tt.serviceID {
				t.Errorf("FindServiceByID() returned wrong service: got %s, want %s", result.ServiceID, tt.serviceID)
			}
		})
	}
}

func TestTimeUntilExpiration(t *testing.T) {
	tests := []struct {
		name     string
		creds    *Credentials
		positive bool
	}{
		{
			name:     "nil credentials returns max duration",
			creds:    nil,
			positive: true,
		},
		{
			name: "future expiration",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			positive: true,
		},
		{
			name: "past expiration",
			creds: &Credentials{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			positive: false,
		},
		{
			name:     "no expiration set",
			creds:    &Credentials{},
			positive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := TimeUntilExpiration(tt.creds)
			if tt.positive && duration <= 0 {
				t.Errorf("TimeUntilExpiration() should return positive duration, got %v", duration)
			}
			if !tt.positive && duration > 0 {
				t.Errorf("TimeUntilExpiration() should return zero or negative, got %v", duration)
			}
		})
	}
}
