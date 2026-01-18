// Package billing provides an HTTP client for interacting with CloudStation
// backend billing API endpoints. It supports operations for managing plans,
// subscriptions, invoices, and payment methods using JWT Bearer authentication.
package billing

import (
	"fmt"
	"time"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/httpclient"
)

// Client is the HTTP client for billing operations with the CloudStation backend API.
// It embeds BaseClient to leverage common HTTP functionality and uses JWT Bearer
// authentication for all billing API requests.
type Client struct {
	*httpclient.BaseClient
}

// NewClientWithToken creates a new billing client using JWT Bearer auth.
// Use this for Google OAuth authenticated users.
//
// Parameters:
//   - baseURL: The base URL of the CloudStation backend API
//   - sessionToken: JWT token from auth service
//
// Returns a configured Client ready for making billing API calls.
func NewClientWithToken(baseURL, sessionToken string) *Client {
	client := &Client{
		BaseClient: httpclient.NewBaseClient(baseURL, 30*time.Second),
	}
	client.SetHeader("Authorization", "Bearer "+sessionToken)
	return client
}

// GetCurrentPlan retrieves the user's current subscription details.
// It makes a GET request to the /plan endpoint.
//
// Returns the current Subscription or an error if the request fails.
func (c *Client) GetCurrentPlan() (*Subscription, error) {
	var resp PlanResponse
	if err := c.DoJSON("GET", "/plan", nil, &resp); err != nil {
		return nil, fmt.Errorf("get plan failed: %w", err)
	}
	// Map API response to CLI Subscription type
	return &Subscription{
		PlanCode: resp.Plan.Code,
		PlanName: resp.Plan.Name,
		Status:   resp.Subscription.Status,
	}, nil
}

// ListPlans retrieves all available subscription plans.
// It makes a GET request to the /plan/list endpoint.
//
// Returns a slice of Plan objects or an error if the request fails.
func (c *Client) ListPlans() ([]Plan, error) {
	var resp struct {
		Plans []Plan `json:"plans"`
	}
	if err := c.DoJSON("GET", "/plan/list", nil, &resp); err != nil {
		return nil, fmt.Errorf("list plans failed: %w", err)
	}
	return resp.Plans, nil
}

// GetTrialStatus retrieves the user's current trial status.
// It makes a GET request to the /trial/status endpoint.
//
// Returns the TrialStatus or an error if the request fails.
func (c *Client) GetTrialStatus() (*TrialStatus, error) {
	var resp TrialStatus
	if err := c.DoJSON("GET", "/trial/status", nil, &resp); err != nil {
		return nil, fmt.Errorf("get trial status failed: %w", err)
	}
	return &resp, nil
}

// ChangePlan changes the user's subscription to a different plan.
// It makes a POST request to the /plan/change endpoint.
//
// Parameters:
//   - planCode: The code of the plan to change to
//
// Returns nil on success or an error if the request fails.
func (c *Client) ChangePlan(planCode string) error {
	req := map[string]string{"plan_code": planCode}
	if err := c.DoJSON("POST", "/plan/change", req, nil); err != nil {
		return fmt.Errorf("change plan failed: %w", err)
	}
	return nil
}

// ListInvoices retrieves a paginated list of the user's invoices.
// It makes a GET request to the /invoices endpoint with pagination parameters.
//
// Parameters:
//   - page: The page number to retrieve (1-indexed)
//   - pageSize: The number of invoices per page
//
// Returns a slice of Invoice objects or an error if the request fails.
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

// GetCheckoutURL generates a Stripe checkout URL for adding a payment method.
// It makes a POST request to the /payment-methods/checkout-url endpoint.
//
// Returns a CheckoutURLResponse containing the checkout URL or an error if the request fails.
func (c *Client) GetCheckoutURL() (*CheckoutURLResponse, error) {
	var resp CheckoutURLResponse
	if err := c.DoJSON("POST", "/payment-methods/checkout-url", nil, &resp); err != nil {
		return nil, fmt.Errorf("get checkout URL failed: %w", err)
	}
	return &resp, nil
}
