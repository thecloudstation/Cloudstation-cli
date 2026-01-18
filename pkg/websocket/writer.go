package websocket

import (
	"bytes"
	"sync"
)

// StreamWriter implements io.Writer to stream logs line-by-line to WebSocket
type StreamWriter struct {
	wsClient *Client
	output   string // "stdout" or "stderr"
	buffer   []byte
	mu       sync.Mutex
}

// NewStreamWriter creates a new streaming writer for WebSocket output
func NewStreamWriter(wsClient *Client, output string) *StreamWriter {
	return &StreamWriter{
		wsClient: wsClient,
		output:   output,
		buffer:   []byte{},
	}
}

// Write implements io.Writer interface - writes data and sends complete lines to WebSocket
func (w *StreamWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Append incoming bytes to buffer
	w.buffer = append(w.buffer, p...)

	// Process complete lines (ending with newline)
	for {
		// Find newline character
		idx := bytes.IndexByte(w.buffer, '\n')
		if idx == -1 {
			// No complete line found, keep buffering
			break
		}

		// Extract line including newline
		line := string(w.buffer[:idx+1])

		// Remove processed line from buffer
		w.buffer = w.buffer[idx+1:]

		// Send line to WebSocket client
		// Remove trailing newline as it will be re-added during message formatting
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		w.wsClient.Write(line, w.output)
	}

	// Return number of bytes consumed (all of them)
	return len(p), nil
}

// Flush sends any remaining buffered data to WebSocket
func (w *StreamWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Send remaining buffer contents if any
	if len(w.buffer) > 0 {
		w.wsClient.Write(string(w.buffer), w.output)
		w.buffer = []byte{}
	}
}
