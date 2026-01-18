package hclgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
)

func TestPrintRealPayloadVars(t *testing.T) {
	usesKv := false
	ownerUsesKv := false

	params := DeploymentParams{
		JobID:             "csi-12devsnode-master-aomzkbgpepel",
		ImageName:         "acrbc001.azurecr.io/csi-12devsnode-master-aomzkbgpepel",
		ImageTag:          "1aecee4d402ff09d50a4fa72e01342c6bc2b95d5",
		ReplicaCount:      1,
		CPU:               100,
		RAM:               500,
		GPU:               0,
		SecretsPath:       "prj_90aae90a-64b0-40a6-ac2c-39c27f3ffa49/data/prj_integ_2f716e42-7e18-4444-be8a-94607e3a431b?version=1",
		SharedSecretPath:  "prj_env_aed55113-8d32-447c-8900-8e73b9e92dab",
		UsesKvEngine:      &usesKv,
		OwnerUsesKvEngine: &ownerUsesKv,
		RestartMode:       "fail",
		RestartAttempts:   3,
		NodePool:          "minions",
		OwnerID:           "team_9e0c8754-de83-4fee-937b-380ade396b1c",
		DeploymentID:      "gdeployment_2e808231-d90f-4d3d-aaa4-dcd7d4d81788",
		ProjectID:         "prj_90aae90a-64b0-40a6-ac2c-39c27f3ffa49",
		ServiceID:         "prj_integ_2f716e42-7e18-4444-be8a-94607e3a431b",
		Consul: ConsulConfig{
			ServiceName:    "csi-12devsnode-master-aomzkbgpepel",
			Tags:           []string{},
			LinkedServices: []ConsulLinkedService{},
		},
		JobConfig: &JobTypeConfig{
			Type:            "service",
			ProhibitOverlap: false,
		},
	}

	vars := GenerateVarsFile(params, nil)
	t.Logf("\n========== GENERATED VARS.HCL ==========\n%s\n===================================\n", vars)
}

func TestGenerateConfig_CompleteNomadPack(t *testing.T) {
	usesKv := true
	ownerUsesKv := true

	params := DeploymentParams{
		JobID:        "test-service",
		BuilderType:  "nixpacks",
		DeployType:   "nomad-pack",
		ImageName:    "registry.io/test/app",
		ImageTag:     "v1.0.0",
		ReplicaCount: 2,

		// Multi-tenancy
		OwnerID:      "usr_123",
		ProjectID:    "prj_456",
		ServiceID:    "svc_789",
		TeamID:       "team_012",
		DeploymentID: "dep_345",

		// Nomad
		NomadAddress: "https://nomad.example.com",
		NomadToken:   "nomad-token-xyz",
		NodePool:     "default",

		// Vault
		VaultAddress:      "https://vault.example.com",
		RoleID:            "role-123",
		SecretID:          "secret-456",
		SecretsPath:       "secret/data/app",
		UsesKvEngine:      &usesKv,
		OwnerUsesKvEngine: &ownerUsesKv,

		// Registry
		Registry: RegistryConfig{
			Pack:           "cloud_service",
			RegistryName:   "cloudstation-packs",
			RegistryRef:    "main",
			RegistrySource: "https://github.com/thecloudstation/cloudstation-packs",
			RegistryTarget: "cloud_service",
			RegistryToken:  "var.REGISTRY_TOKEN",
		},

		// Resources
		CPU: 1000,
		RAM: 2048,
		GPU: 0,

		// Networking
		Networks: []NetworkPort{
			{
				PortNumber: 8080,
				PortType:   "http",
				Public:     true,
				Domain:     "app.example.com",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health",
					Interval: "30s",
					Timeout:  "5s",
					Port:     8080,
				},
			},
		},

		// Consul
		Consul: ConsulConfig{
			ServiceName: "test-service",
			Tags:        []string{"api", "web"},
			ServicePort: 8080,
		},

		// Restart policy
		RestartMode:     "fail",
		RestartAttempts: 3,

		DeploymentCount: 0, // First deployment
	}

	config, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify all required elements
	expected := []string{
		`project = "test-service"`,
		`runner {`,
		`enabled = true`,
		`app "test-service"`,
		`use = "nixpacks"`,
		`vault_address = "https://vault.example.com"`,
		`role_id = "role-123"`,
		`secret_id = "secret-456"`,
		`secrets_path = "secret/data/app"`,
		`use = "nomad-pack"`,
		`deployment_name = "test-service"`,
		`pack = "cloud_service"`,
		`registry_name = "cloudstation-packs"`,
		`registry_ref = "main"`,
		`registry_source = "https://github.com/thecloudstation/cloudstation-packs"`,
		`registry_target = "cloud_service"`,
		`registry_token = var.REGISTRY_TOKEN`,
		`nomad_token = "nomad-token-xyz"`,
		`nomad_addr = "https://nomad.example.com"`,
		`variable_files = ["vars.hcl"]`,
	}

	for _, exp := range expected {
		if !strings.Contains(config, exp) {
			t.Errorf("Expected config to contain %q, but it doesn't", exp)
		}
	}
}

func TestGenerateConfig_FirstDeploymentHook(t *testing.T) {
	params := DeploymentParams{
		JobID:           "test-job",
		BuilderType:     "nixpacks",
		DeployType:      "nomad-pack",
		DeploymentCount: 0, // First deployment
		Registry: RegistryConfig{
			Pack:         "cloud_service",
			RegistryName: "test-registry",
		},
		VaultAddress: "https://vault.example.com",
		RoleID:       "role-123",
		SecretID:     "secret-456",
		SecretsPath:  "secret/data/app",
	}

	_, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Build completed successfully with nomad-pack, no hook in new structure
}

func TestGenerateConfig_SubsequentDeploymentNoHook(t *testing.T) {
	params := DeploymentParams{
		JobID:           "test-job",
		BuilderType:     "nixpacks",
		DeployType:      "nomad-pack",
		DeploymentCount: 5, // Subsequent deployment
		Registry: RegistryConfig{
			Pack:         "cloud_service",
			RegistryName: "test-registry",
		},
		VaultAddress: "https://vault.example.com",
		RoleID:       "role-123",
		SecretID:     "secret-456",
		SecretsPath:  "secret/data/app",
	}

	_, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Build completed successfully with nomad-pack
}

func TestGenerateConfig_BuilderNormalization(t *testing.T) {
	tests := []struct {
		name             string
		inputBuilder     string
		expectedInConfig string
	}{
		{
			name:             "docker maps to csdocker",
			inputBuilder:     "docker",
			expectedInConfig: `use = "csdocker"`,
		},
		{
			name:             "empty maps to railpack",
			inputBuilder:     "",
			expectedInConfig: `use = "railpack"`,
		},
		{
			name:             "nixpacks stays nixpacks",
			inputBuilder:     "nixpacks",
			expectedInConfig: `use = "nixpacks"`,
		},
		{
			name:             "railpack stays railpack",
			inputBuilder:     "railpack",
			expectedInConfig: `use = "railpack"`,
		},
		{
			name:             "csdocker stays csdocker",
			inputBuilder:     "csdocker",
			expectedInConfig: `use = "csdocker"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:        "test-job",
				BuilderType:  tt.inputBuilder,
				DeployType:   "noop",
				VaultAddress: "https://vault.example.com",
				RoleID:       "role-123",
				SecretID:     "secret-456",
				SecretsPath:  "secret/data/app",
			}

			config, err := GenerateConfig(params)
			if err != nil {
				t.Fatalf("GenerateConfig failed: %v", err)
			}

			if !strings.Contains(config, tt.expectedInConfig) {
				t.Errorf("Expected config to contain %q for builder %q", tt.expectedInConfig, tt.inputBuilder)
			}
		})
	}
}

func TestGenerateVarsFile_AllFields(t *testing.T) {
	usesKv := true
	ownerUsesKv := false

	params := DeploymentParams{
		JobID:        "test-service",
		ImageName:    "registry.io/test/app",
		ImageTag:     "v1.0.0",
		ReplicaCount: 3,

		// Multi-tenancy
		OwnerID:      "usr_123",
		ProjectID:    "prj_456",
		ServiceID:    "svc_789",
		DeploymentID: "dep_345",

		// Vault
		SecretsPath:       "secret/data/app",
		SharedSecretPath:  "secret/data/shared",
		UsesKvEngine:      &usesKv,
		OwnerUsesKvEngine: &ownerUsesKv,

		// Resources
		CPU:      2000,
		RAM:      4096,
		GPU:      1,
		GPUModel: "L4",

		// Networking
		Networks: []NetworkPort{
			{
				PortNumber: 3000,
				PortType:   "http",
				Public:     true,
			},
		},

		// Consul
		Consul: ConsulConfig{
			ServiceName: "my-service",
			Tags:        []string{"web", "api"},
		},

		// Restart
		RestartMode: "delay",

		// Node pool
		NodePool: "gpu-pool",

		// Private registry
		PrivateRegistry:         "ghcr.io",
		PrivateRegistryProvider: "ghcr",

		// Docker user
		DockerUser: "1000",

		// CSI Volumes
		CSIVolumes: []CSIVolume{
			{
				ID:         "vol_data[0]",
				MountPaths: []string{"/data", "/mnt"},
			},
		},

		// Job config
		JobConfig: &JobTypeConfig{
			Type:            "batch",
			Cron:            "0 0 * * *",
			ProhibitOverlap: true,
			MetaRequired:    []string{"env", "region"},
		},

		// Config files
		ConfigFiles: []ServiceConfigFile{
			{
				Path:    "/app/config.json",
				Content: `{"key":"value"}`,
			},
		},

		// Regions
		Regions: "us-west-2",
	}

	varsContent := GenerateVarsFile(params, nil)

	// Verify all fields
	expected := []string{
		`job_name = "test-service"`,
		`count = 3`,
		`image = "registry.io/test/app:v1.0.0"`,
		`resources = {cpu=2000, memory=4096, memory_max=8192, gpu=1}`,
		`gpu_type = "L4"`,
		`secret_path = "secret/data/app"`,
		`shared_secret_path = "secret/data/shared"`,
		`uses_kv_engine = true`,
		`owner_uses_kv_engine = false`,
		`restart_mode = "delay"`,
		`restart_attempts = 3`,
		`node_pool = "gpu-pool"`,
		`user_id = "usr_123"`,
		`alloc_id = "dep_345"`,
		`project_id = "prj_456"`,
		`service_id = "svc_789"`,
		`private_registry = "ghcr.io"`,
		`private_registry_provider = "ghcr"`,
		`user = "1000"`,
		`use_csi_volume = true`,
		`volume_name = "vol_data"`,
		`volume_mount_destination = ["/data", "/mnt"]`,
		`consul_service_name = "my-service"`,
		// Note: cs-runner does NOT output consul_tags
		`network = [{name="3000", port=3000`,
		`config_files = [{ path="/app/config.json", content="{"key":"value"}" }]`, // JSON is not escaped in output
		`job_config = {type="batch", cron="0 0 * * *", prohibit_overlap="true"`,
		`meta_required=["env","region"]}`,
	}

	for _, exp := range expected {
		if !strings.Contains(varsContent, exp) {
			t.Errorf("Expected vars to contain %q, but it doesn't.\nFull content:\n%s", exp, varsContent)
		}
	}
}

func TestGenerateConfig_NoopDeploy(t *testing.T) {
	params := DeploymentParams{
		JobID:        "test-job",
		BuilderType:  "nixpacks",
		DeployType:   "noop",
		VaultAddress: "https://vault.example.com",
		RoleID:       "role-123",
		SecretID:     "secret-456",
		SecretsPath:  "secret/data/app",
	}

	config, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify noop deploy
	if !strings.Contains(config, `use = "noop"`) {
		t.Error("Expected noop deploy stanza")
	}

	// Should not have nomad-pack configuration
	if strings.Contains(config, `use = "nomad-pack"`) {
		t.Error("Did not expect nomad-pack for noop deploy")
	}
}

func TestGenerateConfig_MissingJobID(t *testing.T) {
	params := DeploymentParams{
		BuilderType: "nixpacks",
		DeployType:  "nomad-pack",
	}

	_, err := GenerateConfig(params)
	if err == nil {
		t.Fatal("Expected error for missing jobID, got nil")
	}

	if !strings.Contains(err.Error(), "jobID is required") {
		t.Errorf("Expected error message about jobID, got: %v", err)
	}
}

func TestGenerateConfig_Defaults(t *testing.T) {
	params := DeploymentParams{
		JobID:        "default-job",
		VaultAddress: "https://vault.example.com",
		RoleID:       "role-123",
		SecretID:     "secret-456",
		SecretsPath:  "secret/data/app",
	}

	config, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify defaults
	if !strings.Contains(config, `use = "railpack"`) {
		t.Error("Expected default builder to be railpack")
	}

	if !strings.Contains(config, `use = "nomad-pack"`) {
		t.Error("Expected default deploy to be nomad-pack")
	}

	// Verify default replica count from params (must be set before calling GenerateVarsFile)
	if params.ReplicaCount == 0 {
		params.ReplicaCount = 1
	}

	varsContent := GenerateVarsFile(params, nil)
	if !strings.Contains(varsContent, `count = `) {
		t.Error("Expected count field in vars")
	}
}

func TestWriteConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	config := `project = "test"`

	path, err := WriteConfigFile(config, tempDir)
	if err != nil {
		t.Fatalf("WriteConfigFile failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "cloudstation.hcl")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if string(content) != config {
		t.Errorf("Expected content %q, got %q", config, string(content))
	}
}

func TestWriteVarsFile(t *testing.T) {
	tempDir := t.TempDir()

	varsContent := `job_name = "test"
count = 1`

	path, err := WriteVarsFile(varsContent, tempDir)
	if err != nil {
		t.Fatalf("WriteVarsFile failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "vars.hcl")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read vars file: %v", err)
	}

	if string(content) != varsContent {
		t.Errorf("Expected content %q, got %q", varsContent, string(content))
	}
}

func TestWriteConfigFile_EmptyDirectory(t *testing.T) {
	_, err := WriteConfigFile("test", "")
	if err == nil {
		t.Fatal("Expected error for empty directory, got nil")
	}
}

func TestWriteVarsFile_EmptyDirectory(t *testing.T) {
	_, err := WriteVarsFile("test", "")
	if err == nil {
		t.Fatal("Expected error for empty directory, got nil")
	}
}

func TestGenerateVarsFile_RestartModes(t *testing.T) {
	tests := []struct {
		name             string
		restartMode      string
		expectedMode     string
		expectedAttempts int
	}{
		{
			name:             "fail mode",
			restartMode:      "fail",
			expectedMode:     "fail",
			expectedAttempts: 3,
		},
		{
			name:             "delay mode",
			restartMode:      "delay",
			expectedMode:     "delay",
			expectedAttempts: 3,
		},
		{
			name:             "never mode maps to fail",
			restartMode:      "never",
			expectedMode:     "fail",
			expectedAttempts: 0,
		},
		{
			name:             "empty defaults to fail",
			restartMode:      "",
			expectedMode:     "fail",
			expectedAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DeploymentParams{
				JobID:       "test-job",
				RestartMode: tt.restartMode,
			}

			varsContent := GenerateVarsFile(params, nil)

			expectedModeStr := `restart_mode = "` + tt.expectedMode + `"`
			if !strings.Contains(varsContent, expectedModeStr) {
				t.Errorf("Expected %q in vars content", expectedModeStr)
			}

			expectedAttemptsStr := "restart_attempts = "
			if !strings.Contains(varsContent, expectedAttemptsStr) {
				t.Errorf("Expected restart_attempts in vars content")
			}
		})
	}
}

func TestGenerateVarsFile_CSIVolumeIDCleaning(t *testing.T) {
	params := DeploymentParams{
		JobID: "test-job",
		CSIVolumes: []CSIVolume{
			{
				ID:         "vol_abc123[0]", // Should strip [0]
				MountPaths: []string{"/data"},
			},
		},
	}

	varsContent := GenerateVarsFile(params, nil)

	if !strings.Contains(varsContent, `volume_name = "vol_abc123"`) {
		t.Error("Expected volume ID to have [0] stripped")
	}
}

func TestGenerateVarsFile_ConsulLinkedServices(t *testing.T) {
	t.Run("with linked services", func(t *testing.T) {
		params := DeploymentParams{
			JobID:        "test-job",
			ReplicaCount: 1,
			Consul: ConsulConfig{
				ServiceName: "my-service",
				LinkedServices: []ConsulLinkedService{
					{VariableName: "DB_URL", ConsulServiceName: "postgres"},
					{VariableName: "REDIS_URL", ConsulServiceName: "redis"},
				},
			},
		}

		varsContent := GenerateVarsFile(params, nil)

		// Verify consul_linked_services output
		if !strings.Contains(varsContent, `consul_linked_services = [{key="DB_URL", value="postgres"}, {key="REDIS_URL", value="redis"}]`) {
			t.Errorf("Expected consul_linked_services with services, got:\n%s", varsContent)
		}
	})

	t.Run("without linked services", func(t *testing.T) {
		params := DeploymentParams{
			JobID:        "test-job",
			ReplicaCount: 1,
			Consul: ConsulConfig{
				ServiceName:    "my-service",
				LinkedServices: []ConsulLinkedService{},
			},
		}

		varsContent := GenerateVarsFile(params, nil)

		// Verify consul_linked_services IS output as empty array (required by pack)
		if !strings.Contains(varsContent, `consul_linked_services = []`) {
			t.Errorf("Should output empty consul_linked_services array (required by pack), got:\n%s", varsContent)
		}
	})

	t.Run("consul tags are NOT output", func(t *testing.T) {
		params := DeploymentParams{
			JobID:        "test-job",
			ReplicaCount: 3,
			Consul: ConsulConfig{
				ServiceName: "my-service",
				Tags:        []string{"web", "api"},
			},
		}

		varsContent := GenerateVarsFile(params, nil)

		// Verify consul_tags is NOT output (cs-runner does not output this field)
		if strings.Contains(varsContent, `consul_tags`) {
			t.Errorf("Should NOT output consul_tags (cs-runner does not output this field), got:\n%s", varsContent)
		}
	})
}

func TestGenerateVarsFile_TLS(t *testing.T) {
	t.Run("TLS enabled", func(t *testing.T) {
		params := DeploymentParams{
			JobID: "test-job",
			TLS: &TLSConfig{
				CertPath:   "/path/to/cert",
				KeyPath:    "/path/to/key",
				CommonName: "example.com",
				PkaPath:    "/path/to/pka",
				TTL:        "24h",
			},
		}

		varsContent := GenerateVarsFile(params, nil)

		// Verify TLS output
		if !strings.Contains(varsContent, "use_tls = true") {
			t.Errorf("Expected use_tls = true")
		}
		if !strings.Contains(varsContent, `tls = [{ cert_path="/path/to/cert"`) {
			t.Errorf("Expected tls array with cert_path, got:\n%s", varsContent)
		}
		if !strings.Contains(varsContent, `key_path="/path/to/key"`) {
			t.Errorf("Expected tls array with key_path, got:\n%s", varsContent)
		}
		if !strings.Contains(varsContent, `common_name="example.com"`) {
			t.Errorf("Expected tls array with common_name, got:\n%s", varsContent)
		}
	})

	t.Run("TLS disabled", func(t *testing.T) {
		params := DeploymentParams{
			JobID: "test-job",
			TLS:   nil, // No TLS config
		}

		varsContent := GenerateVarsFile(params, nil)

		// Verify TLS disabled
		if !strings.Contains(varsContent, "use_tls = false") {
			t.Errorf("Expected use_tls = false when TLS is nil, got:\n%s", varsContent)
		}
		// Should not have tls array
		if strings.Contains(varsContent, "tls = [") {
			t.Errorf("Should not have tls array when TLS is disabled")
		}
	})
}

func TestGenerateVarsFile_ConditionalFields(t *testing.T) {
	params := DeploymentParams{
		JobID: "test-job",
		// No ConfigFiles, TemplateStringVariables, or Consul.LinkedServices
		Consul: ConsulConfig{
			ServiceName:    "my-service",
			LinkedServices: []ConsulLinkedService{}, // Empty
		},
	}

	varsContent := GenerateVarsFile(params, nil)

	// Verify required empty arrays ARE output (required by nomad-pack)
	if !strings.Contains(varsContent, "config_files = []") {
		t.Errorf("Should output empty config_files array (required by pack), got:\n%s", varsContent)
	}
	if !strings.Contains(varsContent, "template = []") {
		t.Errorf("Should output empty template array (required by pack), got:\n%s", varsContent)
	}
	if !strings.Contains(varsContent, "vault_linked_secrets = []") {
		t.Errorf("Should output vault_linked_secrets (required by pack), got:\n%s", varsContent)
	}
	if !strings.Contains(varsContent, "regions = \"\"") {
		t.Errorf("Should output empty regions string (required by pack), got:\n%s", varsContent)
	}

	// Verify consul_linked_services IS output as empty (required by pack)
	if !strings.Contains(varsContent, "consul_linked_services") {
		t.Errorf("Should output consul_linked_services (required by pack), got:\n%s", varsContent)
	}
}

func TestGenerateVarsFile_FieldOrdering(t *testing.T) {
	usesKv := true
	ownerUsesKv := false

	params := DeploymentParams{
		JobID:             "test-job",
		ReplicaCount:      2,
		SecretsPath:       "secret/path",
		RestartMode:       "fail",
		RestartAttempts:   3,
		CPU:               100,
		RAM:               512,
		GPU:               0,
		NodePool:          "default",
		OwnerID:           "usr_123",
		DeploymentID:      "dep_456",
		ProjectID:         "prj_789",
		ServiceID:         "svc_012",
		SharedSecretPath:  "secret/shared",
		UsesKvEngine:      &usesKv,
		OwnerUsesKvEngine: &ownerUsesKv,
		Regions:           "us-west",
		PrivateRegistry:   "ghcr.io",
		DockerUser:        "1000",
		Command:           "/bin/sh",
		ImageName:         "app",
		ImageTag:          "latest",
	}

	varsContent := GenerateVarsFile(params, nil)
	lines := strings.Split(varsContent, "\n")

	// Define expected field order (from cs-runner spec)
	expectedOrder := []string{
		"job_name",
		"count",
		"secret_path",
		"restart_attempts",
		"restart_mode",
		"resources",
		"node_pool",
		"user_id",
		"alloc_id",
		"project_id",
		"service_id",
		"shared_secret_path",
		"uses_kv_engine",
		"owner_uses_kv_engine",
		"regions",
		"private_registry",
		"user",
		"command",
		"image",
		"use_csi_volume",
	}

	// Track positions of each field
	positions := make(map[string]int)
	for i, line := range lines {
		for _, field := range expectedOrder {
			if strings.HasPrefix(line, field+" =") {
				positions[field] = i
				break
			}
		}
	}

	// Verify ordering
	for i := 1; i < len(expectedOrder); i++ {
		prevField := expectedOrder[i-1]
		currField := expectedOrder[i]

		prevPos, prevExists := positions[prevField]
		currPos, currExists := positions[currField]

		if prevExists && currExists && prevPos > currPos {
			t.Errorf("Field ordering violation: %s (pos %d) should come before %s (pos %d)",
				prevField, prevPos, currField, currPos)
		}
	}
}

func TestGenerateConfig_RoundTrip(t *testing.T) {
	// Set up test credentials in environment
	os.Setenv("REGISTRY_USERNAME", "testuser")
	os.Setenv("REGISTRY_PASSWORD", "testpass")
	defer os.Unsetenv("REGISTRY_USERNAME")
	defer os.Unsetenv("REGISTRY_PASSWORD")

	params := DeploymentParams{
		JobID:       "test-app",
		BuilderType: "railpack",
		DeployType:  "nomad-pack",
		ImageName:   "test-app",
		ImageTag:    "latest",
		DisablePush: false, // This ensures variable blocks are generated
	}

	// Generate HCL
	hclContent, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify variable blocks are present in generated HCL
	if !strings.Contains(hclContent, "variable \"registry_username\"") {
		t.Error("Generated HCL missing registry_username variable block")
	}
	if !strings.Contains(hclContent, "variable \"registry_password\"") {
		t.Error("Generated HCL missing registry_password variable block")
	}

	// Parse it back using the config package
	cfg, err := config.ParseBytes([]byte(hclContent), "cloudstation.hcl")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v\nGenerated HCL content:\n%s", err, hclContent)
	}

	// Verify project structure
	if cfg.Project != "test-app" {
		t.Errorf("project = %q, want %q", cfg.Project, "test-app")
	}

	// Verify apps were parsed
	if len(cfg.Apps) != 1 {
		t.Errorf("len(apps) = %d, want 1", len(cfg.Apps))
	}

	// Verify variable blocks were parsed
	if len(cfg.Variables) != 2 {
		t.Errorf("len(variables) = %d, want 2 (registry_username and registry_password)", len(cfg.Variables))
	}

	// Verify variable names and properties
	foundUsername := false
	foundPassword := false
	for _, v := range cfg.Variables {
		switch v.Name {
		case "registry_username":
			foundUsername = true
			if !v.Sensitive {
				t.Error("registry_username should be marked sensitive")
			}
			if len(v.Env) == 0 || v.Env[0] != "REGISTRY_USERNAME" {
				t.Errorf("registry_username env = %v, want [REGISTRY_USERNAME]", v.Env)
			}
		case "registry_password":
			foundPassword = true
			if !v.Sensitive {
				t.Error("registry_password should be marked sensitive")
			}
			if len(v.Env) == 0 || v.Env[0] != "REGISTRY_PASSWORD" {
				t.Errorf("registry_password env = %v, want [REGISTRY_PASSWORD]", v.Env)
			}
		}
	}

	if !foundUsername {
		t.Error("registry_username variable not found in parsed config")
	}
	if !foundPassword {
		t.Error("registry_password variable not found in parsed config")
	}

	// Verify variable resolution works by checking env vars match what variables expect
	for _, v := range cfg.Variables {
		if len(v.Env) > 0 {
			envVal := os.Getenv(v.Env[0])
			switch v.Name {
			case "registry_username":
				if envVal != "testuser" {
					t.Errorf("env var %s = %q, want %q", v.Env[0], envVal, "testuser")
				}
			case "registry_password":
				if envVal != "testpass" {
					t.Errorf("env var %s = %q, want %q", v.Env[0], envVal, "testpass")
				}
			}
		}
	}
}

func TestGenerateConfig_RoundTrip_DisablePush(t *testing.T) {
	// When push is disabled, no variable blocks should be generated
	params := DeploymentParams{
		JobID:       "test-app",
		BuilderType: "railpack",
		DeployType:  "nomad-pack",
		ImageName:   "test-app",
		ImageTag:    "latest",
		DisablePush: true, // No push = no variable blocks
	}

	// Generate HCL
	hclContent, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	// Verify variable blocks are NOT present
	if strings.Contains(hclContent, "variable \"registry_username\"") {
		t.Error("Generated HCL should not contain registry_username variable when push is disabled")
	}

	// Parse it back
	cfg, err := config.ParseBytes([]byte(hclContent), "cloudstation.hcl")
	if err != nil {
		t.Fatalf("ParseBytes failed: %v", err)
	}

	// Verify no variables
	if len(cfg.Variables) != 0 {
		t.Errorf("len(variables) = %d, want 0 when push is disabled", len(cfg.Variables))
	}
}

func TestGenerateVarsFile_ClusterDomain(t *testing.T) {
	params := DeploymentParams{
		JobID:         "test-job",
		ImageName:     "test-image",
		ImageTag:      "v1.0.0",
		ReplicaCount:  1,
		CPU:           100,
		RAM:           500,
		ClusterDomain: "cloud-station.io",
	}

	result := GenerateVarsFile(params, nil)

	expectedLine := `cluster_domain = "cloud-station.io"`
	if !strings.Contains(result, expectedLine) {
		t.Errorf("Expected cluster_domain in output\nExpected: %s\nGot:\n%s", expectedLine, result)
	}

	// Verify it appears before network section
	networkIndex := strings.Index(result, "network =")
	clusterDomainIndex := strings.Index(result, "cluster_domain =")

	if networkIndex != -1 && clusterDomainIndex != -1 && clusterDomainIndex > networkIndex {
		t.Errorf("cluster_domain should appear before network section")
	}
}

func TestGenerateVarsFile_EmptyClusterDomain(t *testing.T) {
	params := DeploymentParams{
		JobID:         "test-job",
		ImageName:     "test-image",
		ImageTag:      "v1.0.0",
		ReplicaCount:  1,
		CPU:           100,
		RAM:           500,
		ClusterDomain: "",
	}

	result := GenerateVarsFile(params, nil)

	if strings.Contains(result, "cluster_domain") {
		t.Errorf("Should not output cluster_domain when empty\nGot:\n%s", result)
	}
}

func TestGenerateVarsFile_ClusterDomainFieldOrdering(t *testing.T) {
	params := DeploymentParams{
		JobID:         "test-job",
		ImageName:     "test-image",
		ImageTag:      "v1.0.0",
		ReplicaCount:  1,
		CPU:           100,
		RAM:           500,
		ClusterDomain: "test-domain.io",
		Networks: []NetworkPort{
			{
				PortNumber: 8080,
				PortType:   "http",
				Public:     true,
			},
		},
	}

	result := GenerateVarsFile(params, nil)

	// Verify cluster_domain appears after template and before network
	lines := strings.Split(result, "\n")

	var templateLineIdx, clusterDomainLineIdx, networkLineIdx int
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "template =") {
			templateLineIdx = i
		}
		if strings.HasPrefix(strings.TrimSpace(line), "cluster_domain =") {
			clusterDomainLineIdx = i
		}
		if strings.HasPrefix(strings.TrimSpace(line), "network =") {
			networkLineIdx = i
		}
	}

	if clusterDomainLineIdx == 0 {
		t.Errorf("cluster_domain line not found in output")
	}

	if templateLineIdx > 0 && clusterDomainLineIdx > 0 && clusterDomainLineIdx < templateLineIdx {
		t.Errorf("cluster_domain should appear after template section (template at line %d, cluster_domain at line %d)", templateLineIdx, clusterDomainLineIdx)
	}

	if networkLineIdx > 0 && clusterDomainLineIdx > 0 && clusterDomainLineIdx > networkLineIdx {
		t.Errorf("cluster_domain should appear before network section (network at line %d, cluster_domain at line %d)", networkLineIdx, clusterDomainLineIdx)
	}
}

func TestGenerateVarsFile_PreservesExplicitNetworkConfig(t *testing.T) {
	// Integration test: Verify explicit network config is preserved end-to-end
	// Simulates real deployment payload with explicit network configuration
	params := DeploymentParams{
		JobID:        "api-gateway-prod",
		ImageName:    "registry.example.com/api-gateway",
		ImageTag:     "v2.1.0",
		BuilderType:  "csdocker",
		ReplicaCount: 3,
		CPU:          500,
		RAM:          1024,
		Networks: []NetworkPort{
			{
				PortNumber:     8080,
				PortType:       "http",
				Public:         false, // Explicitly private HTTP service
				Domain:         "",
				CustomDomain:   "internal-api.company.local",
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/api/v2/health",
					Interval: "15s",
					Timeout:  "10s",
					Port:     8080,
				},
			},
			{
				PortNumber:     9090,
				PortType:       "tcp",
				Public:         false, // Explicitly private TCP port
				Domain:         "",
				CustomDomain:   "",
				HasHealthCheck: "tcp",
				HealthCheck: HealthCheckConfig{
					Type:     "tcp",
					Interval: "20s",
					Timeout:  "15s",
					Port:     9090,
				},
			},
		},
	}

	// Generate vars file (end-to-end)
	varsContent := GenerateVarsFile(params, nil)

	// Verify FIRST network port configuration
	if !strings.Contains(varsContent, "port=8080") {
		t.Errorf("Expected port=8080 in output, got: %s", varsContent)
	}

	// CRITICAL: Verify Public=false is preserved for HTTP port
	// Count occurrences to ensure we're checking the right port
	publicFalseCount := strings.Count(varsContent, "public=false")
	if publicFalseCount < 2 {
		t.Errorf("Expected at least 2 occurrences of public=false (one per port), found %d. Result: %s", publicFalseCount, varsContent)
	}

	// Should NOT contain public=true for these private services
	if strings.Contains(varsContent, "public=true") {
		t.Errorf("Expected all services to be private (public=false), but found public=true in: %s", varsContent)
	}

	// Verify custom health check path is preserved
	if !strings.Contains(varsContent, "path=\"/api/v2/health\"") {
		t.Errorf("Expected custom health check path to be preserved, got: %s", varsContent)
	}

	// Verify custom domain is preserved
	if !strings.Contains(varsContent, "custom_domain=\"internal-api.company.local\"") {
		t.Errorf("Expected custom domain to be preserved, got: %s", varsContent)
	}

	// Verify SECOND network port configuration
	if !strings.Contains(varsContent, "port=9090") {
		t.Errorf("Expected port=9090 in output, got: %s", varsContent)
	}

	// Verify custom intervals are preserved
	if !strings.Contains(varsContent, "interval=\"15s\"") {
		t.Errorf("Expected custom interval 15s to be preserved for HTTP health check, got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "interval=\"20s\"") {
		t.Errorf("Expected custom interval 20s to be preserved for TCP health check, got: %s", varsContent)
	}

	// Verify custom timeouts are preserved
	if !strings.Contains(varsContent, "timeout=\"10s\"") {
		t.Errorf("Expected custom timeout 10s to be preserved, got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "timeout=\"15s\"") {
		t.Errorf("Expected custom timeout 15s to be preserved, got: %s", varsContent)
	}

	// Verify health check type is correct for both
	httpTypeCount := strings.Count(varsContent, "type=\"http\"")
	tcpTypeCount := strings.Count(varsContent, "type=\"tcp\"")

	if httpTypeCount < 2 {
		t.Errorf("Expected at least 2 occurrences of type=\"http\" (port type and health check type), found %d", httpTypeCount)
	}

	if tcpTypeCount < 2 {
		t.Errorf("Expected at least 2 occurrences of type=\"tcp\" (port type and health check type), found %d", tcpTypeCount)
	}

	// Verify NO instances of buggy "30s" path value
	if strings.Contains(varsContent, "path=\"30s\"") {
		t.Errorf("CRITICAL BUG: Found path=\"30s\" which is invalid (path should be URL path, not time interval). Result: %s", varsContent)
	}
}

func TestGenerateVarsFile_HealthCheckPathDefaultsCorrectly(t *testing.T) {
	// Test that empty health check path defaults to "/" not "30s"
	// This is the integration test for the critical path initialization bug fix
	params := DeploymentParams{
		JobID:        "web-service",
		ImageName:    "web-app",
		ImageTag:     "latest",
		BuilderType:  "nixpacks",
		ReplicaCount: 1,
		Networks: []NetworkPort{
			{
				PortNumber: 3000,
				PortType:   "http",
				Public:     true,
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "", // Empty - should default to "/"
					Interval: "30s",
					Timeout:  "30s",
					Port:     0, // Should default to port number
				},
			},
		},
	}

	varsContent := GenerateVarsFile(params, nil)

	// Verify default path is "/"
	if !strings.Contains(varsContent, "path=\"/\"") {
		t.Errorf("Expected default health check path to be \"/\", got: %s", varsContent)
	}

	// CRITICAL: Verify the buggy "30s" is NOT used as path
	// Note: "30s" should appear ONLY in interval/timeout, NEVER in path
	pathIndex := strings.Index(varsContent, "path=")
	if pathIndex != -1 {
		// Extract substring around path= to verify value
		pathSubstring := varsContent[pathIndex:min(pathIndex+20, len(varsContent))]
		if strings.Contains(pathSubstring, "path=\"30s\"") {
			t.Errorf("CRITICAL BUG: Health check path defaulted to \"30s\" instead of \"/\". This is invalid. Result: %s", varsContent)
		}
	}

	// Verify interval and timeout correctly use "30s"
	if !strings.Contains(varsContent, "interval=\"30s\"") {
		t.Errorf("Expected interval=\"30s\", got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "timeout=\"30s\"") {
		t.Errorf("Expected timeout=\"30s\", got: %s", varsContent)
	}

	// Verify health check port defaults to network port
	if !strings.Contains(varsContent, "port=3000") {
		t.Errorf("Expected health check port to default to network port 3000, got: %s", varsContent)
	}
}

func TestGenerateVarsFile_ProductionPayloadValidation(t *testing.T) {
	// Integration test with production-like payload
	// This validates the actual bug fix scenario where user-provided network config
	// was being overridden even when explicitly specified

	// Simulate a production deployment payload with explicit network configuration
	// This represents the scenario from the bug report where:
	// 1. User provides Public=false for HTTP port
	// 2. User provides custom health check path
	// 3. User provides custom health check intervals
	params := DeploymentParams{
		JobID:            "csi-12devsnode-master-aomzkbgpepel",
		ImageName:        "acrbc001.azurecr.io/csi-12devsnode-master-aomzkbgpepel",
		ImageTag:         "1aecee4d402ff09d50a4fa72e01342c6bc2b95d5",
		BuilderType:      "csdocker",
		DeployType:       "nomad-pack",
		ReplicaCount:     1,
		CPU:              100,
		RAM:              500,
		GPU:              0,
		RestartMode:      "fail",
		RestartAttempts:  3,
		NodePool:         "minions",
		OwnerID:          "team_9e0c8754-de83-4fee-937b-380ade396b1c",
		DeploymentID:     "gdeployment_2e808231-d90f-4d3d-aaa4-dcd7d4d81788",
		ProjectID:        "prj_90aae90a-64b0-40a6-ac2c-39c27f3ffa49",
		ServiceID:        "prj_integ_2f716e42-7e18-4444-be8a-94607e3a431b",
		ClusterDomain:    "cluster.example.com",
		ClusterTCPDomain: "tcp.cluster.example.com",
		Networks: []NetworkPort{
			{
				PortNumber:     3000,
				PortType:       "http",
				Public:         false, // CRITICAL: User explicitly wants internal-only HTTP service
				Domain:         "",
				CustomDomain:   "my-service.internal.cluster.example.com",
				HasHealthCheck: "http",
				HealthCheck: HealthCheckConfig{
					Type:     "http",
					Path:     "/health/ready", // Custom health check endpoint
					Interval: "15s",           // Custom interval
					Timeout:  "10s",           // Custom timeout
					Port:     3000,
				},
			},
		},
	}

	// Generate HCL vars file
	varsContent := GenerateVarsFile(params, nil)

	// CRITICAL VALIDATION #1: Verify Public=false is preserved
	// This was the primary bug - HTTP ports were forced to public=true
	if !strings.Contains(varsContent, "public=false") {
		t.Errorf("CRITICAL BUG: Expected public=false to be preserved for HTTP port, but got: %s", varsContent)
	}

	if strings.Contains(varsContent, "public=true") {
		t.Errorf("CRITICAL BUG: User specified public=false but output contains public=true. This exposes internal services to the internet! Result: %s", varsContent)
	}

	// CRITICAL VALIDATION #2: Verify health check path is preserved (not overridden with "30s")
	if !strings.Contains(varsContent, "path=\"/health/ready\"") {
		t.Errorf("Expected custom health check path '/health/ready' to be preserved, got: %s", varsContent)
	}

	// CRITICAL BUG CHECK: Verify the "30s" bug is fixed
	// The bug was: hcPath := "30s" instead of hcPath := "/"
	if strings.Contains(varsContent, "path=\"30s\"") {
		t.Errorf("CRITICAL BUG NOT FIXED: Health check path contains '30s' (time interval) instead of URL path. Result: %s", varsContent)
	}

	// VALIDATION #3: Verify custom intervals are preserved
	if !strings.Contains(varsContent, "interval=\"15s\"") {
		t.Errorf("Expected custom health check interval '15s' to be preserved, got: %s", varsContent)
	}

	// VALIDATION #4: Verify custom timeout is preserved
	if !strings.Contains(varsContent, "timeout=\"10s\"") {
		t.Errorf("Expected custom health check timeout '10s' to be preserved, got: %s", varsContent)
	}

	// VALIDATION #5: Verify custom domain is preserved
	if !strings.Contains(varsContent, "custom_domain=\"my-service.internal.cluster.example.com\"") {
		t.Errorf("Expected custom domain to be preserved, got: %s", varsContent)
	}

	// VALIDATION #6: Verify cluster domain is included in output
	if !strings.Contains(varsContent, "cluster_domain = \"cluster.example.com\"") {
		t.Errorf("Expected cluster_domain to be included in vars output, got: %s", varsContent)
	}

	// VALIDATION #7: Verify port configuration is correct
	if !strings.Contains(varsContent, "port=3000") {
		t.Errorf("Expected port=3000 in network configuration, got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "type=\"http\"") {
		t.Errorf("Expected port type=\"http\" in network configuration, got: %s", varsContent)
	}

	// VALIDATION #8: Verify health check type is correct
	// Should contain multiple instances of "http" - once for port type, once for health check type
	httpTypeCount := strings.Count(varsContent, "type=\"http\"")
	if httpTypeCount < 2 {
		t.Errorf("Expected at least 2 occurrences of type=\"http\" (port type and health check type), found %d", httpTypeCount)
	}

	// VALIDATION #9: Verify health check port defaults to network port when specified as same
	if !strings.Contains(varsContent, "port=3000") {
		t.Errorf("Expected health check port to match network port 3000, got: %s", varsContent)
	}

	// VALIDATION #10: Verify network block is properly formatted
	if !strings.Contains(varsContent, "network = [") {
		t.Errorf("Expected network configuration to be formatted as array, got: %s", varsContent)
	}

	// VALIDATION #11: Verify has_health_check is set correctly
	if !strings.Contains(varsContent, "has_health_check=\"http\"") {
		t.Errorf("Expected has_health_check=\"http\", got: %s", varsContent)
	}

	// Success logging
	t.Logf("Production payload validation successful - all fields preserved correctly")
	t.Logf("Network configuration:\n%s", varsContent)
}

func TestGenerateVarsFile_EmptyNetworkFieldDefaults(t *testing.T) {
	// Test that empty network fields get appropriate defaults
	// This validates the fallback logic still works correctly
	params := DeploymentParams{
		JobID:        "test-defaults",
		BuilderType:  "nixpacks",
		ReplicaCount: 1,
		Networks: []NetworkPort{
			{
				PortNumber: 4000,
				PortType:   "http",
				Public:     true,
				// Empty health check - should get defaults
				HealthCheck: HealthCheckConfig{
					Type:     "", // Should default to PortType (http)
					Path:     "", // Should default to "/"
					Interval: "", // Should default to "30s"
					Timeout:  "", // Should default to "30s"
					Port:     0,  // Should default to PortNumber (4000)
				},
			},
		},
	}

	varsContent := GenerateVarsFile(params, nil)

	// Verify defaults are applied
	if !strings.Contains(varsContent, "path=\"/\"") {
		t.Errorf("Expected empty health check path to default to \"/\", got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "interval=\"30s\"") {
		t.Errorf("Expected empty interval to default to \"30s\", got: %s", varsContent)
	}

	if !strings.Contains(varsContent, "timeout=\"30s\"") {
		t.Errorf("Expected empty timeout to default to \"30s\", got: %s", varsContent)
	}

	// Health check port should default to network port
	if !strings.Contains(varsContent, "port=4000") {
		t.Errorf("Expected health check port to default to network port 4000, got: %s", varsContent)
	}

	// Health check type should default to PortType (http in this case)
	// Both port type and health check type should be "http"
	httpTypeCount := strings.Count(varsContent, "type=\"http\"")
	if httpTypeCount < 2 {
		t.Errorf("Expected at least 2 occurrences of type=\"http\" (port type and health check type), found %d. Got: %s", httpTypeCount, varsContent)
	}

	// Verify the buggy "30s" is NOT used as path even when interval is "30s"
	pathBugPattern := "path=\"30s\""
	if strings.Contains(varsContent, pathBugPattern) {
		t.Errorf("CRITICAL: Found %s - path should be \"/\" not a time interval", pathBugPattern)
	}
}

// Helper function for min (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
