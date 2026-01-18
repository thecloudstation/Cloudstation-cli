package dispatch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
)

// Exit code constants
const (
	ExitCodeSuccess    = 0
	ExitCodeRuntime    = 1
	ExitCodeParseError = 2
	ExitCodeValidation = 3
	ExitCodeTimeout    = 4
)

// FormatParameterError formats a parameter validation error with context
func FormatParameterError(fieldName string, received interface{}) string {
	return fmt.Sprintf("Validation failed: %s is required (received: %+v)", fieldName, received)
}

// LogToStdout writes a formatted message to stdout with timestamp and immediate flush
func LogToStdout(format string, args ...interface{}) {
	// Write to stdout
	fmt.Fprintf(os.Stdout, format, args...)
	// Ensure immediate flush
	os.Stdout.Sync()
}

// LogErrorToStderr logs an error to both hclog and stdout (for Nomad visibility)
func LogErrorToStderr(logger hclog.Logger, phase string, err error) {
	// Log via hclog
	logger.Error("Operation failed", "phase", phase, "error", err)

	// Write to stdout for Nomad visibility (not stderr)
	fmt.Fprintf(os.Stdout, "ERROR [%s]: %v\n", phase, err)

	// Flush stdout
	os.Stdout.Sync()
}

// LogParameterValidation logs received parameters for debugging validation failures
func LogParameterValidation(logger hclog.Logger, params interface{}) error {
	// Pretty-print parameters
	jsonBytes, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	// Log to hclog
	logger.Error("Parameter validation failed", "received_params", string(jsonBytes))

	// Also write to stdout for Nomad visibility
	fmt.Fprintf(os.Stdout, "Received parameters:\n%s\n", string(jsonBytes))
	os.Stdout.Sync()

	return nil
}
