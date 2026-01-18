package nomadpack

import (
	"context"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

func TestConfigSet(t *testing.T) {
	tests := []struct {
		name     string
		config   interface{}
		expected *PlatformConfig
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: &PlatformConfig{},
		},
		{
			name: "map config with all fields",
			config: map[string]interface{}{
				"deployment_name": "test-deployment",
				"pack":            "nginx",
				"nomad_addr":      "http://localhost:4646",
				"nomad_token":     "test-token",
				"registry_name":   "test-registry",
				"registry_source": "https://github.com/hashicorp/nomad-pack-community-registry",
				"registry_ref":    "main",
				"registry_target": "packs/nginx",
				"registry_token":  "ghp_test",
				"variables": map[string]interface{}{
					"port": "8080",
					"host": "example.com",
				},
				"variable_files": []interface{}{
					"vars.hcl",
					"override.hcl",
				},
			},
			expected: &PlatformConfig{
				DeploymentName: "test-deployment",
				Pack:           "nginx",
				NomadAddr:      "http://localhost:4646",
				NomadToken:     "test-token",
				RegistryName:   "test-registry",
				RegistrySource: "https://github.com/hashicorp/nomad-pack-community-registry",
				RegistryRef:    "main",
				RegistryTarget: "packs/nginx",
				RegistryToken:  "ghp_test",
				Variables: map[string]string{
					"port": "8080",
					"host": "example.com",
				},
				VariableFiles: []string{
					"vars.hcl",
					"override.hcl",
				},
			},
		},
		{
			name: "map config with minimal fields",
			config: map[string]interface{}{
				"deployment_name": "minimal",
				"pack":            "nginx",
				"registry_name":   "test",
				"registry_source": "https://github.com/test/registry",
			},
			expected: &PlatformConfig{
				DeploymentName: "minimal",
				Pack:           "nginx",
				RegistryName:   "test",
				RegistrySource: "https://github.com/test/registry",
			},
		},
		{
			name: "typed config",
			config: &PlatformConfig{
				DeploymentName: "typed-deployment",
				Pack:           "redis",
				RegistryName:   "typed-registry",
				RegistrySource: "https://github.com/typed/registry",
			},
			expected: &PlatformConfig{
				DeploymentName: "typed-deployment",
				Pack:           "redis",
				RegistryName:   "typed-registry",
				RegistrySource: "https://github.com/typed/registry",
			},
		},
		{
			name: "map config with parser_v1 true",
			config: map[string]interface{}{
				"deployment_name": "v1-pack",
				"pack":            "legacy-pack",
				"registry_name":   "test",
				"registry_source": "https://github.com/test/registry",
				"parser_v1":       true,
			},
			expected: &PlatformConfig{
				DeploymentName: "v1-pack",
				Pack:           "legacy-pack",
				RegistryName:   "test",
				RegistrySource: "https://github.com/test/registry",
				ParserV1:       true,
			},
		},
		{
			name: "map config with parser_v1 false (explicit)",
			config: map[string]interface{}{
				"deployment_name": "v2-pack",
				"pack":            "modern-pack",
				"registry_name":   "test",
				"registry_source": "https://github.com/test/registry",
				"parser_v1":       false,
			},
			expected: &PlatformConfig{
				DeploymentName: "v2-pack",
				Pack:           "modern-pack",
				RegistryName:   "test",
				RegistrySource: "https://github.com/test/registry",
				ParserV1:       false,
			},
		},
		{
			name: "map config without parser_v1 (default false)",
			config: map[string]interface{}{
				"deployment_name": "default-pack",
				"pack":            "new-pack",
				"registry_name":   "test",
				"registry_source": "https://github.com/test/registry",
			},
			expected: &PlatformConfig{
				DeploymentName: "default-pack",
				Pack:           "new-pack",
				RegistryName:   "test",
				RegistrySource: "https://github.com/test/registry",
				ParserV1:       false, // Default value
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{}
			err := p.ConfigSet(tt.config)
			if err != nil {
				t.Fatalf("ConfigSet() error = %v", err)
			}

			if p.config.DeploymentName != tt.expected.DeploymentName {
				t.Errorf("DeploymentName = %v, want %v", p.config.DeploymentName, tt.expected.DeploymentName)
			}
			if p.config.Pack != tt.expected.Pack {
				t.Errorf("Pack = %v, want %v", p.config.Pack, tt.expected.Pack)
			}
			if p.config.NomadAddr != tt.expected.NomadAddr {
				t.Errorf("NomadAddr = %v, want %v", p.config.NomadAddr, tt.expected.NomadAddr)
			}
			if p.config.NomadToken != tt.expected.NomadToken {
				t.Errorf("NomadToken = %v, want %v", p.config.NomadToken, tt.expected.NomadToken)
			}
			if p.config.RegistryName != tt.expected.RegistryName {
				t.Errorf("RegistryName = %v, want %v", p.config.RegistryName, tt.expected.RegistryName)
			}
			if p.config.RegistrySource != tt.expected.RegistrySource {
				t.Errorf("RegistrySource = %v, want %v", p.config.RegistrySource, tt.expected.RegistrySource)
			}
			if p.config.RegistryRef != tt.expected.RegistryRef {
				t.Errorf("RegistryRef = %v, want %v", p.config.RegistryRef, tt.expected.RegistryRef)
			}
			if p.config.RegistryTarget != tt.expected.RegistryTarget {
				t.Errorf("RegistryTarget = %v, want %v", p.config.RegistryTarget, tt.expected.RegistryTarget)
			}
			if p.config.RegistryToken != tt.expected.RegistryToken {
				t.Errorf("RegistryToken = %v, want %v", p.config.RegistryToken, tt.expected.RegistryToken)
			}
			if p.config.ParserV1 != tt.expected.ParserV1 {
				t.Errorf("ParserV1 = %v, want %v", p.config.ParserV1, tt.expected.ParserV1)
			}

			// Check variables map
			if tt.expected.Variables != nil {
				if len(p.config.Variables) != len(tt.expected.Variables) {
					t.Errorf("Variables length = %v, want %v", len(p.config.Variables), len(tt.expected.Variables))
				}
				for k, v := range tt.expected.Variables {
					if p.config.Variables[k] != v {
						t.Errorf("Variables[%s] = %v, want %v", k, p.config.Variables[k], v)
					}
				}
			}

			// Check variable files slice
			if tt.expected.VariableFiles != nil {
				if len(p.config.VariableFiles) != len(tt.expected.VariableFiles) {
					t.Errorf("VariableFiles length = %v, want %v", len(p.config.VariableFiles), len(tt.expected.VariableFiles))
				}
				for i, vf := range tt.expected.VariableFiles {
					if p.config.VariableFiles[i] != vf {
						t.Errorf("VariableFiles[%d] = %v, want %v", i, p.config.VariableFiles[i], vf)
					}
				}
			}
		})
	}
}

func TestConfig(t *testing.T) {
	expectedConfig := &PlatformConfig{
		DeploymentName: "test",
		Pack:           "nginx",
		RegistryName:   "test-registry",
		RegistrySource: "https://github.com/test/registry",
	}

	p := &Platform{config: expectedConfig}

	cfg, err := p.Config()
	if err != nil {
		t.Fatalf("Config() error = %v", err)
	}

	configTyped, ok := cfg.(*PlatformConfig)
	if !ok {
		t.Fatalf("Config() returned wrong type")
	}

	if configTyped != expectedConfig {
		t.Errorf("Config() returned different pointer")
	}
}

func TestSetVarArgs(t *testing.T) {
	tests := []struct {
		name         string
		config       *PlatformConfig
		initialArgs  []string
		expectedArgs []string
	}{
		{
			name: "with variables and variable files",
			config: &PlatformConfig{
				Variables: map[string]string{
					"port": "8080",
					"host": "example.com",
				},
				VariableFiles: []string{
					"vars.hcl",
					"override.hcl",
				},
			},
			initialArgs: []string{"run", "nginx"},
			expectedArgs: []string{
				"run", "nginx",
				"--var=port=8080",
				"--var=host=example.com",
				"--var-file=vars.hcl",
				"--var-file=override.hcl",
			},
		},
		{
			name: "with only variables",
			config: &PlatformConfig{
				Variables: map[string]string{
					"count": "3",
				},
			},
			initialArgs: []string{"run", "redis"},
			expectedArgs: []string{
				"run", "redis",
				"--var=count=3",
			},
		},
		{
			name: "with only variable files",
			config: &PlatformConfig{
				VariableFiles: []string{
					"prod.hcl",
				},
			},
			initialArgs: []string{"run", "postgres"},
			expectedArgs: []string{
				"run", "postgres",
				"--var-file=prod.hcl",
			},
		},
		{
			name:         "with no variables or files",
			config:       &PlatformConfig{},
			initialArgs:  []string{"run", "app"},
			expectedArgs: []string{"run", "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{config: tt.config}
			result := p.setVarArgs(tt.initialArgs)

			// Check length
			if len(result) < len(tt.initialArgs) {
				t.Errorf("setVarArgs() result length = %v, should be >= %v", len(result), len(tt.initialArgs))
			}

			// Check that initial args are preserved
			for i, arg := range tt.initialArgs {
				if result[i] != arg {
					t.Errorf("Initial arg[%d] = %v, want %v", i, result[i], arg)
				}
			}

			// For tests with variables, check that --var args are present
			if tt.config.Variables != nil {
				varCount := 0
				for _, arg := range result {
					if len(arg) > 6 && arg[:6] == "--var=" {
						varCount++
					}
				}
				if varCount != len(tt.config.Variables) {
					t.Errorf("Number of --var args = %v, want %v", varCount, len(tt.config.Variables))
				}
			}

			// For tests with variable files, check that --var-file args are present
			if tt.config.VariableFiles != nil {
				varFileCount := 0
				for _, arg := range result {
					if len(arg) > 11 && arg[:11] == "--var-file=" {
						varFileCount++
					}
				}
				if varFileCount != len(tt.config.VariableFiles) {
					t.Errorf("Number of --var-file args = %v, want %v", varFileCount, len(tt.config.VariableFiles))
				}
			}
		})
	}
}

func TestDeploy_Validation(t *testing.T) {
	ctx := context.Background()
	art := &artifact.Artifact{ID: "test-artifact"}

	tests := []struct {
		name    string
		config  *PlatformConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration is not set",
		},
		{
			name: "missing pack",
			config: &PlatformConfig{
				DeploymentName: "test",
				RegistryName:   "test-registry",
				RegistrySource: "https://github.com/test/registry",
			},
			wantErr: true,
			errMsg:  "pack name is required",
		},
		{
			name: "missing deployment_name",
			config: &PlatformConfig{
				Pack:           "nginx",
				RegistryName:   "test-registry",
				RegistrySource: "https://github.com/test/registry",
			},
			wantErr: true,
			errMsg:  "deployment_name is required",
		},
		{
			name: "missing registry_name",
			config: &PlatformConfig{
				DeploymentName: "test",
				Pack:           "nginx",
				RegistrySource: "https://github.com/test/registry",
			},
			wantErr: true,
			errMsg:  "registry_name is required",
		},
		{
			name: "missing registry_source",
			config: &PlatformConfig{
				DeploymentName: "test",
				Pack:           "nginx",
				RegistryName:   "test-registry",
			},
			wantErr: true,
			errMsg:  "registry_source is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{config: tt.config}
			_, err := p.Deploy(ctx, art)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Deploy() expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Deploy() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Deploy() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestDestroy_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *PlatformConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{config: tt.config}
			err := p.Destroy(ctx, "test-deployment")

			if tt.wantErr {
				if err == nil {
					t.Errorf("Destroy() expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Destroy() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Destroy() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestStatus_Validation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  *PlatformConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{config: tt.config}
			_, err := p.Status(ctx, "test-deployment")

			if tt.wantErr {
				if err == nil {
					t.Errorf("Status() expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Status() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Status() unexpected error = %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestConfigSet_EmbeddedFields(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *PlatformConfig
	}{
		{
			name: "with use_embedded true and embedded_pack",
			config: map[string]interface{}{
				"deployment_name": "test-embedded",
				"pack":            "cloudstation",
				"use_embedded":    true,
				"embedded_pack":   "cloudstation",
			},
			expected: &PlatformConfig{
				DeploymentName: "test-embedded",
				Pack:           "cloudstation",
				UseEmbedded:    true,
				EmbeddedPack:   "cloudstation",
			},
		},
		{
			name: "with use_embedded true without embedded_pack",
			config: map[string]interface{}{
				"deployment_name": "test-embedded2",
				"pack":            "cloudstation",
				"use_embedded":    true,
			},
			expected: &PlatformConfig{
				DeploymentName: "test-embedded2",
				Pack:           "cloudstation",
				UseEmbedded:    true,
				EmbeddedPack:   "",
			},
		},
		{
			name: "with use_embedded false (remote registry)",
			config: map[string]interface{}{
				"deployment_name": "test-remote",
				"pack":            "nginx",
				"use_embedded":    false,
				"registry_name":   "test-registry",
				"registry_source": "https://github.com/test/registry",
			},
			expected: &PlatformConfig{
				DeploymentName: "test-remote",
				Pack:           "nginx",
				UseEmbedded:    false,
				EmbeddedPack:   "",
				RegistryName:   "test-registry",
				RegistrySource: "https://github.com/test/registry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{}
			err := p.ConfigSet(tt.config)
			if err != nil {
				t.Fatalf("ConfigSet() error = %v", err)
			}

			if p.config.DeploymentName != tt.expected.DeploymentName {
				t.Errorf("DeploymentName = %v, want %v", p.config.DeploymentName, tt.expected.DeploymentName)
			}
			if p.config.Pack != tt.expected.Pack {
				t.Errorf("Pack = %v, want %v", p.config.Pack, tt.expected.Pack)
			}
			if p.config.UseEmbedded != tt.expected.UseEmbedded {
				t.Errorf("UseEmbedded = %v, want %v", p.config.UseEmbedded, tt.expected.UseEmbedded)
			}
			if p.config.EmbeddedPack != tt.expected.EmbeddedPack {
				t.Errorf("EmbeddedPack = %v, want %v", p.config.EmbeddedPack, tt.expected.EmbeddedPack)
			}
		})
	}
}

func TestDeploy_EmbeddedPack_Validation(t *testing.T) {
	ctx := context.Background()
	art := &artifact.Artifact{ID: "test-artifact"}

	tests := []struct {
		name    string
		config  *PlatformConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "embedded pack with valid pack name",
			config: &PlatformConfig{
				DeploymentName: "test-embedded",
				Pack:           "cloudstation",
				UseEmbedded:    true,
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "embedded pack with explicit embedded_pack field",
			config: &PlatformConfig{
				DeploymentName: "test-embedded2",
				Pack:           "cloudstation",
				UseEmbedded:    true,
				EmbeddedPack:   "cloudstation",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "embedded pack with non-existent pack",
			config: &PlatformConfig{
				DeploymentName: "test-bad",
				Pack:           "nonexistent",
				UseEmbedded:    true,
			},
			wantErr: true,
			errMsg:  "embedded pack",
		},
		{
			name: "embedded pack without registry fields (should pass)",
			config: &PlatformConfig{
				DeploymentName: "test-no-registry",
				Pack:           "cloudstation",
				UseEmbedded:    true,
				// No RegistryName or RegistrySource - should not error
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "remote pack requires registry_name",
			config: &PlatformConfig{
				DeploymentName: "test-remote",
				Pack:           "nginx",
				UseEmbedded:    false,
				// Missing RegistryName and RegistrySource
			},
			wantErr: true,
			errMsg:  "registry_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{config: tt.config}
			_, err := p.Deploy(ctx, art)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Deploy() expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Deploy() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				// For successful cases, we expect the command to fail (no nomad-pack installed)
				// but we should get past validation
				if err != nil && contains(err.Error(), "required") {
					t.Errorf("Deploy() should pass validation but got: %v", err)
				}
			}
		})
	}
}
