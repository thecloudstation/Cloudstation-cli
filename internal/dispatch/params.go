package dispatch

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
)

// ParseTaskType reads and validates the NOMAD_META_TASK environment variable
func ParseTaskType() (TaskType, error) {
	taskStr := os.Getenv("NOMAD_META_TASK")
	if taskStr == "" {
		return "", fmt.Errorf("NOMAD_META_TASK environment variable is not set")
	}

	taskType := TaskType(taskStr)

	// Validate task type
	switch taskType {
	case TaskDeployRepository, TaskRedeployRepository, TaskDeployImage, TaskDestroyJob:
		return taskType, nil
	default:
		return "", fmt.Errorf("unknown task type: %s", taskStr)
	}
}

// ParseParams reads and parses the NOMAD_META_PARAMS environment variable
// The params are expected to be base64-encoded JSON (matching cs-runner behavior)
func ParseParams(taskType TaskType) (interface{}, error) {
	logger := hclog.New(&hclog.LoggerOptions{
		Output: os.Stderr,
		Level:  hclog.Info,
	})

	paramsStr := os.Getenv("NOMAD_META_PARAMS")
	if paramsStr == "" {
		return nil, fmt.Errorf("NOMAD_META_PARAMS environment variable is not set")
	}

	// Decode base64 to get JSON bytes
	jsonBytes, err := base64.StdEncoding.DecodeString(paramsStr)
	if err != nil {
		// Log preview of base64 string for debugging
		preview := paramsStr
		if len(preview) > 200 {
			preview = preview[:200]
		}
		logger.Error("Failed to decode base64", "params_preview", preview)
		return nil, fmt.Errorf("failed to decode base64 parameters: %w", err)
	}

	switch taskType {
	case TaskDeployRepository:
		var params DeployRepositoryParams
		if err := json.Unmarshal(jsonBytes, &params); err != nil {
			logger.Error("Failed to parse JSON", "json", string(jsonBytes))
			return nil, fmt.Errorf("failed to parse deploy-repository parameters: %w", err)
		}

		// Inject backend configuration from environment variables if not set in params
		if params.BackendURL == "" {
			params.BackendURL = os.Getenv("BACKEND_URL")
		}
		if params.AccessToken == "" {
			params.AccessToken = os.Getenv("ACCESS_TOKEN")
		}

		// Debug logging to trace backend config injection
		logger.Info("Backend config after env injection (deploy-repository)",
			"params.BackendURL", params.BackendURL,
			"params.AccessToken_len", len(params.AccessToken),
			"env_BACKEND_URL", os.Getenv("BACKEND_URL"),
			"env_ACCESS_TOKEN_len", len(os.Getenv("ACCESS_TOKEN")))

		if err := validateDeployRepositoryParams(params, logger); err != nil {
			LogParameterValidation(logger, params)
			return nil, err
		}
		return params, nil

	case TaskRedeployRepository:
		var params DeployRepositoryParams
		if err := json.Unmarshal(jsonBytes, &params); err != nil {
			logger.Error("Failed to parse JSON", "json", string(jsonBytes))
			return nil, fmt.Errorf("failed to parse redeploy-repository parameters: %w", err)
		}

		// Inject backend configuration from environment variables if not set in params
		if params.BackendURL == "" {
			params.BackendURL = os.Getenv("BACKEND_URL")
		}
		if params.AccessToken == "" {
			params.AccessToken = os.Getenv("ACCESS_TOKEN")
		}

		if err := validateDeployRepositoryParams(params, logger); err != nil {
			LogParameterValidation(logger, params)
			return nil, err
		}
		return params, nil

	case TaskDeployImage:
		var params DeployImageParams
		if err := json.Unmarshal(jsonBytes, &params); err != nil {
			logger.Error("Failed to parse JSON", "json", string(jsonBytes))
			return nil, fmt.Errorf("failed to parse deploy-image parameters: %w", err)
		}

		// Inject backend configuration from environment variables if not set in params
		if params.BackendURL == "" {
			params.BackendURL = os.Getenv("BACKEND_URL")
		}
		if params.AccessToken == "" {
			params.AccessToken = os.Getenv("ACCESS_TOKEN")
		}

		if err := validateDeployImageParams(params, logger); err != nil {
			LogParameterValidation(logger, params)
			return nil, err
		}
		return params, nil

	case TaskDestroyJob:
		var params DestroyJobParams
		if err := json.Unmarshal(jsonBytes, &params); err != nil {
			logger.Error("Failed to parse JSON", "json", string(jsonBytes))
			return nil, fmt.Errorf("failed to parse destroy-job parameters: %w", err)
		}
		if err := validateDestroyJobParams(params, logger); err != nil {
			LogParameterValidation(logger, params)
			return nil, err
		}
		return params, nil

	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}
}

// validateDeployRepositoryParams validates deploy repository parameters
func validateDeployRepositoryParams(params DeployRepositoryParams, logger hclog.Logger) error {
	if params.JobID == "" {
		logger.Error("Validation failed", "field", "jobId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("jobId is required (received params: repository=%s, branch=%s)", params.Repository, params.Branch)
	}

	// For local_upload source type, require sourceUrl instead of repository
	if params.SourceType == "local_upload" {
		if params.SourceUrl == "" {
			logger.Error("Validation failed", "field", "sourceUrl", "params", fmt.Sprintf("%+v", params))
			return fmt.Errorf("sourceUrl is required for local_upload source type (received params: jobId=%s, sourceType=%s)", params.JobID, params.SourceType)
		}
		// Branch is not required for local uploads
	} else {
		// For git deployments, require repository and branch
		if params.Repository == "" {
			logger.Error("Validation failed", "field", "repository", "params", fmt.Sprintf("%+v", params))
			return fmt.Errorf("repository is required (received params: jobId=%s, branch=%s)", params.JobID, params.Branch)
		}
		if params.Branch == "" {
			logger.Error("Validation failed", "field", "branch", "params", fmt.Sprintf("%+v", params))
			return fmt.Errorf("branch is required (received params: jobId=%s, repository=%s)", params.JobID, params.Repository)
		}
	}

	if params.DeploymentID == "" {
		logger.Error("Validation failed", "field", "deploymentId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("deploymentId is required (received params: jobId=%s, repository=%s)", params.JobID, params.Repository)
	}
	if params.ServiceID == "" {
		logger.Error("Validation failed", "field", "serviceId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("serviceId is required (received params: jobId=%s, repository=%s)", params.JobID, params.Repository)
	}
	return nil
}

// validateDeployImageParams validates deploy image parameters
func validateDeployImageParams(params DeployImageParams, logger hclog.Logger) error {
	if params.JobID == "" {
		logger.Error("Validation failed", "field", "jobId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("jobId is required (received params: imageName=%s, deploymentId=%s)", params.ImageName, params.DeploymentID)
	}
	if params.ImageName == "" {
		logger.Error("Validation failed", "field", "imageName", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("imageName is required (received params: jobId=%s, deploymentId=%s)", params.JobID, params.DeploymentID)
	}
	if params.DeploymentID == "" {
		logger.Error("Validation failed", "field", "deploymentId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("deploymentId is required (received params: jobId=%s, imageName=%s)", params.JobID, params.ImageName)
	}
	if params.ServiceID == "" {
		logger.Error("Validation failed", "field", "serviceId", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("serviceId is required (received params: jobId=%s, imageName=%s)", params.JobID, params.ImageName)
	}
	return nil
}

// validateDestroyJobParams validates destroy job parameters
func validateDestroyJobParams(params DestroyJobParams, logger hclog.Logger) error {
	if len(params.Jobs) == 0 {
		logger.Error("Validation failed", "field", "jobs", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("at least one job is required")
	}
	if params.Reason == "" {
		logger.Error("Validation failed", "field", "reason", "params", fmt.Sprintf("%+v", params))
		return fmt.Errorf("reason is required")
	}
	return nil
}
