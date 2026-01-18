package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/envvar"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/portdetector"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/volume"
	"github.com/urfave/cli/v2"
)

// volumeSpec holds parsed volume configuration for on-the-fly creation
type volumeSpec struct {
	Name      string
	Capacity  float64
	MountPath string
}

// imageCommand returns the main image command with subcommands
func imageCommand() *cli.Command {
	return &cli.Command{
		Name:  "image",
		Usage: "Manage Docker image deployments",
		Subcommands: []*cli.Command{
			imageDeployCommand(),
		},
	}
}

// imageDeployCommand creates a CLI command for deploying Docker images directly
func imageDeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy a Docker image directly to CloudStation",
		ArgsUsage: "<image:tag>",
		Flags: []cli.Flag{
			// =====================================================
			// Basic Configuration
			// =====================================================
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Service name (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "project",
				Usage: "Project ID to deploy to",
			},
			&cli.IntFlag{
				Name:  "replicas",
				Usage: "Number of replicas",
				Value: 1,
			},

			// =====================================================
			// Resource Configuration
			// =====================================================
			&cli.IntFlag{
				Name:  "ram",
				Usage: "RAM allocation in MB",
				Value: 512,
			},
			&cli.Float64Flag{
				Name:  "cpu",
				Usage: "CPU allocation in cores",
				Value: 0.25,
			},
			&cli.IntFlag{
				Name:  "gpu",
				Usage: "GPU count (0 for no GPU)",
				Value: 0,
			},
			&cli.StringFlag{
				Name:  "gpu-model",
				Usage: "GPU model constraint (e.g., 'nvidia-a100', 'nvidia-t4')",
			},
			&cli.StringFlag{
				Name:  "node-pool",
				Usage: "Nomad node pool for scheduling",
				Value: "minions",
			},

			// =====================================================
			// Container Configuration
			// =====================================================
			&cli.StringFlag{
				Name:  "command",
				Usage: "Override container start command",
			},
			&cli.StringSliceFlag{
				Name:  "entrypoint",
				Usage: "Override container entrypoint (can be specified multiple times)",
			},
			&cli.StringFlag{
				Name:  "docker-user",
				Usage: "Container user to run as",
			},

			// =====================================================
			// Network Configuration
			// =====================================================
			&cli.StringSliceFlag{
				Name:  "port",
				Usage: "Port mapping (format: port:type, e.g., 8080:http). Can be specified multiple times",
			},
			&cli.BoolFlag{
				Name:  "public",
				Usage: "Expose service publicly via Traefik",
				Value: true,
			},
			&cli.StringFlag{
				Name:  "domain",
				Usage: "Custom domain for the service",
			},
			&cli.BoolFlag{
				Name:  "auto-port",
				Usage: "Automatically detect exposed ports from Docker image (default: true when no --port specified)",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "no-port",
				Usage: "Deploy without any port configuration (internal service only)",
				Value: false,
			},

			// =====================================================
			// Health Check Configuration
			// =====================================================
			&cli.StringFlag{
				Name:  "health-type",
				Usage: "Health check type: http, tcp, or none",
				Value: "http",
			},
			&cli.StringFlag{
				Name:  "health-path",
				Usage: "HTTP health check path",
				Value: "/",
			},
			&cli.StringFlag{
				Name:  "health-interval",
				Usage: "Health check interval (e.g., 30s, 1m)",
				Value: "30s",
			},
			&cli.StringFlag{
				Name:  "health-timeout",
				Usage: "Health check timeout (e.g., 30s, 1m)",
				Value: "30s",
			},

			// =====================================================
			// Job Type Configuration
			// =====================================================
			&cli.StringFlag{
				Name:  "job-type",
				Usage: "Nomad job type: service, batch, sysbatch, or system",
				Value: "service",
			},
			&cli.StringFlag{
				Name:  "cron",
				Usage: "Cron schedule for batch jobs (e.g., '0 0 * * *')",
			},
			&cli.BoolFlag{
				Name:  "prohibit-overlap",
				Usage: "Prohibit overlapping batch job executions",
				Value: true,
			},

			// =====================================================
			// Restart and Update Policy
			// =====================================================
			&cli.StringFlag{
				Name:  "restart-mode",
				Usage: "Restart mode: fail or delay",
				Value: "fail",
			},
			&cli.IntFlag{
				Name:  "restart-attempts",
				Usage: "Number of restart attempts before failure",
				Value: 3,
			},
			&cli.BoolFlag{
				Name:  "canary",
				Usage: "Enable canary deployments",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  "auto-revert",
				Usage: "Automatically revert failed deployments",
				Value: true,
			},

			// =====================================================
			// Storage Configuration
			// =====================================================
			&cli.StringSliceFlag{
				Name:  "volume",
				Usage: "Create and attach volume (format: name:capacity:/mount/path, e.g., mydata:10:/app/data for 10GB). Can be specified multiple times",
			},

			// =====================================================
			// Consul Service Discovery
			// =====================================================
			&cli.StringFlag{
				Name:  "consul-service",
				Usage: "Consul service name for service discovery",
			},
			&cli.StringSliceFlag{
				Name:  "consul-tag",
				Usage: "Consul service tag. Can be specified multiple times",
			},
			&cli.StringSliceFlag{
				Name:  "consul-link",
				Usage: "Link to Consul service (format: VAR_NAME=service-name). Can be specified multiple times",
			},

			// =====================================================
			// Environment Variables and Secrets
			// =====================================================
			&cli.StringSliceFlag{
				Name:  "env",
				Usage: "Environment variable (format: KEY=value). Can be specified multiple times",
			},
			&cli.StringSliceFlag{
				Name:  "secret",
				Usage: "Secret stored in Vault (format: KEY=value). Can be specified multiple times",
			},

			// =====================================================
			// Output and Debug Options
			// =====================================================
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output in JSON format for LLM parsing",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show configuration without executing deployment",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
			},
		},
		Action: func(c *cli.Context) error {
			// =====================================================
			// 1. Validate and parse basic arguments
			// =====================================================
			if c.NArg() < 1 {
				return fmt.Errorf("image:tag argument required (e.g., nginx:latest)")
			}

			imageArg := c.Args().First()
			imageName, imageTag := parseImageTag(imageArg)

			apiURL := c.String("api-url")
			outputJSON := c.Bool("json")
			dryRun := c.Bool("dry-run")

			// Load credentials
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first: %w", err)
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			// Get project ID - from flag or linked service
			projectID := c.String("project")
			if projectID == "" {
				serviceID, err := auth.LoadServiceLink()
				if err == nil && serviceID != "" {
					// Extract project ID from service if available
					authClient := auth.NewClient(apiURL)
					details, err := authClient.GetServiceDetails(creds.SessionToken, serviceID)
					if err == nil && details.ProjectID != "" {
						projectID = details.ProjectID
					}
				}
			}

			// =====================================================
			// 2. Parse container configuration
			// =====================================================
			command := c.String("command")
			entrypoint := c.StringSlice("entrypoint")
			dockerUser := c.String("docker-user")

			// =====================================================
			// 3. Parse resource configuration
			// =====================================================
			gpu := c.Int("gpu")
			gpuModel := c.String("gpu-model")
			nodePool := c.String("node-pool")

			// =====================================================
			// 4. Parse health check configuration
			// =====================================================
			healthType := c.String("health-type")
			healthPath := c.String("health-path")
			healthInterval := c.String("health-interval")
			healthTimeout := c.String("health-timeout")

			// Validate health type
			validHealthTypes := map[string]bool{"http": true, "tcp": true, "none": true}
			if !validHealthTypes[healthType] {
				return fmt.Errorf("invalid health-type '%s': must be http, tcp, or none", healthType)
			}

			// =====================================================
			// 5. Parse job type configuration
			// =====================================================
			jobType := c.String("job-type")
			cron := c.String("cron")
			prohibitOverlap := c.Bool("prohibit-overlap")

			// Validate job type
			validJobTypes := map[string]bool{"service": true, "batch": true, "sysbatch": true, "system": true}
			if !validJobTypes[jobType] {
				return fmt.Errorf("invalid job-type '%s': must be service, batch, sysbatch, or system", jobType)
			}

			// =====================================================
			// 6. Parse restart and update policy
			// =====================================================
			restartMode := c.String("restart-mode")
			restartAttempts := c.Int("restart-attempts")
			canary := c.Bool("canary")
			autoRevert := c.Bool("auto-revert")

			// Validate restart mode
			validRestartModes := map[string]bool{"fail": true, "delay": true}
			if !validRestartModes[restartMode] {
				return fmt.Errorf("invalid restart-mode '%s': must be fail or delay", restartMode)
			}

			// =====================================================
			// 7. Auto-detect ports if no --port flags provided
			// =====================================================
			var networks []map[string]interface{}
			public := c.Bool("public")
			domain := c.String("domain")
			userPorts := c.StringSlice("port")
			autoPort := c.Bool("auto-port")
			noPort := c.Bool("no-port")

			if len(userPorts) == 0 && autoPort && !noPort {
				// Build full image name with tag
				fullImage := imageName
				if imageTag != "" && imageTag != "latest" {
					fullImage = fmt.Sprintf("%s:%s", imageName, imageTag)
				}

				fmt.Printf("Detecting ports from image %s...\n", fullImage)
				detectedPorts, err := portdetector.DetectPorts(fullImage)
				if err != nil {
					fmt.Printf("Warning: Port detection failed: %v\n", err)
					fmt.Printf("Using default port 3000. Use --port to specify ports explicitly.\n")
					detectedPorts = []int{3000}
				} else {
					fmt.Printf("Detected ports: %v\n", detectedPorts)
				}

				// Build network config from detected ports
				for _, port := range detectedPorts {
					portType := inferPortType(port)
					network := map[string]interface{}{
						"portNumber": port,
						"portType":   portType,
						"public":     public,
					}

					// Add health check configuration based on port type
					// HTTP ports get HTTP health checks, others get TCP health checks
					if healthType != "none" {
						effectiveHealthType := healthType
						effectiveHealthPath := healthPath
						// For non-HTTP ports, use TCP health check regardless of --health-type flag
						if portType != "http" && portType != "https" {
							effectiveHealthType = "tcp"
							effectiveHealthPath = ""
						}
						network["has_health_check"] = effectiveHealthType
						network["health_check"] = map[string]interface{}{
							"type":     effectiveHealthType,
							"path":     effectiveHealthPath,
							"interval": healthInterval,
							"timeout":  healthTimeout,
							"port":     port,
						}
					}

					// Add domain to first network if specified
					if domain != "" && len(networks) == 0 {
						network["custom_domain"] = domain
					}

					networks = append(networks, network)
				}
			}

			// =====================================================
			// 8. Parse manual port mappings (if --port flags provided)
			// =====================================================
			for _, p := range userPorts {
				port, portType, err := parsePort(p)
				if err != nil {
					return fmt.Errorf("invalid port format '%s': %w", p, err)
				}

				network := map[string]interface{}{
					"portNumber": port,
					"portType":   portType,
					"public":     public,
				}

				// Add health check configuration if not "none"
				if healthType != "none" {
					network["has_health_check"] = healthType
					network["health_check"] = map[string]interface{}{
						"type":     healthType,
						"path":     healthPath,
						"interval": healthInterval,
						"timeout":  healthTimeout,
						"port":     port,
					}
				}

				// Add domain to first network if specified
				if domain != "" && len(networks) == 0 {
					network["custom_domain"] = domain
				}

				networks = append(networks, network)
			}

			// =====================================================
			// 8. Parse volumes (new format: name:capacity:/mount/path)
			// =====================================================
			var volumeSpecs []volumeSpec
			for _, v := range c.StringSlice("volume") {
				name, capacity, mountPath, err := parseVolume(v)
				if err != nil {
					return fmt.Errorf("invalid volume format '%s': %w", v, err)
				}
				volumeSpecs = append(volumeSpecs, volumeSpec{
					Name:      name,
					Capacity:  capacity,
					MountPath: mountPath,
				})
			}

			// =====================================================
			// 9. Parse Consul configuration
			// =====================================================
			consulService := c.String("consul-service")
			consulTags := c.StringSlice("consul-tag")
			var consulLinks []map[string]string
			for _, link := range c.StringSlice("consul-link") {
				varName, svcName, err := parseConsulLink(link)
				if err != nil {
					return fmt.Errorf("invalid consul-link format '%s': %w", link, err)
				}
				consulLinks = append(consulLinks, map[string]string{
					"variableName":      varName,
					"consulServiceName": svcName,
				})
			}

			// =====================================================
			// 10. Parse environment variables (container config)
			// =====================================================
			var variables []map[string]string
			for _, e := range c.StringSlice("env") {
				key, value, err := parseEnvVar(e)
				if err != nil {
					return fmt.Errorf("invalid env format '%s': %w", e, err)
				}
				variables = append(variables, map[string]string{
					"key":   key,
					"value": value,
				})
			}

			// =====================================================
			// 11. Parse secrets (Vault storage)
			// =====================================================
			var secrets []envvar.Variable
			for _, s := range c.StringSlice("secret") {
				key, value, err := parseEnvVar(s) // reuse existing parser
				if err != nil {
					return fmt.Errorf("invalid secret format '%s': %w", s, err)
				}
				secrets = append(secrets, envvar.Variable{
					Key:   key,
					Value: value,
				})
			}

			// =====================================================
			// 12. Build request body
			// =====================================================
			reqBody := map[string]interface{}{
				"name":         c.String("name"),
				"imageUrl":     imageName,
				"tag":          imageTag,
				"replicaCount": float64(c.Int("replicas")),
				"ram":          float64(c.Int("ram")),
				"cpu":          c.Float64("cpu"),
			}

			// Add container configuration
			if command != "" {
				reqBody["startCommand"] = command
			}
			if len(entrypoint) > 0 {
				reqBody["entryPoint"] = strings.Join(entrypoint, " ")
			}
			if dockerUser != "" {
				reqBody["dockerUser"] = dockerUser
			}

			// Add resource configuration
			if gpu > 0 {
				reqBody["gpu"] = gpu
				if gpuModel != "" {
					reqBody["gpuModel"] = gpuModel
				}
			}
			if nodePool != "" && nodePool != "minions" {
				reqBody["nodePool"] = nodePool
			}

			// Add job type configuration
			if jobType != "service" || cron != "" {
				reqBody["type"] = jobType
				if cron != "" {
					reqBody["cron"] = cron
					reqBody["prohibitOverlap"] = prohibitOverlap
				}
			}

			// Add restart and update policy
			reqBody["restartMode"] = restartMode
			reqBody["restartAttempts"] = restartAttempts
			if canary {
				reqBody["canary"] = true
			}
			reqBody["autoRevert"] = autoRevert

			// Volumes are now created separately via POST /volumes/:service_id after service creation

			// Add Consul configuration
			if consulService != "" {
				consul := map[string]interface{}{
					"serviceName": consulService,
				}
				if len(consulTags) > 0 {
					consul["tags"] = consulTags
				}
				if len(consulLinks) > 0 {
					consul["linkedServices"] = consulLinks
				}
				reqBody["consul"] = consul
			}

			// Add project ID
			if projectID != "" {
				reqBody["projectId"] = projectID
			}

			// Add networks
			if len(networks) > 0 {
				reqBody["networks"] = networks
			}

			// Add environment variables (container config)
			if len(variables) > 0 {
				reqBody["variables"] = variables
			}

			// =====================================================
			// 13. Handle dry-run mode
			// =====================================================
			if dryRun {
				fmt.Println("DRY RUN MODE - Configuration preview:")
				fmt.Println("=====================================")
				jsonData, _ := json.MarshalIndent(reqBody, "", "  ")
				fmt.Println(string(jsonData))
				if len(secrets) > 0 {
					fmt.Printf("\nSecrets to be created (%d):\n", len(secrets))
					for _, s := range secrets {
						fmt.Printf("  - %s=***\n", s.Key)
					}
				}
				if len(volumeSpecs) > 0 {
					fmt.Printf("\nVolumes to be created (%d):\n", len(volumeSpecs))
					for _, vol := range volumeSpecs {
						fmt.Printf("  - %s (%.0f GB at %s)\n", vol.Name, vol.Capacity, vol.MountPath)
					}
				}
				return nil
			}

			// =====================================================
			// 14. Execute deployment
			// =====================================================
			// Create HTTP client with proper authentication
			client := httpclient.NewBaseClient(apiURL, 30*time.Second)
			client.SetHeader("Authorization", "Bearer "+creds.SessionToken)

			if !outputJSON {
				fmt.Printf("Creating service '%s' from image %s:%s...\n", c.String("name"), imageName, imageTag)
			}

			// Step 1: Create service via POST /services/images
			var createResp struct {
				ID        string `json:"id"`
				ServiceID string `json:"service_id"`
				Name      string `json:"name"`
				Status    string `json:"status"`
			}

			if err := client.DoJSON("POST", "/services/images", reqBody, &createResp); err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}

			serviceID := createResp.ID
			if serviceID == "" {
				serviceID = createResp.ServiceID
			}

			if serviceID == "" {
				return fmt.Errorf("no service ID returned from API")
			}

			if !outputJSON {
				fmt.Printf("Service created: %s\n", serviceID)
			}

			// Step 2: Create secrets in Vault if provided
			if len(secrets) > 0 {
				if !outputJSON {
					fmt.Printf("Creating %d secrets in Vault...\n", len(secrets))
				}

				envvarClient := envvar.NewClientWithToken(apiURL, creds.SessionToken)
				if err := envvarClient.BulkCreate(serviceID, secrets); err != nil {
					return fmt.Errorf("failed to create secrets: %w", err)
				}

				if !outputJSON {
					fmt.Println("Secrets created successfully")
				}
			}

			// Step 3: Create and attach volumes if provided
			if len(volumeSpecs) > 0 {
				if !outputJSON {
					fmt.Printf("Creating %d volume(s)...\n", len(volumeSpecs))
				}

				if creds.SessionToken == "" {
					return fmt.Errorf("session token required for volume operations: run 'cs login' again")
				}
				volumeClient := volume.NewClientWithToken(apiURL, creds.SessionToken)

				for _, vol := range volumeSpecs {
					name := vol.Name
					req := volume.AttachRequest{
						Capacity:   vol.Capacity,
						Name:       &name,
						MountPaths: []string{vol.MountPath},
					}

					resp, err := volumeClient.Attach(serviceID, req)
					if err != nil {
						return fmt.Errorf("failed to create volume '%s': %w", vol.Name, err)
					}

					if !outputJSON {
						fmt.Printf("  Volume created: %s (%.0f GB at %s)\n", resp.ID, vol.Capacity, vol.MountPath)
					}
				}

				if !outputJSON {
					fmt.Println("Volumes created successfully")
				}
			}

			// Step 4: Deploy service via POST /services/:id/deploy
			if !outputJSON {
				fmt.Println("Triggering deployment...")
			}

			var deployResp struct {
				DeploymentID string `json:"deployment_id"`
				Status       string `json:"status"`
				Message      string `json:"message"`
			}

			deployPath := fmt.Sprintf("/services/%s/deploy", serviceID)
			if err := client.DoJSON("POST", deployPath, nil, &deployResp); err != nil {
				return fmt.Errorf("failed to trigger deployment: %w", err)
			}

			// =====================================================
			// 15. Output results
			// =====================================================
			if outputJSON {
				result := map[string]interface{}{
					"serviceId":    serviceID,
					"deploymentId": deployResp.DeploymentID,
					"status":       deployResp.Status,
					"image":        imageName,
					"tag":          imageTag,
					"name":         c.String("name"),
				}
				if len(secrets) > 0 {
					result["secretsCreated"] = len(secrets)
				}
				if len(volumeSpecs) > 0 {
					result["volumesCreated"] = len(volumeSpecs)
				}
				jsonOutput, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(jsonOutput))
			} else {
				fmt.Println()
				fmt.Println("Deployment initiated successfully!")
				fmt.Printf("  Service ID:    %s\n", serviceID)
				fmt.Printf("  Deployment ID: %s\n", deployResp.DeploymentID)
				fmt.Printf("  Status:        %s\n", deployResp.Status)
				fmt.Printf("  Image:         %s:%s\n", imageName, imageTag)
				if len(secrets) > 0 {
					fmt.Printf("  Secrets:       %d created\n", len(secrets))
				}
				if len(volumeSpecs) > 0 {
					fmt.Printf("  Volumes:       %d created\n", len(volumeSpecs))
				}
			}

			return nil
		},
	}
}

// parseImageTag splits an image string into name and tag components
// If no tag is specified, defaults to "latest"
// Examples:
//   - "nginx" -> ("nginx", "latest")
//   - "nginx:1.21" -> ("nginx", "1.21")
//   - "registry.io/org/image:v1" -> ("registry.io/org/image", "v1")
func parseImageTag(image string) (name string, tag string) {
	// Handle images with registry prefix (may contain multiple colons)
	// Find the last colon that's not part of a port number
	lastColon := strings.LastIndex(image, ":")

	if lastColon == -1 {
		// No colon found, use "latest" as default tag
		return image, "latest"
	}

	// Check if the part after the colon looks like a tag or a port
	afterColon := image[lastColon+1:]

	// If it contains a slash, it's part of the registry path, not a tag
	if strings.Contains(afterColon, "/") {
		return image, "latest"
	}

	// Otherwise, split at the last colon
	return image[:lastColon], afterColon
}

// parsePort parses a port specification in the format "port:type"
// If type is not specified, defaults to "tcp"
// Examples:
//   - "8080" -> (8080, "tcp")
//   - "8080:http" -> (8080, "http")
//   - "443:https" -> (443, "https")
func parsePort(p string) (port int, portType string, err error) {
	parts := strings.SplitN(p, ":", 2)

	port, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid port number: %s", parts[0])
	}

	if port < 1 || port > 65535 {
		return 0, "", fmt.Errorf("port must be between 1 and 65535")
	}

	if len(parts) == 2 && parts[1] != "" {
		portType = parts[1]
	} else {
		portType = "tcp"
	}

	return port, portType, nil
}

// parseEnvVar parses an environment variable in the format "KEY=value"
// Returns an error if the format is invalid
// Examples:
//   - "DATABASE_URL=postgres://..." -> ("DATABASE_URL", "postgres://...")
//   - "DEBUG=true" -> ("DEBUG", "true")
//   - "EMPTY=" -> ("EMPTY", "")
func parseEnvVar(e string) (key string, value string, err error) {
	idx := strings.Index(e, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("must be in KEY=value format")
	}

	key = e[:idx]
	value = e[idx+1:]

	if key == "" {
		return "", "", fmt.Errorf("key cannot be empty")
	}

	return key, value, nil
}

// parseVolume parses volume specification in format "name:capacity:/mount/path"
// Example: "mydata:10:/app/data" creates a 10GB volume named "mydata" mounted at /app/data
// Examples:
//   - "myvolume:10:/app/data" -> ("myvolume", 10.0, "/app/data")
//   - "production-db:100:/var/lib/postgres" -> ("production-db", 100.0, "/var/lib/postgres")
//   - "logs-vol:2.5:/var/log" -> ("logs-vol", 2.5, "/var/log")
func parseVolume(v string) (name string, capacity float64, mountPath string, err error) {
	// Find the last colon that precedes a path (starts with /)
	lastSlashIdx := strings.LastIndex(v, ":/")
	if lastSlashIdx == -1 {
		return "", 0, "", fmt.Errorf("must be in name:capacity:/mount/path format")
	}

	mountPath = v[lastSlashIdx+1:]
	prefix := v[:lastSlashIdx]

	// Split prefix into name:capacity
	colonIdx := strings.LastIndex(prefix, ":")
	if colonIdx == -1 {
		return "", 0, "", fmt.Errorf("must be in name:capacity:/mount/path format")
	}

	name = prefix[:colonIdx]
	capacityStr := prefix[colonIdx+1:]

	if name == "" {
		return "", 0, "", fmt.Errorf("volume name cannot be empty")
	}
	if len(name) < 8 || len(name) > 64 {
		return "", 0, "", fmt.Errorf("volume name must be 8-64 characters")
	}

	capacity, err = strconv.ParseFloat(capacityStr, 64)
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid capacity: %w", err)
	}
	if capacity < 1 {
		return "", 0, "", fmt.Errorf("capacity must be at least 1 GB")
	}

	if mountPath == "" {
		return "", 0, "", fmt.Errorf("mount path cannot be empty")
	}
	if !strings.HasPrefix(mountPath, "/") {
		return "", 0, "", fmt.Errorf("mount path must be absolute (start with /)")
	}

	return name, capacity, mountPath, nil
}

// parseConsulLink parses a Consul link specification in the format "VAR_NAME=service-name"
// This creates an environment variable that will be populated with the Consul service address
// Examples:
//   - "DATABASE_HOST=postgres-primary" -> ("DATABASE_HOST", "postgres-primary")
//   - "REDIS_URL=redis-cache" -> ("REDIS_URL", "redis-cache")
func parseConsulLink(link string) (varName string, serviceName string, err error) {
	idx := strings.Index(link, "=")
	if idx == -1 {
		return "", "", fmt.Errorf("must be in VAR_NAME=service-name format")
	}

	varName = link[:idx]
	serviceName = link[idx+1:]

	if varName == "" {
		return "", "", fmt.Errorf("variable name cannot be empty")
	}
	if serviceName == "" {
		return "", "", fmt.Errorf("service name cannot be empty")
	}

	return varName, serviceName, nil
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
