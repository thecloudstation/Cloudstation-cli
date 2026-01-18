package websocket

import "io"

// WebSocketLogger implements io.Writer to stream logs to WebSocket
type WebSocketLogger struct {
	wsClient    *Client
	output      string
	localWriter io.Writer
}

// Write implements io.Writer interface
func (w *WebSocketLogger) Write(p []byte) (n int, err error) {
	// Write to WebSocket client
	if w.wsClient != nil {
		w.wsClient.Write(string(p), w.output)
	}

	// Also write to local writer for immediate visibility
	if w.localWriter != nil {
		w.localWriter.Write(p)
	}

	// Always succeed to avoid blocking
	return len(p), nil
}

// NewWebSocketLogger creates a new WebSocketLogger instance
func NewWebSocketLogger(wsClient *Client, output string, localWriter io.Writer) io.Writer {
	return &WebSocketLogger{
		wsClient:    wsClient,
		output:      output,
		localWriter: localWriter,
	}
}
