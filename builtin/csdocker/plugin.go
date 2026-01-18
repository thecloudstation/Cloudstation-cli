package csdocker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/portdetector"
)

// Builder implements Docker build with Vault integration
type Builder struct {
	config *BuilderConfig
}

type BuilderConfig struct {
	// Name is the Docker image name (e.g., "myapp")
	Name string

	// Image is the full image path including registry (e.g., "myregistry.azurecr.io/myapp")
	// If set, this takes precedence over Name
	Image string

	// Tag is the Docker image tag (defaults to "latest")
	Tag string

	// Namespace is the image namespace prefix for registry push
	Namespace string

	// Dockerfile path (defaults to "Dockerfile")
	Dockerfile string

	// Context is the build directory path (defaults to ".")
	Context string

	// BuildArgs are additional build arguments for Docker
	BuildArgs map[string]string

	// Env contains environment variables (including secrets injected by secret providers)
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
	// Validate configuration
	if b.config == nil {
		return nil, fmt.Errorf("csdocker builder configuration is not set")
	}

	// Require either Name or Image to be set
	if b.config.Name == "" && b.config.Image == "" {
		return nil, fmt.Errorf("csdocker builder requires either 'name' or 'image' field to be set")
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}

	// Apply defaults
	dockerfile := b.config.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	context := b.config.Context
	if context == "" {
		context = "."
	}

	tag := b.config.Tag
	if tag == "" {
		tag = "latest"
	}

	// Determine image name (Image takes precedence over Name)
	imageName := b.config.Image
	if imageName == "" {
		imageName = b.config.Name
	}

	// Apply namespace if provided
	if b.config.Namespace != "" {
		imageName = fmt.Sprintf("%s/%s", b.config.Namespace, imageName)
	}

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)

	logger.Info("starting Docker build", "image", fullImageName, "context", context)

	// Build Docker command arguments
	// Determine the build context argument
	// If we're setting cmd.Dir, use "." as context to avoid double-pathing
	// (e.g., passing "/tmp/upload-xyz" as arg while also setting cmd.Dir to "/tmp/upload-xyz"
	// would cause docker to look for "/tmp/upload-xyz/tmp/upload-xyz")
	buildContext := context
	if context != "." && context != "" {
		buildContext = "." // Will be relative to cmd.Dir
	}
	args := []string{"build", buildContext}
	args = append(args, "-f", dockerfile)
	args = append(args, "-t", fullImageName)

	// Add build args if provided
	for key, value := range b.config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add environment variables as build args (including Vault secrets!)
	for key, value := range b.config.Env {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	logger.Debug("executing docker", "args", args)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set working directory if context is not current directory
	if context != "." && context != "" {
		cmd.Dir = context
	}

	// Check for NATS writers in context, otherwise use buffer
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
			logger.Error("docker build failed", "error", err)
			return nil, fmt.Errorf("docker build failed: %w", err)
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
			logger.Debug("docker stdout", "output", stdout.String())
		}

		if err != nil {
			if stderr.Len() > 0 {
				logger.Error("docker build failed", "error", err, "stderr", stderr.String())
				return nil, fmt.Errorf("docker build failed: %w\nstderr: %s", err, stderr.String())
			}
			return nil, fmt.Errorf("docker build failed: %w", err)
		}
	}

	logger.Info("docker build completed successfully", "image", fullImageName)

	// Detect exposed ports from the built image
	detectedPorts, err := portdetector.DetectPorts(fullImageName)
	if err != nil {
		logger.Warn("failed to detect ports from image", "image", fullImageName, "error", err)
		// Continue with empty ports - port detection failure shouldn't fail the build
	} else {
		logger.Info("detected ports from image", "image", fullImageName, "ports", detectedPorts)
	}

	// Create artifact
	artifactID := fmt.Sprintf("csdocker-%s-%d", imageName, time.Now().Unix())

	art := &artifact.Artifact{
		ID:           artifactID,
		Image:        imageName,
		Tag:          tag,
		ExposedPorts: detectedPorts,
		Labels: map[string]string{
			"builder": "csdocker",
		},
		Metadata: map[string]interface{}{
			"builder":     "csdocker",
			"context":     context,
			"dockerfile":  dockerfile,
			"docker_args": strings.Join(args, " "),
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

		// Parse image-related fields
		b.config.Name = getString("name")
		b.config.Image = getString("image")
		b.config.Tag = getString("tag")
		b.config.Namespace = getString("namespace")
		b.config.Dockerfile = getString("dockerfile")
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

		// Env map (THIS IS WHERE VAULT SECRETS ARRIVE)
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
	plugin.Register("csdocker", &plugin.Plugin{
		Builder: &Builder{config: &BuilderConfig{}},
	})
}
