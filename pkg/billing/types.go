// Package billing provides types and client for CloudStation billing API.
package billing

import "time"

// Plan represents an available subscription plan.
type Plan struct {
	ID                string  `json:"id"`
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	SubTitle          *string `json:"sub_title,omitempty"`
	Label             *string `json:"label,omitempty"`
	Description       *string `json:"description,omitempty"`
	AmountCents       int     `json:"amount_cents"`
	Interval          string  `json:"interval"`
	Type              string  `json:"type,omitempty"`
	Status            string  `json:"status,omitempty"`
	MaxAppDeployments int     `json:"max_app_deployments"`
	MaxTeamMembers    int     `json:"max_team_members"`
	MaxDiskPerApp     int     `json:"max_disk_per_app"`
}

// PlanDetails represents the plan information from the API
type PlanDetails struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	SubTitle     string `json:"sub_title"`
	Description  string `json:"description"`
	AmountCents  int    `json:"amount_cents"`
	Interval     string `json:"interval"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	IsEnterprise bool   `json:"is_enterprise"`
}

// SubscriptionDetails represents subscription status from the API
type SubscriptionDetails struct {
	Status     string `json:"status"`
	ExternalID string `json:"external_id"`
	StartedAt  string `json:"started_at"`
}

// PlanResponse is the full response from GET /plan
type PlanResponse struct {
	Plan         PlanDetails         `json:"plan"`
	Subscription SubscriptionDetails `json:"subscription"`
}

// Subscription represents a user's current subscription status (for CLI display)
type Subscription struct {
	PlanCode      string     `json:"plan_code"`
	PlanName      string     `json:"plan_name"`
	Status        string     `json:"status"`
	TrialEndsAt   *time.Time `json:"trial_ends_at,omitempty"`
	NextBillingAt *time.Time `json:"next_billing_at,omitempty"`
}

// TrialStatus represents the user's trial period status.
type TrialStatus struct {
	IsActive      bool      `json:"is_active"`
	DaysRemaining int       `json:"days_remaining"`
	EndsAt        time.Time `json:"ends_at"`
	HasPayment    bool      `json:"has_payment_method"`
	CanCommit     bool      `json:"can_commit"`
}

// Invoice represents a billing invoice.
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

// CheckoutURLResponse contains the Stripe checkout URL for adding a payment method.
type CheckoutURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at,omitempty"`
}
