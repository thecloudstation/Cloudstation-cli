package artifact

import "time"

// Artifact represents a build artifact (typically a Docker image)
type Artifact struct {
	// ID is a unique identifier for the artifact
	ID string `json:"id"`

	// Image is the Docker image name (e.g., "myapp:latest")
	Image string `json:"image"`

	// Tag is the image tag
	Tag string `json:"tag,omitempty"`

	// Digest is the SHA256 digest of the image
	Digest string `json:"digest,omitempty"`

	// Labels are metadata labels attached to the artifact
	Labels map[string]string `json:"labels,omitempty"`

	// Metadata contains additional information about the artifact
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ExposedPorts contains the ports detected from the Docker image (EXPOSE directives or PORT env vars)
	ExposedPorts []int `json:"exposed_ports,omitempty"`

	// BuildTime is when the artifact was built
	BuildTime time.Time `json:"build_time"`

	// BuildID is the ID of the build that created this artifact
	BuildID string `json:"build_id,omitempty"`
}

// GetPrimaryPort returns the first exposed port or 0 if no ports are detected
func (a *Artifact) GetPrimaryPort() int {
	if len(a.ExposedPorts) > 0 {
		return a.ExposedPorts[0]
	}
	return 0
}

// RegistryRef represents a reference to an artifact in a container registry
type RegistryRef struct {
	// Registry is the registry URL (e.g., "acrbc001.azurecr.io")
	Registry string `json:"registry"`

	// Repository is the repository name (e.g., "myapp")
	Repository string `json:"repository"`

	// Tag is the image tag
	Tag string `json:"tag"`

	// Digest is the SHA256 digest of the pushed image
	Digest string `json:"digest,omitempty"`

	// FullImage is the complete image reference (registry/repository:tag)
	FullImage string `json:"full_image"`

	// PushedAt is when the artifact was pushed to the registry
	PushedAt time.Time `json:"pushed_at"`
}

// BuildContext provides context information for the build process
type BuildContext struct {
	// WorkDir is the working directory for the build
	WorkDir string

	// AppName is the name of the application being built
	AppName string

	// ProjectName is the name of the project
	ProjectName string

	// Variables contains environment variables and config variables
	Variables map[string]string

	// Labels contains labels to attach to the artifact
	Labels map[string]string
}
