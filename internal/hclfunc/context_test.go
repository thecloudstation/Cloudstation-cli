package hclfunc

import (
	"os"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestNewEvalContext(t *testing.T) {
	t.Run("with variables", func(t *testing.T) {
		variables := map[string]string{
			"environment": "production",
			"region":      "us-west-2",
		}

		ctx := NewEvalContext(variables)

		// Check that context is not nil
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Check that variables are set under the 'var' namespace
		if ctx.Variables == nil {
			t.Fatal("Expected variables to be set")
		}

		// Check that 'var' namespace exists
		varObj, ok := ctx.Variables["var"]
		if !ok {
			t.Fatal("Expected 'var' namespace to exist")
		}

		// Check that specific variables are present and correct under 'var' namespace
		envVal := varObj.GetAttr("environment")
		if envVal.IsNull() || envVal.AsString() != "production" {
			t.Errorf("Expected var.environment to be 'production', got %v", envVal)
		}

		regionVal := varObj.GetAttr("region")
		if regionVal.IsNull() || regionVal.AsString() != "us-west-2" {
			t.Errorf("Expected var.region to be 'us-west-2', got %v", regionVal)
		}

		// Check that functions are set
		if ctx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}

		// Check that env function is present
		if _, ok := ctx.Functions["env"]; !ok {
			t.Error("Expected env function to be present")
		}
	})

	t.Run("with nil variables", func(t *testing.T) {
		ctx := NewEvalContext(nil)

		// Check that context is not nil
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Check that variables is nil (not an empty map)
		if ctx.Variables != nil {
			t.Error("Expected variables to be nil when no variables provided")
		}

		// Check that functions are still set
		if ctx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}

		// Check that all expected functions are present
		expectedFuncs := []string{"env", "lower", "upper", "concat"}
		for _, fname := range expectedFuncs {
			if _, ok := ctx.Functions[fname]; !ok {
				t.Errorf("Expected %s function to be present", fname)
			}
		}
	})

	t.Run("empty variables map", func(t *testing.T) {
		variables := map[string]string{}

		ctx := NewEvalContext(variables)

		// Check that context is not nil
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Check that variables is nil when empty map provided (same as nil)
		if ctx.Variables != nil {
			t.Error("Expected variables to be nil when empty map provided")
		}

		// Check that functions are still set
		if ctx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}
	})
}

func TestNewEvalContextWithVars(t *testing.T) {
	t.Run("creates var namespace", func(t *testing.T) {
		variables := map[string]string{
			"registry_username": "myuser",
			"registry_password": "mypass",
		}

		ctx := NewEvalContextWithVars(variables)

		// Check that context is not nil
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Check that 'var' namespace exists
		varObj, ok := ctx.Variables["var"]
		if !ok {
			t.Fatal("Expected 'var' namespace to exist")
		}

		// Check that variables are accessible via var.X syntax
		usernameVal := varObj.GetAttr("registry_username")
		if usernameVal.IsNull() || usernameVal.AsString() != "myuser" {
			t.Errorf("Expected var.registry_username to be 'myuser', got %v", usernameVal)
		}

		passwordVal := varObj.GetAttr("registry_password")
		if passwordVal.IsNull() || passwordVal.AsString() != "mypass" {
			t.Errorf("Expected var.registry_password to be 'mypass', got %v", passwordVal)
		}

		// Check that functions are set
		if ctx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}
	})

	t.Run("with empty map", func(t *testing.T) {
		variables := map[string]string{}

		ctx := NewEvalContextWithVars(variables)

		// Check that context is not nil
		if ctx == nil {
			t.Fatal("Expected non-nil context")
		}

		// Check that 'var' namespace exists (even if empty)
		_, ok := ctx.Variables["var"]
		if !ok {
			t.Fatal("Expected 'var' namespace to exist even with empty variables")
		}

		// Check that functions are set
		if ctx.Functions == nil {
			t.Fatal("Expected functions to be set")
		}
	})
}

func TestNewEvalContextWithEnv(t *testing.T) {
	ctx := NewEvalContextWithEnv()

	// Check that context is not nil
	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	// Check that variables is nil
	if ctx.Variables != nil {
		t.Error("Expected variables to be nil")
	}

	// Check that functions are set
	if ctx.Functions == nil {
		t.Fatal("Expected functions to be set")
	}

	// Check that env function is present and works
	if envFunc, ok := ctx.Functions["env"]; ok {
		// Set a test environment variable
		testVar := "TEST_HCLFUNC_VAR"
		testValue := "test_value_123"
		os.Setenv(testVar, testValue)
		defer os.Unsetenv(testVar)

		// Call the env function
		result, err := envFunc.Call([]cty.Value{cty.StringVal(testVar)})
		if err != nil {
			t.Errorf("Failed to call env function: %v", err)
		}

		if result.AsString() != testValue {
			t.Errorf("Expected env function to return %s, got %s", testValue, result.AsString())
		}
	} else {
		t.Error("Expected env function to be present")
	}
}
