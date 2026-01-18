package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/dispatch"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/nats"
	"github.com/urfave/cli/v2"
)

func dispatchCommand() *cli.Command {
	return &cli.Command{
		Name:   "dispatch",
		Usage:  "Execute as Nomad dispatched job (internal use)",
		Hidden: true,
		Action: func(c *cli.Context) error {
			startTime := time.Now()
			logger := hclog.New(&hclog.LoggerOptions{
				Output: os.Stderr,
				Level:  hclog.Info,
			})

			// Parse task type from environment
			taskType, err := dispatch.ParseTaskType()
			if err != nil {
				dispatch.LogErrorToStderr(logger, "parse_task_type", err)
				os.Exit(dispatch.ExitCodeParseError)
			}

			// Log to stdout for Nomad visibility
			fmt.Println("=== Dispatch Execution Started ===")
			fmt.Printf("Task: %s\n", taskType)
			fmt.Printf("Start time: %s\n", startTime.Format(time.RFC3339))
			fmt.Printf("Orchestrator version: %s\n", Version)

			logger.Info("=== Dispatch Execution Started ===", "task", taskType, "time", startTime)
			defer func() {
				logger.Info("=== Dispatch Execution Completed ===", "duration", time.Since(startTime))
				fmt.Printf("=== Dispatch Execution Completed ===\nDuration: %s\n", time.Since(startTime))
				os.Stdout.Sync()
			}()

			// Parse parameters from environment
			params, err := dispatch.ParseParams(taskType)
			if err != nil {
				dispatch.LogErrorToStderr(logger, "parse_params", err)
				os.Exit(dispatch.ExitCodeParseError)
			}

			// Extract deployment context for log streaming
			var deploymentID, serviceID, ownerID string
			var deploymentJobID int

			switch p := params.(type) {
			case dispatch.DeployRepositoryParams:
				deploymentID = p.DeploymentID
				deploymentJobID = int(p.DeploymentJobID)
				serviceID = p.ServiceID
				ownerID = string(p.OwnerID)
			case dispatch.DeployImageParams:
				deploymentID = p.DeploymentID
				deploymentJobID = int(p.DeploymentJobID)
				serviceID = p.ServiceID
				ownerID = string(p.OwnerID)
			}

			// Log key parameters to stdout
			if deploymentID != "" {
				fmt.Printf("Deployment ID: %s, Job ID: %d\n", deploymentID, deploymentJobID)
			}

			// Initialize NATS client
			natsServers := os.Getenv("NATS_SERVERS")
			natsKey := os.Getenv("NATS_CLIENT_PRIVATE_KEY")
			natsPrefix := os.Getenv("NATS_STREAM_PREFIX")

			var natsClient *nats.Client
			if natsServers != "" && natsKey != "" {
				natsClient, err = nats.NewClientWithPrefix(natsServers, natsKey, natsPrefix, logger)
				if err != nil {
					logger.Warn("Failed to initialize NATS client, continuing without NATS", "error", err)
					// Continue without NATS - don't fail the entire operation
				} else {
					defer natsClient.Close()
					logger.Info("NATS client initialized successfully", "prefix", natsPrefix)
				}
			} else {
				logger.Warn("NATS configuration not provided, skipping NATS events")
			}

			// Create NATS log writers if NATS client is available
			var stdoutWriter, stderrWriter io.Writer
			if natsClient != nil && deploymentID != "" {
				stdoutWriter = nats.NewLogWriter(
					natsClient,
					deploymentID,
					deploymentJobID,
					serviceID,
					ownerID,
					"stdout",
					"init",
					logger,
				)
				stderrWriter = nats.NewLogWriter(
					natsClient,
					deploymentID,
					deploymentJobID,
					serviceID,
					ownerID,
					"stderr",
					"init",
					logger,
				)

				// Ensure cleanup on exit
				defer func() {
					if lw, ok := stdoutWriter.(*nats.LogWriter); ok {
						lw.Close()
					}
					if lw, ok := stderrWriter.(*nats.LogWriter); ok {
						lw.Close()
					}
				}()

				logger.Info("NATS log streaming enabled", "deploymentID", deploymentID)
			} else {
				logger.Warn("NATS unavailable for log streaming, logs will be local only")
			}

			// Set up 15-minute timeout
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			// Add NATS log writers to context for handlers to use
			// Uses exported ContextKey type from dispatch package to ensure type match
			if stdoutWriter != nil {
				ctx = context.WithValue(ctx, dispatch.CtxKeyStdoutWriter, stdoutWriter)
			}
			if stderrWriter != nil {
				ctx = context.WithValue(ctx, dispatch.CtxKeyStderrWriter, stderrWriter)
			}

			// Start timeout monitoring goroutine
			timeoutChan := make(chan struct{})
			go func() {
				<-ctx.Done()
				if ctx.Err() == context.DeadlineExceeded {
					logger.Error("Task execution timed out after 15 minutes")
					// Publish timeout failure event if possible
					if natsClient != nil {
						// Best effort - try to publish timeout event
						logger.Warn("Attempting to publish timeout failure event")
					}
					close(timeoutChan)
				}
			}()

			// Route to appropriate handler based on task type
			var handlerErr error
			switch taskType {
			case dispatch.TaskDeployRepository:
				repoParams, ok := params.(dispatch.DeployRepositoryParams)
				if !ok {
					err := fmt.Errorf("invalid parameters for deploy-repository task")
					dispatch.LogErrorToStderr(logger, "type_assertion", err)
					os.Exit(dispatch.ExitCodeValidation)
				}
				handlerErr = dispatch.HandleDeployRepository(ctx, repoParams, natsClient, logger)

			case dispatch.TaskRedeployRepository:
				repoParams, ok := params.(dispatch.DeployRepositoryParams)
				if !ok {
					err := fmt.Errorf("invalid parameters for redeploy-repository task")
					dispatch.LogErrorToStderr(logger, "type_assertion", err)
					os.Exit(dispatch.ExitCodeValidation)
				}
				handlerErr = dispatch.HandleDeployRepository(ctx, repoParams, natsClient, logger)

			case dispatch.TaskDeployImage:
				imageParams, ok := params.(dispatch.DeployImageParams)
				if !ok {
					err := fmt.Errorf("invalid parameters for deploy-image task")
					dispatch.LogErrorToStderr(logger, "type_assertion", err)
					os.Exit(dispatch.ExitCodeValidation)
				}
				handlerErr = dispatch.HandleDeployImage(ctx, imageParams, natsClient, logger)

			case dispatch.TaskDestroyJob:
				destroyParams, ok := params.(dispatch.DestroyJobParams)
				if !ok {
					err := fmt.Errorf("invalid parameters for destroy-job-pack task")
					dispatch.LogErrorToStderr(logger, "type_assertion", err)
					os.Exit(dispatch.ExitCodeValidation)
				}
				handlerErr = dispatch.HandleDestroyJob(ctx, destroyParams, natsClient, logger)

			default:
				return fmt.Errorf("unsupported task type: %s", taskType)
			}

			// Check if timeout occurred
			select {
			case <-timeoutChan:
				return fmt.Errorf("task execution timed out")
			default:
				// No timeout, continue
			}

			if handlerErr != nil {
				dispatch.LogErrorToStderr(logger, "handler_execution", handlerErr)
				// Small delay to ensure Nomad captures error logs
				time.Sleep(100 * time.Millisecond)
				os.Exit(dispatch.ExitCodeRuntime)
			}

			logger.Info("Dispatch task completed successfully")
			fmt.Println("Dispatch task completed successfully")

			// Small delay to ensure Nomad captures all logs
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}
}
