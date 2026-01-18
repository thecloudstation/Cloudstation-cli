// Package hclfunc provides utilities to create HCL EvalContext instances with custom functions.
// The EvalContext is passed to HCL's DecodeBody to enable function evaluation during configuration parsing.
package hclfunc

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// NewEvalContext creates a new HCL evaluation context with custom functions and optional variables.
// The context can be used during HCL parsing to enable function evaluation and variable substitution.
//
// Parameters:
//   - variables: Optional map of string variables that will be converted to cty.Value and made
//     available in the evaluation context under the 'var' namespace. Can be nil if no variables are needed.
//
// Returns:
//   - *hcl.EvalContext: An evaluation context containing the custom functions from Functions()
//     and any provided variables exposed under the 'var' namespace.
//
// Example usage:
//
//	// With variables
//	vars := map[string]string{
//	    "environment": "production",
//	    "region": "us-west-2",
//	}
//	ctx := NewEvalContext(vars)
//	diags := gohcl.DecodeBody(file.Body, ctx, &config)
//
//	// Without variables
//	ctx := NewEvalContext(nil)
//	diags := gohcl.DecodeBody(file.Body, ctx, &config)
func NewEvalContext(variables map[string]string) *hcl.EvalContext {
	if variables == nil || len(variables) == 0 {
		return &hcl.EvalContext{
			Functions: Functions(),
		}
	}

	// Use NewEvalContextWithVars for consistency when variables are provided
	return NewEvalContextWithVars(variables)
}

// NewEvalContextWithVars creates an evaluation context with variables
// exposed under the 'var' namespace for HCL2 variable reference syntax.
// This enables var.registry_username syntax in HCL expressions.
//
// Parameters:
//   - variables: Map of variable names to their string values
//
// Returns:
//   - *hcl.EvalContext: An evaluation context with variables under 'var' namespace
//
// Example usage:
//
//	vars := map[string]string{
//	    "registry_username": "myuser",
//	    "registry_password": "mypass",
//	}
//	ctx := NewEvalContextWithVars(vars)
//	// Now var.registry_username will resolve to "myuser" in HCL expressions
func NewEvalContextWithVars(variables map[string]string) *hcl.EvalContext {
	// Convert string variables to cty.Value
	varMap := make(map[string]cty.Value)
	for k, v := range variables {
		varMap[k] = cty.StringVal(v)
	}

	// Create 'var' object to hold all variables
	// This enables var.registry_username syntax
	varsObject := cty.ObjectVal(varMap)

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": varsObject,
		},
		Functions: Functions(),
	}
}

// NewEvalContextWithEnv creates a new HCL evaluation context with custom functions but no variables.
// This is a convenience function for simple use cases where no variables are needed.
//
// Returns:
//   - *hcl.EvalContext: An evaluation context containing only the custom functions from Functions()
//
// Example usage:
//
//	ctx := NewEvalContextWithEnv()
//	diags := gohcl.DecodeBody(file.Body, ctx, &config)
//
// This is equivalent to calling NewEvalContext(nil).
func NewEvalContextWithEnv() *hcl.EvalContext {
	return NewEvalContext(nil)
}
