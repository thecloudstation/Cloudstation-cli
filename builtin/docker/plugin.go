package docker

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
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/websocket"
)

// Builder implements Docker image building
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

	// Dockerfile path (defaults to "Dockerfile")
	Dockerfile string

	// Context is the build directory path (defaults to ".")
	Context string

	// BuildArgs are additional build arguments for Docker
	BuildArgs map[string]string

	// Env contains environment variables
	Env map[string]string
}

func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
	// Validate configuration
	if b.config == nil {
		return nil, fmt.Errorf("docker builder configuration is not set")
	}

	// Require either Name or Image to be set
	if b.config.Name == "" && b.config.Image == "" {
		return nil, fmt.Errorf("docker builder requires either 'name' or 'image' field to be set")
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

	buildContext := b.config.Context
	if buildContext == "" {
		buildContext = "."
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

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)

	logger.Info("starting Docker build", "image", fullImageName, "context", buildContext)

	// Build Docker command arguments
	// Determine the build context argument for docker command
	// If we're setting cmd.Dir, use "." as context to avoid double-pathing
	// (e.g., passing "/tmp/upload-xyz" as arg while also setting cmd.Dir to "/tmp/upload-xyz"
	// would cause docker to look for "/tmp/upload-xyz/tmp/upload-xyz")
	buildArg := buildContext
	if buildContext != "." && buildContext != "" {
		buildArg = "." // Will be relative to cmd.Dir
	}
	args := []string{"build", buildArg}
	args = append(args, "-f", dockerfile)
	args = append(args, "-t", fullImageName)

	// Add build args if provided
	for key, value := range b.config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add environment variables as build args
	for key, value := range b.config.Env {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	logger.Debug("executing docker", "args", args)

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set working directory if context is not current directory
	if buildContext != "." && buildContext != "" {
		cmd.Dir = buildContext
	}

	// Check for NATS writers in context first (preferred), then WebSocket, then buffer
	stdoutWriter, hasStdoutWriter := ctx.Value("stdoutWriter").(io.Writer)
	stderrWriter, hasStderrWriter := ctx.Value("stderrWriter").(io.Writer)
	wsClient, _ := ctx.Value("wsClient").(*websocket.Client)

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
	} else if wsClient != nil {
		// Fall back to WebSocket streaming writers
		wsStdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
		wsStderrWriter := websocket.NewStreamWriter(wsClient, "stderr")
		cmd.Stdout = wsStdoutWriter
		cmd.Stderr = wsStderrWriter

		// Execute the command
		err := cmd.Run()

		// Flush any remaining buffered output
		wsStdoutWriter.Flush()
		wsStderrWriter.Flush()

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

	// Create artifact
	artifactID := fmt.Sprintf("docker-%s-%d", imageName, time.Now().Unix())

	return &artifact.Artifact{
		ID:        artifactID,
		Image:     imageName,
		Tag:       tag,
		BuildTime: time.Now(),
		Labels: map[string]string{
			"builder": "docker",
		},
		Metadata: map[string]interface{}{
			"builder":    "docker",
			"context":    buildContext,
			"dockerfile": dockerfile,
		},
	}, nil
}

func (b *Builder) Config() (interface{}, error) {
	return b.config, nil
}

func (b *Builder) ConfigSet(config interface{}) error {
	b.config = &BuilderConfig{}
	if configMap, ok := config.(map[string]interface{}); ok {
		if name, ok := configMap["name"].(string); ok {
			b.config.Name = name
		}
		if image, ok := configMap["image"].(string); ok {
			b.config.Image = image
		}
		if tag, ok := configMap["tag"].(string); ok {
			b.config.Tag = tag
		}
		if dockerfile, ok := configMap["dockerfile"].(string); ok {
			b.config.Dockerfile = dockerfile
		}
		if context, ok := configMap["context"].(string); ok {
			b.config.Context = context
		}
		if buildArgs, ok := configMap["build_args"].(map[string]interface{}); ok {
			b.config.BuildArgs = make(map[string]string)
			for k, v := range buildArgs {
				if strVal, ok := v.(string); ok {
					b.config.BuildArgs[k] = strVal
				}
			}
		}
		if env, ok := configMap["env"].(map[string]interface{}); ok {
			b.config.Env = make(map[string]string)
			for k, v := range env {
				if strVal, ok := v.(string); ok {
					b.config.Env[k] = strVal
				}
			}
		}
	}
	return nil
}

// Registry implements Docker registry operations
type Registry struct {
	config *RegistryConfig
	logger hclog.Logger
}

type RegistryConfig struct {
	Image     string
	Tag       string
	Username  string
	Password  string
	Registry  string
	Namespace string // Image namespace prefix for ACR token-based deployments
}

func (r *Registry) Push(ctx context.Context, art *artifact.Artifact) (*artifact.RegistryRef, error) {
	// Initialize logger if not set
	if r.logger == nil {
		r.logger = hclog.Default()
	}

	// Validate inputs
	if r.config == nil {
		return nil, fmt.Errorf("registry config is nil")
	}
	if art == nil {
		return nil, fmt.Errorf("artifact is nil")
	}
	if r.config.Username == "" {
		return nil, fmt.Errorf("registry username is required")
	}
	if r.config.Password == "" {
		return nil, fmt.Errorf("registry password is required")
	}
	if art.Image == "" {
		return nil, fmt.Errorf("artifact image is required")
	}

	// Extract registry URL and image name
	var registryURL, imageName string

	// If config.Image contains a registry prefix, extract it
	if strings.Contains(r.config.Image, "/") {
		parts := strings.SplitN(r.config.Image, "/", 2)
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			// First part looks like a registry URL
			registryURL = parts[0]
			imageName = parts[1]
		} else {
			// First part is just a namespace
			imageName = r.config.Image
			registryURL = r.config.Registry
		}
	} else {
		imageName = r.config.Image
		registryURL = r.config.Registry
	}

	// If no registry URL was determined, return error
	if registryURL == "" {
		return nil, fmt.Errorf("registry URL could not be determined from config")
	}

	// Determine tag to use
	tag := r.config.Tag
	if tag == "" {
		tag = art.Tag
	}
	if tag == "" {
		tag = "latest"
	}

	// Build full image name with namespace if provided
	var fullImage string
	if r.config.Namespace != "" {
		// Use namespace from credentials (e.g., "cli/user_abc123/myapp")
		fullImage = fmt.Sprintf("%s/%s:%s", registryURL, r.config.Namespace, tag)
	} else {
		// Fallback to existing logic
		fullImage = fmt.Sprintf("%s/%s:%s", registryURL, imageName, tag)
	}

	r.logger.Info("pushing image to registry",
		"source_image", art.Image,
		"target_image", fullImage,
		"registry", registryURL)

	// Step 1: Docker login
	r.logger.Debug("authenticating with registry", "registry", registryURL)
	loginCmd := exec.CommandContext(ctx, "docker", "login", registryURL,
		"-u", r.config.Username,
		"--password-stdin")

	// Pass password via stdin to avoid it appearing in process list
	loginCmd.Stdin = strings.NewReader(r.config.Password)

	if output, err := loginCmd.CombinedOutput(); err != nil {
		r.logger.Error("docker login failed", "error", err, "output", string(output))
		return nil, fmt.Errorf("docker login failed: %w (output: %s)", err, string(output))
	}
	r.logger.Debug("docker login successful")

	// Step 2: Tag the image
	// Use the full source image name with tag from the artifact
	sourceImage := art.Image
	if art.Tag != "" && !strings.Contains(art.Image, ":") {
		sourceImage = fmt.Sprintf("%s:%s", art.Image, art.Tag)
	}
	r.logger.Debug("tagging image", "source", sourceImage, "target", fullImage)
	tagCmd := exec.CommandContext(ctx, "docker", "tag", sourceImage, fullImage)
	if output, err := tagCmd.CombinedOutput(); err != nil {
		r.logger.Error("docker tag failed", "error", err, "output", string(output))
		return nil, fmt.Errorf("docker tag failed: %w (output: %s)", err, string(output))
	}
	r.logger.Debug("docker tag successful")

	// Step 3: Push the image
	r.logger.Info("pushing image", "image", fullImage)
	pushCmd := exec.CommandContext(ctx, "docker", "push", fullImage)
	output, err := pushCmd.CombinedOutput()
	if err != nil {
		r.logger.Error("docker push failed", "error", err, "output", string(output))
		return nil, fmt.Errorf("docker push failed: %w (output: %s)", err, string(output))
	}
	r.logger.Info("docker push successful")

	// Step 4: Extract digest from push output
	digest := ""
	outputStr := string(output)
	// Docker push output typically contains a line like: "sha256:abc123..."
	for _, line := range strings.Split(outputStr, "\n") {
		if strings.Contains(line, "sha256:") {
			// Extract the digest
			if idx := strings.Index(line, "sha256:"); idx >= 0 {
				digestPart := line[idx:]
				// Take only the digest part (64 hex chars after sha256:)
				if len(digestPart) >= 71 { // "sha256:" (7) + 64 hex chars
					digest = digestPart[:71]
				} else {
					digest = strings.Fields(digestPart)[0]
				}
				break
			}
		}
	}

	return &artifact.RegistryRef{
		Registry:   registryURL,
		Repository: imageName,
		Tag:        tag,
		Digest:     digest,
		FullImage:  fullImage,
		PushedAt:   time.Now(),
	}, nil
}

func (r *Registry) Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *Registry) Config() (interface{}, error) {
	return r.config, nil
}

func (r *Registry) ConfigSet(config interface{}) error {
	r.config = &RegistryConfig{}
	if configMap, ok := config.(map[string]interface{}); ok {
		if image, ok := configMap["image"].(string); ok {
			r.config.Image = image
		}
		if tag, ok := configMap["tag"].(string); ok {
			r.config.Tag = tag
		}
		if username, ok := configMap["username"].(string); ok {
			r.config.Username = username
		}
		if password, ok := configMap["password"].(string); ok {
			r.config.Password = password
		}
		if registry, ok := configMap["registry"].(string); ok {
			r.config.Registry = registry
		}
		if namespace, ok := configMap["namespace"].(string); ok {
			r.config.Namespace = namespace
		}

		// Support nested auth block: auth { username = "...", password = "..." }
		if authConfig, ok := configMap["auth"].(map[string]interface{}); ok {
			if username, ok := authConfig["username"].(string); ok {
				r.config.Username = username
			}
			if password, ok := authConfig["password"].(string); ok {
				r.config.Password = password
			}
		}
	}

	// Fallback to environment variables if config values are not set
	if r.config.Registry == "" {
		r.config.Registry = os.Getenv("REGISTRY_URL")
	}
	if r.config.Username == "" {
		r.config.Username = os.Getenv("REGISTRY_USERNAME")
	}
	if r.config.Password == "" {
		r.config.Password = os.Getenv("REGISTRY_PASSWORD")
	}
	if r.config.Namespace == "" {
		r.config.Namespace = os.Getenv("REGISTRY_NAMESPACE")
	}

	return nil
}

// Platform implements Docker deployment (optional)
type Platform struct {
	config *PlatformConfig
}

type PlatformConfig struct {
	Network string
	Ports   []string
}

func (p *Platform) Deploy(ctx context.Context, artifact *artifact.Artifact) (*deployment.Deployment, error) {
	return &deployment.Deployment{
		ID:         "docker-deployment",
		Name:       artifact.Image,
		Platform:   "docker",
		ArtifactID: artifact.ID,
		Status: deployment.DeploymentStatus{
			State: deployment.StateRunning,
		},
		DeployedAt: time.Now(),
	}, nil
}

func (p *Platform) Destroy(ctx context.Context, deploymentID string) error {
	return nil
}

func (p *Platform) Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error) {
	return &deployment.DeploymentStatus{
		State: deployment.StateRunning,
	}, nil
}

func (p *Platform) Config() (interface{}, error) {
	return p.config, nil
}

func (p *Platform) ConfigSet(config interface{}) error {
	p.config = &PlatformConfig{}
	return nil
}

func init() {
	plugin.Register("docker", &plugin.Plugin{
		Builder:  &Builder{config: &BuilderConfig{}},
		Registry: &Registry{config: &RegistryConfig{}},
		Platform: &Platform{config: &PlatformConfig{}},
	})
}
