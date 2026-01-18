package volume

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientWithToken(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "my-jwt-token")

	if client == nil {
		t.Fatal("NewClientWithToken returned nil")
	}

	if client.BaseClient == nil {
		t.Fatal("BaseClient is nil")
	}

	// Verify Authorization header is set
	if got := client.GetHeader("Authorization"); got != "Bearer my-jwt-token" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer my-jwt-token")
	}
}

func TestAttach_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Verify request path
		if r.URL.Path != "/volumes/svc-123" {
			t.Errorf("expected /volumes/svc-123 path, got %s", r.URL.Path)
		}

		// Verify Authorization header
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}

		// Verify request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var reqBody AttachRequest
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		if reqBody.Capacity != 10 {
			t.Errorf("Capacity = %f, want 10", reqBody.Capacity)
		}
		if len(reqBody.MountPaths) != 1 || reqBody.MountPaths[0] != "/data" {
			t.Errorf("MountPaths = %v, want [\"/data\"]", reqBody.MountPaths)
		}

		// Return success response
		resp := AttachResponse{
			ID:        "vol_abc123",
			ServiceID: "svc-123",
			Message:   "Volume attached successfully",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)
	if err != nil {
		t.Errorf("Attach() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
	if resp.ID != "vol_abc123" {
		t.Errorf("Attach() response.ID = %q, want %q", resp.ID, "vol_abc123")
	}
	if resp.ServiceID != "svc-123" {
		t.Errorf("Attach() response.ServiceID = %q, want %q", resp.ServiceID, "svc-123")
	}
	if resp.Message != "Volume attached successfully" {
		t.Errorf("Attach() response.Message = %q, want %q", resp.Message, "Volume attached successfully")
	}
}

func TestAttach_WithToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		if got := r.Header.Get("Authorization"); got != "Bearer my-session-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer my-session-token")
		}

		// Return success response
		resp := AttachResponse{
			ID:        "vol_def456",
			ServiceID: "svc-456",
			Message:   "Volume attached",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "my-session-token")

	req := AttachRequest{
		Capacity:   5,
		MountPaths: []string{"/app/data"},
	}

	resp, err := client.Attach("svc-456", req)
	if err != nil {
		t.Errorf("Attach() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
	if resp.ID != "vol_def456" {
		t.Errorf("Attach() response.ID = %q, want %q", resp.ID, "vol_def456")
	}
}

func TestAttach_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("", req)

	if err == nil {
		t.Error("Attach() expected error for empty serviceID")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "service ID cannot be empty") {
		t.Errorf("Attach() error = %v, want 'service ID cannot be empty'", err)
	}
}

func TestAttach_InvalidCapacity_Zero(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   0,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for zero capacity")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "capacity must be at least 1 GB") {
		t.Errorf("Attach() error = %v, want 'capacity must be at least 1 GB'", err)
	}
}

func TestAttach_InvalidCapacity_Negative(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   -5,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for negative capacity")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "capacity must be at least 1 GB") {
		t.Errorf("Attach() error = %v, want 'capacity must be at least 1 GB'", err)
	}
}

func TestAttach_InvalidCapacity_LessThanOne(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   0.5,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for capacity less than 1")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "capacity must be at least 1 GB") {
		t.Errorf("Attach() error = %v, want 'capacity must be at least 1 GB'", err)
	}
}

func TestAttach_EmptyMountPaths(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for empty mount paths")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "at least one mount path is required") {
		t.Errorf("Attach() error = %v, want 'at least one mount path is required'", err)
	}
}

func TestAttach_NilMountPaths(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: nil,
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for nil mount paths")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "at least one mount path is required") {
		t.Errorf("Attach() error = %v, want 'at least one mount path is required'", err)
	}
}

func TestAttach_MultipleMountPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var reqBody AttachRequest
		json.Unmarshal(body, &reqBody)

		if len(reqBody.MountPaths) != 3 {
			t.Errorf("MountPaths count = %d, want 3", len(reqBody.MountPaths))
		}

		// Return success response
		resp := AttachResponse{
			ID:        "vol_xyz789",
			ServiceID: "svc-789",
			Message:   "Volume attached",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   20,
		MountPaths: []string{"/data", "/logs", "/cache"},
	}

	resp, err := client.Attach("svc-789", req)
	if err != nil {
		t.Errorf("Attach() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
}

func TestAttach_WithName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body includes name
		body, _ := io.ReadAll(r.Body)
		var reqBody AttachRequest
		json.Unmarshal(body, &reqBody)

		if reqBody.Name == nil {
			t.Error("Name should not be nil")
		} else if *reqBody.Name != "my-volume" {
			t.Errorf("Name = %q, want %q", *reqBody.Name, "my-volume")
		}

		// Return success response
		resp := AttachResponse{
			ID:        "vol_named",
			ServiceID: "svc-123",
			Message:   "Volume attached",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	name := "my-volume"
	req := AttachRequest{
		Capacity:   10,
		Name:       &name,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)
	if err != nil {
		t.Errorf("Attach() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
}

func TestAttach_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for API failure")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
}

func TestAttach_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "invalid-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for unauthorized request")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Errorf("Attach() error should mention 'invalid token', got: %v", err)
	}
}

func TestAttach_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "service not found"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("nonexistent-service", req)

	if err == nil {
		t.Error("Attach() expected error for not found")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "service not found") {
		t.Errorf("Attach() error should mention 'service not found', got: %v", err)
	}
}

func TestAttach_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid mount path"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for bad request")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
}

func TestAttach_ConnectionRefused(t *testing.T) {
	client := NewClientWithToken("http://localhost:65535", "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)

	if err == nil {
		t.Error("Attach() expected error for connection refused")
	}
	if resp != nil {
		t.Error("Attach() expected nil response for error case")
	}
	if !strings.Contains(err.Error(), "failed to attach volume") {
		t.Errorf("Attach() error should mention failure, got: %v", err)
	}
}

// TestAttach_TableDriven provides comprehensive test cases using table-driven tests
func TestAttach_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		request        AttachRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errorContains  string
	}{
		{
			name:      "success with minimum capacity",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   1,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(AttachResponse{ID: "vol_123", ServiceID: "svc-123"})
			},
			wantErr: false,
		},
		{
			name:      "success with large capacity",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   1000,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(AttachResponse{ID: "vol_large", ServiceID: "svc-123"})
			},
			wantErr: false,
		},
		{
			name:      "forbidden response",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "access denied"})
			},
			wantErr:       true,
			errorContains: "access denied",
		},
		{
			name:      "conflict - volume already exists",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "volume already attached"})
			},
			wantErr:       true,
			errorContains: "volume already attached",
		},
		{
			name:      "rate limited",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			},
			wantErr:       true,
			errorContains: "rate limit exceeded",
		},
		{
			name:      "service unavailable",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Service Unavailable"))
			},
			wantErr:       true,
			errorContains: "503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			resp, err := client.Attach(tt.serviceID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if resp != nil {
					t.Error("expected nil response for error case")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("expected non-nil response")
				}
			}
		})
	}
}

// TestAttach_ValidationTableDriven tests input validation using table-driven tests
func TestAttach_ValidationTableDriven(t *testing.T) {
	tests := []struct {
		name          string
		serviceID     string
		request       AttachRequest
		errorContains string
	}{
		{
			name:      "empty service ID",
			serviceID: "",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			},
			errorContains: "service ID cannot be empty",
		},
		{
			name:      "zero capacity",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   0,
				MountPaths: []string{"/data"},
			},
			errorContains: "capacity must be at least 1 GB",
		},
		{
			name:      "negative capacity",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   -10,
				MountPaths: []string{"/data"},
			},
			errorContains: "capacity must be at least 1 GB",
		},
		{
			name:      "fractional capacity less than 1",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   0.99,
				MountPaths: []string{"/data"},
			},
			errorContains: "capacity must be at least 1 GB",
		},
		{
			name:      "empty mount paths",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: []string{},
			},
			errorContains: "at least one mount path is required",
		},
		{
			name:      "nil mount paths",
			serviceID: "svc-123",
			request: AttachRequest{
				Capacity:   10,
				MountPaths: nil,
			},
			errorContains: "at least one mount path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No server needed - validation should fail before HTTP call
			client := NewClientWithToken("https://api.example.com", "test-token")
			resp, err := client.Attach(tt.serviceID, tt.request)

			if err == nil {
				t.Error("expected validation error, got nil")
			}
			if resp != nil {
				t.Error("expected nil response for validation error")
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
			}
		})
	}
}

// TestNewClientWithToken_TrailingSlash tests that NewClientWithToken handles trailing slashes properly
func TestNewClientWithToken_TrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no trailing slash",
			input:    "https://api.example.com",
			expected: "https://api.example.com",
		},
		{
			name:     "single trailing slash",
			input:    "https://api.example.com/",
			expected: "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithToken(tt.input, "token")
			if client.GetBaseURL() != tt.expected {
				t.Errorf("expected baseURL %s, got %s", tt.expected, client.GetBaseURL())
			}
		})
	}
}

// TestAttach_VerifyRequestPath tests that Attach constructs the URL correctly
func TestAttach_VerifyRequestPath(t *testing.T) {
	tests := []struct {
		name         string
		serviceID    string
		expectedPath string
	}{
		{
			name:         "simple service ID",
			serviceID:    "svc-123",
			expectedPath: "/volumes/svc-123",
		},
		{
			name:         "UUID service ID",
			serviceID:    "550e8400-e29b-41d4-a716-446655440000",
			expectedPath: "/volumes/550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:         "service ID with dashes",
			serviceID:    "my-service-with-dashes",
			expectedPath: "/volumes/my-service-with-dashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.expectedPath {
					t.Errorf("path = %q, want %q", r.URL.Path, tt.expectedPath)
				}
				json.NewEncoder(w).Encode(AttachResponse{ID: "vol_123", ServiceID: tt.serviceID})
			}))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			req := AttachRequest{
				Capacity:   10,
				MountPaths: []string{"/data"},
			}
			_, _ = client.Attach(tt.serviceID, req)
		})
	}
}

// TestAttach_LargeCapacity tests handling of large capacity values
func TestAttach_LargeCapacity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody AttachRequest
		json.Unmarshal(body, &reqBody)

		if reqBody.Capacity != 10000 {
			t.Errorf("Capacity = %f, want 10000", reqBody.Capacity)
		}

		resp := AttachResponse{
			ID:        "vol_large",
			ServiceID: "svc-123",
			Message:   "Large volume attached",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10000, // 10TB
		MountPaths: []string{"/data"},
	}

	resp, err := client.Attach("svc-123", req)
	if err != nil {
		t.Errorf("Attach() unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
}

// TestAttach_VerifyHeadersAreSent tests that authentication headers are properly sent
func TestAttach_VerifyHeadersAreSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		if got := r.Header.Get("Authorization"); got != "Bearer my-session-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer my-session-token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", got, "application/json")
		}

		json.NewEncoder(w).Encode(AttachResponse{ID: "vol_123", ServiceID: "svc-123"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "my-session-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/data"},
	}
	_, _ = client.Attach("svc-123", req)
}

// TestAttach_SpecialCharactersInMountPath tests handling of special characters in mount paths
func TestAttach_SpecialCharactersInMountPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody AttachRequest
		json.Unmarshal(body, &reqBody)

		// Verify special characters are preserved
		if len(reqBody.MountPaths) > 0 && reqBody.MountPaths[0] != "/app/data-store_v2/cache" {
			t.Errorf("mount path not preserved: %s", reqBody.MountPaths[0])
		}

		resp := AttachResponse{ID: "vol_123", ServiceID: "svc-123"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := AttachRequest{
		Capacity:   10,
		MountPaths: []string{"/app/data-store_v2/cache"},
	}

	resp, err := client.Attach("svc-123", req)
	if err != nil {
		t.Errorf("Attach() with special characters unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("Attach() returned nil response")
	}
}
