package auth

import "time"

// Credentials holds all authentication and infrastructure credentials
type Credentials struct {
	// Auth metadata - JWT token from Google OAuth
	SessionToken string `json:"session_token"` // JWT from auth service (Google OAuth)

	// User info
	UserID       int    `json:"user_id"`
	UserUUID     string `json:"user_uuid"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"is_super_admin"`

	// Infrastructure credentials
	Vault    VaultCreds    `json:"vault"`
	Nomad    NomadCreds    `json:"nomad"`
	NATS     NATSCreds     `json:"nats"`
	Registry RegistryCreds `json:"registry"`

	// Service links
	Services []ServiceLink `json:"services"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VaultCreds holds HashiCorp Vault authentication credentials
type VaultCreds struct {
	Address   string `json:"address"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// NomadCreds holds HashiCorp Nomad authentication credentials
type NomadCreds struct {
	Address   string `json:"address"`
	SecretID  string `json:"secret_id"`
	Namespace string `json:"namespace"`
	Token     string `json:"token"`
}

// NATSCreds holds NATS messaging system credentials
type NATSCreds struct {
	URLs     []string `json:"urls"`
	User     string   `json:"user"`
	Password string   `json:"password"`
	JWT      string   `json:"jwt"`
	Seed     string   `json:"seed"`
}

// RegistryCreds holds Docker registry authentication credentials
type RegistryCreds struct {
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"` // Image namespace prefix (e.g., "cli/user_abc123")
}

// ServiceLink maps a local project to a CloudStation service
type ServiceLink struct {
	ProjectPath string    `json:"project_path"`
	ServiceID   string    `json:"service_id"`
	ServiceName string    `json:"service_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// ServiceDetails contains full service information for remote deployment
type ServiceDetails struct {
	ServiceID     string `json:"service_id"`
	ServiceName   string `json:"service_name"`
	IntegrationID string `json:"integration_id"` // Needed for external API
	ProjectID     string `json:"project_id"`
	TeamSlug      string `json:"team_slug,omitempty"`
}
