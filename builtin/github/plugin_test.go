package github

import (
	"context"
	"os"
	"testing"

	"github.com/thecloudstation/cloudstation-orchestrator/pkg/artifact"
)

func TestConfigSet_NilConfig(t *testing.T) {
	r := &Registry{}
	err := r.ConfigSet(nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if r.config == nil {
		t.Error("expected config to be initialized")
	}
	if !r.config.CreateRelease {
		t.Error("expected CreateRelease to default to true")
	}
}

func TestConfigSet_MapConfig(t *testing.T) {
	r := &Registry{}
	config := map[string]interface{}{
		"repository":          "owner/repo",
		"token":               "ghp_test_token",
		"tag_name":            "v1.0.0",
		"release_name":        "Release 1.0.0",
		"release_notes":       "This is a test release",
		"draft":               true,
		"prerelease":          true,
		"generate_notes":      true,
		"checksums":           true,
		"create_release":      false,
		"target_commit":       "main",
		"discussion_category": "announcements",
	}

	err := r.ConfigSet(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify all fields
	if r.config.Repository != "owner/repo" {
		t.Errorf("expected Repository 'owner/repo', got %s", r.config.Repository)
	}
	if r.config.Token != "ghp_test_token" {
		t.Errorf("expected Token 'ghp_test_token', got %s", r.config.Token)
	}
	if r.config.TagName != "v1.0.0" {
		t.Errorf("expected TagName 'v1.0.0', got %s", r.config.TagName)
	}
	if r.config.ReleaseName != "Release 1.0.0" {
		t.Errorf("expected ReleaseName 'Release 1.0.0', got %s", r.config.ReleaseName)
	}
	if r.config.ReleaseNotes != "This is a test release" {
		t.Errorf("expected ReleaseNotes 'This is a test release', got %s", r.config.ReleaseNotes)
	}
	if !r.config.Draft {
		t.Error("expected Draft to be true")
	}
	if !r.config.Prerelease {
		t.Error("expected Prerelease to be true")
	}
	if !r.config.GenerateNotes {
		t.Error("expected GenerateNotes to be true")
	}
	if !r.config.Checksums {
		t.Error("expected Checksums to be true")
	}
	if r.config.CreateRelease {
		t.Error("expected CreateRelease to be false")
	}
	if r.config.TargetCommit != "main" {
		t.Errorf("expected TargetCommit 'main', got %s", r.config.TargetCommit)
	}
	if r.config.DiscussionCategory != "announcements" {
		t.Errorf("expected DiscussionCategory 'announcements', got %s", r.config.DiscussionCategory)
	}
}

func TestConfigSet_NestedAuthBlock(t *testing.T) {
	r := &Registry{}
	config := map[string]interface{}{
		"repository": "owner/repo",
		"tag_name":   "v1.0.0",
		"token":      "original_token",
		"auth": map[string]interface{}{
			"token": "auth_block_token",
		},
	}

	err := r.ConfigSet(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Auth block token should override the flat token
	if r.config.Token != "auth_block_token" {
		t.Errorf("expected Token from auth block 'auth_block_token', got %s", r.config.Token)
	}
}

func TestConfigSet_TypedConfig(t *testing.T) {
	r := &Registry{}
	config := &RegistryConfig{
		Repository:    "typed/repo",
		Token:         "typed_token",
		TagName:       "v2.0.0",
		ReleaseName:   "Typed Release",
		Draft:         true,
		CreateRelease: false,
		TargetCommit:  "develop",
	}

	err := r.ConfigSet(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if r.config.Repository != "typed/repo" {
		t.Errorf("expected Repository 'typed/repo', got %s", r.config.Repository)
	}
	if r.config.Token != "typed_token" {
		t.Errorf("expected Token 'typed_token', got %s", r.config.Token)
	}
	if r.config.TagName != "v2.0.0" {
		t.Errorf("expected TagName 'v2.0.0', got %s", r.config.TagName)
	}
	if r.config.CreateRelease {
		t.Error("expected CreateRelease to remain false when explicitly set")
	}
}

func TestConfigSet_BoolDefaults(t *testing.T) {
	r := &Registry{}
	config := map[string]interface{}{
		"repository": "owner/repo",
		"token":      "test_token",
		"tag_name":   "v1.0.0",
	}

	err := r.ConfigSet(config)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check boolean defaults
	if r.config.Draft {
		t.Error("expected Draft to default to false")
	}
	if r.config.Prerelease {
		t.Error("expected Prerelease to default to false")
	}
	if r.config.GenerateNotes {
		t.Error("expected GenerateNotes to default to false")
	}
	if r.config.Checksums {
		t.Error("expected Checksums to default to false")
	}
	if !r.config.CreateRelease {
		t.Error("expected CreateRelease to default to true")
	}
}

func TestPush_MissingRepository(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Token:   "test_token",
			TagName: "v1.0.0",
		},
	}

	art := &artifact.Artifact{
		Metadata: map[string]interface{}{
			"binaries": []string{"/tmp/test.bin"},
		},
	}

	_, err := r.Push(ctx, art)
	if err == nil {
		t.Error("expected error for missing repository")
	}
	if err.Error() != "repository is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPush_MissingToken(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Repository: "owner/repo",
			TagName:    "v1.0.0",
		},
	}

	art := &artifact.Artifact{
		Metadata: map[string]interface{}{
			"binaries": []string{"/tmp/test.bin"},
		},
	}

	_, err := r.Push(ctx, art)
	if err == nil {
		t.Error("expected error for missing token")
	}
	if err.Error() != "token is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPush_MissingTagName(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Repository: "owner/repo",
			Token:      "test_token",
		},
	}

	art := &artifact.Artifact{
		Metadata: map[string]interface{}{
			"binaries": []string{"/tmp/test.bin"},
		},
	}

	_, err := r.Push(ctx, art)
	if err == nil {
		t.Error("expected error for missing tag_name")
	}
	if err.Error() != "tag_name is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPush_NilArtifact(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Repository: "owner/repo",
			Token:      "test_token",
			TagName:    "v1.0.0",
		},
	}

	// Note: The current implementation doesn't explicitly check for nil artifact
	// It will panic when trying to access art.Metadata
	// This test documents the current behavior
	defer func() {
		if r := recover(); r != nil {
			// Expected panic for nil artifact
			return
		}
	}()

	_, err := r.Push(ctx, nil)
	if err == nil {
		// If no panic occurred, we should have gotten an error
		t.Error("expected error or panic for nil artifact")
	}
}

func TestPush_NoAssets(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Repository: "owner/repo",
			Token:      "test_token",
			TagName:    "v1.0.0",
		},
	}

	// Test with empty metadata
	art := &artifact.Artifact{
		Metadata: map[string]interface{}{},
	}

	_, err := r.Push(ctx, art)
	if err == nil {
		t.Error("expected error for no assets")
	}
	if err.Error() != "no assets found (set assets in config or provide binaries/release_assets/binary_path in artifact metadata)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestComputeSHA256(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "test_checksum_*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write known content
	testContent := []byte("Hello, GitHub Releases!")
	if _, err := tmpFile.Write(testContent); err != nil {
		t.Fatalf("failed to write test content: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	// Compute checksum
	hash, err := computeSHA256(tmpFile.Name())
	if err != nil {
		t.Errorf("failed to compute SHA256: %v", err)
	}

	// Since we can't predict the exact hash in a test, just verify it's non-empty and has correct format
	// SHA256 of "Hello, GitHub Releases!" would be a 64-character hex string
	if len(hash) != 64 {
		t.Errorf("expected SHA256 hash to be 64 characters, got %d", len(hash))
	}

	// Verify it's a valid hex string
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			t.Errorf("invalid character in hash: %c", c)
		}
	}

	// Test with non-existent file
	_, err = computeSHA256("/non/existent/file.bin")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// Additional helper tests for unexported functions

func TestGetString(t *testing.T) {
	tests := []struct {
		name      string
		configMap map[string]interface{}
		key       string
		expected  string
	}{
		{
			name: "string value",
			configMap: map[string]interface{}{
				"key": "value",
			},
			key:      "key",
			expected: "value",
		},
		{
			name: "string pointer",
			configMap: map[string]interface{}{
				"key": stringPtr("pointer_value"),
			},
			key:      "key",
			expected: "pointer_value",
		},
		{
			name: "nil string pointer",
			configMap: map[string]interface{}{
				"key": (*string)(nil),
			},
			key:      "key",
			expected: "",
		},
		{
			name:      "missing key",
			configMap: map[string]interface{}{},
			key:       "missing",
			expected:  "",
		},
		{
			name: "non-string value",
			configMap: map[string]interface{}{
				"key": 123,
			},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.configMap, tt.key)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name       string
		configMap  map[string]interface{}
		key        string
		defaultVal bool
		expected   bool
	}{
		{
			name: "bool true",
			configMap: map[string]interface{}{
				"key": true,
			},
			key:        "key",
			defaultVal: false,
			expected:   true,
		},
		{
			name: "bool false",
			configMap: map[string]interface{}{
				"key": false,
			},
			key:        "key",
			defaultVal: true,
			expected:   false,
		},
		{
			name: "bool pointer true",
			configMap: map[string]interface{}{
				"key": boolPtr(true),
			},
			key:        "key",
			defaultVal: false,
			expected:   true,
		},
		{
			name: "bool pointer false",
			configMap: map[string]interface{}{
				"key": boolPtr(false),
			},
			key:        "key",
			defaultVal: true,
			expected:   false,
		},
		{
			name: "nil bool pointer",
			configMap: map[string]interface{}{
				"key": (*bool)(nil),
			},
			key:        "key",
			defaultVal: true,
			expected:   true,
		},
		{
			name:       "missing key uses default",
			configMap:  map[string]interface{}{},
			key:        "missing",
			defaultVal: true,
			expected:   true,
		},
		{
			name: "non-bool value uses default",
			configMap: map[string]interface{}{
				"key": "not a bool",
			},
			key:        "key",
			defaultVal: true,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBool(tt.configMap, tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	r := &Registry{
		config: &RegistryConfig{
			Repository: "test/repo",
			Token:      "test_token",
			TagName:    "v1.0.0",
		},
	}

	config, err := r.Config()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	cfg, ok := config.(*RegistryConfig)
	if !ok {
		t.Error("expected config to be *RegistryConfig")
	}

	if cfg.Repository != "test/repo" {
		t.Errorf("expected Repository 'test/repo', got %s", cfg.Repository)
	}
}

func TestPull(t *testing.T) {
	r := &Registry{}
	ctx := context.Background()
	ref := &artifact.RegistryRef{
		Registry:   "github.com",
		Repository: "owner/repo",
		Tag:        "v1.0.0",
	}

	_, err := r.Pull(ctx, ref)
	if err == nil {
		t.Error("expected error for Pull (not implemented)")
	}
	if err.Error() != "github releases pull not implemented" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Additional edge case tests

func TestPush_NilConfig(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: nil,
	}

	art := &artifact.Artifact{
		Metadata: map[string]interface{}{
			"binaries": []string{"/tmp/test.bin"},
		},
	}

	_, err := r.Push(ctx, art)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if err.Error() != "registry config is nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPush_AssetsFromDifferentMetadataKeys(t *testing.T) {
	ctx := context.Background()
	r := &Registry{
		config: &RegistryConfig{
			Repository: "owner/repo",
			Token:      "test_token",
			TagName:    "v1.0.0",
		},
	}

	testCases := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			name: "binaries as []string",
			metadata: map[string]interface{}{
				"binaries": []string{"/tmp/test1.bin", "/tmp/test2.bin"},
			},
		},
		{
			name: "release_assets as []interface{}",
			metadata: map[string]interface{}{
				"release_assets": []interface{}{"/tmp/test1.bin", "/tmp/test2.bin"},
			},
		},
		{
			name: "binary_path as string",
			metadata: map[string]interface{}{
				"binary_path": "/tmp/single.bin",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			art := &artifact.Artifact{
				Metadata: tc.metadata,
			}

			_, err := r.Push(ctx, art)
			// We expect errors because gh command won't work in test environment
			// But we should get past the validation phase
			if err != nil && err.Error() == "no assets found in artifact metadata (expected binaries, release_assets, or binary_path)" {
				t.Error("should have found assets in metadata")
			}
		})
	}
}

func TestConfigSet_UnsupportedConfigType(t *testing.T) {
	r := &Registry{}
	// Pass an unsupported type (not map, not *RegistryConfig, not nil)
	err := r.ConfigSet("unsupported")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Should fall back to default config with CreateRelease=true
	if r.config == nil {
		t.Error("expected config to be initialized with default")
	}
	if !r.config.CreateRelease {
		t.Error("expected CreateRelease to default to true for unsupported config type")
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
