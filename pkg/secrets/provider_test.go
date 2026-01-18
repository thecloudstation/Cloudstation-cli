package secrets

import (
	"context"
	"errors"
	"testing"
)

// MockProvider is a test implementation of the Provider interface
type MockProvider struct {
	name        string
	secrets     map[string]string
	err         error
	validateErr error
	shouldPanic bool
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) FetchSecrets(ctx context.Context, config ProviderConfig) (map[string]string, error) {
	if m.shouldPanic {
		panic("mock panic")
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.secrets, nil
}

func (m *MockProvider) ValidateConfig(config ProviderConfig) error {
	if m.validateErr != nil {
		return m.validateErr
	}
	return nil
}

func TestMockProvider(t *testing.T) {
	ctx := context.Background()
	config := ProviderConfig{"key": "value"}

	t.Run("returns expected secrets", func(t *testing.T) {
		expectedSecrets := map[string]string{
			"DB_PASSWORD": "secret123",
			"API_KEY":     "key456",
		}

		mock := &MockProvider{
			name:    "test",
			secrets: expectedSecrets,
		}

		secrets, err := mock.FetchSecrets(ctx, config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(secrets) != len(expectedSecrets) {
			t.Fatalf("expected %d secrets, got %d", len(expectedSecrets), len(secrets))
		}

		for key, expectedValue := range expectedSecrets {
			if value, ok := secrets[key]; !ok {
				t.Errorf("expected key %q not found", key)
			} else if value != expectedValue {
				t.Errorf("for key %q: expected %q, got %q", key, expectedValue, value)
			}
		}
	})

	t.Run("handles errors correctly", func(t *testing.T) {
		expectedErr := errors.New("fetch failed")
		mock := &MockProvider{
			name: "test",
			err:  expectedErr,
		}

		secrets, err := mock.FetchSecrets(ctx, config)
		if err != expectedErr {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if secrets != nil {
			t.Errorf("expected nil secrets on error, got %v", secrets)
		}
	})

	t.Run("returns nil secrets", func(t *testing.T) {
		mock := &MockProvider{
			name:    "test",
			secrets: nil,
		}

		secrets, err := mock.FetchSecrets(ctx, config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if secrets != nil {
			t.Errorf("expected nil secrets, got %v", secrets)
		}
	})

	t.Run("validation works", func(t *testing.T) {
		mock := &MockProvider{
			name: "test",
		}

		err := mock.ValidateConfig(config)
		if err != nil {
			t.Fatalf("expected no validation error, got %v", err)
		}
	})

	t.Run("validation fails", func(t *testing.T) {
		expectedErr := errors.New("invalid config")
		mock := &MockProvider{
			name:        "test",
			validateErr: expectedErr,
		}

		err := mock.ValidateConfig(config)
		if err != expectedErr {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestProviderRegistry(t *testing.T) {
	// Clear registry before tests
	ClearProviders()

	t.Run("register and get provider", func(t *testing.T) {
		mock := &MockProvider{name: "test1"}
		RegisterProvider("test1", mock)

		retrieved, err := GetProvider("test1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if retrieved != mock {
			t.Errorf("retrieved provider doesn't match registered provider")
		}
	})

	t.Run("get non-existent provider", func(t *testing.T) {
		_, err := GetProvider("nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent provider")
		}

		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("register replaces existing provider", func(t *testing.T) {
		mock1 := &MockProvider{name: "test2"}
		mock2 := &MockProvider{name: "test2"}

		RegisterProvider("test2", mock1)
		RegisterProvider("test2", mock2)

		retrieved, err := GetProvider("test2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if retrieved != mock2 {
			t.Errorf("expected second provider, got first")
		}
	})

	t.Run("list providers", func(t *testing.T) {
		ClearProviders()

		RegisterProvider("provider1", &MockProvider{name: "provider1"})
		RegisterProvider("provider2", &MockProvider{name: "provider2"})
		RegisterProvider("provider3", &MockProvider{name: "provider3"})

		names := ListProviders()
		if len(names) != 3 {
			t.Fatalf("expected 3 providers, got %d", len(names))
		}

		// Check all expected names are present
		expected := map[string]bool{
			"provider1": false,
			"provider2": false,
			"provider3": false,
		}

		for _, name := range names {
			if _, ok := expected[name]; ok {
				expected[name] = true
			} else {
				t.Errorf("unexpected provider name: %q", name)
			}
		}

		for name, found := range expected {
			if !found {
				t.Errorf("expected provider %q not found in list", name)
			}
		}
	})

	t.Run("unregister provider", func(t *testing.T) {
		ClearProviders()

		RegisterProvider("test3", &MockProvider{name: "test3"})
		UnregisterProvider("test3")

		_, err := GetProvider("test3")
		if err == nil {
			t.Fatal("expected error after unregistering provider")
		}
	})

	t.Run("clear providers", func(t *testing.T) {
		RegisterProvider("test4", &MockProvider{name: "test4"})
		RegisterProvider("test5", &MockProvider{name: "test5"})

		ClearProviders()

		names := ListProviders()
		if len(names) != 0 {
			t.Errorf("expected 0 providers after clear, got %d", len(names))
		}
	})

	t.Run("panic on empty name", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty name")
			}
		}()

		RegisterProvider("", &MockProvider{name: ""})
	})

	t.Run("panic on nil provider", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil provider")
			}
		}()

		RegisterProvider("test", nil)
	})
}

func TestProviderError(t *testing.T) {
	t.Run("error formatting", func(t *testing.T) {
		baseErr := errors.New("connection refused")
		providerErr := NewProviderError("vault", "authenticate", baseErr)

		expected := "vault provider: authenticate failed: connection refused"
		if providerErr.Error() != expected {
			t.Errorf("expected %q, got %q", expected, providerErr.Error())
		}
	})

	t.Run("error unwrapping", func(t *testing.T) {
		baseErr := errors.New("underlying error")
		providerErr := NewProviderError("vault", "fetch", baseErr)

		if !errors.Is(providerErr, baseErr) {
			t.Error("expected error to unwrap to base error")
		}
	})
}

func TestProviderConfig(t *testing.T) {
	t.Run("create config", func(t *testing.T) {
		config := ProviderConfig{
			"address":   "https://vault.example.com",
			"role_id":   "test-role",
			"secret_id": "test-secret",
		}

		if config["address"] != "https://vault.example.com" {
			t.Error("failed to get address from config")
		}
	})

	t.Run("empty config", func(t *testing.T) {
		config := ProviderConfig{}

		if len(config) != 0 {
			t.Error("expected empty config")
		}
	})
}
