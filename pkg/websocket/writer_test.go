package websocket

import (
	"io"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
)

// mockClient is a mock WebSocket client for testing
type mockClient struct {
	messages []string
	mu       sync.Mutex
}

func (m *mockClient) Write(chunk string, output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, output+":"+chunk)
}

func TestStreamWriter_ImplementsWriter(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Verify it implements io.Writer
	var _ io.Writer = writer
}

func TestStreamWriter_BuffersPartialLines(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write partial line (no newline)
	n, err := writer.Write([]byte("partial"))
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if n != 7 {
		t.Errorf("Expected 7 bytes written, got: %d", n)
	}

	// Buffer should contain data
	if len(writer.buffer) != 7 {
		t.Errorf("Expected 7 bytes in buffer, got: %d", len(writer.buffer))
	}
}

func TestStreamWriter_FlushesCompleteLines(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write complete line with newline
	writer.Write([]byte("complete line\n"))

	// Buffer should be empty after processing complete line
	if len(writer.buffer) != 0 {
		t.Errorf("Expected empty buffer after complete line, got %d bytes", len(writer.buffer))
	}

	// Client buffer should have received the message
	client.mu.Lock()
	bufferLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if bufferLen != 1 {
		t.Errorf("Expected 1 message in client buffer, got %d", bufferLen)
	}
}

func TestStreamWriter_FlushMethod(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write partial line
	writer.Write([]byte("partial without newline"))

	// Buffer should contain data
	if len(writer.buffer) == 0 {
		t.Error("Expected data in buffer before flush")
	}

	// Call Flush
	writer.Flush()

	// Buffer should be empty
	if len(writer.buffer) != 0 {
		t.Errorf("Expected empty buffer after flush, got %d bytes", len(writer.buffer))
	}

	// Client buffer should have received the message
	client.mu.Lock()
	bufferLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if bufferLen != 1 {
		t.Errorf("Expected 1 message in client buffer after flush, got %d", bufferLen)
	}
}

func TestStreamWriter_MultipleLines(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write multiple lines at once
	writer.Write([]byte("line 1\nline 2\nline 3\n"))

	// Buffer should be empty (all lines processed)
	if len(writer.buffer) != 0 {
		t.Errorf("Expected empty buffer, got %d bytes", len(writer.buffer))
	}

	// Client buffer should have 3 messages
	client.mu.Lock()
	bufferLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if bufferLen != 3 {
		t.Errorf("Expected 3 messages in client buffer, got %d", bufferLen)
	}
}

func TestStreamWriter_MixedCompleteAndPartial(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write mixed: complete lines + partial
	writer.Write([]byte("line 1\nline 2\npartial"))

	// Buffer should contain "partial"
	if string(writer.buffer) != "partial" {
		t.Errorf("Expected buffer to contain 'partial', got '%s'", string(writer.buffer))
	}

	// Client buffer should have 2 complete lines
	client.mu.Lock()
	bufferLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if bufferLen != 2 {
		t.Errorf("Expected 2 messages in client buffer, got %d", bufferLen)
	}
}

func TestStreamWriter_ThreadSafety(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stdout")

	// Write from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			writer.Write([]byte("line from goroutine\n"))
		}(i)
	}

	wg.Wait()

	// Should not panic and should have processed all lines
	client.mu.Lock()
	bufferLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if bufferLen != 10 {
		t.Errorf("Expected 10 messages, got %d", bufferLen)
	}
}

func TestStreamWriter_StderrOutput(t *testing.T) {
	client, _ := NewClient("ws://localhost:9999", "test", hclog.NewNullLogger())
	writer := NewStreamWriter(client, "stderr")

	// Write to stderr
	writer.Write([]byte("error message\n"))

	// Check stderr buffer
	client.mu.Lock()
	bufferLen := len(client.stderrBuffer)
	client.mu.Unlock()

	if bufferLen != 1 {
		t.Errorf("Expected 1 message in stderr buffer, got %d", bufferLen)
	}

	// Stdout buffer should be empty
	client.mu.Lock()
	stdoutLen := len(client.stdoutBuffer)
	client.mu.Unlock()

	if stdoutLen != 0 {
		t.Errorf("Expected 0 messages in stdout buffer, got %d", stdoutLen)
	}
}
