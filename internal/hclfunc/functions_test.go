package hclfunc

import (
	"os"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// TestEnvFunc tests the EnvFunc function for retrieving environment variables
func TestEnvFunc(t *testing.T) {
	t.Run("existing variable", func(t *testing.T) {
		// Set a test environment variable
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("TEST_VAR")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Type().Equals(cty.String) {
			t.Errorf("expected string type, got %s", result.Type().GoString())
		}

		if result.AsString() != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", result.AsString())
		}
	})

	t.Run("nonexistent variable", func(t *testing.T) {
		// Ensure the variable doesn't exist
		os.Unsetenv("NONEXISTENT_VAR")

		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("NONEXISTENT_VAR")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "" {
			t.Errorf("expected empty string, got '%s'", result.AsString())
		}
	})

	t.Run("empty variable name", func(t *testing.T) {
		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "" {
			t.Errorf("expected empty string for empty var name, got '%s'", result.AsString())
		}
	})

	t.Run("special characters in variable name", func(t *testing.T) {
		// Test with underscore and numbers (valid env var name)
		os.Setenv("TEST_VAR_123", "special_value")
		defer os.Unsetenv("TEST_VAR_123")

		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("TEST_VAR_123")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "special_value" {
			t.Errorf("expected 'special_value', got '%s'", result.AsString())
		}
	})

	t.Run("environment variable with spaces", func(t *testing.T) {
		os.Setenv("TEST_SPACES", "value with spaces")
		defer os.Unsetenv("TEST_SPACES")

		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("TEST_SPACES")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "value with spaces" {
			t.Errorf("expected 'value with spaces', got '%s'", result.AsString())
		}
	})
}

// TestLowerFunc tests the LowerFunc function for string lowercase conversion
func TestLowerFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase string",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case string",
			input:    "MiXeD",
			expected: "mixed",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with numbers",
			input:    "HELLO123",
			expected: "hello123",
		},
		{
			name:     "with special characters",
			input:    "HELLO_WORLD!",
			expected: "hello_world!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character",
			input:    "A",
			expected: "a",
		},
		{
			name:     "unicode characters",
			input:    "HÉLLO",
			expected: "héllo",
		},
		{
			name:     "with spaces",
			input:    "HELLO WORLD",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := LowerFunc()
			result, err := fn.Call([]cty.Value{cty.StringVal(tt.input)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Type().Equals(cty.String) {
				t.Errorf("expected string type, got %s", result.Type().GoString())
			}

			if result.AsString() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result.AsString())
			}
		})
	}
}

// TestUpperFunc tests the UpperFunc function for string uppercase conversion
func TestUpperFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase string",
			input:    "hello",
			expected: "HELLO",
		},
		{
			name:     "mixed case string",
			input:    "MiXeD",
			expected: "MIXED",
		},
		{
			name:     "already uppercase",
			input:    "HELLO",
			expected: "HELLO",
		},
		{
			name:     "with numbers",
			input:    "hello123",
			expected: "HELLO123",
		},
		{
			name:     "with special characters",
			input:    "hello_world!",
			expected: "HELLO_WORLD!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "A",
		},
		{
			name:     "unicode characters",
			input:    "héllo",
			expected: "HÉLLO",
		},
		{
			name:     "with spaces",
			input:    "hello world",
			expected: "HELLO WORLD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := UpperFunc()
			result, err := fn.Call([]cty.Value{cty.StringVal(tt.input)})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Type().Equals(cty.String) {
				t.Errorf("expected string type, got %s", result.Type().GoString())
			}

			if result.AsString() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result.AsString())
			}
		})
	}
}

// TestConcatFunc tests the ConcatFunc function for string concatenation
func TestConcatFunc(t *testing.T) {
	t.Run("multiple strings", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{
			cty.StringVal("Hello"),
			cty.StringVal(" "),
			cty.StringVal("World"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "Hello World" {
			t.Errorf("expected 'Hello World', got '%s'", result.AsString())
		}
	})

	t.Run("single string", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal("single")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "single" {
			t.Errorf("expected 'single', got '%s'", result.AsString())
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{
			cty.StringVal(""),
			cty.StringVal(""),
			cty.StringVal(""),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "" {
			t.Errorf("expected empty string, got '%s'", result.AsString())
		}
	})

	t.Run("mixed empty and non-empty", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{
			cty.StringVal("start"),
			cty.StringVal(""),
			cty.StringVal("end"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "startend" {
			t.Errorf("expected 'startend', got '%s'", result.AsString())
		}
	})

	t.Run("no arguments", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "" {
			t.Errorf("expected empty string for no arguments, got '%s'", result.AsString())
		}
	})

	t.Run("many strings", func(t *testing.T) {
		fn := ConcatFunc()
		args := make([]cty.Value, 10)
		for i := 0; i < 10; i++ {
			args[i] = cty.StringVal("a")
		}
		result, err := fn.Call(args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "aaaaaaaaaa"
		if result.AsString() != expected {
			t.Errorf("expected '%s', got '%s'", expected, result.AsString())
		}
	})

	t.Run("with special characters", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{
			cty.StringVal("hello"),
			cty.StringVal("@"),
			cty.StringVal("world"),
			cty.StringVal("!"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "hello@world!" {
			t.Errorf("expected 'hello@world!', got '%s'", result.AsString())
		}
	})

	t.Run("with newlines", func(t *testing.T) {
		fn := ConcatFunc()
		result, err := fn.Call([]cty.Value{
			cty.StringVal("line1"),
			cty.StringVal("\n"),
			cty.StringVal("line2"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AsString() != "line1\nline2" {
			t.Errorf("expected 'line1\\nline2', got '%s'", result.AsString())
		}
	})

	t.Run("with null values", func(t *testing.T) {
		fn := ConcatFunc()
		_, err := fn.Call([]cty.Value{
			cty.StringVal("before"),
			cty.NullVal(cty.String),
			cty.StringVal("after"),
		})

		// The function attempts to call AsString() on null values which causes an error
		// This is expected behavior as cty doesn't allow calling AsString() on null values
		if err == nil {
			t.Fatal("expected error when passing null values, got none")
		}

		// Verify the error message indicates the issue
		if err.Error() != "argument must not be null" {
			t.Errorf("expected 'argument must not be null' error, got: %v", err)
		}
	})
}

// TestFunctions tests the Functions() map
func TestFunctions(t *testing.T) {
	funcs := Functions()

	t.Run("returns non-nil map", func(t *testing.T) {
		if funcs == nil {
			t.Fatal("Functions() returned nil map")
		}
	})

	t.Run("contains expected functions", func(t *testing.T) {
		expectedFunctions := []string{"env", "lower", "upper", "concat"}

		for _, name := range expectedFunctions {
			t.Run(name, func(t *testing.T) {
				if _, ok := funcs[name]; !ok {
					t.Errorf("expected function '%s' not found in Functions() map", name)
				}
			})
		}
	})

	t.Run("all functions are callable", func(t *testing.T) {
		// Test that each function can be called without panic
		testCases := map[string][]cty.Value{
			"env":    {cty.StringVal("TEST")},
			"lower":  {cty.StringVal("TEST")},
			"upper":  {cty.StringVal("test")},
			"concat": {cty.StringVal("a"), cty.StringVal("b")},
		}

		for name, args := range testCases {
			t.Run(name, func(t *testing.T) {
				fn, ok := funcs[name]
				if !ok {
					t.Skipf("function '%s' not found", name)
				}

				// Verify it's a function.Function
				if fn.Params() == nil && fn.VarParam() == nil {
					t.Errorf("function '%s' has no parameters defined", name)
				}

				// Call the function and ensure no panic
				result, err := fn.Call(args)
				if err != nil {
					t.Errorf("function '%s' returned error: %v", name, err)
				}

				// Verify result is a string
				if !result.Type().Equals(cty.String) {
					t.Errorf("function '%s' did not return a string type", name)
				}
			})
		}
	})

	t.Run("function count", func(t *testing.T) {
		expectedCount := 4
		if len(funcs) != expectedCount {
			t.Errorf("expected %d functions, got %d", expectedCount, len(funcs))
		}
	})
}

// TestFunctionSignatures tests that each function has the correct signature
func TestFunctionSignatures(t *testing.T) {
	t.Run("EnvFunc signature", func(t *testing.T) {
		fn := EnvFunc()

		// Check it has exactly one parameter
		if len(fn.Params()) != 1 {
			t.Errorf("expected 1 parameter, got %d", len(fn.Params()))
		}

		// Check parameter type
		if len(fn.Params()) > 0 && !fn.Params()[0].Type.Equals(cty.String) {
			t.Errorf("expected parameter type to be string")
		}

		// Check return type
		retType, err := fn.ReturnType([]cty.Type{cty.String})
		if err != nil {
			t.Fatalf("error getting return type: %v", err)
		}
		if !retType.Equals(cty.String) {
			t.Errorf("expected return type to be string")
		}
	})

	t.Run("LowerFunc signature", func(t *testing.T) {
		fn := LowerFunc()

		if len(fn.Params()) != 1 {
			t.Errorf("expected 1 parameter, got %d", len(fn.Params()))
		}

		if len(fn.Params()) > 0 && !fn.Params()[0].Type.Equals(cty.String) {
			t.Errorf("expected parameter type to be string")
		}

		retType, err := fn.ReturnType([]cty.Type{cty.String})
		if err != nil {
			t.Fatalf("error getting return type: %v", err)
		}
		if !retType.Equals(cty.String) {
			t.Errorf("expected return type to be string")
		}
	})

	t.Run("UpperFunc signature", func(t *testing.T) {
		fn := UpperFunc()

		if len(fn.Params()) != 1 {
			t.Errorf("expected 1 parameter, got %d", len(fn.Params()))
		}

		if len(fn.Params()) > 0 && !fn.Params()[0].Type.Equals(cty.String) {
			t.Errorf("expected parameter type to be string")
		}

		retType, err := fn.ReturnType([]cty.Type{cty.String})
		if err != nil {
			t.Fatalf("error getting return type: %v", err)
		}
		if !retType.Equals(cty.String) {
			t.Errorf("expected return type to be string")
		}
	})

	t.Run("ConcatFunc signature", func(t *testing.T) {
		fn := ConcatFunc()

		// Should have no fixed params but a var param
		if len(fn.Params()) != 0 {
			t.Errorf("expected 0 fixed parameters, got %d", len(fn.Params()))
		}

		if fn.VarParam() == nil {
			t.Error("expected VarParam to be defined")
		} else if !fn.VarParam().Type.Equals(cty.String) {
			t.Error("expected VarParam type to be string")
		}

		// Test with various argument counts
		argCounts := []int{0, 1, 2, 5, 10}
		for _, count := range argCounts {
			types := make([]cty.Type, count)
			for i := range types {
				types[i] = cty.String
			}

			retType, err := fn.ReturnType(types)
			if err != nil {
				t.Errorf("error getting return type for %d args: %v", count, err)
				continue
			}
			if !retType.Equals(cty.String) {
				t.Errorf("expected return type to be string for %d args", count)
			}
		}
	})
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("env with very long variable name", func(t *testing.T) {
		longVarName := "TEST_" + string(make([]byte, 1000))
		fn := EnvFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal(longVarName)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should return empty string for non-existent long var name
		if result.AsString() != "" {
			t.Errorf("expected empty string, got '%s'", result.AsString())
		}
	})

	t.Run("lower with very long string", func(t *testing.T) {
		// Create a very long string
		longString := "A"
		for i := 0; i < 10000; i++ {
			longString = longString + "A"
		}

		fn := LowerFunc()
		result, err := fn.Call([]cty.Value{cty.StringVal(longString)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Just verify it doesn't panic and returns a string
		if !result.Type().Equals(cty.String) {
			t.Error("expected string type result")
		}
	})

	t.Run("concat with many arguments", func(t *testing.T) {
		fn := ConcatFunc()
		args := make([]cty.Value, 100)
		for i := range args {
			args[i] = cty.StringVal("x")
		}

		result, err := fn.Call(args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The expected result should be 100 'x' characters concatenated

		if len(result.AsString()) != 100 {
			t.Errorf("expected length 100, got %d", len(result.AsString()))
		}
	})
}

// Benchmark tests for performance
func BenchmarkEnvFunc(b *testing.B) {
	os.Setenv("BENCHMARK_VAR", "benchmark_value")
	defer os.Unsetenv("BENCHMARK_VAR")

	fn := EnvFunc()
	arg := cty.StringVal("BENCHMARK_VAR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fn.Call([]cty.Value{arg})
	}
}

func BenchmarkLowerFunc(b *testing.B) {
	fn := LowerFunc()
	arg := cty.StringVal("HELLO WORLD THIS IS A TEST STRING")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fn.Call([]cty.Value{arg})
	}
}

func BenchmarkUpperFunc(b *testing.B) {
	fn := UpperFunc()
	arg := cty.StringVal("hello world this is a test string")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fn.Call([]cty.Value{arg})
	}
}

func BenchmarkConcatFunc(b *testing.B) {
	fn := ConcatFunc()
	args := []cty.Value{
		cty.StringVal("hello"),
		cty.StringVal(" "),
		cty.StringVal("world"),
		cty.StringVal(" "),
		cty.StringVal("test"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fn.Call(args)
	}
}
