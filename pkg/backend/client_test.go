package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		backendURL  string
		accessToken string
		wantErr     bool
	}{
		{
			name:        "valid configuration",
			backendURL:  "https://api.example.com",
			accessToken: "test-token",
			wantErr:     false,
		},
		{
			name:        "empty backend URL",
			backendURL:  "",
			accessToken: "test-token",
			wantErr:     true,
		},
		{
			name:        "empty access token",
			backendURL:  "https://api.example.com",
			accessToken: "",
			wantErr:     true,
		},
		{
			name:        "backend URL with trailing slash",
			backendURL:  "https://api.example.com/",
			accessToken: "test-token",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.backendURL, tt.accessToken, hclog.NewNullLogger())
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestClient_AskDomain(t *testing.T) {
	tests := []struct {
		name           string
		serviceID      string
		responseCode   int
		responseBody   interface{}
		wantDomain     string
		wantErr        bool
		validateParams bool
	}{
		{
			name:         "successful domain allocation",
			serviceID:    "service-123",
			responseCode: http.StatusOK,
			responseBody: AskDomainResponse{Domain: "subdomain123"},
			wantDomain:   "subdomain123",
			wantErr:      false,
		},
		{
			name:         "empty serviceID",
			serviceID:    "",
			responseCode: http.StatusOK,
			responseBody: AskDomainResponse{Domain: "subdomain123"},
			wantDomain:   "",
			wantErr:      true,
		},
		{
			name:         "server error",
			serviceID:    "service-123",
			responseCode: http.StatusInternalServerError,
			responseBody: "Internal server error",
			wantDomain:   "",
			wantErr:      true,
		},
		{
			name:         "invalid JSON response",
			serviceID:    "service-123",
			responseCode: http.StatusOK,
			responseBody: "invalid json",
			wantDomain:   "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request method
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				// Validate URL path
				if r.URL.Path != "/api/local/ask-domain" {
					t.Errorf("Expected path /api/local/ask-domain, got %s", r.URL.Path)
				}

				// Validate query parameters
				if tt.serviceID != "" {
					serviceID := r.URL.Query().Get("serviceId")
					if serviceID != tt.serviceID {
						t.Errorf("Expected serviceId %s, got %s", tt.serviceID, serviceID)
					}

					accessToken := r.URL.Query().Get("accessToken")
					if accessToken != "test-token" {
						t.Errorf("Expected accessToken test-token, got %s", accessToken)
					}
				}

				// Write response
				w.WriteHeader(tt.responseCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			// Create client
			client, err := NewClient(server.URL, "test-token", hclog.NewNullLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Call AskDomain
			domain, err := client.AskDomain(tt.serviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AskDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if domain != tt.wantDomain {
				t.Errorf("AskDomain() domain = %v, want %v", domain, tt.wantDomain)
			}
		})
	}
}

func TestClient_UpdateService(t *testing.T) {
	tests := []struct {
		name         string
		request      UpdateServiceRequest
		responseCode int
		responseBody interface{}
		wantErr      bool
	}{
		{
			name: "successful service update",
			request: UpdateServiceRequest{
				ServiceID:  "service-123",
				DockerUser: "appuser",
				Network: []NetworkConfig{
					{Port: 3000, Domain: "subdomain123.example.com"},
				},
			},
			responseCode: http.StatusOK,
			responseBody: UpdateServiceResponse{Message: "updated!"},
			wantErr:      false,
		},
		{
			name: "empty serviceID",
			request: UpdateServiceRequest{
				ServiceID: "",
			},
			responseCode: http.StatusOK,
			responseBody: UpdateServiceResponse{Message: "updated!"},
			wantErr:      true,
		},
		{
			name: "server error",
			request: UpdateServiceRequest{
				ServiceID: "service-123",
			},
			responseCode: http.StatusInternalServerError,
			responseBody: "Internal server error",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request method
				if r.Method != "PUT" {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}

				// Validate URL path
				if r.URL.Path != "/api/local/service-update/" {
					t.Errorf("Expected path /api/local/service-update/, got %s", r.URL.Path)
				}

				// Validate Content-Type header
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Validate access token in query
				if tt.request.ServiceID != "" {
					accessToken := r.URL.Query().Get("accessToken")
					if accessToken != "test-token" {
						t.Errorf("Expected accessToken test-token, got %s", accessToken)
					}
				}

				// Write response
				w.WriteHeader(tt.responseCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			// Create client
			client, err := NewClient(server.URL, "test-token", hclog.NewNullLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Call UpdateService
			err = client.UpdateService(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_UpdateDeploymentStep(t *testing.T) {
	tests := []struct {
		name         string
		request      UpdateDeploymentStepRequest
		responseCode int
		responseBody interface{}
		wantErr      bool
	}{
		{
			name: "successful step update",
			request: UpdateDeploymentStepRequest{
				DeploymentID:   "deploy-123",
				DeploymentType: "repository",
				Step:           StepBuild,
				Status:         StatusCompleted,
			},
			responseCode: http.StatusOK,
			responseBody: UpdateDeploymentStepResponse{Success: true},
			wantErr:      false,
		},
		{
			name: "failed step with error message",
			request: UpdateDeploymentStepRequest{
				DeploymentID:   "deploy-123",
				DeploymentType: "repository",
				Step:           StepBuild,
				Status:         StatusFailed,
				Error:          "build failed: compilation error",
			},
			responseCode: http.StatusOK,
			responseBody: UpdateDeploymentStepResponse{Success: true},
			wantErr:      false,
		},
		{
			name: "empty deploymentID",
			request: UpdateDeploymentStepRequest{
				DeploymentID: "",
				Step:         StepBuild,
				Status:       StatusCompleted,
			},
			responseCode: http.StatusOK,
			responseBody: UpdateDeploymentStepResponse{Success: true},
			wantErr:      true,
		},
		{
			name: "empty step",
			request: UpdateDeploymentStepRequest{
				DeploymentID: "deploy-123",
				Step:         "",
				Status:       StatusCompleted,
			},
			responseCode: http.StatusOK,
			responseBody: UpdateDeploymentStepResponse{Success: true},
			wantErr:      true,
		},
		{
			name: "empty status",
			request: UpdateDeploymentStepRequest{
				DeploymentID: "deploy-123",
				Step:         StepBuild,
				Status:       "",
			},
			responseCode: http.StatusOK,
			responseBody: UpdateDeploymentStepResponse{Success: true},
			wantErr:      true,
		},
		{
			name: "server error",
			request: UpdateDeploymentStepRequest{
				DeploymentID: "deploy-123",
				Step:         StepBuild,
				Status:       StatusCompleted,
			},
			responseCode: http.StatusInternalServerError,
			responseBody: "Internal server error",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request method
				if r.Method != "PUT" {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}

				// Validate URL path
				if r.URL.Path != "/api/local/deployment-step/update" {
					t.Errorf("Expected path /api/local/deployment-step/update, got %s", r.URL.Path)
				}

				// Validate Content-Type header
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Validate access token in query
				if tt.request.DeploymentID != "" {
					accessToken := r.URL.Query().Get("accessToken")
					if accessToken != "test-token" {
						t.Errorf("Expected accessToken test-token, got %s", accessToken)
					}
				}

				// Write response
				w.WriteHeader(tt.responseCode)
				if str, ok := tt.responseBody.(string); ok {
					w.Write([]byte(str))
				} else {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			// Create client
			client, err := NewClient(server.URL, "test-token", hclog.NewNullLogger())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Call UpdateDeploymentStep
			err = client.UpdateDeploymentStep(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDeploymentStep() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_RedactToken(t *testing.T) {
	client, err := NewClient("https://api.example.com", "secret-token-123", hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "redact token in URL",
			input: "https://api.example.com/endpoint?accessToken=secret-token-123",
			want:  "https://api.example.com/endpoint?accessToken=[REDACTED]",
		},
		{
			name:  "redact token in plain text",
			input: "Token: secret-token-123",
			want:  "Token: [REDACTED]",
		},
		{
			name:  "no token to redact",
			input: "https://api.example.com/endpoint",
			want:  "https://api.example.com/endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.redactToken(tt.input)
			if got != tt.want {
				t.Errorf("redactToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
