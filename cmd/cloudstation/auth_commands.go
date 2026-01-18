package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/urfave/cli/v2"
)

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Authenticate with CloudStation using Google OAuth",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
			},
		},
		Action: func(c *cli.Context) error {
			apiURL := c.String("api-url")
			return loginWithGoogle(apiURL)
		},
	}
}

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Remove stored credentials",
		Action: func(c *cli.Context) error {
			if err := auth.DeleteCredentials(); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Not logged in")
					return nil
				}
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
		Usage: "Display current authenticated user or service token info",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return err
			}

			// Check if this is a service token (from USER_TOKEN or CS_TOKEN env)
			hasUserToken := os.Getenv("USER_TOKEN") != ""
			hasDeprecatedToken := os.Getenv("CS_TOKEN") != ""
			isServiceToken := creds.Email == "" && creds.SessionToken != "" && (hasUserToken || hasDeprecatedToken)

			if isServiceToken {
				// Decode JWT to show token info
				claims, err := auth.DecodeJWTClaims(creds.SessionToken)
				if err != nil {
					return fmt.Errorf("failed to decode service token: %w", err)
				}

				if c.Bool("json") {
					output := map[string]interface{}{
						"auth_type":   "service_token",
						"token_type":  claims["token_type"],
						"role":        claims["role"],
						"user_id":     claims["user_id"],
						"team_slug":   claims["team_slug"],
						"sandbox_id":  claims["sandbox_id"],
						"expires_at":  nil,
						"token_valid": true,
					}
					// Parse expiration
					if exp, ok := claims["exp"].(float64); ok {
						expTime := time.Unix(int64(exp), 0)
						output["expires_at"] = expTime.Format(time.RFC3339)
						output["token_valid"] = time.Now().Before(expTime)
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
					return nil
				}

				if hasUserToken {
					fmt.Println("Authentication: Service Token (USER_TOKEN)")
				} else if hasDeprecatedToken {
					fmt.Println("Authentication: Service Token (CS_TOKEN - deprecated, use USER_TOKEN)")
				} else {
					fmt.Println("Authentication: Service Token")
				}
				fmt.Printf("Token Type: %v\n", claims["token_type"])
				fmt.Printf("Role: %v\n", claims["role"])
				fmt.Printf("User ID: %v\n", claims["user_id"])
				if teamSlug, ok := claims["team_slug"].(string); ok && teamSlug != "" {
					fmt.Printf("Team: %s\n", teamSlug)
				}
				if exp, ok := claims["exp"].(float64); ok {
					expTime := time.Unix(int64(exp), 0)
					fmt.Printf("Expires: %s\n", expTime.Format("2006-01-02 15:04:05"))
					if time.Now().After(expTime) {
						fmt.Println("⚠️  Token has expired!")
					}
				}
				return nil
			}

			// Regular user credentials
			if c.Bool("json") {
				output := map[string]interface{}{
					"auth_type":      "user",
					"email":          creds.Email,
					"user_id":        creds.UserID,
					"user_uuid":      creds.UserUUID,
					"services_count": len(creds.Services),
					"expires_at":     creds.ExpiresAt,
					"is_super_admin": creds.IsSuperAdmin,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Email: %s\n", creds.Email)
			fmt.Printf("User ID: %d\n", creds.UserID)
			fmt.Printf("Services: %d linked\n", len(creds.Services))

			// Display expiration information if available
			if expiry := auth.GetExpirationTime(creds); expiry != nil {
				fmt.Printf("Expires: %s\n", expiry.Format("2006-01-02 15:04:05"))
			}

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
			&cli.StringFlag{
				Name:  "team",
				Usage: "Filter services by team slug",
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "CloudStation API URL",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output as JSON",
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return err
			}

			serviceID := c.String("service")
			apiURL := c.String("api-url")
			teamFilter := auth.GetTeamContext(c.String("team"))

			// Fetch services dynamically from API
			authClient := auth.NewClient(apiURL)
			services, err := authClient.ListUserServices(creds.SessionToken)
			if err != nil {
				// Fall back to cached services if API fails
				fmt.Printf("Warning: Could not fetch services from API: %v\n", err)
				fmt.Println("Falling back to cached services...")
				services = make([]auth.UserService, 0, len(creds.Services))
				for _, svc := range creds.Services {
					services = append(services, auth.UserService{
						ID:   svc.ServiceID,
						Name: svc.ServiceName,
					})
				}
			}

			// Filter by team if specified
			if teamFilter != "" {
				filtered := make([]auth.UserService, 0)
				for _, svc := range services {
					if svc.TeamSlug == teamFilter {
						filtered = append(filtered, svc)
					}
				}
				services = filtered
			}

			if serviceID == "" {
				// Interactive selection from available services
				if len(services) == 0 {
					if teamFilter != "" {
						return fmt.Errorf("no services found for team '%s'", teamFilter)
					}
					return fmt.Errorf("no services available to link")
				}

				fmt.Println("Available services:")
				for i, svc := range services {
					teamInfo := ""
					if svc.TeamSlug != "" {
						teamInfo = fmt.Sprintf(" [%s]", svc.TeamSlug)
					}
					fmt.Printf("  %d. %s (%s)%s\n", i+1, svc.Name, svc.ID, teamInfo)
				}

				// TODO: Interactive selection
				return fmt.Errorf("interactive service selection not yet implemented. Please use --service flag")
			}

			// Validate that the service ID exists in the fetched services
			var matchedService *auth.UserService
			for i := range services {
				if services[i].ID == serviceID || services[i].Name == serviceID {
					matchedService = &services[i]
					if services[i].Name == serviceID {
						serviceID = services[i].ID // Use ID if name was provided
					}
					break
				}
			}
			if matchedService == nil {
				return fmt.Errorf("service '%s' not found in your available services", serviceID)
			}

			// Save link to local project
			if err := auth.SaveServiceLink(serviceID); err != nil {
				return fmt.Errorf("failed to save service link: %w", err)
			}

			if c.Bool("json") {
				output := map[string]interface{}{
					"success":    true,
					"service_id": serviceID,
				}
				if matchedService != nil {
					output["service_name"] = matchedService.Name
					if matchedService.TeamSlug != "" {
						output["team"] = matchedService.TeamSlug
					}
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("✓ Linked to service %s\n", serviceID)
			if matchedService != nil {
				fmt.Printf("  Service name: %s\n", matchedService.Name)
				if matchedService.TeamSlug != "" {
					fmt.Printf("  Team: %s\n", matchedService.TeamSlug)
				}
			}

			return nil
		},
	}
}
