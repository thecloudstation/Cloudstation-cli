package templates

import "time"

// Template represents a CloudStation template
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Visibility  string                 `json:"visibility"`
	Status      string                 `json:"status"`
	Tags        []string               `json:"tags"`
	Author      *string                `json:"author,omitempty"`
	Definition  map[string]interface{} `json:"definition"`
	Deployments float64                `json:"deployments"`
	Slug        *string                `json:"slug,omitempty"`
	TotalApps   float64                `json:"totalApps"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// ListTemplatesParams for filtering templates
type ListTemplatesParams struct {
	Search   string
	Tags     string // comma-separated
	Sort     string // created_DESC, name_ASC, deployments_DESC
	Page     int
	PageSize int
}

// ListTemplatesResponse paginated response
type ListTemplatesResponse struct {
	Data       []Template `json:"data"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// DeployTemplateRequest for POST /templates/deploy
type DeployTemplateRequest struct {
	TemplateID  string                `json:"templateId"`
	ProjectID   string                `json:"projectId"`
	Environment string                `json:"environment,omitempty"`
	Variables   []DeployVariableInput `json:"variables,omitempty"`
	Team        string                `json:"-"` // Team slug passed as query param, not in body
}

// DeployVariableInput for overriding template variables
type DeployVariableInput struct {
	TemplateName string `json:"templateName,omitempty"`
	Key          string `json:"key"`
	Value        string `json:"value"`
}

// DeployTemplateResponse from deployment
type DeployTemplateResponse struct {
	ID                   string   `json:"id"`
	TemplateID           string   `json:"templateId"`
	ProjectID            string   `json:"projectId"`
	ServiceIDs           []string `json:"serviceIds"`
	DeploymentInstanceID string   `json:"deploymentInstanceId"`
	Message              string   `json:"message,omitempty"`
}

// TagsResponse for GET /templates/tags
type TagsResponse struct {
	Tags []string `json:"tags"`
}
