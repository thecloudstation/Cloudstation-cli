package hclgen

import (
	"strings"
	"testing"
)

func TestGenerateRegistryStanza_Behavior(t *testing.T) {
	tests := []struct {
		name           string
		params         DeploymentParams
		expectStanza   bool
		expectContains []string
	}{
		{
			name: "With credentials generates stanza",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "nixpacks",
				ImageName:        "registry.io/test/app",
				ImageTag:         "v1.0.0",
				DisablePush:      false,
				RegistryUsername: "user",
				RegistryPassword: "pass",
				RegistryURL:      "registry.io",
			},
			expectStanza:   true,
			expectContains: []string{"registry {", "use = \"docker\"", "image =", "tag ="},
		},
		{
			name: "Without explicit credentials still generates stanza (env var fallback)",
			params: DeploymentParams{
				JobID:       "test-job",
				BuilderType: "nixpacks",
				ImageName:   "registry.io/test/app",
				ImageTag:    "v1.0.0",
				DisablePush: false,
				// No explicit credentials - env vars will provide at runtime
			},
			expectStanza:   true, // Generates stanza, relies on REGISTRY_USERNAME/REGISTRY_PASSWORD env vars
			expectContains: []string{"registry {", "use = \"docker\"", "image =", "tag ="},
		},
		{
			name: "DisablePush=true returns empty even with credentials",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "nixpacks",
				ImageName:        "registry.io/test/app",
				ImageTag:         "v1.0.0",
				DisablePush:      true,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			},
			expectStanza:   false,
			expectContains: []string{},
		},
		{
			name: "Noop builder returns empty even with credentials",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "noop",
				ImageName:        "minio/minio",
				ImageTag:         "latest",
				DisablePush:      false,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			},
			expectStanza:   false,
			expectContains: []string{},
		},
		{
			name: "Only username provided generates stanza",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "csdocker",
				ImageName:        "test-app",
				ImageTag:         "latest",
				DisablePush:      false,
				RegistryUsername: "user",
				// No password - still generates (partial credentials)
			},
			expectStanza:   true,
			expectContains: []string{"registry {"},
		},
		{
			name: "Only password provided generates stanza",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "csdocker",
				ImageName:        "test-app",
				ImageTag:         "latest",
				DisablePush:      false,
				RegistryPassword: "pass",
				// No username - still generates (partial credentials)
			},
			expectStanza:   true,
			expectContains: []string{"registry {"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRegistryStanza(tt.params)

			if tt.expectStanza && result == "" {
				t.Errorf("Expected registry stanza to be generated, got empty string")
			}
			if !tt.expectStanza && result != "" {
				t.Errorf("Expected empty string, got: %s", result)
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected stanza to contain %q, got:\n%s", expected, result)
				}
			}

			t.Logf("Generated stanza:\n%s", result)
		})
	}
}

func TestGenerateConfig_RegistryVariableBlocks(t *testing.T) {
	tests := []struct {
		name                 string
		params               DeploymentParams
		expectVariableBlocks bool
	}{
		{
			name: "With credentials includes variable blocks",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "nixpacks",
				DisablePush:      false,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			},
			expectVariableBlocks: true,
		},
		{
			name: "Without explicit credentials still generates variable blocks (env var fallback)",
			params: DeploymentParams{
				JobID:       "test-job",
				BuilderType: "nixpacks",
				DisablePush: false,
				// No explicit credentials - env vars will provide at runtime
			},
			expectVariableBlocks: true, // Env var fallback via REGISTRY_USERNAME/REGISTRY_PASSWORD
		},
		{
			name: "DisablePush=true no variable blocks even with credentials",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "nixpacks",
				DisablePush:      true,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			},
			expectVariableBlocks: false,
		},
		{
			name: "Noop builder no variable blocks even with credentials",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "noop",
				DisablePush:      false,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			},
			expectVariableBlocks: false,
		},
		{
			name: "Only username provided includes variable blocks",
			params: DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "csdocker",
				DisablePush:      false,
				RegistryUsername: "user",
			},
			expectVariableBlocks: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GenerateConfig(tt.params)
			if err != nil {
				t.Fatalf("GenerateConfig failed: %v", err)
			}

			hasVariableBlocks := strings.Contains(config, `variable "registry_username"`) ||
				strings.Contains(config, `variable "registry_password"`)

			if tt.expectVariableBlocks && !hasVariableBlocks {
				t.Errorf("Expected variable blocks in config, got:\n%s", config)
			}
			if !tt.expectVariableBlocks && hasVariableBlocks {
				t.Errorf("Did not expect variable blocks in config, got:\n%s", config)
			}

			t.Logf("Generated config:\n%s", config)
		})
	}
}

func TestGenerateRegistryStanza_ImageNameHandling(t *testing.T) {
	tests := []struct {
		name        string
		imageName   string
		imageTag    string
		expectImage string
		expectTag   string
	}{
		{
			name:        "Full registry URL in image name",
			imageName:   "acrbc001.azurecr.io/my-app",
			imageTag:    "v1.0.0",
			expectImage: `image = "acrbc001.azurecr.io/my-app"`,
			expectTag:   `tag = "v1.0.0"`,
		},
		{
			name:        "Docker Hub image",
			imageName:   "nginx",
			imageTag:    "latest",
			expectImage: `image = "nginx"`,
			expectTag:   `tag = "latest"`,
		},
		{
			name:        "GitHub Container Registry",
			imageName:   "ghcr.io/org/repo",
			imageTag:    "sha-abc123",
			expectImage: `image = "ghcr.io/org/repo"`,
			expectTag:   `tag = "sha-abc123"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:            "test-job",
				BuilderType:      "nixpacks",
				ImageName:        tt.imageName,
				ImageTag:         tt.imageTag,
				RegistryUsername: "user",
				RegistryPassword: "pass",
			}

			result := generateRegistryStanza(params)

			if !strings.Contains(result, tt.expectImage) {
				t.Errorf("Expected %q in stanza, got:\n%s", tt.expectImage, result)
			}
			if !strings.Contains(result, tt.expectTag) {
				t.Errorf("Expected %q in stanza, got:\n%s", tt.expectTag, result)
			}
		})
	}
}

func TestGenerateRegistryStanza_CredentialVariables(t *testing.T) {
	params := DeploymentParams{
		JobID:            "test-job",
		BuilderType:      "csdocker",
		ImageName:        "test-app",
		ImageTag:         "v1.0.0",
		RegistryUsername: "deploy-user",
		RegistryPassword: "deploy-secret",
	}

	result := generateRegistryStanza(params)

	// Verify credential variable references
	expectedContents := []string{
		"username = var.registry_username",
		"password = var.registry_password",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected %q in stanza, got:\n%s", expected, result)
		}
	}
}

func TestGenerateRegistryStanza_NoopBuilderScenarios(t *testing.T) {
	t.Run("Minio image deployment", func(t *testing.T) {
		params := DeploymentParams{
			JobID:       "cst-minio-wolrmoyg",
			BuilderType: "noop",
			ImageName:   "minio/minio",
			ImageTag:    "latest",
		}

		result := generateRegistryStanza(params)
		if result != "" {
			t.Errorf("Expected empty stanza for noop builder, got:\n%s", result)
		}
	})

	t.Run("Redis image deployment", func(t *testing.T) {
		params := DeploymentParams{
			JobID:       "redis-cache",
			BuilderType: "noop",
			ImageName:   "redis",
			ImageTag:    "7-alpine",
		}

		result := generateRegistryStanza(params)
		if result != "" {
			t.Errorf("Expected empty stanza for noop builder, got:\n%s", result)
		}
	})

	t.Run("PostgreSQL image deployment", func(t *testing.T) {
		params := DeploymentParams{
			JobID:       "postgres-db",
			BuilderType: "noop",
			ImageName:   "postgres",
			ImageTag:    "16",
		}

		result := generateRegistryStanza(params)
		if result != "" {
			t.Errorf("Expected empty stanza for noop builder, got:\n%s", result)
		}
	})
}
