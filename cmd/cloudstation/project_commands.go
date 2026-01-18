package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
	"github.com/urfave/cli/v2"
)

// Project represents a CloudStation project
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	TeamID      string    `json:"teamId,omitempty"`
	UserID      int       `json:"userId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ProjectsResponse is the API response for listing projects
type ProjectsResponse struct {
	Data       []Project `json:"data"`
	TotalCount int       `json:"totalCount"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
}

// ProjectClient handles project API calls
type ProjectClient struct {
	*httpclient.BaseClient
}

// NewProjectClient creates a project client with JWT auth
func NewProjectClient(baseURL, sessionToken string) *ProjectClient {
	client := &ProjectClient{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+sessionToken)
	return client
}

// List retrieves user's projects
func (c *ProjectClient) List() (*ProjectsResponse, error) {
	var resp ProjectsResponse
	if err := c.DoJSON("GET", "/projects", nil, &resp); err != nil {
		return nil, fmt.Errorf("list projects failed: %w", err)
	}
	return &resp, nil
}

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name           string  `json:"name"`
	Description    *string `json:"description,omitempty"`
	Team           string  `json:"team,omitempty"`
	RenameIfExists bool    `json:"renameIfExists,omitempty"`
}

// Create creates a new project
func (c *ProjectClient) Create(req *CreateProjectRequest) (*Project, error) {
	var resp Project
	if err := c.DoJSON("POST", "/projects", req, &resp); err != nil {
		return nil, fmt.Errorf("create project failed: %w", err)
	}
	return &resp, nil
}

func projectCommand() *cli.Command {
	return &cli.Command{
		Name:  "project",
		Usage: "Manage CloudStation projects",
		Subcommands: []*cli.Command{
			projectListCommand(),
			projectCreateCommand(),
		},
	}
}

func projectCreateCommand() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new project",
		ArgsUsage: "<project-name>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "description",
				Usage: "Project description",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "Team slug or ID (auto-detected from service token if not provided)",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
			&cli.StringFlag{
				Name:    "auth-url",
				Value:   DefaultAuthURL,
				EnvVars: []string{"CS_AUTH_URL"},
				Hidden:  true,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("project name required\n\nUsage: cs project create <project-name>")
			}

			projectName := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if creds.SessionToken == "" {
				return fmt.Errorf("session token required: run 'cs login' first")
			}

			client := NewProjectClient(c.String("auth-url"), creds.SessionToken)

			req := &CreateProjectRequest{
				Name: projectName,
			}

			if desc := c.String("description"); desc != "" {
				req.Description = &desc
			}

			team := auth.GetTeamContext(c.String("team"))
			if team != "" {
				req.Team = team
			}

			project, err := client.Create(req)
			if err != nil {
				return err
			}

			if c.Bool("json") {
				output := map[string]interface{}{
					"success":    true,
					"project_id": project.ID,
					"name":       project.Name,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("✓ Project created successfully!\n")
			fmt.Printf("  ID:   %s\n", project.ID)
			fmt.Printf("  Name: %s\n", project.Name)
			fmt.Println("\nDeploy a template to this project:")
			fmt.Printf("  cs template deploy redis --project %s\n", project.ID)

			return nil
		},
	}
}

func projectListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List your projects",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
			&cli.StringFlag{
				Name:    "auth-url",
				Value:   DefaultAuthURL,
				EnvVars: []string{"CS_AUTH_URL"},
				Hidden:  true,
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			if creds.SessionToken == "" {
				return fmt.Errorf("session token required: run 'cs login' first")
			}

			client := NewProjectClient(c.String("auth-url"), creds.SessionToken)
			resp, err := client.List()
			if err != nil {
				return err
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(resp.Data, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			if len(resp.Data) == 0 {
				fmt.Println("No projects found")
				fmt.Println("\nCreate a project at https://app.cloud-station.io")
				return nil
			}

			fmt.Printf("Projects (%d total)\n", resp.TotalCount)
			fmt.Println("------------------------------------------------------------")
			for _, p := range resp.Data {
				fmt.Printf("  %s - %s\n", p.ID, p.Name)
				if p.Description != "" {
					desc := p.Description
					if len(desc) > 60 {
						desc = desc[:60] + "..."
					}
					fmt.Printf("    %s\n", desc)
				}
			}
			fmt.Println("\nUse project ID with: cs template deploy --project=<id> <template>")
			return nil
		},
	}
}
