// Package volume provides an HTTP client for managing volumes
// via the cs-backend /volumes endpoints. Volumes are CSI-based
// persistent storage attached to services.
package volume

import (
	"fmt"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// AttachRequest represents a request to create and attach a volume to a service.
type AttachRequest struct {
	Capacity   float64  `json:"capacity"`
	Name       *string  `json:"name,omitempty"`
	MountPaths []string `json:"mountPaths"`
}

// AttachResponse represents the response after attaching a volume.
type AttachResponse struct {
	ID        string `json:"id"`
	ServiceID string `json:"serviceId"`
	Message   string `json:"message,omitempty"`
}

// Client provides methods for managing volumes via the cs-backend API.
// It embeds BaseClient for HTTP functionality and handles authentication via
// JWT Bearer token.
type Client struct {
	*httpclient.BaseClient
}

// NewClientWithToken creates a new volume client configured to communicate
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

// Attach creates and attaches a volume to a service.
//
// Parameters:
//   - serviceID: The UUID of the service to attach the volume to
//   - req: The attach request containing capacity, optional name, and mount paths
//
// Returns the attach response with volume ID, or an error if the request fails.
func (c *Client) Attach(serviceID string, req AttachRequest) (*AttachResponse, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	}
	if req.Capacity < 1 {
		return nil, fmt.Errorf("capacity must be at least 1 GB")
	}
	if len(req.MountPaths) == 0 {
		return nil, fmt.Errorf("at least one mount path is required")
	}

	var resp AttachResponse
	path := fmt.Sprintf("/volumes/%s", serviceID)
	if err := c.DoJSON("POST", path, req, &resp); err != nil {
		return nil, fmt.Errorf("failed to attach volume: %w", err)
	}

	return &resp, nil
}
