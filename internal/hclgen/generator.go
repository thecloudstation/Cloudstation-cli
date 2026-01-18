package hclgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

// GenerateConfig generates an HCL configuration from deployment parameters
func GenerateConfig(params DeploymentParams) (string, error) {
	// Set defaults
	if params.JobID == "" {
		return "", fmt.Errorf("jobID is required")
	}

	// Normalize builder name: docker -> csdocker, "" -> railpack (zero-config default)
	if params.BuilderType == "docker" {
		params.BuilderType = "csdocker"
	} else if params.BuilderType == "" {
		// Default to railpack for zero-config builds
		// Railpack is Railway's successor to nixpacks with better detection
		params.BuilderType = "railpack"
	}

	// Normalize deploy type - handle "nomad-pack" vs "nomadpack"
	if params.DeployType == "" {
		params.DeployType = "nomad-pack"
	}

	// Set defaults
	if params.ImageName == "" {
		params.ImageName = params.JobID
	}

	if params.ImageTag == "" {
		params.ImageTag = "latest"
	}

	if params.ReplicaCount == 0 {
		params.ReplicaCount = 1
	}

	if params.DockerfilePath == "" {
		params.DockerfilePath = "Dockerfile"
	}

	// Generate HCL content
	var hcl strings.Builder

	// Project declaration
	hcl.WriteString(fmt.Sprintf("project = \"%s\"\n\n", params.JobID))

	// Runner block
	hcl.WriteString("runner {\n")
	hcl.WriteString("  enabled = true\n")
	hcl.WriteString("}\n\n")

	// App block
	hcl.WriteString(fmt.Sprintf("app \"%s\" {\n", params.JobID))

	// Build stanza
	hcl.WriteString(generateBuildStanza(params))
	hcl.WriteString("\n")

	// Registry stanza (if configured)
	registryStanza := generateRegistryStanza(params)
	if registryStanza != "" {
		hcl.WriteString(registryStanza)
		hcl.WriteString("\n")
	}

	// Deploy stanza
	hcl.WriteString(generateDeployStanza(params))

	hcl.WriteString("}\n")

	// Add variable blocks for registry credentials when push is needed
	// Credentials can come from explicit params or REGISTRY_USERNAME/REGISTRY_PASSWORD env vars
	// 1. Push is not disabled
	// 2. Not a noop builder (image deployments don't push)
	if !params.DisablePush && params.BuilderType != "noop" {
		hcl.WriteString("\n")
		hcl.WriteString("variable \"registry_username\" {\n")
		hcl.WriteString("  type = \"string\"\n")
		hcl.WriteString("  sensitive = true\n")
		hcl.WriteString("  env = [\"REGISTRY_USERNAME\"]\n")
		hcl.WriteString("}\n\n")
		hcl.WriteString("variable \"registry_password\" {\n")
		hcl.WriteString("  type = \"string\"\n")
		hcl.WriteString("  sensitive = true\n")
		hcl.WriteString("  env = [\"REGISTRY_PASSWORD\"]\n")
		hcl.WriteString("}\n")
	}

	return hcl.String(), nil
}

// generateBuildStanza generates the build stanza
func generateBuildStanza(params DeploymentParams) string {
	var build strings.Builder

	build.WriteString("  build {\n")
	build.WriteString(fmt.Sprintf("    use = \"%s\"\n", params.BuilderType))

	// Add builder-specific configuration as flat attributes
	if params.BuilderType != "noop" {
		if params.ImageName != "" {
			build.WriteString(fmt.Sprintf("    name = \"%s\"\n", params.ImageName))
		}
		if params.ImageTag != "" {
			build.WriteString(fmt.Sprintf("    tag = \"%s\"\n", params.ImageTag))
		}
		if params.VaultAddress != "" {
			build.WriteString(fmt.Sprintf("    vault_address = \"%s\"\n", params.VaultAddress))
		}
		if params.RoleID != "" {
			build.WriteString(fmt.Sprintf("    role_id = \"%s\"\n", params.RoleID))
		}
		if params.SecretID != "" {
			build.WriteString(fmt.Sprintf("    secret_id = \"%s\"\n", params.SecretID))
		}
		if params.SecretsPath != "" {
			build.WriteString(fmt.Sprintf("    secrets_path = \"%s\"\n", params.SecretsPath))
		}
		if params.RootDirectory != "" {
			// Defensive sanitization - strip leading/trailing slashes
			context := strings.TrimLeft(params.RootDirectory, "/")
			context = strings.TrimRight(context, "/")
			if context != "" && context != "." {
				build.WriteString(fmt.Sprintf("    context = \"%s\"\n", context))
			} else {
				build.WriteString("    context = \".\"\n")
			}
		} else {
			build.WriteString("    context = \".\"\n")
		}
		if params.DockerfilePath != "" && params.BuilderType == "csdocker" {
			build.WriteString(fmt.Sprintf("    dockerfile = \"%s\"\n", params.DockerfilePath))
		}
		if params.BuildCommand != "" {
			build.WriteString(fmt.Sprintf("    build_cmd = \"%s\"\n", params.BuildCommand))
		}
		if params.StartCommand != "" {
			build.WriteString(fmt.Sprintf("    start_cmd = \"%s\"\n", params.StartCommand))
		}
	}

	build.WriteString("  }\n")

	return build.String()
}

// generateRegistryStanza generates the registry stanza
func generateRegistryStanza(params DeploymentParams) string {
	// Skip registry stanza if:
	// 1. Explicitly disabled via DisablePush
	// 2. Using noop builder (image deployment - no build, no push needed)
	// 3. No credentials provided (can't push without auth)
	if params.DisablePush {
		return ""
	}
	if params.BuilderType == "noop" {
		return "" // Image deployments don't need registry push
	}
	// Credentials can come from explicit params or REGISTRY_USERNAME/REGISTRY_PASSWORD env vars at runtime
	// Generate registry block regardless - env var fallback handles credentials

	var registry strings.Builder

	registry.WriteString("  registry {\n")
	registry.WriteString("    use = \"docker\"\n")

	// Determine the image name to use
	imageName := params.ImageName
	if imageName == "" {
		imageName = params.JobID
	}

	// Extract registry URL from ImageName if it contains a registry prefix
	registryURL := params.RegistryURL
	if registryURL == "" && strings.Contains(imageName, "/") {
		parts := strings.SplitN(imageName, "/", 2)
		// Check if first part looks like a registry (contains . or :)
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			registryURL = parts[0]
			imageName = parts[1]
		}
	}

	// Build the full image name with registry prefix
	if registryURL != "" {
		registry.WriteString(fmt.Sprintf("    image = \"%s/%s\"\n", registryURL, imageName))
	} else {
		registry.WriteString(fmt.Sprintf("    image = \"%s\"\n", imageName))
	}

	// Add tag
	tag := params.ImageTag
	if tag == "" {
		tag = "latest"
	}
	registry.WriteString(fmt.Sprintf("    tag = \"%s\"\n", tag))

	// Use variable references for credentials to keep secrets out of HCL
	registry.WriteString("    username = var.registry_username\n")
	registry.WriteString("    password = var.registry_password\n")

	registry.WriteString("  }\n")

	return registry.String()
}

// generateDeployStanza generates the deploy stanza
func generateDeployStanza(params DeploymentParams) string {
	var deploy strings.Builder

	deploy.WriteString("  deploy {\n")
	deploy.WriteString(fmt.Sprintf("    use = \"%s\"\n", params.DeployType))

	// Add nomad-pack specific configuration as flat attributes
	if params.DeployType != "noop" {
		if params.JobID != "" {
			deploy.WriteString(fmt.Sprintf("    deployment_name = \"%s\"\n", params.JobID))
		}

		// Registry configuration
		if params.Registry.Pack != "" {
			deploy.WriteString(fmt.Sprintf("    pack = \"%s\"\n", params.Registry.Pack))
			if params.Registry.UseEmbedded {
				deploy.WriteString("    use_embedded = true\n")
			}
			deploy.WriteString(fmt.Sprintf("    registry_name = \"%s\"\n", params.Registry.RegistryName))
			deploy.WriteString(fmt.Sprintf("    registry_ref = \"%s\"\n", params.Registry.RegistryRef))
			deploy.WriteString(fmt.Sprintf("    registry_source = \"%s\"\n", params.Registry.RegistrySource))
			deploy.WriteString(fmt.Sprintf("    registry_target = \"%s\"\n", params.Registry.RegistryTarget))

			if params.Registry.RegistryToken != "" {
				// Handle var.REGISTRY_TOKEN reference
				if strings.HasPrefix(params.Registry.RegistryToken, "var.") {
					deploy.WriteString(fmt.Sprintf("    registry_token = %s\n", params.Registry.RegistryToken))
				} else {
					deploy.WriteString(fmt.Sprintf("    registry_token = \"%s\"\n", params.Registry.RegistryToken))
				}
			}
		}

		// Nomad configuration
		if params.NomadToken != "" {
			deploy.WriteString(fmt.Sprintf("    nomad_token = \"%s\"\n", params.NomadToken))
		}
		if params.NomadAddress != "" {
			deploy.WriteString(fmt.Sprintf("    nomad_addr = \"%s\"\n", params.NomadAddress))
		}

		// Variable files reference
		deploy.WriteString("    variable_files = [\"vars.hcl\"]\n")
	}

	deploy.WriteString("  }\n")

	return deploy.String()
}

// GenerateVarsFile generates the vars.hcl content
// Field order matches cs-runner exactly for cloudstation-packs compatibility
func GenerateVarsFile(params DeploymentParams, art *artifact.Artifact) string {
	var vars strings.Builder

	// 1. job_name
	vars.WriteString(fmt.Sprintf("job_name = \"%s\"\n", params.JobID))

	// 2. count
	if params.ReplicaCount > 0 {
		vars.WriteString(fmt.Sprintf("count = %d\n", params.ReplicaCount))
	}

	// 3. secret_path
	if params.SecretsPath != "" {
		vars.WriteString(fmt.Sprintf("secret_path = \"%s\"\n", params.SecretsPath))
	}

	// 4. restart_attempts
	// 5. restart_mode
	restartMode := getRestartMode(params.RestartMode)
	restartAttempts := getRestartAttempts(params.RestartMode, params.RestartAttempts)
	vars.WriteString(fmt.Sprintf("restart_attempts = %d\n", restartAttempts))
	vars.WriteString(fmt.Sprintf("restart_mode = \"%s\"\n", restartMode))

	// 6. resources (including memory_max = 2x memory)
	if params.CPU > 0 || params.RAM > 0 || params.GPU > 0 {
		memoryMax := params.RAM * 2
		if memoryMax < 512 {
			memoryMax = 512
		}
		vars.WriteString(fmt.Sprintf("resources = {cpu=%d, memory=%d, memory_max=%d, gpu=%d}\n", params.CPU, params.RAM, memoryMax, params.GPU))
	}

	// 7. gpu_type (if gpu > 0)
	if params.GPUModel != "" && params.GPU > 0 {
		vars.WriteString(fmt.Sprintf("gpu_type = \"%s\"\n", params.GPUModel))
	}

	// 8. node_pool (if present)
	if params.NodePool != "" {
		vars.WriteString(fmt.Sprintf("node_pool = \"%s\"\n", params.NodePool))
	}

	// 9. user_id
	if params.OwnerID != "" {
		vars.WriteString(fmt.Sprintf("user_id = \"%s\"\n", params.OwnerID))
	}

	// 10. alloc_id
	if params.DeploymentID != "" {
		vars.WriteString(fmt.Sprintf("alloc_id = \"%s\"\n", params.DeploymentID))
	}

	// 11. project_id
	if params.ProjectID != "" {
		vars.WriteString(fmt.Sprintf("project_id = \"%s\"\n", params.ProjectID))
	}

	// 12. service_id
	if params.ServiceID != "" {
		vars.WriteString(fmt.Sprintf("service_id = \"%s\"\n", params.ServiceID))
	}

	// 13. shared_secret_path (if present)
	if params.SharedSecretPath != "" {
		vars.WriteString(fmt.Sprintf("shared_secret_path = \"%s\"\n", params.SharedSecretPath))
	}

	// 14. uses_kv_engine (if defined)
	if params.UsesKvEngine != nil {
		vars.WriteString(fmt.Sprintf("uses_kv_engine = %t\n", *params.UsesKvEngine))
	}

	// 15. owner_uses_kv_engine (if defined)
	if params.OwnerUsesKvEngine != nil {
		vars.WriteString(fmt.Sprintf("owner_uses_kv_engine = %t\n", *params.OwnerUsesKvEngine))
	}

	// 16. regions (always output - required by pack)
	if params.Regions != "" && params.GPU == 0 {
		vars.WriteString(fmt.Sprintf("regions = \"%s\"\n", params.Regions))
	} else {
		vars.WriteString("regions = \"\"\n")
	}

	// 17. private_registry (if present)
	if params.PrivateRegistry != "" {
		vars.WriteString(fmt.Sprintf("private_registry = \"%s\"\n", params.PrivateRegistry))
	}

	// 18. private_registry_provider (if present)
	if params.PrivateRegistryProvider != "" {
		vars.WriteString(fmt.Sprintf("private_registry_provider = \"%s\"\n", params.PrivateRegistryProvider))
	}

	// 19. user (docker_user)
	if params.DockerUser != "" {
		vars.WriteString(fmt.Sprintf("user = \"%s\"\n", params.DockerUser))
	} else {
		vars.WriteString("user = \"0\"\n")
	}

	// 20. command (if present)
	if params.Command != "" {
		vars.WriteString(fmt.Sprintf("command = \"%s\"\n", params.Command))
	}

	// 21. image
	if params.ImageName != "" {
		imageURL := params.ImageName
		if params.ImageTag != "" {
			imageURL = fmt.Sprintf("%s:%s", params.ImageName, params.ImageTag)
		}
		vars.WriteString(fmt.Sprintf("image = \"%s\"\n", imageURL))
	}

	// 22. use_csi_volume, volume_name, volume_mount_destination
	if len(params.CSIVolumes) > 0 {
		vars.WriteString("use_csi_volume = true\n")
		vol := params.CSIVolumes[0]
		// Remove array index pattern like vol_xxx[0]
		volumeID := regexp.MustCompile(`^(vol_.+)\[\d+\]$`).ReplaceAllString(vol.ID, "$1")
		vars.WriteString(fmt.Sprintf("volume_name = \"%s\"\n", volumeID))

		if len(vol.MountPaths) > 0 {
			paths := make([]string, len(vol.MountPaths))
			for i, p := range vol.MountPaths {
				paths[i] = fmt.Sprintf("\"%s\"", p)
			}
			vars.WriteString(fmt.Sprintf("volume_mount_destination = [%s]\n", strings.Join(paths, ", ")))
		}
	} else {
		vars.WriteString("use_csi_volume = false\n")
	}

	// 23. config_files (always output - required by pack)
	if len(params.ConfigFiles) > 0 {
		configFiles := make([]string, len(params.ConfigFiles))
		for i, cf := range params.ConfigFiles {
			configFiles[i] = fmt.Sprintf("{ path=\"%s\", content=\"%s\" }", cf.Path, cf.Content)
		}
		vars.WriteString(fmt.Sprintf("config_files = [%s]\n", strings.Join(configFiles, ", ")))
	} else {
		vars.WriteString("config_files = []\n")
	}

	// 24. consul_service_name, consul_tags, consul_linked_services (if consul configured)
	vars.WriteString(generateConsulVariables(params.Consul, params.ReplicaCount))

	// 25. entrypoint (if present)
	if len(params.Entrypoint) > 0 {
		entrypoints := make([]string, len(params.Entrypoint))
		for i, ep := range params.Entrypoint {
			entrypoints[i] = fmt.Sprintf("\"%s\"", ep)
		}
		vars.WriteString(fmt.Sprintf("entrypoint = \"[%s]\"\n", strings.Join(entrypoints, ", ")))
	}

	// 26. template (always output - required by pack)
	if len(params.TemplateStringVariables) > 0 {
		templateVars := make([]string, len(params.TemplateStringVariables))
		for i, tv := range params.TemplateStringVariables {
			linkedVars := make([]string, len(tv.LinkedVars))
			for j, lv := range tv.LinkedVars {
				linkedVars[j] = fmt.Sprintf("\"%s\"", lv)
			}
			templateVars[i] = fmt.Sprintf("{name=\"%s\",pattern=\"%s\",service_name=\"%s\",service_secret_path=\"%s\",linked_vars=[%s]}",
				tv.Name, tv.Pattern, tv.ServiceName, tv.ServiceSecretPath, strings.Join(linkedVars, ","))
		}
		vars.WriteString(fmt.Sprintf("template = [%s]\n", strings.Join(templateVars, ", ")))
	} else {
		vars.WriteString("template = []\n")
	}

	// 27. cluster_domain (if present - pack has default, so optional)
	// Used by template to construct full FQDN for Consul service tags
	if params.ClusterDomain != "" {
		vars.WriteString(fmt.Sprintf("cluster_domain = \"%s\"\n", params.ClusterDomain))
	}

	// 28. network (if present or detected)
	vars.WriteString(generateNetworking(params.Networks, art, params.BuilderType))

	// 29. use_tls, tls (TLS variables)
	vars.WriteString(generateTLSVariables(params.TLS))

	// vault_linked_secrets (always output - required by pack)
	vars.WriteString("vault_linked_secrets = []\n")

	// 30. "update" (if present)
	if params.Update != nil {
		vars.WriteString(generateUpdateParams(params.Update))
	}

	// 31. job_config (if present)
	if params.JobConfig != nil {
		vars.WriteString(generateJobTypeConfig(params.JobConfig))
	}

	// 32. args (if vault linked secrets OR start command)
	if len(params.VaultLinkedSecrets) > 0 || (params.BuilderType != "nixpacks" && params.BuilderType != "railpack" && params.StartCommand != "") {
		var inject string
		for _, secret := range params.VaultLinkedSecrets {
			inject += fmt.Sprintf("export %s=%s;", secret.Secret, secret.Template)
		}
		if params.BuilderType != "nixpacks" && params.BuilderType != "railpack" && params.StartCommand != "" {
			inject += params.StartCommand
		}
		if inject != "" {
			vars.WriteString(fmt.Sprintf("args = [\"%s\"]\n", inject))
		}
	}

	return vars.String()
}

// Helper functions

func getRestartMode(mode string) string {
	switch mode {
	case "fail":
		return "fail"
	case "never":
		// Map to fail since never isn't supported yet
		return "fail"
	case "delay":
		return "delay"
	default:
		return "fail"
	}
}

func getRestartAttempts(mode string, attempts int) int {
	if mode == "never" {
		return 0
	}
	if attempts > 0 {
		return attempts
	}
	return 3
}

func ensureIntervalHasUnit(interval string) string {
	numberPattern := regexp.MustCompile(`^[0-9]+$`)
	durationPattern := regexp.MustCompile(`^[0-9]+[smhd]$`)

	if numberPattern.MatchString(interval) {
		return interval + "s"
	} else if durationPattern.MatchString(interval) {
		return interval
	}
	return "30s"
}

func generateTLSVariables(tls *TLSConfig) string {
	if tls == nil {
		return "use_tls = false\n"
	}

	var vars strings.Builder
	vars.WriteString("use_tls = true\n")
	vars.WriteString(fmt.Sprintf("tls = [{ cert_path=\"%s\", key_path=\"%s\", common_name=\"%s\", pka_path=\"%s\", ttl=\"%s\" }]\n",
		tls.CertPath, tls.KeyPath, tls.CommonName, tls.PkaPath, tls.TTL))

	return vars.String()
}

func generateConsulVariables(consul ConsulConfig, replicaCount int) string {
	if consul.ServiceName == "" {
		return ""
	}

	var vars strings.Builder

	vars.WriteString(fmt.Sprintf("consul_service_name = \"%s\"\n", consul.ServiceName))

	// Linked services - always output (required by pack, even if empty)
	if len(consul.LinkedServices) > 0 {
		linkedServices := make([]string, len(consul.LinkedServices))
		for i, ls := range consul.LinkedServices {
			linkedServices[i] = fmt.Sprintf("{key=\"%s\", value=\"%s\"}", ls.VariableName, ls.ConsulServiceName)
		}
		vars.WriteString(fmt.Sprintf("consul_linked_services = [%s]\n", strings.Join(linkedServices, ", ")))
	} else {
		vars.WriteString("consul_linked_services = []\n")
	}

	return vars.String()
}

// isEmptyString checks if a string is empty or whitespace-only
func isEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

// isValidHealthCheckType checks if a health check type is valid
// Valid types: grpc, tcp, http, script
// Invalid types: "", "no", "none", or any other value
func isValidHealthCheckType(hcType string) bool {
	switch strings.ToLower(hcType) {
	case "grpc", "tcp", "http", "script":
		return true
	default:
		return false
	}
}

// normalizeHealthCheckType returns a valid health check type
// If the input type is invalid or empty, returns "tcp" as the safe default
// Otherwise returns the input type unchanged
func normalizeHealthCheckType(hcType string) string {
	if !isValidHealthCheckType(hcType) {
		return "tcp"
	}
	return hcType
}

// generateNetworking generates HCL network configuration using a 3-tier fallback system:
//
// Tier 1 (Highest Priority): User-specified networks array from deployment payload
//   - When len(networks) > 0, user networks are used as-is
//   - User-provided values are respected and preserved (no overrides)
//   - Empty fields may receive defaults, but explicit values are never modified
//
// Tier 2 (Medium Priority): Detected ports from artifact inspection
//   - When networks array is empty but artifact.ExposedPorts exists
//   - Creates network config from first detected port
//   - Applies reasonable defaults (Public=false, HealthCheck type=tcp)
//
// Tier 3 (Lowest Priority): Framework-specific defaults
//   - When both networks and artifact ports are empty
//   - Uses GetFrameworkDefault() to determine port by builder type
//   - nixpacks=3000, csdocker=8000, railpack=3000
//
// Field Processing:
//   - Public: Respects user setting (no auto-public for HTTP)
//   - HasHealthCheck: Defaults to PortType if empty
//   - HealthCheck.Type: Defaults to "tcp" for invalid values ("no", "none", empty)
//   - HealthCheck.Path: Defaults to "/" if empty
//   - HealthCheck.Interval: Defaults to "30s" if empty, ensures time unit suffix
//   - HealthCheck.Timeout: Defaults to "30s" if empty, ensures time unit suffix
//   - HealthCheck.Port: Defaults to network.PortNumber if not specified
//
// Examples:
//
//	User provides Public=false with PortType="http" -> Public=false is preserved
//	User provides HealthCheck.Path="/custom" -> "/custom" is preserved
//	User provides empty networks -> Uses detected port or framework default
func generateNetworking(networks []NetworkPort, art *artifact.Artifact, builderType string) string {
	// 3-tier fallback logic for port detection:
	// Tier 1: User-specified networks (highest priority)
	// Tier 2: Detected ports from artifact
	// Tier 3: Framework defaults (lowest priority)

	var networksToUse []NetworkPort

	if len(networks) > 0 {
		// Tier 1: User explicitly provided networks
		networksToUse = networks
	} else if art != nil && len(art.ExposedPorts) > 0 {
		// Tier 2: Use detected ports from artifact
		detectedPort := art.ExposedPorts[0]
		networksToUse = []NetworkPort{{
			PortNumber:     detectedPort,
			PortType:       "http",
			Public:         false,
			HasHealthCheck: "tcp",
			HealthCheck: HealthCheckConfig{
				Type:     "tcp",
				Path:     "/",
				Interval: "30s",
				Timeout:  "30s",
				Port:     detectedPort,
			},
		}}
	} else {
		// Tier 3: Use framework default
		defaultPort := GetFrameworkDefault(builderType)
		networksToUse = []NetworkPort{{
			PortNumber:     defaultPort,
			PortType:       "http",
			Public:         false,
			HasHealthCheck: "tcp",
			HealthCheck: HealthCheckConfig{
				Type:     "tcp",
				Path:     "/",
				Interval: "30s",
				Timeout:  "30s",
				Port:     defaultPort,
			},
		}}
	}

	if len(networksToUse) == 0 {
		return ""
	}

	formattedPorts := make([]string, 0)
	for _, network := range networksToUse {
		if network.PortNumber == 0 {
			continue
		}

		// Respect user's explicit Public setting - do not auto-enable for HTTP
		// Note: Users can create internal-only HTTP services by setting Public=false
		public := network.Public

		// Default hasHealthCheck to port type if not explicitly specified
		hasHealthCheck := network.HasHealthCheck
		if isEmptyString(hasHealthCheck) {
			hasHealthCheck = network.PortType // Default to port type if not specified
		}

		// Health check params - use explicit validation helpers
		hcType := network.HealthCheck.Type
		if isEmptyString(hcType) {
			hcType = network.PortType
		}
		// Ensure valid health check type (must be grpc, tcp, http, or script)
		// Default to tcp for invalid or empty types
		hcType = normalizeHealthCheckType(hcType)

		// Default HTTP health check path - use "/" not a time interval
		hcPath := "/"
		if !isEmptyString(network.HealthCheck.Path) {
			hcPath = network.HealthCheck.Path // Preserve user's custom path
		}
		hcInterval := ensureIntervalHasUnit(network.HealthCheck.Interval)
		if hcInterval == "" {
			hcInterval = "30s"
		}
		hcTimeout := ensureIntervalHasUnit(network.HealthCheck.Timeout)
		if hcTimeout == "" {
			hcTimeout = "30s"
		}
		hcPort := network.PortNumber
		if network.HealthCheck.Port > 0 {
			hcPort = network.HealthCheck.Port
		}

		portStr := fmt.Sprintf("{name=\"%d\", port=%d, type=\"%s\", public=%t, domain=\"%s\", custom_domain=\"%s\", has_health_check=\"%s\", health_check={type=\"%s\",interval=\"%s\",path=\"%s\",timeout=\"%s\",port=%d}}",
			network.PortNumber,
			network.PortNumber,
			network.PortType,
			public,
			network.Domain,
			network.CustomDomain,
			hasHealthCheck,
			hcType,
			hcInterval,
			hcPath,
			hcTimeout,
			hcPort,
		)

		formattedPorts = append(formattedPorts, portStr)
	}

	if len(formattedPorts) > 0 {
		return fmt.Sprintf("network = [%s]\n", strings.Join(formattedPorts, ", "))
	}

	return ""
}

func generateUpdateParams(update *UpdateParameters) string {
	if update == nil {
		return ""
	}

	params := make([]string, 0)

	if update.MinHealthyTime != "" {
		params = append(params, fmt.Sprintf("min_healthy_time=\"%s\"", update.MinHealthyTime))
	}
	if update.HealthyDeadline != "" {
		params = append(params, fmt.Sprintf("healthy_deadline=\"%s\"", update.HealthyDeadline))
	}
	if update.ProgressDeadline != "" {
		params = append(params, fmt.Sprintf("progress_deadline=\"%s\"", update.ProgressDeadline))
	}
	params = append(params, fmt.Sprintf("auto_revert=%t", update.AutoRevert))
	params = append(params, fmt.Sprintf("auto_promote=%t", update.AutoPromote))
	if update.MaxParallel > 0 {
		params = append(params, fmt.Sprintf("max_parallel=%d", update.MaxParallel))
	}
	if update.Canary > 0 {
		params = append(params, fmt.Sprintf("canary=%d", update.Canary))
	}

	if len(params) > 0 {
		return fmt.Sprintf("\"update\" = \"{%s}\"\n", strings.Join(params, ", "))
	}

	return ""
}

func generateJobTypeConfig(jobConfig *JobTypeConfig) string {
	if jobConfig == nil {
		return ""
	}

	jobType := jobConfig.Type
	if jobType == "" {
		jobType = "service"
	}

	cron := jobConfig.Cron
	prohibitOverlap := jobConfig.ProhibitOverlap
	if jobType != "service" && jobType != "system" {
		prohibitOverlap = true
	}

	payload := jobConfig.Payload
	metaRequired := jobConfig.MetaRequired

	metaStr := ""
	if len(metaRequired) > 0 {
		metas := make([]string, len(metaRequired))
		for i, m := range metaRequired {
			metas[i] = fmt.Sprintf("\"%s\"", m)
		}
		metaStr = fmt.Sprintf("[%s]", strings.Join(metas, ","))
	} else {
		metaStr = "[]"
	}

	return fmt.Sprintf("job_config = {type=\"%s\", cron=\"%s\", prohibit_overlap=\"%t\", payload=\"%s\", meta_required=%s}\n",
		jobType, cron, prohibitOverlap, payload, metaStr)
}

// WriteConfigFile writes the HCL config to a file in the specified directory
func WriteConfigFile(config string, directory string) (string, error) {
	if directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the config file
	configPath := filepath.Join(directory, "cloudstation.hcl")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return configPath, nil
}

// WriteVarsFile writes the vars.hcl file in the specified directory
func WriteVarsFile(varsContent string, directory string) (string, error) {
	if directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the vars file
	varsPath := filepath.Join(directory, "vars.hcl")
	if err := os.WriteFile(varsPath, []byte(varsContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write vars file: %w", err)
	}

	return varsPath, nil
}
