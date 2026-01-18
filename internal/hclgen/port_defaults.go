package hclgen

const (
	// DefaultWebPort is the default port for most web frameworks (Node.js, Ruby, etc.)
	DefaultWebPort = 3000

	// GoDefaultPort is the default port for Go applications
	GoDefaultPort = 8080

	// PythonDefaultPort is the default port for Python applications
	PythonDefaultPort = 8000

	// JavaDefaultPort is the default port for Java applications
	JavaDefaultPort = 8080
)

// GetFrameworkDefault returns the default port based on the builder type
func GetFrameworkDefault(builderType string) int {
	switch builderType {
	case "nixpacks":
		// Nixpacks typically builds Node.js apps which default to 3000
		return DefaultWebPort
	case "railpack":
		// Railpack is the successor to Nixpacks, uses same default
		return DefaultWebPort
	case "csdocker":
		// Docker builds can be anything, but 8000 is a common default
		return PythonDefaultPort
	default:
		// Default fallback
		return DefaultWebPort
	}
}

// DetectFrameworkFromMetadata inspects artifact metadata to determine the framework
// This is a future enhancement placeholder for more intelligent framework detection
func DetectFrameworkFromMetadata(metadata map[string]interface{}) string {
	// Future enhancement: Parse nixpacks plan.json or detect from build output
	// For now, return empty string to indicate no framework detected
	return ""
}
