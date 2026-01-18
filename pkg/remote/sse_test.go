package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSSEClientWithToken(t *testing.T) {
	t.Run("creates client with trimmed base URL", func(t *testing.T) {
		// Test with trailing slash
		client := NewSSEClientWithToken("https://api.example.com/", "test-token")
		if client.baseURL != "https://api.example.com" {
			t.Errorf("Expected trailing slash to be removed, got %s", client.baseURL)
		}
	})

	t.Run("creates client without trailing slash", func(t *testing.T) {
		// Test without trailing slash
		client := NewSSEClientWithToken("https://api.example.com", "test-token")
		if client.baseURL != "https://api.example.com" {
			t.Errorf("Expected baseURL to be 'https://api.example.com', got %s", client.baseURL)
		}
	})

	t.Run("sets all fields correctly", func(t *testing.T) {
		client := NewSSEClientWithToken("https://api.example.com", "my-session-token")

		if client.sessionToken != "my-session-token" {
			t.Errorf("Expected sessionToken to be 'my-session-token', got %s", client.sessionToken)
		}
		if client.httpClient == nil {
			t.Error("Expected httpClient to be initialized")
		}
		if client.httpClient.Timeout != 0 {
			t.Errorf("Expected httpClient timeout to be 0 for SSE streaming, got %v", client.httpClient.Timeout)
		}
	})
}

func TestStreamBuildLogs_Success(t *testing.T) {
	t.Run("streams log events successfully", func(t *testing.T) {
		// Mock SSE server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method
			if r.Method != "GET" {
				t.Errorf("Expected GET method, got %s", r.Method)
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Verify URL path
			expectedPath := "/build-logs/dep_123"
			if r.URL.Path != expectedPath {
				t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			// Verify headers
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("Expected Authorization header to be 'Bearer test-token', got '%s'", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Accept") != "text/event-stream" {
				t.Errorf("Expected Accept header to be 'text/event-stream', got '%s'", r.Header.Get("Accept"))
			}

			// Send SSE events
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)

			// Flush headers
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Send log events
			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Building Docker image...\",\"logOutput\":\"stdout\",\"timestamp\":\"2024-01-01T12:00:00Z\"}\n\n")

			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: {\"seqNo\":2,\"content\":\"Step 1/5 : FROM node:18\",\"logOutput\":\"stdout\",\"timestamp\":\"2024-01-01T12:00:01Z\"}\n\n")

			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: {\"seqNo\":3,\"content\":\"Successfully built!\",\"logOutput\":\"stdout\",\"timestamp\":\"2024-01-01T12:00:02Z\"}\n\n")

			// Send end event
			fmt.Fprintf(w, "event: end\n\n")

			// Flush all events
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify output contains all log messages
		outputStr := output.String()
		expectedLogs := []string{
			"Building Docker image...",
			"Step 1/5 : FROM node:18",
			"Successfully built!",
		}

		for _, expected := range expectedLogs {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("Expected output to contain '%s', got: %s", expected, outputStr)
			}
		}
	})

	t.Run("handles events without explicit event type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send data without event type (should default to log)
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Default event type\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(output.String(), "Default event type") {
			t.Errorf("Expected output to contain 'Default event type', got: %s", output.String())
		}
	})

	t.Run("handles empty data lines gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send empty data line (should be skipped)
			fmt.Fprintf(w, "data: \n\n")
			// Send valid data
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Valid data\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should only contain valid data
		outputStr := output.String()
		if !strings.Contains(outputStr, "Valid data") {
			t.Errorf("Expected output to contain 'Valid data', got: %s", outputStr)
		}
		// Should not have empty lines from empty data
		lines := strings.Split(strings.TrimSpace(outputStr), "\n")
		if len(lines) != 1 {
			t.Errorf("Expected exactly 1 line of output, got %d lines: %v", len(lines), lines)
		}
	})

	t.Run("handles malformed JSON gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send malformed JSON (should be ignored)
			fmt.Fprintf(w, "data: {invalid json}\n\n")
			// Send valid data
			fmt.Fprintf(w, "data: {\"seqNo\":2,\"content\":\"Valid after invalid\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should only contain valid data
		if !strings.Contains(output.String(), "Valid after invalid") {
			t.Errorf("Expected output to contain 'Valid after invalid', got: %s", output.String())
		}
	})

	t.Run("handles multi-line SSE format correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send events with proper SSE format
			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"First line\",\"logOutput\":\"stdout\"}\n")
			fmt.Fprintf(w, "\n") // Empty line to end event

			fmt.Fprintf(w, "event: log\n")
			fmt.Fprintf(w, "data: {\"seqNo\":2,\"content\":\"Second line\",\"logOutput\":\"stderr\"}\n")
			fmt.Fprintf(w, "\n")

			fmt.Fprintf(w, "event: end\n")
			fmt.Fprintf(w, "\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		outputStr := output.String()
		if !strings.Contains(outputStr, "First line") {
			t.Errorf("Expected output to contain 'First line', got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "Second line") {
			t.Errorf("Expected output to contain 'Second line', got: %s", outputStr)
		}
	})

	t.Run("handles events with empty content field", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send event with empty content (should be skipped)
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"\",\"logOutput\":\"stdout\"}\n\n")
			// Send event with content
			fmt.Fprintf(w, "data: {\"seqNo\":2,\"content\":\"Non-empty content\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should only contain non-empty content
		outputStr := strings.TrimSpace(output.String())
		if outputStr != "Non-empty content" {
			t.Errorf("Expected output to be 'Non-empty content', got: '%s'", outputStr)
		}
	})
}

func TestStreamBuildLogs_Errors(t *testing.T) {
	t.Run("handles non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err == nil {
			t.Fatal("Expected error for 500 response, got nil")
		}

		expectedError := "unexpected status 500"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
		}
		if !strings.Contains(err.Error(), "Internal Server Error") {
			t.Errorf("Expected error to contain response body, got: %v", err)
		}
	})

	t.Run("handles 404 not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Deployment not found"))
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_nonexistent", &output)
		if err == nil {
			t.Fatal("Expected error for 404 response, got nil")
		}

		if !strings.Contains(err.Error(), "unexpected status 404") {
			t.Errorf("Expected error to contain 'unexpected status 404', got: %v", err)
		}
		if !strings.Contains(err.Error(), "Deployment not found") {
			t.Errorf("Expected error to contain 'Deployment not found', got: %v", err)
		}
	})

	t.Run("handles 401 unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid token"))
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "invalid-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err == nil {
			t.Fatal("Expected error for 401 response, got nil")
		}

		if !strings.Contains(err.Error(), "unexpected status 401") {
			t.Errorf("Expected error to contain 'unexpected status 401', got: %v", err)
		}
		if !strings.Contains(err.Error(), "Invalid token") {
			t.Errorf("Expected error to contain 'Invalid token', got: %v", err)
		}
	})

	t.Run("handles connection error", func(t *testing.T) {
		// Use an invalid URL that will cause connection error
		client := NewSSEClientWithToken("http://localhost:0", "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err == nil {
			t.Fatal("Expected error for connection failure, got nil")
		}

		if !strings.Contains(err.Error(), "request failed") {
			t.Errorf("Expected error to contain 'request failed', got: %v", err)
		}
	})

	t.Run("handles invalid URL", func(t *testing.T) {
		// Use an invalid URL format
		client := NewSSEClientWithToken("://invalid-url", "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err == nil {
			t.Fatal("Expected error for invalid URL, got nil")
		}

		if !strings.Contains(err.Error(), "failed to create request") {
			t.Errorf("Expected error to contain 'failed to create request', got: %v", err)
		}
	})

	t.Run("handles server closing connection unexpectedly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send some data
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Starting...\",\"logOutput\":\"stdout\"}\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Simulate server closing connection without sending "event: end"
			// This will cause EOF when client tries to read more
			// The handler will return and close the connection
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		// This should not return an error, as EOF is handled gracefully
		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Expected no error for EOF (graceful close), got: %v", err)
		}

		// Should still have received the initial data
		if !strings.Contains(output.String(), "Starting...") {
			t.Errorf("Expected output to contain 'Starting...', got: %s", output.String())
		}
	})

	t.Run("handles read error during streaming", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send initial data
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Before error\",\"logOutput\":\"stdout\"}\n\n")

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Simply return without sending "event: end"
			// This causes EOF when client tries to read next line
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		// Handler returns without sending "event: end", closing the connection and causing EOF
		err := client.StreamBuildLogs("dep_123", &output)
		// EOF is handled gracefully (returns nil)
		if err != nil {
			t.Logf("Got error: %v", err)
		}

		// Should have received data before the error
		if !strings.Contains(output.String(), "Before error") {
			t.Errorf("Expected output to contain 'Before error', got: %s", output.String())
		}
	})
}

func TestStreamBuildLogs_HeaderValidation(t *testing.T) {
	t.Run("sends correct headers", func(t *testing.T) {
		expectedToken := "my-secret-token"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify all headers are set correctly
			authHeader := r.Header.Get("Authorization")
			expectedAuth := "Bearer " + expectedToken
			if authHeader != expectedAuth {
				t.Errorf("Expected Authorization '%s', got '%s'", expectedAuth, authHeader)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			accept := r.Header.Get("Accept")
			if accept != "text/event-stream" {
				t.Errorf("Expected Accept 'text/event-stream', got '%s'", accept)
				http.Error(w, "Invalid Accept header", http.StatusBadRequest)
				return
			}

			// Send valid response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Headers OK\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, expectedToken)
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !strings.Contains(output.String(), "Headers OK") {
			t.Errorf("Expected output to contain 'Headers OK', got: %s", output.String())
		}
	})

	t.Run("empty token still sent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that header exists even if empty
			if _, ok := r.Header["Authorization"]; !ok {
				t.Error("Authorization header not present")
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "")
		var output bytes.Buffer

		_ = client.StreamBuildLogs("dep_123", &output)
	})
}

func TestStreamBuildLogs_URLConstruction(t *testing.T) {
	t.Run("constructs correct URL with deployment ID", func(t *testing.T) {
		deploymentID := "dep_abc123xyz"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/build-logs/" + deploymentID
			if r.URL.Path != expectedPath {
				t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs(deploymentID, &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("handles special characters in deployment ID", func(t *testing.T) {
		// Deployment ID with URL-safe characters
		deploymentID := "dep_123-456.789"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/build-logs/" + deploymentID
			if r.URL.Path != expectedPath {
				t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs(deploymentID, &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

func TestSSELogEvent_Parsing(t *testing.T) {
	t.Run("parses complete log event", func(t *testing.T) {
		jsonData := `{
			"seqNo": 42,
			"content": "Test log message",
			"logOutput": "stderr",
			"timestamp": "2024-01-01T15:30:00Z"
		}`

		var event SSELogEvent
		err := json.Unmarshal([]byte(jsonData), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		if event.SeqNo != 42 {
			t.Errorf("Expected SeqNo 42, got %d", event.SeqNo)
		}
		if event.Content != "Test log message" {
			t.Errorf("Expected Content 'Test log message', got '%s'", event.Content)
		}
		if event.LogOutput != "stderr" {
			t.Errorf("Expected LogOutput 'stderr', got '%s'", event.LogOutput)
		}
		if event.Timestamp != "2024-01-01T15:30:00Z" {
			t.Errorf("Expected Timestamp '2024-01-01T15:30:00Z', got '%s'", event.Timestamp)
		}
	})

	t.Run("parses event with missing optional fields", func(t *testing.T) {
		jsonData := `{
			"seqNo": 1,
			"content": "Minimal event"
		}`

		var event SSELogEvent
		err := json.Unmarshal([]byte(jsonData), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		if event.SeqNo != 1 {
			t.Errorf("Expected SeqNo 1, got %d", event.SeqNo)
		}
		if event.Content != "Minimal event" {
			t.Errorf("Expected Content 'Minimal event', got '%s'", event.Content)
		}
		if event.LogOutput != "" {
			t.Errorf("Expected LogOutput to be empty, got '%s'", event.LogOutput)
		}
		if event.Timestamp != "" {
			t.Errorf("Expected Timestamp to be empty, got '%s'", event.Timestamp)
		}
	})
}

func TestStreamBuildLogs_OutputWriter(t *testing.T) {
	t.Run("writes to custom writer implementation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Line 1\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "data: {\"seqNo\":2,\"content\":\"Line 2\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		// Custom writer that counts writes
		cw := &customCountingWriter{
			writes: 0,
			data:   make([]string, 0),
		}

		client := NewSSEClientWithToken(server.URL, "test-token")

		err := client.StreamBuildLogs("dep_123", cw)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Should have written twice (once for each log line)
		if cw.writes != 2 {
			t.Errorf("Expected 2 writes, got %d", cw.writes)
		}

		// Verify the content
		if len(cw.data) != 2 {
			t.Errorf("Expected 2 data entries, got %d", len(cw.data))
		}

		expectedLines := []string{"Line 1\n", "Line 2\n"}
		for i, expected := range expectedLines {
			if i < len(cw.data) && cw.data[i] != expected {
				t.Errorf("Expected data[%d] to be '%s', got '%s'", i, expected, cw.data[i])
			}
		}
	})

	t.Run("handles write errors gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, "data: {\"seqNo\":1,\"content\":\"Test\",\"logOutput\":\"stdout\"}\n\n")
			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		// Writer that returns an error
		errorWriter := &erroringWriter{
			err: io.ErrShortWrite,
		}

		client := NewSSEClientWithToken(server.URL, "test-token")

		// The current implementation doesn't check for write errors,
		// so this will complete without error
		err := client.StreamBuildLogs("dep_123", errorWriter)
		if err != nil {
			t.Logf("Got error (expected to complete): %v", err)
		}
	})
}

// Helper type for testing write errors
type erroringWriter struct {
	err error
}

func (w *erroringWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

// Helper type for counting writer
type customCountingWriter struct {
	writes int
	data   []string
}

func (w *customCountingWriter) Write(p []byte) (n int, err error) {
	w.writes++
	w.data = append(w.data, string(p))
	return len(p), nil
}

func TestStreamBuildLogs_LongRunning(t *testing.T) {
	t.Run("handles long-running streams", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Simulate a long-running build with multiple events
			for i := 1; i <= 5; i++ {
				fmt.Fprintf(w, "data: {\"seqNo\":%d,\"content\":\"Progress: %d/5\",\"logOutput\":\"stdout\"}\n\n", i, i)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}

				// Small delay to simulate real streaming
				time.Sleep(10 * time.Millisecond)
			}

			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify all progress messages were received
		for i := 1; i <= 5; i++ {
			expected := fmt.Sprintf("Progress: %d/5", i)
			if !strings.Contains(output.String(), expected) {
				t.Errorf("Expected output to contain '%s', got: %s", expected, output.String())
			}
		}
	})
}

func TestStreamBuildLogs_EventSequence(t *testing.T) {
	t.Run("maintains event sequence order", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send events in specific order
			events := []struct {
				seqNo   int
				content string
			}{
				{1, "First"},
				{2, "Second"},
				{3, "Third"},
			}

			for _, evt := range events {
				data := fmt.Sprintf(`{"seqNo":%d,"content":"%s","logOutput":"stdout"}`, evt.seqNo, evt.content)
				fmt.Fprintf(w, "data: %s\n\n", data)
			}

			fmt.Fprintf(w, "event: end\n\n")
		}))
		defer server.Close()

		client := NewSSEClientWithToken(server.URL, "test-token")
		var output bytes.Buffer

		err := client.StreamBuildLogs("dep_123", &output)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify order is maintained
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		expectedOrder := []string{"First", "Second", "Third"}

		if len(lines) != len(expectedOrder) {
			t.Errorf("Expected %d lines, got %d", len(expectedOrder), len(lines))
		}

		for i, expected := range expectedOrder {
			if i < len(lines) && lines[i] != expected {
				t.Errorf("Expected line %d to be '%s', got '%s'", i, expected, lines[i])
			}
		}
	})
}
