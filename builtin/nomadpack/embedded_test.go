package nomadpack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvailableEmbeddedPacks(t *testing.T) {
	packs := AvailableEmbeddedPacks()

	// Verify at least one pack is returned
	if len(packs) == 0 {
		t.Error("Expected at least one embedded pack")
	}

	// Verify "cloudstation" is in the list
	found := false
	for _, p := range packs {
		if p == "cloudstation" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'cloudstation' in embedded packs")
	}
}

func TestHasEmbeddedPack(t *testing.T) {
	// Test that HasEmbeddedPack("cloudstation") returns true
	if !HasEmbeddedPack("cloudstation") {
		t.Error("Expected HasEmbeddedPack(\"cloudstation\") to return true")
	}

	// Test that HasEmbeddedPack("nonexistent") returns false
	if HasEmbeddedPack("nonexistent") {
		t.Error("Expected HasEmbeddedPack(\"nonexistent\") to return false")
	}
}

func TestExtractEmbeddedPack(t *testing.T) {
	// Extract the cloudstation pack
	path, err := ExtractEmbeddedPack("cloudstation")
	if err != nil {
		t.Fatalf("ExtractEmbeddedPack(\"cloudstation\") error = %v", err)
	}

	// Ensure cleanup after test
	defer os.RemoveAll(filepath.Dir(path))

	// Verify key files exist
	expectedFiles := []string{
		"metadata.hcl",
		"variables.hcl",
		"outputs.tpl",
		"templates/_helpers.tpl",
		"templates/cloudstation.nomad.tpl",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(path, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist at %s", file, filePath)
		}
	}
}

func TestExtractEmbeddedPack_NotFound(t *testing.T) {
	// Try to extract a non-existent pack
	_, err := ExtractEmbeddedPack("nonexistent")

	// Expect error to be returned
	if err == nil {
		t.Error("Expected ExtractEmbeddedPack(\"nonexistent\") to return an error, got nil")
	}
}
