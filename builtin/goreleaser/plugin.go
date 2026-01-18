package goreleaser

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
)

// BuilderConfig holds the configuration for GoReleaser builds
type BuilderConfig struct {
	// Name is the application name (required)
	Name string

	// Path is the path to main package (defaults to ".")
	Path string

	// Version is the version to embed in the binary
	Version string

	// Targets is the list of GOOS/GOARCH targets (defaults to ["linux/amd64", "darwin/arm64"])
	Targets []string

	// LdFlags are additional linker flags to pass to go build
	LdFlags string

	// BuildArgs contains additional build arguments
	BuildArgs map[string]string

	// OutputDir is the directory to output binaries (defaults to "./dist")
	OutputDir string
}

// Builder implements GoReleaser build functionality
type Builder struct {
	config *BuilderConfig
	logger hclog.Logger
}

// Build executes the GoReleaser build process
func (b *Builder) Build(ctx context.Context) (*artifact.Artifact, error) {
	// Validate configuration
	if b.config == nil {
		return nil, fmt.Errorf("goreleaser builder configuration is not set")
	}

	if b.config.Name == "" {
		return nil, fmt.Errorf("goreleaser builder requires 'name' field to be set")
	}

	// Get logger from context
	logger := hclog.FromContext(ctx)
	if logger == nil {
		logger = hclog.Default()
	}
	b.logger = logger.Named("goreleaser")

	// Apply defaults
	if b.config.Path == "" {
		b.config.Path = "."
	}
	if b.config.OutputDir == "" {
		b.config.OutputDir = "./dist"
	}
	if len(b.config.Targets) == 0 {
		b.config.Targets = []string{"linux/amd64", "darwin/arm64"}
	}

	b.logger.Info("starting Go build",
		"name", b.config.Name,
		"path", b.config.Path,
		"version", b.config.Version,
		"targets", b.config.Targets,
		"output_dir", b.config.OutputDir)

	// Create output directory
	if err := os.MkdirAll(b.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Collect built binary paths
	binaryPaths := []string{}

	// Build for each target
	for _, target := range b.config.Targets {
		// Parse target platform
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid target format '%s': expected GOOS/GOARCH", target)
		}
		goos := parts[0]
		goarch := parts[1]

		// Construct binary name
		binaryName := fmt.Sprintf("%s-%s-%s", b.config.Name, goos, goarch)
		if goos == "windows" {
			binaryName += ".exe"
		}
		outputPath := filepath.Join(b.config.OutputDir, binaryName)

		b.logger.Info("building binary",
			"target", target,
			"output", outputPath)

		// Build ldflags
		ldflags := "-s -w"
		if b.config.Version != "" {
			ldflags += fmt.Sprintf(" -X main.Version=%s", b.config.Version)
			ldflags += fmt.Sprintf(" -X main.version=%s", b.config.Version) // Alternative version variable
		}
		if b.config.LdFlags != "" {
			ldflags += " " + b.config.LdFlags
		}

		// Prepare go build command
		args := []string{"build", "-ldflags", ldflags, "-o", outputPath}

		// Add any additional build arguments
		for key, value := range b.config.BuildArgs {
			args = append(args, fmt.Sprintf("-%s=%s", key, value))
		}

		// Add the package path
		args = append(args, b.config.Path)

		cmd := exec.CommandContext(ctx, "go", args...)

		// Set environment variables
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			fmt.Sprintf("GOOS=%s", goos),
			fmt.Sprintf("GOARCH=%s", goarch),
		)

		// Execute the build
		b.logger.Debug("executing go build",
			"command", "go",
			"args", args,
			"env", []string{"CGO_ENABLED=0", fmt.Sprintf("GOOS=%s", goos), fmt.Sprintf("GOARCH=%s", goarch)})

		output, err := cmd.CombinedOutput()
		if err != nil {
			b.logger.Error("build failed",
				"target", target,
				"error", err,
				"output", string(output))
			return nil, fmt.Errorf("failed to build for %s: %w\nOutput: %s", target, err, string(output))
		}

		if len(output) > 0 {
			b.logger.Debug("build output", "target", target, "output", string(output))
		}

		// Verify the binary was created
		if _, err := os.Stat(outputPath); err != nil {
			return nil, fmt.Errorf("binary was not created at %s: %w", outputPath, err)
		}

		b.logger.Info("successfully built binary",
			"target", target,
			"output", outputPath)

		// Add to binary paths
		binaryPaths = append(binaryPaths, outputPath)
	}

	// Create artifact
	artifact := &artifact.Artifact{
		ID:    fmt.Sprintf("goreleaser-%s-%d", b.config.Name, time.Now().Unix()),
		Image: b.config.Name,
		Tag:   b.config.Version,
		Labels: map[string]string{
			"builder": "goreleaser",
		},
		Metadata: map[string]interface{}{
			"builder":  "goreleaser",
			"binaries": binaryPaths, // GitHub registry looks for this
			"targets":  b.config.Targets,
			"version":  b.config.Version,
		},
		BuildTime: time.Now(),
	}

	b.logger.Info("build completed successfully",
		"name", b.config.Name,
		"binaries_count", len(binaryPaths),
		"artifact_id", artifact.ID)

	return artifact, nil
}

// Config returns the current configuration
func (b *Builder) Config() (interface{}, error) {
	return b.config, nil
}

// ConfigSet sets the configuration from various input types
func (b *Builder) ConfigSet(config interface{}) error {
	// Handle nil config - apply defaults
	if config == nil {
		b.config = &BuilderConfig{
			OutputDir: "./dist",
			Targets:   []string{"linux/amd64", "darwin/arm64"},
		}
		return nil
	}

	// Handle map[string]interface{} from HCL parsing
	if configMap, ok := config.(map[string]interface{}); ok {
		b.config = &BuilderConfig{}

		// Set string fields
		b.config.Name = getString(configMap, "name")
		b.config.Path = getString(configMap, "path")
		b.config.Version = getString(configMap, "version")
		b.config.OutputDir = getString(configMap, "output_dir")
		b.config.LdFlags = getString(configMap, "ldflags")

		// Handle targets
		b.config.Targets = getStringSlice(configMap, "targets")

		// Handle build_args
		b.config.BuildArgs = getStringMap(configMap, "build_args")

		// Apply defaults for empty fields
		if b.config.OutputDir == "" {
			b.config.OutputDir = "./dist"
		}
		if len(b.config.Targets) == 0 {
			b.config.Targets = []string{"linux/amd64", "darwin/arm64"}
		}

		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*BuilderConfig); ok {
		b.config = cfg
		return nil
	}

	return nil
}

// getString is a helper function to get string value from config map
func getString(configMap map[string]interface{}, key string) string {
	if val, ok := configMap[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
		// Handle pointer to string
		if strPtr, ok := val.(*string); ok && strPtr != nil {
			return *strPtr
		}
	}
	return ""
}

// getStringSlice is a helper function to get string slice from config map
func getStringSlice(configMap map[string]interface{}, key string) []string {
	if val, ok := configMap[key]; ok {
		// Handle []interface{} from HCL
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		// Handle []string directly
		if slice, ok := val.([]string); ok {
			return slice
		}
	}
	return nil
}

// getStringMap is a helper function to get string map from config map
func getStringMap(configMap map[string]interface{}, key string) map[string]string {
	if val, ok := configMap[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			result := make(map[string]string)
			for k, v := range m {
				if strVal, ok := v.(string); ok {
					result[k] = strVal
				}
			}
			return result
		}
		// Handle map[string]string directly
		if m, ok := val.(map[string]string); ok {
			return m
		}
	}
	return nil
}

func init() {
	plugin.Register("goreleaser", &plugin.Plugin{
		Builder: &Builder{
			config: &BuilderConfig{
				OutputDir: "./dist",
				Targets:   []string{"linux/amd64", "darwin/arm64"},
			},
		},
	})
}
