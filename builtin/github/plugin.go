package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/plugin"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/websocket"
)

// Registry implements GitHub Releases registry operations
type Registry struct {
	config *RegistryConfig
	logger hclog.Logger
}

// RegistryConfig contains configuration for GitHub Releases
type RegistryConfig struct {
	// Required
	Repository string // format: "owner/repo"
	Token      string // GitHub token (can use env())
	TagName    string // release tag (e.g., "v1.0.0")

	// Optional
	ReleaseName        string   // defaults to TagName
	ReleaseNotes       string   // markdown content
	Draft              bool     // create as draft
	Prerelease         bool     // mark as prerelease
	GenerateNotes      bool     // auto-generate from commits
	Checksums          bool     // generate SHA256 checksums
	CreateRelease      bool     // create if doesn't exist (default: true)
	TargetCommit       string   // target commitish
	DiscussionCategory string   // discussion category name
	Assets             []string // explicit asset paths (overrides artifact metadata)
}

// Push uploads artifacts to GitHub Releases
func (r *Registry) Push(ctx context.Context, art *artifact.Artifact) (*artifact.RegistryRef, error) {
	// Initialize logger if not set
	if r.logger == nil {
		r.logger = hclog.FromContext(ctx)
		if r.logger == nil {
			r.logger = hclog.Default()
		}
	}

	// Validate config
	if r.config == nil {
		return nil, fmt.Errorf("registry config is nil")
	}
	if r.config.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}
	if r.config.Token == "" {
		return nil, fmt.Errorf("token is required")
	}
	if r.config.TagName == "" {
		return nil, fmt.Errorf("tag_name is required")
	}

	// Extract assets - config.Assets takes priority, then artifact metadata
	var assets []string
	if len(r.config.Assets) > 0 {
		assets = r.config.Assets
	} else if art != nil && art.Metadata != nil {
		if binaries, ok := art.Metadata["binaries"].([]string); ok {
			assets = binaries
		} else if binaries, ok := art.Metadata["release_assets"].([]interface{}); ok {
			for _, b := range binaries {
				if s, ok := b.(string); ok {
					assets = append(assets, s)
				}
			}
		} else if binaryPath, ok := art.Metadata["binary_path"].(string); ok {
			assets = []string{binaryPath}
		}
	}

	if len(assets) == 0 {
		return nil, fmt.Errorf("no assets found (set assets in config or provide binaries/release_assets/binary_path in artifact metadata)")
	}

	r.logger.Info("pushing to github releases",
		"repository", r.config.Repository,
		"tag", r.config.TagName,
		"assets", len(assets))

	// Get WebSocket client from context
	wsClient, _ := ctx.Value("wsClient").(*websocket.Client)

	// Check if release exists
	checkCmd := exec.Command("gh", "release", "view", r.config.TagName, "--repo", r.config.Repository)
	checkCmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)

	releaseExists := false
	if wsClient != nil {
		// Stream output to WebSocket
		stdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
		stderrWriter := websocket.NewStreamWriter(wsClient, "stderr")
		checkCmd.Stdout = stdoutWriter
		checkCmd.Stderr = stderrWriter

		if err := checkCmd.Run(); err == nil {
			releaseExists = true
		}

		stdoutWriter.Flush()
		stderrWriter.Flush()
	} else {
		// Run without streaming
		if output, err := checkCmd.CombinedOutput(); err == nil {
			releaseExists = true
		} else {
			r.logger.Debug("release check output", "output", string(output))
		}
	}

	// Create release if needed
	if !releaseExists && r.config.CreateRelease {
		r.logger.Info("creating release", "tag", r.config.TagName)

		releaseName := r.config.ReleaseName
		if releaseName == "" {
			releaseName = r.config.TagName
		}

		createArgs := []string{"release", "create", r.config.TagName,
			"--repo", r.config.Repository,
			"--title", releaseName}

		if r.config.Draft {
			createArgs = append(createArgs, "--draft")
		}
		if r.config.Prerelease {
			createArgs = append(createArgs, "--prerelease")
		}
		if r.config.GenerateNotes {
			createArgs = append(createArgs, "--generate-notes")
		}
		if r.config.ReleaseNotes != "" {
			createArgs = append(createArgs, "--notes", r.config.ReleaseNotes)
		}
		if r.config.TargetCommit != "" {
			createArgs = append(createArgs, "--target", r.config.TargetCommit)
		}
		if r.config.DiscussionCategory != "" {
			createArgs = append(createArgs, "--discussion-category", r.config.DiscussionCategory)
		}

		createCmd := exec.Command("gh", createArgs...)
		createCmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)

		if wsClient != nil {
			// Stream output to WebSocket
			stdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
			stderrWriter := websocket.NewStreamWriter(wsClient, "stderr")
			createCmd.Stdout = stdoutWriter
			createCmd.Stderr = stderrWriter

			if err := createCmd.Run(); err != nil {
				stdoutWriter.Flush()
				stderrWriter.Flush()
				return nil, fmt.Errorf("failed to create release: %w", err)
			}

			stdoutWriter.Flush()
			stderrWriter.Flush()
		} else {
			// Run without streaming
			if output, err := createCmd.CombinedOutput(); err != nil {
				r.logger.Error("failed to create release", "output", string(output))
				return nil, fmt.Errorf("failed to create release: %w", err)
			}
		}
	} else if !releaseExists {
		return nil, fmt.Errorf("release %s does not exist and create_release is false", r.config.TagName)
	}

	// Generate checksums if enabled
	if r.config.Checksums {
		r.logger.Info("generating checksums")
		checksumsPath := filepath.Join(filepath.Dir(assets[0]), "checksums.txt")

		checksumFile, err := os.Create(checksumsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create checksums file: %w", err)
		}
		defer checksumFile.Close()

		for _, asset := range assets {
			hash, err := computeSHA256(asset)
			if err != nil {
				return nil, fmt.Errorf("failed to compute checksum for %s: %w", asset, err)
			}

			// Write in format: SHA256_HASH  filename
			_, err = fmt.Fprintf(checksumFile, "%s  %s\n", hash, filepath.Base(asset))
			if err != nil {
				return nil, fmt.Errorf("failed to write checksum: %w", err)
			}
		}

		// Add checksums file to assets
		assets = append(assets, checksumsPath)
	}

	// Upload each asset
	for _, asset := range assets {
		r.logger.Info("uploading asset", "asset", filepath.Base(asset))

		uploadCmd := exec.Command("gh", "release", "upload", r.config.TagName, asset,
			"--repo", r.config.Repository, "--clobber")
		uploadCmd.Env = append(os.Environ(), "GH_TOKEN="+r.config.Token)

		if wsClient != nil {
			// Stream output to WebSocket
			stdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
			stderrWriter := websocket.NewStreamWriter(wsClient, "stderr")
			uploadCmd.Stdout = stdoutWriter
			uploadCmd.Stderr = stderrWriter

			if err := uploadCmd.Run(); err != nil {
				stdoutWriter.Flush()
				stderrWriter.Flush()
				return nil, fmt.Errorf("failed to upload asset %s: %w", asset, err)
			}

			stdoutWriter.Flush()
			stderrWriter.Flush()
		} else {
			// Run without streaming
			if output, err := uploadCmd.CombinedOutput(); err != nil {
				r.logger.Error("failed to upload asset", "asset", asset, "output", string(output))
				return nil, fmt.Errorf("failed to upload asset %s: %w", asset, err)
			}
		}
	}

	releaseURL := fmt.Sprintf("https://github.com/%s/releases/tag/%s", r.config.Repository, r.config.TagName)
	r.logger.Info("successfully pushed to github releases", "url", releaseURL)

	return &artifact.RegistryRef{
		Registry:   "github.com",
		Repository: r.config.Repository,
		Tag:        r.config.TagName,
		FullImage:  releaseURL,
		PushedAt:   time.Now(),
	}, nil
}

// Pull is not implemented for GitHub Releases
func (r *Registry) Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error) {
	return nil, fmt.Errorf("github releases pull not implemented")
}

// Config returns the current configuration
func (r *Registry) Config() (interface{}, error) {
	return r.config, nil
}

// ConfigSet sets the configuration for the registry
func (r *Registry) ConfigSet(config interface{}) error {
	// Handle nil config
	if config == nil {
		r.config = &RegistryConfig{CreateRelease: true}
		return nil
	}

	// Handle map[string]interface{} from HCL parsing
	if configMap, ok := config.(map[string]interface{}); ok {
		r.config = &RegistryConfig{CreateRelease: true}

		// Required fields
		r.config.Repository = getString(configMap, "repository")
		r.config.Token = getString(configMap, "token")
		r.config.TagName = getString(configMap, "tag_name")

		// Optional fields
		r.config.ReleaseName = getString(configMap, "release_name")
		r.config.ReleaseNotes = getString(configMap, "release_notes")
		r.config.Draft = getBool(configMap, "draft", false)
		r.config.Prerelease = getBool(configMap, "prerelease", false)
		r.config.GenerateNotes = getBool(configMap, "generate_notes", false)
		r.config.Checksums = getBool(configMap, "checksums", false)
		r.config.CreateRelease = getBool(configMap, "create_release", true)
		r.config.TargetCommit = getString(configMap, "target_commit")
		r.config.DiscussionCategory = getString(configMap, "discussion_category")
		r.config.Assets = getStringSlice(configMap, "assets")

		// Support nested auth block: auth { token = "..." }
		if authConfig, ok := configMap["auth"].(map[string]interface{}); ok {
			if token := getString(authConfig, "token"); token != "" {
				r.config.Token = token
			}
		}

		return nil
	}

	// Handle typed configuration
	if cfg, ok := config.(*RegistryConfig); ok {
		r.config = cfg
		if r.config.CreateRelease == false {
			// Preserve explicit false, but default unset to true
		} else {
			r.config.CreateRelease = true
		}
		return nil
	}

	// Default to empty config with CreateRelease true
	r.config = &RegistryConfig{CreateRelease: true}
	return nil
}

// getString handles both string and *string types from config map
func getString(configMap map[string]interface{}, key string) string {
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

// getBool handles both bool and *bool types from config map
func getBool(configMap map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := configMap[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
		if boolPtr, ok := val.(*bool); ok && boolPtr != nil {
			return *boolPtr
		}
	}
	return defaultVal
}

// getStringSlice extracts a string slice from config map
func getStringSlice(configMap map[string]interface{}, key string) []string {
	if val, ok := configMap[key]; ok {
		// Handle []string directly
		if strSlice, ok := val.([]string); ok {
			return strSlice
		}
		// Handle []interface{} (common from HCL parsing)
		if ifaceSlice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(ifaceSlice))
			for _, item := range ifaceSlice {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
		// Handle single string as slice of one
		if s, ok := val.(string); ok {
			return []string{s}
		}
	}
	return nil
}

// computeSHA256 computes the SHA256 hash of a file
func computeSHA256(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func init() {
	plugin.Register("github", &plugin.Plugin{
		Registry: &Registry{config: &RegistryConfig{CreateRelease: true}},
	})
}
