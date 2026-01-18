package dispatch

import (
	"testing"
)

func TestMapDeployRepositoryToHCLParams_RegistryFields(t *testing.T) {
	tests := []struct {
		name                     string
		buildOptions             BuildOptions
		expectedRegistryUsername string
		expectedRegistryPassword string
		expectedRegistryURL      string
		expectedDisablePush      bool
	}{
		{
			name: "All registry fields mapped correctly",
			buildOptions: BuildOptions{
				Builder:          "nixpacks",
				RegistryUsername: "my-user",
				RegistryPassword: "my-secret-pass",
				RegistryURL:      "registry.example.com",
				DisablePush:      false,
			},
			expectedRegistryUsername: "my-user",
			expectedRegistryPassword: "my-secret-pass",
			expectedRegistryURL:      "registry.example.com",
			expectedDisablePush:      false,
		},
		{
			name: "DisablePush=true is mapped correctly",
			buildOptions: BuildOptions{
				Builder:          "nixpacks",
				RegistryUsername: "user",
				RegistryPassword: "pass",
				DisablePush:      true,
			},
			expectedRegistryUsername: "user",
			expectedRegistryPassword: "pass",
			expectedRegistryURL:      "",
			expectedDisablePush:      true,
		},
		{
			name: "Empty credentials mapped as empty strings",
			buildOptions: BuildOptions{
				Builder:     "nixpacks",
				DisablePush: false,
				// No credentials provided
			},
			expectedRegistryUsername: "",
			expectedRegistryPassword: "",
			expectedRegistryURL:      "",
			expectedDisablePush:      false,
		},
		{
			name:         "Default BuildOptions (zero values)",
			buildOptions: BuildOptions{
				// All zero values
			},
			expectedRegistryUsername: "",
			expectedRegistryPassword: "",
			expectedRegistryURL:      "",
			expectedDisablePush:      false, // Default is false (push enabled)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeployRepositoryParams{
				BaseDeploymentParams: BaseDeploymentParams{
					JobID:        "test-job",
					DeploymentID: "dep-123",
					ServiceID:    "svc-123",
				},
				Repository: "user/repo",
				Branch:     "main",
				Build:      tt.buildOptions,
			}

			result := mapDeployRepositoryToHCLParams(params)

			if result.RegistryUsername != tt.expectedRegistryUsername {
				t.Errorf("RegistryUsername: expected %q, got %q", tt.expectedRegistryUsername, result.RegistryUsername)
			}
			if result.RegistryPassword != tt.expectedRegistryPassword {
				t.Errorf("RegistryPassword: expected %q, got %q", tt.expectedRegistryPassword, result.RegistryPassword)
			}
			if result.RegistryURL != tt.expectedRegistryURL {
				t.Errorf("RegistryURL: expected %q, got %q", tt.expectedRegistryURL, result.RegistryURL)
			}
			if result.DisablePush != tt.expectedDisablePush {
				t.Errorf("DisablePush: expected %v, got %v", tt.expectedDisablePush, result.DisablePush)
			}
		})
	}
}

func TestMapDeployImageToHCLParams_DisablePushAlwaysFalse(t *testing.T) {
	// Image deployments should always have DisablePush=false
	// because pre-built images need to be pushed to the registry

	params := DeployImageParams{
		BaseDeploymentParams: BaseDeploymentParams{
			JobID:        "test-job",
			ImageName:    "my-image",
			ImageTag:     "v1.0.0",
			DeploymentID: "dep-123",
			ServiceID:    "svc-123",
		},
	}

	result := mapDeployImageToHCLParams(params)

	if result.DisablePush != false {
		t.Errorf("Image deployments should always have DisablePush=false, got %v", result.DisablePush)
	}
}

func TestMapDeployRepositoryToHCLParams_RegistryFieldsWithOtherFields(t *testing.T) {
	// Verify that registry fields are mapped alongside other fields correctly
	params := DeployRepositoryParams{
		BaseDeploymentParams: BaseDeploymentParams{
			JobID:                   "full-test-job",
			DeploymentID:            "dep-456",
			ServiceID:               "svc-456",
			ImageName:               "myapp",
			ImageTag:                "latest",
			PrivateRegistry:         "acr.azurecr.io",
			PrivateRegistryProvider: "acr",
		},
		Repository: "owner/repo",
		Branch:     "develop",
		Build: BuildOptions{
			Builder:          "csdocker",
			DockerfilePath:   "Dockerfile.prod",
			RegistryUsername: "deploy-user",
			RegistryPassword: "deploy-pass",
			RegistryURL:      "acr.azurecr.io",
			DisablePush:      false,
		},
	}

	result := mapDeployRepositoryToHCLParams(params)

	// Verify registry fields
	if result.RegistryUsername != "deploy-user" {
		t.Errorf("RegistryUsername not mapped correctly")
	}
	if result.RegistryPassword != "deploy-pass" {
		t.Errorf("RegistryPassword not mapped correctly")
	}
	if result.RegistryURL != "acr.azurecr.io" {
		t.Errorf("RegistryURL not mapped correctly")
	}
	if result.DisablePush != false {
		t.Errorf("DisablePush not mapped correctly")
	}

	// Verify other fields still work
	if result.JobID != "full-test-job" {
		t.Errorf("JobID not mapped correctly")
	}
	if result.PrivateRegistry != "acr.azurecr.io" {
		t.Errorf("PrivateRegistry not mapped correctly")
	}
	if result.PrivateRegistryProvider != "acr" {
		t.Errorf("PrivateRegistryProvider not mapped correctly")
	}
	if result.BuilderType != "csdocker" {
		t.Errorf("BuilderType not mapped correctly")
	}
}

func TestMapDeployImageToHCLParams_FieldsIntact(t *testing.T) {
	// Verify that other fields are still mapped correctly when DisablePush is added
	params := DeployImageParams{
		BaseDeploymentParams: BaseDeploymentParams{
			JobID:                   "image-job",
			ImageName:               "prebuilt-image",
			ImageTag:                "v2.0.0",
			DeploymentID:            "dep-789",
			ServiceID:               "svc-789",
			PrivateRegistry:         "ghcr.io",
			PrivateRegistryProvider: "ghcr",
		},
	}

	result := mapDeployImageToHCLParams(params)

	// Verify DisablePush is false
	if result.DisablePush != false {
		t.Errorf("DisablePush should be false for image deployments")
	}

	// Verify other fields
	if result.JobID != "image-job" {
		t.Errorf("JobID not mapped correctly")
	}
	if result.ImageName != "prebuilt-image" {
		t.Errorf("ImageName not mapped correctly")
	}
	if result.ImageTag != "v2.0.0" {
		t.Errorf("ImageTag not mapped correctly")
	}
	if result.PrivateRegistry != "ghcr.io" {
		t.Errorf("PrivateRegistry not mapped correctly")
	}
	if result.PrivateRegistryProvider != "ghcr" {
		t.Errorf("PrivateRegistryProvider not mapped correctly")
	}
}

func TestMapDeployImageToHCLParams_StartCommand(t *testing.T) {
	tests := []struct {
		name                 string
		buildOptions         BuildOptions
		expectedStartCommand string
	}{
		{
			name: "StartCommand mapped correctly",
			buildOptions: BuildOptions{
				StartCommand: "minio server --address 0.0.0.0:9000 /data",
			},
			expectedStartCommand: "minio server --address 0.0.0.0:9000 /data",
		},
		{
			name:                 "Empty StartCommand",
			buildOptions:         BuildOptions{},
			expectedStartCommand: "",
		},
		{
			name: "StartCommand with noop builder",
			buildOptions: BuildOptions{
				Builder:      "noop",
				StartCommand: "npm run start",
			},
			expectedStartCommand: "npm run start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeployImageParams{
				BaseDeploymentParams: BaseDeploymentParams{
					JobID:        "test-job",
					ImageName:    "my-image",
					ImageTag:     "v1.0.0",
					DeploymentID: "dep-123",
					ServiceID:    "svc-123",
				},
				Build: tt.buildOptions,
			}

			result := mapDeployImageToHCLParams(params)

			if result.StartCommand != tt.expectedStartCommand {
				t.Errorf("StartCommand: expected %q, got %q",
					tt.expectedStartCommand, result.StartCommand)
			}
		})
	}
}
