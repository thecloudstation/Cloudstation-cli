package secrets

import (
	"fmt"
	"sync"
)

var (
	// Global provider registry
	providers = make(map[string]Provider)
	mu        sync.RWMutex
)

// RegisterProvider registers a secret provider with the given name.
// If a provider with the same name already exists, it will be replaced.
// This function is typically called in init() functions of provider packages.
func RegisterProvider(name string, provider Provider) {
	mu.Lock()
	defer mu.Unlock()

	if name == "" {
		panic("cannot register provider with empty name")
	}
	if provider == nil {
		panic("cannot register nil provider")
	}

	providers[name] = provider
}

// GetProvider retrieves a registered provider by name.
// Returns ErrProviderNotFound if no provider with the given name is registered.
func GetProvider(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()

	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}

	return provider, nil
}

// ListProviders returns the names of all registered providers.
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// UnregisterProvider removes a provider from the registry.
// This is primarily useful for testing.
func UnregisterProvider(name string) {
	mu.Lock()
	defer mu.Unlock()

	delete(providers, name)
}

// ClearProviders removes all registered providers.
// This is primarily useful for testing.
func ClearProviders() {
	mu.Lock()
	defer mu.Unlock()

	providers = make(map[string]Provider)
}
