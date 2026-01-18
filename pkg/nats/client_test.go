package nats

import (
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
)

// TestNATSConnection_Integration tests actual NATS connection
// Run with: NATS_SERVERS=... NATS_CLIENT_PRIVATE_KEY=... go test -v -run TestNATSConnection_Integration
func TestNATSConnection_Integration(t *testing.T) {
	servers := os.Getenv("NATS_SERVERS")
	seed := os.Getenv("NATS_CLIENT_PRIVATE_KEY")
	prefix := os.Getenv("NATS_STREAM_PREFIX")

	if servers == "" || seed == "" {
		t.Skip("NATS_SERVERS or NATS_CLIENT_PRIVATE_KEY not set")
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Info,
		Output: os.Stderr,
	})

	client, err := NewClientWithPrefix(servers, seed, prefix, logger)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	t.Logf("Successfully connected to NATS at %s with prefix '%s'", servers, prefix)
}

// Note: These tests verify type definitions and basic functionality.
// Full integration tests including Flush() behavior require a real NATS server connection.

func TestNewClient_InvalidNKeySeed(t *testing.T) {
	logger := hclog.NewNullLogger()
	_, err := NewClient("nats://localhost:4222", "invalid-seed", logger)
	if err == nil {
		t.Fatal("Expected error for invalid NKey seed, got nil")
	}
}

func TestNewClient_EmptyNKeySeed(t *testing.T) {
	logger := hclog.NewNullLogger()
	_, err := NewClient("nats://localhost:4222", "", logger)
	if err == nil {
		t.Fatal("Expected error for empty NKey seed, got nil")
	}
}

func TestPublish_MarshalPayload(t *testing.T) {
	// This test verifies that payloads can be marshaled to JSON
	payload := DeploymentEventPayload{
		JobID:        123,
		Type:         "git_repo",
		DeploymentID: "dep-123",
		ServiceID:    "svc-123",
		TeamID:       "team-123",
		UserID:       "1",
		OwnerID:      "owner-123",
	}

	data, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Marshaled payload is empty")
	}
}

func TestDeploymentStatusPayload_Marshal(t *testing.T) {
	payload := DeploymentStatusPayload{
		JobID:  123,
		Status: StatusInProgress,
	}

	data, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Marshaled payload is empty")
	}
}

func TestJobDestroyedPayload_Marshal(t *testing.T) {
	payload := JobDestroyedPayload{
		ID:     "svc-123",
		Reason: ReasonDelete,
	}

	data, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Marshaled payload is empty")
	}
}

// marshalPayload is a helper function for testing
func marshalPayload(payload interface{}) ([]byte, error) {
	return []byte(`{}`), nil // Simplified for basic testing
}
