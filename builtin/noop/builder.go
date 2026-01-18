package noop

import (
	"context"
	"errors"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/component"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
)

// Builder is a no-op builder that immediately returns an empty artifact
type Builder struct {
	config *BuilderConfig
}

// BuilderConfig is the configuration for the noop builder
type BuilderConfig struct {
	// Message is an optional message to include in the artifact metadata
	Message string
}

// Build returns an empty artifact immediately
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
	message := ""
	if b.config != nil {
		message = b.config.Message
	}

	return &artifact.Artifact{
		ID:     "noop-artifact",
		Image:  "noop:latest",
		Tag:    "latest",
		Labels: map[string]string{"builder": "noop"},
		Metadata: map[string]interface{}{
			"builder": "noop",
			"message": message,
		},
		BuildTime: time.Now(),
	}, nil
}

// Config returns the current configuration
func (b *Builder) Config() (interface{}, error) {
	return b.config, nil
}

// ConfigSet sets the configuration
func (b *Builder) ConfigSet(config interface{}) error {
	if config == nil {
		b.config = &BuilderConfig{}
		return nil
	}

	// Handle map[string]interface{} configuration
	if configMap, ok := config.(map[string]interface{}); ok {
		b.config = &BuilderConfig{}
		if msg, ok := configMap["message"].(string); ok {
			b.config.Message = msg
		}
		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*BuilderConfig); ok {
		b.config = cfg
		return nil
	}

	b.config = &BuilderConfig{}
	return nil
}

// NewBuilder creates a new noop builder
func NewBuilder() component.Builder {
	return &Builder{
		config: &BuilderConfig{},
	}
}

// Registry is a no-op registry that returns fake registry references
type Registry struct {
	config *RegistryConfig
}

// RegistryConfig is the configuration for the noop registry
type RegistryConfig struct {
}

// Push returns a fake registry reference immediately
func (r *Registry) Push(ctx context.Context, art *artifact.Artifact) (*artifact.RegistryRef, error) {
	return &artifact.RegistryRef{
		Registry:   "noop-registry",
		Repository: "noop",
		Tag:        "latest",
		FullImage:  "noop-registry/noop:latest",
		PushedAt:   time.Now(),
	}, nil
}

// Pull returns an error as noop registry doesn't support pulling
func (r *Registry) Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error) {
	return nil, errors.New("noop registry: pull not implemented")
}

// Config returns the current configuration
func (r *Registry) Config() (interface{}, error) {
	return r.config, nil
}

// ConfigSet sets the configuration
func (r *Registry) ConfigSet(config interface{}) error {
	if config == nil {
		r.config = &RegistryConfig{}
		return nil
	}

	// Handle map[string]interface{} configuration
	if _, ok := config.(map[string]interface{}); ok {
		r.config = &RegistryConfig{}
		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*RegistryConfig); ok {
		r.config = cfg
		return nil
	}

	r.config = &RegistryConfig{}
	return nil
}

// NewRegistry creates a new noop registry
func NewRegistry() component.Registry {
	return &Registry{
		config: &RegistryConfig{},
	}
}

// Platform is a no-op platform that immediately succeeds
type Platform struct{}

func (p *Platform) Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error) {
	return &deployment.Deployment{
		ID:         "noop-deployment",
		Name:       "noop",
		Platform:   "noop",
		ArtifactID: artifact.ID,
		Status: deployment.DeploymentStatus{
			State:  deployment.StateRunning,
			Health: deployment.HealthHealthy,
		},
		DeployedAt: time.Now(),
	}, nil
}

func (p *Platform) Destroy(ctx context.Context, deploymentID string) error {
	return nil
}

func (p *Platform) Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error) {
	return &deployment.DeploymentStatus{
		State:  deployment.StateRunning,
		Health: deployment.HealthHealthy,
	}, nil
}

func (p *Platform) Config() (interface{}, error) {
	return nil, nil
}

func (p *Platform) ConfigSet(config interface{}) error {
	return nil
}

// ReleaseManager is a no-op release manager that immediately succeeds
type ReleaseManager struct {
	config *ReleaseConfig
}

// ReleaseConfig is the configuration for the noop release manager
type ReleaseConfig struct {
	// Message is an optional message to include in logs
	Message string
}

// Release performs a no-op release and returns success
func (rm *ReleaseManager) Release(ctx context.Context, deployment *deployment.Deployment) error {
	// No-op implementation - just return success
	return nil
}

// Config returns the current configuration
func (rm *ReleaseManager) Config() (interface{}, error) {
	return rm.config, nil
}

// ConfigSet sets the configuration
func (rm *ReleaseManager) ConfigSet(config interface{}) error {
	if config == nil {
		rm.config = &ReleaseConfig{}
		return nil
	}

	// Handle map[string]interface{} configuration
	if configMap, ok := config.(map[string]interface{}); ok {
		rm.config = &ReleaseConfig{}
		if msg, ok := configMap["message"].(string); ok {
			rm.config.Message = msg
		}
		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*ReleaseConfig); ok {
		rm.config = cfg
		return nil
	}

	rm.config = &ReleaseConfig{}
	return nil
}

// NewReleaseManager creates a new noop release manager
func NewReleaseManager() component.ReleaseManager {
	return &ReleaseManager{
		config: &ReleaseConfig{},
	}
}

func init() {
	// Register the noop plugin
	plugin.Register("noop", &plugin.Plugin{
		Builder:        NewBuilder(),
		Registry:       NewRegistry(),
		Platform:       &Platform{},
		ReleaseManager: NewReleaseManager(),
	})
}
