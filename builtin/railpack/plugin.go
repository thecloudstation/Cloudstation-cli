package railpack

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/portdetector"
)

// Builder implements Railpack build
type Builder struct {
	config *BuilderConfig
}

// BuilderConfig holds the configuration for railpack builds
type BuilderConfig struct {
	// Name is the Docker image name (required)
	Name string

	// Tag is the Docker image tag (defaults to "latest")
	Tag string

	// Context is the build directory path (defaults to ".")
	Context string

	// BuildArgs are additional build arguments for railpack
	BuildArgs map[string]string

	// Env contains environment variables to pass to build
	Env map[string]string

	// Deprecated: Vault fields are no longer used directly.
	// Secrets are now injected via the env map by the secret provider at the lifecycle layer.
	// These fields are kept for backward compatibility but have no effect.
	VaultAddress string
	RoleID       string
	SecretID     string
	SecretsPath  string
}

func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
	// Check if config is set
	if b.config == nil {
		return nil, fmt.Errorf("railpack builder configuration is not set")
	}

	// Validate required fields
	if b.config.Name == "" {
		return nil, fmt.Errorf("railpack builder requires 'name' field to be set")
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	// Set defaults
	context := b.config.Context
	if context == "" {
		context = "."
	}

	tag := b.config.Tag
	if tag == "" {
		tag = "latest"
	}

	imageName := fmt.Sprintf("%s:%s", b.config.Name, tag)

	logger.Info("starting railpack build", "image", imageName, "context", context)

	// Build railpack command arguments
	// Note: railpack uses --name with the full image:tag format
	// Determine the build context argument
	// If we're setting cmd.Dir, use "." as context to avoid double-pathing
	// (e.g., passing "/tmp/upload-xyz" as arg while also setting cmd.Dir to "/tmp/upload-xyz"
	// would cause railpack to look for "/tmp/upload-xyz/tmp/upload-xyz")
	buildContext := context
	if context != "." && context != "" {
		buildContext = "." // Will be relative to cmd.Dir
	}
	args := []string{"build", buildContext}
	args = append(args, "--name", imageName)

	// Note: railpack doesn't support --build-arg like Docker
	// BuildArgs are ignored for railpack builds

	// Add environment variables if provided
	for key, value := range b.config.Env {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
	}

	logger.Debug("executing railpack", "args", args, "workdir", context)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, "railpack", args...)

	// Set working directory if context is not current directory
	if context != "." && context != "" {
		cmd.Dir = context
	}

	// Inherit environment variables (includes BUILDKIT_HOST if set)
	cmd.Env = append(cmd.Env, os.Environ()...)

	// Ensure Docker BuildKit is enabled for railpack
	cmd.Env = append(cmd.Env, "DOCKER_BUILDKIT=1")

	// Check for NATS writers in context first (preferred), then buffer
	stdoutWriter, hasStdoutWriter := ctx.Value("stdoutWriter").(io.Writer)
	stderrWriter, hasStderrWriter := ctx.Value("stderrWriter").(io.Writer)

	if hasStdoutWriter && hasStderrWriter {
		// Use NATS writers for real-time streaming
		cmd.Stdout = stdoutWriter
		cmd.Stderr = stderrWriter

		// Execute the command
		err := cmd.Run()

		// Flush NATS writers if they support it
		if flusher, ok := stdoutWriter.(interface{ Flush() error }); ok {
			flusher.Flush()
		}
		if flusher, ok := stderrWriter.(interface{ Flush() error }); ok {
			flusher.Flush()
		}

		if err != nil {
			logger.Error("railpack build failed", "error", err)
			return nil, fmt.Errorf("railpack build failed: %w", err)
		}
	} else {
		// Fallback to buffer for non-dispatch scenarios
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// Execute the command
		err := cmd.Run()

		// Log output
		if stdout.Len() > 0 {
			logger.Debug("railpack stdout", "output", stdout.String())
		}

		if err != nil {
			if stderr.Len() > 0 {
				logger.Error("railpack build failed", "error", err, "stderr", stderr.String())
				return nil, fmt.Errorf("railpack build failed: %w\nstderr: %s", err, stderr.String())
			}
			return nil, fmt.Errorf("railpack build failed: %w", err)
		}
	}

	logger.Info("railpack build completed successfully", "image", imageName)

	// Detect exposed ports from the built image
	detectedPorts, err := portdetector.DetectPorts(imageName)
	if err != nil {
		logger.Warn("failed to detect ports from image", "image", imageName, "error", err)
		// Continue with empty ports - port detection failure shouldn't fail the build
	} else {
		logger.Info("detected ports from image", "image", imageName, "ports", detectedPorts)
	}

	// Create artifact
	artifactID := fmt.Sprintf("railpack-%s-%d", b.config.Name, time.Now().Unix())

	art := &artifact.Artifact{
		ID:           artifactID,
		Image:        b.config.Name,
		Tag:          tag,
		ExposedPorts: detectedPorts,
		Labels: map[string]string{
			"builder": "railpack",
		},
		Metadata: map[string]interface{}{
			"builder":       "railpack",
			"context":       context,
			"railpack_args": strings.Join(args, " "),
		},
		BuildTime: time.Now(),
	}

	// Add detected ports to metadata for debugging
	if len(detectedPorts) > 0 {
		art.Metadata["detected_ports"] = detectedPorts
	}

	return art, nil
}

func (b *Builder) Config() (interface{}, error) {
	return b.config, nil
}

func (b *Builder) ConfigSet(config interface{}) error {
	if config == nil {
		b.config = &BuilderConfig{}
		return nil
	}

	// Handle map[string]interface{} configuration
	if configMap, ok := config.(map[string]interface{}); ok {
		b.config = &BuilderConfig{}

		// Helper function to get string value (handles both string and *string)
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

		// Required fields
		b.config.Name = getString("name")

		// Optional fields
		b.config.Tag = getString("tag")
		b.config.Context = getString("context")

		// BuildArgs map
		if buildArgs, ok := configMap["build_args"].(map[string]interface{}); ok {
			b.config.BuildArgs = make(map[string]string)
			for k, v := range buildArgs {
				if strVal, ok := v.(string); ok {
					b.config.BuildArgs[k] = strVal
				} else if strPtr, ok := v.(*string); ok && strPtr != nil {
					b.config.BuildArgs[k] = *strPtr
				}
			}
		}

		// Env map
		if env, ok := configMap["env"].(map[string]interface{}); ok {
			b.config.Env = make(map[string]string)
			for k, v := range env {
				if strVal, ok := v.(string); ok {
					b.config.Env[k] = strVal
				} else if strPtr, ok := v.(*string); ok && strPtr != nil {
					b.config.Env[k] = *strPtr
				}
			}
		}

		// Deprecated Vault fields (backward compatibility)
		b.config.VaultAddress = getString("vault_address")
		b.config.RoleID = getString("role_id")
		b.config.SecretID = getString("secret_id")
		b.config.SecretsPath = getString("secrets_path")

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

func init() {
	plugin.Register("railpack", &plugin.Plugin{
		Builder: &Builder{config: &BuilderConfig{}},
	})
}
