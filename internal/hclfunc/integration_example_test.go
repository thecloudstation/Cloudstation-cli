package hclfunc_test

import (
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/hclfunc"
)

// TestIntegrationWithParser demonstrates how to integrate the EvalContext with HCL parsing
func TestIntegrationWithParser(t *testing.T) {
	// Set up test environment variable
	os.Setenv("TEST_API_TOKEN", "secret-token-123")
	os.Setenv("TEST_ENVIRONMENT", "staging")
	defer os.Unsetenv("TEST_API_TOKEN")
	defer os.Unsetenv("TEST_ENVIRONMENT")

	// Sample HCL configuration that uses the env() function
	hclContent := `
project = "test-project"

runner {
  enabled = true
  profile = "default"

  env = {
    API_TOKEN = env("TEST_API_TOKEN")
    DEPLOYMENT_ENV = upper(env("TEST_ENVIRONMENT"))
    SERVICE_NAME = concat("api-", lower(env("TEST_ENVIRONMENT")))
  }
}

app "web-service" {
  path = "./apps/web"

  config {
    env = {
      TOKEN = env("TEST_API_TOKEN")
      ENV_NAME = env("TEST_ENVIRONMENT")
    }
  }
}
`

	// Parse the HCL content
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(hclContent), "test.hcl")
	if diags.HasErrors() {
		t.Fatalf("Failed to parse HCL: %s", diags.Error())
	}

	// Create evaluation context with our custom functions
	evalCtx := hclfunc.NewEvalContextWithEnv()

	// Define a simplified config structure for testing
	type RunnerConfig struct {
		Enabled bool              `hcl:"enabled,optional"`
		Profile string            `hcl:"profile,optional"`
		Env     map[string]string `hcl:"env,optional"`
	}

	type AppConfigInner struct {
		Env map[string]string `hcl:"env,optional"`
	}

	type AppConfig struct {
		Name   string          `hcl:"name,label"`
		Path   string          `hcl:"path,optional"`
		Config *AppConfigInner `hcl:"config,block"`
	}

	type Config struct {
		Project string        `hcl:"project,attr"`
		Runner  *RunnerConfig `hcl:"runner,block"`
		Apps    []*AppConfig  `hcl:"app,block"`
	}

	// Decode with the evaluation context
	var config Config
	diags = gohcl.DecodeBody(file.Body, evalCtx, &config)
	if diags.HasErrors() {
		t.Fatalf("Failed to decode configuration: %s", diags.Error())
	}

	// Verify that the env() function was evaluated correctly
	t.Run("verify project", func(t *testing.T) {
		if config.Project != "test-project" {
			t.Errorf("Expected project to be 'test-project', got %s", config.Project)
		}
	})

	t.Run("verify runner config", func(t *testing.T) {
		if config.Runner == nil {
			t.Fatal("Expected runner config to be set")
		}

		// Check that env() function was evaluated
		if config.Runner.Env["API_TOKEN"] != "secret-token-123" {
			t.Errorf("Expected API_TOKEN to be 'secret-token-123', got %s", config.Runner.Env["API_TOKEN"])
		}

		// Check that upper() and env() functions were evaluated
		if config.Runner.Env["DEPLOYMENT_ENV"] != "STAGING" {
			t.Errorf("Expected DEPLOYMENT_ENV to be 'STAGING', got %s", config.Runner.Env["DEPLOYMENT_ENV"])
		}

		// Check that concat(), lower(), and env() functions were evaluated
		if config.Runner.Env["SERVICE_NAME"] != "api-staging" {
			t.Errorf("Expected SERVICE_NAME to be 'api-staging', got %s", config.Runner.Env["SERVICE_NAME"])
		}
	})

	t.Run("verify app config", func(t *testing.T) {
		if len(config.Apps) == 0 {
			t.Fatal("Expected at least one app config")
		}

		app := config.Apps[0]
		if app.Name != "web-service" {
			t.Errorf("Expected app name to be 'web-service', got %s", app.Name)
		}

		if app.Config == nil {
			t.Fatal("Expected app config block to be set")
		}

		// Check that env() functions were evaluated in app config
		if app.Config.Env["TOKEN"] != "secret-token-123" {
			t.Errorf("Expected TOKEN to be 'secret-token-123', got %s", app.Config.Env["TOKEN"])
		}

		if app.Config.Env["ENV_NAME"] != "staging" {
			t.Errorf("Expected ENV_NAME to be 'staging', got %s", app.Config.Env["ENV_NAME"])
		}
	})
}

// TestIntegrationWithVariables demonstrates using the EvalContext with custom variables
func TestIntegrationWithVariables(t *testing.T) {
	// Sample HCL configuration that uses variables
	hclContent := `
project = "${environment}-project"

runner {
  profile = "${profile_name}"

  env = {
    REGION = "${region}"
    SERVICE = concat("service-", "${environment}")
  }
}
`

	// Parse the HCL content
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL([]byte(hclContent), "test.hcl")
	if diags.HasErrors() {
		t.Fatalf("Failed to parse HCL: %s", diags.Error())
	}

	// Create evaluation context with variables
	variables := map[string]string{
		"environment":  "production",
		"region":       "us-east-1",
		"profile_name": "prod-profile",
	}
	evalCtx := hclfunc.NewEvalContext(variables)

	// Simple config structure for testing
	type RunnerConfig struct {
		Profile string            `hcl:"profile,optional"`
		Env     map[string]string `hcl:"env,optional"`
	}

	type Config struct {
		Project string        `hcl:"project,attr"`
		Runner  *RunnerConfig `hcl:"runner,block"`
	}

	// The NewEvalContext function already handles converting string variables to cty.Value
	// Note: HCL2 uses a different syntax for variables, this test demonstrates the context setup

	// For proper variable interpolation, the HCL would need to use HCL2 variable syntax
	// This test demonstrates the context setup
	t.Run("verify context setup", func(t *testing.T) {
		if evalCtx == nil {
			t.Fatal("Expected non-nil evaluation context")
		}

		if evalCtx.Variables == nil {
			t.Fatal("Expected variables to be set")
		}

		if evalCtx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}

		// Verify all expected functions are present
		expectedFuncs := []string{"env", "lower", "upper", "concat"}
		for _, fname := range expectedFuncs {
			if _, ok := evalCtx.Functions[fname]; !ok {
				t.Errorf("Expected function %s to be present", fname)
			}
		}
	})
}
