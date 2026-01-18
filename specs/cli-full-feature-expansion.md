# Plan: CLI Full Feature Expansion - Billing, OAuth, TUI & AI Deployment

## Task Description

Comprehensive expansion of the CloudStation CLI (`cs`) to support:
1. Fix template deploy authentication for API key users
2. Billing and subscription management (subscribe, payment links, invoices)
3. Google OAuth login (browser-based authentication)
4. Terminal User Interface (TUI) for interactive experience
5. AI-driven automatic deployment in Orbit sandbox
6. Full feature parity with web dashboard from CLI

## Objective

When complete, users will be able to:
- Login via Google OAuth from CLI (browser opens, returns to terminal)
- Subscribe to CloudStation plans and manage billing entirely from CLI
- AI agents in Orbit sandbox can deploy applications automatically
- Non-subscribed users get payment links generated for them
- Rich interactive TUI for service selection, deployment progress, etc.

## Problem Statement

### Current Limitations:
1. **Template Deploy Broken for API Keys**: The `/templates/deploy` endpoint fails because API key auth doesn't carry team context, causing "project not found" errors
2. **No Billing in CLI**: Users must visit web dashboard to subscribe, add payment methods, or view invoices
3. **No OAuth Login**: Only API key + client ID login supported - no Google/GitHub SSO
4. **Primitive UI**: Basic `fmt.Scanln` prompts with no TUI - poor UX for interactive workflows
5. **AI Can't Deploy Automatically**: Orbit sandbox AI agents need CLI access but lack subscription/billing context

### Root Cause Analysis (from scout reports):

**Template Deploy Auth Issue:**
```go
// In templates/service.go - DeployTemplate checks team first:
if params.Team != "" {
    if err := s.checkTeamAuthorization(params.UserID, params.Team); err != nil {
        return nil, err
    }
}
// BUT: API key middleware only sets user_id, NOT team context
// Result: params.Team is empty, so team check is skipped
// Then FindProjectByIDWithAuth fails because it queries by userId for team projects
```

**Solution**: Auto-resolve team from project's teamId before access check, OR require CLI to pass team parameter.

## Solution Approach

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                     CloudStation CLI v2                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   Auth      │  │   Billing   │  │  Templates  │  │    TUI     │ │
│  │ ─────────── │  │ ─────────── │  │ ─────────── │  │ ────────── │ │
│  │ • API Key   │  │ • Subscribe │  │ • List      │  │ • Menus    │ │
│  │ • Google    │  │ • Plans     │  │ • Deploy    │  │ • Progress │ │
│  │ • GitHub    │  │ • Invoices  │  │ • Info      │  │ • Forms    │ │
│  │ • Refresh   │  │ • Pay Links │  │ • Create    │  │ • Spinners │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │
│                                                                     │
│  Backend APIs: cs-backend (billing, auth, templates)                │
│                admin-api (billing management, stripe, lago)         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Relevant Files

### Backend Files to Modify (cs-backend)

- `apps/cs-backend/api/internal/auth/apikey_config.go`
  - Add billing routes to API key allowlist

- `apps/cs-backend/api/internal/templates/service.go` (Lines 406-434)
  - Fix DeployTemplate to auto-resolve team from project

- `apps/cs-backend/api/internal/cliauth/handler.go`
  - Add OAuth token exchange endpoint for CLI

- `apps/cs-backend/shared/repository/service_repository.go` (Lines 383-403)
  - Fix FindProjectByIDWithAuth to include team membership check

### CLI Files to Modify (cloudstation-orchestrator)

- `apps/cloudstation-orchestrator/go.mod`
  - Add: bubbletea, lipgloss, promptui, oauth2

- `apps/cloudstation-orchestrator/pkg/auth/types.go`
  - Add OAuth token fields to Credentials struct

- `apps/cloudstation-orchestrator/pkg/auth/client.go`
  - Add OAuth token exchange method

- `apps/cloudstation-orchestrator/cmd/cloudstation/main.go`
  - Register new billing commands

- `apps/cloudstation-orchestrator/cmd/cloudstation/auth_commands.go`
  - Add `--google` flag, implement OAuth PKCE flow

- `apps/cloudstation-orchestrator/cmd/cloudstation/template_commands.go`
  - Pass team parameter to deploy endpoint

### New Files to Create

#### CLI Commands
- `cmd/cloudstation/billing_commands.go` - Billing/subscription management
- `cmd/cloudstation/oauth.go` - OAuth PKCE flow implementation

#### CLI Packages
- `pkg/billing/client.go` - Billing API client
- `pkg/billing/types.go` - Billing types
- `pkg/oauth/google.go` - Google OAuth handler
- `pkg/oauth/callback.go` - Local callback server

#### TUI Components
- `internal/tui/select.go` - Interactive selection menus
- `internal/tui/spinner.go` - Progress spinners
- `internal/tui/forms.go` - Input forms
- `internal/tui/theme.go` - Consistent styling

## Implementation Phases

### Phase 1: Fix Template Deploy Auth (Backend)
- Fix the immediate blocker for AI deployments
- Auto-resolve team context from project
- Add team membership verification to repository

### Phase 2: Add Billing to CLI
- Add billing routes to API key allowlist
- Create billing client package
- Implement billing commands (plan, subscribe, invoices)
- Generate payment links for non-subscribers

### Phase 3: Google OAuth Login
- Create OAuth PKCE flow with local callback
- Add backend endpoint for OAuth token exchange
- Extend credentials to store OAuth tokens
- Implement token refresh

### Phase 4: TUI Enhancement
- Add bubbletea/lipgloss dependencies
- Implement interactive service selection
- Add progress spinners for deployments
- Create subscription plan selector

### Phase 5: AI Deployment Integration
- Test full flow in Orbit sandbox
- Add `--ai-mode` flag for non-interactive
- Implement automatic subscription check
- Generate payment links when needed

## Step by Step Tasks

### 1. Fix Template Deploy Auth (cs-backend)

**File: `apps/cs-backend/api/internal/templates/service.go`**

- Add auto-resolve team from project before access check:
```go
// In DeployTemplate, before line 429:
func (s *service) DeployTemplate(params DeployTemplateParams) (*DeployTemplateResponse, error) {
    // ... existing validation ...

    // Auto-resolve team if not provided (for API key auth)
    if params.Team == "" {
        // Look up project to get its team
        var project model.Project
        if err := s.db.Where("id = ?", req.ProjectID).First(&project).Error; err == nil {
            if project.TeamID != nil {
                // Get team slug from team ID
                var team model.Team
                if err := s.db.Where("id = ?", project.TeamID).First(&team).Error; err == nil {
                    params.Team = team.Slug
                }
            }
        }
    }

    // Now proceed with team authorization check
    if params.Team != "" {
        if err := s.checkTeamAuthorization(params.UserID, params.Team); err != nil {
            return nil, err
        }
    }
    // ... rest of function
}
```

**File: `apps/cs-backend/shared/repository/service_repository.go`**

- Add team membership verification to `FindProjectByIDWithAuth`:
```go
func (r *serviceRepository) FindProjectByIDWithAuth(id string, teamSlug string, userID int) (*model.Project, error) {
    var project model.Project
    query := r.db

    if teamSlug != "" {
        // Team project: verify user is member of team
        query = query.
            Joins("JOIN teams ON teams.id = projects.\"teamId\"").
            Joins("JOIN team_members ON team_members.team_id = teams.id").
            Where("projects.id = ? AND (teams.slug = ? OR teams.id = ?) AND team_members.user_id = ?",
                  id, teamSlug, teamSlug, userID)
    } else {
        // Personal project: filter by user ID
        query = query.Where("id = ? AND \"userId\" = ?", id, userID)
    }

    if err := query.First(&project).Error; err != nil {
        return nil, err
    }
    return &project, nil
}
```

### 2. Add Billing Routes to API Key Allowlist

**File: `apps/cs-backend/api/internal/auth/apikey_config.go`**

- Add billing endpoints:
```go
var APIKeyAllowedRoutes = map[string][]string{
    "GET": {
        // ... existing routes ...
        // Billing routes
        "/plan",                    // Get current plan
        "/plan/list",               // List available plans
        "/trial/status",            // Get trial status
        "/invoices",                // List invoices
        "/invoices/current-usage",  // Current billing usage
        "/payment-methods",         // List payment methods
        "/payment-methods/wallet",  // Wallet/credits balance
    },
    "POST": {
        // ... existing routes ...
        // Billing routes
        "/plan/change",             // Change subscription plan
        "/plan/commit",             // Commit to plan (end trial)
        "/payment-methods",         // Add payment method
        "/coupons/apply",           // Apply coupon code
    },
    "DELETE": {
        "/plan/cancel",             // Cancel subscription
    },
}
```

### 3. Create Billing Client Package

**File: `apps/cloudstation-orchestrator/pkg/billing/types.go`**

```go
package billing

import "time"

type Plan struct {
    ID          string  `json:"id"`
    Code        string  `json:"code"`
    Name        string  `json:"name"`
    AmountCents int     `json:"amount_cents"`
    Interval    string  `json:"interval"`
    Features    []string `json:"features,omitempty"`
}

type Subscription struct {
    PlanCode      string    `json:"plan_code"`
    PlanName      string    `json:"plan_name"`
    Status        string    `json:"status"`
    TrialEndsAt   *time.Time `json:"trial_ends_at,omitempty"`
    NextBillingAt *time.Time `json:"next_billing_at,omitempty"`
}

type TrialStatus struct {
    IsActive      bool      `json:"is_active"`
    DaysRemaining int       `json:"days_remaining"`
    EndsAt        time.Time `json:"ends_at"`
    HasPayment    bool      `json:"has_payment_method"`
    CanCommit     bool      `json:"can_commit"`
}

type Invoice struct {
    ID            string    `json:"id"`
    Number        string    `json:"number"`
    Status        string    `json:"status"`
    AmountCents   int       `json:"amount_cents"`
    Currency      string    `json:"currency"`
    IssuingDate   time.Time `json:"issuing_date"`
    PaymentStatus string    `json:"payment_status"`
    FileURL       string    `json:"file_url,omitempty"`
}

type CheckoutURLResponse struct {
    URL       string `json:"url"`
    ExpiresAt string `json:"expires_at,omitempty"`
}
```

**File: `apps/cloudstation-orchestrator/pkg/billing/client.go`**

```go
package billing

import (
    "fmt"
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

func (c *Client) GetCurrentPlan() (*Subscription, error) {
    var resp Subscription
    if err := c.DoJSON("GET", "/plan", nil, &resp); err != nil {
        return nil, fmt.Errorf("get plan failed: %w", err)
    }
    return &resp, nil
}

func (c *Client) ListPlans() ([]Plan, error) {
    var resp struct {
        Plans []Plan `json:"plans"`
    }
    if err := c.DoJSON("GET", "/plan/list", nil, &resp); err != nil {
        return nil, fmt.Errorf("list plans failed: %w", err)
    }
    return resp.Plans, nil
}

func (c *Client) GetTrialStatus() (*TrialStatus, error) {
    var resp TrialStatus
    if err := c.DoJSON("GET", "/trial/status", nil, &resp); err != nil {
        return nil, fmt.Errorf("get trial status failed: %w", err)
    }
    return &resp, nil
}

func (c *Client) ChangePlan(planCode string) error {
    req := map[string]string{"plan_code": planCode}
    if err := c.DoJSON("POST", "/plan/change", req, nil); err != nil {
        return fmt.Errorf("change plan failed: %w", err)
    }
    return nil
}

func (c *Client) ListInvoices(page, pageSize int) ([]Invoice, error) {
    path := fmt.Sprintf("/invoices?page=%d&pageSize=%d", page, pageSize)
    var resp struct {
        Data []Invoice `json:"data"`
    }
    if err := c.DoJSON("GET", path, nil, &resp); err != nil {
        return nil, fmt.Errorf("list invoices failed: %w", err)
    }
    return resp.Data, nil
}

func (c *Client) GetCheckoutURL() (*CheckoutURLResponse, error) {
    var resp CheckoutURLResponse
    if err := c.DoJSON("POST", "/payment-methods/checkout-url", nil, &resp); err != nil {
        return nil, fmt.Errorf("get checkout URL failed: %w", err)
    }
    return &resp, nil
}
```

### 4. Create Billing CLI Commands

**File: `apps/cloudstation-orchestrator/cmd/cloudstation/billing_commands.go`**

```go
package main

import (
    "encoding/json"
    "fmt"
    "strings"

    "github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
    "github.com/thecloudstation/cloudstation-orchestrator/pkg/billing"
    "github.com/urfave/cli/v2"
)

func billingCommand() *cli.Command {
    return &cli.Command{
        Name:  "billing",
        Usage: "Manage subscription and billing",
        Subcommands: []*cli.Command{
            billingStatusCommand(),
            billingPlansCommand(),
            billingSubscribeCommand(),
            billingInvoicesCommand(),
            billingAddCardCommand(),
        },
    }
}

func billingStatusCommand() *cli.Command {
    return &cli.Command{
        Name:  "status",
        Usage: "Show current subscription status",
        Flags: []cli.Flag{
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            creds, err := auth.LoadCredentials()
            if err != nil {
                return fmt.Errorf("not logged in: run 'cs login' first")
            }

            client := billing.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)

            plan, err := client.GetCurrentPlan()
            if err != nil {
                return err
            }

            trial, _ := client.GetTrialStatus()

            if c.Bool("json") {
                result := map[string]interface{}{
                    "plan": plan,
                    "trial": trial,
                }
                data, _ := json.MarshalIndent(result, "", "  ")
                fmt.Println(string(data))
                return nil
            }

            fmt.Println("Subscription Status")
            fmt.Println(strings.Repeat("-", 40))
            fmt.Printf("Plan: %s\n", plan.PlanName)
            fmt.Printf("Status: %s\n", plan.Status)

            if trial != nil && trial.IsActive {
                fmt.Printf("\nTrial: %d days remaining\n", trial.DaysRemaining)
                if !trial.HasPayment {
                    fmt.Println("  ⚠️  No payment method on file")
                    fmt.Println("  Run 'cs billing add-card' to add one")
                }
            }

            return nil
        },
    }
}

func billingPlansCommand() *cli.Command {
    return &cli.Command{
        Name:  "plans",
        Usage: "List available subscription plans",
        Flags: []cli.Flag{
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            creds, err := auth.LoadCredentials()
            if err != nil {
                return fmt.Errorf("not logged in: run 'cs login' first")
            }

            client := billing.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)
            plans, err := client.ListPlans()
            if err != nil {
                return err
            }

            if c.Bool("json") {
                data, _ := json.MarshalIndent(plans, "", "  ")
                fmt.Println(string(data))
                return nil
            }

            fmt.Println("Available Plans")
            fmt.Println(strings.Repeat("-", 50))
            for _, p := range plans {
                price := float64(p.AmountCents) / 100
                fmt.Printf("  %s - $%.2f/%s\n", p.Name, price, p.Interval)
                fmt.Printf("    Code: %s\n", p.Code)
            }

            return nil
        },
    }
}

func billingSubscribeCommand() *cli.Command {
    return &cli.Command{
        Name:      "subscribe",
        Usage:     "Subscribe to a plan",
        ArgsUsage: "<plan-code>",
        Flags: []cli.Flag{
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            if c.NArg() < 1 {
                return fmt.Errorf("plan code required. Run 'cs billing plans' to see available plans")
            }

            creds, err := auth.LoadCredentials()
            if err != nil {
                return fmt.Errorf("not logged in: run 'cs login' first")
            }

            client := billing.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)

            // Check if user has payment method
            trial, _ := client.GetTrialStatus()
            if trial != nil && !trial.HasPayment {
                // Generate checkout URL
                checkout, err := client.GetCheckoutURL()
                if err != nil {
                    return fmt.Errorf("failed to generate checkout URL: %w", err)
                }

                if c.Bool("json") {
                    data, _ := json.MarshalIndent(map[string]string{
                        "message": "Payment method required",
                        "checkout_url": checkout.URL,
                    }, "", "  ")
                    fmt.Println(string(data))
                    return nil
                }

                fmt.Println("⚠️  Payment method required before subscribing")
                fmt.Println("\nAdd a payment method here:")
                fmt.Printf("  %s\n", checkout.URL)
                fmt.Println("\nAfter adding payment, run this command again.")
                return nil
            }

            planCode := c.Args().First()
            if err := client.ChangePlan(planCode); err != nil {
                return err
            }

            fmt.Printf("✓ Subscribed to plan: %s\n", planCode)
            return nil
        },
    }
}

func billingInvoicesCommand() *cli.Command {
    return &cli.Command{
        Name:  "invoices",
        Usage: "List invoices",
        Flags: []cli.Flag{
            &cli.IntFlag{Name: "limit", Value: 10, Usage: "Number of invoices to show"},
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            creds, err := auth.LoadCredentials()
            if err != nil {
                return fmt.Errorf("not logged in: run 'cs login' first")
            }

            client := billing.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)
            invoices, err := client.ListInvoices(1, c.Int("limit"))
            if err != nil {
                return err
            }

            if c.Bool("json") {
                data, _ := json.MarshalIndent(invoices, "", "  ")
                fmt.Println(string(data))
                return nil
            }

            fmt.Println("Invoices")
            fmt.Println(strings.Repeat("-", 60))
            for _, inv := range invoices {
                amount := float64(inv.AmountCents) / 100
                fmt.Printf("  %s - $%.2f %s (%s)\n", inv.Number, amount, inv.Currency, inv.PaymentStatus)
            }

            return nil
        },
    }
}

func billingAddCardCommand() *cli.Command {
    return &cli.Command{
        Name:  "add-card",
        Usage: "Get link to add payment method",
        Flags: []cli.Flag{
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            creds, err := auth.LoadCredentials()
            if err != nil {
                return fmt.Errorf("not logged in: run 'cs login' first")
            }

            client := billing.NewClient(c.String("api-url"), creds.APIKey, creds.ClientID)
            checkout, err := client.GetCheckoutURL()
            if err != nil {
                return err
            }

            if c.Bool("json") {
                data, _ := json.MarshalIndent(checkout, "", "  ")
                fmt.Println(string(data))
                return nil
            }

            fmt.Println("Add Payment Method")
            fmt.Println(strings.Repeat("-", 40))
            fmt.Println("Open this URL in your browser:")
            fmt.Printf("\n  %s\n\n", checkout.URL)

            return nil
        },
    }
}
```

### 5. Implement Google OAuth Login

**File: `apps/cloudstation-orchestrator/pkg/oauth/google.go`**

```go
package oauth

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "net/http"
    "time"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

const (
    // CloudStation's Google OAuth client ID (public)
    GoogleClientID = "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com"
)

type GoogleOAuth struct {
    config       *oauth2.Config
    codeVerifier string
    state        string
}

func NewGoogleOAuth(callbackPort int) (*GoogleOAuth, error) {
    // Generate PKCE code verifier
    verifier := make([]byte, 32)
    if _, err := rand.Read(verifier); err != nil {
        return nil, err
    }
    codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)

    // Generate state
    stateBytes := make([]byte, 16)
    rand.Read(stateBytes)
    state := base64.RawURLEncoding.EncodeToString(stateBytes)

    config := &oauth2.Config{
        ClientID: GoogleClientID,
        Scopes:   []string{"email", "profile"},
        Endpoint: google.Endpoint,
        RedirectURL: fmt.Sprintf("http://localhost:%d/callback", callbackPort),
    }

    return &GoogleOAuth{
        config:       config,
        codeVerifier: codeVerifier,
        state:        state,
    }, nil
}

func (g *GoogleOAuth) GetAuthURL() string {
    // Generate code challenge from verifier (S256)
    hash := sha256.Sum256([]byte(g.codeVerifier))
    codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

    return g.config.AuthCodeURL(g.state,
        oauth2.SetAuthURLParam("code_challenge", codeChallenge),
        oauth2.SetAuthURLParam("code_challenge_method", "S256"),
    )
}

func (g *GoogleOAuth) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
    return g.config.Exchange(ctx, code,
        oauth2.SetAuthURLParam("code_verifier", g.codeVerifier),
    )
}

func (g *GoogleOAuth) GetState() string {
    return g.state
}
```

**File: `apps/cloudstation-orchestrator/pkg/oauth/callback.go`**

```go
package oauth

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "time"
)

type CallbackResult struct {
    Code  string
    Error string
}

// StartCallbackServer starts a local HTTP server to receive OAuth callback
func StartCallbackServer(port int, expectedState string, timeout time.Duration) (*CallbackResult, error) {
    resultChan := make(chan *CallbackResult, 1)

    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, fmt.Errorf("failed to start callback server: %w", err)
    }

    server := &http.Server{
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.URL.Path != "/callback" {
                http.NotFound(w, r)
                return
            }

            // Verify state
            state := r.URL.Query().Get("state")
            if state != expectedState {
                resultChan <- &CallbackResult{Error: "invalid state parameter"}
                fmt.Fprintf(w, "<h1>Error</h1><p>Invalid state parameter. Please try again.</p>")
                return
            }

            // Check for error
            if errParam := r.URL.Query().Get("error"); errParam != "" {
                resultChan <- &CallbackResult{Error: errParam}
                fmt.Fprintf(w, "<h1>Error</h1><p>%s</p>", errParam)
                return
            }

            // Get authorization code
            code := r.URL.Query().Get("code")
            if code == "" {
                resultChan <- &CallbackResult{Error: "no authorization code received"}
                fmt.Fprintf(w, "<h1>Error</h1><p>No authorization code received.</p>")
                return
            }

            resultChan <- &CallbackResult{Code: code}
            fmt.Fprintf(w, `
                <h1>Success!</h1>
                <p>You can close this window and return to the terminal.</p>
                <script>setTimeout(function(){window.close();}, 2000);</script>
            `)
        }),
    }

    go server.Serve(listener)

    // Wait for result or timeout
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    select {
    case result := <-resultChan:
        server.Shutdown(context.Background())
        return result, nil
    case <-ctx.Done():
        server.Shutdown(context.Background())
        return nil, fmt.Errorf("timeout waiting for OAuth callback")
    }
}

// FindAvailablePort finds an available port for the callback server
func FindAvailablePort() (int, error) {
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        return 0, err
    }
    port := listener.Addr().(*net.TCPAddr).Port
    listener.Close()
    return port, nil
}
```

### 6. Add OAuth Login to Auth Commands

**File: `apps/cloudstation-orchestrator/cmd/cloudstation/auth_commands.go`**

Add to loginCommand():

```go
func loginCommand() *cli.Command {
    return &cli.Command{
        Name:  "login",
        Usage: "Authenticate with CloudStation",
        Flags: []cli.Flag{
            &cli.BoolFlag{
                Name:  "google",
                Usage: "Login with Google account",
            },
            &cli.StringFlag{
                Name:    "api-key",
                Usage:   "Your CloudStation API key",
                EnvVars: []string{"CS_API_KEY"},
            },
            // ... existing flags ...
        },
        Action: func(c *cli.Context) error {
            apiURL := c.String("api-url")

            // Google OAuth flow
            if c.Bool("google") {
                return loginWithGoogle(apiURL)
            }

            // Existing API key flow
            // ... existing code ...
        },
    }
}

func loginWithGoogle(apiURL string) error {
    fmt.Println("Logging in with Google...")

    // Find available port
    port, err := oauth.FindAvailablePort()
    if err != nil {
        return fmt.Errorf("failed to find available port: %w", err)
    }

    // Create OAuth handler
    googleAuth, err := oauth.NewGoogleOAuth(port)
    if err != nil {
        return fmt.Errorf("failed to initialize OAuth: %w", err)
    }

    // Get auth URL and open browser
    authURL := googleAuth.GetAuthURL()
    fmt.Printf("\nOpening browser for authentication...\n")
    fmt.Printf("If browser doesn't open, visit:\n  %s\n\n", authURL)

    // Try to open browser
    openBrowser(authURL)

    // Wait for callback
    fmt.Println("Waiting for authentication...")
    result, err := oauth.StartCallbackServer(port, googleAuth.GetState(), 5*time.Minute)
    if err != nil {
        return fmt.Errorf("OAuth callback failed: %w", err)
    }
    if result.Error != "" {
        return fmt.Errorf("OAuth error: %s", result.Error)
    }

    // Exchange code for token
    ctx := context.Background()
    token, err := googleAuth.ExchangeCode(ctx, result.Code)
    if err != nil {
        return fmt.Errorf("failed to exchange code: %w", err)
    }

    // Exchange Google token for CloudStation credentials
    client := auth.NewClient(apiURL)
    creds, err := client.AuthenticateWithGoogle(token.Extra("id_token").(string))
    if err != nil {
        return fmt.Errorf("failed to authenticate with CloudStation: %w", err)
    }

    // Save credentials
    if err := auth.SaveCredentials(creds); err != nil {
        return fmt.Errorf("failed to save credentials: %w", err)
    }

    fmt.Printf("✓ Authenticated as %s\n", creds.Email)
    return nil
}

func openBrowser(url string) {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "darwin":
        cmd = "open"
        args = []string{url}
    case "linux":
        cmd = "xdg-open"
        args = []string{url}
    case "windows":
        cmd = "rundll32"
        args = []string{"url.dll,FileProtocolHandler", url}
    }

    exec.Command(cmd, args...).Start()
}
```

### 7. Add Backend OAuth Token Exchange Endpoint

**File: `apps/cs-backend/api/internal/cliauth/handler.go`**

Add new endpoint:

```go
// AuthenticateWithGoogle exchanges a Google ID token for CLI credentials
func (h *Handler) AuthenticateWithGoogle(c *gin.Context) {
    var req struct {
        IDToken string `json:"id_token" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    // Verify Google token and get user
    // This reuses the existing OAuth service
    socialData, err := h.googleService.GetProfileByToken(req.IDToken)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Google token"})
        return
    }

    // Find or create user
    user, err := h.userRepo.FindByEmail(socialData.Email)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    // Generate CLI credentials (same as API key auth)
    creds, err := h.generateCLICredentials(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate credentials"})
        return
    }

    c.JSON(http.StatusOK, creds)
}
```

### 8. Register New Commands in main.go

**File: `apps/cloudstation-orchestrator/cmd/cloudstation/main.go`**

```go
Commands: []*cli.Command{
    // Lifecycle commands
    initCommand(),
    buildCommand(),
    deployCommand(),
    upCommand(),

    // Template & Image commands
    templateCommand(),
    imageCommand(),

    // Billing commands (NEW)
    billingCommand(),

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

### 9. Update go.mod Dependencies

```bash
cd apps/cloudstation-orchestrator
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/manifoldco/promptui@latest
go get golang.org/x/oauth2@latest
go get golang.org/x/oauth2/google@latest
```

### 10. Update Template Deploy to Pass Team Parameter

**File: `apps/cloudstation-orchestrator/cmd/cloudstation/template_commands.go`**

Add `--team` flag and pass to API:

```go
func templateDeployCommand() *cli.Command {
    return &cli.Command{
        Name:      "deploy",
        Usage:     "Deploy a template (one-click deployment)",
        ArgsUsage: "<template-id-or-slug>",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "project", Usage: "Project ID"},
            &cli.StringFlag{Name: "team", Usage: "Team slug (auto-detected if not specified)"},
            &cli.StringFlag{Name: "env", Usage: "Environment ID (optional)"},
            &cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
            &cli.StringFlag{Name: "api-url", Value: "https://cst-cs-backend-gmlyovvq.cloud-station.io", EnvVars: []string{"CS_API_URL"}},
        },
        Action: func(c *cli.Context) error {
            // ... existing code ...

            // Build deploy URL with team parameter
            path := "/templates/deploy"
            if team := c.String("team"); team != "" {
                path = fmt.Sprintf("/templates/deploy?team=%s", team)
            }

            // Use path in DoJSON call
            // ...
        },
    }
}
```

## Testing Strategy

### Unit Tests
- Test OAuth PKCE code generation and verification
- Test billing client methods with mock HTTP
- Test credential storage with OAuth tokens
- Test TUI components in isolation

### Integration Tests
- Test full Google OAuth flow (manual with real Google account)
- Test billing endpoints with real API
- Test template deploy with team auto-resolution
- Test in Orbit sandbox with AI agent

### End-to-End Tests
- Complete subscription flow: login → billing status → subscribe → deploy
- AI deployment flow: Orbit agent → check subscription → deploy or generate payment link
- OAuth token refresh after expiration

## Acceptance Criteria

- [ ] `cs login --google` opens browser, completes OAuth, saves credentials
- [ ] `cs billing status` shows current plan and trial info
- [ ] `cs billing plans` lists available plans with prices
- [ ] `cs billing subscribe <plan>` subscribes or prompts for payment method
- [ ] `cs billing add-card` generates Stripe checkout URL
- [ ] `cs billing invoices` lists recent invoices
- [ ] `cs template deploy` works for API key users (team auto-resolved)
- [ ] Non-subscribed users get payment link when trying to deploy
- [ ] AI agents in Orbit can deploy via CLI
- [ ] All commands support `--json` flag for programmatic use

## Validation Commands

```bash
# Build and test
cd apps/cloudstation-orchestrator
go build -o cs ./cmd/cloudstation/

# Test help
./cs billing --help
./cs login --help

# Test billing commands (requires login)
./cs billing status
./cs billing plans --json
./cs billing invoices --limit 5

# Test OAuth (opens browser)
./cs login --google

# Test template deploy (should work now)
./cs template deploy redis

# Test in non-interactive mode (for AI)
./cs billing status --json
./cs template deploy --json redis
```

## Notes

### Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - TUI styling
- `github.com/manifoldco/promptui` - Interactive prompts
- `golang.org/x/oauth2` - OAuth2 client
- `golang.org/x/oauth2/google` - Google OAuth provider

### Security Considerations
- OAuth uses PKCE flow (no client secret in CLI)
- Credentials stored with 0600 permissions
- OAuth tokens stored alongside API key
- Token refresh implemented for long sessions

### Backend Changes Required
1. Add billing routes to API key allowlist
2. Add `/api/cli/auth/google` endpoint
3. Fix team auto-resolution in DeployTemplate
4. Add team membership check in FindProjectByIDWithAuth

### AI Integration Notes
- AI agents should use `--json` flag for all commands
- Check `cs billing status --json` before deployments
- If no subscription, parse checkout_url from `cs billing add-card --json`
- Use non-interactive mode when running in Orbit sandbox
