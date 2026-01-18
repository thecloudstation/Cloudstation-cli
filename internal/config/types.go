package config

// Config represents the root configuration structure
type Config struct {
	// Project is the project name
	Project string `hcl:"project,attr"`

	// Runner contains runner configuration
	Runner *RunnerConfig `hcl:"runner,block"`

	// Apps contains application configurations
	Apps []*AppConfig `hcl:"app,block"`

	// Variables contains variable definitions
	Variables []*VariableConfig `hcl:"variable,block"`
}

// VariableConfig represents an HCL variable block definition
type VariableConfig struct {
	// Name is the variable name (block label)
	Name string `hcl:"name,label"`

	// Type is the variable type (string, number, bool, etc.)
	Type string `hcl:"type,optional"`

	// Sensitive marks the variable as sensitive (suppresses logging)
	Sensitive bool `hcl:"sensitive,optional"`

	// Default is the default value if not provided
	Default string `hcl:"default,optional"`

	// Env is a list of environment variable names to check for value
	Env []string `hcl:"env,optional"`

	// Description documents the variable purpose
	Description string `hcl:"description,optional"`
}

// RunnerConfig represents runner configuration
type RunnerConfig struct {
	// Enabled indicates if the runner is enabled
	Enabled bool `hcl:"enabled,optional"`

	// Profile is the runner profile name
	Profile string `hcl:"profile,optional"`

	// DataSource contains data source configuration
	DataSource *DataSourceConfig `hcl:"data_source,block"`

	// Environment variables
	Env map[string]string `hcl:"env,optional"`
}

// DataSourceConfig represents data source configuration
type DataSourceConfig struct {
	// Type is the data source type (e.g., "git")
	Type string `hcl:"type,attr"`

	// Config contains type-specific configuration
	Config map[string]interface{} `hcl:",remain"`
}

// AppConfig represents an application configuration
type AppConfig struct {
	// Name is the application name
	Name string `hcl:"name,label"`

	// Path is the application path (optional)
	Path string `hcl:"path,optional"`

	// Labels are metadata labels
	Labels map[string]string `hcl:"labels,optional"`

	// Build contains build configuration
	Build *PluginConfig `hcl:"build,block"`

	// Registry contains registry configuration
	Registry *PluginConfig `hcl:"registry,block"`

	// Deploy contains deployment configuration
	Deploy *PluginConfig `hcl:"deploy,block"`

	// Release contains release configuration (optional)
	Release *PluginConfig `hcl:"release,block"`

	// URL is the application URL template
	URL *URLConfig `hcl:"url,block"`

	// Config contains app-specific configuration
	Config *AppSpecificConfig `hcl:"config,block"`
}

// PluginConfig represents a plugin configuration block
type PluginConfig struct {
	// Use specifies the plugin to use
	Use string `hcl:"use,attr"`

	// Config contains plugin-specific configuration
	Config map[string]interface{} `hcl:",remain"`
}

// URLConfig represents URL configuration
type URLConfig struct {
	// AutoHostname indicates if hostname should be auto-generated
	AutoHostname bool `hcl:"auto_hostname,optional"`

	// Path is the URL path
	Path string `hcl:"path,optional"`

	// Port is the port number
	Port int `hcl:"port,optional"`
}

// AppSpecificConfig contains app-specific configuration
type AppSpecificConfig struct {
	// Env contains environment variables
	Env map[string]string `hcl:"env,optional"`

	// InternalPort is the internal port the app listens on
	InternalPort int `hcl:"internal_port,optional"`

	// Other configuration as key-value pairs
	Config map[string]interface{} `hcl:",remain"`
}

// GetApp returns an app configuration by name
func (c *Config) GetApp(name string) *AppConfig {
	for _, app := range c.Apps {
		if app.Name == name {
			return app
		}
	}
	return nil
}

// ListAppNames returns a list of all app names
func (c *Config) ListAppNames() []string {
	names := make([]string, len(c.Apps))
	for i, app := range c.Apps {
		names[i] = app.Name
	}
	return names
}
