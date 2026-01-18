package nomadpack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
)

// Platform implements Nomad Pack deployments
type Platform struct {
	config *PlatformConfig
}

// PlatformConfig defines the configuration for Nomad Pack deployments
type PlatformConfig struct {
	// DeploymentName is the name for the Nomad Pack deployment
	DeploymentName string

	// Pack is the name of the pack to deploy
	Pack string

	// NomadAddr is the Nomad API address (e.g., "http://localhost:4646")
	NomadAddr string

	// NomadToken is the Nomad ACL token for authentication
	NomadToken string

	// RegistryName is the name for the Nomad Pack registry
	RegistryName string

	// RegistrySource is the Git URL for the pack registry
	RegistrySource string

	// RegistryRef is the specific git ref/tag/branch (optional)
	RegistryRef string

	// RegistryTarget is the specific pack within the registry (optional)
	RegistryTarget string

	// RegistryToken is a personal access token for private registries (optional)
	RegistryToken string

	// Variables are variable overrides for the pack (optional)
	Variables map[string]string

	// VariableFiles are paths to variable files (optional)
	VariableFiles []string

	// ParserV1 enables legacy pack template syntax (optional, default: false)
	ParserV1 bool

	// Registry is deprecated - use RegistryName instead
	Registry string

	// UseEmbedded uses built-in embedded pack instead of remote registry
	// When true, ignores RegistryName and RegistrySource
	UseEmbedded bool

	// EmbeddedPack overrides Pack name when using embedded packs
	// If empty, uses Pack field. Available: "cloudstation"
	EmbeddedPack string
}

// Deploy deploys a Nomad Pack to the configured Nomad cluster
func (p *Platform) Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error) {
	// Validate configuration
	if p.config == nil {
		return nil, fmt.Errorf("nomadpack platform configuration is not set")
	}

	if p.config.Pack == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	if p.config.DeploymentName == "" {
		return nil, fmt.Errorf("deployment_name is required")
	}

	// Registry fields are only required when not using embedded packs
	if !p.config.UseEmbedded {
		if p.config.RegistryName == "" {
			return nil, fmt.Errorf("registry_name is required")
		}

		if p.config.RegistrySource == "" {
			return nil, fmt.Errorf("registry_source is required")
		}
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	logger.Info("deploying nomad pack", "pack", p.config.Pack, "deployment", p.config.DeploymentName)

	// Determine pack source
	var packPath string
	var cleanupFunc func()

	if p.config.UseEmbedded {
		packName := p.config.EmbeddedPack
		if packName == "" {
			packName = p.config.Pack
		}

		if !HasEmbeddedPack(packName) {
			return nil, fmt.Errorf("embedded pack %q not found, available: %v",
				packName, AvailableEmbeddedPacks())
		}

		var err error
		packPath, err = ExtractEmbeddedPack(packName)
		if err != nil {
			return nil, fmt.Errorf("failed to extract embedded pack: %w", err)
		}
		cleanupFunc = func() { os.RemoveAll(filepath.Dir(packPath)) }

		logger.Info("using embedded pack", "pack", packName, "path", packPath)
	} else {
		// Add registry for remote packs
		if err := p.addRegistry(ctx); err != nil {
			return nil, fmt.Errorf("failed to add registry: %w", err)
		}
	}

	// Cleanup embedded pack on exit
	if cleanupFunc != nil {
		defer cleanupFunc()
	}

	// Build nomad-pack run command
	// For embedded packs, use the extracted path as the pack argument
	// For remote packs, use the pack name with --registry flag
	var args []string
	if p.config.UseEmbedded {
		// nomad-pack run <path> --name <deployment-name>
		args = []string{"run", packPath}
		args = append(args, "--name", p.config.DeploymentName)
	} else {
		// nomad-pack run <pack-name> --name <deployment-name> --registry <registry>
		args = []string{"run", p.config.Pack}
		args = append(args, "--name", p.config.DeploymentName)
		args = append(args, "--registry", p.config.RegistryName)

		// Use the same ref that was used during registry add
		if p.config.RegistryRef != "" {
			args = append(args, "--ref", p.config.RegistryRef)
		}

		// Add legacy parser flag for compatibility (only if explicitly configured)
		if p.config.ParserV1 {
			args = append(args, "--parser-v1")
		}
	}

	// Add variables
	args = p.setVarArgs(args)

	logger.Debug("executing nomad-pack run", "args", args)

	// Create command with context
	cmd := exec.CommandContext(ctx, "nomad-pack", args...)

	// Set environment variables
	cmd.Env = os.Environ()
	if p.config.NomadToken != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_TOKEN=%s", p.config.NomadToken))
	}
	if p.config.NomadAddr != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_ADDR=%s", p.config.NomadAddr))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check if deployment actually succeeded even if nomad-pack exits with error
	// This can happen when output template rendering fails but deployment succeeded
	if err != nil && !strings.Contains(outputStr, "Pack successfully deployed") {
		logger.Error("nomad-pack run failed", "error", err, "output", outputStr)
		return nil, fmt.Errorf("nomad-pack run failed: %w\noutput: %s", err, outputStr)
	}

	if err != nil {
		// Deployment succeeded but output template failed - log warning but continue
		logger.Warn("nomad-pack output template rendering failed, but deployment succeeded", "error", err)
	}

	logger.Info("nomad pack deployed successfully", "pack", p.config.Pack, "deployment", p.config.DeploymentName)
	logger.Debug("nomad-pack run output", "output", outputStr)

	// Generate deployment ID
	deploymentID := fmt.Sprintf("nomadpack-%s-%s", p.config.Pack, p.config.DeploymentName)

	// Create deployment record
	return &deployment.Deployment{
		ID:         deploymentID,
		Name:       p.config.DeploymentName,
		Platform:   "nomad",
		ArtifactID: artifact.ID,
		Status: deployment.DeploymentStatus{
			State:  deployment.StateRunning,
			Health: deployment.HealthHealthy,
		},
		Metadata: map[string]interface{}{
			"pack":          p.config.Pack,
			"nomad_addr":    p.config.NomadAddr,
			"registry_name": p.config.RegistryName,
		},
		DeployedAt: time.Now(),
	}, nil
}

// Destroy removes a Nomad Pack deployment from the Nomad cluster using Nomad CLI
func (p *Platform) Destroy(ctx context.Context, deploymentID string) error {
	// Validate configuration
	if p.config == nil {
		return fmt.Errorf("nomadpack platform configuration is not set")
	}

	if p.config.DeploymentName == "" {
		return fmt.Errorf("deployment_name is required")
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	logger.Info("destroying nomad job", "job", p.config.DeploymentName)

	// Use Nomad CLI directly: nomad job stop -purge {job_name}
	args := []string{"job", "stop", "-purge", p.config.DeploymentName}

	logger.Debug("executing nomad job stop", "args", args)

	// Create command with context
	cmd := exec.CommandContext(ctx, "nomad", args...)

	// Set environment variables
	cmd.Env = os.Environ()
	if p.config.NomadToken != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_TOKEN=%s", p.config.NomadToken))
	}
	if p.config.NomadAddr != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_ADDR=%s", p.config.NomadAddr))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Check if job doesn't exist (not an error - case-insensitive check)
		if strings.Contains(strings.ToLower(outputStr), "no job") {
			logger.Info("job does not exist, nothing to destroy", "job", p.config.DeploymentName)
			return nil
		}
		logger.Error("nomad job stop failed", "error", err, "output", outputStr)
		return fmt.Errorf("nomad job stop failed: %w\noutput: %s", err, outputStr)
	}

	logger.Info("nomad job destroyed successfully", "job", p.config.DeploymentName)
	logger.Debug("nomad job stop output", "output", outputStr)

	return nil
}

// Status returns the status of a Nomad Pack deployment using Nomad CLI
func (p *Platform) Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error) {
	// Validate configuration
	if p.config == nil {
		return nil, fmt.Errorf("nomadpack platform configuration is not set")
	}

	if p.config.DeploymentName == "" {
		return nil, fmt.Errorf("deployment_name is required")
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	logger.Debug("checking nomad job status", "job", p.config.DeploymentName)

	// Use Nomad CLI directly: nomad job status {job_name}
	args := []string{"job", "status", p.config.DeploymentName}

	logger.Debug("executing nomad job status", "args", args)

	// Create command with context
	cmd := exec.CommandContext(ctx, "nomad", args...)

	// Set environment variables
	cmd.Env = os.Environ()
	if p.config.NomadToken != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_TOKEN=%s", p.config.NomadToken))
	}
	if p.config.NomadAddr != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("NOMAD_ADDR=%s", p.config.NomadAddr))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Check if job doesn't exist (case-insensitive check)
		if strings.Contains(strings.ToLower(outputStr), "no job") {
			logger.Debug("job not found", "job", p.config.DeploymentName)
			return &deployment.DeploymentStatus{
				State:       deployment.StateUnknown,
				Health:      deployment.HealthUnknown,
				Message:     "job not found",
				LastChecked: time.Now(),
			}, nil
		}
		logger.Error("nomad job status failed", "error", err, "output", outputStr)
		return nil, fmt.Errorf("nomad job status failed: %w\noutput: %s", err, outputStr)
	}

	logger.Debug("nomad job status output", "output", outputStr)

	// Parse Nomad job status output
	// Looking for "Status       = running" or "Status       = dead"
	var state, health string
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				statusStr := strings.TrimSpace(parts[1])

				switch strings.ToLower(statusStr) {
				case "running":
					state = deployment.StateRunning
					health = deployment.HealthHealthy
				case "pending":
					state = deployment.StatePending
					health = deployment.HealthUnknown
				case "dead":
					state = deployment.StateStopped
					health = deployment.HealthUnknown
				default:
					state = deployment.StateUnknown
					health = deployment.HealthUnknown
				}

				logger.Debug("parsed job status", "status", statusStr, "state", state, "health", health)

				return &deployment.DeploymentStatus{
					State:       state,
					Health:      health,
					Message:     statusStr,
					LastChecked: time.Now(),
				}, nil
			}
		}
	}

	// If we couldn't parse status, return unknown
	return &deployment.DeploymentStatus{
		State:       deployment.StateUnknown,
		Health:      deployment.HealthUnknown,
		Message:     "unable to parse status",
		LastChecked: time.Now(),
	}, nil
}

// Config returns the current configuration
func (p *Platform) Config() (interface{}, error) {
	return p.config, nil
}

// ConfigSet sets the configuration for the platform
func (p *Platform) ConfigSet(config interface{}) error {
	if config == nil {
		p.config = &PlatformConfig{}
		return nil
	}

	// Handle map[string]interface{} configuration
	if configMap, ok := config.(map[string]interface{}); ok {
		p.config = &PlatformConfig{}

		// Helper function to get string value
		getString := func(key string) string {
			if val, ok := configMap[key]; ok {
				if strVal, ok := val.(string); ok {
					return strVal
				}
				if strPtr, ok := val.(*string); ok && strPtr != nil {
					return *strPtr
				}
			}
			return ""
		}

		// Parse string fields
		p.config.DeploymentName = getString("deployment_name")
		p.config.Pack = getString("pack")
		p.config.NomadAddr = getString("nomad_addr")
		p.config.NomadToken = getString("nomad_token")
		p.config.RegistryName = getString("registry_name")
		p.config.RegistrySource = getString("registry_source")
		p.config.RegistryRef = getString("registry_ref")
		p.config.RegistryTarget = getString("registry_target")
		p.config.RegistryToken = getString("registry_token")
		p.config.Registry = getString("registry") // Deprecated

		// Parse use_embedded boolean field
		if val, ok := configMap["use_embedded"]; ok {
			if boolVal, ok := val.(bool); ok {
				p.config.UseEmbedded = boolVal
			}
		}

		// Parse parser_v1 boolean field
		if val, ok := configMap["parser_v1"]; ok {
			if boolVal, ok := val.(bool); ok {
				p.config.ParserV1 = boolVal
			}
		}

		// Parse embedded_pack string field
		if val, ok := configMap["embedded_pack"]; ok {
			if strVal, ok := val.(string); ok {
				p.config.EmbeddedPack = strVal
			}
		}

		// Parse variables map
		if variables, ok := configMap["variables"].(map[string]interface{}); ok {
			p.config.Variables = make(map[string]string)
			for k, v := range variables {
				if strVal, ok := v.(string); ok {
					p.config.Variables[k] = strVal
				} else if strPtr, ok := v.(*string); ok && strPtr != nil {
					p.config.Variables[k] = *strPtr
				}
			}
		}

		// Parse variable_files slice
		if varFiles, ok := configMap["variable_files"].([]interface{}); ok {
			p.config.VariableFiles = make([]string, 0, len(varFiles))
			for _, vf := range varFiles {
				if strVal, ok := vf.(string); ok {
					p.config.VariableFiles = append(p.config.VariableFiles, strVal)
				}
			}
		}

		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*PlatformConfig); ok {
		p.config = cfg
		return nil
	}

	p.config = &PlatformConfig{}
	return nil
}

// addRegistry adds or updates the Nomad Pack registry configuration
func (p *Platform) addRegistry(ctx context.Context) error {
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	// Build registry URL with embedded token if provided
	registryURL := p.config.RegistrySource
	token := p.config.RegistryToken

	// If token is a var reference (e.g., "var.REGISTRY_TOKEN"), resolve from environment
	if strings.HasPrefix(token, "var.") {
		envVarName := strings.TrimPrefix(token, "var.")
		token = os.Getenv(envVarName)
	}

	// If still empty, try to get from REGISTRY_TOKEN env var directly
	if token == "" {
		token = os.Getenv("REGISTRY_TOKEN")
		logger.Info("using REGISTRY_TOKEN from environment", "has_token", token != "")
	}

	if token != "" {
		// Extract protocol and path
		parts := strings.SplitN(registryURL, "://", 2)
		if len(parts) == 2 {
			registryURL = fmt.Sprintf("%s://%s@%s", parts[0], token, parts[1])
		}
	}

	// Build nomad-pack registry add command
	args := []string{"registry", "add", p.config.RegistryName, registryURL}

	if p.config.RegistryTarget != "" {
		args = append(args, "--target", p.config.RegistryTarget)
	}

	if p.config.RegistryRef != "" {
		args = append(args, "--ref", p.config.RegistryRef)
	}

	logger.Debug("adding nomad-pack registry", "name", p.config.RegistryName, "url", registryURL)

	// Create command with context
	cmd := exec.CommandContext(ctx, "nomad-pack", args...)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Registry add may fail if already exists - check if it's that error
		if strings.Contains(outputStr, "already exists") || strings.Contains(outputStr, "Already exists") {
			logger.Debug("nomad-pack registry already exists", "name", p.config.RegistryName)
			return nil
		}
		// Otherwise, this is a real error
		logger.Error("nomad-pack registry add failed", "error", err, "output", outputStr)
		return fmt.Errorf("failed to add nomad-pack registry: %w\noutput: %s", err, outputStr)
	}

	logger.Info("nomad-pack registry added successfully", "name", p.config.RegistryName)
	return nil
}

// setVarArgs appends variable and variable file arguments to the command args
func (p *Platform) setVarArgs(args []string) []string {
	// Add variable arguments
	for key, value := range p.config.Variables {
		args = append(args, fmt.Sprintf("--var=%s=%s", key, value))
	}

	// Add variable file arguments
	for _, varFile := range p.config.VariableFiles {
		args = append(args, fmt.Sprintf("--var-file=%s", varFile))
	}

	return args
}

func init() {
	// Register with both "nomadpack" and "nomad-pack" for compatibility
	p := &plugin.Plugin{
		Platform: &Platform{config: &PlatformConfig{}},
	}
	plugin.Register("nomadpack", p)
	plugin.Register("nomad-pack", p)
}
