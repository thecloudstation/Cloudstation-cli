package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create test credentials with all fields populated
	testTime := time.Now().Truncate(time.Second) // Truncate for JSON round-trip comparison
	creds := &Credentials{
		SessionToken: "test_session_token_jwt",
		UserID:       789,
		UserUUID:     "test-uuid-abc-def",
		Email:        "test@example.com",
		Vault: VaultCreds{
			Address:   "https://vault.example.com",
			Token:     "vault_token_xyz",
			ExpiresAt: 1234567890,
		},
		Nomad: NomadCreds{
			Address:   "https://nomad.example.com",
			SecretID:  "nomad_secret_123",
			Namespace: "default",
			Token:     "nomad_token_456",
		},
		NATS: NATSCreds{
			URLs:     []string{"nats://localhost:4222", "nats://localhost:4223"},
			User:     "nats_user",
			Password: "nats_pass",
			JWT:      "nats_jwt_token",
			Seed:     "nats_seed_key",
		},
		Registry: RegistryCreds{
			URL:      "https://registry.example.com",
			Username: "registry_user",
			Password: "registry_pass",
		},
		Services: []ServiceLink{
			{
				ProjectPath: "/path/to/project1",
				ServiceID:   "svc_001",
				ServiceName: "api-service",
				CreatedAt:   testTime,
			},
			{
				ProjectPath: "/path/to/project2",
				ServiceID:   "svc_002",
				ServiceName: "web-service",
				CreatedAt:   testTime,
			},
		},
		CreatedAt: testTime,
		ExpiresAt: testTime.Add(24 * time.Hour),
	}

	// Save credentials
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Verify file was created
	configDir := filepath.Join(tmpHome, configDirName)
	credPath := filepath.Join(configDir, credentialsFile)
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Fatal("Credentials file was not created")
	}

	// Load credentials back
	loadedCreds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	// Verify all fields match
	if loadedCreds.SessionToken != creds.SessionToken {
		t.Errorf("SessionToken mismatch: got %q, want %q", loadedCreds.SessionToken, creds.SessionToken)
	}
	if loadedCreds.UserID != creds.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", loadedCreds.UserID, creds.UserID)
	}
	if loadedCreds.UserUUID != creds.UserUUID {
		t.Errorf("UserUUID mismatch: got %q, want %q", loadedCreds.UserUUID, creds.UserUUID)
	}
	if loadedCreds.Email != creds.Email {
		t.Errorf("Email mismatch: got %q, want %q", loadedCreds.Email, creds.Email)
	}

	// Verify Vault credentials
	if loadedCreds.Vault.Address != creds.Vault.Address {
		t.Errorf("Vault.Address mismatch: got %q, want %q", loadedCreds.Vault.Address, creds.Vault.Address)
	}
	if loadedCreds.Vault.Token != creds.Vault.Token {
		t.Errorf("Vault.Token mismatch: got %q, want %q", loadedCreds.Vault.Token, creds.Vault.Token)
	}
	if loadedCreds.Vault.ExpiresAt != creds.Vault.ExpiresAt {
		t.Errorf("Vault.ExpiresAt mismatch: got %d, want %d", loadedCreds.Vault.ExpiresAt, creds.Vault.ExpiresAt)
	}

	// Verify Nomad credentials
	if loadedCreds.Nomad.Address != creds.Nomad.Address {
		t.Errorf("Nomad.Address mismatch: got %q, want %q", loadedCreds.Nomad.Address, creds.Nomad.Address)
	}
	if loadedCreds.Nomad.SecretID != creds.Nomad.SecretID {
		t.Errorf("Nomad.SecretID mismatch: got %q, want %q", loadedCreds.Nomad.SecretID, creds.Nomad.SecretID)
	}
	if loadedCreds.Nomad.Namespace != creds.Nomad.Namespace {
		t.Errorf("Nomad.Namespace mismatch: got %q, want %q", loadedCreds.Nomad.Namespace, creds.Nomad.Namespace)
	}
	if loadedCreds.Nomad.Token != creds.Nomad.Token {
		t.Errorf("Nomad.Token mismatch: got %q, want %q", loadedCreds.Nomad.Token, creds.Nomad.Token)
	}

	// Verify NATS credentials
	if len(loadedCreds.NATS.URLs) != len(creds.NATS.URLs) {
		t.Errorf("NATS.URLs length mismatch: got %d, want %d", len(loadedCreds.NATS.URLs), len(creds.NATS.URLs))
	} else {
		for i, url := range loadedCreds.NATS.URLs {
			if url != creds.NATS.URLs[i] {
				t.Errorf("NATS.URLs[%d] mismatch: got %q, want %q", i, url, creds.NATS.URLs[i])
			}
		}
	}
	if loadedCreds.NATS.User != creds.NATS.User {
		t.Errorf("NATS.User mismatch: got %q, want %q", loadedCreds.NATS.User, creds.NATS.User)
	}
	if loadedCreds.NATS.Password != creds.NATS.Password {
		t.Errorf("NATS.Password mismatch: got %q, want %q", loadedCreds.NATS.Password, creds.NATS.Password)
	}
	if loadedCreds.NATS.JWT != creds.NATS.JWT {
		t.Errorf("NATS.JWT mismatch: got %q, want %q", loadedCreds.NATS.JWT, creds.NATS.JWT)
	}
	if loadedCreds.NATS.Seed != creds.NATS.Seed {
		t.Errorf("NATS.Seed mismatch: got %q, want %q", loadedCreds.NATS.Seed, creds.NATS.Seed)
	}

	// Verify Registry credentials
	if loadedCreds.Registry.URL != creds.Registry.URL {
		t.Errorf("Registry.URL mismatch: got %q, want %q", loadedCreds.Registry.URL, creds.Registry.URL)
	}
	if loadedCreds.Registry.Username != creds.Registry.Username {
		t.Errorf("Registry.Username mismatch: got %q, want %q", loadedCreds.Registry.Username, creds.Registry.Username)
	}
	if loadedCreds.Registry.Password != creds.Registry.Password {
		t.Errorf("Registry.Password mismatch: got %q, want %q", loadedCreds.Registry.Password, creds.Registry.Password)
	}

	// Verify Services
	if len(loadedCreds.Services) != len(creds.Services) {
		t.Errorf("Services length mismatch: got %d, want %d", len(loadedCreds.Services), len(creds.Services))
	} else {
		for i, svc := range loadedCreds.Services {
			if svc.ProjectPath != creds.Services[i].ProjectPath {
				t.Errorf("Services[%d].ProjectPath mismatch: got %q, want %q", i, svc.ProjectPath, creds.Services[i].ProjectPath)
			}
			if svc.ServiceID != creds.Services[i].ServiceID {
				t.Errorf("Services[%d].ServiceID mismatch: got %q, want %q", i, svc.ServiceID, creds.Services[i].ServiceID)
			}
			if svc.ServiceName != creds.Services[i].ServiceName {
				t.Errorf("Services[%d].ServiceName mismatch: got %q, want %q", i, svc.ServiceName, creds.Services[i].ServiceName)
			}
			if !svc.CreatedAt.Equal(creds.Services[i].CreatedAt) {
				t.Errorf("Services[%d].CreatedAt mismatch: got %v, want %v", i, svc.CreatedAt, creds.Services[i].CreatedAt)
			}
		}
	}

	// Verify timestamps
	if !loadedCreds.CreatedAt.Equal(creds.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", loadedCreds.CreatedAt, creds.CreatedAt)
	}
	if !loadedCreds.ExpiresAt.Equal(creds.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", loadedCreds.ExpiresAt, creds.ExpiresAt)
	}
}

func TestLoadCredentials_NotLoggedIn(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Clear any environment tokens to test "not logged in" scenario
	originalUserToken := os.Getenv("USER_TOKEN")
	originalCSToken := os.Getenv("CS_TOKEN")
	os.Unsetenv("USER_TOKEN")
	os.Unsetenv("CS_TOKEN")
	defer func() {
		if originalUserToken != "" {
			os.Setenv("USER_TOKEN", originalUserToken)
		}
		if originalCSToken != "" {
			os.Setenv("CS_TOKEN", originalCSToken)
		}
	}()

	// Try to load credentials when file doesn't exist
	creds, err := LoadCredentials()

	// Should return nil credentials
	if creds != nil {
		t.Error("Expected nil credentials when not logged in")
	}

	// Should return "not logged in" error
	if err == nil {
		t.Fatal("Expected error when loading credentials without being logged in")
	}

	expectedError := "not logged in: run 'cs login' or set USER_TOKEN env var"
	if err.Error() != expectedError {
		t.Errorf("Error message mismatch: got %q, want %q", err.Error(), expectedError)
	}
}

func TestFilePermissions(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create and save test credentials
	creds := &Credentials{
		Email:        "permissions@test.com",
		UserID:       123,
		SessionToken: "secret_session_token",
		CreatedAt:    time.Now(),
	}

	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Check file permissions
	configDir := filepath.Join(tmpHome, configDirName)
	credPath := filepath.Join(configDir, credentialsFile)
	fileInfo, err := os.Stat(credPath)
	if err != nil {
		t.Fatalf("Failed to stat credentials file: %v", err)
	}

	// Get file mode (permissions)
	mode := fileInfo.Mode()
	perm := mode.Perm()

	// Should have 0600 permissions (owner read/write only)
	expectedPerm := os.FileMode(0600)
	if perm != expectedPerm {
		t.Errorf("File permissions mismatch: got %o, want %o", perm, expectedPerm)
	}
}

func TestDeleteCredentials(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Clear any environment tokens to test "not logged in" scenario after deletion
	originalUserToken := os.Getenv("USER_TOKEN")
	originalCSToken := os.Getenv("CS_TOKEN")
	os.Unsetenv("USER_TOKEN")
	os.Unsetenv("CS_TOKEN")
	defer func() {
		if originalUserToken != "" {
			os.Setenv("USER_TOKEN", originalUserToken)
		}
		if originalCSToken != "" {
			os.Setenv("CS_TOKEN", originalCSToken)
		}
	}()

	// Create and save test credentials
	creds := &Credentials{
		Email:        "delete@test.com",
		UserID:       456,
		SessionToken: "to_be_deleted_token",
		CreatedAt:    time.Now(),
	}

	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// Verify file exists
	configDir := filepath.Join(tmpHome, configDirName)
	credPath := filepath.Join(configDir, credentialsFile)
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Fatal("Credentials file should exist before deletion")
	}

	// Delete credentials
	err = DeleteCredentials()
	if err != nil {
		t.Fatalf("DeleteCredentials failed: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(credPath); !os.IsNotExist(err) {
		t.Error("Credentials file should not exist after deletion")
	}

	// Try to load credentials after deletion
	loadedCreds, err := LoadCredentials()
	if loadedCreds != nil {
		t.Error("Should not be able to load credentials after deletion")
	}
	if err == nil || err.Error() != "not logged in: run 'cs login' or set USER_TOKEN env var" {
		t.Errorf("Expected 'not logged in' error after deletion, got: %v", err)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	configDir := filepath.Join(tmpHome, configDirName)

	// Ensure directory doesn't exist initially
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal("Config directory should not exist initially")
	}

	// Create config directory
	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Verify directory was created
	dirInfo, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}

	if !dirInfo.IsDir() {
		t.Error("Config path should be a directory")
	}

	// Check directory permissions (should be 0700)
	mode := dirInfo.Mode()
	perm := mode.Perm()
	expectedPerm := os.FileMode(0700)
	if perm != expectedPerm {
		t.Errorf("Directory permissions mismatch: got %o, want %o", perm, expectedPerm)
	}

	// Call EnsureConfigDir again (should be idempotent)
	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir failed on second call: %v", err)
	}

	// Directory should still exist with same permissions
	dirInfo2, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config directory after second call: %v", err)
	}

	if dirInfo2.Mode().Perm() != expectedPerm {
		t.Error("Directory permissions changed after second EnsureConfigDir call")
	}
}

func TestSaveCredentials_InvalidJSON(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create config directory first
	configDir := filepath.Join(tmpHome, configDirName)
	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write invalid JSON to credentials file
	credPath := filepath.Join(configDir, credentialsFile)
	invalidJSON := []byte("{ this is not valid json }")
	err = os.WriteFile(credPath, invalidJSON, 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Try to load credentials with invalid JSON
	creds, err := LoadCredentials()

	// Should return error
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}

	// Should return nil credentials
	if creds != nil {
		t.Error("Expected nil credentials when JSON is invalid")
	}

	// Verify it's a JSON unmarshal error
	if _, ok := err.(*json.SyntaxError); !ok {
		// Could also be json.UnmarshalTypeError
		if _, ok := err.(*json.UnmarshalTypeError); !ok {
			t.Errorf("Expected JSON unmarshal error, got: %T", err)
		}
	}
}

func TestDeleteCredentials_FileNotExists(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Try to delete credentials when file doesn't exist
	err := DeleteCredentials()

	// Should return an error (file not found)
	if err == nil {
		t.Error("Expected error when deleting non-existent credentials file")
	}

	// Verify it's a "no such file" error
	if !os.IsNotExist(err) {
		t.Errorf("Expected 'file not exist' error, got: %v", err)
	}
}

func TestSaveCredentials_MinimalData(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Save minimal credentials (only required fields)
	creds := &Credentials{
		Email: "minimal@test.com",
	}

	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials failed with minimal data: %v", err)
	}

	// Load and verify
	loadedCreds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if loadedCreds.Email != creds.Email {
		t.Errorf("Email mismatch: got %q, want %q", loadedCreds.Email, creds.Email)
	}

	// Verify empty fields remain empty
	if loadedCreds.SessionToken != "" {
		t.Error("SessionToken should be empty for minimal credentials")
	}
	if loadedCreds.UserID != 0 {
		t.Error("UserID should be 0 for minimal credentials")
	}
}

// TestTableDrivenErrors uses table-driven tests for error scenarios
func TestTableDrivenErrors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(tmpHome string) error
		operation   func() error
		wantErr     bool
		errCheck    func(error) bool
		description string
	}{
		{
			name: "LoadCredentials_FileNotExist",
			setup: func(tmpHome string) error {
				// No setup needed - file should not exist
				return nil
			},
			operation: func() error {
				_, err := LoadCredentials()
				return err
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "not logged in: run 'cs login' or set USER_TOKEN env var"
			},
			description: "Loading credentials when not logged in",
		},
		{
			name: "DeleteCredentials_FileNotExist",
			setup: func(tmpHome string) error {
				// No setup needed - file should not exist
				return nil
			},
			operation: func() error {
				return DeleteCredentials()
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return os.IsNotExist(err)
			},
			description: "Deleting non-existent credentials",
		},
		{
			name: "LoadCredentials_CorruptJSON",
			setup: func(tmpHome string) error {
				configDir := filepath.Join(tmpHome, configDirName)
				if err := os.MkdirAll(configDir, 0700); err != nil {
					return err
				}
				credPath := filepath.Join(configDir, credentialsFile)
				return os.WriteFile(credPath, []byte("{corrupt json"), 0600)
			},
			operation: func() error {
				_, err := LoadCredentials()
				return err
			},
			wantErr: true,
			errCheck: func(err error) bool {
				_, ok := err.(*json.SyntaxError)
				return ok
			},
			description: "Loading corrupt JSON credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary home directory
			tmpHome := t.TempDir()

			// Save original HOME and restore after test
			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpHome)
			defer os.Setenv("HOME", originalHome)

			// Clear any environment tokens to ensure test isolation
			originalUserToken := os.Getenv("USER_TOKEN")
			originalCSToken := os.Getenv("CS_TOKEN")
			os.Unsetenv("USER_TOKEN")
			os.Unsetenv("CS_TOKEN")
			defer func() {
				if originalUserToken != "" {
					os.Setenv("USER_TOKEN", originalUserToken)
				}
				if originalCSToken != "" {
					os.Setenv("CS_TOKEN", originalCSToken)
				}
			}()

			// Run setup if provided
			if tt.setup != nil {
				if err := tt.setup(tmpHome); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Run the operation
			err := tt.operation()

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: error = %v, wantErr %v", tt.description, err, tt.wantErr)
			}

			// Check specific error type/content if provided
			if err != nil && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("%s: unexpected error type/content: %v", tt.description, err)
			}
		})
	}
}

// TestConcurrentAccess tests concurrent read/write operations
func TestConcurrentAccess(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create initial credentials
	creds := &Credentials{
		Email:        "concurrent@test.com",
		UserID:       999,
		SessionToken: "concurrent_session_token",
	}

	// Save initial credentials
	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("Initial SaveCredentials failed: %v", err)
	}

	// Run concurrent reads (should all succeed)
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			loaded, err := LoadCredentials()
			if err != nil {
				t.Errorf("Concurrent load %d failed: %v", id, err)
				return
			}
			if loaded.Email != creds.Email {
				t.Errorf("Concurrent load %d got wrong email: %s", id, loaded.Email)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestLoadCredentials_CSTokenBackwardCompatibility verifies CS_TOKEN still works with deprecation warning
func TestLoadCredentials_CSTokenBackwardCompatibility(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Save and clear original tokens for test isolation
	originalUserToken := os.Getenv("USER_TOKEN")
	originalCSToken := os.Getenv("CS_TOKEN")
	os.Unsetenv("USER_TOKEN") // Clear USER_TOKEN so CS_TOKEN is used
	defer func() {
		if originalUserToken != "" {
			os.Setenv("USER_TOKEN", originalUserToken)
		} else {
			os.Unsetenv("USER_TOKEN")
		}
		if originalCSToken != "" {
			os.Setenv("CS_TOKEN", originalCSToken)
		} else {
			os.Unsetenv("CS_TOKEN")
		}
	}()

	// Set CS_TOKEN environment variable
	testToken := "test_cs_token_jwt"
	os.Setenv("CS_TOKEN", testToken)

	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials with CS_TOKEN failed: %v", err)
	}

	// Verify token was loaded
	if creds.SessionToken != testToken {
		t.Errorf("SessionToken mismatch: got %q, want %q", creds.SessionToken, testToken)
	}

	// Note: We can't easily test stderr output in unit tests without redirecting it
	// The deprecation warning will be tested in integration tests
}

// TestLoadCredentials_UserTokenPriority verifies USER_TOKEN takes priority over CS_TOKEN
func TestLoadCredentials_UserTokenPriority(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Save original tokens for restoration
	originalUserToken := os.Getenv("USER_TOKEN")
	originalCSToken := os.Getenv("CS_TOKEN")
	defer func() {
		if originalUserToken != "" {
			os.Setenv("USER_TOKEN", originalUserToken)
		} else {
			os.Unsetenv("USER_TOKEN")
		}
		if originalCSToken != "" {
			os.Setenv("CS_TOKEN", originalCSToken)
		} else {
			os.Unsetenv("CS_TOKEN")
		}
	}()

	// Set both tokens - USER_TOKEN should win
	userToken := "user_token_jwt"
	csToken := "cs_token_jwt"
	os.Setenv("USER_TOKEN", userToken)
	os.Setenv("CS_TOKEN", csToken)

	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials with both tokens failed: %v", err)
	}

	// Verify USER_TOKEN was used (not CS_TOKEN)
	if creds.SessionToken != userToken {
		t.Errorf("Expected USER_TOKEN to be used, got: %q", creds.SessionToken)
	}
}

// TestLoadCredentials_UserToken verifies USER_TOKEN works as primary token
func TestLoadCredentials_UserToken(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Save original tokens for restoration
	originalUserToken := os.Getenv("USER_TOKEN")
	originalCSToken := os.Getenv("CS_TOKEN")
	os.Unsetenv("CS_TOKEN") // Clear CS_TOKEN so only USER_TOKEN is set
	defer func() {
		if originalUserToken != "" {
			os.Setenv("USER_TOKEN", originalUserToken)
		} else {
			os.Unsetenv("USER_TOKEN")
		}
		if originalCSToken != "" {
			os.Setenv("CS_TOKEN", originalCSToken)
		} else {
			os.Unsetenv("CS_TOKEN")
		}
	}()

	// Set USER_TOKEN environment variable
	testToken := "test_user_token_jwt"
	os.Setenv("USER_TOKEN", testToken)

	// Load credentials
	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials with USER_TOKEN failed: %v", err)
	}

	// Verify token was loaded
	if creds.SessionToken != testToken {
		t.Errorf("SessionToken mismatch: got %q, want %q", creds.SessionToken, testToken)
	}

	// Verify it's recognized as a service token (no email, no expiration)
	if creds.Email != "" {
		t.Error("Service token should not have email")
	}
	if !creds.ExpiresAt.IsZero() {
		t.Error("Service token should not have ExpiresAt set in credentials")
	}
}

// TestSaveCredentials_LargeData tests saving credentials with large data
func TestSaveCredentials_LargeData(t *testing.T) {
	// Create temporary home directory for test isolation
	tmpHome := t.TempDir()

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create credentials with many services
	creds := &Credentials{
		Email:  "large@test.com",
		UserID: 1000,
	}

	// Add 100 service links
	for i := 0; i < 100; i++ {
		creds.Services = append(creds.Services, ServiceLink{
			ProjectPath: filepath.Join("/projects", "service", string(rune(i))),
			ServiceID:   string(rune(1000 + i)),
			ServiceName: "service-" + string(rune(i)),
			CreatedAt:   time.Now(),
		})
	}

	// Save credentials
	err := SaveCredentials(creds)
	if err != nil {
		t.Fatalf("SaveCredentials with large data failed: %v", err)
	}

	// Load and verify
	loadedCreds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	if len(loadedCreds.Services) != 100 {
		t.Errorf("Service count mismatch: got %d, want 100", len(loadedCreds.Services))
	}
}
