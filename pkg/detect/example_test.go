package detect_test

import (
	"fmt"
	"log"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
)

func ExampleDetectBuilder() {
	// Example: Detecting builder for a project directory
	result := detect.DetectBuilder("/path/to/project")

	fmt.Printf("Selected builder: %s\n", result.Builder)
	fmt.Printf("Reason: %s\n", result.Reason)

	if result.HasDocker {
		fmt.Println("Dockerfile detected - using Vault integration")
	} else {
		fmt.Println("No Dockerfile - using zero-config build")
	}

	// Log detected project signals
	for _, signal := range result.Signals {
		log.Printf("Detected: %s", signal)
	}
}

func ExampleHasDockerfile() {
	// Quick check if a project has a Dockerfile
	if detect.HasDockerfile("/path/to/project") {
		fmt.Println("Project has Dockerfile")
	} else {
		fmt.Println("Project does not have Dockerfile")
	}
}

func ExampleGetDefaultBuilder() {
	// Get the recommended builder for a directory
	builder := detect.GetDefaultBuilder("/path/to/project")

	switch builder {
	case "csdocker":
		fmt.Println("Using csdocker builder with Vault integration")
	case "railpack":
		fmt.Println("Using railpack for zero-config build")
	default:
		fmt.Printf("Unknown builder: %s\n", builder)
	}
}
