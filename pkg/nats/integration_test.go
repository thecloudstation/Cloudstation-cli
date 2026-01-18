//go:build integration
// +build integration

package nats

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nats-io/nats.go"
)

// TestLogWriter_Integration tests the LogWriter against a real NATS server
// Run with: go test -tags=integration -v ./pkg/nats/...
func TestLogWriter_Integration(t *testing.T) {
	// Check if NATS server is available
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Skipf("NATS server not available at %s: %v", natsURL, err)
	}
	defer nc.Close()

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}

	// Subscribe to build logs
	deploymentID := "integration-test-" + time.Now().Format("20060102150405")
	subject := "build.log." + deploymentID

	received := make(chan *nats.Msg, 10)
	sub, err := js.Subscribe(subject, func(msg *nats.Msg) {
		received <- msg
		msg.Ack()
	}, nats.DeliverNew())
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Create a mock client that publishes to JetStream
	mockClient := &Client{
		conn:   nc,
		logger: hclog.NewNullLogger(),
		prefix: "",
	}

	// Create LogWriter
	logger := hclog.NewNullLogger()
	writer := NewLogWriter(mockClient, deploymentID, 123, "svc-001", "owner-001", "stdout", "build", logger)

	// Write some log lines
	testLines := []string{
		"Step 1/10: FROM node:18-alpine\n",
		"Step 2/10: WORKDIR /app\n",
		"Step 3/10: COPY package*.json ./\n",
	}

	for _, line := range testLines {
		n, err := writer.Write([]byte(line))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if n != len(line) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(line), n)
		}
	}

	// Wait for messages to be received
	timeout := time.After(5 * time.Second)
	messagesReceived := 0

	for messagesReceived < len(testLines) {
		select {
		case msg := <-received:
			t.Logf("Received message: %s", string(msg.Data))
			messagesReceived++
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Expected %d, got %d", len(testLines), messagesReceived)
		}
	}

	t.Logf("Successfully received %d messages", messagesReceived)

	// Close writer and verify final message
	if err := writer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestBuildLogEndToEnd tests the complete build log flow
func TestBuildLogEndToEnd(t *testing.T) {
	// Check if NATS server is available
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Skipf("NATS server not available at %s: %v", natsURL, err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}

	deploymentID := "e2e-test-" + time.Now().Format("20060102150405")

	// Subscribe to both log and end events
	logSubject := "build.log." + deploymentID
	endSubject := "build.log.end." + deploymentID

	logMsgs := make(chan *nats.Msg, 100)
	endMsgs := make(chan *nats.Msg, 1)

	logSub, err := js.Subscribe(logSubject, func(msg *nats.Msg) {
		logMsgs <- msg
		msg.Ack()
	}, nats.DeliverNew())
	if err != nil {
		t.Fatalf("Failed to subscribe to logs: %v", err)
	}
	defer logSub.Unsubscribe()

	endSub, err := js.Subscribe(endSubject, func(msg *nats.Msg) {
		endMsgs <- msg
		msg.Ack()
	}, nats.DeliverNew())
	if err != nil {
		t.Fatalf("Failed to subscribe to end: %v", err)
	}
	defer endSub.Unsubscribe()

	// Create mock client
	mockClient := &Client{
		conn:   nc,
		logger: hclog.NewNullLogger(),
		prefix: "",
	}

	// Simulate a build
	logger := hclog.NewNullLogger()
	stdout := NewLogWriter(mockClient, deploymentID, 1, "svc", "owner", "stdout", "clone", logger)
	stderr := NewLogWriter(mockClient, deploymentID, 1, "svc", "owner", "stderr", "clone", logger)

	// Clone phase
	stdout.Write([]byte("Cloning repository...\n"))
	stdout.Write([]byte("Clone completed.\n"))

	// Build phase
	stdout.SetPhase("build")
	stderr.SetPhase("build")
	stdout.Write([]byte("Building image...\n"))
	stderr.Write([]byte("Warning: deprecated feature\n"))
	stdout.Write([]byte("Build completed.\n"))

	// Registry phase
	stdout.SetPhase("registry")
	stdout.Write([]byte("Pushing to registry...\n"))
	stdout.Write([]byte("Push completed.\n"))

	// Close writers
	stdout.Close()
	stderr.Close()

	// Send end event
	mockClient.PublishBuildLogEnd(BuildLogEndPayload{
		DeploymentID: deploymentID,
		JobID:        1,
		Status:       "success",
	})

	// Wait and count messages
	timeout := time.After(5 * time.Second)
	logCount := 0
	endReceived := false

	for !endReceived || logCount < 7 {
		select {
		case msg := <-logMsgs:
			logCount++
			t.Logf("Log #%d: %s", logCount, string(msg.Data))
		case msg := <-endMsgs:
			endReceived = true
			t.Logf("End event: %s", string(msg.Data))
		case <-timeout:
			t.Logf("Timeout. Logs received: %d, End received: %v", logCount, endReceived)
			break
		}

		if endReceived && logCount >= 7 {
			break
		}
	}

	if logCount < 7 {
		t.Errorf("Expected at least 7 log messages, got %d", logCount)
	}
	if !endReceived {
		t.Errorf("Did not receive end event")
	}

	t.Logf("End-to-end test complete: %d logs, end=%v", logCount, endReceived)
}
