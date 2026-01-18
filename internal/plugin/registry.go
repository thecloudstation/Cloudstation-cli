package plugin

import (
	"fmt"
	"sync"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/component"
)

// Plugin represents a registered plugin with its components
type Plugin struct {
	// Name is the plugin name
	Name string

	// Builder is the builder component (optional)
	Builder component.Builder

	// Registry is the registry component (optional)
	Registry component.Registry

	// Platform is the platform component (optional)
	Platform component.Platform

	// ReleaseManager is the release manager component (optional)
	ReleaseManager component.ReleaseManager
}

// Registry manages the collection of registered plugins
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*Plugin
}

// Global registry instance
var globalRegistry = &Registry{
	plugins: make(map[string]*Plugin),
}

// Register registers a plugin in the global registry
func Register(name string, plugin *Plugin) {
	globalRegistry.Register(name, plugin)
}

// Get retrieves a plugin from the global registry
func Get(name string) (*Plugin, error) {
	return globalRegistry.Get(name)
}

// List returns all registered plugin names
func List() []string {
	return globalRegistry.List()
}

// HasBuilder checks if a builder plugin exists
func HasBuilder(name string) bool {
	return globalRegistry.HasBuilder(name)
}

// HasRegistry checks if a registry plugin exists
func HasRegistry(name string) bool {
	return globalRegistry.HasRegistry(name)
}

// HasPlatform checks if a platform plugin exists
func HasPlatform(name string) bool {
	return globalRegistry.HasPlatform(name)
}

// Register registers a plugin in the registry
func (r *Registry) Register(name string, plugin *Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plugin == nil {
		return
	}

	plugin.Name = name
	r.plugins[name] = plugin
}

// Get retrieves a plugin by name
func (r *Registry) Get(name string) (*Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return plugin, nil
}

// List returns all registered plugin names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	return names
}

// HasBuilder checks if a builder plugin exists
func (r *Registry) HasBuilder(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return exists && plugin.Builder != nil
}

// HasRegistry checks if a registry plugin exists
func (r *Registry) HasRegistry(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return exists && plugin.Registry != nil
}

// HasPlatform checks if a platform plugin exists
func (r *Registry) HasPlatform(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return exists && plugin.Platform != nil
}

// GetBuilder retrieves a builder component by plugin name
func (r *Registry) GetBuilder(name string) (component.Builder, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	if plugin.Builder == nil {
		return nil, fmt.Errorf("plugin %s does not provide a builder component", name)
	}

	return plugin.Builder, nil
}

// GetRegistry retrieves a registry component by plugin name
func (r *Registry) GetRegistry(name string) (component.Registry, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	if plugin.Registry == nil {
		return nil, fmt.Errorf("plugin %s does not provide a registry component", name)
	}

	return plugin.Registry, nil
}

// GetPlatform retrieves a platform component by plugin name
func (r *Registry) GetPlatform(name string) (component.Platform, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	if plugin.Platform == nil {
		return nil, fmt.Errorf("plugin %s does not provide a platform component", name)
	}

	return plugin.Platform, nil
}

// GetReleaseManager retrieves a release manager component by plugin name
func (r *Registry) GetReleaseManager(name string) (component.ReleaseManager, error) {
	plugin, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	if plugin.ReleaseManager == nil {
		return nil, fmt.Errorf("plugin %s does not provide a release manager component", name)
	}

	return plugin.ReleaseManager, nil
}

// Global convenience functions

// GetBuilder retrieves a builder from the global registry
func GetBuilder(name string) (component.Builder, error) {
	return globalRegistry.GetBuilder(name)
}

// GetRegistry retrieves a registry from the global registry
func GetRegistry(name string) (component.Registry, error) {
	return globalRegistry.GetRegistry(name)
}

// GetPlatform retrieves a platform from the global registry
func GetPlatform(name string) (component.Platform, error) {
	return globalRegistry.GetPlatform(name)
}

// GetReleaseManager retrieves a release manager from the global registry
func GetReleaseManager(name string) (component.ReleaseManager, error) {
	return globalRegistry.GetReleaseManager(name)
}
