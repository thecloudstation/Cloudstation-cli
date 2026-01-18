package websocket

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

func TestNewClient_InvalidURL(t *testing.T) {
	logger := hclog.NewNullLogger()

	// Test with invalid URL
	_, err := NewClient("ht!tp://invalid url", "test-deployment", logger)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestClient_Write(t *testing.T) {
	logger := hclog.NewNullLogger()

	// Create client (will fail to connect but that's ok for this test)
	client, _ := NewClient("ws://localhost:9999", "test-deployment", logger)
	defer client.End()

	// Test writing to stdout buffer
	client.Write("test message stdout", "stdout")

	client.mu.Lock()
	if len(client.stdoutBuffer) != 1 {
		t.Errorf("Expected 1 stdout buffer entry, got %d", len(client.stdoutBuffer))
	}
	if client.stdoutBuffer[0] != "test message stdout" {
		t.Errorf("Expected 'test message stdout', got '%s'", client.stdoutBuffer[0])
	}
	client.mu.Unlock()

	// Test writing to stderr buffer
	client.Write("test message stderr", "stderr")

	client.mu.Lock()
	if len(client.stderrBuffer) != 1 {
		t.Errorf("Expected 1 stderr buffer entry, got %d", len(client.stderrBuffer))
	}
	if client.stderrBuffer[0] != "test message stderr" {
		t.Errorf("Expected 'test message stderr', got '%s'", client.stderrBuffer[0])
	}
	client.mu.Unlock()
}

func TestClient_FlushClearsBuffers(t *testing.T) {
	logger := hclog.NewNullLogger()

	// Create client (will fail to connect but that's ok for this test)
	client, _ := NewClient("ws://localhost:9999", "test-deployment", logger)
	defer client.End()

	// Add messages to buffers
	client.Write("message 1", "stdout")
	client.Write("message 2", "stderr")

	// Mark as connected to allow flush (even though socket won't work)
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	// Flush should clear buffers (even if emit fails)
	client.Flush()

	// Give it a moment for flush to complete
	time.Sleep(100 * time.Millisecond)

	client.mu.Lock()
	stdoutLen := len(client.stdoutBuffer)
	stderrLen := len(client.stderrBuffer)
	client.mu.Unlock()

	if stdoutLen != 0 {
		t.Errorf("Expected stdout buffer to be empty after flush, got %d entries", stdoutLen)
	}
	if stderrLen != 0 {
		t.Errorf("Expected stderr buffer to be empty after flush, got %d entries", stderrLen)
	}
}

func TestClient_End(t *testing.T) {
	logger := hclog.NewNullLogger()

	// Create client
	client, _ := NewClient("ws://localhost:9999", "test-deployment", logger)

	// End should not panic
	err := client.End()
	if err != nil {
		t.Errorf("Expected nil error from End(), got %v", err)
	}
}

func TestClient_CounterIncrementsOnFlush(t *testing.T) {
	logger := hclog.NewNullLogger()

	// Create client
	client, _ := NewClient("ws://localhost:9999", "test-deployment", logger)
	defer client.End()

	// Initial counter should be 0
	if client.counter != 0 {
		t.Errorf("Expected initial counter to be 0, got %d", client.counter)
	}

	// Add messages and mark as connected
	client.Write("message 1", "stdout")
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	// Flush should increment counter
	client.Flush()
	time.Sleep(100 * time.Millisecond)

	if client.counter != 1 {
		t.Errorf("Expected counter to be 1 after flush, got %d", client.counter)
	}
}
