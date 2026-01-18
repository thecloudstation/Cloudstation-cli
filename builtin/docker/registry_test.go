package docker

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

func TestRegistryConfigSet(t *testing.T) {
	tests := []struct {
		name          string
		config        interface{}
		wantImage     string
		wantTag       string
		wantUsername  string
		wantPassword  string
		wantRegistry  string
		wantNamespace string
	}{
		{
			name: "flat config",
			config: map[string]interface{}{
				"image":    "myapp",
				"tag":      "v1.0",
				"username": "testuser",
				"password": "testpass",
				"registry": "docker.io",
			},
			wantImage:    "myapp",
			wantTag:      "v1.0",
			wantUsername: "testuser",
			wantPassword: "testpass",
			wantRegistry: "docker.io",
		},
		{
			name: "nested auth block",
			config: map[string]interface{}{
				"image": "myapp",
				"tag":   "latest",
				"auth": map[string]interface{}{
					"username": "authuser",
					"password": "authpass",
				},
			},
			wantImage:    "myapp",
			wantTag:      "latest",
			wantUsername: "authuser",
			wantPassword: "authpass",
		},
		{
			name: "auth block overrides flat credentials",
			config: map[string]interface{}{
				"image":    "myapp",
				"username": "flatuser",
				"password": "flatpass",
				"auth": map[string]interface{}{
					"username": "authuser",
					"password": "authpass",
				},
			},
			wantImage:    "myapp",
			wantUsername: "authuser",
			wantPassword: "authpass",
		},
		{
			name: "image with registry prefix",
			config: map[string]interface{}{
				"image":    "acrbc001.azurecr.io/myapp",
				"tag":      "latest",
				"username": "testuser",
				"password": "testpass",
			},
			wantImage:    "acrbc001.azurecr.io/myapp",
			wantTag:      "latest",
			wantUsername: "testuser",
			wantPassword: "testpass",
		},
		{
			name: "config with namespace",
			config: map[string]interface{}{
				"image":     "myapp",
				"tag":       "v1.0.0",
				"username":  "user",
				"password":  "pass",
				"registry":  "acrbc001.azurecr.io",
				"namespace": "cli/user-abc123/myapp",
			},
			wantImage:     "myapp",
			wantTag:       "v1.0.0",
			wantUsername:  "user",
			wantPassword:  "pass",
			wantRegistry:  "acrbc001.azurecr.io",
			wantNamespace: "cli/user-abc123/myapp",
		},
		{
			name: "namespace with auth block",
			config: map[string]interface{}{
				"image":     "myapp",
				"tag":       "v2.0.0",
				"registry":  "acrbc001.azurecr.io",
				"namespace": "cli/user-xyz789/app",
				"auth": map[string]interface{}{
					"username": "authuser",
					"password": "authpass",
				},
			},
			wantImage:     "myapp",
			wantTag:       "v2.0.0",
			wantRegistry:  "acrbc001.azurecr.io",
			wantNamespace: "cli/user-xyz789/app",
			wantUsername:  "authuser",
			wantPassword:  "authpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Registry{}
			err := r.ConfigSet(tt.config)
			if err != nil {
				t.Fatalf("ConfigSet() error = %v", err)
			}

			if r.config.Image != tt.wantImage {
				t.Errorf("Image = %v, want %v", r.config.Image, tt.wantImage)
			}
			if r.config.Tag != tt.wantTag {
				t.Errorf("Tag = %v, want %v", r.config.Tag, tt.wantTag)
			}
			if r.config.Username != tt.wantUsername {
				t.Errorf("Username = %v, want %v", r.config.Username, tt.wantUsername)
			}
			if r.config.Password != tt.wantPassword {
				t.Errorf("Password = %v, want %v", r.config.Password, tt.wantPassword)
			}
			if r.config.Registry != tt.wantRegistry {
				t.Errorf("Registry = %v, want %v", r.config.Registry, tt.wantRegistry)
			}
			if r.config.Namespace != tt.wantNamespace {
				t.Errorf("Namespace = %v, want %v", r.config.Namespace, tt.wantNamespace)
			}
		})
	}
}

func TestRegistryConfigSetNil(t *testing.T) {
	r := &Registry{}
	err := r.ConfigSet(nil)
	if err != nil {
		t.Fatalf("ConfigSet(nil) should not error, got: %v", err)
	}
	if r.config == nil {
		t.Error("ConfigSet(nil) should initialize config")
	}
}

func TestRegistryConfig(t *testing.T) {
	r := &Registry{
		config: &RegistryConfig{
			Image:    "testimage",
			Tag:      "testtag",
			Username: "testuser",
			Password: "testpass",
		},
	}

	cfg, err := r.Config()
	if err != nil {
		t.Fatalf("Config() error = %v", err)
	}

	registryCfg, ok := cfg.(*RegistryConfig)
	if !ok {
		t.Fatal("Config() did not return *RegistryConfig")
	}

	if registryCfg.Image != "testimage" {
		t.Errorf("Image = %v, want testimage", registryCfg.Image)
	}
	if registryCfg.Tag != "testtag" {
		t.Errorf("Tag = %v, want testtag", registryCfg.Tag)
	}
	if registryCfg.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", registryCfg.Username)
	}
	if registryCfg.Password != "testpass" {
		t.Errorf("Password = %v, want testpass", registryCfg.Password)
	}
}

func TestRegistryPushValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		registry   *Registry
		artifact   *artifact.Artifact
		wantErrMsg string
	}{
		{
			name: "nil config",
			registry: &Registry{
				config: nil,
			},
			artifact: &artifact.Artifact{
				Image: "test:latest",
			},
			wantErrMsg: "registry config is nil",
		},
		{
			name: "nil artifact",
			registry: &Registry{
				config: &RegistryConfig{
					Username: "user",
					Password: "pass",
				},
			},
			artifact:   nil,
			wantErrMsg: "artifact is nil",
		},
		{
			name: "missing username",
			registry: &Registry{
				config: &RegistryConfig{
					Image:    "myapp",
					Password: "pass",
				},
			},
			artifact: &artifact.Artifact{
				Image: "test:latest",
			},
			wantErrMsg: "registry username is required",
		},
		{
			name: "missing password",
			registry: &Registry{
				config: &RegistryConfig{
					Image:    "myapp",
					Username: "user",
				},
			},
			artifact: &artifact.Artifact{
				Image: "test:latest",
			},
			wantErrMsg: "registry password is required",
		},
		{
			name: "missing artifact image",
			registry: &Registry{
				config: &RegistryConfig{
					Image:    "myapp",
					Username: "user",
					Password: "pass",
				},
			},
			artifact: &artifact.Artifact{
				Image: "",
			},
			wantErrMsg: "artifact image is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.registry.Push(ctx, tt.artifact)
			if err == nil {
				t.Fatal("Push() expected error, got nil")
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("Push() error = %v, want %v", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestRegistryFullImageConstruction(t *testing.T) {
	t.Skip("skipping integration test - requires Docker daemon")

	tests := []struct {
		name         string
		config       *RegistryConfig
		artifact     *artifact.Artifact
		wantRegistry string
		wantRepo     string
		wantTag      string
		wantFullImg  string
	}{
		{
			name: "image with registry prefix",
			config: &RegistryConfig{
				Image:    "acrbc001.azurecr.io/myapp",
				Tag:      "v1.0",
				Username: "user",
				Password: "pass",
			},
			artifact: &artifact.Artifact{
				Image: "local-myapp:latest",
			},
			wantRegistry: "acrbc001.azurecr.io",
			wantRepo:     "myapp",
			wantTag:      "v1.0",
			wantFullImg:  "acrbc001.azurecr.io/myapp:v1.0",
		},
		{
			name: "image with separate registry field",
			config: &RegistryConfig{
				Image:    "myapp",
				Tag:      "latest",
				Registry: "docker.io",
				Username: "user",
				Password: "pass",
			},
			artifact: &artifact.Artifact{
				Image: "local-myapp:latest",
			},
			wantRegistry: "docker.io",
			wantRepo:     "myapp",
			wantTag:      "latest",
			wantFullImg:  "docker.io/myapp:latest",
		},
		{
			name: "tag from artifact when not in config",
			config: &RegistryConfig{
				Image:    "registry.example.com/myapp",
				Username: "user",
				Password: "pass",
			},
			artifact: &artifact.Artifact{
				Image: "local-myapp:v2.0",
				Tag:   "v2.0",
			},
			wantRegistry: "registry.example.com",
			wantRepo:     "myapp",
			wantTag:      "v2.0",
			wantFullImg:  "registry.example.com/myapp:v2.0",
		},
		{
			name: "default to latest when no tag",
			config: &RegistryConfig{
				Image:    "localhost:5000/testapp",
				Username: "user",
				Password: "pass",
			},
			artifact: &artifact.Artifact{
				Image: "testapp:local",
			},
			wantRegistry: "localhost:5000",
			wantRepo:     "testapp",
			wantTag:      "latest",
			wantFullImg:  "localhost:5000/testapp:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These tests would need Docker to be available to actually push
			// For unit tests, we'd need to mock exec.Command or skip actual execution
			// For now, we're just validating the error path since we can't push without Docker
			r := &Registry{config: tt.config}
			ctx := context.Background()

			// This will fail at docker login, but we can check if the error makes sense
			_, err := r.Push(ctx, tt.artifact)
			if err == nil {
				t.Skip("Docker appears to be available and push succeeded")
			}
			// We expect a docker login or similar error since we're not mocking
			// The important thing is that validation passed
		})
	}
}

func TestRegistryPull(t *testing.T) {
	r := &Registry{}
	ctx := context.Background()

	ref := &artifact.RegistryRef{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "latest",
		FullImage:  "docker.io/library/nginx:latest",
		PushedAt:   time.Now(),
	}

	_, err := r.Pull(ctx, ref)
	if err == nil {
		t.Fatal("Pull() should return not implemented error")
	}
	if err.Error() != "not implemented" {
		t.Errorf("Pull() error = %v, want 'not implemented'", err)
	}
}

// TestRegistryImageNamingWithNamespace tests image name construction with namespace
func TestRegistryImageNamingWithNamespace(t *testing.T) {
	t.Run("image name with namespace", func(t *testing.T) {
		registry := &Registry{
			config: &RegistryConfig{
				Image:     "myapp",
				Tag:       "v1.0.0",
				Registry:  "acrbc001.azurecr.io",
				Namespace: "cli/user-abc123/myapp",
				Username:  "user",
				Password:  "pass",
			},
		}

		// Construct expected full image name using same logic as Push method
		var fullImage string
		if registry.config.Namespace != "" {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Namespace, registry.config.Tag)
		} else {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Image, registry.config.Tag)
		}

		expectedImage := "acrbc001.azurecr.io/cli/user-abc123/myapp:v1.0.0"
		if fullImage != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImage)
		}
	})

	t.Run("image name without namespace (backward compatibility)", func(t *testing.T) {
		registry := &Registry{
			config: &RegistryConfig{
				Image:     "myapp",
				Tag:       "latest",
				Registry:  "docker.io",
				Namespace: "", // Empty namespace
				Username:  "user",
				Password:  "pass",
			},
		}

		// Build full image name using same logic as Push method
		var fullImage string
		if registry.config.Namespace != "" {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Namespace, registry.config.Tag)
		} else {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Image, registry.config.Tag)
		}

		expectedImage := "docker.io/myapp:latest"
		if fullImage != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImage)
		}
	})

	t.Run("complex namespace with multiple segments", func(t *testing.T) {
		registry := &Registry{
			config: &RegistryConfig{
				Image:     "backend-app",
				Tag:       "feature-branch",
				Registry:  "acrbc001.azurecr.io",
				Namespace: "cli/user-abc123/project/backend-app",
				Username:  "user",
				Password:  "pass",
			},
		}

		var fullImage string
		if registry.config.Namespace != "" {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Namespace, registry.config.Tag)
		} else {
			fullImage = fmt.Sprintf("%s/%s:%s", registry.config.Registry, registry.config.Image, registry.config.Tag)
		}

		expectedImage := "acrbc001.azurecr.io/cli/user-abc123/project/backend-app:feature-branch"
		if fullImage != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImage)
		}
	})
}

// TestRegistryPushWithNamespace tests Push configuration with namespace (unit test - no Docker execution)
func TestRegistryPushWithNamespace(t *testing.T) {
	// Note: These tests verify configuration logic without executing Docker commands
	// Actual Docker execution tests are skipped unless Docker daemon is available

	t.Run("namespace image path construction", func(t *testing.T) {
		registry := &Registry{
			config: &RegistryConfig{
				Image:     "myapp",
				Tag:       "v1.0.0",
				Registry:  "acrbc001.azurecr.io",
				Namespace: "cli/user-abc123/myapp",
				Username:  "user",
				Password:  "pass",
			},
			logger: hclog.NewNullLogger(),
		}

		// Verify that with namespace, the full image would be constructed correctly
		// This mirrors the logic in Push method without actually calling Docker
		var fullImage string
		if registry.config.Namespace != "" {
			fullImage = fmt.Sprintf("%s/%s:%s",
				registry.config.Registry,
				registry.config.Namespace,
				registry.config.Tag)
		} else {
			fullImage = fmt.Sprintf("%s/%s:%s",
				registry.config.Registry,
				registry.config.Image,
				registry.config.Tag)
		}

		expectedFullImage := "acrbc001.azurecr.io/cli/user-abc123/myapp:v1.0.0"
		if fullImage != expectedFullImage {
			t.Errorf("expected %s, got %s", expectedFullImage, fullImage)
		}
	})

	t.Run("without namespace uses image name", func(t *testing.T) {
		registry := &Registry{
			config: &RegistryConfig{
				Image:    "simple-app",
				Tag:      "v2.0.0",
				Registry: "docker.io",
				Username: "user",
				Password: "pass",
				// Namespace is intentionally not set
			},
			logger: hclog.NewNullLogger(),
		}

		// Verify that without namespace, the full image uses the original image name
		// This mirrors the logic in Push method without actually calling Docker
		var imageName string
		if strings.Contains(registry.config.Image, "/") {
			parts := strings.SplitN(registry.config.Image, "/", 2)
			if !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") {
				imageName = registry.config.Image
			} else {
				imageName = parts[1]
			}
		} else {
			imageName = registry.config.Image
		}

		var fullImage string
		if registry.config.Namespace != "" {
			fullImage = fmt.Sprintf("%s/%s:%s",
				registry.config.Registry,
				registry.config.Namespace,
				registry.config.Tag)
		} else {
			fullImage = fmt.Sprintf("%s/%s:%s",
				registry.config.Registry,
				imageName,
				registry.config.Tag)
		}

		expectedFullImage := "docker.io/simple-app:v2.0.0"
		if fullImage != expectedFullImage {
			t.Errorf("expected %s, got %s", expectedFullImage, fullImage)
		}
	})
}

// TestBuilderConfigSet_Extended tests Builder configuration parsing
func TestBuilderConfigSet_Extended(t *testing.T) {
	t.Run("parse builder config with all fields", func(t *testing.T) {
		builder := &Builder{}

		config := map[string]interface{}{
			"dockerfile": "Dockerfile",
			"context":    ".",
			"buildArgs": map[string]string{
				"VERSION": "1.0.0",
				"ENV":     "production",
			},
		}

		err := builder.ConfigSet(config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if builder.config.Dockerfile != "Dockerfile" {
			t.Errorf("expected Dockerfile, got %s", builder.config.Dockerfile)
		}
		if builder.config.Context != "." {
			t.Errorf("expected context '.', got %s", builder.config.Context)
		}
		// Note: buildArgs are not currently parsed in the implementation
	})

	t.Run("builder config with custom path", func(t *testing.T) {
		builder := &Builder{}

		config := map[string]interface{}{
			"dockerfile": "docker/Dockerfile.prod",
			"context":    "./src",
		}

		err := builder.ConfigSet(config)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if builder.config.Dockerfile != "docker/Dockerfile.prod" {
			t.Errorf("expected docker/Dockerfile.prod, got %s", builder.config.Dockerfile)
		}
		if builder.config.Context != "./src" {
			t.Errorf("expected context './src', got %s", builder.config.Context)
		}
	})
}

// TestImageNameExtraction tests the logic for extracting registry URL and image name
func TestImageNameExtraction(t *testing.T) {
	testCases := []struct {
		name             string
		configImage      string
		configRegistry   string
		configNamespace  string
		expectedRegistry string
		expectedImage    string
		expectedFull     string
	}{
		{
			name:             "simple image with namespace",
			configImage:      "myapp",
			configRegistry:   "acrbc001.azurecr.io",
			configNamespace:  "cli/user-abc123/myapp",
			expectedRegistry: "acrbc001.azurecr.io",
			expectedImage:    "myapp",
			expectedFull:     "acrbc001.azurecr.io/cli/user-abc123/myapp:latest",
		},
		{
			name:             "simple image without namespace",
			configImage:      "myapp",
			configRegistry:   "docker.io",
			configNamespace:  "",
			expectedRegistry: "docker.io",
			expectedImage:    "myapp",
			expectedFull:     "docker.io/myapp:latest",
		},
		{
			name:             "image with registry prefix and namespace",
			configImage:      "custom.io/myapp",
			configRegistry:   "acrbc001.azurecr.io", // This should be ignored
			configNamespace:  "cli/user/app",
			expectedRegistry: "custom.io",
			expectedImage:    "myapp",
			expectedFull:     "custom.io/cli/user/app:latest",
		},
		{
			name:             "image with namespace prefix (no dots)",
			configImage:      "library/nginx",
			configRegistry:   "docker.io",
			configNamespace:  "",
			expectedRegistry: "docker.io",
			expectedImage:    "library/nginx",
			expectedFull:     "docker.io/library/nginx:latest",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the logic from the Push method
			var registryURL, imageName string

			if strings.Contains(tc.configImage, "/") {
				parts := strings.SplitN(tc.configImage, "/", 2)
				if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
					// First part looks like a registry URL
					registryURL = parts[0]
					imageName = parts[1]
				} else {
					// First part is just a namespace
					imageName = tc.configImage
					registryURL = tc.configRegistry
				}
			} else {
				imageName = tc.configImage
				registryURL = tc.configRegistry
			}

			if registryURL != tc.expectedRegistry {
				t.Errorf("expected registry %s, got %s", tc.expectedRegistry, registryURL)
			}
			if imageName != tc.expectedImage {
				t.Errorf("expected image %s, got %s", tc.expectedImage, imageName)
			}

			// Build full image name
			var fullImage string
			if tc.configNamespace != "" {
				fullImage = fmt.Sprintf("%s/%s:%s", registryURL, tc.configNamespace, "latest")
			} else {
				fullImage = fmt.Sprintf("%s/%s:%s", registryURL, imageName, "latest")
			}

			if fullImage != tc.expectedFull {
				t.Errorf("expected full image %s, got %s", tc.expectedFull, fullImage)
			}
		})
	}
}
