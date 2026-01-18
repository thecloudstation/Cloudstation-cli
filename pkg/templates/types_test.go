package templates

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTemplate_JSONMarshal(t *testing.T) {
	author := "test-author"
	slug := "test-slug"
	template := Template{
		ID:          "tmpl-123",
		Name:        "Test Template",
		Description: "A test template",
		Image:       "nginx:latest",
		Visibility:  "public",
		Status:      "active",
		Tags:        []string{"web", "nginx"},
		Author:      &author,
		Definition:  map[string]interface{}{"key": "value"},
		Deployments: 42,
		Slug:        &slug,
		TotalApps:   3,
		CreatedAt:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify key fields are present
	jsonStr := string(data)
	expectedFields := []string{
		`"id":"tmpl-123"`,
		`"name":"Test Template"`,
		`"visibility":"public"`,
		`"tags":["web","nginx"]`,
		`"author":"test-author"`,
		`"deployments":42`,
		`"totalApps":3`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("Expected JSON to contain %s, got %s", field, jsonStr)
		}
	}
}

func TestTemplate_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": "tmpl-456",
		"name": "Unmarshal Test",
		"description": "Testing unmarshal",
		"image": "redis:7",
		"visibility": "private",
		"status": "active",
		"tags": ["cache", "database"],
		"deployments": 100,
		"totalApps": 5,
		"createdAt": "2024-01-20T15:00:00Z"
	}`

	var template Template
	if err := json.Unmarshal([]byte(jsonData), &template); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if template.ID != "tmpl-456" {
		t.Errorf("Expected ID tmpl-456, got %s", template.ID)
	}
	if template.Name != "Unmarshal Test" {
		t.Errorf("Expected Name 'Unmarshal Test', got %s", template.Name)
	}
	if len(template.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(template.Tags))
	}
	if template.Deployments != 100 {
		t.Errorf("Expected Deployments 100, got %f", template.Deployments)
	}
	if template.Author != nil {
		t.Errorf("Expected Author to be nil, got %v", template.Author)
	}
}

func TestTemplate_JSONRoundTrip(t *testing.T) {
	author := "roundtrip-author"
	slug := "roundtrip-slug"
	original := Template{
		ID:          "tmpl-roundtrip",
		Name:        "Roundtrip Template",
		Description: "Testing roundtrip",
		Image:       "postgres:15",
		Visibility:  "private",
		Status:      "active",
		Tags:        []string{"database", "sql"},
		Author:      &author,
		Definition:  map[string]interface{}{"env": "production"},
		Deployments: 250,
		Slug:        &slug,
		TotalApps:   10,
		CreatedAt:   time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Template
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if original.ID != decoded.ID {
		t.Errorf("ID mismatch: %s != %s", original.ID, decoded.ID)
	}
	if original.Name != decoded.Name {
		t.Errorf("Name mismatch: %s != %s", original.Name, decoded.Name)
	}
	if *original.Author != *decoded.Author {
		t.Errorf("Author mismatch: %s != %s", *original.Author, *decoded.Author)
	}
	if *original.Slug != *decoded.Slug {
		t.Errorf("Slug mismatch: %s != %s", *original.Slug, *decoded.Slug)
	}
}

func TestListTemplatesResponse_JSON(t *testing.T) {
	resp := ListTemplatesResponse{
		Data: []Template{
			{ID: "t1", Name: "Template 1"},
			{ID: "t2", Name: "Template 2"},
		},
		Total:      50,
		Page:       1,
		PageSize:   10,
		TotalPages: 5,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled ListTemplatesResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(unmarshaled.Data) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(unmarshaled.Data))
	}
	if unmarshaled.Total != 50 {
		t.Errorf("Expected Total 50, got %d", unmarshaled.Total)
	}
	if unmarshaled.TotalPages != 5 {
		t.Errorf("Expected TotalPages 5, got %d", unmarshaled.TotalPages)
	}
}

func TestListTemplatesResponse_EmptyData(t *testing.T) {
	resp := ListTemplatesResponse{
		Data:       []Template{},
		Total:      0,
		Page:       1,
		PageSize:   10,
		TotalPages: 0,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"data":[]`) {
		t.Errorf("Expected empty data array in JSON, got %s", jsonStr)
	}
}

func TestDeployTemplateRequest_JSON(t *testing.T) {
	req := DeployTemplateRequest{
		TemplateID:  "tmpl-789",
		ProjectID:   "proj-123",
		Environment: "production",
		Variables: []DeployVariableInput{
			{Key: "DB_HOST", Value: "localhost"},
			{TemplateName: "db", Key: "PORT", Value: "5432"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"templateId":"tmpl-789"`) {
		t.Error("Missing templateId in JSON")
	}
	if !strings.Contains(jsonStr, `"projectId":"proj-123"`) {
		t.Error("Missing projectId in JSON")
	}
	if !strings.Contains(jsonStr, `"environment":"production"`) {
		t.Error("Missing environment in JSON")
	}
}

func TestDeployTemplateRequest_OmitEmpty(t *testing.T) {
	// Test that optional fields with omitempty are not included when empty
	req := DeployTemplateRequest{
		TemplateID: "tmpl-minimal",
		ProjectID:  "proj-minimal",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, `"environment"`) {
		t.Error("Empty environment should not be in JSON due to omitempty")
	}
	if strings.Contains(jsonStr, `"variables"`) {
		t.Error("Empty variables should not be in JSON due to omitempty")
	}
}

func TestDeployTemplateRequest_Unmarshal(t *testing.T) {
	jsonData := `{
		"templateId": "tmpl-unmarshal",
		"projectId": "proj-unmarshal",
		"environment": "staging",
		"variables": [
			{"key": "API_KEY", "value": "secret123"},
			{"templateName": "cache", "key": "TTL", "value": "3600"}
		]
	}`

	var req DeployTemplateRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.TemplateID != "tmpl-unmarshal" {
		t.Errorf("Expected TemplateID tmpl-unmarshal, got %s", req.TemplateID)
	}
	if req.ProjectID != "proj-unmarshal" {
		t.Errorf("Expected ProjectID proj-unmarshal, got %s", req.ProjectID)
	}
	if req.Environment != "staging" {
		t.Errorf("Expected Environment staging, got %s", req.Environment)
	}
	if len(req.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(req.Variables))
	}
	if req.Variables[0].Key != "API_KEY" {
		t.Errorf("Expected first variable key API_KEY, got %s", req.Variables[0].Key)
	}
	if req.Variables[1].TemplateName != "cache" {
		t.Errorf("Expected second variable templateName cache, got %s", req.Variables[1].TemplateName)
	}
}

func TestDeployVariableInput_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    DeployVariableInput
		expected string
	}{
		{
			name: "with template name",
			input: DeployVariableInput{
				TemplateName: "db",
				Key:          "PORT",
				Value:        "5432",
			},
			expected: `"templateName":"db"`,
		},
		{
			name: "without template name",
			input: DeployVariableInput{
				Key:   "API_URL",
				Value: "https://api.example.com",
			},
			expected: `"key":"API_URL"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			jsonStr := string(data)
			if !strings.Contains(jsonStr, tt.expected) {
				t.Errorf("Expected JSON to contain %s, got %s", tt.expected, jsonStr)
			}
		})
	}
}

func TestDeployVariableInput_OmitEmpty(t *testing.T) {
	// Without template name, it should be omitted
	input := DeployVariableInput{
		Key:   "TEST",
		Value: "value",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, `"templateName"`) {
		t.Error("Empty templateName should not be in JSON due to omitempty")
	}
}

func TestDeployTemplateResponse_JSON(t *testing.T) {
	jsonData := `{
		"id": "deploy-123",
		"templateId": "tmpl-1",
		"projectId": "proj-1",
		"serviceIds": ["svc-1", "svc-2", "svc-3"],
		"deploymentInstanceId": "di-456",
		"message": "Deployment started"
	}`

	var resp DeployTemplateResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if resp.ID != "deploy-123" {
		t.Errorf("Expected ID deploy-123, got %s", resp.ID)
	}
	if len(resp.ServiceIDs) != 3 {
		t.Errorf("Expected 3 service IDs, got %d", len(resp.ServiceIDs))
	}
	if resp.DeploymentInstanceID != "di-456" {
		t.Errorf("Expected DeploymentInstanceID di-456, got %s", resp.DeploymentInstanceID)
	}
	if resp.Message != "Deployment started" {
		t.Errorf("Expected Message 'Deployment started', got %s", resp.Message)
	}
}

func TestDeployTemplateResponse_Marshal(t *testing.T) {
	resp := DeployTemplateResponse{
		ID:                   "deploy-marshal",
		TemplateID:           "tmpl-marshal",
		ProjectID:            "proj-marshal",
		ServiceIDs:           []string{"svc-a", "svc-b"},
		DeploymentInstanceID: "di-marshal",
		Message:              "Success",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{
		`"id":"deploy-marshal"`,
		`"templateId":"tmpl-marshal"`,
		`"projectId":"proj-marshal"`,
		`"serviceIds":["svc-a","svc-b"]`,
		`"deploymentInstanceId":"di-marshal"`,
		`"message":"Success"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("Expected JSON to contain %s, got %s", field, jsonStr)
		}
	}
}

func TestDeployTemplateResponse_OmitEmpty(t *testing.T) {
	resp := DeployTemplateResponse{
		ID:                   "deploy-no-message",
		TemplateID:           "tmpl-1",
		ProjectID:            "proj-1",
		ServiceIDs:           []string{"svc-1"},
		DeploymentInstanceID: "di-1",
		// Message is empty
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, `"message"`) {
		t.Error("Empty message should not be in JSON due to omitempty")
	}
}

func TestTagsResponse_JSON(t *testing.T) {
	resp := TagsResponse{
		Tags: []string{"database", "web", "cache", "monitoring"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled TagsResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(unmarshaled.Tags) != 4 {
		t.Errorf("Expected 4 tags, got %d", len(unmarshaled.Tags))
	}

	expectedTags := []string{"database", "web", "cache", "monitoring"}
	for i, tag := range expectedTags {
		if unmarshaled.Tags[i] != tag {
			t.Errorf("Expected tag[%d] to be %s, got %s", i, tag, unmarshaled.Tags[i])
		}
	}
}

func TestTagsResponse_EmptyTags(t *testing.T) {
	resp := TagsResponse{
		Tags: []string{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"tags":[]`) {
		t.Errorf("Expected empty tags array in JSON, got %s", jsonStr)
	}
}

func TestTemplate_NilOptionalFields(t *testing.T) {
	template := Template{
		ID:          "tmpl-nil",
		Name:        "Nil Fields Template",
		Description: "Testing nil optional fields",
		Image:       "alpine:latest",
		Visibility:  "public",
		Status:      "active",
		Tags:        []string{"test"},
		Author:      nil, // nil optional field
		Definition:  nil,
		Deployments: 0,
		Slug:        nil, // nil optional field
		TotalApps:   0,
		CreatedAt:   time.Now(),
	}

	data, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	// Author and Slug should be omitted due to omitempty
	if strings.Contains(jsonStr, `"author"`) {
		t.Error("nil author should not be in JSON due to omitempty")
	}
	if strings.Contains(jsonStr, `"slug"`) {
		t.Error("nil slug should not be in JSON due to omitempty")
	}
}

func TestListTemplatesParams_FieldValues(t *testing.T) {
	// Test that ListTemplatesParams has the correct field types
	params := ListTemplatesParams{
		Search:   "nginx",
		Tags:     "web,database",
		Sort:     "created_DESC",
		Page:     2,
		PageSize: 25,
	}

	if params.Search != "nginx" {
		t.Errorf("Expected Search 'nginx', got %s", params.Search)
	}
	if params.Tags != "web,database" {
		t.Errorf("Expected Tags 'web,database', got %s", params.Tags)
	}
	if params.Sort != "created_DESC" {
		t.Errorf("Expected Sort 'created_DESC', got %s", params.Sort)
	}
	if params.Page != 2 {
		t.Errorf("Expected Page 2, got %d", params.Page)
	}
	if params.PageSize != 25 {
		t.Errorf("Expected PageSize 25, got %d", params.PageSize)
	}
}
