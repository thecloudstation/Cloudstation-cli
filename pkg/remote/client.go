package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// DeploymentResponse represents the response from the backend deployment API
type DeploymentResponse struct {
	DeploymentID string `json:"deploymentId"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// InitUploadResponse represents the response from init-upload API
type InitUploadResponse struct {
	UploadID  string `json:"uploadId"`
	UploadURL string `json:"uploadUrl"`
	MaxSize   int64  `json:"maxSize"`
	ExpiresAt string `json:"expiresAt"`
}

// CompleteUploadResponse represents the response from complete-upload API
type CompleteUploadResponse struct {
	UploadID        string `json:"uploadId"`
	Status          string `json:"status"`
	StorageLocation string `json:"storageLocation"`
}

// Client is the HTTP client for triggering remote deployments via the backend API
type Client struct {
	*httpclient.BaseClient
}

// NewClientWithToken creates a new remote deployment client with JWT Bearer auth
func NewClientWithToken(baseURL, sessionToken string) *Client {
	client := &Client{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}

	// Set Bearer token for JWT authentication
	client.SetHeader("Authorization", "Bearer "+sessionToken)

	return client
}

// TriggerDeployment triggers a deployment for the specified integration
func (c *Client) TriggerDeployment(integrationID string, team string) (*DeploymentResponse, error) {
	// Validate inputs
	if integrationID == "" {
		return nil, fmt.Errorf("integration ID cannot be empty")
	}

	// Build path with optional team query parameter
	path := fmt.Sprintf("/api/external/integrations/deploy/%s", integrationID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, team)
	}

	// Create an empty request body
	reqBody := map[string]interface{}{}
	var resp DeploymentResponse

	// Use DoJSON from BaseClient which handles all headers, JSON marshaling, and error responses
	if err := c.DoJSON("POST", path, reqBody, &resp); err != nil {
		// Check if the error already contains "request failed", avoid double wrapping
		return nil, fmt.Errorf("deployment trigger failed: %w", err)
	}

	return &resp, nil
}

// InitUpload initializes a new upload session and returns a presigned URL
func (c *Client) InitUpload(serviceID string) (*InitUploadResponse, error) {
	if serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	}

	path := "/api/cli/uploads/init"
	reqBody := map[string]string{"serviceId": serviceID}
	var resp InitUploadResponse

	if err := c.DoJSON("POST", path, reqBody, &resp); err != nil {
		return nil, fmt.Errorf("init upload failed: %w", err)
	}

	return &resp, nil
}

// UploadFile uploads a file to the presigned URL
// Note: This method doesn't use the BaseClient since it uploads to an external presigned URL
func (c *Client) UploadFile(presignedURL string, data []byte) error {
	// Create HTTP client with longer timeout for large uploads
	uploadClient := &http.Client{Timeout: 10 * time.Minute}

	req, err := http.NewRequest("PUT", presignedURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-tar")
	req.ContentLength = int64(len(data))

	resp, err := uploadClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CompleteUpload marks an upload as completed
func (c *Client) CompleteUpload(uploadID string, fileSize int64, checksum string) (*CompleteUploadResponse, error) {
	if uploadID == "" {
		return nil, fmt.Errorf("upload ID cannot be empty")
	}

	path := fmt.Sprintf("/api/cli/uploads/%s/complete", uploadID)
	reqBody := map[string]interface{}{
		"fileSize": fileSize,
		"checksum": checksum,
	}
	var resp CompleteUploadResponse

	if err := c.DoJSON("POST", path, reqBody, &resp); err != nil {
		return nil, fmt.Errorf("complete upload failed: %w", err)
	}

	return &resp, nil
}

// TriggerLocalDeploy triggers a deployment using the uploaded source
func (c *Client) TriggerLocalDeploy(uploadID string) (*DeploymentResponse, error) {
	if uploadID == "" {
		return nil, fmt.Errorf("upload ID cannot be empty")
	}

	path := fmt.Sprintf("/api/cli/uploads/%s/deploy", uploadID)
	var resp DeploymentResponse

	// POST request with nil body (no JSON body needed)
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("trigger local deploy failed: %w", err)
	}

	return &resp, nil
}

// DeploymentDetails contains full deployment information including error details
type DeploymentDetails struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	IntegrationID   string            `json:"integrationId"`
	IntegrationName string            `json:"integrationName"`
	Repo            string            `json:"repo"`
	Branch          string            `json:"branch"`
	DispatcherID    string            `json:"dispatcherId"`
	BuildSuccessful bool              `json:"buildSuccessful"`
	Active          bool              `json:"active"`
	CreatedAt       string            `json:"createdAt"`
	Phases          []DeploymentPhase `json:"phases"`
	Error           string            `json:"error,omitempty"`
	ErrorDetails    string            `json:"errorDetails,omitempty"`
}

// DeploymentPhase represents a phase in the deployment process
type DeploymentPhase struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt,omitempty"`
	EndedAt   string `json:"endedAt,omitempty"`
	Error     string `json:"error,omitempty"`
	Logs      string `json:"logs,omitempty"`
}

// GetDeploymentStatus retrieves the status of a deployment by its ID
func (c *Client) GetDeploymentStatus(deploymentID string) (string, error) {
	// Validate input
	if deploymentID == "" {
		return "", fmt.Errorf("deployment ID cannot be empty")
	}

	// Construct the URL path (note: /api prefix required for backend routes)
	path := fmt.Sprintf("/api/v1/deployments/%s", deploymentID)

	// Parse the response to extract just the status field
	var statusResp struct {
		Status string `json:"status"`
	}

	// GET request with no body
	if err := c.DoJSON("GET", path, nil, &statusResp); err != nil {
		// Wrap the error to provide context about what failed
		return "", fmt.Errorf("failed to get deployment status: %w", err)
	}

	return statusResp.Status, nil
}

// GetDeploymentDetails retrieves full deployment details including error information
func (c *Client) GetDeploymentDetails(deploymentID string) (*DeploymentDetails, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/deployments/%s", deploymentID)

	var details DeploymentDetails
	if err := c.DoJSON("GET", path, nil, &details); err != nil {
		return nil, fmt.Errorf("failed to get deployment details: %w", err)
	}

	return &details, nil
}

// GetFailureReason analyzes deployment details and returns an actionable error message
func (d *DeploymentDetails) GetFailureReason() string {
	if d.Status != "FAILED" {
		return ""
	}

	// Check for explicit error
	if d.Error != "" {
		return d.Error
	}

	// Check phases for errors
	for _, phase := range d.Phases {
		if phase.Status == "FAILED" || phase.Error != "" {
			if phase.Error != "" {
				return fmt.Sprintf("%s phase failed: %s", phase.Name, phase.Error)
			}
			return fmt.Sprintf("%s phase failed", phase.Name)
		}
	}

	// No phases means build never started
	if len(d.Phases) == 0 {
		if !d.BuildSuccessful {
			return "Build failed to start - check if the source code was uploaded correctly"
		}
		return "Deployment failed before build started"
	}

	return "Deployment failed (no specific error available)"
}

// GetActionableSuggestions returns suggestions to fix common deployment issues
func (d *DeploymentDetails) GetActionableSuggestions() []string {
	var suggestions []string

	if len(d.Phases) == 0 && !d.BuildSuccessful {
		suggestions = append(suggestions,
			"Verify the source archive was uploaded successfully",
			"Check if the dispatcher service is running",
			"Try running 'cs deploy' again",
		)
	}

	// Check for common issues based on phase errors
	for _, phase := range d.Phases {
		if phase.Error != "" {
			errorLower := strings.ToLower(phase.Error)
			if strings.Contains(errorLower, "dockerfile") {
				suggestions = append(suggestions,
					"Ensure a Dockerfile exists in the root directory",
					"Or use --builder=nixpacks for auto-detection",
				)
			}
			if strings.Contains(errorLower, "npm") || strings.Contains(errorLower, "node") {
				suggestions = append(suggestions,
					"Check package.json for valid dependencies",
					"Ensure node version is specified in package.json engines",
				)
			}
			if strings.Contains(errorLower, "memory") || strings.Contains(errorLower, "oom") {
				suggestions = append(suggestions,
					"Build may need more memory - contact support",
				)
			}
		}
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions,
			"Check the build logs for more details",
			"Verify your project builds locally",
			"Contact support if the issue persists",
		)
	}

	return suggestions
}

// DeployImageResponse represents the response from deploying an image
type DeployImageResponse struct {
	ServiceID    string `json:"serviceId"`
	DeploymentID string `json:"deploymentId"`
	Status       string `json:"status"`
	URL          string `json:"url,omitempty"`
	Message      string `json:"message,omitempty"`
}

// DeployImage deploys a pre-built Docker image to a CloudStation project
func (c *Client) DeployImage(projectID, serviceName, image string, port int) (*DeployImageResponse, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	if image == "" {
		return nil, fmt.Errorf("image cannot be empty")
	}

	path := fmt.Sprintf("/api/v1/projects/%s/services", projectID)
	reqBody := map[string]interface{}{
		"name":  serviceName,
		"image": image,
		"port":  port,
	}

	var resp DeployImageResponse
	if err := c.DoJSON("POST", path, reqBody, &resp); err != nil {
		return nil, fmt.Errorf("deploy image failed: %w", err)
	}

	return &resp, nil
}
