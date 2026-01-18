package noop

import (
	"context"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
	"github.com/thecloudstation/cloudstation-orchestrator/pkg/deployment"
)

// Test Registry component
func TestRegistry_ConfigSet_NilConfig(t *testing.T) {
	r := NewRegistry()
	err := r.ConfigSet(nil)
	if err != nil {
		t.Errorf("ConfigSet(nil) returned error: %v", err)
	}

	cfg, err := r.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil config")
	}
}

func TestRegistry_ConfigSet_MapConfig(t *testing.T) {
	r := NewRegistry()
	configMap := map[string]interface{}{}
	err := r.ConfigSet(configMap)
	if err != nil {
		t.Errorf("ConfigSet(map) returned error: %v", err)
	}

	cfg, err := r.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil config")
	}
}

func TestRegistry_ConfigSet_TypedConfig(t *testing.T) {
	r := NewRegistry()
	typedConfig := &RegistryConfig{}
	err := r.ConfigSet(typedConfig)
	if err != nil {
		t.Errorf("ConfigSet(typed) returned error: %v", err)
	}

	cfg, err := r.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg != typedConfig {
		t.Error("Config() did not return the same typed config")
	}
}

func TestRegistry_Config(t *testing.T) {
	r := NewRegistry()
	cfg, err := r.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil")
	}
}

func TestRegistry_Push(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	art := &artifact.Artifact{
		ID:    "test-artifact",
		Image: "test:latest",
	}

	ref, err := r.Push(ctx, art)
	if err != nil {
		t.Errorf("Push() returned error: %v", err)
	}
	if ref == nil {
		t.Fatal("Push() returned nil RegistryRef")
	}
	if ref.Registry != "noop-registry" {
		t.Errorf("Expected registry 'noop-registry', got '%s'", ref.Registry)
	}
	if ref.Repository != "noop" {
		t.Errorf("Expected repository 'noop', got '%s'", ref.Repository)
	}
	if ref.Tag != "latest" {
		t.Errorf("Expected tag 'latest', got '%s'", ref.Tag)
	}
	if ref.FullImage != "noop-registry/noop:latest" {
		t.Errorf("Expected full image 'noop-registry/noop:latest', got '%s'", ref.FullImage)
	}
	if ref.PushedAt.IsZero() {
		t.Error("PushedAt should not be zero")
	}
}

func TestRegistry_Pull(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	ref := &artifact.RegistryRef{
		Registry:   "noop-registry",
		Repository: "noop",
		Tag:        "latest",
		FullImage:  "noop-registry/noop:latest",
	}

	art, err := r.Pull(ctx, ref)
	if err == nil {
		t.Error("Pull() should return error")
	}
	if art != nil {
		t.Error("Pull() should return nil artifact")
	}
}

// Test ReleaseManager component
func TestReleaseManager_ConfigSet_NilConfig(t *testing.T) {
	rm := NewReleaseManager()
	err := rm.ConfigSet(nil)
	if err != nil {
		t.Errorf("ConfigSet(nil) returned error: %v", err)
	}

	cfg, err := rm.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil config")
	}
}

func TestReleaseManager_ConfigSet_MapConfig(t *testing.T) {
	rm := NewReleaseManager()
	configMap := map[string]interface{}{
		"message": "test message",
	}
	err := rm.ConfigSet(configMap)
	if err != nil {
		t.Errorf("ConfigSet(map) returned error: %v", err)
	}

	cfg, err := rm.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Config() returned nil config")
	}

	releaseConfig, ok := cfg.(*ReleaseConfig)
	if !ok {
		t.Fatal("Config() did not return *ReleaseConfig")
	}
	if releaseConfig.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", releaseConfig.Message)
	}
}

func TestReleaseManager_ConfigSet_TypedConfig(t *testing.T) {
	rm := NewReleaseManager()
	typedConfig := &ReleaseConfig{
		Message: "typed message",
	}
	err := rm.ConfigSet(typedConfig)
	if err != nil {
		t.Errorf("ConfigSet(typed) returned error: %v", err)
	}

	cfg, err := rm.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg != typedConfig {
		t.Error("Config() did not return the same typed config")
	}
}

func TestReleaseManager_Config(t *testing.T) {
	rm := NewReleaseManager()
	cfg, err := rm.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil")
	}
}

func TestReleaseManager_Release(t *testing.T) {
	rm := NewReleaseManager()
	ctx := context.Background()
	dep := &deployment.Deployment{
		ID:       "test-deployment",
		Name:     "test",
		Platform: "noop",
	}

	err := rm.Release(ctx, dep)
	if err != nil {
		t.Errorf("Release() returned error: %v", err)
	}
}

// Test Builder component
func TestBuilder_ConfigSet_NilConfig(t *testing.T) {
	b := NewBuilder()
	err := b.ConfigSet(nil)
	if err != nil {
		t.Errorf("ConfigSet(nil) returned error: %v", err)
	}

	cfg, err := b.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Error("Config() returned nil config")
	}
}

func TestBuilder_ConfigSet_MapConfig(t *testing.T) {
	b := NewBuilder()
	configMap := map[string]interface{}{
		"message": "build message",
	}
	err := b.ConfigSet(configMap)
	if err != nil {
		t.Errorf("ConfigSet(map) returned error: %v", err)
	}

	cfg, err := b.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Config() returned nil config")
	}

	builderConfig, ok := cfg.(*BuilderConfig)
	if !ok {
		t.Fatal("Config() did not return *BuilderConfig")
	}
	if builderConfig.Message != "build message" {
		t.Errorf("Expected message 'build message', got '%s'", builderConfig.Message)
	}
}

func TestBuilder_ConfigSet_TypedConfig(t *testing.T) {
	b := NewBuilder()
	typedConfig := &BuilderConfig{
		Message: "typed build message",
	}
	err := b.ConfigSet(typedConfig)
	if err != nil {
		t.Errorf("ConfigSet(typed) returned error: %v", err)
	}

	cfg, err := b.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg != typedConfig {
		t.Error("Config() did not return the same typed config")
	}
}

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder()
	ctx := context.Background()

	art, err := b.Build(ctx)
	if err != nil {
		t.Errorf("Build() returned error: %v", err)
	}
	if art == nil {
		t.Fatal("Build() returned nil artifact")
	}
	if art.ID != "noop-artifact" {
		t.Errorf("Expected artifact ID 'noop-artifact', got '%s'", art.ID)
	}
	if art.Image != "noop:latest" {
		t.Errorf("Expected image 'noop:latest', got '%s'", art.Image)
	}
	if art.Tag != "latest" {
		t.Errorf("Expected tag 'latest', got '%s'", art.Tag)
	}
	if art.BuildTime.IsZero() {
		t.Error("BuildTime should not be zero")
	}
}

// Test Platform component
func TestPlatform_Deploy(t *testing.T) {
	p := &Platform{}
	ctx := context.Background()
	art := &artifact.Artifact{
		ID:    "test-artifact",
		Image: "test:latest",
	}

	dep, err := p.Deploy(ctx, art)
	if err != nil {
		t.Errorf("Deploy() returned error: %v", err)
	}
	if dep == nil {
		t.Fatal("Deploy() returned nil deployment")
	}
	if dep.ID != "noop-deployment" {
		t.Errorf("Expected deployment ID 'noop-deployment', got '%s'", dep.ID)
	}
	if dep.Name != "noop" {
		t.Errorf("Expected deployment name 'noop', got '%s'", dep.Name)
	}
	if dep.Platform != "noop" {
		t.Errorf("Expected platform 'noop', got '%s'", dep.Platform)
	}
	if dep.ArtifactID != art.ID {
		t.Errorf("Expected artifact ID '%s', got '%s'", art.ID, dep.ArtifactID)
	}
	if dep.Status.State != deployment.StateRunning {
		t.Errorf("Expected state Running, got %v", dep.Status.State)
	}
	if dep.Status.Health != deployment.HealthHealthy {
		t.Errorf("Expected health Healthy, got %v", dep.Status.Health)
	}
	if dep.DeployedAt.IsZero() {
		t.Error("DeployedAt should not be zero")
	}
}

func TestPlatform_Destroy(t *testing.T) {
	p := &Platform{}
	ctx := context.Background()

	err := p.Destroy(ctx, "test-deployment")
	if err != nil {
		t.Errorf("Destroy() returned error: %v", err)
	}
}

func TestPlatform_Status(t *testing.T) {
	p := &Platform{}
	ctx := context.Background()

	status, err := p.Status(ctx, "test-deployment")
	if err != nil {
		t.Errorf("Status() returned error: %v", err)
	}
	if status == nil {
		t.Fatal("Status() returned nil status")
	}
	if status.State != deployment.StateRunning {
		t.Errorf("Expected state Running, got %v", status.State)
	}
	if status.Health != deployment.HealthHealthy {
		t.Errorf("Expected health Healthy, got %v", status.Health)
	}
}

func TestPlatform_Config(t *testing.T) {
	p := &Platform{}

	cfg, err := p.Config()
	if err != nil {
		t.Errorf("Config() returned error: %v", err)
	}
	if cfg != nil {
		t.Error("Config() should return nil for Platform")
	}
}

func TestPlatform_ConfigSet(t *testing.T) {
	p := &Platform{}

	err := p.ConfigSet(nil)
	if err != nil {
		t.Errorf("ConfigSet() returned error: %v", err)
	}

	err = p.ConfigSet(map[string]interface{}{})
	if err != nil {
		t.Errorf("ConfigSet() returned error: %v", err)
	}
}
