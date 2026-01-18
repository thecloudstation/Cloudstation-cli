package templates

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

func TestNewClientWithToken(t *testing.T) {
	t.Run("creates client with correct configuration", func(t *testing.T) {
		client := NewClientWithToken("https://api.example.com", "test-token")

		if client == nil {
			t.Fatal("NewClientWithToken returned nil")
		}

		if client.GetBaseURL() != "https://api.example.com" {
			t.Errorf("expected base URL https://api.example.com, got %s", client.GetBaseURL())
		}

		if client.GetHeader("Authorization") != "Bearer test-token" {
			t.Error("Authorization header not set correctly")
		}
	})

	t.Run("removes trailing slash from base URL", func(t *testing.T) {
		client := NewClientWithToken("https://api.example.com/", "token")

		if client.GetBaseURL() != "https://api.example.com" {
			t.Errorf("expected trailing slash to be removed, got %s", client.GetBaseURL())
		}
	})
}

func TestClient_List(t *testing.T) {
	tests := []struct {
		name          string
		params        ListTemplatesParams
		response      ListTemplatesResponse
		statusCode    int
		wantErr       bool
		errorContains string
		checkRequest  func(t *testing.T, r *http.Request)
	}{
		{
			name:   "successful list with no params",
			params: ListTemplatesParams{},
			response: ListTemplatesResponse{
				Data:       []Template{{ID: "t1", Name: "Test Template"}},
				Total:      1,
				Page:       1,
				PageSize:   10,
				TotalPages: 1,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/templates" {
					t.Errorf("expected path /templates, got %s", r.URL.Path)
				}
			},
		},
		{
			name:   "successful list with pagination params",
			params: ListTemplatesParams{Page: 2, PageSize: 20},
			response: ListTemplatesResponse{
				Data:       []Template{{ID: "t2", Name: "Template 2"}},
				Total:      25,
				Page:       2,
				PageSize:   20,
				TotalPages: 2,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request) {
				query := r.URL.Query()
				if query.Get("page") != "2" {
					t.Errorf("expected page=2, got %s", query.Get("page"))
				}
				if query.Get("pageSize") != "20" {
					t.Errorf("expected pageSize=20, got %s", query.Get("pageSize"))
				}
			},
		},
		{
			name:   "successful list with search and tags",
			params: ListTemplatesParams{Search: "database", Tags: "postgres,mysql"},
			response: ListTemplatesResponse{
				Data:       []Template{{ID: "t3", Name: "PostgreSQL", Tags: []string{"postgres", "database"}}},
				Total:      1,
				Page:       1,
				PageSize:   10,
				TotalPages: 1,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request) {
				query := r.URL.Query()
				if query.Get("search") != "database" {
					t.Errorf("expected search=database, got %s", query.Get("search"))
				}
				if query.Get("tags") != "postgres,mysql" {
					t.Errorf("expected tags=postgres,mysql, got %s", query.Get("tags"))
				}
			},
		},
		{
			name:   "successful list with sort param",
			params: ListTemplatesParams{Sort: "name_ASC"},
			response: ListTemplatesResponse{
				Data:       []Template{},
				Total:      0,
				Page:       1,
				PageSize:   10,
				TotalPages: 0,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request) {
				query := r.URL.Query()
				if query.Get("sort") != "name_ASC" {
					t.Errorf("expected sort=name_ASC, got %s", query.Get("sort"))
				}
			},
		},
		{
			name:          "server error returns error",
			params:        ListTemplatesParams{},
			statusCode:    http.StatusInternalServerError,
			wantErr:       true,
			errorContains: "500",
		},
		{
			name:          "unauthorized returns error",
			params:        ListTemplatesParams{},
			statusCode:    http.StatusUnauthorized,
			wantErr:       true,
			errorContains: "401",
		},
		{
			name:          "bad request returns error",
			params:        ListTemplatesParams{},
			statusCode:    http.StatusBadRequest,
			wantErr:       true,
			errorContains: "400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify auth header
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing or incorrect Authorization header")
				}

				// Run custom request checks if provided
				if tt.checkRequest != nil {
					tt.checkRequest(t, r)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			resp, err := client.List(tt.params)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if resp.Total != tt.response.Total {
				t.Errorf("expected total %d, got %d", tt.response.Total, resp.Total)
			}
			if len(resp.Data) != len(tt.response.Data) {
				t.Errorf("expected %d templates, got %d", len(tt.response.Data), len(resp.Data))
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name          string
		idOrSlug      string
		response      Template
		statusCode    int
		wantErr       bool
		errorContains string
	}{
		{
			name:     "successful get by ID",
			idOrSlug: "template-123",
			response: Template{
				ID:          "template-123",
				Name:        "Test Template",
				Description: "A test template",
				Tags:        []string{"database", "postgres"},
				Status:      "published",
				Visibility:  "public",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:     "successful get by slug",
			idOrSlug: "postgres-template",
			response: Template{
				ID:   "t-456",
				Name: "PostgreSQL Template",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:          "not found returns error",
			idOrSlug:      "nonexistent",
			statusCode:    http.StatusNotFound,
			wantErr:       true,
			errorContains: "404",
		},
		{
			name:          "server error returns error",
			idOrSlug:      "template-123",
			statusCode:    http.StatusInternalServerError,
			wantErr:       true,
			errorContains: "500",
		},
		{
			name:          "unauthorized returns error",
			idOrSlug:      "template-123",
			statusCode:    http.StatusUnauthorized,
			wantErr:       true,
			errorContains: "401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}

				// Verify path
				expectedPath := "/templates/" + tt.idOrSlug
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify auth header
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing or incorrect Authorization header")
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					// Return wrapped response matching API structure
					json.NewEncoder(w).Encode(map[string]Template{"data": tt.response})
				}
			}))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			resp, err := client.Get(tt.idOrSlug)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if resp.ID != tt.response.ID {
				t.Errorf("expected ID %s, got %s", tt.response.ID, resp.ID)
			}
			if resp.Name != tt.response.Name {
				t.Errorf("expected Name %s, got %s", tt.response.Name, resp.Name)
			}
		})
	}
}

func TestClient_GetTags(t *testing.T) {
	t.Run("successful get tags", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify method
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			// Verify path
			if r.URL.Path != "/templates/tags" {
				t.Errorf("expected path /templates/tags, got %s", r.URL.Path)
			}

			// Verify auth header
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("missing or incorrect Authorization header")
			}

			json.NewEncoder(w).Encode(TagsResponse{Tags: []string{"database", "web", "cache", "monitoring"}})
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		tags, err := client.GetTags()

		if err != nil {
			t.Fatalf("GetTags() error = %v", err)
		}

		if len(tags) != 4 {
			t.Errorf("expected 4 tags, got %d", len(tags))
		}

		expectedTags := []string{"database", "web", "cache", "monitoring"}
		for i, tag := range tags {
			if tag != expectedTags[i] {
				t.Errorf("expected tag %s at index %d, got %s", expectedTags[i], i, tag)
			}
		}
	})

	t.Run("empty tags list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(TagsResponse{Tags: []string{}})
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		tags, err := client.GetTags()

		if err != nil {
			t.Fatalf("GetTags() error = %v", err)
		}

		if len(tags) != 0 {
			t.Errorf("expected 0 tags, got %d", len(tags))
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.GetTags()

		if err == nil {
			t.Error("expected error for server error")
		}

		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected error to contain '500', got: %v", err)
		}
	})

	t.Run("unauthorized returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.GetTags()

		if err == nil {
			t.Error("expected error for unauthorized")
		}

		if !strings.Contains(err.Error(), "invalid token") {
			t.Errorf("expected error to contain 'invalid token', got: %v", err)
		}
	})
}

func TestClient_Deploy(t *testing.T) {
	tests := []struct {
		name          string
		request       DeployTemplateRequest
		response      DeployTemplateResponse
		statusCode    int
		wantErr       bool
		errorContains string
		checkRequest  func(t *testing.T, r *http.Request, body map[string]interface{})
	}{
		{
			name: "successful deploy with minimal params",
			request: DeployTemplateRequest{
				TemplateID: "t1",
				ProjectID:  "proj1",
			},
			response: DeployTemplateResponse{
				ID:                   "dep1",
				TemplateID:           "t1",
				ProjectID:            "proj1",
				ServiceIDs:           []string{"svc1", "svc2"},
				DeploymentInstanceID: "di1",
				Message:              "Deployment initiated",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request, body map[string]interface{}) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/templates/deploy" {
					t.Errorf("expected path /templates/deploy, got %s", r.URL.Path)
				}
				if body["templateId"] != "t1" {
					t.Errorf("expected templateId 't1', got %v", body["templateId"])
				}
				if body["projectId"] != "proj1" {
					t.Errorf("expected projectId 'proj1', got %v", body["projectId"])
				}
			},
		},
		{
			name: "successful deploy with environment",
			request: DeployTemplateRequest{
				TemplateID:  "t2",
				ProjectID:   "proj2",
				Environment: "production",
			},
			response: DeployTemplateResponse{
				ID:                   "dep2",
				TemplateID:           "t2",
				ProjectID:            "proj2",
				ServiceIDs:           []string{"svc3"},
				DeploymentInstanceID: "di2",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request, body map[string]interface{}) {
				if body["environment"] != "production" {
					t.Errorf("expected environment 'production', got %v", body["environment"])
				}
			},
		},
		{
			name: "successful deploy with variables",
			request: DeployTemplateRequest{
				TemplateID: "t3",
				ProjectID:  "proj3",
				Variables: []DeployVariableInput{
					{Key: "DB_HOST", Value: "localhost"},
					{Key: "DB_PORT", Value: "5432"},
				},
			},
			response: DeployTemplateResponse{
				ID:                   "dep3",
				TemplateID:           "t3",
				ProjectID:            "proj3",
				ServiceIDs:           []string{"svc4"},
				DeploymentInstanceID: "di3",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			checkRequest: func(t *testing.T, r *http.Request, body map[string]interface{}) {
				variables, ok := body["variables"].([]interface{})
				if !ok {
					t.Error("expected variables to be an array")
					return
				}
				if len(variables) != 2 {
					t.Errorf("expected 2 variables, got %d", len(variables))
				}
			},
		},
		{
			name: "bad request returns error",
			request: DeployTemplateRequest{
				TemplateID: "invalid",
				ProjectID:  "proj1",
			},
			statusCode:    http.StatusBadRequest,
			wantErr:       true,
			errorContains: "400",
		},
		{
			name: "not found template returns error",
			request: DeployTemplateRequest{
				TemplateID: "nonexistent",
				ProjectID:  "proj1",
			},
			statusCode:    http.StatusNotFound,
			wantErr:       true,
			errorContains: "404",
		},
		{
			name: "server error returns error",
			request: DeployTemplateRequest{
				TemplateID: "t1",
				ProjectID:  "proj1",
			},
			statusCode:    http.StatusInternalServerError,
			wantErr:       true,
			errorContains: "500",
		},
		{
			name: "unauthorized returns error",
			request: DeployTemplateRequest{
				TemplateID: "t1",
				ProjectID:  "proj1",
			},
			statusCode:    http.StatusUnauthorized,
			wantErr:       true,
			errorContains: "401",
		},
		{
			name: "forbidden returns error",
			request: DeployTemplateRequest{
				TemplateID: "t1",
				ProjectID:  "proj1",
			},
			statusCode:    http.StatusForbidden,
			wantErr:       true,
			errorContains: "403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify auth header
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing or incorrect Authorization header")
				}

				// Verify Content-Type
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("missing Content-Type header")
				}

				// Parse request body for custom checks
				if tt.checkRequest != nil {
					var body map[string]interface{}
					json.NewDecoder(r.Body).Decode(&body)
					tt.checkRequest(t, r, body)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClientWithToken(server.URL, "test-token")
			resp, err := client.Deploy(tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Deploy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if resp.DeploymentInstanceID != tt.response.DeploymentInstanceID {
				t.Errorf("expected deployment instance %s, got %s", tt.response.DeploymentInstanceID, resp.DeploymentInstanceID)
			}
			if resp.TemplateID != tt.response.TemplateID {
				t.Errorf("expected template ID %s, got %s", tt.response.TemplateID, resp.TemplateID)
			}
			if len(resp.ServiceIDs) != len(tt.response.ServiceIDs) {
				t.Errorf("expected %d service IDs, got %d", len(tt.response.ServiceIDs), len(resp.ServiceIDs))
			}
		})
	}
}

func TestClient_InvalidJSON(t *testing.T) {
	t.Run("List with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.List(ListTemplatesParams{})

		if err == nil {
			t.Error("expected error for invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal response body") {
			t.Errorf("expected unmarshal error, got: %v", err)
		}
	})

	t.Run("Get with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid}"))
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.Get("template-123")

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("GetTags with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.GetTags()

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("Deploy with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid"))
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.Deploy(DeployTemplateRequest{TemplateID: "t1", ProjectID: "p1"})

		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestClient_NetworkError(t *testing.T) {
	// Use an invalid URL that will cause connection failure
	client := NewClientWithToken("http://localhost:65535", "test-token")

	t.Run("List network error", func(t *testing.T) {
		_, err := client.List(ListTemplatesParams{})

		if err == nil {
			t.Error("expected error for network failure")
		}

		if !strings.Contains(err.Error(), "list templates failed") {
			t.Errorf("expected 'list templates failed' error, got: %v", err)
		}
	})

	t.Run("Get network error", func(t *testing.T) {
		_, err := client.Get("template-123")

		if err == nil {
			t.Error("expected error for network failure")
		}

		if !strings.Contains(err.Error(), "get template failed") {
			t.Errorf("expected 'get template failed' error, got: %v", err)
		}
	})

	t.Run("GetTags network error", func(t *testing.T) {
		_, err := client.GetTags()

		if err == nil {
			t.Error("expected error for network failure")
		}

		if !strings.Contains(err.Error(), "get tags failed") {
			t.Errorf("expected 'get tags failed' error, got: %v", err)
		}
	})

	t.Run("Deploy network error", func(t *testing.T) {
		_, err := client.Deploy(DeployTemplateRequest{TemplateID: "t1", ProjectID: "p1"})

		if err == nil {
			t.Error("expected error for network failure")
		}

		if !strings.Contains(err.Error(), "deploy template failed") {
			t.Errorf("expected 'deploy template failed' error, got: %v", err)
		}
	})
}

func TestClient_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with very short timeout
	client := &Client{
		BaseClient: httpclient.NewBaseClient(server.URL, 100*time.Millisecond),
	}
	client.SetHeader("Authorization", "Bearer test-token")

	t.Run("List timeout", func(t *testing.T) {
		_, err := client.List(ListTemplatesParams{})

		if err == nil {
			t.Error("expected error for timeout")
		}
	})

	t.Run("Get timeout", func(t *testing.T) {
		_, err := client.Get("template-123")

		if err == nil {
			t.Error("expected error for timeout")
		}
	})

	t.Run("GetTags timeout", func(t *testing.T) {
		_, err := client.GetTags()

		if err == nil {
			t.Error("expected error for timeout")
		}
	})

	t.Run("Deploy timeout", func(t *testing.T) {
		_, err := client.Deploy(DeployTemplateRequest{TemplateID: "t1", ProjectID: "p1"})

		if err == nil {
			t.Error("expected error for timeout")
		}
	})
}

func TestClient_ErrorMessageParsing(t *testing.T) {
	t.Run("JSON error message is parsed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "template not found"})
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.Get("nonexistent")

		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), "template not found") {
			t.Errorf("expected error message to contain 'template not found', got: %v", err)
		}
	})

	t.Run("message field is also parsed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "deployment failed"})
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.Deploy(DeployTemplateRequest{TemplateID: "t1", ProjectID: "p1"})

		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), "deployment failed") {
			t.Errorf("expected error message to contain 'deployment failed', got: %v", err)
		}
	})

	t.Run("plain text error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service Unavailable"))
		}))
		defer server.Close()

		client := NewClientWithToken(server.URL, "test-token")
		_, err := client.List(ListTemplatesParams{})

		if err == nil {
			t.Error("expected error")
		}

		if !strings.Contains(err.Error(), "503") {
			t.Errorf("expected error to contain status code 503, got: %v", err)
		}
	})
}

func TestClient_LargeResponse(t *testing.T) {
	// Test handling of response with many templates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templates := make([]Template, 100)
		for i := 0; i < 100; i++ {
			templates[i] = Template{
				ID:          string(rune('a' + i%26)),
				Name:        "Template " + string(rune('A'+i%26)),
				Description: "Description for template",
				Tags:        []string{"tag1", "tag2"},
			}
		}

		response := ListTemplatesResponse{
			Data:       templates,
			Total:      100,
			Page:       1,
			PageSize:   100,
			TotalPages: 1,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")
	resp, err := client.List(ListTemplatesParams{PageSize: 100})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Data) != 100 {
		t.Errorf("expected 100 templates, got %d", len(resp.Data))
	}
}

func TestClient_CompleteTemplate(t *testing.T) {
	// Test complete template with all fields populated
	expectedTemplate := Template{
		ID:          "tmpl-123",
		Name:        "Full Featured Template",
		Description: "A template with all fields",
		Image:       "https://example.com/image.png",
		Visibility:  "public",
		Status:      "published",
		Tags:        []string{"database", "postgres", "production"},
		Definition: map[string]interface{}{
			"services": []interface{}{
				map[string]interface{}{"name": "postgres", "image": "postgres:15"},
			},
		},
		Deployments: 42,
		TotalApps:   10,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return wrapped response matching API structure
		json.NewEncoder(w).Encode(map[string]Template{"data": expectedTemplate})
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")
	resp, err := client.Get("tmpl-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != expectedTemplate.ID {
		t.Errorf("expected ID %s, got %s", expectedTemplate.ID, resp.ID)
	}
	if resp.Name != expectedTemplate.Name {
		t.Errorf("expected Name %s, got %s", expectedTemplate.Name, resp.Name)
	}
	if resp.Description != expectedTemplate.Description {
		t.Errorf("expected Description %s, got %s", expectedTemplate.Description, resp.Description)
	}
	if resp.Image != expectedTemplate.Image {
		t.Errorf("expected Image %s, got %s", expectedTemplate.Image, resp.Image)
	}
	if resp.Visibility != expectedTemplate.Visibility {
		t.Errorf("expected Visibility %s, got %s", expectedTemplate.Visibility, resp.Visibility)
	}
	if resp.Status != expectedTemplate.Status {
		t.Errorf("expected Status %s, got %s", expectedTemplate.Status, resp.Status)
	}
	if len(resp.Tags) != len(expectedTemplate.Tags) {
		t.Errorf("expected %d tags, got %d", len(expectedTemplate.Tags), len(resp.Tags))
	}
	if resp.Deployments != expectedTemplate.Deployments {
		t.Errorf("expected Deployments %f, got %f", expectedTemplate.Deployments, resp.Deployments)
	}
	if resp.TotalApps != expectedTemplate.TotalApps {
		t.Errorf("expected TotalApps %f, got %f", expectedTemplate.TotalApps, resp.TotalApps)
	}
	if resp.Definition == nil {
		t.Error("expected Definition to be populated")
	}
}

func TestClient_DeployWithAllFields(t *testing.T) {
	// Test deploy with all request fields and verify response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req DeployTemplateRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify all fields are received
		if req.TemplateID != "tmpl-full" {
			t.Errorf("expected templateId 'tmpl-full', got %s", req.TemplateID)
		}
		if req.ProjectID != "proj-full" {
			t.Errorf("expected projectId 'proj-full', got %s", req.ProjectID)
		}
		if req.Environment != "staging" {
			t.Errorf("expected environment 'staging', got %s", req.Environment)
		}
		if len(req.Variables) != 2 {
			t.Errorf("expected 2 variables, got %d", len(req.Variables))
		}

		// Check variable details
		if len(req.Variables) >= 1 {
			if req.Variables[0].Key != "VAR1" {
				t.Errorf("expected first variable key 'VAR1', got %s", req.Variables[0].Key)
			}
			if req.Variables[0].Value != "value1" {
				t.Errorf("expected first variable value 'value1', got %s", req.Variables[0].Value)
			}
			if req.Variables[0].TemplateName != "service-a" {
				t.Errorf("expected first variable template name 'service-a', got %s", req.Variables[0].TemplateName)
			}
		}

		response := DeployTemplateResponse{
			ID:                   "deploy-full",
			TemplateID:           req.TemplateID,
			ProjectID:            req.ProjectID,
			ServiceIDs:           []string{"svc-a", "svc-b", "svc-c"},
			DeploymentInstanceID: "di-full",
			Message:              "Deployment successful",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithToken(server.URL, "test-token")

	req := DeployTemplateRequest{
		TemplateID:  "tmpl-full",
		ProjectID:   "proj-full",
		Environment: "staging",
		Variables: []DeployVariableInput{
			{TemplateName: "service-a", Key: "VAR1", Value: "value1"},
			{Key: "VAR2", Value: "value2"},
		},
	}

	resp, err := client.Deploy(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != "deploy-full" {
		t.Errorf("expected ID 'deploy-full', got %s", resp.ID)
	}
	if len(resp.ServiceIDs) != 3 {
		t.Errorf("expected 3 service IDs, got %d", len(resp.ServiceIDs))
	}
	if resp.Message != "Deployment successful" {
		t.Errorf("expected message 'Deployment successful', got %s", resp.Message)
	}
}
