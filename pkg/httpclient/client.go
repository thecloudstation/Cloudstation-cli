package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BaseClient provides common HTTP client functionality
type BaseClient struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewBaseClient creates a new base HTTP client
// Parameters:
//   - baseURL: The base URL for API requests (trailing slash will be removed)
//   - timeout: HTTP client timeout duration
func NewBaseClient(baseURL string, timeout time.Duration) *BaseClient {
	// Normalize baseURL by removing trailing slash for consistency
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &BaseClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
	}
}

// SetHeader sets a custom header that will be included in all requests
func (c *BaseClient) SetHeader(key, value string) {
	c.headers[key] = value
}

// DoJSON performs an HTTP request with JSON request/response bodies
// Parameters:
//   - method: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - path: API path (will be appended to baseURL)
//   - reqBody: Request body (will be JSON marshaled), can be nil for GET requests
//   - respBody: Response body (will be JSON unmarshaled into), can be nil if response not needed
//
// Returns error if request fails or response status is not 2xx
func (c *BaseClient) DoJSON(method, path string, reqBody, respBody interface{}) error {
	return c.DoJSONWithContext(context.Background(), method, path, reqBody, respBody)
}

// DoJSONWithContext performs an HTTP request with context and JSON request/response bodies
// Same as DoJSON but accepts a context for cancellation and timeout control
func (c *BaseClient) DoJSONWithContext(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	// Build full URL by combining baseURL and path
	// Ensure path starts with / for proper URL construction
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := c.baseURL + path

	// Prepare request body if provided
	var bodyReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type header for JSON requests
	if reqBody != nil || method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set all custom headers from headers map
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if status code is in 2xx range (success)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to parse error message from JSON response
		var errorResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}

		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil {
			// Return the most descriptive error message available
			if errorResp.Error != "" {
				return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Error)
			}
			if errorResp.Message != "" {
				return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Message)
			}
		}

		// If no JSON error message, return status code and raw body (truncated if too long)
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, bodyStr)
	}

	// Unmarshal response body if respBody is provided
	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}

// DoJSONWithAuth performs an HTTP request with Bearer token authentication
func (c *BaseClient) DoJSONWithAuth(method, path string, reqBody, respBody interface{}, bearerToken string) error {
	// Build full URL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := c.baseURL + path

	// Prepare request body if provided
	var bodyReader io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil {
			if errorResp.Error != "" {
				return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Error)
			}
			if errorResp.Message != "" {
				return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorResp.Message)
			}
		}
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, bodyStr)
	}

	// Unmarshal response
	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}

// GetBaseURL returns the base URL of the client
func (c *BaseClient) GetBaseURL() string {
	return c.baseURL
}

// GetHeader returns the value of a specific header
func (c *BaseClient) GetHeader(key string) string {
	return c.headers[key]
}
