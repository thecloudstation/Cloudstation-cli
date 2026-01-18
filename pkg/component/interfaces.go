package component

import (
	"context"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
)

// Builder defines the interface for build plugins that create artifacts
// from source code using various build tools (Docker, Nixpacks, etc.)
type Builder interface {
	// Build executes the build process and returns an artifact
	Build(ctx context.Context) (*artifact.Artifact, error)

	// Config returns the current configuration
	Config() (interface{}, error)

	// ConfigSet sets the configuration for the builder
	ConfigSet(config interface{}) error
}

// Registry defines the interface for registry plugins that push/pull
// artifacts to/from container registries (Docker Hub, ACR, etc.)
type Registry interface {
	// Push pushes an artifact to the registry and returns a reference
	Push(ctx context.Context, artifact *artifact.Artifact) (*artifact.RegistryRef, error)

	// Pull pulls an artifact from the registry (optional)
	Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error)

	// Config returns the current configuration
	Config() (interface{}, error)

	// ConfigSet sets the configuration for the registry
	ConfigSet(config interface{}) error
}

// Platform defines the interface for deployment plugins that deploy
// artifacts to various platforms (Nomad, Kubernetes, Docker, etc.)
type Platform interface {
	// Deploy deploys an artifact and returns deployment information
	Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error)

	// Destroy removes a deployment (optional)
	Destroy(ctx context.Context, deploymentID string) error

	// Status returns the status of a deployment (optional)
	Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error)

	// Config returns the current configuration
	Config() (interface{}, error)

	// ConfigSet sets the configuration for the platform
	ConfigSet(config interface{}) error
}

// ReleaseManager defines the interface for release management plugins
// that handle traffic shifting, canary deployments, etc. (future use)
type ReleaseManager interface {
	// Release performs a release operation
	Release(ctx context.Context, deployment *deployment.Deployment) error

	// Config returns the current configuration
	Config() (interface{}, error)

	// ConfigSet sets the configuration for the release manager
	ConfigSet(config interface{}) error
}

// Configurable is a common interface for all components that can be configured
type Configurable interface {
	// Config returns the current configuration
	Config() (interface{}, error)

	// ConfigSet sets the configuration
	ConfigSet(config interface{}) error
}
