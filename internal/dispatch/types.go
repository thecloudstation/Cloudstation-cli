package dispatch

import (
	"encoding/json"
	"strconv"
)

// TaskType represents the type of dispatch task
type TaskType string

const (
	TaskDeployRepository   TaskType = "deploy-repository"
	TaskRedeployRepository TaskType = "redeploy-repository"
	TaskDeployImage        TaskType = "deploy-image"
	TaskDestroyJob         TaskType = "destroy-job-pack"
)

// FlexInt is a type that can unmarshal from both string and int
type FlexInt int

// UnmarshalJSON implements json.Unmarshaler for FlexInt
func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*fi = FlexInt(i)
		return nil
	}

	// Try to unmarshal as string
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Convert string to int
	if s == "" {
		*fi = FlexInt(0)
		return nil
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	*fi = FlexInt(i)
	return nil
}

// FlexString is a type that can unmarshal from both string and int/number
type FlexString string

// UnmarshalJSON implements json.Unmarshaler for FlexString
func (fs *FlexString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexString(s)
		return nil
	}

	// Try to unmarshal as int
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*fs = FlexString(strconv.Itoa(i))
		return nil
	}

	// Try to unmarshal as float (in case it's a large number represented as float)
	var f float64
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}

	*fs = FlexString(strconv.FormatFloat(f, 'f', 0, 64))
	return nil
}

// BuildOptions represents build configuration options
type BuildOptions struct {
	Builder           string            `json:"builder"`
	DockerfilePath    string            `json:"dockerfilePath,omitempty"`
	BuildCommand      string            `json:"buildCommand,omitempty"`
	StartCommand      string            `json:"startCommand,omitempty"`
	RootDirectory     string            `json:"rootDirectory,omitempty"`
	BuildArgs         map[string]string `json:"buildArgs,omitempty"`
	StaticBuildEnv    map[string]string `json:"staticBuildEnv,omitempty"`
	DisablePush       bool              `json:"disablePush,omitempty"`
	PrivateRegistry   bool              `json:"privateRegistry,omitempty"`
	RegistryURL       string            `json:"registryUrl,omitempty"`
	RegistryNamespace string            `json:"registryNamespace,omitempty"`
	RegistryUsername  string            `json:"registryUsername,omitempty"`
	RegistryPassword  string            `json:"registryPassword,omitempty"`
}

// RegistrySettings represents registry pack configuration
type RegistrySettings struct {
	Pack           string `json:"pack"`
	RegistryName   string `json:"registryName"`
	RegistryRef    string `json:"registryRef"`
	RegistrySource string `json:"registrySource"`
	RegistryTarget string `json:"registryTarget"`
	RegistryToken  string `json:"registryToken"`
	UseEmbedded    bool   `json:"useEmbedded,omitempty"`
}

// NetworkPortSettings represents network port configuration
type NetworkPortSettings struct {
	PortNumber     FlexInt             `json:"portNumber"`
	PortType       string              `json:"portType"`
	Public         bool                `json:"public"`
	Domain         string              `json:"domain,omitempty"`
	CustomDomain   string              `json:"custom_domain,omitempty"`
	HasHealthCheck string              `json:"has_health_check,omitempty"`
	HealthCheck    HealthCheckSettings `json:"health_check,omitempty"`
}

// HealthCheckSettings represents health check configuration
type HealthCheckSettings struct {
	Type     string  `json:"type"`
	Path     string  `json:"path"`
	Interval string  `json:"interval"`
	Timeout  string  `json:"timeout"`
	Port     FlexInt `json:"port"`
}

// ConsulLinkedServiceSettings represents a linked Consul service
type ConsulLinkedServiceSettings struct {
	VariableName      string `json:"variableName"`
	ConsulServiceName string `json:"consulServiceName"`
}

// ConsulSettings represents Consul service discovery configuration
type ConsulSettings struct {
	ServiceName    string                        `json:"serviceName"`
	Tags           []string                      `json:"tags,omitempty"`
	ServicePort    FlexInt                       `json:"servicePort,omitempty"`
	LinkedServices []ConsulLinkedServiceSettings `json:"linkedServices,omitempty"`
}

// CSIVolumeSettings represents CSI volume configuration
type CSIVolumeSettings struct {
	ID         string   `json:"id"`
	MountPaths []string `json:"mountPaths"`
}

// JobTypeConfig represents job type configuration
type JobTypeConfig struct {
	Type            string   `json:"type"`
	Cron            string   `json:"cron,omitempty"`
	ProhibitOverlap bool     `json:"prohibit_overlap,omitempty"`
	Payload         string   `json:"payload,omitempty"`
	MetaRequired    []string `json:"meta_required,omitempty"`
}

// UpdateParameters represents deployment update configuration
type UpdateParameters struct {
	MinHealthyTime   string  `json:"minHealthyTime,omitempty"`
	HealthyDeadline  string  `json:"healthyDeadline,omitempty"`
	ProgressDeadline string  `json:"progressDeadline,omitempty"`
	AutoRevert       bool    `json:"autoRevert,omitempty"`
	AutoPromote      bool    `json:"autoPromote,omitempty"`
	MaxParallel      FlexInt `json:"maxParallel,omitempty"`
	Canary           FlexInt `json:"canary,omitempty"`
}

// VaultLinkedSecretPath represents a linked Vault secret path
type VaultLinkedSecretPath struct {
	Prefix     string `json:"prefix"`
	SecretPath string `json:"secretPath"`
}

// VaultLinkedSecret represents a linked Vault secret
type VaultLinkedSecret struct {
	Secret   string `json:"secret"`
	Template string `json:"template"`
}

// TLSSettings represents TLS certificate configuration
type TLSSettings struct {
	CertPath   string `json:"cert_path"`
	KeyPath    string `json:"key_path"`
	CommonName string `json:"common_name"`
	PkaPath    string `json:"pka_path"`
	TTL        string `json:"ttl"`
}

// TemplateStringVariable represents a template string variable
type TemplateStringVariable struct {
	Name              string   `json:"name"`
	Pattern           string   `json:"pattern"`
	ServiceName       string   `json:"serviceName"`
	ServiceSecretPath string   `json:"serviceSecretPath"`
	LinkedVars        []string `json:"linkedVars"`
}

// ServiceConfigFile represents a service configuration file
type ServiceConfigFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// BaseDeploymentParams contains fields common to all deployment types
type BaseDeploymentParams struct {
	// Basic identifiers
	JobID           string     `json:"jobId"`
	DeploymentID    string     `json:"deploymentId"`
	ServiceID       string     `json:"serviceId"`
	TeamID          string     `json:"teamId,omitempty"`
	UserID          FlexInt    `json:"userId"`
	OwnerID         FlexString `json:"ownerId"`
	DeploymentJobID FlexInt    `json:"deploymentJobId"`
	ProjectID       string     `json:"projectId,omitempty"`
	ImageName       string     `json:"imageName"`
	ImageTag        string     `json:"imageTag"`
	Deploy          string     `json:"deploy"`
	ReplicaCount    FlexInt    `json:"replicaCount,omitempty"`

	// Nomad configuration
	NomadAddress string `json:"nomadAddress,omitempty"`
	NomadToken   string `json:"nomadToken,omitempty"`
	NodePool     string `json:"node_pool,omitempty"`

	// Vault configuration
	VaultAddress           string                  `json:"vaultAddress,omitempty"`
	RoleID                 string                  `json:"roleId,omitempty"`
	SecretID               string                  `json:"secretId,omitempty"`
	SecretsPath            string                  `json:"secretsPath,omitempty"`
	SharedSecretPath       string                  `json:"sharedSecretPath,omitempty"`
	UsesKvEngine           *bool                   `json:"usesKvEngine,omitempty"`
	OwnerUsesKvEngine      *bool                   `json:"ownerUsesKvEngine,omitempty"`
	VaultLinkedSecretPaths []VaultLinkedSecretPath `json:"vaultLinkedSecretPaths,omitempty"`
	VaultLinkedSecrets     []VaultLinkedSecret     `json:"vaultLinkedSecrets,omitempty"`

	// Registry configuration
	Registry                RegistrySettings `json:"registry,omitempty"`
	PrivateRegistry         string           `json:"privateRegistry,omitempty"`
	PrivateRegistryProvider string           `json:"privateRegistryProvider,omitempty"`

	// Resource configuration
	CPU      FlexInt `json:"cpu,omitempty"`
	RAM      FlexInt `json:"ram,omitempty"`
	GPU      FlexInt `json:"gpu,omitempty"`
	GPUModel string  `json:"gpu_model,omitempty"`

	// Networking
	Networks []NetworkPortSettings `json:"networks,omitempty"`

	// Consul
	Consul ConsulSettings `json:"consul,omitempty"`

	// Storage
	CSIVolume []CSIVolumeSettings `json:"csiVolume,omitempty"`

	// Restart policy
	RestartMode     string `json:"restartMode,omitempty"`
	RestartAttempts int    `json:"restartAttempts,omitempty"`

	// Job configuration
	JobConfig *JobTypeConfig `json:"jobConfig,omitempty"`

	// Container configuration
	Command    string   `json:"command,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	DockerUser string   `json:"docker_user,omitempty"`

	// Deployment tracking
	DeploymentCount int `json:"deploymentCount,omitempty"`

	// Advanced features
	Update                  *UpdateParameters        `json:"update,omitempty"`
	TemplateStringVariables []TemplateStringVariable `json:"templateStringVariables,omitempty"`
	ConfigFiles             []ServiceConfigFile      `json:"configFiles,omitempty"`
	Regions                 string                   `json:"regions,omitempty"`
	TLS                     *TLSSettings             `json:"tls,omitempty"`

	// Cloud provider
	CloudRegion      string `json:"cloudRegion,omitempty"`
	CloudProvider    string `json:"cloudProvider,omitempty"`
	ClusterDomain    string `json:"clusterDomain,omitempty"`
	ClusterTCPDomain string `json:"clusterTcpDomain,omitempty"`

	// Backend API integration
	BackendURL  string `json:"backendUrl,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`

	// Legacy
	NomadPackConfig map[string]interface{} `json:"nomadPackConfig,omitempty"`
}

// DeployRepositoryParams represents parameters for deploying from a repository
type DeployRepositoryParams struct {
	BaseDeploymentParams

	// Repository-specific fields
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	GitPass    string `json:"gitPass,omitempty"`
	Provider   string `json:"provider,omitempty"` // github, gitlab, bitbucket

	// Local upload source (alternative to Repository)
	SourceType string `json:"sourceType,omitempty"` // "local_upload" for uploaded tarballs
	SourceUrl  string `json:"sourceUrl,omitempty"`  // URL to download the source tarball from MinIO
	UploadId   string `json:"uploadId,omitempty"`   // Upload ID for tracking

	// Build configuration (only for repository deployments)
	Build BuildOptions `json:"build"`
}

// DeployImageParams represents parameters for deploying a pre-built image
type DeployImageParams struct {
	BaseDeploymentParams

	// Build configuration (optional - mainly used for StartCommand override)
	Build BuildOptions `json:"build,omitempty"`
}

// DestroyJobInfo represents information about a job to destroy
type DestroyJobInfo struct {
	JobID        string `json:"jobId"`
	ServiceID    string `json:"serviceId"`
	NomadAddress string `json:"nomadAddress"`
	NomadToken   string `json:"nomadToken"`
}

// DestroyJobParams represents parameters for destroying jobs
type DestroyJobParams struct {
	Jobs   []DestroyJobInfo `json:"jobs"`
	Reason string           `json:"reason"` // delete, upgrade_volume, migrate_volume, suspend_account, pause_service
}

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

// Context keys for NATS log writers
const (
	CtxKeyStdoutWriter ContextKey = "stdoutWriter"
	CtxKeyStderrWriter ContextKey = "stderrWriter"
)
