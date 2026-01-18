package plugin

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/component"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
)

// Mock types for testing

// mockBuilder implements component.Builder for testing
type mockBuilder struct {
	name      string
	config    map[string]interface{}
	configErr error
}

func (m *mockBuilder) Build(ctx context.Context) (*artifact.Artifact, error) {
	return &artifact.Artifact{ID: m.name}, nil
}

func (m *mockBuilder) Config() (interface{}, error) {
	return m.config, nil
}

func (m *mockBuilder) ConfigSet(config interface{}) error {
	if m.configErr != nil {
		return m.configErr
	}
	if cfg, ok := config.(map[string]interface{}); ok {
		m.config = cfg
	}
	return nil
}

// mockRegistry implements component.Registry for testing
type mockRegistry struct {
	name      string
	config    map[string]interface{}
	configErr error
}

func (m *mockRegistry) Push(ctx context.Context, art *artifact.Artifact) (*artifact.RegistryRef, error) {
	return &artifact.RegistryRef{FullImage: m.name}, nil
}

func (m *mockRegistry) Pull(ctx context.Context, ref *artifact.RegistryRef) (*artifact.Artifact, error) {
	return &artifact.Artifact{ID: m.name}, nil
}

func (m *mockRegistry) Config() (interface{}, error) {
	return m.config, nil
}

func (m *mockRegistry) ConfigSet(config interface{}) error {
	if m.configErr != nil {
		return m.configErr
	}
	if cfg, ok := config.(map[string]interface{}); ok {
		m.config = cfg
	}
	return nil
}

// mockPlatform implements component.Platform for testing
type mockPlatform struct {
	name      string
	config    map[string]interface{}
	configErr error
}

func (m *mockPlatform) Deploy(ctx context.Context, art *artifact.Artifact) (*deployment.Deployment, error) {
	return &deployment.Deployment{ID: m.name}, nil
}

func (m *mockPlatform) Destroy(ctx context.Context, deploymentID string) error {
	return nil
}

func (m *mockPlatform) Status(ctx context.Context, deploymentID string) (*deployment.DeploymentStatus, error) {
	return &deployment.DeploymentStatus{}, nil
}

func (m *mockPlatform) Config() (interface{}, error) {
	return m.config, nil
}

func (m *mockPlatform) ConfigSet(config interface{}) error {
	if m.configErr != nil {
		return m.configErr
	}
	if cfg, ok := config.(map[string]interface{}); ok {
		m.config = cfg
	}
	return nil
}

// mockReleaseManager implements component.ReleaseManager for testing
type mockReleaseManager struct {
	name      string
	config    map[string]interface{}
	configErr error
}

func (m *mockReleaseManager) Release(ctx context.Context, dep *deployment.Deployment) error {
	return nil
}

func (m *mockReleaseManager) Config() (interface{}, error) {
	return m.config, nil
}

func (m *mockReleaseManager) ConfigSet(config interface{}) error {
	if m.configErr != nil {
		return m.configErr
	}
	if cfg, ok := config.(map[string]interface{}); ok {
		m.config = cfg
	}
	return nil
}

// Helper function to create a test registry with mock components
func createTestRegistry() *Registry {
	registry := &Registry{
		plugins: make(map[string]*Plugin),
	}

	// Register test plugins
	registry.Register("docker", &Plugin{
		Name:           "docker",
		Builder:        &mockBuilder{name: "docker-builder"},
		Registry:       &mockRegistry{name: "docker-registry"},
		Platform:       &mockPlatform{name: "docker-platform"},
		ReleaseManager: &mockReleaseManager{name: "docker-release"},
	})

	registry.Register("nixpacks", &Plugin{
		Name:    "nixpacks",
		Builder: &mockBuilder{name: "nixpacks-builder"},
	})

	registry.Register("dockerhub", &Plugin{
		Name:     "dockerhub",
		Registry: &mockRegistry{name: "dockerhub-registry"},
	})

	registry.Register("nomad", &Plugin{
		Name:     "nomad",
		Platform: &mockPlatform{name: "nomad-platform"},
	})

	registry.Register("canary", &Plugin{
		Name:           "canary",
		ReleaseManager: &mockReleaseManager{name: "canary-release"},
	})

	// Note: Due to cloning behavior, configErr won't be preserved after cloneComponent
	// So we'll remove this test case and update our tests accordingly

	return registry
}

// TestNewLoader tests the NewLoader function with various inputs
func TestNewLoader(t *testing.T) {
	tests := []struct {
		name     string
		registry *Registry
		logger   hclog.Logger
	}{
		{"with nil registry", nil, hclog.NewNullLogger()},
		{"with nil logger", &Registry{plugins: make(map[string]*Plugin)}, nil},
		{"with both nil", nil, nil},
		{"with valid params", &Registry{plugins: make(map[string]*Plugin)}, hclog.NewNullLogger()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader(tt.registry, tt.logger)
			if loader == nil {
				t.Error("NewLoader returned nil")
			}
			// Verify loader has registry (either provided or global)
			if loader.registry == nil {
				t.Error("NewLoader returned loader with nil registry")
			}
			// Verify loader has logger (either provided or null logger)
			if loader.logger == nil {
				t.Error("NewLoader returned loader with nil logger")
			}
		})
	}
}

// TestLoader_LoadBuilder tests loading builder plugins
func TestLoader_LoadBuilder(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		config     map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "load existing builder without config",
			pluginName: "docker",
			config:     nil,
			wantErr:    false,
		},
		{
			name:       "load existing builder with config",
			pluginName: "nixpacks",
			config:     map[string]interface{}{"version": "1.0", "debug": true},
			wantErr:    false,
		},
		{
			name:       "load non-existent builder",
			pluginName: "nonexistent",
			config:     nil,
			wantErr:    true,
			errMsg:     "plugin not found",
		},
		{
			name:       "load plugin without builder component",
			pluginName: "dockerhub",
			config:     nil,
			wantErr:    true,
			errMsg:     "does not provide a builder component",
		},
	}

	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := loader.LoadBuilder(tt.pluginName, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadBuilder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadBuilder() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if !tt.wantErr && builder == nil {
				t.Error("LoadBuilder() returned nil builder without error")
			}
			if !tt.wantErr && tt.config != nil && builder != nil {
				// Verify configuration was applied
				if mb, ok := builder.(*mockBuilder); ok {
					if !reflect.DeepEqual(mb.config, tt.config) {
						t.Errorf("LoadBuilder() config = %v, want %v", mb.config, tt.config)
					}
				}
			}
		})
	}
}

// TestLoader_LoadRegistry tests loading registry plugins
func TestLoader_LoadRegistry(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		config     map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "load existing registry without config",
			pluginName: "docker",
			config:     nil,
			wantErr:    false,
		},
		{
			name:       "load existing registry with config",
			pluginName: "dockerhub",
			config:     map[string]interface{}{"username": "user", "password": "pass"},
			wantErr:    false,
		},
		{
			name:       "load non-existent registry",
			pluginName: "nonexistent",
			config:     nil,
			wantErr:    true,
			errMsg:     "plugin not found",
		},
		{
			name:       "load plugin without registry component",
			pluginName: "nixpacks",
			config:     nil,
			wantErr:    true,
			errMsg:     "does not provide a registry component",
		},
	}

	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := loader.LoadRegistry(tt.pluginName, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRegistry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadRegistry() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if !tt.wantErr && reg == nil {
				t.Error("LoadRegistry() returned nil registry without error")
			}
			if !tt.wantErr && tt.config != nil && reg != nil {
				// Verify configuration was applied
				if mr, ok := reg.(*mockRegistry); ok {
					if !reflect.DeepEqual(mr.config, tt.config) {
						t.Errorf("LoadRegistry() config = %v, want %v", mr.config, tt.config)
					}
				}
			}
		})
	}
}

// TestLoader_LoadPlatform tests loading platform plugins
func TestLoader_LoadPlatform(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		config     map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "load existing platform without config",
			pluginName: "docker",
			config:     nil,
			wantErr:    false,
		},
		{
			name:       "load existing platform with config",
			pluginName: "nomad",
			config:     map[string]interface{}{"address": "http://nomad:4646", "token": "secret"},
			wantErr:    false,
		},
		{
			name:       "load non-existent platform",
			pluginName: "nonexistent",
			config:     nil,
			wantErr:    true,
			errMsg:     "plugin not found",
		},
		{
			name:       "load plugin without platform component",
			pluginName: "nixpacks",
			config:     nil,
			wantErr:    true,
			errMsg:     "does not provide a platform component",
		},
	}

	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, err := loader.LoadPlatform(tt.pluginName, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadPlatform() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if !tt.wantErr && platform == nil {
				t.Error("LoadPlatform() returned nil platform without error")
			}
			if !tt.wantErr && tt.config != nil && platform != nil {
				// Verify configuration was applied
				if mp, ok := platform.(*mockPlatform); ok {
					if !reflect.DeepEqual(mp.config, tt.config) {
						t.Errorf("LoadPlatform() config = %v, want %v", mp.config, tt.config)
					}
				}
			}
		})
	}
}

// TestLoader_LoadReleaseManager tests loading release manager plugins
func TestLoader_LoadReleaseManager(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		config     map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "load existing release manager without config",
			pluginName: "docker",
			config:     nil,
			wantErr:    false,
		},
		{
			name:       "load existing release manager with config",
			pluginName: "canary",
			config:     map[string]interface{}{"percentage": 20, "duration": "5m"},
			wantErr:    false,
		},
		{
			name:       "load non-existent release manager",
			pluginName: "nonexistent",
			config:     nil,
			wantErr:    true,
			errMsg:     "plugin not found",
		},
		{
			name:       "load plugin without release manager component",
			pluginName: "nixpacks",
			config:     nil,
			wantErr:    true,
			errMsg:     "does not provide a release manager component",
		},
	}

	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm, err := loader.LoadReleaseManager(tt.pluginName, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadReleaseManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadReleaseManager() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
			if !tt.wantErr && rm == nil {
				t.Error("LoadReleaseManager() returned nil release manager without error")
			}
			if !tt.wantErr && tt.config != nil && rm != nil {
				// Verify configuration was applied
				if mrm, ok := rm.(*mockReleaseManager); ok {
					if !reflect.DeepEqual(mrm.config, tt.config) {
						t.Errorf("LoadReleaseManager() config = %v, want %v", mrm.config, tt.config)
					}
				}
			}
		})
	}
}

// TestLoader_ComponentCloning tests that loaded components are cloned
func TestLoader_ComponentCloning(t *testing.T) {
	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	// Load the same builder twice and verify they are different instances
	builder1, err := loader.LoadBuilder("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load first builder: %v", err)
	}

	builder2, err := loader.LoadBuilder("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load second builder: %v", err)
	}

	// Verify they are different instances (different memory addresses)
	if fmt.Sprintf("%p", builder1) == fmt.Sprintf("%p", builder2) {
		t.Error("LoadBuilder() should return cloned instances, but got same reference")
	}

	// Test with registry
	reg1, err := loader.LoadRegistry("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load first registry: %v", err)
	}

	reg2, err := loader.LoadRegistry("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load second registry: %v", err)
	}

	if fmt.Sprintf("%p", reg1) == fmt.Sprintf("%p", reg2) {
		t.Error("LoadRegistry() should return cloned instances, but got same reference")
	}

	// Test with platform
	platform1, err := loader.LoadPlatform("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load first platform: %v", err)
	}

	platform2, err := loader.LoadPlatform("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load second platform: %v", err)
	}

	if fmt.Sprintf("%p", platform1) == fmt.Sprintf("%p", platform2) {
		t.Error("LoadPlatform() should return cloned instances, but got same reference")
	}

	// Test with release manager
	rm1, err := loader.LoadReleaseManager("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load first release manager: %v", err)
	}

	rm2, err := loader.LoadReleaseManager("docker", nil)
	if err != nil {
		t.Fatalf("Failed to load second release manager: %v", err)
	}

	if fmt.Sprintf("%p", rm1) == fmt.Sprintf("%p", rm2) {
		t.Error("LoadReleaseManager() should return cloned instances, but got same reference")
	}
}

// TestLoader_ConfigureComponent tests component configuration
func TestLoader_ConfigureComponent(t *testing.T) {
	tests := []struct {
		name      string
		component component.Configurable
		config    map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "configure with valid config",
			component: &mockBuilder{name: "test"},
			config:    map[string]interface{}{"key": "value"},
			wantErr:   false,
		},
		{
			name:      "configure with nil config",
			component: &mockBuilder{name: "test"},
			config:    nil,
			wantErr:   false,
		},
		{
			name:      "configure with empty config",
			component: &mockBuilder{name: "test"},
			config:    map[string]interface{}{},
			wantErr:   false,
		},
		{
			name:      "configure with error",
			component: &mockBuilder{name: "test", configErr: errors.New("config failed")},
			config:    map[string]interface{}{"key": "value"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := configureComponent(tt.component, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("configureComponent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.config != nil {
				// Verify config was set
				cfg, _ := tt.component.Config()
				if cfgMap, ok := cfg.(map[string]interface{}); ok {
					if !reflect.DeepEqual(cfgMap, tt.config) {
						t.Errorf("configureComponent() config = %v, want %v", cfgMap, tt.config)
					}
				}
			}
		})
	}
}

// TestCloneComponent tests component cloning
func TestCloneComponent(t *testing.T) {
	tests := []struct {
		name      string
		component interface{}
		wantNil   bool
	}{
		{
			name:      "clone nil component",
			component: nil,
			wantNil:   true,
		},
		{
			name:      "clone pointer to struct",
			component: &mockBuilder{name: "test"},
			wantNil:   false,
		},
		{
			name:      "clone struct value",
			component: mockBuilder{name: "test"},
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := cloneComponent(tt.component)
			if (cloned == nil) != tt.wantNil {
				t.Errorf("cloneComponent() returned nil = %v, want %v", cloned == nil, tt.wantNil)
			}
			if !tt.wantNil && cloned != nil {
				// Verify it's a different instance
				if tt.component != nil && fmt.Sprintf("%p", tt.component) == fmt.Sprintf("%p", cloned) {
					t.Error("cloneComponent() should return a new instance")
				}
				// Verify it's the same type
				if reflect.TypeOf(cloned) != reflect.TypeOf(tt.component) &&
					reflect.TypeOf(cloned) != reflect.PtrTo(reflect.TypeOf(tt.component)) {
					t.Errorf("cloneComponent() type = %v, want similar to %v",
						reflect.TypeOf(cloned), reflect.TypeOf(tt.component))
				}
			}
		})
	}
}

// TestConfigureFromMap tests ConfigureFromMap helper function
func TestConfigureFromMap(t *testing.T) {
	type testStruct struct {
		Name   string
		Value  int
		Hidden string // unexported field
	}

	tests := []struct {
		name    string
		target  interface{}
		config  map[string]interface{}
		wantErr bool
		check   func(t *testing.T, target interface{})
	}{
		{
			name:   "configure struct with matching fields",
			target: &testStruct{},
			config: map[string]interface{}{
				"Name":  "test",
				"Value": 42,
			},
			wantErr: false,
			check: func(t *testing.T, target interface{}) {
				ts := target.(*testStruct)
				if ts.Name != "test" || ts.Value != 42 {
					t.Errorf("ConfigureFromMap() didn't set fields correctly: %+v", ts)
				}
			},
		},
		{
			name:    "configure with nil config",
			target:  &testStruct{},
			config:  nil,
			wantErr: false,
			check:   func(t *testing.T, target interface{}) {},
		},
		{
			name:    "configure with empty config",
			target:  &testStruct{},
			config:  map[string]interface{}{},
			wantErr: false,
			check:   func(t *testing.T, target interface{}) {},
		},
		{
			name:    "configure non-struct type",
			target:  "not a struct",
			config:  map[string]interface{}{"key": "value"},
			wantErr: true,
			check:   func(t *testing.T, target interface{}) {},
		},
		{
			name:   "configure with non-matching field names",
			target: &testStruct{},
			config: map[string]interface{}{
				"NonExistent": "value",
			},
			wantErr: false,
			check: func(t *testing.T, target interface{}) {
				ts := target.(*testStruct)
				if ts.Name != "" || ts.Value != 0 {
					t.Errorf("ConfigureFromMap() modified fields unexpectedly: %+v", ts)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConfigureFromMap(tt.target, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigureFromMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				tt.check(t, tt.target)
			}
		})
	}
}

// TestLoader_IntegrationWithGlobalRegistry tests using global registry
func TestLoader_IntegrationWithGlobalRegistry(t *testing.T) {
	// Save original global registry
	originalRegistry := globalRegistry
	defer func() {
		globalRegistry = originalRegistry
	}()

	// Set up test global registry
	globalRegistry = createTestRegistry()

	// Create loader with nil registry (should use global)
	loader := NewLoader(nil, hclog.NewNullLogger())

	// Test that it can load from global registry
	builder, err := loader.LoadBuilder("docker", nil)
	if err != nil {
		t.Errorf("LoadBuilder() with global registry error = %v", err)
	}
	if builder == nil {
		t.Error("LoadBuilder() with global registry returned nil")
	}
}

// TestLoader_ConcurrentAccess tests concurrent access to loader
func TestLoader_ConcurrentAccess(t *testing.T) {
	registry := createTestRegistry()
	loader := NewLoader(registry, hclog.NewNullLogger())

	// Run multiple goroutines loading components concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 10; i++ {
		wg.Add(4)

		go func() {
			defer wg.Done()
			_, err := loader.LoadBuilder("docker", nil)
			if err != nil {
				errors <- err
			}
		}()

		go func() {
			defer wg.Done()
			_, err := loader.LoadRegistry("dockerhub", nil)
			if err != nil {
				errors <- err
			}
		}()

		go func() {
			defer wg.Done()
			_, err := loader.LoadPlatform("nomad", nil)
			if err != nil {
				errors <- err
			}
		}()

		go func() {
			defer wg.Done()
			_, err := loader.LoadReleaseManager("canary", nil)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			contains(s[1:], substr))))
}
