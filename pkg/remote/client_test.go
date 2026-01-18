package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTriggerDeployment(t *testing.T) {
	// Create a mock server to simulate the backend API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method is POST
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify the URL path
		expectedPath := "/api/external/integrations/deploy/test-integration-id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Verify Authorization header is set
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Authorization header is missing")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if auth != "Bearer test-session-token" {
			t.Errorf("Expected Authorization 'Bearer test-session-token', got '%s'", auth)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Return a successful deployment response
		response := DeploymentResponse{
			DeploymentID: "dep_123",
			Status:       "pending",
			Message:      "Deployment triggered successfully",
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client pointing to the mock server
	client := NewClientWithToken(server.URL, "test-session-token")

	// Trigger a deployment
	resp, err := client.TriggerDeployment("test-integration-id", "")
	if err != nil {
		t.Fatalf("TriggerDeployment failed: %v", err)
	}

	// Verify the deployment ID is returned properly
	if resp.DeploymentID != "dep_123" {
		t.Errorf("Expected deployment ID 'dep_123', got '%s'", resp.DeploymentID)
	}

	// Verify other response fields
	if resp.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", resp.Status)
	}
	if resp.Message != "Deployment triggered successfully" {
		t.Errorf("Expected message 'Deployment triggered successfully', got '%s'", resp.Message)
	}
}

func TestTriggerDeployment_InvalidMethod(t *testing.T) {
	// Create a mock server that only accepts GET (to simulate wrong method)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client
	client := NewClientWithToken(server.URL, "test-session-token")

	// Try to trigger a deployment (should fail because server expects GET, not POST)
	_, err := client.TriggerDeployment("test-integration-id", "")
	if err == nil {
		t.Fatal("Expected error for invalid method, got nil")
	}

	// Verify the error message contains status code
	expectedError := "deployment trigger failed: request failed with status 405"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestTriggerDeployment_MissingAuthorizationHeader(t *testing.T) {
	testCases := []struct {
		name          string
		sessionToken  string
		expectedError bool
	}{
		{
			name:          "missing session token",
			sessionToken:  "",
			expectedError: true,
		},
		{
			name:          "valid session token",
			sessionToken:  "test-session-token",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server that checks for Authorization header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check for missing or invalid Authorization header
				// Note: Go's http library trims trailing whitespace from headers,
				// so "Bearer " becomes "Bearer" when transmitted
				auth := r.Header.Get("Authorization")
				if auth == "" || auth == "Bearer" {
					http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
					return
				}

				// Return success
				response := DeploymentResponse{
					DeploymentID: "dep_456",
					Status:       "pending",
					Message:      "OK",
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create a client with the test case parameters
			client := NewClientWithToken(server.URL, tc.sessionToken)

			// Trigger deployment
			resp, err := client.TriggerDeployment("test-integration-id", "")

			if tc.expectedError {
				if err == nil {
					t.Error("Expected error for missing Authorization header, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp == nil || resp.DeploymentID != "dep_456" {
					t.Error("Expected successful response with deployment ID 'dep_456'")
				}
			}
		})
	}
}

func TestTriggerDeployment_WithTeamParameter(t *testing.T) {
	// Create a mock server that checks for team parameter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if team query parameter is present
		team := r.URL.Query().Get("team")
		if team != "engineering" {
			t.Errorf("Expected team parameter 'engineering', got '%s'", team)
		}

		// Return success
		response := DeploymentResponse{
			DeploymentID: "dep_789",
			Status:       "pending",
			Message:      "Deployment for team engineering",
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client
	client := NewClientWithToken(server.URL, "test-session-token")

	// Trigger deployment with team parameter
	resp, err := client.TriggerDeployment("test-integration-id", "engineering")
	if err != nil {
		t.Fatalf("TriggerDeployment failed: %v", err)
	}

	// Verify response
	if resp.DeploymentID != "dep_789" {
		t.Errorf("Expected deployment ID 'dep_789', got '%s'", resp.DeploymentID)
	}
}

func TestTriggerDeployment_EmptyIntegrationID(t *testing.T) {
	// Create a client (server not needed for this test)
	client := NewClientWithToken("http://example.com", "test-session-token")

	// Try to trigger deployment with empty integration ID
	_, err := client.TriggerDeployment("", "")
	if err == nil {
		t.Fatal("Expected error for empty integration ID, got nil")
	}

	expectedError := "integration ID cannot be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestTriggerDeployment_ServerError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return server error
		errorResp := struct {
			Error string `json:"error"`
		}{
			Error: "Internal server error",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	// Create a client
	client := NewClientWithToken(server.URL, "test-session-token")

	// Try to trigger deployment
	_, err := client.TriggerDeployment("test-integration-id", "")
	if err == nil {
		t.Fatal("Expected error for server error response, got nil")
	}

	expectedError := "deployment trigger failed: request failed with status 500: Internal server error"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestTriggerDeployment_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	// Create a client
	client := NewClientWithToken(server.URL, "test-session-token")

	// Try to trigger deployment
	_, err := client.TriggerDeployment("test-integration-id", "")
	if err == nil {
		t.Fatal("Expected error for invalid JSON response, got nil")
	}

	expectedError := "failed to unmarshal response body"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetDeploymentStatus(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method is GET
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify the URL path
		expectedPath := "/api/v1/deployments/dep_123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-session-token" {
			t.Error("Invalid or missing Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return status response
		response := struct {
			Status string `json:"status"`
		}{
			Status: "completed",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a client
	client := NewClientWithToken(server.URL, "test-session-token")

	// Get deployment status
	status, err := client.GetDeploymentStatus("dep_123")
	if err != nil {
		t.Fatalf("GetDeploymentStatus failed: %v", err)
	}

	// Verify status
	if status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", status)
	}
}

func TestGetDeploymentStatus_EmptyDeploymentID(t *testing.T) {
	// Create a client
	client := NewClientWithToken("http://example.com", "test-session-token")

	// Try to get status with empty deployment ID
	_, err := client.GetDeploymentStatus("")
	if err == nil {
		t.Fatal("Expected error for empty deployment ID, got nil")
	}

	expectedError := "deployment ID cannot be empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
