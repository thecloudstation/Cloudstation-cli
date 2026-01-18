package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/lifecycle"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/remote"
	"github.com/urfave/cli/v2"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize CloudStation project (interactive setup wizard)",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "non-interactive",
				Usage:   "Skip interactive prompts, use defaults",
				EnvVars: []string{"CS_NON_INTERACTIVE"},
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			apiURL := c.String("api-url")
			nonInteractive := c.Bool("non-interactive")

			// Check if already initialized
			if _, err := os.Stat(configPath); err == nil {
				fmt.Printf("Configuration file already exists: %s\n", configPath)
				fmt.Println("Use 'cs up' to deploy or edit the config file manually.")
				return nil
			}

			fmt.Println("Welcome to CloudStation!")
			fmt.Println(strings.Repeat("=", 40))

			// Step 1: Check/perform authentication
			creds, err := auth.LoadCredentials()
			if err != nil || !auth.IsValid(creds) {
				fmt.Println("\n[Step 1/3] Authentication")
				return fmt.Errorf("not logged in: please run 'cs login' first")
			} else {
				fmt.Println("\n[Step 1/3] Authentication")
				fmt.Printf("✓ Already logged in as %s\n", creds.Email)
			}

			// Step 2: Select or create service
			fmt.Println("\n[Step 2/3] Service Setup")

			authClient := auth.NewClient(apiURL)
			services, err := authClient.ListUserServices(creds.SessionToken)
			if err != nil {
				fmt.Printf("Warning: Could not fetch services: %v\n", err)
				services = []auth.UserService{}
			}

			var selectedServiceID string
			projectName := config.DetectProjectName()

			if len(services) == 0 {
				fmt.Println("No existing services found.")
				if nonInteractive {
					fmt.Println("Creating new service with detected name...")
				} else {
					fmt.Printf("Create a new service? Service name [%s]: ", projectName)
					var input string
					fmt.Scanln(&input)
					if input != "" {
						projectName = input
					}
				}

				// Create new service
				fmt.Printf("Creating service '%s'...\n", projectName)
				newService, err := authClient.CreateService(creds.SessionToken, &auth.CreateServiceRequest{
					Name:        projectName,
					ProjectName: projectName,
				})
				if err != nil {
					return fmt.Errorf("failed to create service: %w", err)
				}
				selectedServiceID = newService.ServiceID
				fmt.Printf("✓ Created service: %s\n", newService.ServiceName)

			} else {
				fmt.Printf("Found %d existing service(s):\n", len(services))
				for i, svc := range services {
					status := "●"
					if svc.Status == "running" {
						status = "✓"
					}
					fmt.Printf("  %d. %s %s (%s)\n", i+1, status, svc.Name, svc.ID)
				}
				fmt.Printf("  %d. [Create new service]\n", len(services)+1)

				if nonInteractive {
					// Use first service or create new
					selectedServiceID = services[0].ID
					fmt.Printf("Using first service: %s\n", services[0].Name)
				} else {
					fmt.Printf("\nSelect service [1-%d]: ", len(services)+1)
					var choice int
					fmt.Scanln(&choice)

					if choice > 0 && choice <= len(services) {
						selectedServiceID = services[choice-1].ID
						fmt.Printf("✓ Selected: %s\n", services[choice-1].Name)
					} else {
						// Create new service
						fmt.Printf("Service name [%s]: ", projectName)
						var input string
						fmt.Scanln(&input)
						if input != "" {
							projectName = input
						}

						fmt.Printf("Creating service '%s'...\n", projectName)
						newService, err := authClient.CreateService(creds.SessionToken, &auth.CreateServiceRequest{
							Name:        projectName,
							ProjectName: projectName,
						})
						if err != nil {
							return fmt.Errorf("failed to create service: %w", err)
						}
						selectedServiceID = newService.ServiceID
						fmt.Printf("✓ Created service: %s\n", newService.ServiceName)
					}
				}
			}

			// Save service link
			if err := auth.SaveServiceLink(selectedServiceID); err != nil {
				return fmt.Errorf("failed to link service: %w", err)
			}
			fmt.Println("✓ Service linked!")

			// Step 3: Generate config file
			fmt.Println("\n[Step 3/3] Configuration")

			configContent := fmt.Sprintf(`project = "%s"

app "%s" {
  build {
    use = "nixpacks"
  }

  deploy {
    use = "nomadpack"
  }
}
`, projectName, projectName)

			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
			fmt.Printf("✓ Created %s\n", configPath)

			// JSON output
			if c.Bool("json") {
				output := map[string]interface{}{
					"success":     true,
					"service_id":  selectedServiceID,
					"config_path": configPath,
					"project":     projectName,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Summary
			fmt.Println("\n" + strings.Repeat("=", 40))
			fmt.Println("Setup complete! Next steps:")
			fmt.Println("  cs up              # Deploy your application")
			fmt.Println("  cs up --remote     # Deploy using cloud infrastructure")
			fmt.Println("  cs build --remote  # Test build without deploying")

			return nil
		},
	}
}

func buildCommand() *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "Build an application (no config file required)",
		Description: `Build an application using auto-detection or explicit flags.

EXAMPLES:
  # Auto-detect everything (zero-config)
  cs build

  # Explicit builder selection
  cs build --builder=nixpacks
  cs build --builder=railpack
  cs build --builder=docker

  # Full control via flags (LLM-friendly)
  cs build --builder=nixpacks --name=myapp --tag=v1.0.0 --path=./src

  # Remote build on CloudStation
  cs build --remote`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "app",
				Usage: "Application name (auto-detected from directory/git)",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Image name (same as --app, alias for clarity)",
			},
			&cli.StringFlag{
				Name:  "builder",
				Usage: "Builder to use: nixpacks, railpack, docker, csdocker (auto-detected if not specified)",
			},
			&cli.StringFlag{
				Name:  "tag",
				Usage: "Image tag (default: latest)",
				Value: "latest",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Path to source code (default: current directory)",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "Output directory for build artifacts",
				Value: "./dist",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   DefaultAPIURL,
				EnvVars: []string{"CS_API_URL"},
			},
			&cli.BoolFlag{
				Name:    "remote",
				Aliases: []string{"r"},
				Usage:   "Build remotely using CloudStation infrastructure",
				EnvVars: []string{"CS_REMOTE"},
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:    "no-fallback",
				Usage:   "Disable automatic fallback to other builders on failure",
				EnvVars: []string{"CS_NO_FALLBACK"},
			},
		},
		Action: func(c *cli.Context) error {
			// Resolve app name: --name takes precedence, then --app, then auto-detect
			appName := c.String("name")
			if appName == "" {
				appName = c.String("app")
			}
			if appName == "" {
				appName = config.DetectProjectName()
			}

			apiURL := c.String("api-url")
			logger := hclog.Default()

			// Remote build
			if c.Bool("remote") {
				return executeRemoteBuild(c, appName, apiURL, logger)
			}

			// Determine builder: explicit flag > auto-detect
			builder := c.String("builder")
			noFallback := c.Bool("no-fallback")

			if builder == "" {
				detection := detect.DetectBuilder(c.String("path"))
				builder = detection.Builder
				if !c.Bool("json") {
					fmt.Printf("Auto-detected builder: %s (%s)\n", builder, detection.Reason)
					if noFallback {
						fmt.Printf("Fallback disabled, using only: %s\n", builder)
					} else if len(detection.Builders) > 1 {
						fmt.Printf("Fallback chain: %v\n", detection.Builders)
					}
				}
			} else {
				// User specified a builder - show the chain they'd get with fallback
				if !c.Bool("json") {
					chain := detect.GetBuilderChain(c.String("path"), builder)
					fmt.Printf("Using builder: %s\n", builder)
					if noFallback {
						fmt.Printf("Fallback disabled, using only: %s\n", builder)
					} else if len(chain) > 1 {
						fmt.Printf("Fallback chain: %v\n", chain)
					}
				}
			}

			// Build config from CLI flags (no HCL file needed)
			cfg := &config.Config{
				Project: appName,
				Apps: []*config.AppConfig{
					{
						Name: appName,
						Path: c.String("path"),
						Build: &config.PluginConfig{
							Use: builder,
							Config: map[string]interface{}{
								"name":    appName,
								"tag":     c.String("tag"),
								"context": c.String("path"),
							},
						},
						Deploy: &config.PluginConfig{
							Use:    "noop",
							Config: map[string]interface{}{},
						},
					},
				},
			}

			// Check if HCL config exists and user didn't specify explicit flags
			configPath := c.String("config")
			if c.String("builder") == "" && c.String("name") == "" {
				if loadedCfg, err := config.LoadConfigFile(configPath); err == nil {
					// Use config file if it exists and no explicit flags
					cfg = loadedCfg
					if appName == "" && len(cfg.Apps) > 0 {
						appName = cfg.Apps[0].Name
					}
				}
			} else {
				// Config file exists, auto-detect app name if not provided
				if appName == "" && len(cfg.Apps) > 0 {
					appName = cfg.Apps[0].Name
				}
			}

			if appName == "" {
				return fmt.Errorf("--app flag required when config has no apps defined")
			}

			// Get the builder chain for fallback
			var builderChain []string
			userBuilder := c.String("builder")
			if userBuilder != "" {
				builderChain = detect.GetBuilderChain(c.String("path"), userBuilder)
			} else {
				detection := detect.DetectBuilder(c.String("path"))
				builderChain = detection.Builders
			}

			// If fallback is disabled, only use the first builder
			if noFallback {
				builderChain = builderChain[:1]
			}

			// Try each builder in the chain until one succeeds
			var lastErr error
			var buildArtifact *artifact.Artifact
			ctx := context.Background()

			for attemptNum, tryBuilder := range builderChain {
				// Update config with current builder
				if len(cfg.Apps) > 0 && cfg.Apps[0].Build != nil {
					cfg.Apps[0].Build.Use = tryBuilder
				}

				// Log fallback attempt
				if attemptNum > 0 {
					if !c.Bool("json") {
						fmt.Printf("\n⚠️  Builder '%s' failed, trying '%s' (attempt %d/%d)...\n",
							builderChain[attemptNum-1], tryBuilder, attemptNum+1, len(builderChain))
					}
				} else {
					if !c.Bool("json") {
						fmt.Printf("Building with %s...\n", tryBuilder)
					}
				}

				executor := lifecycle.NewExecutor(cfg, logger)
				buildArtifact, lastErr = executor.BuildOnly(ctx, appName)

				if lastErr == nil {
					// Success!
					if !c.Bool("json") {
						fmt.Printf("✅ Build succeeded with %s\n", tryBuilder)
					}
					break
				}

				// Log failure
				if !c.Bool("json") {
					fmt.Printf("❌ Builder '%s' failed: %v\n", tryBuilder, lastErr)
				}
				logger.Warn("Builder failed, trying next",
					"builder", tryBuilder,
					"error", lastErr,
					"attempt", attemptNum+1,
					"remaining", len(builderChain)-attemptNum-1,
				)
			}

			// Check if all builders failed
			if buildArtifact == nil {
				return fmt.Errorf("all builders failed. Last error: %w", lastErr)
			}

			if c.Bool("json") {
				output := map[string]interface{}{
					"success":     true,
					"artifact_id": buildArtifact.ID,
					"image":       buildArtifact.Image,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("\nBuild completed successfully\n")
			fmt.Printf("  Artifact ID: %s\n", buildArtifact.ID)
			fmt.Printf("  Image: %s\n", buildArtifact.Image)
			return nil
		},
	}
}

func deployCommand() *cli.Command {
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy an application to CloudStation (no config file required)",
		Description: `Deploy an application using explicit flags or auto-detection.

EXAMPLES:
  # Deploy from linked service (requires 'cs link' first)
  cs deploy

  # Deploy with explicit project
  cs deploy --project=proj_abc123

  # Deploy a pre-built image
  cs deploy --image=myapp:v1.0.0 --project=proj_abc123

  # Full control via flags (LLM-friendly)
  cs deploy --name=myapp --project=proj_abc123 --port=3000`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "app",
				Usage: "Application name (auto-detected if not specified)",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Service name (same as --app, alias for clarity)",
			},
			&cli.StringFlag{
				Name:  "image",
				Usage: "Docker image to deploy (if pre-built)",
			},
			&cli.StringFlag{
				Name:  "project",
				Usage: "CloudStation project ID (e.g., proj_abc123)",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Internal port the application listens on",
				Value: 3000,
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   DefaultAPIURL,
				EnvVars: []string{"CS_API_URL"},
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be deployed without deploying",
			},
		},
		Action: func(c *cli.Context) error {
			// Resolve app name
			appName := c.String("name")
			if appName == "" {
				appName = c.String("app")
			}
			if appName == "" {
				appName = config.DetectProjectName()
			}

			apiURL := c.String("api-url")

			// Get credentials
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			// Resolve project ID
			projectID := c.String("project")
			if projectID == "" {
				// Try to get from linked service
				serviceID, err := auth.LoadServiceLink()
				if err != nil || serviceID == "" {
					return fmt.Errorf("no project specified: use --project flag or run 'cs link' first")
				}
				// Get project from service
				authClient := auth.NewClient(apiURL)
				details, err := authClient.GetServiceDetails(creds.SessionToken, serviceID)
				if err != nil {
					return fmt.Errorf("failed to get service details: %w", err)
				}
				projectID = details.ProjectID
			}

			image := c.String("image")
			port := c.Int("port")

			if c.Bool("dry-run") {
				output := map[string]interface{}{
					"dry_run": true,
					"name":    appName,
					"project": projectID,
					"image":   image,
					"port":    port,
				}
				if c.Bool("json") {
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
				} else {
					fmt.Println("DRY RUN - Would deploy:")
					fmt.Printf("  Name: %s\n", appName)
					fmt.Printf("  Project: %s\n", projectID)
					if image != "" {
						fmt.Printf("  Image: %s\n", image)
					}
					fmt.Printf("  Port: %d\n", port)
				}
				return nil
			}

			// Create remote client and deploy
			remoteClient := remote.NewClientWithToken(apiURL, creds.SessionToken)

			// If no image specified, trigger a remote build first
			if image == "" {
				fmt.Println("No --image specified, triggering remote build...")
				return executeRemoteBuild(c, appName, apiURL, hclog.Default())
			}

			// Deploy existing image
			fmt.Printf("Deploying %s to project %s...\n", image, projectID)

			deployResp, err := remoteClient.DeployImage(projectID, appName, image, port)
			if err != nil {
				return fmt.Errorf("deployment failed: %w", err)
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(deployResp, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("\n✓ Deployed %s\n", appName)
				fmt.Printf("  Service ID: %s\n", deployResp.ServiceID)
				if deployResp.URL != "" {
					fmt.Printf("  URL: %s\n", deployResp.URL)
				}
			}

			return nil
		},
	}
}

func upCommand() *cli.Command {
	return &cli.Command{
		Name:  "up",
		Usage: "Build and deploy an application (no config file required)",
		Description: `Build and deploy in one command. Auto-detects everything or use explicit flags.

EXAMPLES:
  # Simplest: auto-detect everything, deploy to linked service
  cs up

  # Remote build (faster, uses CloudStation infrastructure)
  cs up --remote

  # Explicit project (skip 'cs link')
  cs up --project=proj_abc123

  # Full control via flags (LLM-friendly)
  cs up --builder=nixpacks --name=myapp --project=proj_abc123 --port=8080

  # Local build only (no deploy)
  cs up --local-only`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "app",
				Usage: "Application name (auto-detected from directory/git)",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Service name (same as --app, alias for clarity)",
			},
			&cli.StringFlag{
				Name:  "builder",
				Usage: "Builder: nixpacks, railpack, docker (auto-detected if not specified)",
			},
			&cli.StringFlag{
				Name:  "project",
				Usage: "CloudStation project ID (e.g., proj_abc123)",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Path to source code",
				Value: ".",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Internal port the application listens on",
				Value: 3000,
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   DefaultAPIURL,
				EnvVars: []string{"CS_API_URL"},
			},
			&cli.BoolFlag{
				Name:    "remote",
				Aliases: []string{"r"},
				Usage:   "Build remotely using CloudStation infrastructure (faster, with caching)",
				EnvVars: []string{"CS_REMOTE"},
				Value:   true, // Default to remote builds
			},
			&cli.BoolFlag{
				Name:  "local",
				Usage: "Force local build instead of remote",
			},
			&cli.BoolFlag{
				Name:  "local-only",
				Usage: "Build locally without deploying",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would happen without building/deploying",
			},
		},
		Action: func(c *cli.Context) error {
			// Resolve app name: --name > --app > auto-detect
			appName := c.String("name")
			if appName == "" {
				appName = c.String("app")
			}
			if appName == "" {
				appName = config.DetectProjectName()
			}

			apiURL := c.String("api-url")
			logger := hclog.Default()

			// Resolve project ID
			projectID := c.String("project")
			if projectID == "" && !c.Bool("local-only") {
				// Try to get from linked service
				serviceID, _ := auth.LoadServiceLink()
				if serviceID != "" {
					creds, _ := auth.LoadCredentials()
					if creds != nil {
						authClient := auth.NewClient(apiURL)
						details, err := authClient.GetServiceDetails(creds.SessionToken, serviceID)
						if err == nil {
							projectID = details.ProjectID
						}
					}
				}
			}

			// Determine builder and get chain for fallback info
			builder := c.String("builder")
			var builderChain []string
			if builder == "" {
				detection := detect.DetectBuilder(c.String("path"))
				builder = detection.Builder
				builderChain = detection.Builders
				if !c.Bool("json") && !c.Bool("dry-run") {
					fmt.Printf("Auto-detected builder: %s (%s)\n", builder, detection.Reason)
					if len(builderChain) > 1 {
						fmt.Printf("Fallback chain: %v\n", builderChain)
					}
				}
			} else {
				// User specified a builder - get the chain for info
				builderChain = detect.GetBuilderChain(c.String("path"), builder)
				if !c.Bool("json") && !c.Bool("dry-run") {
					fmt.Printf("Using builder: %s\n", builder)
					if len(builderChain) > 1 {
						fmt.Printf("Fallback chain: %v\n", builderChain)
					}
				}
			}

			// Dry run
			if c.Bool("dry-run") {
				output := map[string]interface{}{
					"dry_run":       true,
					"name":          appName,
					"builder":       builder,
					"builder_chain": builderChain,
					"project":       projectID,
					"path":          c.String("path"),
					"port":          c.Int("port"),
					"remote":        c.Bool("remote") && !c.Bool("local"),
				}
				if c.Bool("json") {
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
				} else {
					fmt.Println("DRY RUN - Would execute:")
					fmt.Printf("  Name: %s\n", appName)
					fmt.Printf("  Builder: %s\n", builder)
					if len(builderChain) > 1 {
						fmt.Printf("  Fallback chain: %v\n", builderChain)
					}
					fmt.Printf("  Path: %s\n", c.String("path"))
					if projectID != "" {
						fmt.Printf("  Project: %s\n", projectID)
					}
					fmt.Printf("  Port: %d\n", c.Int("port"))
					if c.Bool("remote") && !c.Bool("local") {
						fmt.Println("  Mode: Remote build")
					} else {
						fmt.Println("  Mode: Local build")
					}
				}
				return nil
			}

			// Local-only build (no deploy)
			if c.Bool("local-only") || c.Bool("local") {
				return executeLocalBuild(c, appName, logger)
			}

			// Remote build and deploy (default)
			if c.Bool("remote") {
				return executeRemoteBuild(c, appName, apiURL, logger)
			}

			// Fall back to local build
			return executeLocalBuild(c, appName, logger)
		},
	}
}

// executeRemoteBuild triggers a remote build via backend API using local directory upload
func executeRemoteBuild(c *cli.Context, appName, apiURL string, logger hclog.Logger) error {
	fmt.Println("Triggering remote build...")

	// Load credentials
	creds, err := auth.LoadCredentials()
	if err != nil {
		return fmt.Errorf("not logged in: run 'cs login' first: %w", err)
	}

	if !auth.IsValid(creds) {
		return fmt.Errorf("credentials expired: run 'cs login' again")
	}

	// Load linked service
	serviceID, err := auth.LoadServiceLink()
	if err != nil {
		return fmt.Errorf("no service linked: run 'cs link' first: %w", err)
	}

	// Create remote client
	remoteClient := remote.NewClientWithToken(apiURL, creds.SessionToken)

	// Step 1: Create tarball of local directory
	fmt.Println("Creating source archive...")
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	tarballData, err := createTarball(cwd)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}
	fmt.Printf("  Archive size: %.2f MB\n", float64(len(tarballData))/(1024*1024))

	// Step 2: Initialize upload
	fmt.Println("Initializing upload...")
	initResp, err := remoteClient.InitUpload(serviceID)
	if err != nil {
		return fmt.Errorf("failed to initialize upload: %w", err)
	}
	fmt.Printf("  Upload ID: %s\n", initResp.UploadID)

	// Step 3: Upload tarball to presigned URL
	fmt.Println("Uploading source...")
	if err := remoteClient.UploadFile(initResp.UploadURL, tarballData); err != nil {
		return fmt.Errorf("failed to upload source: %w", err)
	}
	fmt.Println("  ✓ Upload complete")

	// Step 4: Complete upload with checksum
	checksum := sha256.Sum256(tarballData)
	checksumHex := hex.EncodeToString(checksum[:])
	_, err = remoteClient.CompleteUpload(initResp.UploadID, int64(len(tarballData)), checksumHex)
	if err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	// Step 5: Trigger deployment
	fmt.Println("Triggering deployment...")
	deployResp, err := remoteClient.TriggerLocalDeploy(initResp.UploadID)
	if err != nil {
		return fmt.Errorf("failed to trigger deployment: %w", err)
	}

	fmt.Printf("✓ Deployment started (ID: %s)\n\n", deployResp.DeploymentID)
	fmt.Println("Streaming build logs...")
	fmt.Println(strings.Repeat("-", 50))

	// Stream logs via SSE
	sseClient := remote.NewSSEClientWithToken(apiURL, creds.SessionToken)
	if err := sseClient.StreamBuildLogs(deployResp.DeploymentID, os.Stdout); err != nil {
		logger.Warn("log streaming ended", "error", err)
	}

	fmt.Println(strings.Repeat("-", 50))

	// Poll for deployment completion
	fmt.Println("\nWaiting for deployment to complete...")
	maxWait := 10 * time.Minute
	pollInterval := 5 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		status, err := remoteClient.GetDeploymentStatus(deployResp.DeploymentID)
		if err != nil {
			logger.Warn("failed to get status", "error", err)
			time.Sleep(pollInterval)
			continue
		}

		switch status {
		case "SUCCESS", "SUCCEEDED":
			if c.Bool("json") {
				output := map[string]interface{}{
					"success":       true,
					"deployment_id": deployResp.DeploymentID,
					"status":        status,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			fmt.Println("\n✓ Deployed successfully!")
			return nil
		case "FAILED":
			// Fetch detailed deployment info for actionable error message
			details, detailsErr := remoteClient.GetDeploymentDetails(deployResp.DeploymentID)
			if detailsErr != nil {
				return fmt.Errorf("deployment failed (could not fetch details: %v)", detailsErr)
			}

			// Get failure reason and suggestions
			reason := details.GetFailureReason()
			suggestions := details.GetActionableSuggestions()

			fmt.Println("\n✗ Deployment Failed")
			fmt.Println("─────────────────────────────────────")
			fmt.Printf("Reason: %s\n", reason)
			fmt.Printf("Service: %s\n", details.IntegrationName)
			fmt.Printf("Branch: %s\n", details.Branch)
			fmt.Printf("Deployment ID: %s\n", details.ID)

			if len(suggestions) > 0 {
				fmt.Println("\nSuggested Actions:")
				for i, suggestion := range suggestions {
					fmt.Printf("  %d. %s\n", i+1, suggestion)
				}
			}

			fmt.Println("\nFor more details, run:")
			fmt.Printf("  cs deployment status %s\n", deployResp.DeploymentID)

			return fmt.Errorf("deployment failed: %s", reason)
		case "CANCELLED":
			return fmt.Errorf("deployment was cancelled")
		default:
			fmt.Printf("Deployment status: %s...\n", status)
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("deployment timed out after %v", maxWait)
}

// createTarball creates a gzipped tarball of the specified directory
func createTarball(srcDir string) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Common patterns to exclude
	excludePatterns := []string{
		".git", "node_modules", ".env", ".env.local", "__pycache__",
		".DS_Store", "*.log", ".idea", ".vscode", "vendor", "dist",
		"build", "target", ".terraform", ".next", ".nuxt", "bin",
		"logs", "test-docker", "specs", "*.exe", "*.dll", "*.so",
	}

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Check exclusions
		for _, pattern := range excludePatterns {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// Also check if path starts with excluded directory
			if info.IsDir() && info.Name() == pattern {
				return filepath.SkipDir
			}
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = link
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content (skip for directories and symlinks)
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// executeLocalBuild is the existing local build logic (extracted)
func executeLocalBuild(c *cli.Context, appName string, logger hclog.Logger) error {
	configPath := c.String("config")

	// Try to load stored credentials
	creds, err := auth.LoadCredentials()
	if err == nil && auth.IsValid(creds) {
		// Use stored credentials by setting environment variables
		os.Setenv("VAULT_ADDR", creds.Vault.Address)
		os.Setenv("VAULT_TOKEN", creds.Vault.Token)
		os.Setenv("NOMAD_ADDR", creds.Nomad.Address)
		os.Setenv("NOMAD_TOKEN", creds.Nomad.Token)

		// Set NATS credentials (join URLs array into comma-separated string)
		if len(creds.NATS.URLs) > 0 {
			// Ensure nats:// prefix and join all URLs
			var formattedURLs []string
			for _, url := range creds.NATS.URLs {
				if !strings.HasPrefix(url, "nats://") && !strings.HasPrefix(url, "tls://") {
					url = "nats://" + url
				}
				formattedURLs = append(formattedURLs, url)
			}
			os.Setenv("NATS_SERVERS", strings.Join(formattedURLs, ","))
		}
		if creds.NATS.Seed != "" {
			os.Setenv("NATS_CLIENT_PRIVATE_KEY", creds.NATS.Seed)
		}

		// Set registry credentials for Docker push operations
		if creds.Registry.URL != "" {
			// Request app-specific token for this deployment
			appName := c.String("app")
			if appName != "" {
				apiURL := c.String("api-url")
				if apiURL == "" {
					apiURL = "https://api.cloudstation.io" // Default API URL
				}

				authClient := auth.NewClient(apiURL)
				registryCreds, _, err := authClient.RequestAppToken(creds.SessionToken, appName)
				if err != nil {
					logger.Warn("failed to get app token, using stored credentials", "error", err)
				} else {
					// Use the fresh, scoped token
					creds.Registry = *registryCreds
					logger.Info("using app-scoped registry token", "app", appName)
				}
			}

			os.Setenv("REGISTRY_URL", creds.Registry.URL)
			os.Setenv("REGISTRY_USERNAME", creds.Registry.Username)
			os.Setenv("REGISTRY_PASSWORD", creds.Registry.Password)
			os.Setenv("REGISTRY_NAMESPACE", creds.Registry.Namespace)
		}

		// Log user identifier - prefer email, fall back to user_uuid
		userIdentifier := creds.Email
		if userIdentifier == "" {
			userIdentifier = creds.UserUUID
		}
		logger.Info("using stored credentials", "user", userIdentifier)
	} else {
		// Fall back to environment variables (backward compatibility)
		logger.Debug("no stored credentials found, using environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfigFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create executor
	executor := lifecycle.NewExecutor(cfg, logger)

	// Execute full lifecycle
	ctx := context.Background()
	if err := executor.Execute(ctx, appName); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	if c.Bool("json") {
		output := map[string]interface{}{
			"success": true,
			"app":     appName,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Application deployed successfully\n")
	return nil
}

func runnerCommand() *cli.Command {
	return &cli.Command{
		Name:  "runner",
		Usage: "Runner agent commands",
		Subcommands: []*cli.Command{
			{
				Name:  "agent",
				Usage: "Start a runner agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "server-addr",
						Usage:    "Server address (host:port)",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "token",
						Usage:    "Authentication token",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return fmt.Errorf("runner agent not yet implemented")
				},
			},
		},
	}
}
