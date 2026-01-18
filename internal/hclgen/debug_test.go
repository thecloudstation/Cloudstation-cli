package hclgen

import (
	"fmt"
	"testing"
)

func TestDebugGeneration(t *testing.T) {
	params := DeploymentParams{
		JobID:        "test-job",
		BuilderType:  "",           // Empty like payload
		DeployType:   "nomad-pack", // With hyphen like payload
		ImageName:    "test-image",
		ImageTag:     "latest",
		ReplicaCount: 1,
	}

	config, err := GenerateConfig(params)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	fmt.Println("========== GENERATED HCL ==========")
	fmt.Println(config)
	fmt.Println("===================================")
}
