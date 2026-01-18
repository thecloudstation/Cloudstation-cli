package dispatch

import (
	"encoding/json"
	"testing"
)

// TestFlexStringUnmarshalFromNumber tests that FlexString can unmarshal from a number
func TestFlexStringUnmarshalFromNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FlexString
		wantErr  bool
	}{
		{
			name:     "integer as number",
			input:    `{"ownerId": 3694}`,
			expected: "3694",
			wantErr:  false,
		},
		{
			name:     "large integer as number",
			input:    `{"ownerId": 999999}`,
			expected: "999999",
			wantErr:  false,
		},
		{
			name:     "string value",
			input:    `{"ownerId": "3694"}`,
			expected: "3694",
			wantErr:  false,
		},
		{
			name:     "string with text",
			input:    `{"ownerId": "owner_123"}`,
			expected: "owner_123",
			wantErr:  false,
		},
		{
			name:     "float number",
			input:    `{"ownerId": 3694.0}`,
			expected: "3694",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				OwnerID FlexString `json:"ownerId"`
			}
			err := json.Unmarshal([]byte(tt.input), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.OwnerID != tt.expected {
				t.Errorf("OwnerID = %v, expected %v", result.OwnerID, tt.expected)
			}
		})
	}
}

// TestDeployImageParamsUnmarshalOwnerID tests that DeployImageParams can unmarshal ownerId from number
func TestDeployImageParamsUnmarshalOwnerID(t *testing.T) {
	jsonWithNumberOwnerID := `{
		"imageName": "test/image",
		"imageTag": "latest",
		"ownerId": 3694,
		"userId": 1234,
		"serviceId": "svc_123",
		"deploymentJobId": 5678,
		"name": "test-app",
		"cpu": 500,
		"ram": 1024,
		"deploymentId": "dep_123",
		"jobId": "job_123",
		"deploy": "nomad-pack"
	}`

	var params DeployImageParams
	err := json.Unmarshal([]byte(jsonWithNumberOwnerID), &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal DeployImageParams with number ownerId: %v", err)
	}

	if params.OwnerID != "3694" {
		t.Errorf("OwnerID = %v, expected '3694'", params.OwnerID)
	}
}

// TestDeployImageParamsUnmarshalOwnerIDString tests that DeployImageParams can unmarshal ownerId from string
func TestDeployImageParamsUnmarshalOwnerIDString(t *testing.T) {
	jsonWithStringOwnerID := `{
		"imageName": "test/image",
		"imageTag": "latest",
		"ownerId": "owner_3694",
		"userId": 1234,
		"serviceId": "svc_123",
		"deploymentJobId": 5678,
		"name": "test-app",
		"cpu": 500,
		"ram": 1024,
		"deploymentId": "dep_123",
		"jobId": "job_123",
		"deploy": "nomad-pack"
	}`

	var params DeployImageParams
	err := json.Unmarshal([]byte(jsonWithStringOwnerID), &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal DeployImageParams with string ownerId: %v", err)
	}

	if params.OwnerID != "owner_3694" {
		t.Errorf("OwnerID = %v, expected 'owner_3694'", params.OwnerID)
	}
}

// TestDeployRepositoryParamsUnmarshalOwnerID tests that DeployRepositoryParams can unmarshal ownerId from number
func TestDeployRepositoryParamsUnmarshalOwnerID(t *testing.T) {
	jsonWithNumberOwnerID := `{
		"imageName": "test/image",
		"imageTag": "latest",
		"ownerId": 3694,
		"userId": 1234,
		"serviceId": "svc_123",
		"deploymentJobId": 5678,
		"name": "test-app",
		"cpu": 500,
		"ram": 1024,
		"deploymentId": "dep_123",
		"jobId": "job_123",
		"deploy": "nomad-pack"
	}`

	var params DeployRepositoryParams
	err := json.Unmarshal([]byte(jsonWithNumberOwnerID), &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal DeployRepositoryParams with number ownerId: %v", err)
	}

	if params.OwnerID != "3694" {
		t.Errorf("OwnerID = %v, expected '3694'", params.OwnerID)
	}
}

// TestFlexIntUnmarshalFromString tests that FlexInt can unmarshal from string (for backward compatibility)
func TestFlexIntUnmarshalFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected FlexInt
		wantErr  bool
	}{
		{
			name:     "number value",
			input:    `{"userId": 1234}`,
			expected: 1234,
			wantErr:  false,
		},
		{
			name:     "string number",
			input:    `{"userId": "5678"}`,
			expected: 5678,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				UserID FlexInt `json:"userId"`
			}
			err := json.Unmarshal([]byte(tt.input), &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.UserID != tt.expected {
				t.Errorf("UserID = %v, expected %v", result.UserID, tt.expected)
			}
		})
	}
}

func TestDeployImageParamsUnmarshalBuildStartCommand(t *testing.T) {
	jsonData := `{
		"imageName": "minio/minio",
		"imageTag": "latest",
		"jobId": "cst-minio-test",
		"serviceId": "svc_123",
		"deploymentId": "dep_123",
		"ownerId": "team_123",
		"userId": 1234,
		"deploymentJobId": 5678,
		"build": {
			"builder": "noop",
			"startCommand": "minio server --address 0.0.0.0:9000 /data"
		}
	}`

	var params DeployImageParams
	err := json.Unmarshal([]byte(jsonData), &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if params.Build.StartCommand != "minio server --address 0.0.0.0:9000 /data" {
		t.Errorf("Build.StartCommand: expected %q, got %q",
			"minio server --address 0.0.0.0:9000 /data", params.Build.StartCommand)
	}

	if params.Build.Builder != "noop" {
		t.Errorf("Build.Builder: expected %q, got %q", "noop", params.Build.Builder)
	}
}
