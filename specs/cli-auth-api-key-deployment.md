# Plan: CLI Authentication with API Key for Direct Deployment

## Task Description

Implement a complete CLI authentication system for the CloudStation Orchestrator that allows users to:
1. Authenticate using API keys from their CloudStation account
2. Deploy directly to CloudStation infrastructure from the terminal (bypassing the web UI dispatch flow)
3. Access Vault secrets associated with their services
4. Stream logs and receive deployment status updates via NATS

This feature enables a developer workflow similar to Vercel CLI, Railway CLI, or Fly.io CLI where users authenticate once and deploy directly from their terminal.

## Objective

When this plan is complete, users will be able to:
```bash
# Authenticate with CloudStation
$ cs login --api-key sk_xxxxxxxxx
✓ Authenticated as user@example.com
✓ Credentials saved to ~/.cloudstation/credentials

# Link local project to a CloudStation service
$ cs link --service prj_integ_xxx
✓ Linked to my-web-app

# Deploy directly to CloudStation infrastructure
$ cs up --app web
✓ Building with nixpacks...
✓ Pushing to registry.cs.io/my-web-app:abc123
✓ Fetching secrets from Vault (5 vars)
✓ Deploying to Nomad cluster
✓ Available at: https://my-web-app.cs.io
```

## Problem Statement

Currently, the CloudStation Orchestrator can only be invoked via:
1. **Nomad Dispatch** - Backend dispatches jobs with environment variables containing Vault/Nomad credentials
2. **Local HCL Config** - Requires manually setting `VAULT_*`, `NOMAD_*`, `NATS_*` environment variables

There is no way for users to:
- Authenticate with their CloudStation account from the CLI
- Have the CLI automatically resolve infrastructure credentials (Vault, Nomad, NATS)
- Deploy to their services without going through the web UI

## Solution Approach

Implement a two-part solution:

### Part 1: CS-Backend - CLI Auth Endpoint
Create a new endpoint in cs-backend that exchanges an API key for infrastructure credentials:
- Validates API key via existing auth service
- Resolves cluster credentials (Vault, Nomad, NATS) for the user's services
- Returns a `CLICredentials` response with all necessary infrastructure access

### Part 2: Cloudstation-Orchestrator - Auth Package & Commands
Create new packages and CLI commands:
- `pkg/auth/` - Credential management, storage, and API client
- `cs login` - Authenticate with API key
- `cs link` - Link local project to CloudStation service
- Modified `cs up` - Use stored credentials for deployment

## Relevant Files

### CS-Backend Files (Existing - For Reference)

**Auth System:**
- `apps/cs-backend/auth/internal/apikey/service.go` - API key validation logic (bcrypt comparison)
- `apps/cs-backend/auth/internal/apikey/handler.go:26-65` - Validate endpoint handler
- `apps/cs-backend/shared/authclient/client.go` - Auth client pattern to follow
- `apps/cs-backend/shared/authclient/apikey.go` - API key validation client methods

**Configuration & Credentials:**
- `apps/cs-backend/shared/config/config.go:128-188` - NATS/Nomad config structures
- `apps/cs-backend/shared/cluster/resolver.go:72-225` - Cluster credential resolution from Vault
- `apps/cs-backend/shared/cluster/types.go:8-32` - ClusterCredentials struct
- `apps/cs-backend/shared/vault/client.go:224-269` - Vault AppRole authentication

**Models:**
- `apps/cs-backend/shared/model/apikey.go` - ApiKey database model
- `apps/cs-backend/shared/model/user.go` - User model (returned on auth)
- `apps/cs-backend/shared/model/project.go` - Project model
- `apps/cs-backend/shared/model/integration.go` - Service (git-based) model

### Cloudstation-Orchestrator Files (Existing - For Reference)

**CLI Structure:**
- `apps/cloudstation-orchestrator/cmd/cloudstation/main.go:28-73` - App setup, command registration
- `apps/cloudstation-orchestrator/cmd/cloudstation/commands.go` - Existing CLI commands pattern
- `apps/cloudstation-orchestrator/cmd/cloudstation/dispatch.go` - Dispatch command (env var pattern)

**Backend Client:**
- `apps/cloudstation-orchestrator/pkg/backend/client.go` - HTTP client pattern to follow
- `apps/cloudstation-orchestrator/pkg/backend/types.go` - Request/Response types

**Vault Integration:**
- `apps/cloudstation-orchestrator/pkg/secrets/vault/provider.go:54-115` - Secret fetching
- `apps/cloudstation-orchestrator/pkg/secrets/vault/config.go:36-65` - Vault config parsing

**Lifecycle:**
- `apps/cloudstation-orchestrator/internal/lifecycle/executor.go` - Build → Registry → Deploy pipeline
- `apps/cloudstation-orchestrator/internal/lifecycle/secrets.go` - Secret provider detection

### New Files

**CS-Backend - CLI Auth Module:**
- `apps/cs-backend/api/internal/cliauth/handler.go` - HTTP handlers
- `apps/cs-backend/api/internal/cliauth/service.go` - Business logic
- `apps/cs-backend/api/internal/cliauth/dto.go` - Request/Response types
- `apps/cs-backend/api/internal/cliauth/routes.go` - Route registration

**Cloudstation-Orchestrator - Auth Package:**
- `apps/cloudstation-orchestrator/pkg/auth/client.go` - Auth API client
- `apps/cloudstation-orchestrator/pkg/auth/credentials.go` - Credentials management
- `apps/cloudstation-orchestrator/pkg/auth/storage.go` - File-based credential storage
- `apps/cloudstation-orchestrator/pkg/auth/types.go` - Auth types/DTOs
- `apps/cloudstation-orchestrator/cmd/cloudstation/auth_commands.go` - CLI auth commands

## Implementation Phases

### Phase 1: Foundation (CS-Backend CLI Auth Endpoint)

Create the CLI authentication endpoint in cs-backend that:
1. Accepts API key authentication
2. Returns infrastructure credentials for the user's services
3. Includes Vault, Nomad, NATS, and Registry configuration

### Phase 2: Core Implementation (Orchestrator Auth Package)

Build the orchestrator-side authentication system:
1. Auth client to communicate with cs-backend
2. Credential storage in `~/.cloudstation/`
3. CLI commands (`login`, `logout`, `whoami`, `link`)

### Phase 3: Integration & Polish

Integrate auth with existing commands:
1. Modify `up` command to use stored credentials
2. Add credential refresh logic
3. Fallback to environment variables for backward compatibility

## Step by Step Tasks

### 1. Create CLI Auth DTOs in CS-Backend

- Create `apps/cs-backend/api/internal/cliauth/dto.go`:
  ```go
  package cliauth

  // CLIAuthRequest is the request for CLI authentication
  type CLIAuthRequest struct {
      APIKey   string `json:"api_key" binding:"required"`
      ClientID string `json:"client_id" binding:"required"`
  }

  // CLICredentialsResponse contains all infrastructure credentials
  type CLICredentialsResponse struct {
      User     UserInfo           `json:"user"`
      Vault    VaultCredentials   `json:"vault"`
      Nomad    NomadCredentials   `json:"nomad"`
      NATS     NATSCredentials    `json:"nats"`
      Registry RegistryCredentials `json:"registry"`
      Services []ServiceInfo      `json:"services"`
  }

  type UserInfo struct {
      ID       int    `json:"id"`
      UUID     string `json:"uuid"`
      Email    string `json:"email"`
      FullName string `json:"full_name,omitempty"`
  }

  type VaultCredentials struct {
      Address     string `json:"address"`
      // Note: We don't expose AppRole credentials directly
      // Instead, we provide a short-lived token
      Token       string `json:"token"`
      ExpiresAt   int64  `json:"expires_at"`
  }

  type NomadCredentials struct {
      Address string `json:"address"`
      Token   string `json:"token"`
  }

  type NATSCredentials struct {
      Servers  string `json:"servers"`
      NKeySeed string `json:"nkey_seed"`
  }

  type RegistryCredentials struct {
      URL      string `json:"url"`
      Username string `json:"username"`
      Password string `json:"password"`
  }

  type ServiceInfo struct {
      ID            string `json:"id"`
      Name          string `json:"name"`
      ProjectID     string `json:"project_id"`
      EnvironmentID string `json:"environment_id"`
      ClusterID     string `json:"cluster_id"`
      ClusterDomain string `json:"cluster_domain"`
      SecretsPath   string `json:"secrets_path"`
  }
  ```

### 2. Create CLI Auth Service in CS-Backend

- Create `apps/cs-backend/api/internal/cliauth/service.go`:
  - Inject `authclient.Client` for API key validation
  - Inject `cluster.CredentialsResolver` for cluster credentials
  - Inject `repository.IntegrationRepository` for user's services
  - Method: `Authenticate(apiKey, clientID string) (*CLICredentialsResponse, error)`
    1. Validate API key via authclient
    2. Get user details
    3. Get user's services (integrations)
    4. For each unique cluster, resolve credentials
    5. Generate short-lived Vault token for CLI use
    6. Return aggregated credentials

### 3. Create CLI Auth Handler in CS-Backend

- Create `apps/cs-backend/api/internal/cliauth/handler.go`:
  ```go
  func (h *Handler) Authenticate(c *gin.Context) {
      var req CLIAuthRequest
      if err := c.ShouldBindJSON(&req); err != nil {
          c.JSON(400, gin.H{"error": "Invalid request"})
          return
      }

      creds, err := h.service.Authenticate(req.APIKey, req.ClientID)
      if err != nil {
          // Map error to appropriate status code
          c.JSON(statusCode, gin.H{"error": err.Error()})
          return
      }

      c.JSON(200, creds)
  }
  ```

### 4. Register CLI Auth Routes in CS-Backend

- Create `apps/cs-backend/api/internal/cliauth/routes.go`:
  ```go
  func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, cfg *config.Config, ...) {
      service := NewService(...)
      handler := NewHandler(service)

      // No JWT middleware - API key auth is in the request body
      router.POST("/cli/auth", handler.Authenticate)
      router.POST("/cli/refresh", handler.RefreshCredentials)
  }
  ```

- Update `apps/cs-backend/api/cmd/main.go` to register routes:
  ```go
  cliauth.RegisterRoutes(r.Group("/api"), db, cfg, authClient, clusterResolver)
  ```

### 5. Create Auth Types in Orchestrator

- Create `apps/cloudstation-orchestrator/pkg/auth/types.go`:
  ```go
  package auth

  import "time"

  // Credentials holds all authentication and infrastructure credentials
  type Credentials struct {
      // Auth metadata
      APIKey   string    `json:"api_key"`
      ClientID string    `json:"client_id"`

      // User info
      UserID   int       `json:"user_id"`
      UserUUID string    `json:"user_uuid"`
      Email    string    `json:"email"`

      // Infrastructure credentials
      Vault    VaultCreds   `json:"vault"`
      Nomad    NomadCreds   `json:"nomad"`
      NATS     NATSCreds    `json:"nats"`
      Registry RegistryCreds `json:"registry"`

      // Service links
      Services []ServiceLink `json:"services"`

      // Metadata
      CreatedAt time.Time `json:"created_at"`
      ExpiresAt time.Time `json:"expires_at"`
  }

  type VaultCreds struct {
      Address   string `json:"address"`
      Token     string `json:"token"`
      ExpiresAt int64  `json:"expires_at"`
  }

  type NomadCreds struct {
      Address string `json:"address"`
      Token   string `json:"token"`
  }

  type NATSCreds struct {
      Servers  string `json:"servers"`
      NKeySeed string `json:"nkey_seed"`
  }

  type RegistryCreds struct {
      URL      string `json:"url"`
      Username string `json:"username"`
      Password string `json:"password"`
  }

  type ServiceLink struct {
      ID            string `json:"id"`
      Name          string `json:"name"`
      SecretsPath   string `json:"secrets_path"`
      ClusterDomain string `json:"cluster_domain"`
  }
  ```

### 6. Create Auth Client in Orchestrator

- Create `apps/cloudstation-orchestrator/pkg/auth/client.go`:
  ```go
  package auth

  import (
      "bytes"
      "encoding/json"
      "fmt"
      "net/http"
      "time"
  )

  type Client struct {
      baseURL    string
      httpClient *http.Client
  }

  func NewClient(baseURL string) *Client {
      return &Client{
          baseURL: baseURL,
          httpClient: &http.Client{
              Timeout: 30 * time.Second,
          },
      }
  }

  func (c *Client) Authenticate(apiKey, clientID string) (*Credentials, error) {
      reqBody, _ := json.Marshal(map[string]string{
          "api_key":   apiKey,
          "client_id": clientID,
      })

      resp, err := c.httpClient.Post(
          c.baseURL+"/api/cli/auth",
          "application/json",
          bytes.NewBuffer(reqBody),
      )
      if err != nil {
          return nil, fmt.Errorf("auth request failed: %w", err)
      }
      defer resp.Body.Close()

      if resp.StatusCode != 200 {
          // Parse and return error
          return nil, fmt.Errorf("authentication failed: %d", resp.StatusCode)
      }

      var creds Credentials
      if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
          return nil, fmt.Errorf("failed to parse credentials: %w", err)
      }

      return &creds, nil
  }
  ```

### 7. Create Credential Storage in Orchestrator

- Create `apps/cloudstation-orchestrator/pkg/auth/storage.go`:
  ```go
  package auth

  import (
      "encoding/json"
      "fmt"
      "os"
      "path/filepath"
  )

  const (
      configDirName    = ".cloudstation"
      credentialsFile  = "credentials.json"
      serviceLinksFile = "links.json"
  )

  func GetConfigDir() (string, error) {
      home, err := os.UserHomeDir()
      if err != nil {
          return "", err
      }
      return filepath.Join(home, configDirName), nil
  }

  func EnsureConfigDir() error {
      dir, err := GetConfigDir()
      if err != nil {
          return err
      }
      return os.MkdirAll(dir, 0700)
  }

  func SaveCredentials(creds *Credentials) error {
      if err := EnsureConfigDir(); err != nil {
          return err
      }

      dir, _ := GetConfigDir()
      path := filepath.Join(dir, credentialsFile)

      data, err := json.MarshalIndent(creds, "", "  ")
      if err != nil {
          return err
      }

      return os.WriteFile(path, data, 0600)
  }

  func LoadCredentials() (*Credentials, error) {
      dir, err := GetConfigDir()
      if err != nil {
          return nil, err
      }

      path := filepath.Join(dir, credentialsFile)
      data, err := os.ReadFile(path)
      if err != nil {
          if os.IsNotExist(err) {
              return nil, fmt.Errorf("not logged in: run 'cs login' first")
          }
          return nil, err
      }

      var creds Credentials
      if err := json.Unmarshal(data, &creds); err != nil {
          return nil, err
      }

      return &creds, nil
  }

  func DeleteCredentials() error {
      dir, _ := GetConfigDir()
      path := filepath.Join(dir, credentialsFile)
      return os.Remove(path)
  }
  ```

### 8. Create Auth CLI Commands

- Create `apps/cloudstation-orchestrator/cmd/cloudstation/auth_commands.go`:
  ```go
  package main

  import (
      "fmt"
      "os"

      "github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
      "github.com/urfave/cli/v2"
  )

  func authCommand() *cli.Command {
      return &cli.Command{
          Name:  "auth",
          Usage: "Authentication commands",
          Subcommands: []*cli.Command{
              loginCommand(),
              logoutCommand(),
              whoamiCommand(),
          },
      }
  }

  func loginCommand() *cli.Command {
      return &cli.Command{
          Name:  "login",
          Usage: "Authenticate with CloudStation using an API key",
          Flags: []cli.Flag{
              &cli.StringFlag{
                  Name:    "api-key",
                  Usage:   "Your CloudStation API key",
                  EnvVars: []string{"CS_API_KEY"},
              },
              &cli.StringFlag{
                  Name:    "client-id",
                  Usage:   "Your API key client ID",
                  EnvVars: []string{"CS_CLIENT_ID"},
              },
              &cli.StringFlag{
                  Name:    "api-url",
                  Usage:   "CloudStation API URL",
                  Value:   "https://api.cloudstation.io",
                  EnvVars: []string{"CS_API_URL"},
              },
          },
          Action: func(c *cli.Context) error {
              apiKey := c.String("api-key")
              clientID := c.String("client-id")
              apiURL := c.String("api-url")

              // Prompt for credentials if not provided
              if apiKey == "" {
                  fmt.Print("Enter API Key: ")
                  fmt.Scanln(&apiKey)
              }
              if clientID == "" {
                  fmt.Print("Enter Client ID: ")
                  fmt.Scanln(&clientID)
              }

              // Authenticate
              client := auth.NewClient(apiURL)
              creds, err := client.Authenticate(apiKey, clientID)
              if err != nil {
                  return fmt.Errorf("authentication failed: %w", err)
              }

              // Store credentials
              if err := auth.SaveCredentials(creds); err != nil {
                  return fmt.Errorf("failed to save credentials: %w", err)
              }

              fmt.Printf("✓ Authenticated as %s\n", creds.Email)
              fmt.Printf("✓ Credentials saved to ~/.cloudstation/credentials\n")

              return nil
          },
      }
  }

  func logoutCommand() *cli.Command {
      return &cli.Command{
          Name:  "logout",
          Usage: "Remove stored credentials",
          Action: func(c *cli.Context) error {
              if err := auth.DeleteCredentials(); err != nil {
                  return fmt.Errorf("failed to logout: %w", err)
              }
              fmt.Println("✓ Logged out successfully")
              return nil
          },
      }
  }

  func whoamiCommand() *cli.Command {
      return &cli.Command{
          Name:  "whoami",
          Usage: "Display current authenticated user",
          Action: func(c *cli.Context) error {
              creds, err := auth.LoadCredentials()
              if err != nil {
                  return err
              }

              fmt.Printf("Email: %s\n", creds.Email)
              fmt.Printf("User ID: %d\n", creds.UserID)
              fmt.Printf("Services: %d linked\n", len(creds.Services))

              return nil
          },
      }
  }

  func linkCommand() *cli.Command {
      return &cli.Command{
          Name:  "link",
          Usage: "Link local project to a CloudStation service",
          Flags: []cli.Flag{
              &cli.StringFlag{
                  Name:  "service",
                  Usage: "Service ID to link (e.g., prj_integ_xxx)",
              },
          },
          Action: func(c *cli.Context) error {
              creds, err := auth.LoadCredentials()
              if err != nil {
                  return err
              }

              serviceID := c.String("service")
              if serviceID == "" {
                  // Interactive selection from available services
                  fmt.Println("Available services:")
                  for i, svc := range creds.Services {
                      fmt.Printf("  %d. %s (%s)\n", i+1, svc.Name, svc.ID)
                  }
                  // ... interactive selection
              }

              // Save link to local project
              if err := auth.SaveServiceLink(serviceID); err != nil {
                  return err
              }

              fmt.Printf("✓ Linked to %s\n", serviceID)
              return nil
          },
      }
  }
  ```

### 9. Register Auth Commands in Main

- Modify `apps/cloudstation-orchestrator/cmd/cloudstation/main.go`:
  ```go
  Commands: []*cli.Command{
      authCommand(),    // NEW: Add auth commands
      linkCommand(),    // NEW: Add link command
      initCommand(),
      buildCommand(),
      deployCommand(),
      upCommand(),
      runnerCommand(),
      dispatchCommand(),
  },
  ```

### 10. Modify Up Command to Use Stored Credentials

- Modify `apps/cloudstation-orchestrator/cmd/cloudstation/commands.go`:
  - In `upCommand()` action, before executing lifecycle:
    ```go
    // Try to load stored credentials
    creds, err := auth.LoadCredentials()
    if err == nil {
        // Use stored credentials
        os.Setenv("VAULT_ADDR", creds.Vault.Address)
        os.Setenv("VAULT_TOKEN", creds.Vault.Token)
        os.Setenv("NOMAD_ADDR", creds.Nomad.Address)
        os.Setenv("NOMAD_TOKEN", creds.Nomad.Token)
        os.Setenv("NATS_SERVERS", creds.NATS.Servers)
        os.Setenv("NATS_CLIENT_PRIVATE_KEY", creds.NATS.NKeySeed)

        logger.Info("using stored credentials", "user", creds.Email)
    } else {
        // Fall back to environment variables (backward compatibility)
        logger.Debug("no stored credentials, using environment")
    }
    ```

### 11. Validate Implementation

- Write unit tests for:
  - API key validation flow
  - Credential storage/retrieval
  - Auth client HTTP communication
- Write integration tests for:
  - Full login → link → deploy flow
  - Credential refresh
  - Error handling (invalid key, expired token, etc.)

## Testing Strategy

### Unit Tests

1. **Auth Client Tests** (`pkg/auth/client_test.go`):
   - Mock HTTP responses for successful auth
   - Test error handling for invalid credentials
   - Test timeout behavior

2. **Storage Tests** (`pkg/auth/storage_test.go`):
   - Test credential save/load cycle
   - Test file permissions (0600)
   - Test handling of missing credentials file

3. **CS-Backend Auth Service Tests**:
   - Test API key validation
   - Test cluster credential resolution
   - Test error cases (invalid key, missing service, etc.)

### Integration Tests

1. **End-to-End Auth Flow**:
   ```bash
   # Test login
   cs login --api-key test_key --client-id test_client

   # Test whoami
   cs whoami

   # Test link
   cs link --service prj_integ_test

   # Test deploy (with mock Nomad)
   cs up --app test-app
   ```

2. **Credential Refresh**:
   - Test automatic refresh when Vault token expires
   - Test re-authentication flow

### Edge Cases

- Invalid API key format
- Expired Vault token
- Network failures during auth
- Missing service link
- Multiple clusters with different credentials

## Acceptance Criteria

1. **Authentication Flow**:
   - [ ] `cs login --api-key xxx --client-id yyy` successfully authenticates
   - [ ] Credentials are stored securely in `~/.cloudstation/credentials` with 0600 permissions
   - [ ] `cs whoami` displays current user info
   - [ ] `cs logout` removes stored credentials

2. **Service Linking**:
   - [ ] `cs link --service xxx` associates local project with CloudStation service
   - [ ] Link is stored locally and persists across CLI invocations

3. **Deployment**:
   - [ ] `cs up` uses stored credentials automatically
   - [ ] Vault secrets are fetched using authenticated credentials
   - [ ] Deployment proceeds through Build → Registry → Deploy → Release pipeline
   - [ ] Logs are streamed via NATS connection

4. **Backward Compatibility**:
   - [ ] Environment variables still work if no stored credentials
   - [ ] Dispatch mode (Nomad-invoked) continues to work unchanged

5. **Security**:
   - [ ] API keys are never logged
   - [ ] Credentials file has restrictive permissions
   - [ ] Vault tokens are short-lived (< 1 hour)

## Validation Commands

Execute these commands to validate the task is complete:

- `cd apps/cs-backend && go build ./...` - Verify backend compiles
- `cd apps/cs-backend && go test ./api/internal/cliauth/...` - Run auth tests
- `cd apps/cloudstation-orchestrator && go build ./...` - Verify orchestrator compiles
- `cd apps/cloudstation-orchestrator && go test ./pkg/auth/...` - Run auth package tests
- `cs login --api-key test --client-id test` - Test login command (with test backend)
- `cs whoami` - Verify credentials stored
- `cs logout` - Verify cleanup

## Notes

### Dependencies

No new external Go dependencies are required. The implementation uses:
- Standard library `encoding/json`, `net/http`, `os`, `path/filepath`
- Existing `urfave/cli/v2` for CLI commands
- Existing `gin-gonic/gin` for HTTP handlers

### Security Considerations

1. **API Key Handling**:
   - Never log API keys
   - Use bcrypt comparison (already implemented)
   - Update `last_used` timestamp on each validation

2. **Vault Token Lifetime**:
   - Generate short-lived tokens (30-60 minutes) for CLI use
   - Implement automatic refresh before expiration

3. **Credential Storage**:
   - File permissions: 0600 (owner read/write only)
   - Consider future enhancement: OS keychain integration

4. **Network Security**:
   - All communication over HTTPS
   - Validate TLS certificates

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CLI AUTHENTICATION FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Developer Machine                          CloudStation Infrastructure     │
│  ┌───────────────────────┐                 ┌──────────────────────────────┐ │
│  │                       │                 │                              │ │
│  │  $ cs login           │   POST          │    CS-Backend API            │ │
│  │    --api-key sk_xxx   │ ───────────────▶│    /api/cli/auth             │ │
│  │                       │                 │                              │ │
│  │  ┌─────────────────┐  │   Credentials   │    ┌────────────────────┐    │ │
│  │  │ ~/.cloudstation/│  │ ◀───────────────│    │  Auth Service      │    │ │
│  │  │ credentials.json│  │                 │    │  (API Key Validate)│    │ │
│  │  └─────────────────┘  │                 │    └────────────────────┘    │ │
│  │                       │                 │              │               │ │
│  │  $ cs up              │                 │              ▼               │ │
│  │                       │                 │    ┌────────────────────┐    │ │
│  │  ┌─────────────────┐  │   Build         │    │  Cluster Resolver  │    │ │
│  │  │ Lifecycle       │──┼─────────────────┼───▶│  (Vault/Nomad/NATS)│    │ │
│  │  │ Executor        │  │   Deploy        │    └────────────────────┘    │ │
│  │  └─────────────────┘  │                 │              │               │ │
│  │          │            │                 │              ▼               │ │
│  │          │            │                 │    ┌────────────────────┐    │ │
│  │          │            │   Fetch Secrets │    │  Vault (Secrets)   │    │ │
│  │          ├────────────┼─────────────────┼───▶│                    │    │ │
│  │          │            │                 │    └────────────────────┘    │ │
│  │          │            │                 │              │               │ │
│  │          │            │   Deploy Job    │              ▼               │ │
│  │          └────────────┼─────────────────┼───▶┌────────────────────┐    │ │
│  │                       │                 │    │  Nomad (Deploy)    │    │ │
│  │                       │                 │    └────────────────────┘    │ │
│  │                       │                 │                              │ │
│  └───────────────────────┘                 └──────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Related Documentation

- `apps/cloudstation-orchestrator/docs/ARCHITECTURE.md` - Overall architecture
- `apps/cloudstation-orchestrator/docs/PLUGINS.md` - Plugin development
- `apps/cloudstation-orchestrator/docs/SECRETS.md` - Secret provider docs
- `apps/cs-backend/shared/authclient/` - Auth client patterns
