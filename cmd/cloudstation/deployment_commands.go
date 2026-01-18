package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
	"github.com/urfave/cli/v2"
)

// Deployment represents a CloudStation deployment
type Deployment struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	Branch     string `json:"branch,omitempty"`
	CommitHash string `json:"commit_hash,omitempty"`
	ImageTag   string `json:"image_tag,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// DeploymentsListResponse is the API response for listing deployments
type DeploymentsListResponse struct {
	Title string       `json:"title"`
	Data  []Deployment `json:"data"`
}

// DeploymentStatusResponse is the API response for deployment status
type DeploymentStatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// DeploymentActionResponse is the API response for deployment actions (cancel, clear-queue)
type DeploymentActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// DeploymentClient handles deployment API calls
type DeploymentClient struct {
	*httpclient.BaseClient
}

// NewDeploymentClient creates a deployment client with JWT Bearer auth
func NewDeploymentClient(baseURL string, creds *auth.Credentials) (*DeploymentClient, error) {
	if creds.SessionToken == "" {
		return nil, fmt.Errorf("not authenticated: run 'cs login' first")
	}
	client := &DeploymentClient{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+creds.SessionToken)
	return client, nil
}

// List retrieves deployments for a service
func (c *DeploymentClient) List(serviceID string) (*DeploymentsListResponse, error) {
	var resp DeploymentsListResponse
	path := fmt.Sprintf("/deployments/service/%s", serviceID)
	if err := c.DoJSON("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("list deployments failed: %w", err)
	}
	return &resp, nil
}

// Status retrieves the status of a specific deployment
func (c *DeploymentClient) Status(deploymentID string) (*DeploymentStatusResponse, error) {
	var resp DeploymentStatusResponse
	path := fmt.Sprintf("/deployments/%s", deploymentID)
	if err := c.DoJSON("GET", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("get deployment status failed: %w", err)
	}
	return &resp, nil
}

// Cancel stops a running deployment
func (c *DeploymentClient) Cancel(deploymentID string) (*DeploymentActionResponse, error) {
	var resp DeploymentActionResponse
	path := fmt.Sprintf("/deployments/%s/stop", deploymentID)
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("cancel deployment failed: %w", err)
	}
	return &resp, nil
}

// ClearQueue clears stuck deployments in the queue for a service (admin operation)
func (c *DeploymentClient) ClearQueue(serviceID string) (*DeploymentActionResponse, error) {
	var resp DeploymentActionResponse
	path := fmt.Sprintf("/deployments/admin/service/%s/clear-queue", serviceID)
	if err := c.DoJSON("POST", path, nil, &resp); err != nil {
		return nil, fmt.Errorf("clear deployment queue failed: %w", err)
	}
	return &resp, nil
}

// deploymentCommand returns the main deployment command with subcommands
func deploymentCommand() *cli.Command {
	return &cli.Command{
		Name:  "deployment",
		Usage: "Manage CloudStation deployments",
		Subcommands: []*cli.Command{
			deploymentListCommand(),
			deploymentStatusCommand(),
			deploymentCancelCommand(),
			deploymentClearQueueCommand(),
		},
	}
}

// deploymentListCommand lists deployments for a service
func deploymentListCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List deployments for a service",
		ArgsUsage: "<service-id>",
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
			&cli.StringFlag{
				Name:  "status",
				Usage: "Filter by status (e.g., running, completed, failed)",
			},
			&cli.IntFlag{
				Name:  "page",
				Value: 1,
				Usage: "Page number for pagination",
			},
			&cli.IntFlag{
				Name:  "page-size",
				Value: 20,
				Usage: "Number of items per page",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs deployment list <service-id>")
			}

			serviceID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewDeploymentClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			resp, err := client.List(serviceID)
			if err != nil {
				return err
			}

			// Filter by status if specified
			statusFilter := c.String("status")
			var deployments []Deployment
			if statusFilter != "" {
				for _, d := range resp.Data {
					if strings.EqualFold(d.Status, statusFilter) {
						deployments = append(deployments, d)
					}
				}
			} else {
				deployments = resp.Data
			}

			if c.Bool("json") {
				output := map[string]interface{}{
					"title": resp.Title,
					"data":  deployments,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Human-readable output
			if len(deployments) == 0 {
				fmt.Printf("No deployments found for service %s\n", serviceID)
				return nil
			}

			fmt.Printf("Deployments for service %s (%d total)\n", serviceID, len(deployments))
			fmt.Println(strings.Repeat("-", 100))
			fmt.Printf("%-36s %-12s %-10s %-20s %-20s\n", "ID", "STATUS", "TYPE", "BRANCH/TAG", "CREATED")
			fmt.Println(strings.Repeat("-", 100))

			for _, dep := range deployments {
				// Format status with visual indicator
				status := dep.Status
				switch strings.ToLower(status) {
				case "running", "in_progress":
					status = "[*] running"
				case "completed", "success":
					status = "[+] completed"
				case "failed", "error":
					status = "[-] failed"
				case "queued", "pending":
					status = "[~] queued"
				case "cancelled", "stopped":
					status = "[x] cancelled"
				default:
					status = "[ ] " + status
				}

				// Get branch or image tag
				branchOrTag := dep.Branch
				if branchOrTag == "" {
					branchOrTag = dep.ImageTag
				}
				if branchOrTag == "" {
					branchOrTag = "-"
				}
				if len(branchOrTag) > 20 {
					branchOrTag = branchOrTag[:17] + "..."
				}

				// Format created time
				createdAt := dep.CreatedAt
				if len(createdAt) > 20 {
					createdAt = createdAt[:19]
				}

				fmt.Printf("%-36s %-12s %-10s %-20s %-20s\n",
					dep.ID, status, dep.Type, branchOrTag, createdAt)
			}

			return nil
		},
	}
}

// deploymentStatusCommand gets the status of a deployment
func deploymentStatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Get deployment status",
		ArgsUsage: "<deployment-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
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
				return fmt.Errorf("deployment ID required\n\nUsage: cs deployment status <deployment-id>")
			}

			deploymentID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewDeploymentClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			resp, err := client.Status(deploymentID)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(resp, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			// Human-readable output
			fmt.Printf("Deployment: %s\n", resp.ID)
			fmt.Println(strings.Repeat("-", 50))

			// Format status with visual indicator
			status := resp.Status
			switch strings.ToLower(status) {
			case "running", "in_progress":
				status = "[*] Running"
			case "completed", "success":
				status = "[+] Completed"
			case "failed", "error":
				status = "[-] Failed"
			case "queued", "pending":
				status = "[~] Queued"
			case "cancelled", "stopped":
				status = "[x] Cancelled"
			default:
				status = "[ ] " + status
			}

			fmt.Printf("Status: %s\n", status)
			fmt.Printf("Type:   %s\n", resp.Type)

			return nil
		},
	}
}

// deploymentCancelCommand cancels a running deployment
func deploymentCancelCommand() *cli.Command {
	return &cli.Command{
		Name:      "cancel",
		Usage:     "Cancel a running deployment",
		ArgsUsage: "<deployment-id>",
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
				return fmt.Errorf("deployment ID required\n\nUsage: cs deployment cancel <deployment-id>")
			}

			deploymentID := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if !auth.IsValid(creds) {
				return fmt.Errorf("credentials expired: run 'cs login' again")
			}

			client, err := NewDeploymentClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			// Dry-run: preview action without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":       true,
						"action":        "cancel",
						"deployment_id": deploymentID,
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				fmt.Printf("DRY RUN: Would cancel deployment %s\n", deploymentID)
				return nil
			}

			if !c.Bool("json") {
				fmt.Printf("Cancelling deployment %s...\n", deploymentID)
			}

			resp, err := client.Cancel(deploymentID)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":       true,
					"deployment_id": deploymentID,
					"action":        "cancel",
					"message":       resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Deployment %s cancelled successfully\n", deploymentID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}

// deploymentClearQueueCommand clears stuck deployments in the queue for a service (admin)
func deploymentClearQueueCommand() *cli.Command {
	return &cli.Command{
		Name:      "clear-queue",
		Usage:     "Clear stuck queue for a service (admin)",
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
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("service ID required\n\nUsage: cs deployment clear-queue <service-id>")
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
						"action":     "clear_queue",
						"service_id": serviceID,
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				fmt.Printf("DRY RUN: Would clear deployment queue for service %s\n", serviceID)
				fmt.Println("  This would remove all stuck/pending deployments")
				return nil
			}

			// Confirmation prompt unless --force is provided
			if !c.Bool("force") && !c.Bool("json") {
				fmt.Printf("WARNING: This will clear all queued/stuck deployments for service %s.\n", serviceID)
				fmt.Printf("This is an admin operation that cannot be undone.\n")
				fmt.Print("Are you sure? Type 'yes' to confirm: ")

				reader := bufio.NewReader(os.Stdin)
				confirmation, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}

				confirmation = strings.TrimSpace(strings.ToLower(confirmation))
				if confirmation != "yes" {
					fmt.Println("Clear queue cancelled")
					return nil
				}
			}

			client, err := NewDeploymentClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			if !c.Bool("json") {
				fmt.Printf("Clearing deployment queue for service %s...\n", serviceID)
			}

			resp, err := client.ClearQueue(serviceID)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
					"action":     "clear-queue",
					"message":    resp.Message,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Deployment queue cleared for service %s\n", serviceID)
			if resp.Message != "" {
				fmt.Printf("  %s\n", resp.Message)
			}

			return nil
		},
	}
}
