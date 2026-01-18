# Plan: CLI Template and Image Deployment Commands

## Task Description

Implement CLI commands to enable LLM-friendly deployment workflows:
1. `cs template list/deploy` - Browse and deploy from template store (one-click managed deployments)
2. `cs image deploy` - Deploy any Docker image directly with manual configuration
3. `cs template create` - Admin-only command to create new templates (hidden from regular users)

This enables Claude Code to deploy applications by either using pre-configured templates (one-click) or deploying raw Docker images when templates don't exist.

## Objective

When complete, the CLI will support:
- Browsing the CloudStation template store
- One-click deployment from templates (CS manages all config)
- Direct Docker image deployment with user-specified ports/vars/volumes
- Admin-only template creation (hidden command)
- JSON output for LLM parsing

## Problem Statement

Currently, the cloudstation-orchestrator CLI can only:
- Build and deploy from local code (`cs up`)
- Manage authentication (`cs login/logout`)

It CANNOT:
- Browse or deploy from the template store
- Deploy arbitrary Docker images directly
- Create new templates

The cs-backend already has all the APIs needed - the CLI just needs client code to call them.

## Solution Approach

1. **Backend Change**: Add template routes to API key allowlist (currently blocked)
2. **New Package**: Create `pkg/templates/` for template API client
3. **New Commands**: Add `template_commands.go` and `image_commands.go`
4. **Admin Restriction**: Use Hidden flag + backend enforcement for template create
5. **JSON Output**: Add `--json` flag to relevant commands

## Relevant Files

### Existing Files to Modify

- `cmd/cloudstation/main.go` (Lines 49-65)
  - Register new commands: templateCommand(), imageCommand()

- `pkg/auth/types.go` (Lines 5-28)
  - Add `IsSuperAdmin bool` field to Credentials struct

- `pkg/auth/client.go` (Lines 43-318)
  - Add method to fetch user role/admin status during auth

### New Files to Create

#### CLI Commands
- `cmd/cloudstation/template_commands.go`
  - templateCommand() with subcommands: list, info, deploy, create
  - Admin check for create command

- `cmd/cloudstation/image_commands.go`
  - imageCommand() with deploy subcommand

#### Template Client Package
- `pkg/templates/client.go`
  - TemplateClient struct embedding httpclient.BaseClient
  - List(), Get(), Deploy(), GetTags(), Create() methods

- `pkg/templates/types.go`
  - Template, TemplateDefinition, ServiceDefinition structs
  - Request/Response DTOs matching cs-backend

### Backend Files to Modify (cs-backend)

- `apps/cs-backend/api/internal/auth/apikey_config.go` (Lines 10-31)
  - Add template routes to APIKeyAllowedRoutes map

## Implementation Phases

### Phase 1: Foundation
- Add template routes to cs-backend API key allowlist
- Create pkg/templates/ package with types and client
- Add IsSuperAdmin to Credentials struct

### Phase 2: Core Implementation
- Implement template commands (list, info, deploy)
- Implement image deploy command
- Add --json output flag support

### Phase 3: Admin Features & Polish
- Implement hidden template create command
- Add admin check using IsSuperAdmin
- Add comprehensive error handling
- Test all workflows

## Step by Step Tasks

### 1. Add Template Routes to API Key Allowlist (cs-backend)

- Edit `apps/cs-backend/api/internal/auth/apikey_config.go`
- Add to GET allowlist:
  ```go
  "/templates",
  "/templates/:id",
  "/templates/tags",
  "/templates/deployed/:project_id",
  ```
- Add to POST allowlist:
  ```go
  "/templates/deploy",
  ```
- Add new PUT allowlist:
  ```go
  "PUT": {
      "/templates/deployed/:deployment_instance_id/stop",
      "/templates/deployed/:deployment_instance_id/start",
      "/templates/deployed/:deployment_instance_id/redeploy",
  },
  ```

### 2. Create Template Types Package

- Create `pkg/templates/types.go` with:
  ```go
  package templates

  import "time"

  // Template represents a CloudStation template
  type Template struct {
      ID           string                 `json:"id"`
      Name         string                 `json:"name"`
      Description  string                 `json:"description"`
      Image        string                 `json:"image"`
      Visibility   string                 `json:"visibility"`
      Status       string                 `json:"status"`
      Tags         []string               `json:"tags"`
      Author       *string                `json:"author,omitempty"`
      Definition   map[string]interface{} `json:"definition"`
      Deployments  float64                `json:"deployments"`
      Slug         *string                `json:"slug,omitempty"`
      TotalApps    float64                `json:"totalApps"`
      CreatedAt    time.Time              `json:"createdAt"`
  }

  // ListTemplatesParams for filtering templates
  type ListTemplatesParams struct {
      Search   string
      Tags     string // comma-separated
      Sort     string // created_DESC, name_ASC, deployments_DESC
      Page     int
      PageSize int
  }

  // ListTemplatesResponse paginated response
  type ListTemplatesResponse struct {
      Data       []Template `json:"data"`
      Total      int64      `json:"total"`
      Page       int        `json:"page"`
      PageSize   int        `json:"pageSize"`
      TotalPages int        `json:"totalPages"`
  }

  // DeployTemplateRequest for POST /templates/deploy
  type DeployTemplateRequest struct {
      TemplateID  string                 `json:"templateId"`
      ProjectID   string                 `json:"projectId"`
      Environment string                 `json:"environment,omitempty"`
      Variables   []DeployVariableInput  `json:"variables,omitempty"`
  }

  // DeployVariableInput for overriding template variables
  type DeployVariableInput struct {
      TemplateName string `json:"templateName,omitempty"`
      Key          string `json:"key"`
      Value        string `json:"value"`
  }

  // DeployTemplateResponse from deployment
  type DeployTemplateResponse struct {
      ID                   string   `json:"id"`
      TemplateID           string   `json:"templateId"`
      ProjectID            string   `json:"projectId"`
      ServiceIDs           []string `json:"serviceIds"`
      DeploymentInstanceID string   `json:"deploymentInstanceId"`
      Message              string   `json:"message,omitempty"`
  }

  // TagsResponse for GET /templates/tags
  type TagsResponse struct {
      Tags []string `json:"tags"`
  }
  ```

### 3. Create Template Client

- Create `pkg/templates/client.go`:
  ```go
  package templates

  import (
      "fmt"
      "net/url"
      "strconv"
      "time"

      "github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
  )

  type Client struct {
      *httpclient.BaseClient
  }

  func NewClient(baseURL, apiKey, clientID string) *Client {
      client := &Client{
          BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
      }
      client.SetHeader("x-api-key", apiKey)
      client.SetHeader("x-client-id", clientID)
      return client
  }

  func (c *Client) List(params ListTemplatesParams) (*ListTemplatesResponse, error) {
      query := url.Values{}
      if params.Search != "" {
          query.Set("search", params.Search)
      }
      if params.Tags != "" {
          query.Set("tags", params.Tags)
      }
      if params.Sort != "" {
          query.Set("sort", params.Sort)
      }
      if params.Page > 0 {
          query.Set("page", strconv.Itoa(params.Page))
      }
      if params.PageSize > 0 {
          query.Set("pageSize", strconv.Itoa(params.PageSize))
      }

      path := "/templates"
      if len(query) > 0 {
          path = path + "?" + query.Encode()
      }

      var resp ListTemplatesResponse
      if err := c.DoJSON("GET", path, nil, &resp); err != nil {
          return nil, fmt.Errorf("list templates failed: %w", err)
      }
      return &resp, nil
  }

  func (c *Client) Get(idOrSlug string) (*Template, error) {
      path := fmt.Sprintf("/templates/%s", idOrSlug)
      var resp Template
      if err := c.DoJSON("GET", path, nil, &resp); err != nil {
          return nil, fmt.Errorf("get template failed: %w", err)
      }
      return &resp, nil
  }

  func (c *Client) GetTags() ([]string, error) {
      var resp TagsResponse
      if err := c.DoJSON("GET", "/templates/tags", nil, &resp); err != nil {
          return nil, fmt.Errorf("get tags failed: %w", err)
      }
      return resp.Tags, nil
  }

  func (c *Client) Deploy(req DeployTemplateRequest) (*DeployTemplateResponse, error) {
      var resp DeployTemplateResponse
      if err := c.DoJSON("POST", "/templates/deploy", req, &resp); err != nil {
          return nil, fmt.Errorf("deploy template failed: %w", err)
      }
      return &resp, nil
  }
  ```

### 4. Create Template Commands

- Create `cmd/cloudstation/template_commands.go`:
  ```go
  package main

  import (
      "encoding/json"
      "fmt"
      "strings"

      "github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
      "github.com/thecloudstation/cloudstation-orchestrator/pkg/templates"
      "github.com/urfave/cli/v2"
  )

  func templateCommand() *cli.Command {
      return &cli.Command{
          Name:  "template",
          Usage: "Browse and deploy from CloudStation template store",
          Subcommands: []*cli.Command{
              templateListCommand(),
              templateInfoCommand(),
              templateDeployCommand(),
              templateTagsCommand(),
              templateCreateCommand(), // Hidden, admin only
          },
      }
  }

  func templateListCommand() *cli.Command {
      return &cli.Command{
          Name:  "list",
          Usage: "List available templates",
          Flags: []cli.Flag{
              &cli.StringFlag{Name: "search", Usage: "Search term"},
              &cli.StringFlag{Name: "tags", Usage: "Filter by tags (comma-separated)"},
              &cli.StringFlag{Name: "sort", Usage: "Sort: created_DESC, name_ASC, deployments_DESC", Value: "deployments_DESC"},
              &cli.IntFlag{Name: "page", Usage: "Page number", Value: 1},
              &cli.IntFlag{Name: "limit", Usage: "Results per page", Value: 20},
              &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Action: func(c *cli.Context) error {
              creds, err := auth.LoadCredentials()
              if err != nil {
                  return fmt.Errorf("not logged in: run 'cs login' first")
              }

              client := templates.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)

              resp, err := client.List(templates.ListTemplatesParams{
                  Search:   c.String("search"),
                  Tags:     c.String("tags"),
                  Sort:     c.String("sort"),
                  Page:     c.Int("page"),
                  PageSize: c.Int("limit"),
              })
              if err != nil {
                  return err
              }

              if c.Bool("json") {
                  data, _ := json.MarshalIndent(resp, "", "  ")
                  fmt.Println(string(data))
                  return nil
              }

              // Human-readable output
              fmt.Printf("Templates (%d total, page %d/%d)\n", resp.Total, resp.Page, resp.TotalPages)
              fmt.Println(strings.Repeat("-", 60))
              for _, t := range resp.Data {
                  tags := ""
                  if len(t.Tags) > 0 {
                      tags = fmt.Sprintf(" [%s]", strings.Join(t.Tags, ", "))
                  }
                  fmt.Printf("  %s - %s%s\n", t.ID, t.Name, tags)
                  if t.Description != "" {
                      desc := t.Description
                      if len(desc) > 60 {
                          desc = desc[:57] + "..."
                      }
                      fmt.Printf("    %s\n", desc)
                  }
              }
              return nil
          },
      }
  }

  func templateInfoCommand() *cli.Command {
      return &cli.Command{
          Name:      "info",
          Usage:     "Get template details",
          ArgsUsage: "<template-id-or-slug>",
          Flags: []cli.Flag{
              &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Action: func(c *cli.Context) error {
              if c.NArg() < 1 {
                  return fmt.Errorf("template ID or slug required")
              }

              creds, err := auth.LoadCredentials()
              if err != nil {
                  return fmt.Errorf("not logged in: run 'cs login' first")
              }

              client := templates.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)
              template, err := client.Get(c.Args().First())
              if err != nil {
                  return err
              }

              if c.Bool("json") {
                  data, _ := json.MarshalIndent(template, "", "  ")
                  fmt.Println(string(data))
                  return nil
              }

              fmt.Printf("Template: %s\n", template.Name)
              fmt.Printf("ID: %s\n", template.ID)
              if template.Description != "" {
                  fmt.Printf("Description: %s\n", template.Description)
              }
              if len(template.Tags) > 0 {
                  fmt.Printf("Tags: %s\n", strings.Join(template.Tags, ", "))
              }
              fmt.Printf("Deployments: %.0f\n", template.Deployments)
              fmt.Printf("Services: %.0f\n", template.TotalApps)
              return nil
          },
      }
  }

  func templateDeployCommand() *cli.Command {
      return &cli.Command{
          Name:      "deploy",
          Usage:     "Deploy a template (one-click deployment)",
          ArgsUsage: "<template-id-or-slug>",
          Flags: []cli.Flag{
              &cli.StringFlag{Name: "project", Usage: "Project ID (uses linked project if not specified)"},
              &cli.StringFlag{Name: "env", Usage: "Environment ID (optional)"},
              &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Action: func(c *cli.Context) error {
              if c.NArg() < 1 {
                  return fmt.Errorf("template ID or slug required")
              }

              creds, err := auth.LoadCredentials()
              if err != nil {
                  return fmt.Errorf("not logged in: run 'cs login' first")
              }

              // Get project ID from flag or linked service
              projectID := c.String("project")
              if projectID == "" {
                  serviceID, err := auth.LoadServiceLink()
                  if err != nil {
                      return fmt.Errorf("no project specified: use --project or run 'cs link' first")
                  }
                  // Service ID is the project context
                  projectID = serviceID
              }

              client := templates.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)

              resp, err := client.Deploy(templates.DeployTemplateRequest{
                  TemplateID:  c.Args().First(),
                  ProjectID:   projectID,
                  Environment: c.String("env"),
              })
              if err != nil {
                  return err
              }

              if c.Bool("json") {
                  data, _ := json.MarshalIndent(resp, "", "  ")
                  fmt.Println(string(data))
                  return nil
              }

              fmt.Println("Deployment started!")
              fmt.Printf("  Deployment Instance: %s\n", resp.DeploymentInstanceID)
              fmt.Printf("  Services: %d\n", len(resp.ServiceIDs))
              for _, svc := range resp.ServiceIDs {
                  fmt.Printf("    - %s\n", svc)
              }
              return nil
          },
      }
  }

  func templateTagsCommand() *cli.Command {
      return &cli.Command{
          Name:  "tags",
          Usage: "List available template tags/categories",
          Flags: []cli.Flag{
              &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Action: func(c *cli.Context) error {
              creds, err := auth.LoadCredentials()
              if err != nil {
                  return fmt.Errorf("not logged in: run 'cs login' first")
              }

              client := templates.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)
              tags, err := client.GetTags()
              if err != nil {
                  return err
              }

              if c.Bool("json") {
                  data, _ := json.MarshalIndent(tags, "", "  ")
                  fmt.Println(string(data))
                  return nil
              }

              fmt.Println("Available tags:")
              for _, tag := range tags {
                  fmt.Printf("  - %s\n", tag)
              }
              return nil
          },
      }
  }

  func templateCreateCommand() *cli.Command {
      return &cli.Command{
          Name:   "create",
          Usage:  "Create a new template (admin only)",
          Hidden: true, // Hidden from regular users
          Flags: []cli.Flag{
              &cli.StringFlag{Name: "name", Usage: "Template name", Required: true},
              &cli.StringFlag{Name: "definition", Usage: "Path to JSON definition file", Required: true},
              &cli.StringFlag{Name: "visibility", Usage: "Visibility: private, teams, public", Value: "private"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Before: requireAdmin,
          Action: func(c *cli.Context) error {
              // Implementation for admin template creation
              return fmt.Errorf("template creation not yet implemented")
          },
      }
  }

  // requireAdmin checks if current user is an admin
  func requireAdmin(c *cli.Context) error {
      creds, err := auth.LoadCredentials()
      if err != nil {
          return fmt.Errorf("not logged in: run 'cs login' first")
      }

      // Check IsSuperAdmin field (requires backend to return this)
      if !creds.IsSuperAdmin {
          return fmt.Errorf("this command requires admin privileges")
      }
      return nil
  }
  ```

### 5. Create Image Deploy Command

- Create `cmd/cloudstation/image_commands.go`:
  ```go
  package main

  import (
      "encoding/json"
      "fmt"
      "strconv"
      "strings"
      "time"

      "github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
      "github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
      "github.com/urfave/cli/v2"
  )

  func imageCommand() *cli.Command {
      return &cli.Command{
          Name:  "image",
          Usage: "Deploy Docker images directly",
          Subcommands: []*cli.Command{
              imageDeployCommand(),
          },
      }
  }

  func imageDeployCommand() *cli.Command {
      return &cli.Command{
          Name:      "deploy",
          Usage:     "Deploy a Docker image",
          ArgsUsage: "<image:tag>",
          Flags: []cli.Flag{
              &cli.StringFlag{Name: "name", Usage: "Service name", Required: true},
              &cli.StringFlag{Name: "project", Usage: "Project ID (uses linked project if not specified)"},
              &cli.StringSliceFlag{Name: "port", Usage: "Port mapping: <port>:<type> (e.g., 8080:http, 5432:tcp)"},
              &cli.StringSliceFlag{Name: "env", Usage: "Environment variable: KEY=value"},
              &cli.StringFlag{Name: "volume", Usage: "Volume: <path>:<size> (e.g., /data:10Gi)"},
              &cli.IntFlag{Name: "ram", Usage: "Memory in MB", Value: 512},
              &cli.Float64Flag{Name: "cpu", Usage: "CPU cores", Value: 0.25},
              &cli.IntFlag{Name: "replicas", Usage: "Replica count", Value: 1},
              &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
              &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
          },
          Action: func(c *cli.Context) error {
              if c.NArg() < 1 {
                  return fmt.Errorf("image required (e.g., nginx:latest, redis:7-alpine)")
              }

              creds, err := auth.LoadCredentials()
              if err != nil {
                  return fmt.Errorf("not logged in: run 'cs login' first")
              }

              // Parse image:tag
              imageArg := c.Args().First()
              imageName, imageTag := parseImageTag(imageArg)

              // Get project ID
              projectID := c.String("project")
              if projectID == "" {
                  serviceID, err := auth.LoadServiceLink()
                  if err != nil {
                      return fmt.Errorf("no project specified: use --project or run 'cs link' first")
                  }
                  projectID = serviceID
              }

              // Parse ports
              var networks []map[string]interface{}
              for _, p := range c.StringSlice("port") {
                  port, portType, err := parsePort(p)
                  if err != nil {
                      return fmt.Errorf("invalid port %s: %w", p, err)
                  }
                  networks = append(networks, map[string]interface{}{
                      "portNumber": port,
                      "portType":   portType,
                      "public":     portType == "http",
                  })
              }

              // Parse env vars
              var variables []map[string]string
              for _, e := range c.StringSlice("env") {
                  key, value, err := parseEnvVar(e)
                  if err != nil {
                      return fmt.Errorf("invalid env %s: %w", e, err)
                  }
                  variables = append(variables, map[string]string{
                      "key":   key,
                      "value": value,
                  })
              }

              // Build request
              reqBody := map[string]interface{}{
                  "name":          c.String("name"),
                  "image_url":     imageName,
                  "tag":           imageTag,
                  "projectId":     projectID,
                  "replica_count": float64(c.Int("replicas")),
                  "ram":           float64(c.Int("ram")),
                  "cpu":           c.Float64("cpu"),
              }
              if len(networks) > 0 {
                  reqBody["networks"] = networks
              }
              if len(variables) > 0 {
                  reqBody["variables"] = variables
              }

              // Create image service
              client := httpclient.NewBaseClient(c.String("api-url"), 30*time.Second)
              client.SetHeader("x-api-key", creds.APIKey)
              client.SetHeader("x-client-id", creds.ClientID)

              var createResp struct {
                  ID      string `json:"id"`
                  Name    string `json:"name"`
                  Message string `json:"message"`
              }
              if err := client.DoJSON("POST", "/services/images", reqBody, &createResp); err != nil {
                  return fmt.Errorf("create image service failed: %w", err)
              }

              // Deploy the service
              var deployResp struct {
                  ID           string `json:"id"`
                  DeploymentID string `json:"deploymentId"`
                  Status       string `json:"status"`
                  Message      string `json:"message"`
              }
              deployPath := fmt.Sprintf("/services/%s/deploy", createResp.ID)
              if err := client.DoJSON("POST", deployPath, nil, &deployResp); err != nil {
                  return fmt.Errorf("deploy service failed: %w", err)
              }

              if c.Bool("json") {
                  result := map[string]interface{}{
                      "serviceId":    createResp.ID,
                      "serviceName":  createResp.Name,
                      "deploymentId": deployResp.DeploymentID,
                      "status":       deployResp.Status,
                  }
                  data, _ := json.MarshalIndent(result, "", "  ")
                  fmt.Println(string(data))
                  return nil
              }

              fmt.Println("Image deployment started!")
              fmt.Printf("  Service ID: %s\n", createResp.ID)
              fmt.Printf("  Service Name: %s\n", createResp.Name)
              fmt.Printf("  Deployment ID: %s\n", deployResp.DeploymentID)
              fmt.Printf("  Status: %s\n", deployResp.Status)
              return nil
          },
      }
  }

  // parseImageTag splits image:tag into components
  func parseImageTag(image string) (string, string) {
      parts := strings.SplitN(image, ":", 2)
      if len(parts) == 2 {
          return parts[0], parts[1]
      }
      return image, "latest"
  }

  // parsePort parses "8080:http" into port number and type
  func parsePort(p string) (int, string, error) {
      parts := strings.SplitN(p, ":", 2)
      port, err := strconv.Atoi(parts[0])
      if err != nil {
          return 0, "", fmt.Errorf("invalid port number")
      }
      portType := "tcp"
      if len(parts) == 2 {
          portType = parts[1]
      }
      return port, portType, nil
  }

  // parseEnvVar parses "KEY=value" into key and value
  func parseEnvVar(e string) (string, string, error) {
      parts := strings.SplitN(e, "=", 2)
      if len(parts) != 2 {
          return "", "", fmt.Errorf("must be KEY=value format")
      }
      return parts[0], parts[1], nil
  }
  ```

### 6. Update main.go to Register Commands

- Edit `cmd/cloudstation/main.go` (Lines 49-65)
- Add new commands to the Commands slice:
  ```go
  Commands: []*cli.Command{
      // Lifecycle commands
      initCommand(),
      buildCommand(),
      deployCommand(),
      upCommand(),

      // Template & Image commands (NEW)
      templateCommand(),
      imageCommand(),

      // Authentication commands
      loginCommand(),
      logoutCommand(),
      whoamiCommand(),
      linkCommand(),

      // System commands
      runnerCommand(),
      dispatchCommand(),
  },
  ```

### 7. Add IsSuperAdmin to Credentials Struct

- Edit `pkg/auth/types.go`:
  ```go
  type Credentials struct {
      // Auth metadata
      APIKey   string `json:"api_key"`
      ClientID string `json:"client_id"`

      // User info
      UserID       int    `json:"user_id"`
      UserUUID     string `json:"user_uuid"`
      Email        string `json:"email"`
      IsSuperAdmin bool   `json:"is_super_admin"` // NEW

      // ... rest of struct
  }
  ```

- Edit `pkg/auth/client.go` to map IsSuperAdmin from backend response in Authenticate():
  ```go
  creds.IsSuperAdmin = backendResp.User.IsSuperAdmin
  ```

### 8. Test All Commands

- Run `go build -o cs ./cmd/cloudstation/`
- Test: `./cs template list`
- Test: `./cs template list --json`
- Test: `./cs template info <id>`
- Test: `./cs template deploy <id> --project <project>`
- Test: `./cs image deploy redis:7 --name my-redis --port 6379:tcp`
- Test admin restriction: `./cs template create --name test --definition test.json` (should fail for non-admins)

## Testing Strategy

### Unit Tests
- Test parseImageTag() with various inputs: "nginx", "nginx:latest", "ghcr.io/org/image:v1.0"
- Test parsePort() with valid and invalid inputs
- Test parseEnvVar() with valid and invalid inputs
- Test Template client methods with mock HTTP responses

### Integration Tests
- Test template list against real API (requires credentials)
- Test template deploy creates actual services
- Test image deploy creates and deploys service
- Test admin check blocks non-admin users

### Edge Cases
- Empty template list response
- Template not found (404)
- Invalid project ID
- Network errors
- Rate limiting

## Acceptance Criteria

- [ ] `cs template list` shows templates from store
- [ ] `cs template list --json` outputs valid JSON
- [ ] `cs template list --tags database` filters correctly
- [ ] `cs template info <id>` shows template details
- [ ] `cs template deploy <id>` deploys to linked project
- [ ] `cs template deploy <id> --project <id>` deploys to specified project
- [ ] `cs template create` is hidden from `cs template --help`
- [ ] `cs template create` returns error for non-admin users
- [ ] `cs image deploy nginx:latest --name web --port 80:http` creates and deploys service
- [ ] `cs image deploy` requires `--name` flag
- [ ] `cs image deploy` requires project context (link or --project)
- [ ] All commands support `--json` flag for LLM parsing
- [ ] Error messages are clear and actionable

## Validation Commands

Execute these commands to validate the task is complete:

- `go build -o cs ./cmd/cloudstation/` - Verify code compiles
- `./cs template --help` - Verify template subcommands shown
- `./cs image --help` - Verify image subcommands shown
- `./cs template list --json | jq .` - Verify JSON output is valid
- `./cs template create --help 2>&1 | grep -q "admin"` - Verify admin message (hidden command)

## Notes

### Dependencies
- No new Go dependencies required
- Uses existing httpclient, auth packages

### Backend Changes Required
1. Add template routes to API key allowlist in cs-backend
2. Ensure `/api/cli/auth` returns `is_super_admin` field

### Security Considerations
- Template create command is Hidden AND has Before hook for admin check
- Backend should also enforce admin-only access to template creation endpoints
- Project access is validated server-side

### Future Enhancements
- `cs template create --from-service <id>` to reverse-engineer templates
- `cs image save-as-template` to convert deployed service to template
- Log streaming for deployments (similar to `cs up --remote`)
