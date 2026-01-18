package templates

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// Client is the HTTP client for interacting with the CloudStation template API
type Client struct {
	*httpclient.BaseClient
}

// NewClientWithToken creates a new template client with JWT Bearer authentication
func NewClientWithToken(baseURL, sessionToken string) *Client {
	client := &Client{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+sessionToken)
	return client
}

// List retrieves templates with optional filtering and pagination
func (c *Client) List(params ListTemplatesParams) (*ListTemplatesResponse, error) {
	query := url.Values{}
	if params.Search != "" {
		query.Set("search", params.Search)
	}
	if params.Tags != "" {
		query.Set("tags", params.Tags)
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(params.PageSize))
	}

	path := "/templates"
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}

	var resp ListTemplatesResponse
	if err := c.DoJSON("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list templates failed: %w", err)
	}
	return &resp, nil
}

// Get retrieves a single template by ID or slug
func (c *Client) Get(idOrSlug string) (*Template, error) {
	path := fmt.Sprintf("/templates/%s", idOrSlug)
	// API returns {"data": {...}} wrapper
	var resp struct {
		Data Template `json:"data"`
	}
	if err := c.DoJSON("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("get template failed: %w", err)
	}
	return &resp.Data, nil
}

// GetTags retrieves the list of available template tags
func (c *Client) GetTags() ([]string, error) {
	var resp TagsResponse
	if err := c.DoJSON("GET", "/templates/tags", nil, &resp); err != nil {
		return nil, fmt.Errorf("get tags failed: %w", err)
	}
	return resp.Tags, nil
}

// Deploy deploys a template to create new services
func (c *Client) Deploy(req DeployTemplateRequest) (*DeployTemplateResponse, error) {
	path := "/templates/deploy"
	if req.Team != "" {
		path = fmt.Sprintf("/templates/deploy?team=%s", url.QueryEscape(req.Team))
	}

	var resp DeployTemplateResponse
	if err := c.DoJSON("POST", path, req, &resp); err != nil {
		return nil, fmt.Errorf("deploy template failed: %w", err)
	}
	return &resp, nil
}
