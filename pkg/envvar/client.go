// Package envvar provides an HTTP client for managing environment variables
// via the cs-backend /api/envvar endpoints. Environment variables are stored
// in Vault and managed per-service.
package envvar

import (
	"fmt"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// Variable represents a single environment variable with key-value pair.
type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// BulkCreateRequest represents the request body for bulk environment variable creation.
// The Type field should be "image" for Docker image services or "integration" for git repos.
type BulkCreateRequest struct {
	ImageID   string     `json:"imageId,omitempty"`
	Type      string     `json:"type"`
	Variables []Variable `json:"variables"`
}

// BulkCreateResponse represents the response from bulk create operations.
type BulkCreateResponse struct {
	Message string `json:"message"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
}

// Client provides methods for managing environment variables via the cs-backend API.
// It embeds BaseClient for HTTP functionality and handles authentication via
// JWT Bearer token.
type Client struct {
	*httpclient.BaseClient
}

// NewClientWithToken creates a new environment variable client configured to communicate
// with the cs-backend API using JWT token authentication.
//
// Parameters:
//   - baseURL: The base URL of the cs-backend API (e.g., "https://api.cloudstation.io")
//   - sessionToken: The JWT session token for authentication
//
// Returns a configured Client ready to make API calls.
func NewClientWithToken(baseURL, sessionToken string) *Client {
	client := &Client{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+sessionToken)
	return client
}

// BulkCreate creates or updates multiple environment variables for a service in Vault.
// This is an atomic operation - either all variables are created/updated or none are.
//
// Parameters:
//   - serviceID: The UUID of the service (image ID) to associate variables with
//   - variables: A slice of Variable structs containing key-value pairs
//
// Returns an error if the request fails or if validation fails.
// On success, variables are stored in Vault and accessible to the service at runtime.
func (c *Client) BulkCreate(serviceID string, variables []Variable) error {
	// Validate inputs
	if serviceID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}
	if len(variables) == 0 {
		return fmt.Errorf("variables list cannot be empty")
	}

	// Validate each variable has non-empty key
	for i, v := range variables {
		if v.Key == "" {
			return fmt.Errorf("variable at index %d has empty key", i)
		}
	}

	// Build request body
	// Using "image" type as this client is primarily used for Docker image services
	reqBody := BulkCreateRequest{
		ImageID:   serviceID,
		Type:      "image",
		Variables: variables,
	}

	// Make API call - response is optional but we can capture it for logging
	var resp BulkCreateResponse
	if err := c.DoJSON("POST", "/environment-variables/bulk", reqBody, &resp); err != nil {
		return fmt.Errorf("failed to create environment variables: %w", err)
	}

	return nil
}

// BulkCreateWithResponse creates or updates multiple environment variables and returns
// detailed information about the operation results.
//
// Parameters:
//   - serviceID: The UUID of the service (image ID) to associate variables with
//   - variables: A slice of Variable structs containing key-value pairs
//
// Returns the number of created and updated variables, or an error if the request fails.
func (c *Client) BulkCreateWithResponse(serviceID string, variables []Variable) (created, updated int, err error) {
	// Validate inputs
	if serviceID == "" {
		return 0, 0, fmt.Errorf("service ID cannot be empty")
	}
	if len(variables) == 0 {
		return 0, 0, fmt.Errorf("variables list cannot be empty")
	}

	// Validate each variable has non-empty key
	for i, v := range variables {
		if v.Key == "" {
			return 0, 0, fmt.Errorf("variable at index %d has empty key", i)
		}
	}

	// Build request body
	reqBody := BulkCreateRequest{
		ImageID:   serviceID,
		Type:      "image",
		Variables: variables,
	}

	// Make API call
	var resp BulkCreateResponse
	if err := c.DoJSON("POST", "/environment-variables/bulk", reqBody, &resp); err != nil {
		return 0, 0, fmt.Errorf("failed to create environment variables: %w", err)
	}

	return resp.Created, resp.Updated, nil
}

// List retrieves all environment variables for a service from Vault.
//
// Parameters:
//   - serviceID: The UUID of the service to retrieve variables for
//
// Returns a slice of Variable structs, or an error if the request fails.
// Returns an empty slice if no variables are configured for the service.
func (c *Client) List(serviceID string) ([]Variable, error) {
	// Validate input
	if serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	}

	// The API returns an array of variables directly
	var variables []Variable

	path := fmt.Sprintf("/environment-variables/%s", serviceID)
	if err := c.DoJSON("GET", path, nil, &variables); err != nil {
		return nil, fmt.Errorf("failed to list environment variables: %w", err)
	}

	// Handle nil response (no variables configured)
	if variables == nil {
		return []Variable{}, nil
	}

	return variables, nil
}

// Delete removes a specific environment variable from a service.
//
// Parameters:
//   - serviceID: The UUID of the service
//   - key: The key of the environment variable to delete
//
// Returns an error if the request fails or if the variable does not exist.
func (c *Client) Delete(serviceID, key string) error {
	// Validate inputs
	if serviceID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("variable key cannot be empty")
	}

	path := fmt.Sprintf("/environment-variables/%s?key=%s", serviceID, key)
	if err := c.DoJSON("DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("failed to delete environment variable: %w", err)
	}

	return nil
}

// DeleteAll removes all environment variables for a service.
//
// Parameters:
//   - serviceID: The UUID of the service
//
// Returns the number of deleted variables, or an error if the request fails.
func (c *Client) DeleteAll(serviceID string) (int, error) {
	// Validate input
	if serviceID == "" {
		return 0, fmt.Errorf("service ID cannot be empty")
	}

	var resp struct {
		Message string `json:"message"`
		Deleted int    `json:"deleted"`
	}

	path := fmt.Sprintf("/environment-variables/%s/all", serviceID)
	if err := c.DoJSON("DELETE", path, nil, &resp); err != nil {
		return 0, fmt.Errorf("failed to delete all environment variables: %w", err)
	}

	return resp.Deleted, nil
}

// Update updates a specific environment variable for a service.
//
// Parameters:
//   - serviceID: The UUID of the service
//   - key: The key of the environment variable to update
//   - value: The new value for the environment variable
//
// Returns an error if the request fails.
func (c *Client) Update(serviceID, key, value string) error {
	// Validate inputs
	if serviceID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("variable key cannot be empty")
	}

	reqBody := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   key,
		Value: value,
	}

	path := fmt.Sprintf("/environment-variables/%s", serviceID)
	if err := c.DoJSON("PUT", path, reqBody, nil); err != nil {
		return fmt.Errorf("failed to update environment variable: %w", err)
	}

	return nil
}
