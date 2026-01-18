package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
	"github.com/urfave/cli/v2"
)

// Service represents a CloudStation service (docker image or integration)
type Service struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	NomadStatus   string `json:"nomad_status"`
	ImageURL      string `json:"image_url,omitempty"`
	Tag           string `json:"tag,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	ReplicaCount  int    `json:"replica_count,omitempty"`
	HealthyAllocs int    `json:"healthy_allocs,omitempty"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// ServicesResponse is the API response for listing services
type ServicesResponse struct {
	Images       []Service `json:"images"`
	Integrations []Service `json:"integrations"`
}

// AllServices returns combined list of images and integrations
func (r *ServicesResponse) AllServices() []Service {
	all := make([]Service, 0, len(r.Images)+len(r.Integrations))
	all = append(all, r.Images...)
	all = append(all, r.Integrations...)
	return all
}

// ServiceActionResponse is the API response for service actions (start, stop, restart, delete)
type ServiceActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

// ServiceClient handles service API calls
type ServiceClient struct {
	*httpclient.BaseClient
}

// NewServiceClient creates a service client with JWT Bearer auth
func NewServiceClient(baseURL string, creds *auth.Credentials) (*ServiceClient, error) {
	if creds.SessionToken == "" {
		return nil, fmt.Errorf("not authenticated: run 'cs login' first")
	}
	client := &ServiceClient{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+creds.SessionToken)
	return client, nil
}

// List retrieves services for a project
func (c *ServiceClient) List(projectID, team string) (*ServicesResponse, error) {
	var resp ServicesResponse
	path := fmt.Sprintf("/services/list/%s", projectID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
	}
	if err := c.DoJSON("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list services failed: %w", err)
	}
	return &resp, nil
}

// Stop stops a service
func (c *ServiceClient) Stop(serviceID, team string) (*ServiceActionResponse, error) {
	var resp ServiceActionResponse
	path := fmt.Sprintf("/services/%s/stop", serviceID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
	}
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("stop service failed: %w", err)
	}
	return &resp, nil
}

// Start starts a service
func (c *ServiceClient) Start(serviceID, team string) (*ServiceActionResponse, error) {
	var resp ServiceActionResponse
	path := fmt.Sprintf("/services/%s/start", serviceID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
	}
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("start service failed: %w", err)
	}
	return &resp, nil
}

// Restart restarts a service
func (c *ServiceClient) Restart(serviceID, team string) (*ServiceActionResponse, error) {
	var resp ServiceActionResponse
	path := fmt.Sprintf("/services/%s/restart", serviceID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
	}
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("restart service failed: %w", err)
	}
	return &resp, nil
}

// Delete deletes a service
func (c *ServiceClient) Delete(serviceID, team string) (*ServiceActionResponse, error) {
	var resp ServiceActionResponse
	path := fmt.Sprintf("/services/%s", serviceID)
	if team != "" {
		path = fmt.Sprintf("%s?team=%s", path, url.QueryEscape(team))
	}
	if err := c.DoJSON("DELETE", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("delete service failed: %w", err)
	}
	return &resp, nil
}

// serviceCommand returns the main service command with subcommands
func serviceCommand() *cli.Command {
	return &cli.Command{
		Name:  "service",
		Usage: "Manage CloudStation services",
		Subcommands: []*cli.Command{
			serviceListCommand(),
			serviceStopCommand(),
			serviceStartCommand(),
			serviceRestartCommand(),
			serviceDeleteCommand(),
		},
	}
}

// serviceListCommand lists services in a project
func serviceListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List services in a project",
		ArgsUsage: "<project-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("project ID required\n\nUsage: cs service list <project-id>")
			}

			projectID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewServiceClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			team := auth.GetTeamContext(c.String("team"))
			resp, err := client.List(projectID, team)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(resp, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Human-readable output
			services := resp.AllServices()
			if len(services) == 0 {
				fmt.Printf("No services found in project %s\n", projectID)
				return nil
			}

			fmt.Printf("Services in project %s (%d total)\n", projectID, len(services))
			fmt.Println(strings.Repeat("-", 80))
			fmt.Printf("%-42s %-20s %-12s\n", "ID", "NAME", "STATUS")
			fmt.Println(strings.Repeat("-", 80))

			for _, svc := range services {
				// Format status with visual indicator
				status := svc.NomadStatus
				switch strings.ToLower(status) {
				case "running":
					status = "✅ running"
				case "stopped", "dead":
					status = "💀 stopped"
				case "pending":
					status = "⏳ pending"
				case "failed":
					status = "❌ failed"
				default:
					status = "⚪ " + status
				}

				name := svc.Name
				if len(name) > 20 {
					name = name[:17] + "..."
				}

				fmt.Printf("%-42s %-20s %-12s\n", svc.ID, name, status)
			}

			return nil
		},
	}
}

// serviceStopCommand stops a service
func serviceStopCommand() *cli.Command {
	return &cli.Command{
		Name:      "stop",
		Usage:     "Stop a running service",
		ArgsUsage: "<service-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview action without executing",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs service stop <service-id>")
			}

			serviceID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewServiceClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			// Dry-run: preview action without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":    true,
						"action":     "stop",
						"service_id": serviceID,
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				fmt.Printf("DRY RUN: Would stop service %s\n", serviceID)
				return nil
			}

			if !c.Bool("json") {
				fmt.Printf("Stopping service %s...\n", serviceID)
			}

			team := auth.GetTeamContext(c.String("team"))
			resp, err := client.Stop(serviceID, team)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
					"action":     "stop",
					"message":    resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Service %s stopped successfully\n", serviceID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}

// serviceStartCommand starts a service
func serviceStartCommand() *cli.Command {
	return &cli.Command{
		Name:      "start",
		Usage:     "Start a stopped service",
		ArgsUsage: "<service-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview action without executing",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs service start <service-id>")
			}

			serviceID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewServiceClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			// Dry-run: preview action without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":    true,
						"action":     "start",
						"service_id": serviceID,
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				fmt.Printf("DRY RUN: Would start service %s\n", serviceID)
				return nil
			}

			if !c.Bool("json") {
				fmt.Printf("Starting service %s...\n", serviceID)
			}

			team := auth.GetTeamContext(c.String("team"))
			resp, err := client.Start(serviceID, team)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
					"action":     "start",
					"message":    resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Service %s started successfully\n", serviceID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}

// serviceRestartCommand restarts a service
func serviceRestartCommand() *cli.Command {
	return &cli.Command{
		Name:      "restart",
		Usage:     "Restart a service",
		ArgsUsage: "<service-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview action without executing",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs service restart <service-id>")
			}

			serviceID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewServiceClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			// Dry-run: preview action without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":    true,
						"action":     "restart",
						"service_id": serviceID,
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				fmt.Printf("DRY RUN: Would restart service %s\n", serviceID)
				return nil
			}

			if !c.Bool("json") {
				fmt.Printf("Restarting service %s...\n", serviceID)
			}

			team := auth.GetTeamContext(c.String("team"))
			resp, err := client.Restart(serviceID, team)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
					"action":     "restart",
					"message":    resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Service %s restarted successfully\n", serviceID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}

// serviceDeleteCommand deletes a service
func serviceDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a service (permanent)",
		ArgsUsage: "<service-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Skip confirmation prompt",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Preview action without executing",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs service delete <service-id>")
			}

			serviceID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			// Dry-run: preview action without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":    true,
						"action":     "delete",
						"service_id": serviceID,
						"warning":    "This action cannot be undone. All associated data will be removed.",
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				fmt.Printf("DRY RUN: Would permanently delete service %s\n", serviceID)
				fmt.Println("  This action cannot be undone.")
				fmt.Println("  All associated data will be removed.")
				return nil
			}

			// Confirmation prompt unless --force is provided
			if !c.Bool("force") && !c.Bool("json") {
				fmt.Printf("WARNING: This will permanently delete service %s and all associated data.\n", serviceID)
				fmt.Print("Are you sure? Type 'yes' to confirm: ")

				reader := bufio.NewReader(os.Stdin)
				confirmation, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}

				confirmation = strings.TrimSpace(strings.ToLower(confirmation))
				if confirmation != "yes" {
					fmt.Println("Delete cancelled")
					return nil
				}
			}

			client, err := NewServiceClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			if !c.Bool("json") {
				fmt.Printf("Deleting service %s...\n", serviceID)
			}

			team := auth.GetTeamContext(c.String("team"))
			resp, err := client.Delete(serviceID, team)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
					"action":     "delete",
					"message":    resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Service %s deleted successfully\n", serviceID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}
