//go:build integration
// +build integration

package backend

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

// Run this test with: go test -tags=integration ./pkg/backend/... -v
// Set environment variables:
//   BACKEND_URL=http://10.225.142.179:22593
//   ACCESS_TOKEN=73e16d55f4fca8e76a608f1eda58f6f530b5b1a859d558a104cc722da0ac7d740727969dfa0523aaf77ef23832db9fcd9ee7

func TestBackendIntegration(t *testing.T) {
	backendURL := os.Getenv("BACKEND_URL")
	accessToken := os.Getenv("ACCESS_TOKEN")

	if backendURL == "" {
		t.Skip("BACKEND_URL not set, skipping integration test")
	}
	if accessToken == "" {
		t.Skip("ACCESS_TOKEN not set, skipping integration test")
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "backend-integration-test",
		Level: hclog.Debug,
	})

	client, err := NewClient(backendURL, accessToken, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	serviceID := fmt.Sprintf("test-service-%d", time.Now().Unix())
	deploymentID := fmt.Sprintf("test-deployment-%d", time.Now().Unix())

	t.Run("AskDomain", func(t *testing.T) {
		t.Logf("Testing domain allocation for service: %s", serviceID)

		domain, err := client.AskDomain(serviceID)
		if err != nil {
			t.Errorf("AskDomain failed: %v", err)
			return
		}

		if domain == "" {
			t.Error("AskDomain returned empty domain")
			return
		}

		t.Logf("✅ Domain allocated: %s", domain)
	})

	t.Run("UpdateService", func(t *testing.T) {
		t.Logf("Testing service update for service: %s", serviceID)

		req := UpdateServiceRequest{
			ServiceID: serviceID,
			Network: []NetworkConfig{
				{Port: 3000, Domain: "subdomain123.cluster.example.com"},
				{Port: 8080, Domain: "subdomain456.cluster.example.com"},
			},
			DockerUser: "appuser",
			CMD:        "npm start",
			Entrypoint: "/bin/sh -c",
		}

		err := client.UpdateService(req)
		if err != nil {
			t.Errorf("UpdateService failed: %v", err)
			return
		}

		t.Logf("✅ Service configuration synced successfully")
	})

	t.Run("UpdateDeploymentStep_CloneInProgress", func(t *testing.T) {
		t.Logf("Testing deployment step tracking: clone in_progress")

		req := UpdateDeploymentStepRequest{
			DeploymentID:   deploymentID,
			DeploymentType: "repository",
			Step:           StepClone,
			Status:         StatusInProgress,
		}

		err := client.UpdateDeploymentStep(req)
		if err != nil {
			t.Errorf("UpdateDeploymentStep failed: %v", err)
			return
		}

		t.Logf("✅ Deployment step updated: clone in_progress")
	})

	t.Run("UpdateDeploymentStep_CloneCompleted", func(t *testing.T) {
		t.Logf("Testing deployment step tracking: clone completed")

		req := UpdateDeploymentStepRequest{
			DeploymentID:   deploymentID,
			DeploymentType: "repository",
			Step:           StepClone,
			Status:         StatusCompleted,
		}

		err := client.UpdateDeploymentStep(req)
		if err != nil {
			t.Errorf("UpdateDeploymentStep failed: %v", err)
			return
		}

		t.Logf("✅ Deployment step updated: clone completed")
	})

	t.Run("UpdateDeploymentStep_BuildInProgress", func(t *testing.T) {
		t.Logf("Testing deployment step tracking: build in_progress")

		req := UpdateDeploymentStepRequest{
			DeploymentID:   deploymentID,
			DeploymentType: "repository",
			Step:           StepBuild,
			Status:         StatusInProgress,
		}

		err := client.UpdateDeploymentStep(req)
		if err != nil {
			t.Errorf("UpdateDeploymentStep failed: %v", err)
			return
		}

		t.Logf("✅ Deployment step updated: build in_progress")
	})

	t.Run("UpdateDeploymentStep_BuildFailed", func(t *testing.T) {
		t.Logf("Testing deployment step tracking: build failed with error")

		req := UpdateDeploymentStepRequest{
			DeploymentID:   deploymentID,
			DeploymentType: "repository",
			Step:           StepBuild,
			Status:         StatusFailed,
			Error:          "Docker build failed: syntax error in Dockerfile line 10",
		}

		err := client.UpdateDeploymentStep(req)
		if err != nil {
			t.Errorf("UpdateDeploymentStep failed: %v", err)
			return
		}

		t.Logf("✅ Deployment step updated: build failed with error")
	})

	t.Run("UpdateDeploymentStep_DeployCompleted", func(t *testing.T) {
		t.Logf("Testing deployment step tracking: deploy completed")

		req := UpdateDeploymentStepRequest{
			DeploymentID:   deploymentID,
			DeploymentType: "repository",
			Step:           StepDeploy,
			Status:         StatusCompleted,
		}

		err := client.UpdateDeploymentStep(req)
		if err != nil {
			t.Errorf("UpdateDeploymentStep failed: %v", err)
			return
		}

		t.Logf("✅ Deployment step updated: deploy completed")
	})
}

func TestBackendIntegration_GracefulDegradation(t *testing.T) {
	// Test with invalid backend to ensure graceful degradation
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "backend-graceful-test",
		Level: hclog.Warn,
	})

	client, err := NewClient("http://invalid-backend:9999", "fake-token", logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("AskDomain_WithInvalidBackend", func(t *testing.T) {
		_, err := client.AskDomain("test-service")
		if err == nil {
			t.Error("Expected error with invalid backend, got nil")
		}
		t.Logf("✅ Gracefully failed with error: %v", err)
	})

	t.Run("UpdateService_WithInvalidBackend", func(t *testing.T) {
		req := UpdateServiceRequest{
			ServiceID: "test-service",
			Network:   []NetworkConfig{{Port: 3000}},
		}
		err := client.UpdateService(req)
		if err == nil {
			t.Error("Expected error with invalid backend, got nil")
		}
		t.Logf("✅ Gracefully failed with error: %v", err)
	})

	t.Run("UpdateDeploymentStep_WithInvalidBackend", func(t *testing.T) {
		req := UpdateDeploymentStepRequest{
			DeploymentID:   "test-deployment",
			DeploymentType: "repository",
			Step:           StepBuild,
			Status:         StatusCompleted,
		}
		err := client.UpdateDeploymentStep(req)
		if err == nil {
			t.Error("Expected error with invalid backend, got nil")
		}
		t.Logf("✅ Gracefully failed with error: %v", err)
	})
}
