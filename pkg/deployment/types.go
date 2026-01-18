package deployment

import "time"

// Deployment represents a deployed application instance
type Deployment struct {
	// ID is a unique identifier for the deployment
	ID string `json:"id"`

	// Name is the human-readable name of the deployment
	Name string `json:"name"`

	// Status is the current status of the deployment
	Status DeploymentStatus `json:"status"`

	// ArtifactID is the ID of the artifact that was deployed
	ArtifactID string `json:"artifact_id"`

	// Platform is the platform where the deployment is running (e.g., "nomad", "kubernetes")
	Platform string `json:"platform"`

	// URL is the URL where the deployment is accessible (if applicable)
	URL string `json:"url,omitempty"`

	// Metadata contains platform-specific deployment information
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// DeployedAt is when the deployment was created
	DeployedAt time.Time `json:"deployed_at"`

	// UpdatedAt is when the deployment was last updated
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	// State is the current state (e.g., "pending", "running", "failed", "stopped")
	State string `json:"state"`

	// Health is the health status (e.g., "healthy", "unhealthy", "unknown")
	Health string `json:"health,omitempty"`

	// Message provides additional status information
	Message string `json:"message,omitempty"`

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int `json:"ready_replicas,omitempty"`

	// TotalReplicas is the total number of replicas
	TotalReplicas int `json:"total_replicas,omitempty"`

	// LastChecked is when the status was last checked
	LastChecked time.Time `json:"last_checked,omitempty"`
}

// DeploymentContext provides context information for the deployment process
type DeploymentContext struct {
	// AppName is the name of the application being deployed
	AppName string

	// ProjectName is the name of the project
	ProjectName string

	// Variables contains environment variables and config variables
	Variables map[string]string

	// Labels contains labels to attach to the deployment
	Labels map[string]string

	// WorkDir is the working directory for the deployment
	WorkDir string
}

// Common deployment states
const (
	StateUnknown  = "unknown"
	StatePending  = "pending"
	StateRunning  = "running"
	StateFailed   = "failed"
	StateStopped  = "stopped"
	StateComplete = "complete"
)

// Common health statuses
const (
	HealthUnknown   = "unknown"
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
	HealthDegraded  = "degraded"
)
