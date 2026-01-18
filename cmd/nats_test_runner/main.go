package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	natspkg "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/nats"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	servers := os.Getenv("NATS_SERVERS")
	nkey := os.Getenv("NATS_CLIENT_PRIVATE_KEY")
	prefix := os.Getenv("NATS_STREAM_PREFIX") // may be empty
	if prefix == "" {
		prefix = "cs" // default prefix
	}

	fmt.Printf("=== NATS Integration Test ===\n")
	fmt.Printf("Server: %s\n", servers)
	fmt.Printf("Prefix: %s\n", prefix)

	// 0. First ensure the JetStream stream exists
	fmt.Println("\n[0] Ensuring JetStream stream exists...")
	if err := ensureStream(servers, nkey, prefix); err != nil {
		fmt.Printf("❌ Stream setup FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Stream ready!")

	// 1. Test NATS connection with NKey auth
	fmt.Println("\n[1] Connecting to NATS...")
	client, err := nats.NewClientWithPrefix(servers, nkey, prefix, logger)
	if err != nil {
		fmt.Printf("❌ NATS connection FAILED: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	fmt.Println("✓ NATS connected!")

	// 2. Create a test deployment ID
	deploymentID := fmt.Sprintf("test-%d", time.Now().Unix())
	fmt.Printf("\n[2] Testing log publishing (deploymentID: %s)\n", deploymentID)

	// 3. Create LogWriter (same as dispatch.go does)
	stdoutWriter := nats.NewLogWriter(
		client,
		deploymentID,
		12345,          // jobID
		"test-service", // serviceID
		"test-owner",   // ownerID
		"stdout",       // output
		"clone",        // phase
		logger,
	)

	// 4. Write test logs (simulating real build)
	testLogs := []string{
		"=== Phase: Clone ===",
		"Cloning repository...",
		"Clone completed.",
		"=== Phase: Build ===",
		"Building image...",
		"Step 1/5: FROM node:18-alpine",
		"Step 2/5: WORKDIR /app",
		"Build completed successfully!",
	}

	fmt.Println("Publishing logs...")
	for i, log := range testLogs {
		if i < 3 {
			stdoutWriter.SetPhase("clone")
		} else {
			stdoutWriter.SetPhase("build")
		}
		_, err := stdoutWriter.Write([]byte(log + "\n"))
		if err != nil {
			fmt.Printf("❌ Write failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  → %s\n", log)
		time.Sleep(100 * time.Millisecond) // simulate real timing
	}

	// 5. Close writer (flushes buffer)
	stdoutWriter.Close()
	fmt.Println("✓ All logs published!")

	// 6. Publish build end event
	fmt.Println("\n[3] Publishing BuildLogEnd event...")
	err = client.PublishBuildLogEnd(nats.BuildLogEndPayload{
		DeploymentID: deploymentID,
		JobID:        12345,
		Status:       "success",
	})
	if err != nil {
		fmt.Printf("❌ BuildLogEnd FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ BuildLogEnd published!")

	fmt.Printf("\n=== TEST PASSED ===\n")
	fmt.Printf("Deployment ID: %s\n", deploymentID)
	fmt.Println("Check your backend to verify logs were received!")
}

// ensureStream creates the JetStream stream if it doesn't exist
func ensureStream(servers, nkeySeed, prefix string) error {
	// Parse NKey
	kp, err := nkeys.FromSeed([]byte(nkeySeed))
	if err != nil {
		return fmt.Errorf("failed to parse NKey: %w", err)
	}
	pubKey, err := kp.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Connect with NKey auth
	nc, err := natspkg.Connect(servers, natspkg.Nkey(pubKey, func(nonce []byte) ([]byte, error) {
		return kp.Sign(nonce)
	}))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer nc.Close()

	// Get JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("failed to get JetStream: %w", err)
	}

	ctx := context.Background()
	streamName := prefix + "_cloudstation"

	// Try to get existing stream
	stream, err := js.Stream(ctx, streamName)
	if err == nil {
		info, _ := stream.Info(ctx)
		fmt.Printf("  Stream '%s' already exists\n", streamName)
		fmt.Printf("  Configured subjects: %v\n", info.Config.Subjects)

		// Check if build.log subjects are included
		hasSubject := false
		for _, s := range info.Config.Subjects {
			if s == prefix+".>" || s == prefix+".build.>" || s == prefix+".build.log.>" {
				hasSubject = true
				break
			}
		}
		if !hasSubject {
			fmt.Printf("  ⚠️  Stream doesn't include build.log subjects, updating...\n")
			// Update stream to include build log subjects
			newCfg := info.Config
			// Add both specific patterns to avoid overlap issues
			newCfg.Subjects = append(newCfg.Subjects, prefix+".build.log.*", prefix+".build.log.end.*")
			_, err = js.UpdateStream(ctx, newCfg)
			if err != nil {
				fmt.Printf("  Failed to update stream: %v\n", err)
				fmt.Printf("  You may need to manually add subjects to the stream\n")
			} else {
				fmt.Printf("  ✓ Stream updated with build.log subjects\n")
			}
		}
		return nil
	}

	// Create stream with wildcard subject for the prefix
	streamCfg := jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{prefix + ".>"}, // e.g., "cs.>" matches all cs.* subjects
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    24 * time.Hour * 7, // 7 days
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	}

	_, err = js.CreateStream(ctx, streamCfg)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	fmt.Printf("  Created stream '%s' with subjects [%s.>]\n", streamName, prefix)
	return nil
}
