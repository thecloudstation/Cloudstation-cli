package config

import (
	"fmt"
	"strings"
)

// Validate validates a configuration and returns an error if invalid
func Validate(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate project name
	if config.Project == "" {
		return fmt.Errorf("project name is required")
	}

	// Validate project name format
	if !isValidName(config.Project) {
		return fmt.Errorf("project name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate at least one app is defined
	if len(config.Apps) == 0 {
		return fmt.Errorf("at least one app must be defined")
	}

	// Validate each app
	for i, app := range config.Apps {
		if err := validateApp(app, i); err != nil {
			return fmt.Errorf("app %q validation failed: %w", app.Name, err)
		}
	}

	// Check for duplicate app names
	if err := checkDuplicateAppNames(config.Apps); err != nil {
		return err
	}

	// Validate runner config if present
	if config.Runner != nil {
		if err := validateRunner(config.Runner); err != nil {
			return fmt.Errorf("runner validation failed: %w", err)
		}
	}

	return nil
}

// validateApp validates an app configuration
func validateApp(app *AppConfig, index int) error {
	// Validate app name
	if app.Name == "" {
		return fmt.Errorf("app name is required (app at index %d)", index)
	}

	if !isValidName(app.Name) {
		return fmt.Errorf("app name must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Validate build configuration
	if app.Build == nil {
		return fmt.Errorf("build configuration is required")
	}

	if err := validatePlugin(app.Build, "build"); err != nil {
		return err
	}

	// Registry is optional
	if app.Registry != nil {
		if err := validatePlugin(app.Registry, "registry"); err != nil {
			return err
		}
	}

	// Validate deploy configuration
	if app.Deploy == nil {
		return fmt.Errorf("deploy configuration is required")
	}

	if err := validatePlugin(app.Deploy, "deploy"); err != nil {
		return err
	}

	// Release is optional
	if app.Release != nil {
		if err := validatePlugin(app.Release, "release"); err != nil {
			return err
		}
	}

	return nil
}

// validatePlugin validates a plugin configuration
func validatePlugin(plugin *PluginConfig, pluginType string) error {
	if plugin.Use == "" {
		return fmt.Errorf("%s plugin 'use' field is required", pluginType)
	}

	if !isValidPluginName(plugin.Use) {
		return fmt.Errorf("invalid %s plugin name: %q", pluginType, plugin.Use)
	}

	return nil
}

// validateRunner validates runner configuration
func validateRunner(runner *RunnerConfig) error {
	// Runner validation is minimal for now
	// Add more specific validation as needed
	return nil
}

// checkDuplicateAppNames checks for duplicate app names
func checkDuplicateAppNames(apps []*AppConfig) error {
	seen := make(map[string]bool)
	for _, app := range apps {
		if seen[app.Name] {
			return fmt.Errorf("duplicate app name: %q", app.Name)
		}
		seen[app.Name] = true
	}
	return nil
}

// isValidName checks if a name contains only valid characters
func isValidName(name string) bool {
	if name == "" {
		return false
	}

	for _, ch := range name {
		if !isAlphaNumericOrDash(ch) {
			return false
		}
	}

	return true
}

// isValidPluginName checks if a plugin name is valid
func isValidPluginName(name string) bool {
	if name == "" {
		return false
	}

	// Plugin names can contain alphanumeric characters, hyphens, and underscores
	for _, ch := range name {
		if !isAlphaNumericOrDash(ch) && ch != '_' {
			return false
		}
	}

	return true
}

// isAlphaNumericOrDash checks if a character is alphanumeric or a dash
func isAlphaNumericOrDash(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' ||
		ch == '_'
}

// ValidatePluginExists checks if a plugin exists in the registry
// This will be called after the plugin registry is initialized
func ValidatePluginExists(config *Config, pluginRegistry PluginChecker) error {
	for _, app := range config.Apps {
		// Check build plugin
		if !pluginRegistry.HasBuilder(app.Build.Use) {
			return fmt.Errorf("app %q: unknown build plugin: %q", app.Name, app.Build.Use)
		}

		// Check registry plugin (if specified)
		if app.Registry != nil && app.Registry.Use != "" {
			if !pluginRegistry.HasRegistry(app.Registry.Use) {
				return fmt.Errorf("app %q: unknown registry plugin: %q", app.Name, app.Registry.Use)
			}
		}

		// Check deploy plugin
		if !pluginRegistry.HasPlatform(app.Deploy.Use) {
			return fmt.Errorf("app %q: unknown deploy plugin: %q", app.Name, app.Deploy.Use)
		}

		// Check release plugin (if specified)
		if app.Release != nil && app.Release.Use != "" {
			// Release plugins are optional and checked if registry supports them
			// This is a future enhancement
		}
	}

	return nil
}

// PluginChecker interface for checking plugin existence
type PluginChecker interface {
	HasBuilder(name string) bool
	HasRegistry(name string) bool
	HasPlatform(name string) bool
}

// FormatError formats a validation error with helpful context
func FormatError(err error, configPath string) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Configuration validation failed:\n")
	sb.WriteString(fmt.Sprintf("  File: %s\n", configPath))
	sb.WriteString(fmt.Sprintf("  Error: %s\n", err.Error()))

	return sb.String()
}
