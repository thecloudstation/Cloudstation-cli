package remote

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SSELogEvent represents a log event from the backend
type SSELogEvent struct {
	SeqNo     int    `json:"seqNo"`
	Content   string `json:"content"`
	LogOutput string `json:"logOutput"` // stdout or stderr
	Timestamp string `json:"timestamp"`
}

// SSEClient streams logs from the backend via Server-Sent Events
type SSEClient struct {
	baseURL      string
	sessionToken string
	httpClient   *http.Client
}

// NewSSEClientWithToken creates a new SSE client with JWT Bearer auth
func NewSSEClientWithToken(baseURL, sessionToken string) *SSEClient {
	// Force HTTP/1.1 to avoid HTTP/2 stream errors with proxies like Cloudflare
	// Must disable ALPN negotiation via TLS config to truly force HTTP/1.1
	transport := &http.Transport{
		ForceAttemptHTTP2: false,
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"}, // Only allow HTTP/1.1
		},
	}

	return &SSEClient{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		sessionToken: sessionToken,
		httpClient: &http.Client{
			Timeout:   0, // No timeout for SSE streaming
			Transport: transport,
		},
	}
}

// StreamBuildLogs streams build logs from the backend to the provided writer
// It connects to the /build-logs/:deployment_id endpoint and parses SSE events.
// Returns nil when "event: end" is received or error on failure.
func (c *SSEClient) StreamBuildLogs(deploymentID string, output io.Writer) error {
	url := fmt.Sprintf("%s/build-logs/%s", c.baseURL, deploymentID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Bearer token for JWT authentication
	req.Header.Set("Authorization", "Bearer "+c.sessionToken)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	reader := bufio.NewReader(resp.Body)
	var currentEvent string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Stream ended normally
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = strings.TrimSpace(line)

		// Empty line marks end of event
		if line == "" {
			currentEvent = ""
			continue
		}

		// Parse event type
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))

			// Check for end event
			if currentEvent == "end" {
				return nil
			}
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			// Skip empty data
			if data == "" {
				continue
			}

			// For log events, parse JSON and write content
			if currentEvent == "log" || currentEvent == "" {
				var event SSELogEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					// Write log content to output
					if event.Content != "" {
						fmt.Fprintln(output, event.Content)
					}
				}
			}
		}
	}
}
