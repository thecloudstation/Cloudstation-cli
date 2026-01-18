package config

import (
	"os"
	"testing"
)

func TestExpandEnvVarsExtended(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_TOKEN", "secret123")
	os.Setenv("TEST_PATH", "/home/user")
	defer os.Unsetenv("TEST_TOKEN")
	defer os.Unsetenv("TEST_PATH")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "env with double quotes",
			input:    `env("TEST_TOKEN")`,
			expected: "secret123",
		},
		{
			name:     "env with single quotes",
			input:    `env('TEST_TOKEN')`,
			expected: "secret123",
		},
		{
			name:     "dollar brace syntax",
			input:    "${TEST_TOKEN}",
			expected: "secret123",
		},
		{
			name:     "mixed env and text",
			input:    `prefix-env("TEST_TOKEN")-suffix`,
			expected: "prefix-secret123-suffix",
		},
		{
			name:     "multiple env calls",
			input:    `env("TEST_PATH")/env("TEST_TOKEN")`,
			expected: "/home/user/secret123",
		},
		{
			name:     "unset variable returns empty",
			input:    `env("NONEXISTENT_VAR")`,
			expected: "",
		},
		{
			name:     "plain string unchanged",
			input:    "plain string",
			expected: "plain string",
		},
		{
			name:     "mixed dollar and env syntax",
			input:    `${TEST_PATH}/env("TEST_TOKEN")`,
			expected: "/home/user/secret123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessConfigMapEnvExpansion(t *testing.T) {
	os.Setenv("TEST_SECRET", "my-secret-value")
	defer os.Unsetenv("TEST_SECRET")

	input := map[string]interface{}{
		"token": `env("TEST_SECRET")`,
		"url":   "${TEST_SECRET}",
		"plain": "no-expansion",
		"nested": map[string]interface{}{
			"inner_token": `env("TEST_SECRET")`,
		},
	}

	result := processConfigMap(input)

	// Check top-level env() expansion
	if result["token"] != "my-secret-value" {
		t.Errorf("token = %q, want %q", result["token"], "my-secret-value")
	}

	// Check ${} expansion
	if result["url"] != "my-secret-value" {
		t.Errorf("url = %q, want %q", result["url"], "my-secret-value")
	}

	// Check plain string unchanged
	if result["plain"] != "no-expansion" {
		t.Errorf("plain = %q, want %q", result["plain"], "no-expansion")
	}

	// Check nested map expansion
	nested, ok := result["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("nested is not a map[string]interface{}, got %T", result["nested"])
	}
	if nested["inner_token"] != "my-secret-value" {
		t.Errorf("nested.inner_token = %q, want %q", nested["inner_token"], "my-secret-value")
	}
}

func TestProcessConfigMapComplexNesting(t *testing.T) {
	os.Setenv("TEST_VAR", "value123")
	defer os.Unsetenv("TEST_VAR")

	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": `env("TEST_VAR")`,
				"other":  "static",
			},
			"sibling": `prefix-env("TEST_VAR")-suffix`,
		},
		"array": []interface{}{
			`env("TEST_VAR")`,
			"static",
			map[string]interface{}{
				"nested_in_array": `env("TEST_VAR")`,
			},
		},
	}

	result := processConfigMap(input)

	// Check deep nesting
	level1, ok := result["level1"].(map[string]interface{})
	if !ok {
		t.Fatalf("level1 is not a map[string]interface{}, got %T", result["level1"])
	}

	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatalf("level2 is not a map[string]interface{}, got %T", level1["level2"])
	}

	if level2["level3"] != "value123" {
		t.Errorf("level3 = %q, want %q", level2["level3"], "value123")
	}

	if level2["other"] != "static" {
		t.Errorf("other = %q, want %q", level2["other"], "static")
	}

	if level1["sibling"] != "prefix-value123-suffix" {
		t.Errorf("sibling = %q, want %q", level1["sibling"], "prefix-value123-suffix")
	}

	// Check array processing
	array, ok := result["array"].([]interface{})
	if !ok {
		t.Fatalf("array is not a []interface{}, got %T", result["array"])
	}

	if len(array) != 3 {
		t.Fatalf("array length = %d, want 3", len(array))
	}

	if array[0] != "value123" {
		t.Errorf("array[0] = %q, want %q", array[0], "value123")
	}

	if array[1] != "static" {
		t.Errorf("array[1] = %q, want %q", array[1], "static")
	}

	nestedInArray, ok := array[2].(map[string]interface{})
	if !ok {
		t.Fatalf("array[2] is not a map[string]interface{}, got %T", array[2])
	}

	if nestedInArray["nested_in_array"] != "value123" {
		t.Errorf("nested_in_array = %q, want %q", nestedInArray["nested_in_array"], "value123")
	}
}

func TestExpandEnvVarsEdgeCases(t *testing.T) {
	os.Setenv("TEST_EDGE", "edge-value")
	defer os.Unsetenv("TEST_EDGE")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "env without quotes should not expand",
			input:    "env(TEST_EDGE)",
			expected: "env(TEST_EDGE)",
		},
		{
			name:     "env with mismatched quotes",
			input:    `env("TEST_EDGE')`,
			expected: "edge-value", // The regex captures TEST_EDGE despite mismatched quotes
		},
		{
			name:     "nested env calls not supported",
			input:    `env("env("TEST_EDGE")")`,
			expected: `env("edge-value")`, // Inner env gets expanded
		},
		{
			name:     "env with spaces around var name",
			input:    `env(" TEST_EDGE ")`,
			expected: "", // Spaces are part of the var name, which doesn't exist
		},
		{
			name:     "multiple spaces and special chars",
			input:    `path/to/env("TEST_EDGE")/and/${TEST_EDGE}/end`,
			expected: "path/to/edge-value/and/edge-value/end",
		},
		{
			name:     "env in middle of word",
			input:    `someenv("TEST_EDGE")text`,
			expected: "someedge-valuetext",
		},
		{
			name:     "dollar syntax with non-existent var",
			input:    "${NONEXISTENT}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessSlice(t *testing.T) {
	os.Setenv("TEST_SLICE", "slice-value")
	defer os.Unsetenv("TEST_SLICE")

	input := []interface{}{
		`env("TEST_SLICE")`,
		"${TEST_SLICE}",
		"plain",
		[]interface{}{
			`nested-env("TEST_SLICE")`,
			"nested-plain",
		},
		map[string]interface{}{
			"key": `env("TEST_SLICE")`,
		},
		123,  // Non-string value
		true, // Boolean value
	}

	result := processSlice(input)

	if len(result) != len(input) {
		t.Fatalf("result length = %d, want %d", len(result), len(input))
	}

	// Check string expansions
	if result[0] != "slice-value" {
		t.Errorf("result[0] = %q, want %q", result[0], "slice-value")
	}

	if result[1] != "slice-value" {
		t.Errorf("result[1] = %q, want %q", result[1], "slice-value")
	}

	if result[2] != "plain" {
		t.Errorf("result[2] = %q, want %q", result[2], "plain")
	}

	// Check nested slice
	nestedSlice, ok := result[3].([]interface{})
	if !ok {
		t.Fatalf("result[3] is not a []interface{}, got %T", result[3])
	}

	if nestedSlice[0] != "nested-slice-value" {
		t.Errorf("nestedSlice[0] = %q, want %q", nestedSlice[0], "nested-slice-value")
	}

	// Check nested map
	nestedMap, ok := result[4].(map[string]interface{})
	if !ok {
		t.Fatalf("result[4] is not a map[string]interface{}, got %T", result[4])
	}

	if nestedMap["key"] != "slice-value" {
		t.Errorf("nestedMap[\"key\"] = %q, want %q", nestedMap["key"], "slice-value")
	}

	// Check non-string values remain unchanged
	if result[5] != 123 {
		t.Errorf("result[5] = %v, want %v", result[5], 123)
	}

	if result[6] != true {
		t.Errorf("result[6] = %v, want %v", result[6], true)
	}
}

func TestProcessEnvVars_FullConfig(t *testing.T) {
	// Set environment variables
	os.Setenv("INTEGRATION_TOKEN", "test-token-123")
	os.Setenv("INTEGRATION_PATH", "/test/path")
	defer os.Unsetenv("INTEGRATION_TOKEN")
	defer os.Unsetenv("INTEGRATION_PATH")

	// Create a config struct manually to test env expansion
	config := &Config{
		Project: "test-project",
		Runner: &RunnerConfig{
			Enabled: true,
			Env: map[string]string{
				"TOKEN":    `env("INTEGRATION_TOKEN")`,
				"PATH":     `env("INTEGRATION_PATH")`,
				"COMBINED": "${INTEGRATION_TOKEN}/path",
			},
		},
		Apps: []*AppConfig{
			{
				Name: "test-app",
				Build: &PluginConfig{
					Use: "docker",
					Config: map[string]interface{}{
						"api_token": `env("INTEGRATION_TOKEN")`,
						"base_url":  "https://example.com",
						"nested": map[string]interface{}{
							"secret": `env("INTEGRATION_TOKEN")`,
							"path":   "${INTEGRATION_PATH}",
						},
					},
				},
			},
		},
	}

	// Process environment variables
	err := processEnvVars(config)
	if err != nil {
		t.Fatalf("Failed to process env vars: %v", err)
	}

	// Verify runner env expansion
	if config.Runner.Env["TOKEN"] != "test-token-123" {
		t.Errorf("Runner env TOKEN = %q, want %q", config.Runner.Env["TOKEN"], "test-token-123")
	}
	if config.Runner.Env["PATH"] != "/test/path" {
		t.Errorf("Runner env PATH = %q, want %q", config.Runner.Env["PATH"], "/test/path")
	}
	if config.Runner.Env["COMBINED"] != "test-token-123/path" {
		t.Errorf("Runner env COMBINED = %q, want %q", config.Runner.Env["COMBINED"], "test-token-123/path")
	}

	// Verify app build config expansion
	app := config.Apps[0]
	if app.Build.Config["api_token"] != "test-token-123" {
		t.Errorf("Build config api_token = %q, want %q", app.Build.Config["api_token"], "test-token-123")
	}

	// Check nested config values
	nested, ok := app.Build.Config["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("Build config nested is not a map, got %T", app.Build.Config["nested"])
	}

	if nested["secret"] != "test-token-123" {
		t.Errorf("Nested secret = %q, want %q", nested["secret"], "test-token-123")
	}
	if nested["path"] != "/test/path" {
		t.Errorf("Nested path = %q, want %q", nested["path"], "/test/path")
	}
}

func TestParseVariableBlocks(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		envVars  map[string]string
		wantVars map[string]string
		wantErr  bool
	}{
		{
			name: "variable with env fallback",
			hcl: `
project = "test"

variable "registry_username" {
  type = "string"
  sensitive = true
  env = ["REGISTRY_USERNAME"]
}

app "test" {
  build {
    use = "railpack"
  }
  deploy {
    use = "noop"
  }
}
`,
			envVars:  map[string]string{"REGISTRY_USERNAME": "testuser"},
			wantVars: map[string]string{"registry_username": "testuser"},
			wantErr:  false,
		},
		{
			name: "variable with default",
			hcl: `
project = "test"

variable "port" {
  type = "string"
  default = "8080"
}

app "test" {
  build { use = "railpack" }
  deploy { use = "noop" }
}
`,
			envVars:  nil,
			wantVars: map[string]string{"port": "8080"},
			wantErr:  false,
		},
		{
			name: "multiple variables with env and default",
			hcl: `
project = "test"

variable "username" {
  env = ["TEST_USER"]
  default = "defaultuser"
}

variable "password" {
  env = ["TEST_PASS"]
  sensitive = true
}

app "test" {
  build { use = "railpack" }
  deploy { use = "noop" }
}
`,
			envVars: map[string]string{
				"TEST_USER": "actualuser",
				"TEST_PASS": "secretpass",
			},
			wantVars: map[string]string{
				"username": "actualuser",
				"password": "secretpass",
			},
			wantErr: false,
		},
		{
			name: "env var priority over default",
			hcl: `
project = "test"

variable "region" {
  env = ["AWS_REGION"]
  default = "us-east-1"
}

app "test" {
  build { use = "railpack" }
  deploy { use = "noop" }
}
`,
			envVars: map[string]string{
				"AWS_REGION": "eu-west-1",
			},
			wantVars: map[string]string{
				"region": "eu-west-1",
			},
			wantErr: false,
		},
		{
			name: "empty env array with default",
			hcl: `
project = "test"

variable "fallback" {
  env = []
  default = "default_value"
}

app "test" {
  build { use = "railpack" }
  deploy { use = "noop" }
}
`,
			envVars: nil,
			wantVars: map[string]string{
				"fallback": "default_value",
			},
			wantErr: false,
		},
		{
			name: "variable used in var reference",
			hcl: `
project = "test"

variable "image_name" {
  env = ["IMAGE_NAME"]
  default = "myapp"
}

app "test" {
  build {
    use = "railpack"
    name = var.image_name
  }
  deploy { use = "noop" }
}
`,
			envVars: map[string]string{
				"IMAGE_NAME": "customapp",
			},
			wantVars: map[string]string{
				"image_name": "customapp",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg, err := ParseBytes([]byte(tt.hcl), "test.hcl")
			if (err != nil) != tt.wantErr {
				t.Fatalf("parse error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Verify variable resolution
			resolved := resolveVariables(cfg.Variables)
			for k, want := range tt.wantVars {
				if got := resolved[k]; got != want {
					t.Errorf("variable %q = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestResolveVariables(t *testing.T) {
	tests := []struct {
		name     string
		vars     []*VariableConfig
		envVars  map[string]string
		expected map[string]string
	}{
		{
			name: "resolve from env",
			vars: []*VariableConfig{
				{
					Name: "test_var",
					Env:  []string{"TEST_ENV"},
				},
			},
			envVars: map[string]string{
				"TEST_ENV": "test_value",
			},
			expected: map[string]string{
				"test_var": "test_value",
			},
		},
		{
			name: "fallback to default",
			vars: []*VariableConfig{
				{
					Name:    "test_var",
					Env:     []string{"MISSING_ENV"},
					Default: "default_value",
				},
			},
			envVars: nil,
			expected: map[string]string{
				"test_var": "default_value",
			},
		},
		{
			name: "env takes priority over default",
			vars: []*VariableConfig{
				{
					Name:    "test_var",
					Env:     []string{"TEST_ENV"},
					Default: "default_value",
				},
			},
			envVars: map[string]string{
				"TEST_ENV": "env_value",
			},
			expected: map[string]string{
				"test_var": "env_value",
			},
		},
		{
			name: "multiple env options",
			vars: []*VariableConfig{
				{
					Name: "test_var",
					Env:  []string{"PRIMARY_ENV", "FALLBACK_ENV"},
				},
			},
			envVars: map[string]string{
				"FALLBACK_ENV": "fallback_value",
			},
			expected: map[string]string{
				"test_var": "fallback_value",
			},
		},
		{
			name: "nil variable in list",
			vars: []*VariableConfig{
				nil,
				{
					Name:    "valid_var",
					Default: "value",
				},
			},
			envVars: nil,
			expected: map[string]string{
				"valid_var": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := resolveVariables(tt.vars)

			for k, want := range tt.expected {
				if got := result[k]; got != want {
					t.Errorf("variable %q = %q, want %q", k, got, want)
				}
			}
		})
	}
}
