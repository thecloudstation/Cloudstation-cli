package hclgen

import (
	"strings"
	"testing"
)

// TestRegistryPushBehavior_Integration tests the complete flow
// from DeploymentParams to generated HCL for the registry push feature.
func TestRegistryPushBehavior_Integration(t *testing.T) {
	t.Run("With credentials: registry stanza and variables are generated", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "my-service",
			BuilderType:      "nixpacks",
			DeployType:       "nomad-pack",
			ImageName:        "acrbc001.azurecr.io/my-service",
			ImageTag:         "abc123",
			ReplicaCount:     1,
			RegistryUsername: "deploy-user",
			RegistryPassword: "deploy-secret",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify registry stanza IS generated when credentials provided
		if !strings.Contains(config, "registry {") {
			t.Errorf("Expected registry stanza with credentials, got:\n%s", config)
		}

		// Verify it uses docker plugin
		if !strings.Contains(config, `use = "docker"`) {
			t.Errorf("Expected docker registry plugin, got:\n%s", config)
		}

		// Verify variable blocks are generated
		if !strings.Contains(config, `variable "registry_username"`) {
			t.Errorf("Expected registry_username variable, got:\n%s", config)
		}

		t.Logf("Generated config with credentials:\n%s", config)
	})

	t.Run("Without explicit credentials: registry stanza still generated (env var fallback)", func(t *testing.T) {
		params := DeploymentParams{
			JobID:       "my-service",
			BuilderType: "nixpacks",
			DeployType:  "nomad-pack",
			ImageName:   "my-service",
			ImageTag:    "latest",
			DisablePush: false,
			// No explicit credentials - registry stanza should still be generated
			// because REGISTRY_USERNAME/REGISTRY_PASSWORD env vars can provide creds at runtime
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify registry stanza IS generated (relies on env var fallback)
		if !strings.Contains(config, "registry {") {
			t.Errorf("Expected registry stanza (env var fallback), got:\n%s", config)
		}

		// Verify variable blocks with env fallback
		if !strings.Contains(config, `variable "registry_username"`) {
			t.Errorf("Expected registry_username variable with env fallback, got:\n%s", config)
		}
		if !strings.Contains(config, `env = ["REGISTRY_USERNAME"]`) {
			t.Errorf("Expected REGISTRY_USERNAME env fallback, got:\n%s", config)
		}

		t.Logf("Generated config with env var fallback:\n%s", config)
	})

	t.Run("DisablePush=true: no registry stanza even with credentials", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "my-service",
			BuilderType:      "nixpacks",
			DeployType:       "nomad-pack",
			ImageName:        "my-service",
			ImageTag:         "latest",
			DisablePush:      true, // Explicitly disable
			RegistryUsername: "user",
			RegistryPassword: "pass",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify registry stanza is NOT generated
		if strings.Contains(config, "registry {") {
			t.Errorf("Expected NO registry stanza when DisablePush=true, got:\n%s", config)
		}

		t.Logf("Generated config with DisablePush=true:\n%s", config)
	})

	t.Run("Noop builder (image deployment): no registry stanza", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "minio-service",
			BuilderType:      "noop",
			DeployType:       "nomad-pack",
			ImageName:        "minio/minio",
			ImageTag:         "latest",
			RegistryUsername: "user", // Even with credentials
			RegistryPassword: "pass",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify registry stanza is NOT generated for noop builder
		if strings.Contains(config, "registry {") {
			t.Errorf("Expected NO registry stanza for noop builder (image deployment), got:\n%s", config)
		}

		// Verify no variable blocks for noop
		if strings.Contains(config, `variable "registry_username"`) {
			t.Errorf("Expected NO registry variables for noop builder, got:\n%s", config)
		}

		t.Logf("Generated config for noop builder:\n%s", config)
	})
}

// TestRegistryStanzaContent_Integration tests the content of the registry stanza
func TestRegistryStanzaContent_Integration(t *testing.T) {
	t.Run("Registry stanza contains correct image and tag", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "test-service",
			BuilderType:      "csdocker",
			DeployType:       "nomad-pack",
			ImageName:        "acrbc001.azurecr.io/test-service",
			ImageTag:         "v1.0.0",
			RegistryUsername: "deploy-user",
			RegistryPassword: "deploy-secret",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// Verify registry stanza exists
		if !strings.Contains(config, "registry {") {
			t.Errorf("Expected registry stanza, got:\n%s", config)
		}

		// Verify image and tag (separate fields in stanza)
		if !strings.Contains(config, `image = "acrbc001.azurecr.io/test-service"`) {
			t.Errorf("Expected image field in registry stanza, got:\n%s", config)
		}
		if !strings.Contains(config, `tag = "v1.0.0"`) {
			t.Errorf("Expected tag field in registry stanza, got:\n%s", config)
		}

		t.Logf("Generated config:\n%s", config)
	})
}

// TestRealWorldScenarios_Integration tests realistic deployment configurations
func TestRealWorldScenarios_Integration(t *testing.T) {
	t.Run("Azure ACR deployment with credentials", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "csi-myapp-production",
			BuilderType:      "nixpacks",
			DeployType:       "nomad-pack",
			ImageName:        "acrbc001.azurecr.io/csi-myapp-production",
			ImageTag:         "1a2b3c4d",
			ReplicaCount:     3,
			DisablePush:      false,
			RegistryUsername: "acr-user",
			RegistryPassword: "acr-token",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		if !strings.Contains(config, "registry {") {
			t.Errorf("Expected registry stanza for ACR deployment with credentials")
		}
		if !strings.Contains(config, "acrbc001.azurecr.io") {
			t.Errorf("Expected ACR URL in image")
		}
	})

	t.Run("GitHub Container Registry deployment", func(t *testing.T) {
		params := DeploymentParams{
			JobID:            "ghcr-app",
			BuilderType:      "csdocker",
			DeployType:       "nomad-pack",
			ImageName:        "ghcr.io/myorg/myapp",
			ImageTag:         "v2.0.0",
			DisablePush:      false,
			RegistryUsername: "github-deploy-user",
			RegistryPassword: "ghp_token123",
			RegistryURL:      "ghcr.io",
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		if !strings.Contains(config, "registry {") {
			t.Errorf("Expected registry stanza for GHCR deployment")
		}
		if !strings.Contains(config, `variable "registry_username"`) {
			t.Errorf("Expected credential variables for GHCR deployment")
		}
	})

	t.Run("Pre-built image deployment (minio)", func(t *testing.T) {
		// This is the exact scenario from the regression bug
		params := DeploymentParams{
			JobID:        "cst-minio-wolrmoyg",
			BuilderType:  "noop", // Image deployment
			DeployType:   "nomad-pack",
			ImageName:    "minio/minio",
			ImageTag:     "latest",
			DisablePush:  false,
			ReplicaCount: 1,
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		// CRITICAL: No registry stanza for noop/image deployments
		if strings.Contains(config, "registry {") {
			t.Errorf("Expected NO registry stanza for pre-built image deployment, got:\n%s", config)
		}

		// No variable blocks
		if strings.Contains(config, "variable \"") {
			t.Errorf("Expected NO variable blocks for pre-built image deployment, got:\n%s", config)
		}

		t.Logf("Minio deployment config (no registry):\n%s", config)
	})

	t.Run("Local development with disabled push", func(t *testing.T) {
		params := DeploymentParams{
			JobID:       "dev-service",
			BuilderType: "nixpacks",
			DeployType:  "nomad-pack",
			ImageName:   "dev-service",
			ImageTag:    "dev",
			DisablePush: true, // Don't push for local dev
		}

		config, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}

		if strings.Contains(config, "registry {") {
			t.Errorf("Expected NO registry stanza for local dev with DisablePush=true")
		}
	})
}

// TestBackwardsCompatibility_PushToRegistry tests that the old PushToRegistry
// field still exists (deprecated but compiles).
func TestBackwardsCompatibility_PushToRegistry(t *testing.T) {
	t.Run("PushToRegistry field still exists and compiles", func(t *testing.T) {
		params := DeploymentParams{
			JobID:          "test-job",
			PushToRegistry: true, // Deprecated field, but should still compile
			DisablePush:    false,
		}

		// Should compile and generate without error
		_, err := GenerateConfig(params)
		if err != nil {
			t.Fatalf("GenerateConfig failed: %v", err)
		}
	})
}
