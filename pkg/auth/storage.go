package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName         = ".cloudstation"
	credentialsFile       = "credentials.json"
	serviceLinksFile      = "links.json"
	userTokenEnvVar       = "USER_TOKEN" // Primary token environment variable
	deprecatedTokenEnvVar = "CS_TOKEN"   // Deprecated, maintained for backward compatibility
)

// GetConfigDir returns the path to the CloudStation configuration directory (~/.cloudstation)
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName), nil
}

// EnsureConfigDir creates the CloudStation configuration directory if it doesn't exist
// Directory is created with 0700 permissions (owner read/write/execute only)
func EnsureConfigDir() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// SaveCredentials saves authentication credentials to the configuration file
// The credentials file is saved with 0600 permissions (owner read/write only)
func SaveCredentials(creds *Credentials) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	dir, _ := GetConfigDir()
	path := filepath.Join(dir, credentialsFile)

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// LoadCredentials reads authentication credentials from environment or config file
// Priority: USER_TOKEN env var > CS_TOKEN env var (deprecated) > ~/.cloudstation/credentials.json
// Returns a user-friendly error message if the user is not logged in
func LoadCredentials() (*Credentials, error) {
	// Check for primary service token (USER_TOKEN)
	if token := os.Getenv(userTokenEnvVar); token != "" {
		return &Credentials{
			SessionToken: token,
			// Service tokens don't have expiration in credentials
			// The token itself contains expiration info
		}, nil
	}

	// Check for deprecated service token (CS_TOKEN) with warning
	if token := os.Getenv(deprecatedTokenEnvVar); token != "" {
		fmt.Fprintln(os.Stderr, "Warning: CS_TOKEN is deprecated, please use USER_TOKEN instead")
		return &Credentials{
			SessionToken: token,
			// Service tokens don't have expiration in credentials
			// The token itself contains expiration info
		}, nil
	}

	// Fall back to file-based credentials
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, credentialsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in: run 'cs login' or set USER_TOKEN env var")
		}
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// DeleteCredentials removes the credentials file from the configuration directory
// This is used when logging out from the CloudStation CLI
func DeleteCredentials() error {
	dir, _ := GetConfigDir()
	path := filepath.Join(dir, credentialsFile)
	return os.Remove(path)
}
