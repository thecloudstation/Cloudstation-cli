package railpack

import (
	"reflect"
	"testing"
)

func TestConfigSet_NilConfig(t *testing.T) {
	b := &Builder{}
	err := b.ConfigSet(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if b.config == nil {
		t.Fatal("expected config to be initialized")
	}

	if b.config.Name != "" {
		t.Errorf("expected empty name, got %s", b.config.Name)
	}
}

func TestConfigSet_MapConfigAllFields(t *testing.T) {
	b := &Builder{}

	config := map[string]interface{}{
		"name":    "test-image",
		"tag":     "v1.0.0",
		"context": "/app/build",
		"build_args": map[string]interface{}{
			"NODE_ENV": "production",
			"VERSION":  "1.0.0",
		},
		"env": map[string]interface{}{
			"API_KEY": "secret123",
			"DB_HOST": "localhost",
		},
		"vault_address": "https://vault.example.com",
		"role_id":       "role123",
		"secret_id":     "secret456",
		"secrets_path":  "secret/data/app",
	}

	err := b.ConfigSet(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if b.config.Name != "test-image" {
		t.Errorf("expected name 'test-image', got %s", b.config.Name)
	}

	if b.config.Tag != "v1.0.0" {
		t.Errorf("expected tag 'v1.0.0', got %s", b.config.Tag)
	}

	if b.config.Context != "/app/build" {
		t.Errorf("expected context '/app/build', got %s", b.config.Context)
	}

	if len(b.config.BuildArgs) != 2 {
		t.Errorf("expected 2 build args, got %d", len(b.config.BuildArgs))
	}

	if b.config.BuildArgs["NODE_ENV"] != "production" {
		t.Errorf("expected NODE_ENV 'production', got %s", b.config.BuildArgs["NODE_ENV"])
	}

	if len(b.config.Env) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(b.config.Env))
	}

	if b.config.Env["API_KEY"] != "secret123" {
		t.Errorf("expected API_KEY 'secret123', got %s", b.config.Env["API_KEY"])
	}

	if b.config.VaultAddress != "https://vault.example.com" {
		t.Errorf("expected vault_address 'https://vault.example.com', got %s", b.config.VaultAddress)
	}
}

func TestConfigSet_MapConfigMinimal(t *testing.T) {
	b := &Builder{}

	config := map[string]interface{}{
		"name": "minimal-image",
	}

	err := b.ConfigSet(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if b.config.Name != "minimal-image" {
		t.Errorf("expected name 'minimal-image', got %s", b.config.Name)
	}

	if b.config.Tag != "" {
		t.Errorf("expected empty tag (default), got %s", b.config.Tag)
	}

	if b.config.Context != "" {
		t.Errorf("expected empty context (default), got %s", b.config.Context)
	}

	if b.config.BuildArgs != nil {
		t.Errorf("expected nil build args, got %v", b.config.BuildArgs)
	}

	if b.config.Env != nil {
		t.Errorf("expected nil env, got %v", b.config.Env)
	}
}

func TestConfigSet_TypedConfig(t *testing.T) {
	b := &Builder{}

	expectedConfig := &BuilderConfig{
		Name:    "typed-image",
		Tag:     "v2.0.0",
		Context: "/src",
		BuildArgs: map[string]string{
			"ARG1": "value1",
		},
		Env: map[string]string{
			"ENV1": "value1",
		},
	}

	err := b.ConfigSet(expectedConfig)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(b.config, expectedConfig) {
		t.Errorf("expected config to match, got %+v", b.config)
	}
}

func TestBuild_NilConfig(t *testing.T) {
	b := &Builder{}
	b.config = nil

	_, err := b.Build(nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}

	if err.Error() != "railpack builder configuration is not set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestBuild_MissingName(t *testing.T) {
	b := &Builder{
		config: &BuilderConfig{
			Tag:     "latest",
			Context: ".",
		},
	}

	_, err := b.Build(nil)
	if err == nil {
		t.Fatal("expected error for missing name")
	}

	if err.Error() != "railpack builder requires 'name' field to be set" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig(t *testing.T) {
	expectedConfig := &BuilderConfig{
		Name: "test-image",
		Tag:  "latest",
	}

	b := &Builder{config: expectedConfig}

	config, err := b.Config()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Errorf("expected config to match, got %+v", config)
	}
}
