package lifecycle

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/secrets"
)

// Executor orchestrates the build → registry → deploy → release lifecycle
type Executor struct {
	config       *config.Config
	pluginLoader *plugin.Loader
	logger       hclog.Logger
}

// NewExecutor creates a new lifecycle executor
func NewExecutor(cfg *config.Config, logger hclog.Logger) *Executor {
	if logger == nil {
		logger = hclog.Default()
	}

	return &Executor{
		config:       cfg,
		pluginLoader: plugin.NewLoader(nil, logger),
		logger:       logger,
	}
}

// Execute runs the full lifecycle for an application
func (e *Executor) Execute(ctx context.Context, appName string) error {
	e.logger.Info("starting lifecycle execution", "app", appName)

	// Get app config
	app := e.config.GetApp(appName)
	if app == nil {
		return fmt.Errorf("app %q not found in configuration", appName)
	}

	// Build phase
	artifact, err := e.ExecuteBuild(ctx, app)
	if err != nil {
		return fmt.Errorf("build phase failed: %w", err)
	}

	e.logger.Info("build completed", "artifact", artifact.ID)

	// Registry phase (optional)
	if app.Registry != nil {
		registryRef, err := e.ExecuteRegistry(ctx, app, artifact)
		if err != nil {
			return fmt.Errorf("registry phase failed: %w", err)
		}
		e.logger.Info("registry push completed", "image", registryRef.FullImage)
	}

	// Deploy phase
	dep, err := e.ExecuteDeploy(ctx, app, artifact)
	if err != nil {
		return fmt.Errorf("deploy phase failed: %w", err)
	}

	e.logger.Info("deploy completed", "deployment", dep.ID, "status", dep.Status.State)

	// Release phase (optional)
	if app.Release != nil {
		if err := e.ExecuteRelease(ctx, app, dep); err != nil {
			return fmt.Errorf("release phase failed: %w", err)
		}
		e.logger.Info("release completed")
	}

	e.logger.Info("lifecycle execution completed successfully", "app", appName)
	return nil
}

// ExecuteBuild runs the build phase
func (e *Executor) ExecuteBuild(ctx context.Context, app *config.AppConfig) (*artifact.Artifact, error) {
	e.logger.Debug("executing build phase", "plugin", app.Build.Use)

	// Check if build config contains secret provider configuration
	if providerName, hasSecrets := detectSecretProvider(app.Build.Config); hasSecrets {
		e.logger.Info("detected secret provider configuration", "provider", providerName)

		// Get the secret provider
		provider, err := secrets.GetProvider(providerName)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret provider: %w", err)
		}

		// Enrich config with secrets before loading the builder
		e.logger.Debug("enriching build config with secrets", "provider", providerName)
		if err := enrichConfigWithSecrets(ctx, provider, app.Build.Config, e.logger); err != nil {
			return nil, fmt.Errorf("failed to enrich config with secrets: %w", err)
		}

		// Log redacted config for debugging (without secret values)
		redactedConfig := redactSecretsFromLog(app.Build.Config)
		e.logger.Debug("config enriched with secrets", "config", redactedConfig)
	}

	// Load the builder plugin
	builder, err := e.pluginLoader.LoadBuilder(app.Build.Use, app.Build.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load builder: %w", err)
	}

	// Execute build
	artifact, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	return artifact, nil
}

// ExecuteRegistry runs the registry phase
func (e *Executor) ExecuteRegistry(ctx context.Context, app *config.AppConfig, artifact *artifact.Artifact) (*artifact.RegistryRef, error) {
	e.logger.Debug("executing registry phase", "plugin", app.Registry.Use)

	// Load the registry plugin
	registry, err := e.pluginLoader.LoadRegistry(app.Registry.Use, app.Registry.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Push to registry
	ref, err := registry.Push(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("registry push failed: %w", err)
	}

	return ref, nil
}

// ExecuteDeploy runs the deploy phase
func (e *Executor) ExecuteDeploy(ctx context.Context, app *config.AppConfig, artifact *artifact.Artifact) (*deployment.Deployment, error) {
	e.logger.Debug("executing deploy phase", "plugin", app.Deploy.Use)

	// Load the platform plugin
	platform, err := e.pluginLoader.LoadPlatform(app.Deploy.Use, app.Deploy.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load platform: %w", err)
	}

	// Deploy
	dep, err := platform.Deploy(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("deploy failed: %w", err)
	}

	return dep, nil
}

// ExecuteRelease runs the release phase
func (e *Executor) ExecuteRelease(ctx context.Context, app *config.AppConfig, dep *deployment.Deployment) error {
	e.logger.Debug("executing release phase", "plugin", app.Release.Use)

	// Load the release manager plugin
	rm, err := e.pluginLoader.LoadReleaseManager(app.Release.Use, app.Release.Config)
	if err != nil {
		return fmt.Errorf("failed to load release manager: %w", err)
	}

	// Release
	if err := rm.Release(ctx, dep); err != nil {
		return fmt.Errorf("release failed: %w", err)
	}

	return nil
}

// BuildOnly runs only the build phase
func (e *Executor) BuildOnly(ctx context.Context, appName string) (*artifact.Artifact, error) {
	e.logger.Info("starting build-only execution", "app", appName)

	app := e.config.GetApp(appName)
	if app == nil {
		return nil, fmt.Errorf("app %q not found in configuration", appName)
	}

	artifact, err := e.ExecuteBuild(ctx, app)
	if err != nil {
		return nil, err
	}

	e.logger.Info("build completed", "artifact", artifact.ID)
	return artifact, nil
}

// DeployOnly runs only the deploy phase (requires existing artifact)
func (e *Executor) DeployOnly(ctx context.Context, appName string, artifact *artifact.Artifact) (*deployment.Deployment, error) {
	e.logger.Info("starting deploy-only execution", "app", appName)

	app := e.config.GetApp(appName)
	if app == nil {
		return nil, fmt.Errorf("app %q not found in configuration", appName)
	}

	dep, err := e.ExecuteDeploy(ctx, app, artifact)
	if err != nil {
		return nil, err
	}

	e.logger.Info("deploy completed", "deployment", dep.ID)
	return dep, nil
}
