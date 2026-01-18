package nats

import (
	"bytes"
	"io"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestLogWriter_ImplementsWriter(t *testing.T) {
	writer := &LogWriter{
		client:       &Client{},
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Verify it implements io.Writer
	var _ io.Writer = writer
	if writer == nil {
		t.Fatal("writer should not be nil")
	}
}

func TestLogWriter_BuffersPartialLines(t *testing.T) {
	writer := &LogWriter{
		client:       (*Client)(nil), // We'll check buffer, not publishing
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write partial line (no newline)
	n, err := writer.Write([]byte("Partial line"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 12 {
		t.Errorf("expected 12 bytes written, got %d", n)
	}

	// Check that buffer contains the partial line
	if !bytes.Equal(writer.buffer, []byte("Partial line")) {
		t.Errorf("expected buffer to contain 'Partial line', got %s", writer.buffer)
	}

	// Complete the line
	n, err = writer.Write([]byte(" complete\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 10 {
		t.Errorf("expected 10 bytes written, got %d", n)
	}

	// Buffer should be empty after processing complete line
	if len(writer.buffer) != 0 {
		t.Errorf("expected empty buffer, got %s", writer.buffer)
	}
}

func TestLogWriter_ProcessesMultipleLines(t *testing.T) {
	writer := &LogWriter{
		client:       nil, // Skip publishing
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write multiple lines at once
	multiLine := "Line 1\nLine 2\nLine 3\n"
	n, err := writer.Write([]byte(multiLine))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(multiLine) {
		t.Errorf("expected %d bytes written, got %d", len(multiLine), n)
	}

	// Buffer should be empty
	if len(writer.buffer) != 0 {
		t.Errorf("expected empty buffer, got %s", writer.buffer)
	}

	// Sequence should have incremented for each line
	if writer.sequence != 3 {
		t.Errorf("expected sequence to be 3, got %d", writer.sequence)
	}
}

func TestLogWriter_MixedCompleteAndPartial(t *testing.T) {
	writer := &LogWriter{
		client:       nil, // Skip publishing
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write mixed: complete lines + partial
	mixed := "Line 1\nLine 2\nPartial"
	n, err := writer.Write([]byte(mixed))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(mixed) {
		t.Errorf("expected %d bytes written, got %d", len(mixed), n)
	}

	// Buffer should contain the partial line
	if !bytes.Equal(writer.buffer, []byte("Partial")) {
		t.Errorf("expected buffer to contain 'Partial', got %s", writer.buffer)
	}

	// Sequence should have incremented for complete lines only
	if writer.sequence != 2 {
		t.Errorf("expected sequence to be 2, got %d", writer.sequence)
	}
}

func TestLogWriter_FlushMethod(t *testing.T) {
	writer := &LogWriter{
		client:       nil, // Skip publishing
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write partial line
	writer.Write([]byte("Partial line"))
	if !bytes.Equal(writer.buffer, []byte("Partial line")) {
		t.Errorf("expected buffer to contain 'Partial line', got %s", writer.buffer)
	}

	// Flush should send the partial line
	err := writer.Flush()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Buffer should be empty after flush
	if len(writer.buffer) != 0 {
		t.Errorf("expected empty buffer, got %s", writer.buffer)
	}

	// Sequence should have incremented
	if writer.sequence != 1 {
		t.Errorf("expected sequence to be 1, got %d", writer.sequence)
	}
}

func TestLogWriter_SetPhase(t *testing.T) {
	writer := &LogWriter{
		client:       nil,
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Initial phase
	if writer.phase != "build" {
		t.Errorf("expected initial phase to be 'build', got %s", writer.phase)
	}

	// Update phase
	writer.SetPhase("deploy")
	if writer.phase != "deploy" {
		t.Errorf("expected phase to be 'deploy', got %s", writer.phase)
	}
}

func TestLogWriter_ThreadSafety(t *testing.T) {
	writer := &LogWriter{
		client:       nil, // Skip publishing
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			line := []byte("Concurrent line\n")
			writer.Write(line)
		}(i)
	}

	wg.Wait()

	// Should have processed all lines
	if writer.sequence != 10 {
		t.Errorf("expected sequence to be 10, got %d", writer.sequence)
	}
	if len(writer.buffer) != 0 {
		t.Errorf("expected empty buffer, got %s", writer.buffer)
	}
}

func TestLogWriter_StderrOutput(t *testing.T) {
	writer := &LogWriter{
		client:       nil,
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stderr",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Output should be stderr
	if writer.output != "stderr" {
		t.Errorf("expected output to be 'stderr', got %s", writer.output)
	}

	// Write to stderr
	n, err := writer.Write([]byte("Error message\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 14 {
		t.Errorf("expected 14 bytes written, got %d", n)
	}
}

func TestLogWriter_Close(t *testing.T) {
	writer := &LogWriter{
		client:       nil,
		deploymentID: "test-deployment",
		jobID:        1,
		serviceID:    "test-service",
		ownerID:      "test-owner",
		output:       "stdout",
		phase:        "build",
		buffer:       []byte{},
		sequence:     0,
		logger:       hclog.NewNullLogger(),
	}

	// Write partial line
	writer.Write([]byte("Partial"))
	if !bytes.Equal(writer.buffer, []byte("Partial")) {
		t.Errorf("expected buffer to contain 'Partial', got %s", writer.buffer)
	}

	// Close should flush
	err := writer.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Buffer should be empty
	if len(writer.buffer) != 0 {
		t.Errorf("expected empty buffer, got %s", writer.buffer)
	}
	if writer.sequence != 1 {
		t.Errorf("expected sequence to be 1, got %d", writer.sequence)
	}
}
