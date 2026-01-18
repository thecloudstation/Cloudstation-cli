package envvar

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

func TestBulkCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/environment-variables/bulk" {
			t.Errorf("expected /environment-variables/bulk path, got %s", r.URL.Path)
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

		var reqBody BulkCreateRequest
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		if reqBody.ImageID != "service-123" {
			t.Errorf("ImageID = %q, want %q", reqBody.ImageID, "service-123")
		}
		if reqBody.Type != "image" {
			t.Errorf("Type = %q, want %q", reqBody.Type, "image")
		}
		if len(reqBody.Variables) != 2 {
			t.Errorf("Variables count = %d, want 2", len(reqBody.Variables))
		}

		// Return success response
		resp := BulkCreateResponse{
			Message: "success",
			Created: 2,
			Updated: 0,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars := []Variable{
		{Key: "DB_PASSWORD", Value: "secret123"},
		{Key: "API_KEY", Value: "key456"},
	}

	err := client.BulkCreate("service-123", vars)
	if err != nil {
		t.Errorf("BulkCreate() unexpected error: %v", err)
	}
}

func TestBulkCreate_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	vars := []Variable{{Key: "TEST", Value: "value"}}
	err := client.BulkCreate("", vars)

	if err == nil {
		t.Error("BulkCreate() expected error for empty serviceID")
	}
	if !strings.Contains(err.Error(), "service ID cannot be empty") {
		t.Errorf("BulkCreate() error = %v, want 'service ID cannot be empty'", err)
	}
}

func TestBulkCreate_EmptyVariables(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	err := client.BulkCreate("service-123", []Variable{})

	if err == nil {
		t.Error("BulkCreate() expected error for empty variables")
	}
	if !strings.Contains(err.Error(), "variables list cannot be empty") {
		t.Errorf("BulkCreate() error = %v, want 'variables list cannot be empty'", err)
	}
}

func TestBulkCreate_EmptyKey(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	vars := []Variable{
		{Key: "", Value: "value"},
	}
	err := client.BulkCreate("service-123", vars)

	if err == nil {
		t.Error("BulkCreate() expected error for empty key")
	}
	if !strings.Contains(err.Error(), "has empty key") {
		t.Errorf("BulkCreate() error = %v, want 'has empty key'", err)
	}
}

func TestBulkCreate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars := []Variable{{Key: "TEST", Value: "value"}}
	err := client.BulkCreate("service-123", vars)

	if err == nil {
		t.Error("BulkCreate() expected error for API failure")
	}
}

func TestBulkCreateWithResponse_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := BulkCreateResponse{
			Message: "success",
			Created: 2,
			Updated: 1,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars := []Variable{
		{Key: "VAR1", Value: "val1"},
		{Key: "VAR2", Value: "val2"},
		{Key: "VAR3", Value: "val3"},
	}

	created, updated, err := client.BulkCreateWithResponse("service-123", vars)
	if err != nil {
		t.Errorf("BulkCreateWithResponse() unexpected error: %v", err)
	}
	if created != 2 {
		t.Errorf("BulkCreateWithResponse() created = %d, want 2", created)
	}
	if updated != 1 {
		t.Errorf("BulkCreateWithResponse() updated = %d, want 1", updated)
	}
}

func TestBulkCreateWithResponse_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	vars := []Variable{{Key: "TEST", Value: "value"}}
	_, _, err := client.BulkCreateWithResponse("", vars)

	if err == nil {
		t.Error("BulkCreateWithResponse() expected error for empty serviceID")
	}
	if !strings.Contains(err.Error(), "service ID cannot be empty") {
		t.Errorf("BulkCreateWithResponse() error = %v, want 'service ID cannot be empty'", err)
	}
}

func TestBulkCreateWithResponse_EmptyVariables(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	_, _, err := client.BulkCreateWithResponse("service-123", []Variable{})

	if err == nil {
		t.Error("BulkCreateWithResponse() expected error for empty variables")
	}
	if !strings.Contains(err.Error(), "variables list cannot be empty") {
		t.Errorf("BulkCreateWithResponse() error = %v, want 'variables list cannot be empty'", err)
	}
}

func TestBulkCreateWithResponse_EmptyKey(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	vars := []Variable{{Key: "", Value: "value"}}
	_, _, err := client.BulkCreateWithResponse("service-123", vars)

	if err == nil {
		t.Error("BulkCreateWithResponse() expected error for empty key")
	}
	if !strings.Contains(err.Error(), "has empty key") {
		t.Errorf("BulkCreateWithResponse() error = %v, want 'has empty key'", err)
	}
}

func TestList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/environment-variables/service-123" {
			t.Errorf("expected /environment-variables/service-123 path, got %s", r.URL.Path)
		}

		// Return variables
		vars := []Variable{
			{Key: "DB_HOST", Value: "localhost"},
			{Key: "DB_PORT", Value: "5432"},
		}
		json.NewEncoder(w).Encode(vars)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars, err := client.List("service-123")
	if err != nil {
		t.Errorf("List() unexpected error: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("List() returned %d variables, want 2", len(vars))
	}
	if vars[0].Key != "DB_HOST" {
		t.Errorf("List() vars[0].Key = %q, want %q", vars[0].Key, "DB_HOST")
	}
}

func TestList_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	_, err := client.List("")

	if err == nil {
		t.Error("List() expected error for empty serviceID")
	}
}

func TestList_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return null/empty array
		w.Write([]byte("null"))
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars, err := client.List("service-123")
	if err != nil {
		t.Errorf("List() unexpected error: %v", err)
	}
	if vars == nil {
		t.Error("List() returned nil, want empty slice")
	}
	if len(vars) != 0 {
		t.Errorf("List() returned %d variables, want 0", len(vars))
	}
}

func TestList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	_, err := client.List("service-123")

	if err == nil {
		t.Error("List() expected error for API failure")
	}
}

func TestDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		// Verify path contains service ID
		if !strings.HasPrefix(r.URL.Path, "/environment-variables/service-123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify key parameter
		if got := r.URL.Query().Get("key"); got != "DB_PASSWORD" {
			t.Errorf("key param = %q, want %q", got, "DB_PASSWORD")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	err := client.Delete("service-123", "DB_PASSWORD")
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
	}
}

func TestDelete_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	err := client.Delete("", "KEY")

	if err == nil {
		t.Error("Delete() expected error for empty serviceID")
	}
}

func TestDelete_EmptyKey(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	err := client.Delete("service-123", "")

	if err == nil {
		t.Error("Delete() expected error for empty key")
	}
}

func TestDelete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "variable not found"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	err := client.Delete("service-123", "NONEXISTENT")

	if err == nil {
		t.Error("Delete() expected error for API failure")
	}
}

func TestDeleteAll_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/environment-variables/service-123/all" {
			t.Errorf("expected /environment-variables/service-123/all path, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"message": "deleted",
			"deleted": 5,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	deleted, err := client.DeleteAll("service-123")
	if err != nil {
		t.Errorf("DeleteAll() unexpected error: %v", err)
	}
	if deleted != 5 {
		t.Errorf("DeleteAll() deleted = %d, want 5", deleted)
	}
}

func TestDeleteAll_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	_, err := client.DeleteAll("")

	if err == nil {
		t.Error("DeleteAll() expected error for empty serviceID")
	}
}

func TestDeleteAll_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	_, err := client.DeleteAll("service-123")

	if err == nil {
		t.Error("DeleteAll() expected error for API failure")
	}
}

func TestUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "PUT" {
			t.Errorf("expected PUT method, got %s", r.Method)
		}
		if r.URL.Path != "/environment-variables/service-123" {
			t.Errorf("expected /environment-variables/service-123 path, got %s", r.URL.Path)
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		json.Unmarshal(body, &reqBody)

		if reqBody.Key != "DB_PASSWORD" {
			t.Errorf("Key = %q, want %q", reqBody.Key, "DB_PASSWORD")
		}
		if reqBody.Value != "new-secret" {
			t.Errorf("Value = %q, want %q", reqBody.Value, "new-secret")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	err := client.Update("service-123", "DB_PASSWORD", "new-secret")
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}
}

func TestUpdate_EmptyServiceID(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	err := client.Update("", "KEY", "value")

	if err == nil {
		t.Error("Update() expected error for empty serviceID")
	}
}

func TestUpdate_EmptyKey(t *testing.T) {
	client := NewClientWithToken("https://api.example.com", "test-token")

	err := client.Update("service-123", "", "value")

	if err == nil {
		t.Error("Update() expected error for empty key")
	}
}

func TestUpdate_EmptyValueAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	// Empty value should be allowed (e.g., clearing a variable)
	err := client.Update("service-123", "KEY", "")
	if err != nil {
		t.Errorf("Update() should allow empty value, got error: %v", err)
	}
}

func TestUpdate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "variable not found"})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	err := client.Update("service-123", "NONEXISTENT", "value")

	if err == nil {
		t.Error("Update() expected error for API failure")
	}
}

// TestBulkCreate_TableDriven provides comprehensive test cases using table-driven tests
func TestBulkCreate_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		variables      []Variable
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errorContains  string
	}{
		{
			name:      "success with single variable",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(BulkCreateResponse{Message: "success", Created: 1})
			},
			wantErr: false,
		},
		{
			name:      "success with multiple variables",
			serviceID: "service-123",
			variables: []Variable{
				{Key: "VAR1", Value: "val1"},
				{Key: "VAR2", Value: "val2"},
				{Key: "VAR3", Value: "val3"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(BulkCreateResponse{Message: "success", Created: 3})
			},
			wantErr: false,
		},
		{
			name:      "unauthorized response",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid API key"})
			},
			wantErr:       true,
			errorContains: "invalid API key",
		},
		{
			name:      "forbidden response",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "access denied"})
			},
			wantErr:       true,
			errorContains: "access denied",
		},
		{
			name:      "not found response",
			serviceID: "nonexistent-service",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "service not found"})
			},
			wantErr:       true,
			errorContains: "service not found",
		},
		{
			name:      "rate limited response",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			},
			wantErr:       true,
			errorContains: "rate limit exceeded",
		},
		{
			name:      "server error",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
			},
			wantErr:       true,
			errorContains: "internal error",
		},
		{
			name:      "service unavailable",
			serviceID: "service-123",
			variables: []Variable{{Key: "VAR1", Value: "val1"}},
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
			err := client.BulkCreate(tt.serviceID, tt.variables)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestList_TableDriven provides comprehensive test cases for List method
func TestList_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		expectedCount  int
	}{
		{
			name:      "empty list",
			serviceID: "service-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]Variable{})
			},
			wantErr:       false,
			expectedCount: 0,
		},
		{
			name:      "single variable",
			serviceID: "service-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]Variable{{Key: "VAR1", Value: "val1"}})
			},
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name:      "multiple variables",
			serviceID: "service-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				vars := []Variable{
					{Key: "VAR1", Value: "val1"},
					{Key: "VAR2", Value: "val2"},
					{Key: "VAR3", Value: "val3"},
				}
				json.NewEncoder(w).Encode(vars)
			},
			wantErr:       false,
			expectedCount: 3,
		},
		{
			name:      "unauthorized",
			serviceID: "service-123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			vars, err := client.List(tt.serviceID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(vars) != tt.expectedCount {
					t.Errorf("expected %d variables, got %d", tt.expectedCount, len(vars))
				}
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
			client := NewClientWithToken(tt.input, "test-token")
			if client.GetBaseURL() != tt.expected {
				t.Errorf("expected baseURL %s, got %s", tt.expected, client.GetBaseURL())
			}
		})
	}
}

// TestConnectionRefused tests handling of connection refused errors
func TestConnectionRefused(t *testing.T) {
	client := NewClientWithToken("http://localhost:65535", "test-token")

	t.Run("BulkCreate connection refused", func(t *testing.T) {
		err := client.BulkCreate("service-123", []Variable{{Key: "VAR", Value: "val"}})
		if err == nil {
			t.Error("expected error for connection refused")
		}
		if !strings.Contains(err.Error(), "failed to create environment variables") {
			t.Errorf("expected error to mention failure, got: %v", err)
		}
	})

	t.Run("List connection refused", func(t *testing.T) {
		_, err := client.List("service-123")
		if err == nil {
			t.Error("expected error for connection refused")
		}
		if !strings.Contains(err.Error(), "failed to list environment variables") {
			t.Errorf("expected error to mention failure, got: %v", err)
		}
	})

	t.Run("Delete connection refused", func(t *testing.T) {
		err := client.Delete("service-123", "KEY")
		if err == nil {
			t.Error("expected error for connection refused")
		}
		if !strings.Contains(err.Error(), "failed to delete environment variable") {
			t.Errorf("expected error to mention failure, got: %v", err)
		}
	})

	t.Run("DeleteAll connection refused", func(t *testing.T) {
		_, err := client.DeleteAll("service-123")
		if err == nil {
			t.Error("expected error for connection refused")
		}
		if !strings.Contains(err.Error(), "failed to delete all environment variables") {
			t.Errorf("expected error to mention failure, got: %v", err)
		}
	})

	t.Run("Update connection refused", func(t *testing.T) {
		err := client.Update("service-123", "KEY", "value")
		if err == nil {
			t.Error("expected error for connection refused")
		}
		if !strings.Contains(err.Error(), "failed to update environment variable") {
			t.Errorf("expected error to mention failure, got: %v", err)
		}
	})
}

// TestBulkCreate_LargeNumberOfVariables tests handling of many variables at once
func TestBulkCreate_LargeNumberOfVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody BulkCreateRequest
		json.Unmarshal(body, &reqBody)

		resp := BulkCreateResponse{
			Message: "success",
			Created: len(reqBody.Variables),
			Updated: 0,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	// Create 100 variables
	vars := make([]Variable, 100)
	for i := 0; i < 100; i++ {
		vars[i] = Variable{
			Key:   "VAR_" + string(rune('A'+i%26)) + "_" + string(rune('0'+i/26)),
			Value: "value_" + string(rune('0'+i)),
		}
	}

	err := client.BulkCreate("service-123", vars)
	if err != nil {
		t.Errorf("BulkCreate() with 100 variables unexpected error: %v", err)
	}
}

// TestBulkCreate_SpecialCharactersInValues tests handling of special characters
func TestBulkCreate_SpecialCharactersInValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody BulkCreateRequest
		json.Unmarshal(body, &reqBody)

		// Verify special characters are preserved
		if len(reqBody.Variables) > 0 && reqBody.Variables[0].Value != "test\"value'with<special>&chars" {
			t.Errorf("special characters not preserved in value: %s", reqBody.Variables[0].Value)
		}

		resp := BulkCreateResponse{Message: "success", Created: 1}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	vars := []Variable{
		{Key: "SPECIAL_VAR", Value: "test\"value'with<special>&chars"},
	}

	err := client.BulkCreate("service-123", vars)
	if err != nil {
		t.Errorf("BulkCreate() with special characters unexpected error: %v", err)
	}
}

// TestList_VerifyHeadersAreSent tests that authentication headers are properly sent
func TestList_VerifyHeadersAreSent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		if got := r.Header.Get("Authorization"); got != "Bearer my-jwt-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer my-jwt-token")
		}

		json.NewEncoder(w).Encode([]Variable{})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "my-jwt-token")
	_, _ = client.List("service-123")
}

// TestDelete_VerifyPathAndQuery tests that Delete constructs the URL correctly
func TestDelete_VerifyPathAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/environment-variables/service-with-dashes"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}
		expectedKey := "MY_VAR_KEY"
		if got := r.URL.Query().Get("key"); got != expectedKey {
			t.Errorf("key query param = %q, want %q", got, expectedKey)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")
	err := client.Delete("service-with-dashes", "MY_VAR_KEY")
	if err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
	}
}
