package plugin

import (
	"fmt"
	"sync"
	"testing"
)

// TestRegistry_Register tests the Register method
// Note: We're reusing the mock types from loader_test.go
func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		plugin     *Plugin
		want       bool // should be retrievable after
	}{
		{
			name:       "valid plugin",
			pluginName: "test",
			plugin: &Plugin{
				Name:     "original",
				Builder:  &mockBuilder{name: "test-builder"},
				Registry: &mockRegistry{name: "test-registry"},
			},
			want: true,
		},
		{
			name:       "nil plugin",
			pluginName: "nil-test",
			plugin:     nil,
			want:       false,
		},
		{
			name:       "empty plugin",
			pluginName: "empty",
			plugin:     &Plugin{},
			want:       true,
		},
		{
			name:       "plugin with only builder",
			pluginName: "builder-only",
			plugin: &Plugin{
				Builder: &mockBuilder{name: "builder"},
			},
			want: true,
		},
		{
			name:       "plugin with only registry",
			pluginName: "registry-only",
			plugin: &Plugin{
				Registry: &mockRegistry{name: "registry"},
			},
			want: true,
		},
		{
			name:       "plugin with only platform",
			pluginName: "platform-only",
			plugin: &Plugin{
				Platform: &mockPlatform{name: "platform"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}
			reg.Register(tt.pluginName, tt.plugin)

			retrieved, err := reg.Get(tt.pluginName)
			got := err == nil

			if got != tt.want {
				t.Errorf("Register() retrievable = %v, want %v", got, tt.want)
			}

			if tt.want && retrieved != nil {
				// Verify that the name was set correctly
				if retrieved.Name != tt.pluginName {
					t.Errorf("Register() plugin.Name = %v, want %v", retrieved.Name, tt.pluginName)
				}
			}
		})
	}
}

// TestRegistry_Get tests the Get method
func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name        string
		setupPlugin *Plugin
		queryName   string
		wantErr     bool
	}{
		{
			name: "existing plugin",
			setupPlugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "test-builder"},
			},
			queryName: "test",
			wantErr:   false,
		},
		{
			name:        "non-existent plugin",
			setupPlugin: nil,
			queryName:   "non-existent",
			wantErr:     true,
		},
		{
			name: "empty name query",
			setupPlugin: &Plugin{
				Name: "test",
			},
			queryName: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			// Setup registry with plugin if provided
			if tt.setupPlugin != nil {
				reg.Register("test", tt.setupPlugin)
			}

			plugin, err := reg.Get(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && plugin == nil {
				t.Error("Get() returned nil plugin for existing entry")
			}

			if tt.wantErr && plugin != nil {
				t.Error("Get() returned plugin when error was expected")
			}
		})
	}
}

// TestRegistry_List tests the List method
func TestRegistry_List(t *testing.T) {
	tests := []struct {
		name    string
		plugins map[string]*Plugin
		want    []string
	}{
		{
			name:    "empty registry",
			plugins: map[string]*Plugin{},
			want:    []string{},
		},
		{
			name: "single plugin",
			plugins: map[string]*Plugin{
				"plugin1": &Plugin{Name: "plugin1"},
			},
			want: []string{"plugin1"},
		},
		{
			name: "multiple plugins",
			plugins: map[string]*Plugin{
				"plugin1": &Plugin{Name: "plugin1"},
				"plugin2": &Plugin{Name: "plugin2"},
				"plugin3": &Plugin{Name: "plugin3"},
			},
			want: []string{"plugin1", "plugin2", "plugin3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			// Setup registry
			for name, plugin := range tt.plugins {
				reg.Register(name, plugin)
			}

			got := reg.List()

			if len(got) != len(tt.want) {
				t.Errorf("List() returned %d items, want %d", len(got), len(tt.want))
			}

			// Check that all expected names are present
			nameMap := make(map[string]bool)
			for _, name := range got {
				nameMap[name] = true
			}

			for _, wantName := range tt.want {
				if !nameMap[wantName] {
					t.Errorf("List() missing expected name: %s", wantName)
				}
			}
		})
	}
}

// TestRegistry_HasBuilder tests the HasBuilder method
func TestRegistry_HasBuilder(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		want      bool
	}{
		{
			name: "plugin with builder",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			want:      true,
		},
		{
			name: "plugin without builder",
			plugin: &Plugin{
				Name:     "test",
				Registry: &mockRegistry{name: "registry"},
			},
			queryName: "test",
			want:      false,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			want:      false,
		},
		{
			name: "plugin with nil builder",
			plugin: &Plugin{
				Name:    "test",
				Builder: nil,
			},
			queryName: "test",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			got := reg.HasBuilder(tt.queryName)

			if got != tt.want {
				t.Errorf("HasBuilder() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRegistry_HasRegistry tests the HasRegistry method
func TestRegistry_HasRegistry(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		want      bool
	}{
		{
			name: "plugin with registry",
			plugin: &Plugin{
				Name:     "test",
				Registry: &mockRegistry{name: "registry"},
			},
			queryName: "test",
			want:      true,
		},
		{
			name: "plugin without registry",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			want:      false,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			want:      false,
		},
		{
			name: "plugin with nil registry",
			plugin: &Plugin{
				Name:     "test",
				Registry: nil,
			},
			queryName: "test",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			got := reg.HasRegistry(tt.queryName)

			if got != tt.want {
				t.Errorf("HasRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRegistry_HasPlatform tests the HasPlatform method
func TestRegistry_HasPlatform(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		want      bool
	}{
		{
			name: "plugin with platform",
			plugin: &Plugin{
				Name:     "test",
				Platform: &mockPlatform{name: "platform"},
			},
			queryName: "test",
			want:      true,
		},
		{
			name: "plugin without platform",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			want:      false,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			want:      false,
		},
		{
			name: "plugin with nil platform",
			plugin: &Plugin{
				Name:     "test",
				Platform: nil,
			},
			queryName: "test",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			got := reg.HasPlatform(tt.queryName)

			if got != tt.want {
				t.Errorf("HasPlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRegistry_GetBuilder tests the GetBuilder method
func TestRegistry_GetBuilder(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "plugin with builder",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			wantErr:   false,
			wantNil:   false,
		},
		{
			name: "plugin without builder",
			plugin: &Plugin{
				Name:     "test",
				Registry: &mockRegistry{name: "registry"},
			},
			queryName: "test",
			wantErr:   true,
			wantNil:   true,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			wantErr:   true,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			builder, err := reg.GetBuilder(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBuilder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (builder == nil) != tt.wantNil {
				t.Errorf("GetBuilder() builder = %v, wantNil %v", builder, tt.wantNil)
			}

			// Verify the correct builder is returned
			if !tt.wantNil && builder != nil {
				if _, ok := builder.(*mockBuilder); !ok {
					t.Errorf("GetBuilder() returned wrong type: %T", builder)
				}
			}
		})
	}
}

// TestRegistry_GetRegistry tests the GetRegistry method
func TestRegistry_GetRegistry(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "plugin with registry",
			plugin: &Plugin{
				Name:     "test",
				Registry: &mockRegistry{name: "registry"},
			},
			queryName: "test",
			wantErr:   false,
			wantNil:   false,
		},
		{
			name: "plugin without registry",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			wantErr:   true,
			wantNil:   true,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			wantErr:   true,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			registry, err := reg.GetRegistry(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetRegistry() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (registry == nil) != tt.wantNil {
				t.Errorf("GetRegistry() registry = %v, wantNil %v", registry, tt.wantNil)
			}

			// Verify the correct registry is returned
			if !tt.wantNil && registry != nil {
				if _, ok := registry.(*mockRegistry); !ok {
					t.Errorf("GetRegistry() returned wrong type: %T", registry)
				}
			}
		})
	}
}

// TestRegistry_GetPlatform tests the GetPlatform method
func TestRegistry_GetPlatform(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "plugin with platform",
			plugin: &Plugin{
				Name:     "test",
				Platform: &mockPlatform{name: "platform"},
			},
			queryName: "test",
			wantErr:   false,
			wantNil:   false,
		},
		{
			name: "plugin without platform",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			wantErr:   true,
			wantNil:   true,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			wantErr:   true,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			platform, err := reg.GetPlatform(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlatform() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (platform == nil) != tt.wantNil {
				t.Errorf("GetPlatform() platform = %v, wantNil %v", platform, tt.wantNil)
			}

			// Verify the correct platform is returned
			if !tt.wantNil && platform != nil {
				if _, ok := platform.(*mockPlatform); !ok {
					t.Errorf("GetPlatform() returned wrong type: %T", platform)
				}
			}
		})
	}
}

// TestRegistry_GetReleaseManager tests the GetReleaseManager method
func TestRegistry_GetReleaseManager(t *testing.T) {
	tests := []struct {
		name      string
		plugin    *Plugin
		queryName string
		wantErr   bool
		wantNil   bool
	}{
		{
			name: "plugin with release manager",
			plugin: &Plugin{
				Name:           "test",
				ReleaseManager: &mockReleaseManager{name: "release-manager"},
			},
			queryName: "test",
			wantErr:   false,
			wantNil:   false,
		},
		{
			name: "plugin without release manager",
			plugin: &Plugin{
				Name:    "test",
				Builder: &mockBuilder{name: "builder"},
			},
			queryName: "test",
			wantErr:   true,
			wantNil:   true,
		},
		{
			name:      "non-existent plugin",
			plugin:    nil,
			queryName: "non-existent",
			wantErr:   true,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &Registry{plugins: make(map[string]*Plugin)}

			if tt.plugin != nil {
				reg.Register("test", tt.plugin)
			}

			releaseManager, err := reg.GetReleaseManager(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetReleaseManager() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (releaseManager == nil) != tt.wantNil {
				t.Errorf("GetReleaseManager() releaseManager = %v, wantNil %v", releaseManager, tt.wantNil)
			}

			// Verify the correct release manager is returned
			if !tt.wantNil && releaseManager != nil {
				if _, ok := releaseManager.(*mockReleaseManager); !ok {
					t.Errorf("GetReleaseManager() returned wrong type: %T", releaseManager)
				}
			}
		})
	}
}

// TestRegistry_ConcurrentAccess tests thread-safety with concurrent access
func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pluginName := fmt.Sprintf("plugin-%d", idx)
			plugin := &Plugin{
				Name:    pluginName,
				Builder: &mockBuilder{name: fmt.Sprintf("builder-%d", idx)},
			}
			reg.Register(pluginName, plugin)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pluginName := fmt.Sprintf("plugin-%d", idx)

			// Try various read operations
			reg.List()
			reg.Get(pluginName)
			reg.HasBuilder(pluginName)
			reg.HasRegistry(pluginName)
			reg.HasPlatform(pluginName)
			reg.GetBuilder(pluginName)
		}(i)
	}

	// Mixed concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				// Write operation
				pluginName := fmt.Sprintf("mixed-plugin-%d", idx)
				plugin := &Plugin{
					Name:     pluginName,
					Registry: &mockRegistry{name: fmt.Sprintf("registry-%d", idx)},
				}
				reg.Register(pluginName, plugin)
			} else {
				// Read operation
				pluginName := fmt.Sprintf("mixed-plugin-%d", idx-1)
				reg.Get(pluginName)
				reg.HasRegistry(pluginName)
				reg.GetRegistry(pluginName)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that all write operations succeeded
	names := reg.List()
	if len(names) < numGoroutines {
		t.Errorf("Expected at least %d plugins after concurrent writes, got %d", numGoroutines, len(names))
	}
}

// TestRegistry_ConcurrentReadsAndWrites tests concurrent reads during writes
func TestRegistry_ConcurrentReadsAndWrites(t *testing.T) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	// Pre-populate with some plugins
	for i := 0; i < 10; i++ {
		pluginName := fmt.Sprintf("initial-plugin-%d", i)
		plugin := &Plugin{
			Name:     pluginName,
			Builder:  &mockBuilder{name: fmt.Sprintf("builder-%d", i)},
			Registry: &mockRegistry{name: fmt.Sprintf("registry-%d", i)},
			Platform: &mockPlatform{name: fmt.Sprintf("platform-%d", i)},
		}
		reg.Register(pluginName, plugin)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Start continuous readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					// Perform various read operations
					names := reg.List()
					for _, name := range names {
						reg.Get(name)
						reg.HasBuilder(name)
						reg.HasRegistry(name)
						reg.HasPlatform(name)
						if reg.HasBuilder(name) {
							reg.GetBuilder(name)
						}
						if reg.HasRegistry(name) {
							reg.GetRegistry(name)
						}
						if reg.HasPlatform(name) {
							reg.GetPlatform(name)
						}
					}
				}
			}
		}(i)
	}

	// Perform writes while reads are ongoing
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pluginName := fmt.Sprintf("concurrent-plugin-%d", idx)
			plugin := &Plugin{
				Name:     pluginName,
				Builder:  &mockBuilder{name: fmt.Sprintf("builder-%d", idx)},
				Registry: &mockRegistry{name: fmt.Sprintf("registry-%d", idx)},
			}
			reg.Register(pluginName, plugin)
		}(i)
	}

	// Let readers run for a bit
	// Note: In a real test environment, we might use time.Sleep here
	// but for unit tests, we'll just stop immediately after writes complete

	// Signal readers to stop
	close(stop)

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify registry integrity
	names := reg.List()
	if len(names) < 60 { // 10 initial + 50 concurrent
		t.Errorf("Expected at least 60 plugins, got %d", len(names))
	}

	// Verify all plugins are intact
	for _, name := range names {
		plugin, err := reg.Get(name)
		if err != nil {
			t.Errorf("Failed to get plugin %s: %v", name, err)
		}
		if plugin == nil {
			t.Errorf("Plugin %s is nil", name)
		}
		if plugin != nil && plugin.Name != name {
			t.Errorf("Plugin name mismatch: got %s, want %s", plugin.Name, name)
		}
	}
}

// TestRegistry_OverwritePlugin tests that registering with the same name overwrites
func TestRegistry_OverwritePlugin(t *testing.T) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	// Register initial plugin
	plugin1 := &Plugin{
		Name:    "test",
		Builder: &mockBuilder{name: "builder1"},
	}
	reg.Register("test", plugin1)

	// Verify initial plugin
	retrieved1, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Failed to get initial plugin: %v", err)
	}
	if !reg.HasBuilder("test") {
		t.Error("Initial plugin should have builder")
	}
	if reg.HasRegistry("test") {
		t.Error("Initial plugin should not have registry")
	}

	// Overwrite with new plugin
	plugin2 := &Plugin{
		Name:     "test",
		Registry: &mockRegistry{name: "registry2"},
	}
	reg.Register("test", plugin2)

	// Verify overwritten plugin
	retrieved2, err := reg.Get("test")
	if err != nil {
		t.Fatalf("Failed to get overwritten plugin: %v", err)
	}
	if reg.HasBuilder("test") {
		t.Error("Overwritten plugin should not have builder")
	}
	if !reg.HasRegistry("test") {
		t.Error("Overwritten plugin should have registry")
	}

	// Verify it's actually a different plugin
	if retrieved1 == retrieved2 {
		t.Error("Overwritten plugin should be a different instance")
	}
}

// TestRegistry_GlobalFunctions tests the global registry functions
func TestRegistry_GlobalFunctions(t *testing.T) {
	// Note: These tests modify the global registry, so they might affect
	// other tests if run in parallel. In production, you might want to
	// save and restore the global registry state.

	t.Run("global Register and Get", func(t *testing.T) {
		plugin := &Plugin{
			Name:    "global-test",
			Builder: &mockBuilder{name: "global-builder"},
		}

		Register("global-test", plugin)

		retrieved, err := Get("global-test")
		if err != nil {
			t.Errorf("Failed to get global plugin: %v", err)
		}
		if retrieved == nil {
			t.Error("Retrieved plugin is nil")
		}
		if retrieved != nil && retrieved.Name != "global-test" {
			t.Errorf("Plugin name mismatch: got %s, want global-test", retrieved.Name)
		}
	})

	t.Run("global List", func(t *testing.T) {
		// Register another plugin
		Register("global-test-2", &Plugin{Name: "global-test-2"})

		names := List()
		found := false
		for _, name := range names {
			if name == "global-test-2" {
				found = true
				break
			}
		}
		if !found {
			t.Error("global-test-2 not found in List()")
		}
	})

	t.Run("global HasBuilder", func(t *testing.T) {
		if !HasBuilder("global-test") {
			t.Error("HasBuilder() should return true for global-test")
		}
		if HasBuilder("non-existent-global") {
			t.Error("HasBuilder() should return false for non-existent plugin")
		}
	})

	t.Run("global HasRegistry", func(t *testing.T) {
		Register("global-registry-test", &Plugin{
			Name:     "global-registry-test",
			Registry: &mockRegistry{name: "global-registry"},
		})

		if !HasRegistry("global-registry-test") {
			t.Error("HasRegistry() should return true for global-registry-test")
		}
		if HasRegistry("global-test") {
			t.Error("HasRegistry() should return false for plugin without registry")
		}
	})

	t.Run("global HasPlatform", func(t *testing.T) {
		Register("global-platform-test", &Plugin{
			Name:     "global-platform-test",
			Platform: &mockPlatform{name: "global-platform"},
		})

		if !HasPlatform("global-platform-test") {
			t.Error("HasPlatform() should return true for global-platform-test")
		}
		if HasPlatform("global-test") {
			t.Error("HasPlatform() should return false for plugin without platform")
		}
	})

	t.Run("global GetBuilder", func(t *testing.T) {
		builder, err := GetBuilder("global-test")
		if err != nil {
			t.Errorf("GetBuilder() error = %v", err)
		}
		if builder == nil {
			t.Error("GetBuilder() returned nil")
		}
		if _, ok := builder.(*mockBuilder); !ok {
			t.Errorf("GetBuilder() returned wrong type: %T", builder)
		}
	})

	t.Run("global GetRegistry", func(t *testing.T) {
		registry, err := GetRegistry("global-registry-test")
		if err != nil {
			t.Errorf("GetRegistry() error = %v", err)
		}
		if registry == nil {
			t.Error("GetRegistry() returned nil")
		}
		if _, ok := registry.(*mockRegistry); !ok {
			t.Errorf("GetRegistry() returned wrong type: %T", registry)
		}
	})

	t.Run("global GetPlatform", func(t *testing.T) {
		platform, err := GetPlatform("global-platform-test")
		if err != nil {
			t.Errorf("GetPlatform() error = %v", err)
		}
		if platform == nil {
			t.Error("GetPlatform() returned nil")
		}
		if _, ok := platform.(*mockPlatform); !ok {
			t.Errorf("GetPlatform() returned wrong type: %T", platform)
		}
	})

	t.Run("global GetReleaseManager", func(t *testing.T) {
		Register("global-release-test", &Plugin{
			Name:           "global-release-test",
			ReleaseManager: &mockReleaseManager{name: "global-release"},
		})

		releaseManager, err := GetReleaseManager("global-release-test")
		if err != nil {
			t.Errorf("GetReleaseManager() error = %v", err)
		}
		if releaseManager == nil {
			t.Error("GetReleaseManager() returned nil")
		}
		if _, ok := releaseManager.(*mockReleaseManager); !ok {
			t.Errorf("GetReleaseManager() returned wrong type: %T", releaseManager)
		}
	})
}

// Benchmarks for performance testing

func BenchmarkRegistry_Register(b *testing.B) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := &Plugin{
			Name:    fmt.Sprintf("plugin-%d", i),
			Builder: &mockBuilder{name: fmt.Sprintf("builder-%d", i)},
		}
		reg.Register(fmt.Sprintf("plugin-%d", i), plugin)
	}
}

func BenchmarkRegistry_Get(b *testing.B) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	// Pre-populate registry
	for i := 0; i < 1000; i++ {
		plugin := &Plugin{
			Name:    fmt.Sprintf("plugin-%d", i),
			Builder: &mockBuilder{name: fmt.Sprintf("builder-%d", i)},
		}
		reg.Register(fmt.Sprintf("plugin-%d", i), plugin)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Get(fmt.Sprintf("plugin-%d", i%1000))
	}
}

func BenchmarkRegistry_List(b *testing.B) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	// Pre-populate registry
	for i := 0; i < 100; i++ {
		plugin := &Plugin{
			Name:    fmt.Sprintf("plugin-%d", i),
			Builder: &mockBuilder{name: fmt.Sprintf("builder-%d", i)},
		}
		reg.Register(fmt.Sprintf("plugin-%d", i), plugin)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.List()
	}
}

func BenchmarkRegistry_ConcurrentGet(b *testing.B) {
	reg := &Registry{plugins: make(map[string]*Plugin)}

	// Pre-populate registry
	for i := 0; i < 100; i++ {
		plugin := &Plugin{
			Name:    fmt.Sprintf("plugin-%d", i),
			Builder: &mockBuilder{name: fmt.Sprintf("builder-%d", i)},
		}
		reg.Register(fmt.Sprintf("plugin-%d", i), plugin)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			reg.Get(fmt.Sprintf("plugin-%d", i%100))
			i++
		}
	})
}
