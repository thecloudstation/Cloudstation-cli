package auth

import (
	"fmt"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// Client is the HTTP client for authenticating with CloudStation backend API
type Client struct {
	*httpclient.BaseClient
}

// NewClient creates a new authentication client for the CloudStation backend
func NewClient(baseURL string) *Client {
	return &Client{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
}

// UserService represents a service in the list
type UserService struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	TeamSlug    string `json:"team_slug,omitempty"`
	Status      string `json:"status"`
}

// CreateServiceRequest contains parameters for creating a service
type CreateServiceRequest struct {
	Name        string `json:"name"`
	ProjectName string `json:"project_name,omitempty"`
	RepoURL     string `json:"repo_url,omitempty"`
	Branch      string `json:"branch,omitempty"`
}

// AuthenticateWithGoogle exchanges a Google ID token for CloudStation credentials
// It makes a POST request to the cs-backend /api/cli/auth/google endpoint
func (c *Client) AuthenticateWithGoogle(idToken string) (*Credentials, error) {
	// Validate input
	if idToken == "" {
		return nil, fmt.Errorf("ID token cannot be empty")
	}

	// Build request body
	reqBody := map[string]string{
		"id_token": idToken,
	}

	// Parse the backend response - same structure as Authenticate
	var backendResp struct {
		User struct {
			ID           int    `json:"id"`
			UUID         string `json:"uuid"`
			Email        string `json:"email"`
			FullName     string `json:"full_name,omitempty"`
			IsSuperAdmin bool   `json:"is_super_admin"`
		} `json:"user"`
		Vault struct {
			Address   string `json:"address"`
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"vault"`
		Nomad struct {
			Address string `json:"address"`
			Token   string `json:"token"`
		} `json:"nomad"`
		NATS struct {
			Servers  []string `json:"servers"`
			NKeySeed string   `json:"nkey_seed"`
		} `json:"nats"`
		Registry struct {
			URL       string `json:"url"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			Namespace string `json:"namespace"`
		} `json:"registry"`
		Services []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProjectID     string `json:"project_id"`
			EnvironmentID string `json:"environment_id"`
			ClusterID     string `json:"cluster_id"`
			ClusterDomain string `json:"cluster_domain"`
			SecretsPath   string `json:"secrets_path"`
		} `json:"services"`
		// Additional fields for OAuth - backend may return these for credential persistence
		APIKey   string `json:"api_key,omitempty"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.DoJSON("POST", "/api/cli/auth/google", reqBody, &backendResp); err != nil {
		return nil, fmt.Errorf("Google auth request failed: %w", err)
	}

	// Map backend response to our Credentials type
	creds := &Credentials{
		// User info
		UserID:       backendResp.User.ID,
		UserUUID:     backendResp.User.UUID,
		Email:        backendResp.User.Email,
		IsSuperAdmin: backendResp.User.IsSuperAdmin,

		// Infrastructure credentials
		Vault: VaultCreds{
			Address:   backendResp.Vault.Address,
			Token:     backendResp.Vault.Token,
			ExpiresAt: backendResp.Vault.ExpiresAt,
		},
		Nomad: NomadCreds{
			Address: backendResp.Nomad.Address,
			Token:   backendResp.Nomad.Token,
		},
		NATS: NATSCreds{
			URLs: backendResp.NATS.Servers,
			Seed: backendResp.NATS.NKeySeed,
		},
		Registry: RegistryCreds{
			URL:       backendResp.Registry.URL,
			Username:  backendResp.Registry.Username,
			Password:  backendResp.Registry.Password,
			Namespace: backendResp.Registry.Namespace,
		},

		// Service links - map backend services to our ServiceLink type
		Services: make([]ServiceLink, 0, len(backendResp.Services)),

		// Metadata
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Default 24 hour expiry
	}

	// Convert service information
	for _, svc := range backendResp.Services {
		link := ServiceLink{
			ServiceID:   svc.ID,
			ServiceName: svc.Name,
			CreatedAt:   time.Now(),
		}
		creds.Services = append(creds.Services, link)
	}

	// If Vault token has expiry, use that for credential expiry
	if backendResp.Vault.ExpiresAt > 0 {
		creds.ExpiresAt = time.Unix(backendResp.Vault.ExpiresAt, 0)
	}

	return creds, nil
}

// RequestAppToken requests a scoped ACR token for a specific app
func (c *Client) RequestAppToken(sessionToken, appName string) (*RegistryCreds, time.Time, error) {
	// Validate inputs
	if sessionToken == "" {
		return nil, time.Time{}, fmt.Errorf("session token cannot be empty")
	}
	if appName == "" {
		return nil, time.Time{}, fmt.Errorf("app name cannot be empty")
	}

	// Build request body
	reqBody := map[string]string{
		"app_name": appName,
	}

	// Parse the app token response
	var tokenResp struct {
		Registry struct {
			URL       string `json:"url"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			Namespace string `json:"namespace"`
		} `json:"registry"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := c.DoJSONWithAuth("POST", "/api/cli/app-token", reqBody, &tokenResp, sessionToken); err != nil {
		return nil, time.Time{}, fmt.Errorf("app token request failed: %w", err)
	}

	// Map response to RegistryCreds
	registryCreds := &RegistryCreds{
		URL:       tokenResp.Registry.URL,
		Username:  tokenResp.Registry.Username,
		Password:  tokenResp.Registry.Password,
		Namespace: tokenResp.Registry.Namespace,
	}

	return registryCreds, tokenResp.ExpiresAt, nil
}

// GetServiceDetails retrieves full service details including integration ID
func (c *Client) GetServiceDetails(sessionToken, serviceID string) (*ServiceDetails, error) {
	// Validate inputs
	if sessionToken == "" {
		return nil, fmt.Errorf("session token cannot be empty")
	}
	if serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	}

	// Build request body
	reqBody := map[string]string{
		"service_id": serviceID,
	}

	// Parse the service details response
	var details ServiceDetails
	if err := c.DoJSONWithAuth("POST", "/api/cli/service-details", reqBody, &details, sessionToken); err != nil {
		return nil, fmt.Errorf("service details request failed: %w", err)
	}

	return &details, nil
}

// AuthenticateWithBearer uses a JWT token to get CLI credentials
// Calls /api/cli/auth with Bearer token in Authorization header
func (c *Client) AuthenticateWithBearer(jwtToken string) (*Credentials, error) {
	if jwtToken == "" {
		return nil, fmt.Errorf("JWT token cannot be empty")
	}

	// Parse the backend response
	var backendResp struct {
		User struct {
			ID           int    `json:"id"`
			UUID         string `json:"uuid"`
			Email        string `json:"email"`
			FullName     string `json:"full_name,omitempty"`
			IsSuperAdmin bool   `json:"is_super_admin"`
		} `json:"user"`
		Vault struct {
			Address   string `json:"address"`
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"vault"`
		Nomad struct {
			Address string `json:"address"`
			Token   string `json:"token"`
		} `json:"nomad"`
		NATS struct {
			Servers  []string `json:"servers"`
			NKeySeed string   `json:"nkey_seed"`
		} `json:"nats"`
		Registry struct {
			URL       string `json:"url"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			Namespace string `json:"namespace"`
		} `json:"registry"`
		Services []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProjectID     string `json:"project_id"`
			EnvironmentID string `json:"environment_id"`
			ClusterID     string `json:"cluster_id"`
			ClusterDomain string `json:"cluster_domain"`
			SecretsPath   string `json:"secrets_path"`
		} `json:"services"`
	}

	if err := c.DoJSONWithAuth("POST", "/api/cli/auth", nil, &backendResp, jwtToken); err != nil {
		return nil, fmt.Errorf("auth request failed: %w", err)
	}

	// Map backend response to Credentials
	creds := &Credentials{
		UserID:       backendResp.User.ID,
		UserUUID:     backendResp.User.UUID,
		Email:        backendResp.User.Email,
		IsSuperAdmin: backendResp.User.IsSuperAdmin,
		Vault: VaultCreds{
			Address:   backendResp.Vault.Address,
			Token:     backendResp.Vault.Token,
			ExpiresAt: backendResp.Vault.ExpiresAt,
		},
		Nomad: NomadCreds{
			Address: backendResp.Nomad.Address,
			Token:   backendResp.Nomad.Token,
		},
		NATS: NATSCreds{
			URLs: backendResp.NATS.Servers,
			Seed: backendResp.NATS.NKeySeed,
		},
		Registry: RegistryCreds{
			URL:       backendResp.Registry.URL,
			Username:  backendResp.Registry.Username,
			Password:  backendResp.Registry.Password,
			Namespace: backendResp.Registry.Namespace,
		},
		Services:  make([]ServiceLink, 0, len(backendResp.Services)),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	for _, svc := range backendResp.Services {
		creds.Services = append(creds.Services, ServiceLink{
			ServiceID:   svc.ID,
			ServiceName: svc.Name,
			CreatedAt:   time.Now(),
		})
	}

	if backendResp.Vault.ExpiresAt > 0 {
		creds.ExpiresAt = time.Unix(backendResp.Vault.ExpiresAt, 0)
	}

	return creds, nil
}

// AuthenticateWithJWT exchanges a Stella JWT token for CloudStation credentials
// This is used for the copy-paste OAuth flow where users copy their JWT from localStorage
// It makes a POST request to the cs-backend /api/cli/auth/jwt endpoint
func (c *Client) AuthenticateWithJWT(jwtToken string) (*Credentials, error) {
	// Validate input
	if jwtToken == "" {
		return nil, fmt.Errorf("JWT token cannot be empty")
	}

	// Build request body
	reqBody := map[string]string{
		"jwt_token": jwtToken,
	}

	// Parse the backend response - same structure as Authenticate
	var backendResp struct {
		User struct {
			ID           int    `json:"id"`
			UUID         string `json:"uuid"`
			Email        string `json:"email"`
			FullName     string `json:"full_name,omitempty"`
			IsSuperAdmin bool   `json:"is_super_admin"`
		} `json:"user"`
		Vault struct {
			Address   string `json:"address"`
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"vault"`
		Nomad struct {
			Address string `json:"address"`
			Token   string `json:"token"`
		} `json:"nomad"`
		NATS struct {
			Servers  []string `json:"servers"`
			NKeySeed string   `json:"nkey_seed"`
		} `json:"nats"`
		Registry struct {
			URL       string `json:"url"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			Namespace string `json:"namespace"`
		} `json:"registry"`
		Services []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ProjectID     string `json:"project_id"`
			EnvironmentID string `json:"environment_id"`
			ClusterID     string `json:"cluster_id"`
			ClusterDomain string `json:"cluster_domain"`
			SecretsPath   string `json:"secrets_path"`
		} `json:"services"`
		// Additional fields for OAuth - backend may return these for credential persistence
		APIKey   string `json:"api_key,omitempty"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.DoJSON("POST", "/api/cli/auth/jwt", reqBody, &backendResp); err != nil {
		return nil, fmt.Errorf("JWT auth request failed: %w", err)
	}

	// Map backend response to our Credentials type
	creds := &Credentials{
		// User info
		UserID:       backendResp.User.ID,
		UserUUID:     backendResp.User.UUID,
		Email:        backendResp.User.Email,
		IsSuperAdmin: backendResp.User.IsSuperAdmin,

		// Infrastructure credentials
		Vault: VaultCreds{
			Address:   backendResp.Vault.Address,
			Token:     backendResp.Vault.Token,
			ExpiresAt: backendResp.Vault.ExpiresAt,
		},
		Nomad: NomadCreds{
			Address: backendResp.Nomad.Address,
			Token:   backendResp.Nomad.Token,
		},
		NATS: NATSCreds{
			URLs: backendResp.NATS.Servers,
			Seed: backendResp.NATS.NKeySeed,
		},
		Registry: RegistryCreds{
			URL:       backendResp.Registry.URL,
			Username:  backendResp.Registry.Username,
			Password:  backendResp.Registry.Password,
			Namespace: backendResp.Registry.Namespace,
		},

		// Service links
		Services: make([]ServiceLink, 0, len(backendResp.Services)),

		// Metadata
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Convert service information
	for _, svc := range backendResp.Services {
		link := ServiceLink{
			ServiceID:   svc.ID,
			ServiceName: svc.Name,
			CreatedAt:   time.Now(),
		}
		creds.Services = append(creds.Services, link)
	}

	// If Vault token has expiry, use that for credential expiry
	if backendResp.Vault.ExpiresAt > 0 {
		creds.ExpiresAt = time.Unix(backendResp.Vault.ExpiresAt, 0)
	}

	return creds, nil
}

// ValidateConnection tests if the backend is reachable
func (c *Client) ValidateConnection() error {
	// Simple health check - attempt to reach the backend
	// We use nil for both reqBody and respBody since we're just checking connectivity
	// The DoJSON method will still check the response status code
	if err := c.DoJSON("GET", "/api/health", nil, nil); err != nil {
		// Check if this is a server error vs connection error
		// The DoJSON error will already include status code info if available
		return fmt.Errorf("backend connection failed: %w", err)
	}

	return nil
}

// ListUserServices fetches all services accessible to the user
func (c *Client) ListUserServices(sessionToken string) ([]UserService, error) {
	// Validate inputs
	if sessionToken == "" {
		return nil, fmt.Errorf("session token cannot be empty")
	}

	// Parse the services response
	var result struct {
		Services []UserService `json:"services"`
	}
	// Send empty object as body (backend requires valid JSON body)
	emptyBody := struct{}{}
	if err := c.DoJSONWithAuth("POST", "/api/cli/services", emptyBody, &result, sessionToken); err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	return result.Services, nil
}

// CreateService creates a new service via the backend API
func (c *Client) CreateService(sessionToken string, req *CreateServiceRequest) (*ServiceDetails, error) {
	// Validate inputs
	if sessionToken == "" {
		return nil, fmt.Errorf("session token cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("create service request cannot be nil")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	// Build request body
	body := map[string]string{
		"name": req.Name,
	}

	// Add optional fields if provided
	if req.ProjectName != "" {
		body["project_name"] = req.ProjectName
	}
	if req.RepoURL != "" {
		body["repo_url"] = req.RepoURL
	}
	if req.Branch != "" {
		body["branch"] = req.Branch
	}

	// Parse the service details response
	var details ServiceDetails
	if err := c.DoJSONWithAuth("POST", "/api/cli/create-service", body, &details, sessionToken); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &details, nil
}
