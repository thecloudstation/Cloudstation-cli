//go:build ignore
// +build ignore

package main

import (
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/nats"
)

// Example of how to use the NATS LogWriter with command execution
func ExampleUsage() {
	// Initialize NATS client
	natsClient, err := nats.NewClientWithPrefix(
		"nats://localhost:4222",
		"SUAEXAMPLE...", // Your NKey seed
		"cs",            // Prefix for subjects
		hclog.Default(),
	)
	if err != nil {
		panic(err)
	}
	defer natsClient.Close()

	// Create log writers for stdout and stderr
	stdoutWriter := nats.NewLogWriter(
		natsClient,
		"deployment-123", // Deployment ID
		42,               // Job ID
		"service-xyz",    // Service ID
		"owner-abc",      // Owner ID
		"stdout",         // Output type
		"build",          // Initial phase
		hclog.Default(),
	)
	defer stdoutWriter.Close()

	stderrWriter := nats.NewLogWriter(
		natsClient,
		"deployment-123",
		42,
		"service-xyz",
		"owner-abc",
		"stderr",
		"build",
		hclog.Default(),
	)
	defer stderrWriter.Close()

	// Example 1: Use with exec.Command
	cmd := exec.Command("docker", "build", "-t", "myapp", ".")
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	// Update phase as build progresses
	stdoutWriter.SetPhase("build")
	stderrWriter.SetPhase("build")

	if err := cmd.Run(); err != nil {
		// Handle error
	}

	// Example 2: Direct writing (for custom messages)
	stdoutWriter.Write([]byte("Starting deployment phase...\n"))
	stdoutWriter.SetPhase("deploy")
	stderrWriter.SetPhase("deploy")

	// Example 3: Integration with existing builders
	// Replace websocket.NewStreamWriter with nats.NewLogWriter
	// Before:
	//   stdoutWriter := websocket.NewStreamWriter(wsClient, "stdout")
	// After:
	//   stdoutWriter := nats.NewLogWriter(natsClient, deploymentID, jobID, ...)
}
