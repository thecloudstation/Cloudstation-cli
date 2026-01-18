package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/templates"
	"github.com/urfave/cli/v2"
)

// newTemplateClient creates a template client using JWT Bearer auth
func newTemplateClient(baseURL string, creds *auth.Credentials) (*templates.Client, error) {
	if creds.SessionToken == "" {
		return nil, fmt.Errorf("not authenticated: run 'cs login' first")
	}
	return templates.NewClientWithToken(baseURL, creds.SessionToken), nil
}

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

			client, err := newTemplateClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

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

			client, err := newTemplateClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}
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
			&cli.StringFlag{Name: "team", Usage: "Team slug (required for API key auth)"},
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Preview deployment without executing"},
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

			apiURL := c.String("api-url")

			// Get project ID from flag or resolve from linked service
			projectID := c.String("project")
			if projectID == "" {
				serviceID, err := auth.LoadServiceLink()
				if err != nil {
					return fmt.Errorf("no project specified: use --project or run 'cs link' first")
				}
				// Resolve project ID from service details
				authClient := auth.NewClient(apiURL)
				details, err := authClient.GetServiceDetails(creds.SessionToken, serviceID)
				if err != nil {
					return fmt.Errorf("failed to resolve project ID from linked service: %w", err)
				}
				if details.ProjectID == "" {
					return fmt.Errorf("linked service has no project ID: use --project flag")
				}
				projectID = details.ProjectID
			}

			client, err := newTemplateClient(apiURL, creds)
			if err != nil {
				return err
			}

			templateID := c.Args().First()
			envID := c.String("env")
			teamSlug := c.String("team")

			// Dry-run mode: preview deployment without executing
			if c.Bool("dry-run") {
				if c.Bool("json") {
					output := map[string]interface{}{
						"dry_run":     true,
						"template_id": templateID,
						"project_id":  projectID,
					}
					if envID != "" {
						output["environment_id"] = envID
					}
					if teamSlug != "" {
						output["team"] = teamSlug
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				// Human-readable dry-run output
				fmt.Println("DRY RUN MODE - Would deploy:")
				fmt.Printf("  Template: %s\n", templateID)
				fmt.Printf("  Project: %s\n", projectID)
				if envID != "" {
					fmt.Printf("  Environment: %s\n", envID)
				}
				if teamSlug != "" {
					fmt.Printf("  Team: %s\n", teamSlug)
				}
				return nil
			}

			resp, err := client.Deploy(templates.DeployTemplateRequest{
				TemplateID:  templateID,
				ProjectID:   projectID,
				Environment: envID,
				Team:        teamSlug,
			})
			if err != nil {
				return err
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(resp, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Deployment started!\n")
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

			client, err := newTemplateClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}
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
