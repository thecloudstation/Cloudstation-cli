package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/auth"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/billing"
	"github.com/urfave/cli/v2"
)

// newBillingClient creates a billing client using JWT Bearer auth
func newBillingClient(baseURL string, creds *auth.Credentials) (*billing.Client, error) {
	if creds.SessionToken == "" {
		return nil, fmt.Errorf("not authenticated: run 'cs login' first")
	}
	return billing.NewClientWithToken(baseURL, creds.SessionToken), nil
}

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
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			client, err := newBillingClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			subscription, err := client.GetCurrentPlan()
			if err != nil {
				return fmt.Errorf("failed to get subscription status: %w", err)
			}

			trial, _ := client.GetTrialStatus()

			if c.Bool("json") {
				result := map[string]interface{}{
					"subscription": subscription,
					"trial":        trial,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Println("Subscription Status")
			fmt.Println(strings.Repeat("-", 40))
			fmt.Printf("Plan: %s\n", subscription.PlanName)
			fmt.Printf("Status: %s\n", subscription.Status)

			if subscription.NextBillingAt != nil {
				fmt.Printf("Next billing: %s\n", subscription.NextBillingAt.Format("2006-01-02"))
			}

			if trial != nil && trial.IsActive {
				fmt.Printf("\nTrial: %d days remaining\n", trial.DaysRemaining)
				if !trial.EndsAt.IsZero() {
					fmt.Printf("  Ends: %s\n", trial.EndsAt.Format("2006-01-02"))
				}
				if !trial.HasPayment {
					fmt.Println("  Warning: No payment method on file")
					fmt.Println("  Run 'cs billing add-card' to add one")
				} else {
					fmt.Println("  Payment method on file")
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
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			client, err := newBillingClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			plans, err := client.ListPlans()
			if err != nil {
				return fmt.Errorf("failed to list plans: %w", err)
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(plans, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Println("Available Plans")
			fmt.Println(strings.Repeat("=", 50))

			if len(plans) == 0 {
				fmt.Println("No plans available")
				return nil
			}

			for i, plan := range plans {
				if i > 0 {
					fmt.Println(strings.Repeat("-", 50))
				}
				fmt.Printf("\n%s", plan.Name)
				if plan.SubTitle != nil && *plan.SubTitle != "" {
					fmt.Printf(" - %s", *plan.SubTitle)
				}
				fmt.Println()

				fmt.Printf("  Code: %s\n", plan.Code)

				if plan.AmountCents > 0 {
					// Convert cents to dollars for display
					amount := float64(plan.AmountCents) / 100.0
					interval := plan.Interval
					if interval == "" {
						interval = "month"
					}
					fmt.Printf("  Price: $%.2f/%s\n", amount, interval)
				} else {
					fmt.Printf("  Price: Free / Custom\n")
				}

				// Display plan limits
				fmt.Println("  Includes:")
				if plan.MaxAppDeployments > 0 {
					fmt.Printf("    - Up to %d app deployments\n", plan.MaxAppDeployments)
				} else {
					fmt.Printf("    - Unlimited app deployments\n")
				}
				if plan.MaxTeamMembers > 0 {
					fmt.Printf("    - Up to %d team members\n", plan.MaxTeamMembers)
				}
				if plan.MaxDiskPerApp > 0 {
					// Convert MB to GB for display if large enough
					if plan.MaxDiskPerApp >= 1024 {
						fmt.Printf("    - %d GB disk per app\n", plan.MaxDiskPerApp/1024)
					} else {
						fmt.Printf("    - %d MB disk per app\n", plan.MaxDiskPerApp)
					}
				}

				if plan.Description != nil && *plan.Description != "" {
					fmt.Printf("  Description: %s\n", *plan.Description)
				}
			}
			fmt.Println()

			fmt.Println("To subscribe to a plan, run:")
			fmt.Println("  cs billing subscribe <plan-code>")

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
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return fmt.Errorf("plan code is required\n\nUsage: cs billing subscribe <plan-code>\n\nRun 'cs billing plans' to see available plans")
			}

			planCode := c.Args().First()

			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			client, err := newBillingClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			// Check if user has payment method via trial status
			trial, trialErr := client.GetTrialStatus()
			if trialErr == nil && !trial.HasPayment {
				if c.Bool("json") {
					result := map[string]interface{}{
						"success": false,
						"error":   "no payment method on file",
						"action":  "Run 'cs billing add-card' to add a payment method first",
					}
					data, _ := json.MarshalIndent(result, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				fmt.Println("Warning: No payment method on file")
				fmt.Println("\nPlease add a payment method before subscribing:")
				fmt.Println("  cs billing add-card")
				return nil
			}

			err = client.ChangePlan(planCode)
			if err != nil {
				return fmt.Errorf("failed to subscribe: %w", err)
			}

			if c.Bool("json") {
				result := map[string]interface{}{
					"success":   true,
					"plan_code": planCode,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Successfully subscribed to plan: %s\n", planCode)

			return nil
		},
	}
}

func billingInvoicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "invoices",
		Usage: "List billing invoices",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
			&cli.IntFlag{Name: "limit", Value: 10, Usage: "Maximum number of invoices to show"},
			&cli.IntFlag{Name: "page", Value: 1, Usage: "Page number"},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			client, err := newBillingClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			page := c.Int("page")
			limit := c.Int("limit")

			invoices, err := client.ListInvoices(page, limit)
			if err != nil {
				return fmt.Errorf("failed to list invoices: %w", err)
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(invoices, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Println("Billing Invoices")
			fmt.Println(strings.Repeat("=", 60))

			if len(invoices) == 0 {
				fmt.Println("No invoices found")
				return nil
			}

			for i, invoice := range invoices {
				if i > 0 {
					fmt.Println(strings.Repeat("-", 60))
				}

				// Format invoice number/ID
				identifier := invoice.Number
				if identifier == "" {
					identifier = invoice.ID
				}

				// Format payment status with symbol
				statusSymbol := ""
				switch invoice.PaymentStatus {
				case "succeeded", "paid":
					statusSymbol = "[paid]"
				case "pending":
					statusSymbol = "[pending]"
				case "failed":
					statusSymbol = "[failed]"
				default:
					statusSymbol = "[" + invoice.PaymentStatus + "]"
				}

				// Convert cents to dollars for display
				amount := float64(invoice.AmountCents) / 100.0

				fmt.Printf("\nInvoice: %s\n", identifier)
				fmt.Printf("  Amount: $%.2f %s\n", amount, invoice.Currency)
				fmt.Printf("  Status: %s %s\n", invoice.Status, statusSymbol)
				fmt.Printf("  Date: %s\n", invoice.IssuingDate.Format("2006-01-02"))

				if invoice.FileURL != "" {
					fmt.Printf("  PDF: %s\n", invoice.FileURL)
				}
			}
			fmt.Println()

			return nil
		},
	}
}

func billingAddCardCommand() *cli.Command {
	return &cli.Command{
		Name:  "add-card",
		Usage: "Add or update payment method",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
			&cli.StringFlag{
				Name:    "api-url",
				Value:   "https://cst-cs-backend-gmlyovvq.cloud-station.io",
				EnvVars: []string{"CS_API_URL"},
				Usage:   "CloudStation API URL",
			},
		},
		Action: func(c *cli.Context) error {
			creds, err := auth.LoadCredentials()
			if err != nil {
				return fmt.Errorf("not logged in: run 'cs login' first")
			}

			client, err := newBillingClient(c.String("api-url"), creds)
			if err != nil {
				return err
			}

			checkout, err := client.GetCheckoutURL()
			if err != nil {
				return fmt.Errorf("failed to get checkout URL: %w", err)
			}

			if c.Bool("json") {
				data, _ := json.MarshalIndent(checkout, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Println("Add Payment Method")
			fmt.Println(strings.Repeat("-", 40))
			fmt.Println("\nTo add or update your payment method, visit:")
			fmt.Printf("\n  %s\n\n", checkout.URL)

			if checkout.ExpiresAt != "" {
				fmt.Printf("This link expires: %s\n\n", checkout.ExpiresAt)
			}

			fmt.Println("After adding your payment method, you can subscribe to a plan:")
			fmt.Println("  cs billing plans      - View available plans")
			fmt.Println("  cs billing subscribe  - Subscribe to a plan")

			return nil
		},
	}
}
