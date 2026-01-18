package backend

// DeploymentStep represents the deployment lifecycle steps
type DeploymentStep string

const (
	StepClone    DeploymentStep = "clone"
	StepBuild    DeploymentStep = "build"
	StepRegistry DeploymentStep = "registry"
	StepDeploy   DeploymentStep = "deploy"
	StepRelease  DeploymentStep = "release"
)

// DeploymentStatus represents the status of a deployment step
type DeploymentStatus string

const (
	StatusInProgress DeploymentStatus = "in_progress"
	StatusCompleted  DeploymentStatus = "completed"
	StatusFailed     DeploymentStatus = "failed"
)

// AskDomainResponse represents the response from /api/local/ask-domain endpoint
type AskDomainResponse struct {
	Domain string `json:"domain"`
}

// NetworkConfig represents a network configuration entry for a service
type NetworkConfig struct {
	Port           int                 `json:"port"`
	Type           string              `json:"type"`             // "http", "tcp", or "none"
	Public         bool                `json:"public"`           // true if port is publicly accessible
	Domain         string              `json:"domain"`           // allocated domain
	HasHealthCheck string              `json:"has_health_check"` // "yes" or "no"
	HealthCheck    HealthCheckSettings `json:"health_check"`     // health check configuration
}

// HealthCheckSettings represents health check configuration for a network port
type HealthCheckSettings struct {
	Type     string `json:"type,omitempty"`     // "http" or "tcp"
	Path     string `json:"path,omitempty"`     // health check path (for HTTP)
	Interval string `json:"interval,omitempty"` // check interval (e.g., "30s")
	Timeout  string `json:"timeout,omitempty"`  // timeout duration (e.g., "5s")
	Port     int    `json:"port,omitempty"`     // port to check
}

// UpdateServiceRequest represents the request payload for /api/local/service-update/ endpoint
type UpdateServiceRequest struct {
	ServiceID  string          `json:"serviceId"`
	Network    []NetworkConfig `json:"network,omitempty"`
	DockerUser string          `json:"docker_user,omitempty"`
	CMD        string          `json:"cmd,omitempty"`
	Entrypoint string          `json:"entrypoint,omitempty"`
}

// UpdateServiceResponse represents the response from /api/local/service-update/ endpoint
type UpdateServiceResponse struct {
	Message string `json:"message,omitempty"`
}

// UpdateDeploymentStepRequest represents the request payload for /api/local/deployment-step/update endpoint
type UpdateDeploymentStepRequest struct {
	DeploymentID   string           `json:"deploymentId"`
	DeploymentType string           `json:"deployment_type"`
	Step           DeploymentStep   `json:"step"`
	Status         DeploymentStatus `json:"status"`
	Error          string           `json:"error,omitempty"`
}

// UpdateDeploymentStepResponse represents the response from /api/local/deployment-step/update endpoint
type UpdateDeploymentStepResponse struct {
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
}
