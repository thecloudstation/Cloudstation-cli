// Package hclfunc provides custom HCL functions that can be used in HCL configuration files.
// These functions are made available in the HCL evaluation context and can be called
// directly from HCL configurations.
package hclfunc

import (
	"os"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// EnvFunc returns a function that retrieves environment variable values.
// The function accepts a single string parameter (the variable name) and returns
// the value of the environment variable as a string. If the variable is not set,
// it returns an empty string.
//
// Example usage in HCL:
//
//	token = env("GITHUB_TOKEN")
func EnvFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "varname",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			varName := args[0].AsString()
			value := os.Getenv(varName)
			return cty.StringVal(value), nil
		},
	})
}

// LowerFunc returns a function that converts a string to lowercase.
// The function accepts a single string parameter and returns the
// lowercase version of that string.
//
// Example usage in HCL:
//
//	name = lower("HELLO")  // returns "hello"
func LowerFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "str",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			str := args[0].AsString()
			return cty.StringVal(strings.ToLower(str)), nil
		},
	})
}

// UpperFunc returns a function that converts a string to uppercase.
// The function accepts a single string parameter and returns the
// uppercase version of that string.
//
// Example usage in HCL:
//
//	name = upper("hello")  // returns "HELLO"
func UpperFunc() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "str",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			str := args[0].AsString()
			return cty.StringVal(strings.ToUpper(str)), nil
		},
	})
}

// ConcatFunc returns a function that concatenates multiple strings.
// The function accepts a variable number of string parameters and returns
// a single string containing all the input strings concatenated together.
//
// Example usage in HCL:
//
//	message = concat("Hello", " ", "World")  // returns "Hello World"
func ConcatFunc() function.Function {
	return function.New(&function.Spec{
		VarParam: &function.Parameter{
			Name: "values",
			Type: cty.String,
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			// Handle empty arguments
			if len(args) == 0 {
				return cty.StringVal(""), nil
			}

			// Build the concatenated string
			var builder strings.Builder
			for _, arg := range args {
				// Skip null values gracefully
				if arg.IsNull() {
					continue
				}
				builder.WriteString(arg.AsString())
			}

			return cty.StringVal(builder.String()), nil
		},
	})
}

// Functions returns a map of all available custom HCL functions.
// This map can be used to populate the functions in an HCL evaluation context.
// The returned functions include:
//   - env: Retrieve environment variable values
//   - lower: Convert strings to lowercase
//   - upper: Convert strings to uppercase
//   - concat: Concatenate multiple strings
func Functions() map[string]function.Function {
	return map[string]function.Function{
		"env":    EnvFunc(),
		"lower":  LowerFunc(),
		"upper":  UpperFunc(),
		"concat": ConcatFunc(),
	}
}
