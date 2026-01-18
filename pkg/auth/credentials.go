package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// IsValid checks if credentials are still valid (not expired)
func IsValid(creds *Credentials) bool {
	if creds == nil {
		return false
	}

	// Service tokens (from USER_TOKEN or CS_TOKEN env) have no ExpiresAt set
	// They are valid as long as the token string exists
	if creds.SessionToken != "" && creds.ExpiresAt.IsZero() && creds.Email == "" {
		// This is likely a service token - consider valid
		// The backend will validate the actual JWT expiration
		return true
	}

	// Check if credentials have expired
	if !creds.ExpiresAt.IsZero() && time.Now().After(creds.ExpiresAt) {
		return false
	}

	// Check if Vault token has expired
	if creds.Vault.ExpiresAt > 0 && time.Now().Unix() > creds.Vault.ExpiresAt {
		return false
	}

	return true
}

// ValidateServiceID checks if a service ID has valid format
// Valid formats: svc_xxx, prj_integ_xxx, img_xxx
func ValidateServiceID(serviceID string) error {
	if serviceID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}

	// Match: svc_, prj_integ_, img_ followed by alphanumeric/hyphens
	pattern := `^(svc|prj_integ|img)_[a-zA-Z0-9-]+$`
	matched, err := regexp.MatchString(pattern, serviceID)
	if err != nil {
		return fmt.Errorf("failed to validate service ID: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid service ID format '%s': must start with svc_, prj_integ_, or img_", serviceID)
	}
	return nil
}

// SaveServiceLink saves a service ID link to the local project
func SaveServiceLink(serviceID string) error {
	if err := ValidateServiceID(serviceID); err != nil {
		return err
	}

	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}

	dir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	path := filepath.Join(dir, serviceLinksFile)

	data := map[string]string{"service_id": serviceID}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service link: %w", err)
	}

	return os.WriteFile(path, jsonData, 0600)
}

// LoadServiceLink loads the linked service ID from local project
func LoadServiceLink() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	path := filepath.Join(dir, serviceLinksFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no service linked: run 'cs link' first")
		}
		return "", fmt.Errorf("failed to read service link file: %w", err)
	}

	var linkData map[string]string
	if err := json.Unmarshal(data, &linkData); err != nil {
		return "", fmt.Errorf("failed to unmarshal service link: %w", err)
	}

	serviceID, ok := linkData["service_id"]
	if !ok || serviceID == "" {
		return "", fmt.Errorf("invalid service link file: missing service_id")
	}

	if err := ValidateServiceID(serviceID); err != nil {
		return "", fmt.Errorf("corrupted service link: %w (run 'cs link' to fix)", err)
	}

	return serviceID, nil
}

// DeleteServiceLink removes the service link
func DeleteServiceLink() error {
	dir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	path := filepath.Join(dir, serviceLinksFile)

	// Check if file exists before attempting to delete
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, which is fine for delete operation
			return nil
		}
		return fmt.Errorf("failed to check service link file: %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove service link file: %w", err)
	}

	return nil
}

// NeedsRefresh checks if credentials will expire soon and should be refreshed
func NeedsRefresh(creds *Credentials) bool {
	if creds == nil {
		return false
	}

	// Check if already expired
	if !IsValid(creds) {
		return true
	}

	// Refresh if expires within 5 minutes
	refreshThreshold := 5 * time.Minute

	// Check main credential expiration
	if !creds.ExpiresAt.IsZero() && time.Until(creds.ExpiresAt) < refreshThreshold {
		return true
	}

	// Check Vault token expiration
	if creds.Vault.ExpiresAt > 0 {
		vaultExpiry := time.Unix(creds.Vault.ExpiresAt, 0)
		if time.Until(vaultExpiry) < refreshThreshold {
			return true
		}
	}

	return false
}

// GetExpirationTime returns the soonest expiration time from the credentials
func GetExpirationTime(creds *Credentials) *time.Time {
	if creds == nil {
		return nil
	}

	var soonest *time.Time

	// Check main credential expiration
	if !creds.ExpiresAt.IsZero() {
		soonest = &creds.ExpiresAt
	}

	// Check Vault token expiration
	if creds.Vault.ExpiresAt > 0 {
		vaultExpiry := time.Unix(creds.Vault.ExpiresAt, 0)
		if soonest == nil || vaultExpiry.Before(*soonest) {
			soonest = &vaultExpiry
		}
	}

	return soonest
}

// TimeUntilExpiration returns the duration until credentials expire
func TimeUntilExpiration(creds *Credentials) time.Duration {
	expiry := GetExpirationTime(creds)
	if expiry == nil {
		// No expiration set, return max duration
		return time.Duration(1<<63 - 1)
	}

	duration := time.Until(*expiry)
	if duration < 0 {
		return 0
	}

	return duration
}

// FindServiceByID finds a service link by its ID
func FindServiceByID(creds *Credentials, serviceID string) *ServiceLink {
	if creds == nil || serviceID == "" {
		return nil
	}

	for i := range creds.Services {
		if creds.Services[i].ServiceID == serviceID {
			return &creds.Services[i]
		}
	}

	return nil
}

// FindServiceByName finds a service link by its name
func FindServiceByName(creds *Credentials, serviceName string) *ServiceLink {
	if creds == nil || serviceName == "" {
		return nil
	}

	for i := range creds.Services {
		if creds.Services[i].ServiceName == serviceName {
			return &creds.Services[i]
		}
	}

	return nil
}

// DecodeJWTClaims decodes claims from a JWT without signature verification
// Used for extracting service token info like team_slug
func DecodeJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try without padding
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	return claims, nil
}

// GetTeamFromToken extracts team_slug from USER_TOKEN or CS_TOKEN JWT claims
// Priority: USER_TOKEN > CS_TOKEN (deprecated)
// Returns empty string if no token or no team_slug claim
func GetTeamFromToken() string {
	// Check USER_TOKEN first (primary)
	token := os.Getenv(userTokenEnvVar)
	if token == "" {
		// Fall back to deprecated CS_TOKEN
		token = os.Getenv(deprecatedTokenEnvVar)
	}
	if token == "" {
		return ""
	}

	claims, err := DecodeJWTClaims(token)
	if err != nil {
		return ""
	}
	if teamSlug, ok := claims["team_slug"].(string); ok {
		return teamSlug
	}
	return ""
}

// GetTeamContext returns team context from explicit flag or token
// Priority: explicit flag > token claim > empty (personal)
func GetTeamContext(explicitTeam string) string {
	if explicitTeam != "" {
		return explicitTeam
	}
	return GetTeamFromToken()
}
