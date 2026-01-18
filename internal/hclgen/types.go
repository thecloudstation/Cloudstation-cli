package hclgen

// NetworkPort represents a network port configuration
type NetworkPort struct {
	PortNumber     int
	PortType       string // tcp or http
	Public         bool
	Domain         string
	CustomDomain   string
	HasHealthCheck string
	HealthCheck    HealthCheckConfig
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Type     string
	Path     string
	Interval string
	Timeout  string
	Port     int
}

// RegistryConfig represents registry configuration for nomad-pack
type RegistryConfig struct {
	Pack           string
	RegistryName   string
	RegistryRef    string
	RegistrySource string
	RegistryTarget string
	RegistryToken  string
	UseEmbedded    bool
}

// ConsulLinkedService represents a linked Consul service
type ConsulLinkedService struct {
	VariableName      string
	ConsulServiceName string
}

// ConsulConfig represents Consul service discovery configuration
type ConsulConfig struct {
	ServiceName    string
	Tags           []string
	ServicePort    int
	LinkedServices []ConsulLinkedService
}

// TLSConfig represents TLS certificate configuration
type TLSConfig struct {
	CertPath   string
	KeyPath    string
	CommonName string
	PkaPath    string
	TTL        string
}

// CSIVolume represents a CSI volume configuration
type CSIVolume struct {
	ID         string
	MountPaths []string
}

// JobTypeConfig represents job type configuration
type JobTypeConfig struct {
	Type            string // service, batch, sysbatch, system
	Cron            string
	ProhibitOverlap bool
	Payload         string
	MetaRequired    []string
}

// UpdateParameters represents deployment update configuration
type UpdateParameters struct {
	MinHealthyTime   string
	HealthyDeadline  string
	ProgressDeadline string
	AutoRevert       bool
	AutoPromote      bool
	MaxParallel      int
	Canary           int
}

// VaultLinkedSecretPath represents a linked Vault secret path
type VaultLinkedSecretPath struct {
	Prefix     string
	SecretPath string
}

// VaultLinkedSecret represents a linked Vault secret
type VaultLinkedSecret struct {
	Secret   string
	Template string
}

// TemplateStringVariable represents a template string variable
type TemplateStringVariable struct {
	Name              string
	Pattern           string
	ServiceName       string
	ServiceSecretPath string
	LinkedVars        []string
}

// ServiceConfigFile represents a service configuration file
type ServiceConfigFile struct {
	Path    string
	Content string
}

// DeploymentParams represents deployment configuration parameters
type DeploymentParams struct {
	// Basic identifiers
	JobID       string
	ImageName   string
	ImageTag    string
	BuilderType string // csdocker, nixpacks, noop
	DeployType  string // nomad-pack, noop

	// Multi-tenancy
	OwnerID      string
	ProjectID    string
	ServiceID    string
	TeamID       string
	DeploymentID string

	// Nomad configuration
	NomadAddress string
	NomadToken   string
	NodePool     string

	// Vault configuration
	VaultAddress           string
	RoleID                 string
	SecretID               string
	SecretsPath            string
	SharedSecretPath       string
	UsesKvEngine           *bool
	OwnerUsesKvEngine      *bool
	VaultLinkedSecretPaths []VaultLinkedSecretPath
	VaultLinkedSecrets     []VaultLinkedSecret

	// Registry configuration
	Registry                RegistryConfig
	PrivateRegistry         string
	PrivateRegistryProvider string
	RegistryUsername        string
	RegistryPassword        string
	RegistryURL             string
	PushToRegistry          bool // Deprecated: Use DisablePush instead
	DisablePush             bool // When true, skip registry push phase

	// Resource configuration
	CPU          int
	RAM          int
	GPU          int
	GPUModel     string
	ReplicaCount int

	// Networking
	Networks []NetworkPort

	// Consul
	Consul ConsulConfig

	// Storage
	CSIVolumes []CSIVolume

	// Restart policy
	RestartMode     string
	RestartAttempts int

	// Job configuration
	JobConfig *JobTypeConfig

	// Container configuration
	Command    string
	Entrypoint []string
	DockerUser string

	// Build configuration
	DockerfilePath string
	BuildArgs      map[string]string
	StaticBuildEnv map[string]string
	BuildCommand   string
	StartCommand   string
	RootDirectory  string

	// Deployment tracking
	DeploymentCount int

	// Advanced features
	Update                  *UpdateParameters
	TemplateStringVariables []TemplateStringVariable
	ConfigFiles             []ServiceConfigFile
	Regions                 string
	TLS                     *TLSConfig

	// Cloud provider
	CloudRegion      string
	CloudProvider    string
	ClusterDomain    string
	ClusterTCPDomain string

	// Legacy
	VaultConfig     map[string]interface{}
	NomadPackConfig map[string]interface{}
}

// BuildConfig represents build stanza configuration
type BuildConfig struct {
	Use          string
	DockerConfig map[string]interface{}
	BuildEnv     map[string]string
}

// DeployConfig represents deploy stanza configuration
type DeployConfig struct {
	Use         string
	ImageName   string
	ImageTag    string
	PackConfig  map[string]interface{}
	Environment map[string]string
}
