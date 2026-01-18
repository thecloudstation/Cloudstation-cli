package vault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-retryablehttp"
)

// TLSConfig holds TLS-related configuration for Vault client.
type TLSConfig struct {
	// InsecureSkipVerify controls whether the client verifies the server's
	// certificate chain and host name. If true, any certificate presented
	// by the server will be accepted. This should only be used for testing.
	// Default: false (TLS verification enabled)
	InsecureSkipVerify bool

	// CACert is the path to a PEM-encoded CA certificate file.
	// If provided, this CA will be used to verify the Vault server's certificate.
	CACert string
}

// ClientConfig holds configuration for the Vault client.
type ClientConfig struct {
	// Address is the Vault server URL (e.g., "https://vault.example.com:8200")
	Address string

	// Token is the Vault authentication token (obtained via AppRole or other auth method)
	Token string

	// TLS holds TLS-related configuration
	TLS *TLSConfig

	// Timeout is the maximum time to wait for requests
	// Default: 30 seconds
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests
	// Default: 3
	MaxRetries int

	// Logger is used for structured logging
	Logger hclog.Logger
}

// Client is a Vault client that handles authentication and secret retrieval.
type Client struct {
	address    string
	token      string
	httpClient *retryablehttp.Client
	logger     hclog.Logger
}

// NewClient creates a new Vault client with the provided configuration.
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	// Apply defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Logger == nil {
		cfg.Logger = hclog.NewNullLogger()
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if cfg.TLS != nil {
		// Warn if TLS verification is disabled
		if cfg.TLS.InsecureSkipVerify {
			cfg.Logger.Warn("TLS verification is disabled - this should only be used for testing")
			tlsConfig.InsecureSkipVerify = true
		}

		// Load custom CA certificate if provided
		if cfg.TLS.CACert != "" {
			caCert, err := os.ReadFile(cfg.TLS.CACert)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}
	}

	// Create HTTP client with retry logic
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = cfg.MaxRetries
	retryClient.Logger = cfg.Logger

	// Configure standard HTTP client with TLS and timeout
	retryClient.HTTPClient = &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		address:    cfg.Address,
		token:      cfg.Token,
		httpClient: retryClient,
		logger:     cfg.Logger,
	}, nil
}

// AuthenticateAppRole authenticates to Vault using the AppRole auth method
// and stores the resulting token in the client.
func (c *Client) AuthenticateAppRole(ctx context.Context, roleID, secretID string) error {
	if roleID == "" || secretID == "" {
		return fmt.Errorf("role_id and secret_id are required for AppRole authentication")
	}

	url := fmt.Sprintf("%s/v1/auth/approle/login", c.address)

	payload := map[string]string{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	c.logger.Debug("authenticating to vault using AppRole", "address", c.address)

	var response struct {
		Auth struct {
			ClientToken   string `json:"client_token"`
			LeaseDuration int    `json:"lease_duration"`
		} `json:"auth"`
	}

	if err := c.doRequest(ctx, "POST", url, payload, &response); err != nil {
		return fmt.Errorf("AppRole authentication failed: %w", err)
	}

	if response.Auth.ClientToken == "" {
		return fmt.Errorf("received empty token from Vault")
	}

	c.token = response.Auth.ClientToken
	c.logger.Info("successfully authenticated to Vault", "lease_duration", response.Auth.LeaseDuration)

	return nil
}

// ReadSecret reads a secret from Vault at the specified path.
// The path should be the full API path (e.g., "secret/data/myapp" for KV v2
// or "secret/myapp" for KV v1).
func (c *Client) ReadSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	if c.token == "" {
		return nil, fmt.Errorf("vault client is not authenticated")
	}

	url := fmt.Sprintf("%s/v1/%s", c.address, path)

	c.logger.Debug("reading secret from vault", "path", path)

	var response struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := c.doRequest(ctx, "GET", url, nil, &response); err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	if response.Data == nil {
		return nil, fmt.Errorf("secret not found at path: %s", path)
	}

	// KV v2 wraps secrets in a "data" field
	if data, ok := response.Data["data"].(map[string]interface{}); ok {
		c.logger.Debug("extracted KV v2 secret data", "keys", len(data))
		return data, nil
	}

	// KV v1 returns secrets directly
	c.logger.Debug("using KV v1 secret data", "keys", len(response.Data))
	return response.Data, nil
}

// doRequest performs an HTTP request to Vault with retry logic.
func (c *Client) doRequest(ctx context.Context, method, url string, payload interface{}, result interface{}) error {
	// Create request with optional payload
	var req *retryablehttp.Request
	var err error

	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request payload: %w", err)
		}
		req, err = retryablehttp.NewRequestWithContext(ctx, method, url, jsonData)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = retryablehttp.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	// Add Vault token header if available
	if c.token != "" {
		req.Header.Set("X-Vault-Token", c.token)
	}

	// Execute request with retry logic
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

// handleResponse processes the HTTP response and unmarshals the result.
func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var vaultErr struct {
			Errors []string `json:"errors"`
		}
		// Try to parse Vault error format
		if json.Unmarshal(bodyBytes, &vaultErr) == nil && len(vaultErr.Errors) > 0 {
			return fmt.Errorf("vault returned error (status %d): %v", resp.StatusCode, vaultErr.Errors)
		}
		return fmt.Errorf("vault returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Unmarshal successful response
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// Close cleans up any resources held by the client.
func (c *Client) Close() error {
	// Currently no cleanup needed, but this method is here for future use
	// (e.g., token revocation, connection cleanup, etc.)
	return nil
}
