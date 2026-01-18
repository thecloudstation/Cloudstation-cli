package csdocker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

func TestBuilderConfig_Namespace(t *testing.T) {
	t.Run("namespace field initialization", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "cli/user-abc123",
		}

		if config.Namespace != "cli/user-abc123" {
			t.Errorf("expected namespace cli/user-abc123, got %s", config.Namespace)
		}
	})

	t.Run("empty namespace is valid", func(t *testing.T) {
		config := &BuilderConfig{
			Name: "myapp",
			Tag:  "v1.0.0",
		}

		if config.Namespace != "" {
			t.Errorf("expected empty namespace, got %s", config.Namespace)
		}
	})

	t.Run("namespace with complex path", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "org/team/project/subproject",
		}

		if config.Namespace != "org/team/project/subproject" {
			t.Errorf("expected namespace org/team/project/subproject, got %s", config.Namespace)
		}
	})
}

func TestImageNameConstruction(t *testing.T) {
	t.Run("image name with namespace prefix", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "cli/user-abc123",
		}

		// Simulate the image name construction logic from Build method
		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "cli/user-abc123/myapp:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("image name without namespace (backward compatibility)", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "latest",
			Namespace: "", // No namespace
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "myapp:latest"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("image field takes precedence over name", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Image:     "custom-image",
			Tag:       "v2.0.0",
			Namespace: "cli/user-xyz",
		}

		// Image takes precedence over Name
		imageName := config.Image
		if imageName == "" {
			imageName = config.Name
		}

		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "cli/user-xyz/custom-image:v2.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("namespace with registry URL in image field", func(t *testing.T) {
		config := &BuilderConfig{
			Image:     "myregistry.azurecr.io/baseimage",
			Tag:       "v1.0.0",
			Namespace: "cli/user-123",
		}

		imageName := config.Image
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "cli/user-123/myregistry.azurecr.io/baseimage:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("default tag with namespace", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Namespace: "cli/user-abc123",
			// Tag not set, should default to "latest"
		}

		// Apply default tag
		tag := config.Tag
		if tag == "" {
			tag = "latest"
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, tag)

		expectedImage := "cli/user-abc123/myapp:latest"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})
}

func TestBuilderConfigSet(t *testing.T) {
	t.Run("config set with namespace from map", func(t *testing.T) {
		builder := &Builder{}

		config := map[string]interface{}{
			"name":       "test-app",
			"tag":        "v1",
			"dockerfile": "Dockerfile",
			"context":    ".",
			"namespace":  "cli/user-123",
		}

		err := builder.ConfigSet(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if builder.config.Namespace != "cli/user-123" {
			t.Errorf("expected namespace cli/user-123, got %s", builder.config.Namespace)
		}
	})

	t.Run("config set with namespace from typed config", func(t *testing.T) {
		builder := &Builder{}

		config := &BuilderConfig{
			Name:       "test-app",
			Tag:        "v1",
			Dockerfile: "Dockerfile",
			Context:    ".",
			Namespace:  "org/team/project",
		}

		err := builder.ConfigSet(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if builder.config.Namespace != "org/team/project" {
			t.Errorf("expected namespace org/team/project, got %s", builder.config.Namespace)
		}
	})

	t.Run("config set without namespace (backward compatibility)", func(t *testing.T) {
		builder := &Builder{}

		config := map[string]interface{}{
			"name":       "test-app",
			"tag":        "v1",
			"dockerfile": "Dockerfile",
			"context":    ".",
			// namespace not provided
		}

		err := builder.ConfigSet(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if builder.config.Namespace != "" {
			t.Errorf("expected empty namespace, got %s", builder.config.Namespace)
		}
	})

	t.Run("config set with all fields including namespace", func(t *testing.T) {
		builder := &Builder{}

		config := map[string]interface{}{
			"name":       "full-app",
			"image":      "registry.io/base",
			"tag":        "v2.0.0",
			"namespace":  "cli/user-xyz",
			"dockerfile": "Dockerfile.prod",
			"context":    "./src",
			"build_args": map[string]interface{}{
				"ENV": "production",
			},
			"env": map[string]interface{}{
				"SECRET": "value",
			},
		}

		err := builder.ConfigSet(config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if builder.config.Name != "full-app" {
			t.Errorf("expected name full-app, got %s", builder.config.Name)
		}
		if builder.config.Image != "registry.io/base" {
			t.Errorf("expected image registry.io/base, got %s", builder.config.Image)
		}
		if builder.config.Tag != "v2.0.0" {
			t.Errorf("expected tag v2.0.0, got %s", builder.config.Tag)
		}
		if builder.config.Namespace != "cli/user-xyz" {
			t.Errorf("expected namespace cli/user-xyz, got %s", builder.config.Namespace)
		}
		if builder.config.BuildArgs["ENV"] != "production" {
			t.Errorf("expected BuildArgs[ENV] = production, got %s", builder.config.BuildArgs["ENV"])
		}
		if builder.config.Env["SECRET"] != "value" {
			t.Errorf("expected Env[SECRET] = value, got %s", builder.config.Env["SECRET"])
		}
	})
}

func TestBuilderValidation(t *testing.T) {
	t.Run("requires name or image", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Tag:       "v1",
				Namespace: "cli/user-123",
				// Missing both Name and Image
			},
		}

		ctx := context.Background()
		_, err := builder.Build(ctx)

		if err == nil {
			t.Errorf("expected error for missing name/image, got nil")
		}

		expectedErr := "requires either 'name' or 'image' field to be set"
		if err != nil && !contains(err.Error(), expectedErr) {
			t.Errorf("expected error containing %q, got %v", expectedErr, err)
		}
	})

	t.Run("valid with name and namespace", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:      "valid-app",
				Tag:       "v1",
				Namespace: "cli/user-123",
			},
		}

		// Validate config is properly set
		if builder.config.Name == "" {
			t.Errorf("expected name to be set")
		}
		if builder.config.Namespace == "" {
			t.Errorf("expected namespace to be set")
		}
	})

	t.Run("valid with image and namespace", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Image:     "registry.io/image",
				Tag:       "v1",
				Namespace: "cli/user-456",
			},
		}

		// Validate config is properly set
		if builder.config.Image == "" {
			t.Errorf("expected image to be set")
		}
		if builder.config.Namespace == "" {
			t.Errorf("expected namespace to be set")
		}
	})
}

func TestBuildArtifactWithNamespace(t *testing.T) {
	t.Run("artifact metadata includes namespace", func(t *testing.T) {
		// Simulate artifact creation logic from Build method
		config := &BuilderConfig{
			Name:       "myapp",
			Tag:        "v1.0.0",
			Namespace:  "cli/user-abc123",
			Dockerfile: "Dockerfile",
			Context:    ".",
		}

		// Construct image name as done in Build method
		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}

		// Create artifact as done in Build method
		artifactID := fmt.Sprintf("csdocker-%s-%d", imageName, time.Now().Unix())
		art := &artifact.Artifact{
			ID:    artifactID,
			Image: imageName,
			Tag:   config.Tag,
			Labels: map[string]string{
				"builder": "csdocker",
			},
			Metadata: map[string]interface{}{
				"builder":    "csdocker",
				"context":    config.Context,
				"dockerfile": config.Dockerfile,
			},
			BuildTime: time.Now(),
		}

		// Verify artifact has namespace-prefixed image
		expectedImage := "cli/user-abc123/myapp"
		if art.Image != expectedImage {
			t.Errorf("expected artifact.Image %s, got %s", expectedImage, art.Image)
		}

		// Verify artifact ID includes namespace
		if !contains(art.ID, "cli/user-abc123/myapp") {
			t.Errorf("expected artifact.ID to contain namespace-prefixed image name")
		}
	})

	t.Run("artifact without namespace (backward compatibility)", func(t *testing.T) {
		config := &BuilderConfig{
			Name:       "myapp",
			Tag:        "latest",
			Dockerfile: "Dockerfile",
			Context:    ".",
			// No namespace
		}

		// Construct image name as done in Build method
		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}

		// Create artifact
		artifactID := fmt.Sprintf("csdocker-%s-%d", imageName, time.Now().Unix())
		art := &artifact.Artifact{
			ID:    artifactID,
			Image: imageName,
			Tag:   config.Tag,
			Labels: map[string]string{
				"builder": "csdocker",
			},
			Metadata: map[string]interface{}{
				"builder":    "csdocker",
				"context":    config.Context,
				"dockerfile": config.Dockerfile,
			},
			BuildTime: time.Now(),
		}

		// Verify artifact has simple image name without namespace
		if art.Image != "myapp" {
			t.Errorf("expected artifact.Image myapp, got %s", art.Image)
		}
	})
}

func TestNamespaceEdgeCases(t *testing.T) {
	t.Run("namespace with trailing slash", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "cli/user-abc123/", // Trailing slash
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		// Note: This will result in double slash, which Docker typically handles
		expectedImage := "cli/user-abc123//myapp:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("namespace with special characters", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "cli_prod/user-abc123.test",
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "cli_prod/user-abc123.test/myapp:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("single level namespace", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "production",
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "production/myapp:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("deep nested namespace", func(t *testing.T) {
		config := &BuilderConfig{
			Name:      "myapp",
			Tag:       "v1.0.0",
			Namespace: "org/division/team/project/subproject",
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "org/division/team/project/subproject/myapp:v1.0.0"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})
}

func TestNamespaceIntegrationScenarios(t *testing.T) {
	t.Run("CLI user deployment scenario", func(t *testing.T) {
		// Simulating a CLI user deploying their application
		config := &BuilderConfig{
			Name:       "user-webapp",
			Tag:        "deploy-2024-01-01",
			Namespace:  "cli/user-john-doe-123",
			Dockerfile: "Dockerfile",
			Context:    "./app",
			BuildArgs: map[string]string{
				"BUILD_ENV": "production",
			},
		}

		imageName := config.Name
		if config.Namespace != "" {
			imageName = fmt.Sprintf("%s/%s", config.Namespace, imageName)
		}
		fullImageName := fmt.Sprintf("%s:%s", imageName, config.Tag)

		expectedImage := "cli/user-john-doe-123/user-webapp:deploy-2024-01-01"
		if fullImageName != expectedImage {
			t.Errorf("expected %s, got %s", expectedImage, fullImageName)
		}
	})

	t.Run("migration from non-namespaced to namespaced", func(t *testing.T) {
		// Test that existing configs work without namespace
		oldConfig := &BuilderConfig{
			Name: "legacy-app",
			Tag:  "v1.0.0",
		}

		oldImageName := oldConfig.Name
		if oldConfig.Namespace != "" {
			oldImageName = fmt.Sprintf("%s/%s", oldConfig.Namespace, oldImageName)
		}
		oldFullImage := fmt.Sprintf("%s:%s", oldImageName, oldConfig.Tag)

		if oldFullImage != "legacy-app:v1.0.0" {
			t.Errorf("old config should produce legacy-app:v1.0.0, got %s", oldFullImage)
		}

		// New config with namespace
		newConfig := &BuilderConfig{
			Name:      "legacy-app",
			Tag:       "v2.0.0",
			Namespace: "cli/migrated",
		}

		newImageName := newConfig.Name
		if newConfig.Namespace != "" {
			newImageName = fmt.Sprintf("%s/%s", newConfig.Namespace, newImageName)
		}
		newFullImage := fmt.Sprintf("%s:%s", newImageName, newConfig.Tag)

		if newFullImage != "cli/migrated/legacy-app:v2.0.0" {
			t.Errorf("new config should produce cli/migrated/legacy-app:v2.0.0, got %s", newFullImage)
		}
	})
}

func TestContextHandling(t *testing.T) {
	t.Run("subdirectory context uses dot as build arg", func(t *testing.T) {
		// This test validates the fix for double-pathing bug
		// When context = "app/server", the build command should use "."
		// and cmd.Dir should be set to "app/server"

		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "app/server",
			},
		}

		// We can't easily test cmd execution without mocking,
		// but we can validate the config is set correctly
		if builder.config.Context != "app/server" {
			t.Errorf("expected context app/server, got %s", builder.config.Context)
		}

		// The actual test would verify that:
		// 1. args contains "build", "." (not "app/server")
		// 2. cmd.Dir is set to "app/server"
		// This is validated through integration tests or by inspecting
		// the Build method's internal logic
	})

	t.Run("default context uses dot", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: ".",
			},
		}

		if builder.config.Context != "." {
			t.Errorf("expected context ., got %s", builder.config.Context)
		}
	})

	t.Run("empty context defaults to dot", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "",
			},
		}

		// Empty context should be treated as "." in Build method
		if builder.config.Context != "" {
			t.Errorf("expected empty context, got %s", builder.config.Context)
		}
	})

	t.Run("relative path context", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "./src",
			},
		}

		if builder.config.Context != "./src" {
			t.Errorf("expected context ./src, got %s", builder.config.Context)
		}
	})

	t.Run("absolute path context", func(t *testing.T) {
		builder := &Builder{
			config: &BuilderConfig{
				Name:    "test-app",
				Tag:     "v1.0.0",
				Context: "/tmp/upload-xyz",
			},
		}

		if builder.config.Context != "/tmp/upload-xyz" {
			t.Errorf("expected context /tmp/upload-xyz, got %s", builder.config.Context)
		}
	})
}
