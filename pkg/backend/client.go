package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-retryablehttp"
)

// Client is the HTTP client for CloudStation backend API
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *retryablehttp.Client
	logger      hclog.Logger
}

// NewClient creates a new backend API client
func NewClient(backendURL, accessToken string, logger hclog.Logger) (*Client, error) {
	if backendURL == "" {
		return nil, fmt.Errorf("backend URL cannot be empty")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.Logger = nil // Disable default logging

	// Set timeout
	retryClient.HTTPClient.Timeout = 30 * time.Second

	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	return &Client{
		baseURL:     strings.TrimSuffix(backendURL, "/"),
		accessToken: accessToken,
		httpClient:  retryClient,
		logger:      logger,
	}, nil
}

// redactToken redacts the access token from strings for logging
func (c *Client) redactToken(s string) string {
	if c.accessToken != "" {
		return strings.ReplaceAll(s, c.accessToken, "[REDACTED]")
	}
	return s
}

// AskDomain requests a subdomain allocation for a service
// GET /api/local/ask-domain?serviceId={serviceId}&accessToken={token}
func (c *Client) AskDomain(serviceID string) (string, error) {
	if serviceID == "" {
		return "", fmt.Errorf("serviceID cannot be empty")
	}

	// Build URL with query parameters
	endpoint := fmt.Sprintf("%s/api/local/ask-domain", c.baseURL)
	params := url.Values{}
	params.Add("serviceId", serviceID)
	params.Add("accessToken", c.accessToken)
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	c.logger.Debug("requesting domain allocation", "serviceId", serviceID, "url", c.redactToken(fullURL))

	// Create request
	req, err := retryablehttp.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var domainResp AskDomainResponse
	if err := json.Unmarshal(body, &domainResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Info("domain allocated", "serviceId", serviceID, "domain", domainResp.Domain)
	return domainResp.Domain, nil
}

// UpdateService syncs service configuration metadata to backend
// PUT /api/local/service-update/?accessToken={token}
func (c *Client) UpdateService(req UpdateServiceRequest) error {
	if req.ServiceID == "" {
		return fmt.Errorf("serviceID cannot be empty")
	}

	// Build URL with access token as query parameter
	endpoint := fmt.Sprintf("%s/api/local/service-update/", c.baseURL)
	params := url.Values{}
	params.Add("accessToken", c.accessToken)
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	c.logger.Debug("updating service configuration", "serviceId", req.ServiceID, "url", c.redactToken(fullURL))

	// Marshal request body
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := retryablehttp.NewRequest("PUT", fullURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("service configuration updated", "serviceId", req.ServiceID)
	return nil
}

// UpdateDeploymentStep tracks deployment progress for UI visibility
// PUT /api/local/deployment-step/update?accessToken={token}
func (c *Client) UpdateDeploymentStep(req UpdateDeploymentStepRequest) error {
	if req.DeploymentID == "" {
		return fmt.Errorf("deploymentID cannot be empty")
	}
	if req.Step == "" {
		return fmt.Errorf("step cannot be empty")
	}
	if req.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}

	// Build URL with access token as query parameter
	endpoint := fmt.Sprintf("%s/api/local/deployment-step/update", c.baseURL)
	params := url.Values{}
	params.Add("accessToken", c.accessToken)
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	c.logger.Debug("updating deployment step",
		"deploymentId", req.DeploymentID,
		"step", req.Step,
		"status", req.Status,
		"url", c.redactToken(fullURL))

	// Marshal request body
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := retryablehttp.NewRequest("PUT", fullURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("deployment step updated",
		"deploymentId", req.DeploymentID,
		"step", req.Step,
		"status", req.Status)
	return nil
}
