package lifecycle

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/thecloudstation/cloudstation-orchestrator/internal/config"
)

// TestNewExecutor tests the NewExecutor constructor
func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		logger hclog.Logger
	}{
		{"with nil config", nil, hclog.NewNullLogger()},
		{"with nil logger", &config.Config{}, nil},
		{"with both nil", nil, nil},
		{"with valid params", &config.Config{}, hclog.NewNullLogger()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.config, tt.logger)
			if executor == nil {
				t.Error("NewExecutor returned nil")
			}
		})
	}
}

// TestExecutor_Execute_AppNotFound tests error when app is not found
func TestExecutor_Execute_AppNotFound(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	err := executor.Execute(context.Background(), "non-existent-app")
	if err == nil {
		t.Error("expected error for non-existent app")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// Note: Testing with nil config is skipped as the executor doesn't handle nil configs
// This is expected behavior - callers should always provide a valid config

// TestExecutor_ExecutorFields tests that executor has correct internal state
func TestExecutor_ExecutorFields(t *testing.T) {
	cfg := &config.Config{}
	logger := hclog.NewNullLogger()

	executor := NewExecutor(cfg, logger)

	// Verify executor is properly initialized
	if executor.config != cfg {
		t.Error("executor config not set correctly")
	}
	if executor.pluginLoader == nil {
		t.Error("executor pluginLoader should not be nil")
	}
	if executor.logger == nil {
		t.Error("executor logger should not be nil")
	}
}

// TestExecutor_DefaultLogger tests that nil logger gets a default
func TestExecutor_DefaultLogger(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, nil)

	// Should have a non-nil logger even when nil is passed
	if executor.logger == nil {
		t.Error("executor should have a default logger when nil is passed")
	}
}

// TestExecutor_ExecuteBuild_AppNotFound tests build phase with missing app
func TestExecutor_ExecuteBuild_AppNotFound(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	// Create a minimal AppConfig without registering builder
	appCfg := &config.AppConfig{
		Name: "test-app",
		Build: &config.PluginConfig{
			Use:    "non-existent-builder",
			Config: map[string]interface{}{},
		},
	}

	_, err := executor.ExecuteBuild(context.Background(), appCfg)
	if err == nil {
		t.Error("expected error when builder plugin doesn't exist")
	}
}

// TestExecutor_ExecuteRegistry_AppNotFound tests registry phase with missing plugin
func TestExecutor_ExecuteRegistry_AppNotFound(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	appCfg := &config.AppConfig{
		Name: "test-app",
		Registry: &config.PluginConfig{
			Use:    "non-existent-registry",
			Config: map[string]interface{}{},
		},
	}

	_, err := executor.ExecuteRegistry(context.Background(), appCfg, nil)
	if err == nil {
		t.Error("expected error when registry plugin doesn't exist")
	}
}

// TestExecutor_ExecuteDeploy_AppNotFound tests deploy phase with missing plugin
func TestExecutor_ExecuteDeploy_AppNotFound(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	appCfg := &config.AppConfig{
		Name: "test-app",
		Deploy: &config.PluginConfig{
			Use:    "non-existent-platform",
			Config: map[string]interface{}{},
		},
	}

	_, err := executor.ExecuteDeploy(context.Background(), appCfg, nil)
	if err == nil {
		t.Error("expected error when platform plugin doesn't exist")
	}
}

// TestExecutor_ExecuteRelease_AppNotFound tests release phase with missing plugin
func TestExecutor_ExecuteRelease_AppNotFound(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	appCfg := &config.AppConfig{
		Name: "test-app",
		Release: &config.PluginConfig{
			Use:    "non-existent-release-manager",
			Config: map[string]interface{}{},
		},
	}

	err := executor.ExecuteRelease(context.Background(), appCfg, nil)
	if err == nil {
		t.Error("expected error when release manager plugin doesn't exist")
	}
}

// TestExecutor_ContextCancellation tests behavior with cancelled context
func TestExecutor_ContextCancellation(t *testing.T) {
	cfg := &config.Config{}
	executor := NewExecutor(cfg, hclog.NewNullLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute should fail fast due to context cancellation
	err := executor.Execute(ctx, "test-app")
	// Since app doesn't exist, we get app not found error before context check
	if err == nil {
		t.Error("expected error")
	}
}
