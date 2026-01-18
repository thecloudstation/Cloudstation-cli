package lifecycle

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
)

// ExecutionContext provides context for lifecycle execution
type ExecutionContext struct {
	// Context is the Go context
	Context context.Context

	// Logger is the structured logger
	Logger hclog.Logger

	// Config is the parsed configuration
	Config *config.Config

	// AppName is the name of the app being executed
	AppName string

	// WorkDir is the working directory
	WorkDir string

	// Variables contains environment variables and config variables
	Variables map[string]string

	// Labels contains labels to attach to artifacts/deployments
	Labels map[string]string
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(ctx context.Context, cfg *config.Config, appName string, workDir string) *ExecutionContext {
	return &ExecutionContext{
		Context:   ctx,
		Logger:    hclog.Default(),
		Config:    cfg,
		AppName:   appName,
		WorkDir:   workDir,
		Variables: make(map[string]string),
		Labels:    make(map[string]string),
	}
}

// WithLogger sets the logger
func (e *ExecutionContext) WithLogger(logger hclog.Logger) *ExecutionContext {
	e.Logger = logger
	return e
}

// WithVariables sets the variables
func (e *ExecutionContext) WithVariables(vars map[string]string) *ExecutionContext {
	e.Variables = vars
	return e
}

// WithLabels sets the labels
func (e *ExecutionContext) WithLabels(labels map[string]string) *ExecutionContext {
	e.Labels = labels
	return e
}
