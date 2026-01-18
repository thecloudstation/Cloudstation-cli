package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	hclfunc "github.com/thecloudstation/cloudstation-orchestrator/internal/hclfunc"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
	"github.com/zclconf/go-cty/cty"
)

// ParseFile parses an HCL configuration file and returns a Config struct
func ParseFile(path string) (*Config, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", absPath)
	}

	// Parse HCL file
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(absPath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL file: %s", diags.Error())
	}

	// PASS 1: Decode with empty context to extract variable definitions
	var partialConfig Config
	emptyCtx := hclfunc.NewEvalContext(nil)
	diags = gohcl.DecodeBody(file.Body, emptyCtx, &partialConfig)
	// Note: We may get diagnostics here due to unresolved var.X references
	// This is expected - we only need the variable definitions from pass 1

	// Resolve variables from their definitions
	resolvedVars := resolveVariables(partialConfig.Variables)

	// PASS 2: Re-decode with resolved variables in context
	var config Config
	evalCtx := hclfunc.NewEvalContextWithVars(resolvedVars)
	diags = gohcl.DecodeBody(file.Body, evalCtx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode configuration: %s", diags.Error())
	}

	// Process environment variable references
	if err := processEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return &config, nil
}

// ParseBytes parses HCL configuration from a byte slice
func ParseBytes(data []byte, filename string) (*Config, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	// PASS 1: Decode with empty context to extract variable definitions
	var partialConfig Config
	emptyCtx := hclfunc.NewEvalContext(nil)
	diags = gohcl.DecodeBody(file.Body, emptyCtx, &partialConfig)
	// Note: We may get diagnostics here due to unresolved var.X references
	// This is expected - we only need the variable definitions from pass 1

	// Resolve variables from their definitions
	resolvedVars := resolveVariables(partialConfig.Variables)

	// PASS 2: Re-decode with resolved variables in context
	var config Config
	evalCtx := hclfunc.NewEvalContextWithVars(resolvedVars)
	diags = gohcl.DecodeBody(file.Body, evalCtx, &config)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to decode configuration: %s", diags.Error())
	}

	if err := processEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return &config, nil
}

// processEnvVars processes environment variable references in the configuration
// This is a simplified version that handles env("VAR_NAME") syntax
func processEnvVars(config *Config) error {
	// Process runner env vars
	if config.Runner != nil && config.Runner.Env != nil {
		for key, value := range config.Runner.Env {
			config.Runner.Env[key] = expandEnvVars(value)
		}
	}

	// Process app configurations
	for _, app := range config.Apps {
		// Process build config
		if app.Build != nil {
			app.Build.Config = processConfigMap(app.Build.Config)
		}

		// Process registry config
		if app.Registry != nil {
			app.Registry.Config = processConfigMap(app.Registry.Config)
		}

		// Process deploy config
		if app.Deploy != nil {
			app.Deploy.Config = processConfigMap(app.Deploy.Config)
		}

		// Process release config
		if app.Release != nil {
			app.Release.Config = processConfigMap(app.Release.Config)
		}

		// Process app-specific config
		if app.Config != nil && app.Config.Env != nil {
			for key, value := range app.Config.Env {
				app.Config.Env[key] = expandEnvVars(value)
			}
		}
	}

	return nil
}

// resolveVariables resolves variable values from their definitions
// It checks environment variables specified in the Env field
func resolveVariables(variables []*VariableConfig) map[string]string {
	resolved := make(map[string]string)

	for _, v := range variables {
		if v == nil {
			continue
		}

		var value string

		// Check environment variables in order
		for _, envName := range v.Env {
			if envVal := os.Getenv(envName); envVal != "" {
				value = envVal
				break
			}
		}

		// Fall back to default if no env var found
		if value == "" && v.Default != "" {
			value = v.Default
		}

		resolved[v.Name] = value
	}

	return resolved
}

// processConfigMap processes a configuration map and expands environment variables
func processConfigMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range m {
		switch v := value.(type) {
		case string:
			result[key] = expandEnvVars(v)
		case map[string]interface{}:
			result[key] = processConfigMap(v)
		case []interface{}:
			result[key] = processSlice(v)
		case *hcl.Attribute:
			// Evaluate HCL attribute with EvalContext that has env() function
			evalCtx := hclfunc.NewEvalContext(nil)
			val, diags := v.Expr.Value(evalCtx)
			if !diags.HasErrors() && !val.IsNull() {
				// Convert cty.Value to Go type and expand any remaining env vars
				goVal := ctyToGo(val)
				if strVal, ok := goVal.(string); ok {
					result[key] = expandEnvVars(strVal)
				} else {
					result[key] = goVal
				}
			}
		default:
			result[key] = value
		}
	}
	return result
}

// processSlice processes a slice and expands environment variables
func processSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, value := range s {
		switch v := value.(type) {
		case string:
			result[i] = expandEnvVars(v)
		case map[string]interface{}:
			result[i] = processConfigMap(v)
		case []interface{}:
			result[i] = processSlice(v)
		default:
			result[i] = value
		}
	}
	return result
}

// expandEnvVars expands environment variable references in a string
// Supports both ${VAR} and env("VAR") syntax
func expandEnvVars(s string) string {
	// Handle env("VAR") syntax
	if strings.Contains(s, "env(") {
		// Match env("VAR_NAME") or env('VAR_NAME')
		re := regexp.MustCompile(`env\(["']([^"']+)["']\)`)
		s = re.ReplaceAllStringFunc(s, func(match string) string {
			// Extract variable name from env("VAR") or env('VAR')
			submatch := re.FindStringSubmatch(match)
			if len(submatch) > 1 {
				return os.Getenv(submatch[1])
			}
			return match
		})
	}

	// Handle ${VAR} syntax
	return os.ExpandEnv(s)
}

// LoadConfigFile is a convenience function that parses and validates a config file
func LoadConfigFile(path string) (*Config, error) {
	config, err := ParseFile(path)
	if err != nil {
		return nil, err
	}

	if err := Validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

// GenerateDefaultConfig creates a default configuration for zero-config builds
// when no cloudstation.hcl file exists. It uses auto-detection for project name
// and builder type.
func GenerateDefaultConfig(rootDir string) (*Config, error) {
	// Detect project name from git remote or directory
	projectName := DetectProjectName()
	if projectName == "" {
		projectName = "my-app"
	}

	// Detect builder type (railpack or csdocker)
	detection := detect.DetectBuilder(rootDir)

	// Create synthetic config
	cfg := &Config{
		Project: projectName,
		Apps: []*AppConfig{
			{
				Name: projectName,
				Build: &PluginConfig{
					Use: detection.Builder,
					Config: map[string]interface{}{
						"name":    projectName,
						"tag":     "latest",
						"context": ".",
					},
				},
				// Deploy block is optional for build-only operations
				Deploy: &PluginConfig{
					Use: "nomad-pack",
					Config: map[string]interface{}{
						"pack": "cloudstation",
					},
				},
			},
		},
	}

	return cfg, nil
}

// evaluateExpr evaluates an HCL expression and returns its value
// This is used for more complex expression evaluation
func evaluateExpr(expr hcl.Expression, ctx *hcl.EvalContext) (interface{}, error) {
	val, diags := expr.Value(ctx)
	if diags.HasErrors() {
		return nil, fmt.Errorf("expression evaluation failed: %s", diags.Error())
	}

	// Convert cty.Value to Go types
	if val.IsNull() {
		return nil, nil
	}

	return ctyToGo(val), nil
}

// ctyToGo converts a cty.Value to a Go type
func ctyToGo(val cty.Value) interface{} {
	if val.IsNull() {
		return nil
	}

	t := val.Type()

	// Handle string type
	if t.Equals(cty.String) {
		return val.AsString()
	}

	// Handle number type
	if t.Equals(cty.Number) {
		f, _ := val.AsBigFloat().Float64()
		return f
	}

	// Handle bool type
	if t.Equals(cty.Bool) {
		return val.True()
	}

	// Handle list/tuple types
	if t.IsListType() || t.IsTupleType() {
		var result []interface{}
		iter := val.ElementIterator()
		for iter.Next() {
			_, elemVal := iter.Element()
			result = append(result, ctyToGo(elemVal))
		}
		return result
	}

	// Handle map/object types
	if t.IsMapType() || t.IsObjectType() {
		result := make(map[string]interface{})
		iter := val.ElementIterator()
		for iter.Next() {
			keyVal, elemVal := iter.Element()
			key := keyVal.AsString()
			result[key] = ctyToGo(elemVal)
		}
		return result
	}

	// Fallback to string representation
	return val.AsString()
}
