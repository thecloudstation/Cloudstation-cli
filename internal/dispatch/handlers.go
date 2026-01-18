package dispatch

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/hclgen"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/lifecycle"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/backend"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/detect"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/git"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/nats"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/portdetector"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/storage"
)

// writeLog writes a message to NATS writer
func writeLog(writer io.Writer, message string) {
	if writer != nil {
		writer.Write([]byte(message))
	}
}

// setPhaseForWriters updates the phase for both stdout and stderr NATS writers
func setPhaseForWriters(stdoutWriter, stderrWriter io.Writer, phase string) {
	if lw, ok := stdoutWriter.(*nats.LogWriter); ok {
		lw.SetPhase(phase)
	}
	if lw, ok := stderrWriter.(*nats.LogWriter); ok {
		lw.SetPhase(phase)
	}
}

// createBackendClient creates a backend API client if configuration is provided
func createBackendClient(backendURL, accessToken string, logger hclog.Logger) *backend.Client {
	if backendURL == "" || accessToken == "" {
		logger.Info("Backend API integration disabled",
			"backendURL_empty", backendURL == "",
			"accessToken_empty", accessToken == "",
			"backendURL_value", backendURL)
		return nil
	}

	client, err := backend.NewClient(backendURL, accessToken, logger)
	if err != nil {
		logger.Warn("Failed to create backend client", "error", err)
		return nil
	}

	logger.Info("Backend API integration enabled", "backendUrl", backendURL)
	return client
}

// updateDeploymentStep tracks deployment progress via backend API
func updateDeploymentStep(client *backend.Client, deploymentID, deploymentType string, step backend.DeploymentStep, status backend.DeploymentStatus, errorMsg string, logger hclog.Logger) {
	if client == nil {
		return
	}

	req := backend.UpdateDeploymentStepRequest{
		DeploymentID:   deploymentID,
		DeploymentType: deploymentType,
		Step:           step,
		Status:         status,
		Error:          errorMsg,
	}

	if err := client.UpdateDeploymentStep(req); err != nil {
		logger.Warn("Failed to update deployment step", "error", err, "step", step, "status", status)
	}
}

// sanitizeRootDirectory ensures the path is safe and relative for use as a build context.
// It removes leading/trailing slashes and normalizes the path to prevent absolute path
// interpretation that can cause build failures.
func sanitizeRootDirectory(path string) string {
	if path == "" || path == "." {
		return path
	}
	// Remove leading slashes (prevents absolute path interpretation)
	path = strings.TrimLeft(path, "/")
	// Remove trailing slashes
	path = strings.TrimRight(path, "/")
	// Clean the path (normalizes . and ..)
	path = filepath.Clean(path)
	// Handle edge case where Clean returns "."
	if path == "." {
		return ""
	}
	return path
}

// hasUserProvidedDomainForPort checks if the user provided a domain for the given port
func hasUserProvidedDomainForPort(networks []NetworkPortSettings, port int) (bool, string) {
	for _, network := range networks {
		if int(network.PortNumber) == port && network.Domain != "" {
			return true, network.Domain
		}
	}
	return false, ""
}

// HandleDeployRepository handles deployment from a Git repository
func HandleDeployRepository(ctx context.Context, params DeployRepositoryParams, natsClient *nats.Client, logger hclog.Logger) error {
	logger.Info("Handling deploy-repository", "jobId", params.JobID, "repository", params.Repository)

	// Extract NATS log writers from context using exported keys
	var stdoutWriter, stderrWriter io.Writer
	if val := ctx.Value(CtxKeyStdoutWriter); val != nil {
		stdoutWriter = val.(io.Writer)
	}
	if val := ctx.Value(CtxKeyStderrWriter); val != nil {
		stderrWriter = val.(io.Writer)
	}

	// Log to NATS if available
	writeLog(stdoutWriter, "=== Starting deployment from repository ===\n")
	writeLog(stdoutWriter, fmt.Sprintf("Repository: %s\nBranch: %s\nJob ID: %s\n", params.Repository, params.Branch, params.JobID))

	// Create backend client if configured
	backendClient := createBackendClient(params.BackendURL, params.AccessToken, logger)

	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "cs-deploy-*")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	// Cleanup on success only - keep files on error for debugging
	defer func() {
		if err == nil {
			os.RemoveAll(workDir)
		} else {
			logger.Info("Preserving work directory for debugging", "path", workDir)
		}
	}()

	// Publish deployment started event
	if natsClient != nil {
		if err := natsClient.PublishDeploymentStarted(int(params.DeploymentJobID)); err != nil {
			logger.Warn("Failed to publish deployment started event", "error", err)
		}
	}

	// Check if this is a local upload deployment
	if params.SourceType == "local_upload" && params.SourceUrl != "" {
		// Download and extract source from MinIO
		fmt.Println("=== Phase: Download & Extract ===")
		fmt.Printf("Downloading source from storage...\n")
		logger.Info("Downloading uploaded source", "sourceUrl", params.SourceUrl, "uploadId", params.UploadId)

		// Update phase for NATS writer
		setPhaseForWriters(stdoutWriter, stderrWriter, "clone")

		writeLog(stdoutWriter, "=== Phase: Download & Extract ===\nDownloading uploaded source...\n")
		updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepClone, backend.StatusInProgress, "", logger)

		if err := storage.DownloadAndExtract(params.SourceUrl, workDir, logger); err != nil {
			fmt.Fprintf(os.Stdout, "ERROR [download]: %v\n", err)
			os.Stdout.Sync()
			updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepClone, backend.StatusFailed, err.Error(), logger)
			writeLog(stderrWriter, fmt.Sprintf("ERROR: Failed to download source: %v\n", err))
			publishFailure(natsClient, &params, logger, err)
			return fmt.Errorf("failed to download and extract source: %w", err)
		}

		fmt.Printf("Source extracted successfully to: %s\n", workDir)
		writeLog(stdoutWriter, "Source extracted successfully\n")
		updateDeploymentStep(backendClient, params.DeploymentID, "local_upload", backend.StepClone, backend.StatusCompleted, "", logger)
	} else {
		// Determine Git provider
		provider := git.GitHub
		if params.Provider != "" {
			provider = git.Provider(params.Provider)
		}

		// Update phase for NATS writer
		setPhaseForWriters(stdoutWriter, stderrWriter, "clone")

		// Clone the repository
		fmt.Println("=== Phase: Clone ===")
		fmt.Printf("Cloning repository: %s (branch: %s)\n", params.Repository, params.Branch)
		logger.Info("Cloning repository", "repository", params.Repository, "branch", params.Branch)
		writeLog(stdoutWriter, fmt.Sprintf("=== Phase: Clone ===\nCloning repository %s (branch: %s)...\n", params.Repository, params.Branch))
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepClone, backend.StatusInProgress, "", logger)

		// Debug logging for gitPass (redacted for security)
		if params.GitPass != "" {
			hasPercent := false
			hasColon := false
			for _, c := range params.GitPass {
				if c == '%' {
					hasPercent = true
				}
				if c == ':' {
					hasColon = true
				}
			}
			prefix := params.GitPass
			if len(prefix) > 10 {
				prefix = prefix[:10]
			}
			logger.Info("[DEBUG-v20250107] gitPass received",
				"length", len(params.GitPass),
				"hasPercentEncoding", hasPercent,
				"hasColon", hasColon,
				"prefix", prefix+"...",
				"repository", params.Repository)
		}

		cloneOpts := git.CloneOptions{
			Repository:     params.Repository,
			Branch:         params.Branch,
			Token:          params.GitPass,
			Provider:       provider,
			DestinationDir: workDir,
			Logger:         logger,
		}

		if err := git.Clone(cloneOpts); err != nil {
			fmt.Fprintf(os.Stdout, "ERROR [clone]: %v\n", err)
			os.Stdout.Sync()
			updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepClone, backend.StatusFailed, err.Error(), logger)
			writeLog(stderrWriter, fmt.Sprintf("ERROR: Failed to clone repository: %v\n", err))
			publishFailure(natsClient, &params, logger, err)
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		fmt.Printf("Clone completed: %s (branch: %s)\n", params.Repository, params.Branch)
		writeLog(stdoutWriter, "Clone completed successfully\n")
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepClone, backend.StatusCompleted, "", logger)
	}

	// Get builder chain (user choice first, then auto-detected fallbacks)
	builderChain := detect.GetBuilderChain(workDir, params.Build.Builder)

	logger.Info("Builder chain determined", "chain", builderChain)
	writeLog(stdoutWriter, fmt.Sprintf("Builder chain: %v\n", builderChain))

	// Set working directory for lifecycle execution
	if err := os.Chdir(workDir); err != nil {
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("failed to change to work directory: %w", err)
	}

	var lastErr error
	var buildArtifact *artifact.Artifact

	// Try each builder in the chain until one succeeds
	for attemptNum, builder := range builderChain {
		// Log fallback attempt (skip for first attempt)
		if attemptNum > 0 {
			writeLog(stdoutWriter, fmt.Sprintf(
				"Warning: Builder '%s' failed, trying '%s' (attempt %d/%d)...\n",
				builderChain[attemptNum-1], builder, attemptNum+1, len(builderChain),
			))
			logger.Warn("Falling back to next builder",
				"previous_builder", builderChain[attemptNum-1],
				"next_builder", builder,
				"attempt", attemptNum+1,
				"total_builders", len(builderChain),
			)
		}

		// Set build phase on log writers
		setPhaseForWriters(stdoutWriter, stderrWriter, "build")
		writeLog(stdoutWriter, fmt.Sprintf("Building with %s...\n", builder))
		logger.Info("Starting build attempt", "builder", builder, "attempt", attemptNum+1)

		// Update deployment step to in-progress for this attempt
		if attemptNum == 0 {
			fmt.Println("=== Phase: Build ===")
			fmt.Println("Starting build process...")
			writeLog(stdoutWriter, "=== Phase: Build ===\nStarting build process...\n")
			updateDeploymentStep(backendClient, params.DeploymentID, "repository",
				backend.StepBuild, backend.StatusInProgress, "", logger)
		}

		// Pass NATS writers via context to builders
		buildCtx := ctx
		if stdoutWriter != nil {
			buildCtx = context.WithValue(buildCtx, "stdoutWriter", stdoutWriter)
		}
		if stderrWriter != nil {
			buildCtx = context.WithValue(buildCtx, "stderrWriter", stderrWriter)
		}

		// Attempt build with current builder
		buildArtifact, lastErr = attemptBuild(buildCtx, workDir, &params, builder, logger)

		if lastErr == nil {
			// SUCCESS! Log and break out of loop
			writeLog(stdoutWriter, fmt.Sprintf("Build succeeded with %s\n", builder))
			logger.Info("Build succeeded",
				"builder", builder,
				"attempt", attemptNum+1,
				"artifact", buildArtifact.ID,
				"detected_ports", buildArtifact.ExposedPorts,
			)
			updateDeploymentStep(backendClient, params.DeploymentID, "repository",
				backend.StepBuild, backend.StatusCompleted, "", logger)
			break
		}

		// Build failed, log error and try next builder
		logger.Warn("Builder failed",
			"builder", builder,
			"error", lastErr,
			"attempt", attemptNum+1,
			"remaining", len(builderChain)-attemptNum-1,
		)
		writeLog(stderrWriter, fmt.Sprintf("Builder '%s' failed: %v\n", builder, lastErr))
	}

	// Check if all builders failed
	if buildArtifact == nil {
		updateDeploymentStep(backendClient, params.DeploymentID, "repository",
			backend.StepBuild, backend.StatusFailed,
			fmt.Sprintf("All builders failed. Last error: %v", lastErr), logger)
		writeLog(stderrWriter, fmt.Sprintf("ERROR: All builders failed (%v tried). Last error: %v\n", len(builderChain), lastErr))
		fmt.Fprintf(os.Stdout, "ERROR [build]: All builders failed. Last error: %v\n", lastErr)
		os.Stdout.Sync()
		publishFailure(natsClient, &params, logger, lastErr)
		return fmt.Errorf("build failed with all builders: %w", lastErr)
	}

	// Build succeeded, log success and continue with deployment
	fmt.Printf("Build completed: %s (detected ports: %v)\n", buildArtifact.ID, buildArtifact.ExposedPorts)
	writeLog(stdoutWriter, fmt.Sprintf("Build completed successfully\nArtifact ID: %s\nDetected ports: %v\n", buildArtifact.ID, buildArtifact.ExposedPorts))

	// Allocate domains for detected ports if backend client is available
	allocatedDomains := make(map[int]string)
	if backendClient != nil && buildArtifact != nil && len(buildArtifact.ExposedPorts) > 0 {
		logger.Info("Checking which ports need domain allocation", "port_count", len(buildArtifact.ExposedPorts))

		for _, port := range buildArtifact.ExposedPorts {
			// Check if user already provided a domain for this port
			hasUserDomain, userDomain := hasUserProvidedDomainForPort(params.Networks, port)
			if hasUserDomain {
				logger.Info("User provided domain for port, skipping allocation",
					"port", port, "domain", userDomain)
				continue
			}

			// Only allocate domain if user didn't provide one
			subdomain, err := backendClient.AskDomain(params.ServiceID)
			if err != nil {
				logger.Warn("Failed to allocate domain for port", "port", port, "error", err)
				continue
			}

			// Combine subdomain with cluster domain if available
			fullDomain := subdomain
			if params.ClusterDomain != "" && !strings.HasSuffix(subdomain, params.ClusterDomain) {
				fullDomain = fmt.Sprintf("%s.%s", subdomain, params.ClusterDomain)
				logger.Info("Appended cluster domain to subdomain", "subdomain", subdomain, "fullDomain", fullDomain)
			} else if strings.HasSuffix(subdomain, params.ClusterDomain) {
				logger.Info("Subdomain already contains cluster domain, not appending", "fullDomain", subdomain)
			}

			allocatedDomains[port] = fullDomain
			logger.Info("Domain allocated for port", "port", port, "domain", fullDomain)
		}
	}

	// Update network configuration with allocated domains
	if len(allocatedDomains) > 0 {
		// If params.Networks is empty (zero-config deployment), create network entries for detected ports
		if len(params.Networks) == 0 {
			logger.Info("Creating network config for zero-config deployment", "allocated_domains", len(allocatedDomains))
			for port, domain := range allocatedDomains {
				params.Networks = append(params.Networks, NetworkPortSettings{
					PortNumber: FlexInt(port),
					PortType:   "http",
					Public:     true,
					Domain:     domain,
					HealthCheck: HealthCheckSettings{
						Type:     "tcp",
						Interval: "30s",
						Timeout:  "30s",
					},
				})
				logger.Info("Added network config for detected port", "port", port, "domain", domain)
			}
		} else {
			// Update existing network entries with allocated domains (only if not explicitly set)
			for i := range params.Networks {
				// Only override domain if user didn't provide one explicitly
				if params.Networks[i].Domain == "" {
					if domain, exists := allocatedDomains[int(params.Networks[i].PortNumber)]; exists {
						params.Networks[i].Domain = domain
						logger.Info("Allocated domain for port (user did not specify)", "port", params.Networks[i].PortNumber, "domain", domain)
					}
				} else {
					logger.Info("Preserving user-specified domain", "port", params.Networks[i].PortNumber, "domain", params.Networks[i].Domain)
				}
			}
		}

		// Note: HCL params will be regenerated after this block
		// with the updated network configuration
	}

	// Sync service configuration to backend
	if backendClient != nil && len(params.Networks) > 0 {
		logger.Info("Syncing service configuration to backend",
			"networkCount", len(params.Networks))

		// Build network config from params.Networks (contains both user and allocated domains)
		networkConfigs := make([]backend.NetworkConfig, 0)
		for _, network := range params.Networks {
			networkConfigs = append(networkConfigs, backend.NetworkConfig{
				Port:           int(network.PortNumber),
				Type:           network.PortType,
				Public:         network.Public,
				Domain:         network.Domain,
				HasHealthCheck: network.HasHealthCheck,
				HealthCheck: backend.HealthCheckSettings{
					Type:     network.HealthCheck.Type,
					Path:     network.HealthCheck.Path,
					Interval: network.HealthCheck.Interval,
					Timeout:  network.HealthCheck.Timeout,
					Port:     int(network.HealthCheck.Port),
				},
			})
			logger.Info("Network config for backend sync",
				"port", network.PortNumber,
				"domain", network.Domain,
				"type", network.PortType)
		}

		// Prepare service update request
		updateReq := backend.UpdateServiceRequest{
			ServiceID: params.ServiceID,
			Network:   networkConfigs,
		}

		// Add docker user from params if available
		if params.DockerUser != "" {
			updateReq.DockerUser = params.DockerUser
		}

		// Add entrypoint from params if available
		if len(params.Entrypoint) > 0 {
			updateReq.Entrypoint = strings.Join(params.Entrypoint, " ")
		}

		// Add cmd from params if available
		if params.Command != "" {
			updateReq.CMD = params.Command
		}

		if err := backendClient.UpdateService(updateReq); err != nil {
			logger.Warn("Failed to sync service configuration", "error", err)
		} else {
			logger.Info("Service configuration synced successfully",
				"serviceId", params.ServiceID,
				"networksSynced", len(networkConfigs))
		}
	}

	// Now generate and write vars.hcl with artifact metadata
	hclParams := mapDeployRepositoryToHCLParams(params)
	varsContent := hclgen.GenerateVarsFile(hclParams, buildArtifact)
	varsPath, err := hclgen.WriteVarsFile(varsContent, workDir)
	if err != nil {
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("failed to write vars file: %w", err)
	}

	logger.Info("Vars file written", "path", varsPath)

	// Load final configuration for registry/deploy/release phases
	hclConfig, err := hclgen.GenerateConfig(hclParams)
	if err != nil {
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("failed to generate final HCL config: %w", err)
	}

	configPath, err := hclgen.WriteConfigFile(hclConfig, workDir)
	if err != nil {
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("failed to write final HCL config: %w", err)
	}

	cfg, err := config.LoadConfigFile(configPath)
	if err != nil {
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("failed to load final config: %w", err)
	}

	executor := lifecycle.NewExecutor(cfg, logger)

	app := cfg.GetApp(params.JobID)
	if app == nil {
		publishFailure(natsClient, &params, logger, fmt.Errorf("app not found"))
		return fmt.Errorf("app %q not found in final configuration", params.JobID)
	}

	// Continue with registry, deploy, and release phases
	if app.Registry != nil {
		// Update phase for NATS writer
		setPhaseForWriters(stdoutWriter, stderrWriter, "registry")

		fmt.Println("=== Phase: Registry ===")
		fmt.Println("Pushing image to registry...")
		writeLog(stdoutWriter, "=== Phase: Registry ===\nPushing image to registry...\n")
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRegistry, backend.StatusInProgress, "", logger)
		registryRef, err := executor.ExecuteRegistry(ctx, app, buildArtifact)
		if err != nil {
			fmt.Fprintf(os.Stdout, "ERROR [registry]: %v\n", err)
			os.Stdout.Sync()
			updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRegistry, backend.StatusFailed, err.Error(), logger)
			writeLog(stderrWriter, fmt.Sprintf("ERROR: Registry push failed: %v\n", err))
			publishFailure(natsClient, &params, logger, err)
			return fmt.Errorf("registry phase failed: %w", err)
		}
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRegistry, backend.StatusCompleted, "", logger)
		fmt.Printf("Registry push completed: %s\n", registryRef.FullImage)
		logger.Info("Registry push completed", "image", registryRef.FullImage)
		writeLog(stdoutWriter, fmt.Sprintf("Registry push completed\nImage: %s\n", registryRef.FullImage))
	}

	// Update phase for NATS writer
	setPhaseForWriters(stdoutWriter, stderrWriter, "deploy")

	// Deploy phase
	fmt.Println("=== Phase: Deploy ===")
	fmt.Println("Deploying to cluster...")
	writeLog(stdoutWriter, "=== Phase: Deploy ===\nDeploying to cluster...\n")
	updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepDeploy, backend.StatusInProgress, "", logger)
	dep, err := executor.ExecuteDeploy(ctx, app, buildArtifact)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR [deploy]: %v\n", err)
		os.Stdout.Sync()
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepDeploy, backend.StatusFailed, err.Error(), logger)
		writeLog(stderrWriter, fmt.Sprintf("ERROR: Deploy failed: %v\n", err))
		publishFailure(natsClient, &params, logger, err)
		return fmt.Errorf("deploy phase failed: %w", err)
	}
	updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepDeploy, backend.StatusCompleted, "", logger)

	fmt.Printf("Deploy completed: %s (status: %s)\n", dep.ID, dep.Status.State)
	logger.Info("Deploy completed", "deployment", dep.ID, "status", dep.Status.State)
	writeLog(stdoutWriter, fmt.Sprintf("Deploy completed\nDeployment ID: %s\nStatus: %s\n", dep.ID, dep.Status.State))

	// Release phase (optional)
	if app.Release != nil {
		// Update phase for NATS writer
		setPhaseForWriters(stdoutWriter, stderrWriter, "release")

		fmt.Println("=== Phase: Release ===")
		fmt.Println("Executing release steps...")
		writeLog(stdoutWriter, "=== Phase: Release ===\nExecuting release steps...\n")
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRelease, backend.StatusInProgress, "", logger)
		if err := executor.ExecuteRelease(ctx, app, dep); err != nil {
			fmt.Fprintf(os.Stdout, "ERROR [release]: %v\n", err)
			os.Stdout.Sync()
			updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRelease, backend.StatusFailed, err.Error(), logger)
			writeLog(stderrWriter, fmt.Sprintf("ERROR: Release failed: %v\n", err))
			publishFailure(natsClient, &params, logger, err)
			return fmt.Errorf("release phase failed: %w", err)
		}
		updateDeploymentStep(backendClient, params.DeploymentID, "repository", backend.StepRelease, backend.StatusCompleted, "", logger)
		fmt.Println("Release completed successfully")
		logger.Info("Release completed")
		writeLog(stdoutWriter, "Release completed successfully\n")
	}

	// Publish success event
	if natsClient != nil {
		payload := nats.DeploymentEventPayload{
			JobID:        int(params.DeploymentJobID),
			Type:         "git_repo",
			DeploymentID: params.DeploymentID,
			ServiceID:    params.ServiceID,
			TeamID:       params.TeamID,
			UserID:       strconv.Itoa(int(params.UserID)),
			OwnerID:      string(params.OwnerID),
		}
		if err := natsClient.PublishDeploymentSucceeded(payload); err != nil {
			logger.Warn("Failed to publish deployment succeeded event", "error", err)
		}

		// Send BuildLogEnd event to signal completion
		natsClient.PublishBuildLogEnd(nats.BuildLogEndPayload{
			DeploymentID: params.DeploymentID,
			JobID:        int(params.DeploymentJobID),
			Status:       "success",
		})
	}

	logger.Info("Deployment completed successfully")
	fmt.Println("=== Deployment completed successfully ===")
	os.Stdout.Sync()
	writeLog(stdoutWriter, "=== Deployment completed successfully ===\n")
	return nil
}

// HandleDeployImage handles deployment of a pre-built Docker image
func HandleDeployImage(ctx context.Context, params DeployImageParams, natsClient *nats.Client, logger hclog.Logger) error {
	logger.Info("Handling deploy-image", "jobId", params.JobID, "image", params.ImageName)

	// Extract NATS log writers from context using exported keys
	var stdoutWriter, stderrWriter io.Writer
	if val := ctx.Value(CtxKeyStdoutWriter); val != nil {
		stdoutWriter = val.(io.Writer)
	}
	if val := ctx.Value(CtxKeyStderrWriter); val != nil {
		stderrWriter = val.(io.Writer)
	}

	// Log to stdout for Nomad visibility
	fmt.Println("=== Starting deployment from pre-built image ===")
	fmt.Printf("Image: %s\nJob ID: %s\n", params.ImageName, params.JobID)

	// Log to NATS if available
	writeLog(stdoutWriter, "=== Starting deployment from pre-built image ===\n")
	writeLog(stdoutWriter, fmt.Sprintf("Image: %s\nJob ID: %s\n", params.ImageName, params.JobID))

	// Create backend client if configured
	backendClient := createBackendClient(params.BackendURL, params.AccessToken, logger)

	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "cs-deploy-image-*")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Publish deployment started event
	if natsClient != nil {
		if err := natsClient.PublishDeploymentStarted(int(params.DeploymentJobID)); err != nil {
			logger.Warn("Failed to publish deployment started event", "error", err)
		}
	}

	// === Phase: Port Detection ===
	var detectedPorts []int
	var allocatedDomains map[int]string
	fullImageName := params.ImageName
	if params.ImageTag != "" {
		fullImageName = fmt.Sprintf("%s:%s", params.ImageName, params.ImageTag)
	}

	// Only detect if no networks were provided
	if len(params.Networks) == 0 {
		setPhaseForWriters(stdoutWriter, stderrWriter, "detect")
		writeLog(stdoutWriter, "=== Phase: Port Detection ===\n")
		writeLog(stdoutWriter, fmt.Sprintf("Detecting ports from image %s...\n", fullImageName))

		var err error
		detectedPorts, err = portdetector.DetectPorts(fullImageName)
		if err != nil {
			logger.Warn("Port detection failed, using defaults", "image", fullImageName, "error", err)
			writeLog(stdoutWriter, fmt.Sprintf("Warning: %v\nUsing default port 3000\n", err))
			detectedPorts = []int{3000}
		} else {
			logger.Info("Ports detected", "image", fullImageName, "ports", detectedPorts)
			writeLog(stdoutWriter, fmt.Sprintf("Detected ports: %v\n", detectedPorts))
		}

		// Allocate domains for detected ports if backend client is available
		allocatedDomains = make(map[int]string)
		if backendClient != nil && len(detectedPorts) > 0 {
			logger.Info("Checking which ports need domain allocation", "port_count", len(detectedPorts))

			for _, port := range detectedPorts {
				// Check if user already provided a domain for this port
				hasUserDomain, userDomain := hasUserProvidedDomainForPort(params.Networks, port)
				if hasUserDomain {
					logger.Info("User provided domain for port, skipping allocation",
						"port", port, "domain", userDomain)
					continue
				}

				// Only allocate domain if user didn't provide one
				subdomain, err := backendClient.AskDomain(params.ServiceID)
				if err != nil {
					logger.Warn("Failed to allocate domain for port", "port", port, "error", err)
					continue
				}

				// Combine subdomain with cluster domain if available
				fullDomain := subdomain
				if params.ClusterDomain != "" && !strings.HasSuffix(subdomain, params.ClusterDomain) {
					fullDomain = fmt.Sprintf("%s.%s", subdomain, params.ClusterDomain)
					logger.Info("Appended cluster domain to subdomain", "subdomain", subdomain, "fullDomain", fullDomain)
				} else if strings.HasSuffix(subdomain, params.ClusterDomain) {
					logger.Info("Subdomain already contains cluster domain, not appending", "fullDomain", subdomain)
				}

				allocatedDomains[port] = fullDomain
				logger.Info("Domain allocated for port", "port", port, "domain", fullDomain)
				writeLog(stdoutWriter, fmt.Sprintf("Domain allocated for port %d: %s\n", port, fullDomain))
			}

			// Build network configurations with allocated domains
			for port, domain := range allocatedDomains {
				portType := inferPortType(port)
				// Use HTTP health check for HTTP ports, TCP for others
				healthCheckType := "tcp"
				healthCheckPath := ""
				if portType == "http" || portType == "https" {
					healthCheckType = "http"
					healthCheckPath = "/"
				}
				params.Networks = append(params.Networks, NetworkPortSettings{
					PortNumber:     FlexInt(port),
					PortType:       portType,
					Public:         true,
					Domain:         domain,
					HasHealthCheck: healthCheckType,
					HealthCheck: HealthCheckSettings{
						Type:     healthCheckType,
						Path:     healthCheckPath,
						Interval: "30s",
						Timeout:  "30s",
						Port:     FlexInt(port),
					},
				})
			}
		}

		// Sync service configuration to backend
		if backendClient != nil && len(params.Networks) > 0 {
			logger.Info("Syncing service configuration to backend",
				"networkCount", len(params.Networks))

			// Build network config from params.Networks
			networkConfigs := make([]backend.NetworkConfig, 0)
			for _, network := range params.Networks {
				networkConfigs = append(networkConfigs, backend.NetworkConfig{
					Port:           int(network.PortNumber),
					Type:           network.PortType,
					Public:         network.Public,
					Domain:         network.Domain,
					HasHealthCheck: network.HasHealthCheck,
					HealthCheck: backend.HealthCheckSettings{
						Type:     network.HealthCheck.Type,
						Path:     network.HealthCheck.Path,
						Interval: network.HealthCheck.Interval,
						Timeout:  network.HealthCheck.Timeout,
						Port:     int(network.HealthCheck.Port),
					},
				})
				logger.Info("Network config for backend sync",
					"port", network.PortNumber,
					"domain", network.Domain,
					"type", network.PortType)
			}

			// Prepare service update request
			updateReq := backend.UpdateServiceRequest{
				ServiceID: params.ServiceID,
				Network:   networkConfigs,
			}

			// Add docker user from params if available
			if params.DockerUser != "" {
				updateReq.DockerUser = params.DockerUser
			}

			// Add entrypoint from params if available
			if len(params.Entrypoint) > 0 {
				updateReq.Entrypoint = strings.Join(params.Entrypoint, " ")
			}

			// Add cmd from params if available
			if params.Command != "" {
				updateReq.CMD = params.Command
			}

			if err := backendClient.UpdateService(updateReq); err != nil {
				logger.Warn("Failed to sync service configuration", "error", err)
			} else {
				logger.Info("Service configuration synced successfully",
					"serviceId", params.ServiceID,
					"networksSynced", len(networkConfigs))
			}
		}
	}

	// Generate HCL configuration with noop builder (image is pre-built)
	logger.Info("Generating HCL configuration for pre-built image")
	hclParams := mapDeployImageToHCLParams(params) // This now includes updated Networks from port detection

	hclConfig, err := hclgen.GenerateConfig(hclParams)
	if err != nil {
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("failed to generate HCL config: %w", err)
	}

	// Write HCL config to work directory
	configPath, err := hclgen.WriteConfigFile(hclConfig, workDir)
	if err != nil {
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("failed to write HCL config: %w", err)
	}

	logger.Info("HCL configuration written", "path", configPath)

	// Generate and write vars.hcl with synthetic artifact for detected ports
	var synthArtifact *artifact.Artifact
	if len(detectedPorts) > 0 {
		synthArtifact = &artifact.Artifact{
			ID:           "image-deploy-synthetic",
			Image:        params.ImageName,
			Tag:          params.ImageTag,
			ExposedPorts: detectedPorts,
		}
	}
	varsContent := hclgen.GenerateVarsFile(hclParams, synthArtifact)
	varsPath, err := hclgen.WriteVarsFile(varsContent, workDir)
	if err != nil {
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("failed to write vars file: %w", err)
	}

	logger.Info("Vars file written", "path", varsPath)

	// Load configuration
	cfg, err := config.LoadConfigFile(configPath)
	if err != nil {
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set working directory for lifecycle execution
	if err := os.Chdir(workDir); err != nil {
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("failed to change to work directory: %w", err)
	}

	// Create lifecycle executor
	executor := lifecycle.NewExecutor(cfg, logger)

	// Update phase for NATS writer
	setPhaseForWriters(stdoutWriter, stderrWriter, "deploy")

	// Execute lifecycle (build will be skipped due to noop builder)
	fmt.Println("=== Phase: Deploy ===")
	fmt.Println("Deploying image to cluster...")
	logger.Info("Starting lifecycle execution")
	writeLog(stdoutWriter, "=== Phase: Deploy ===\nDeploying image to cluster...\n")
	updateDeploymentStep(backendClient, params.DeploymentID, "image", backend.StepDeploy, backend.StatusInProgress, "", logger)

	if err := executor.Execute(ctx, params.JobID); err != nil {
		fmt.Fprintf(os.Stdout, "ERROR [deploy]: %v\n", err)
		os.Stdout.Sync()
		updateDeploymentStep(backendClient, params.DeploymentID, "image", backend.StepDeploy, backend.StatusFailed, err.Error(), logger)
		writeLog(stderrWriter, fmt.Sprintf("ERROR: Deployment failed: %v\n", err))
		publishImageFailure(natsClient, params, logger, err)
		return fmt.Errorf("lifecycle execution failed: %w", err)
	}

	updateDeploymentStep(backendClient, params.DeploymentID, "image", backend.StepDeploy, backend.StatusCompleted, "", logger)
	fmt.Println("Deploy completed successfully")
	writeLog(stdoutWriter, "Deploy completed successfully\n")

	// Publish success event
	if natsClient != nil {
		payload := nats.DeploymentEventPayload{
			JobID:        int(params.DeploymentJobID),
			Type:         "image",
			DeploymentID: params.DeploymentID,
			ServiceID:    params.ServiceID,
			TeamID:       params.TeamID,
			UserID:       strconv.Itoa(int(params.UserID)),
			OwnerID:      string(params.OwnerID),
		}
		if err := natsClient.PublishDeploymentSucceeded(payload); err != nil {
			logger.Warn("Failed to publish deployment succeeded event", "error", err)
		}

		// Send BuildLogEnd event to signal completion
		natsClient.PublishBuildLogEnd(nats.BuildLogEndPayload{
			DeploymentID: params.DeploymentID,
			JobID:        int(params.DeploymentJobID),
			Status:       "success",
		})
	}

	logger.Info("Image deployment completed successfully")
	fmt.Println("=== Deployment completed successfully ===")
	os.Stdout.Sync()
	writeLog(stdoutWriter, "=== Deployment completed successfully ===\n")
	return nil
}

// HandleDestroyJob handles destruction of Nomad Pack jobs
func HandleDestroyJob(ctx context.Context, params DestroyJobParams, natsClient *nats.Client, logger hclog.Logger) error {
	logger.Info("Handling destroy-job-pack", "jobCount", len(params.Jobs), "reason", params.Reason)

	// Log to stdout for Nomad visibility
	fmt.Printf("=== Destroying %d job(s) ===\n", len(params.Jobs))
	fmt.Printf("Reason: %s\n", params.Reason)

	// Process each job sequentially (matching cs-runner behavior)
	for i, job := range params.Jobs {
		logger.Info("Destroying job", "index", i+1, "total", len(params.Jobs), "jobId", job.JobID)
		fmt.Printf("Destroying job %d/%d: %s\n", i+1, len(params.Jobs), job.JobID)

		// TODO: Implement actual nomad-pack destroy call
		// This would typically involve calling the nomad-pack CLI or API
		// For now, we'll just log and publish the event
		logger.Warn("Nomad Pack destroy not yet implemented", "jobId", job.JobID)

		// Publish job destroyed event
		if natsClient != nil {
			payload := nats.JobDestroyedPayload{
				ID:     job.ServiceID,
				Reason: nats.JobDestroyReason(params.Reason),
			}
			if err := natsClient.PublishJobDestroyed(payload); err != nil {
				logger.Warn("Failed to publish job destroyed event", "error", err, "jobId", job.JobID)
				fmt.Fprintf(os.Stdout, "ERROR [publish_event]: Failed to publish job destroyed event for %s: %v\n", job.JobID, err)
			} else {
				logger.Info("Published job destroyed event", "serviceId", job.ServiceID)
			}
		}
	}

	logger.Info("Job destruction completed")
	fmt.Println("=== Job destruction completed ===")
	os.Stdout.Sync()
	return nil
}

// publishFailure publishes a deployment failure event for repository deployments
func publishFailure(natsClient *nats.Client, params *DeployRepositoryParams, logger hclog.Logger, err error) {
	if natsClient != nil {
		payload := nats.DeploymentEventPayload{
			JobID:        int(params.DeploymentJobID),
			Type:         "git_repo",
			DeploymentID: params.DeploymentID,
			ServiceID:    params.ServiceID,
			TeamID:       params.TeamID,
			UserID:       strconv.Itoa(int(params.UserID)),
			OwnerID:      string(params.OwnerID),
		}
		if pubErr := natsClient.PublishDeploymentFailed(payload); pubErr != nil {
			logger.Warn("Failed to publish deployment failed event", "error", pubErr)
		}

		// Send BuildLogEnd event with failed status
		natsClient.PublishBuildLogEnd(nats.BuildLogEndPayload{
			DeploymentID: params.DeploymentID,
			JobID:        int(params.DeploymentJobID),
			Status:       "failed",
		})
	}
}

// publishImageFailure publishes a deployment failure event for image deployments
func publishImageFailure(natsClient *nats.Client, params DeployImageParams, logger hclog.Logger, err error) {
	if natsClient != nil {
		payload := nats.DeploymentEventPayload{
			JobID:        int(params.DeploymentJobID),
			Type:         "image",
			DeploymentID: params.DeploymentID,
			ServiceID:    params.ServiceID,
			TeamID:       params.TeamID,
			UserID:       strconv.Itoa(int(params.UserID)),
			OwnerID:      string(params.OwnerID),
		}
		if pubErr := natsClient.PublishDeploymentFailed(payload); pubErr != nil {
			logger.Warn("Failed to publish deployment failed event", "error", pubErr)
		}

		// Send BuildLogEnd event with failed status
		natsClient.PublishBuildLogEnd(nats.BuildLogEndPayload{
			DeploymentID: params.DeploymentID,
			JobID:        int(params.DeploymentJobID),
			Status:       "failed",
		})
	}
}

// mapDeployRepositoryToHCLParams maps DeployRepositoryParams to hclgen.DeploymentParams
func mapDeployRepositoryToHCLParams(params DeployRepositoryParams) hclgen.DeploymentParams {
	return hclgen.DeploymentParams{
		// Basic identifiers
		JobID:        params.JobID,
		ImageName:    params.ImageName,
		ImageTag:     params.ImageTag,
		BuilderType:  params.Build.Builder,
		DeployType:   params.Deploy,
		OwnerID:      string(params.OwnerID),
		ProjectID:    params.ProjectID,
		ServiceID:    params.ServiceID,
		TeamID:       params.TeamID,
		DeploymentID: params.DeploymentID,

		// Nomad configuration
		NomadAddress: params.NomadAddress,
		NomadToken:   params.NomadToken,
		NodePool:     params.NodePool,

		// Vault configuration
		VaultAddress:           params.VaultAddress,
		RoleID:                 params.RoleID,
		SecretID:               params.SecretID,
		SecretsPath:            params.SecretsPath,
		SharedSecretPath:       params.SharedSecretPath,
		UsesKvEngine:           params.UsesKvEngine,
		OwnerUsesKvEngine:      params.OwnerUsesKvEngine,
		VaultLinkedSecretPaths: mapVaultLinkedSecretPaths(params.VaultLinkedSecretPaths),
		VaultLinkedSecrets:     mapVaultLinkedSecrets(params.VaultLinkedSecrets),

		// Registry configuration
		Registry: hclgen.RegistryConfig{
			Pack:           params.Registry.Pack,
			RegistryName:   params.Registry.RegistryName,
			RegistryRef:    params.Registry.RegistryRef,
			RegistrySource: params.Registry.RegistrySource,
			RegistryTarget: params.Registry.RegistryTarget,
			RegistryToken:  params.Registry.RegistryToken,
			UseEmbedded:    params.Registry.UseEmbedded,
		},
		PrivateRegistry:         params.PrivateRegistry,
		PrivateRegistryProvider: params.PrivateRegistryProvider,

		// Registry push configuration (credentials optional - plugin falls back to env vars)
		RegistryUsername: params.Build.RegistryUsername,
		RegistryPassword: params.Build.RegistryPassword,
		RegistryURL:      params.Build.RegistryURL,
		DisablePush:      params.Build.DisablePush,

		// Resource configuration
		CPU:          int(params.CPU),
		RAM:          int(params.RAM),
		GPU:          int(params.GPU),
		GPUModel:     params.GPUModel,
		ReplicaCount: int(params.ReplicaCount),

		// Networking
		Networks: mapNetworkPorts(params.Networks),

		// Consul
		Consul: mapConsulConfig(params.Consul),

		// Storage
		CSIVolumes: mapCSIVolumes(params.CSIVolume),

		// Restart policy
		RestartMode:     params.RestartMode,
		RestartAttempts: params.RestartAttempts,

		// Job configuration
		JobConfig: mapJobTypeConfig(params.JobConfig),

		// Container configuration
		Command:    params.Command,
		Entrypoint: params.Entrypoint,
		DockerUser: params.DockerUser,

		// Build configuration
		DockerfilePath: params.Build.DockerfilePath,
		BuildArgs:      params.Build.BuildArgs,
		StaticBuildEnv: params.Build.StaticBuildEnv,
		BuildCommand:   params.Build.BuildCommand,
		StartCommand:   params.Build.StartCommand,
		RootDirectory:  sanitizeRootDirectory(params.Build.RootDirectory),

		// Deployment tracking
		DeploymentCount: params.DeploymentCount,

		// Advanced features
		Update:                  mapUpdateParameters(params.Update),
		TemplateStringVariables: mapTemplateStringVariables(params.TemplateStringVariables),
		ConfigFiles:             mapConfigFiles(params.ConfigFiles),
		Regions:                 params.Regions,
		TLS:                     mapTLSSettings(params.TLS),

		// Cloud provider
		CloudRegion:      params.CloudRegion,
		CloudProvider:    params.CloudProvider,
		ClusterDomain:    params.ClusterDomain,
		ClusterTCPDomain: params.ClusterTCPDomain,

		// Legacy
		NomadPackConfig: params.NomadPackConfig,
	}
}

// mapDeployImageToHCLParams maps DeployImageParams to hclgen.DeploymentParams
func mapDeployImageToHCLParams(params DeployImageParams) hclgen.DeploymentParams {
	return hclgen.DeploymentParams{
		// Basic identifiers
		JobID:        params.JobID,
		ImageName:    params.ImageName,
		ImageTag:     params.ImageTag,
		BuilderType:  "noop", // No build needed for pre-built images
		DeployType:   params.Deploy,
		OwnerID:      string(params.OwnerID),
		ProjectID:    params.ProjectID,
		ServiceID:    params.ServiceID,
		TeamID:       params.TeamID,
		DeploymentID: params.DeploymentID,

		// Nomad configuration
		NomadAddress: params.NomadAddress,
		NomadToken:   params.NomadToken,
		NodePool:     params.NodePool,

		// Vault configuration
		VaultAddress:           params.VaultAddress,
		RoleID:                 params.RoleID,
		SecretID:               params.SecretID,
		SecretsPath:            params.SecretsPath,
		SharedSecretPath:       params.SharedSecretPath,
		UsesKvEngine:           params.UsesKvEngine,
		OwnerUsesKvEngine:      params.OwnerUsesKvEngine,
		VaultLinkedSecretPaths: mapVaultLinkedSecretPaths(params.VaultLinkedSecretPaths),
		VaultLinkedSecrets:     mapVaultLinkedSecrets(params.VaultLinkedSecrets),

		// Registry configuration
		Registry: hclgen.RegistryConfig{
			Pack:           params.Registry.Pack,
			RegistryName:   params.Registry.RegistryName,
			RegistryRef:    params.Registry.RegistryRef,
			RegistrySource: params.Registry.RegistrySource,
			RegistryTarget: params.Registry.RegistryTarget,
			RegistryToken:  params.Registry.RegistryToken,
			UseEmbedded:    params.Registry.UseEmbedded,
		},
		PrivateRegistry:         params.PrivateRegistry,
		PrivateRegistryProvider: params.PrivateRegistryProvider,

		// Image deployments always push (no build to skip)
		DisablePush: false,

		// Resource configuration
		CPU:          int(params.CPU),
		RAM:          int(params.RAM),
		GPU:          int(params.GPU),
		GPUModel:     params.GPUModel,
		ReplicaCount: int(params.ReplicaCount),

		// Networking
		Networks: mapNetworkPorts(params.Networks),

		// Consul
		Consul: mapConsulConfig(params.Consul),

		// Storage
		CSIVolumes: mapCSIVolumes(params.CSIVolume),

		// Restart policy
		RestartMode:     params.RestartMode,
		RestartAttempts: params.RestartAttempts,

		// Job configuration
		JobConfig: mapJobTypeConfig(params.JobConfig),

		// Container configuration
		Command:    params.Command,
		Entrypoint: params.Entrypoint,
		DockerUser: params.DockerUser,

		// Build configuration (for image deployments, StartCommand overrides container CMD)
		StartCommand: params.Build.StartCommand,

		// Deployment tracking
		DeploymentCount: params.DeploymentCount,

		// Advanced features
		Update:                  mapUpdateParameters(params.Update),
		TemplateStringVariables: mapTemplateStringVariables(params.TemplateStringVariables),
		ConfigFiles:             mapConfigFiles(params.ConfigFiles),
		Regions:                 params.Regions,
		TLS:                     mapTLSSettings(params.TLS),

		// Cloud provider
		CloudRegion:      params.CloudRegion,
		CloudProvider:    params.CloudProvider,
		ClusterDomain:    params.ClusterDomain,
		ClusterTCPDomain: params.ClusterTCPDomain,

		// Legacy
		NomadPackConfig: params.NomadPackConfig,
	}
}

// Helper mapping functions

func mapNetworkPorts(networks []NetworkPortSettings) []hclgen.NetworkPort {
	if networks == nil {
		return nil
	}
	result := make([]hclgen.NetworkPort, len(networks))
	for i, n := range networks {
		result[i] = hclgen.NetworkPort{
			PortNumber:     int(n.PortNumber),
			PortType:       n.PortType,
			Public:         n.Public,
			Domain:         n.Domain,
			CustomDomain:   n.CustomDomain,
			HasHealthCheck: n.HasHealthCheck,
			HealthCheck: hclgen.HealthCheckConfig{
				Type:     n.HealthCheck.Type,
				Path:     n.HealthCheck.Path,
				Interval: n.HealthCheck.Interval,
				Timeout:  n.HealthCheck.Timeout,
				Port:     int(n.HealthCheck.Port),
			},
		}
	}
	return result
}

func mapConsulConfig(consul ConsulSettings) hclgen.ConsulConfig {
	linkedServices := make([]hclgen.ConsulLinkedService, len(consul.LinkedServices))
	for i, ls := range consul.LinkedServices {
		linkedServices[i] = hclgen.ConsulLinkedService{
			VariableName:      ls.VariableName,
			ConsulServiceName: ls.ConsulServiceName,
		}
	}

	return hclgen.ConsulConfig{
		ServiceName:    consul.ServiceName,
		Tags:           consul.Tags,
		ServicePort:    int(consul.ServicePort),
		LinkedServices: linkedServices,
	}
}

func mapCSIVolumes(volumes []CSIVolumeSettings) []hclgen.CSIVolume {
	if volumes == nil {
		return nil
	}
	result := make([]hclgen.CSIVolume, len(volumes))
	for i, v := range volumes {
		result[i] = hclgen.CSIVolume{
			ID:         v.ID,
			MountPaths: v.MountPaths,
		}
	}
	return result
}

func mapJobTypeConfig(jobConfig *JobTypeConfig) *hclgen.JobTypeConfig {
	if jobConfig == nil {
		return nil
	}
	return &hclgen.JobTypeConfig{
		Type:            jobConfig.Type,
		Cron:            jobConfig.Cron,
		ProhibitOverlap: jobConfig.ProhibitOverlap,
		Payload:         jobConfig.Payload,
		MetaRequired:    jobConfig.MetaRequired,
	}
}

func mapUpdateParameters(update *UpdateParameters) *hclgen.UpdateParameters {
	if update == nil {
		return nil
	}
	return &hclgen.UpdateParameters{
		MinHealthyTime:   update.MinHealthyTime,
		HealthyDeadline:  update.HealthyDeadline,
		ProgressDeadline: update.ProgressDeadline,
		AutoRevert:       update.AutoRevert,
		AutoPromote:      update.AutoPromote,
		MaxParallel:      int(update.MaxParallel),
		Canary:           int(update.Canary),
	}
}

func mapVaultLinkedSecretPaths(paths []VaultLinkedSecretPath) []hclgen.VaultLinkedSecretPath {
	if paths == nil {
		return nil
	}
	result := make([]hclgen.VaultLinkedSecretPath, len(paths))
	for i, p := range paths {
		result[i] = hclgen.VaultLinkedSecretPath{
			Prefix:     p.Prefix,
			SecretPath: p.SecretPath,
		}
	}
	return result
}

func mapVaultLinkedSecrets(secrets []VaultLinkedSecret) []hclgen.VaultLinkedSecret {
	if secrets == nil {
		return nil
	}
	result := make([]hclgen.VaultLinkedSecret, len(secrets))
	for i, s := range secrets {
		result[i] = hclgen.VaultLinkedSecret{
			Secret:   s.Secret,
			Template: s.Template,
		}
	}
	return result
}

func mapTemplateStringVariables(vars []TemplateStringVariable) []hclgen.TemplateStringVariable {
	if vars == nil {
		return nil
	}
	result := make([]hclgen.TemplateStringVariable, len(vars))
	for i, v := range vars {
		result[i] = hclgen.TemplateStringVariable{
			Name:              v.Name,
			Pattern:           v.Pattern,
			ServiceName:       v.ServiceName,
			ServiceSecretPath: v.ServiceSecretPath,
			LinkedVars:        v.LinkedVars,
		}
	}
	return result
}

func mapConfigFiles(files []ServiceConfigFile) []hclgen.ServiceConfigFile {
	if files == nil {
		return nil
	}
	result := make([]hclgen.ServiceConfigFile, len(files))
	for i, f := range files {
		result[i] = hclgen.ServiceConfigFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}
	return result
}

func mapTLSSettings(tls *TLSSettings) *hclgen.TLSConfig {
	if tls == nil {
		return nil
	}
	return &hclgen.TLSConfig{
		CertPath:   tls.CertPath,
		KeyPath:    tls.KeyPath,
		CommonName: tls.CommonName,
		PkaPath:    tls.PkaPath,
		TTL:        tls.TTL,
	}
}

// inferPortType returns the appropriate port type based on port number
// Common web ports default to http, HTTPS ports to https, others to tcp
func inferPortType(port int) string {
	switch port {
	case 80, 8080, 3000, 5000, 8000, 8888, 9000:
		return "http"
	case 443, 8443:
		return "https"
	case 5432, 3306, 27017, 6379, 9200:
		return "tcp" // Database ports
	default:
		return "tcp"
	}
}

// attemptBuild tries to build with a specific builder
// Returns artifact on success, error on failure
// This is a helper for the builder fallback loop
func attemptBuild(
	ctx context.Context,
	workDir string,
	params *DeployRepositoryParams,
	builder string,
	logger hclog.Logger,
) (*artifact.Artifact, error) {
	// Update params with current builder
	params.Build.Builder = builder

	// Generate HCL configuration for this builder
	hclParams := mapDeployRepositoryToHCLParams(*params)
	hclConfig, err := hclgen.GenerateConfig(hclParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HCL for %s: %w", builder, err)
	}

	// Write HCL config to work directory
	configPath, err := hclgen.WriteConfigFile(hclConfig, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to write config for %s: %w", builder, err)
	}

	// Load configuration
	cfg, err := config.LoadConfigFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config for %s: %w", builder, err)
	}

	// Create lifecycle executor
	executor := lifecycle.NewExecutor(cfg, logger)

	// Get app config
	app := cfg.GetApp(params.JobID)
	if app == nil {
		return nil, fmt.Errorf("app %q not found in config for builder %s", params.JobID, builder)
	}

	// Execute build phase
	return executor.ExecuteBuild(ctx, app)
}
