# HCL Functions Package

This package provides custom HCL functions and evaluation context utilities for the CloudStation Orchestrator configuration parser.

## Components

### context.go
Provides utilities to create HCL EvalContext instances with custom functions and variables.

**Functions:**
- `NewEvalContext(variables map[string]string) *hcl.EvalContext` - Creates an evaluation context with optional variables
- `NewEvalContextWithEnv() *hcl.EvalContext` - Convenience function for creating context without variables

### functions.go
Implements custom HCL functions that can be used in configuration files.

**Available Functions:**
- `env(varname)` - Retrieves environment variable values
- `lower(str)` - Converts strings to lowercase
- `upper(str)` - Converts strings to uppercase
- `concat(values...)` - Concatenates multiple strings

## Usage Example

### In Parser (config/parser.go)

To integrate the evaluation context with the HCL parser, update the ParseFile function:

```go
import (
    "github.com/thecloudstation/cloudstation-orchestrator/internal/hclfunc"
)

func ParseFile(path string) (*Config, error) {
    // ... existing parsing code ...

    // Create evaluation context with custom functions
    evalCtx := hclfunc.NewEvalContextWithEnv()

    // Decode with the evaluation context
    var config Config
    diags = gohcl.DecodeBody(file.Body, evalCtx, &config)
    if diags.HasErrors() {
        return nil, fmt.Errorf("failed to decode configuration: %s", diags.Error())
    }

    // ... rest of the function ...
}
```

### In HCL Configuration Files

Once integrated, users can use the custom functions in their HCL configuration:

```hcl
project = "my-project"

runner {
  enabled = true
  profile = "default"

  env = {
    API_TOKEN = env("GITHUB_TOKEN")
    DEPLOYMENT_ENV = upper(env("ENVIRONMENT"))
    SERVICE_NAME = concat("api-", lower(env("ENVIRONMENT")))
  }
}

app "web-service" {
  path = "./apps/web"

  config {
    env = {
      TOKEN = env("API_TOKEN")
      ENV_NAME = env("ENVIRONMENT")
    }
  }
}
```

## Testing

The package includes comprehensive tests:

- `context_test.go` - Tests for EvalContext creation
- `functions_test.go` - Tests for individual HCL functions
- `integration_example_test.go` - Integration examples showing usage with HCL parser

Run tests with:
```bash
go test ./internal/hclfunc/...
```

## Integration Notes

1. The `processEnvVars` function in `parser.go` can be simplified or removed once the evaluation context is integrated, as the env() function will handle environment variable expansion during HCL parsing.

2. The evaluation context supports both functions and variables, allowing for flexible configuration processing.

3. All functions handle edge cases gracefully (e.g., empty strings, null values where appropriate).