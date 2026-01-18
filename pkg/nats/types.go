package nats

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	StatusInProgress DeploymentStatus = "IN_PROGRESS"
	StatusSucceeded  DeploymentStatus = "SUCCEEDED"
	StatusFailed     DeploymentStatus = "FAILED"
)

// DeploymentEventPayload represents the payload for deployment events
type DeploymentEventPayload struct {
	JobID        int    `json:"jobId"`
	Type         string `json:"type"` // git_repo or image
	DeploymentID string `json:"deploymentId"`
	ServiceID    string `json:"serviceId"`
	TeamID       string `json:"teamId"`
	UserID       string `json:"userId"`
	OwnerID      string `json:"ownerId"`
}

// DeploymentStatusPayload represents the payload for deployment status changes
type DeploymentStatusPayload struct {
	JobID  int              `json:"jobId"`
	Status DeploymentStatus `json:"status"`
}

// JobDestroyReason represents the reason for job destruction
type JobDestroyReason string

const (
	ReasonDelete         JobDestroyReason = "delete"
	ReasonUpgradeVolume  JobDestroyReason = "upgrade_volume"
	ReasonMigrateVolume  JobDestroyReason = "migrate_volume"
	ReasonSuspendAccount JobDestroyReason = "suspend_account"
	ReasonPauseService   JobDestroyReason = "pause_service"
)

// JobDestroyedPayload represents the payload for job destroyed events
type JobDestroyedPayload struct {
	ID     string           `json:"id"`
	Reason JobDestroyReason `json:"reason"`
}

// BuildLogPayload represents a build log event
type BuildLogPayload struct {
	DeploymentID string `json:"deploymentId"`
	JobID        int    `json:"jobId"`
	ServiceID    string `json:"serviceId"`
	OwnerID      string `json:"ownerId"`
	LogOutput    string `json:"logOutput"` // "stdout" or "stderr"
	Content      string `json:"content"`
	Timestamp    int64  `json:"timestamp"` // Unix milliseconds
	Sequence     int    `json:"sequence"`
	Phase        string `json:"phase"` // "clone", "build", "registry", "deploy", "release"
}

// BuildLogEndPayload signals end of build logs
type BuildLogEndPayload struct {
	DeploymentID string `json:"deploymentId"`
	JobID        int    `json:"jobId"`
	Status       string `json:"status"` // "success" or "failed"
}
