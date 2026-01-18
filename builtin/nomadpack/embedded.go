package nomadpack

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EmbeddedPacks contains all embedded pack files from the packs directory.
// The embed directive includes all files and subdirectories under packs/.
// Using "all:" prefix to include files starting with . and _
//
//go:embed all:packs
var EmbeddedPacks embed.FS

// AvailableEmbeddedPacks returns a list of available embedded pack names.
// Currently, only the "cloudstation" pack is embedded.
func AvailableEmbeddedPacks() []string {
	return []string{"cloudstation"}
}

// HasEmbeddedPack checks if a pack with the given name is embedded.
// It returns true if the pack is found in the list of available embedded packs.
func HasEmbeddedPack(name string) bool {
	availablePacks := AvailableEmbeddedPacks()
	for _, pack := range availablePacks {
		if pack == name {
			return true
		}
	}
	return false
}

// ExtractEmbeddedPack extracts an embedded pack to a temporary directory.
// It validates that the pack exists, creates a temporary directory, and copies
// all pack files preserving the directory structure.
//
// The function returns the path to the extracted pack directory (e.g., /tmp/cs-pack-123/cloudstation).
// The caller is responsible for cleaning up the temporary directory using:
//
//	defer os.RemoveAll(filepath.Dir(path))
//
// Returns an error if the pack is not found or extraction fails.
func ExtractEmbeddedPack(packName string) (string, error) {
	// Validate pack exists
	if !HasEmbeddedPack(packName) {
		return "", fmt.Errorf("embedded pack %q not found", packName)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "cs-pack-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Walk embedded FS and copy files
	packPath := filepath.Join("packs", packName)
	err = fs.WalkDir(EmbeddedPacks, packPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(packPath, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}
		destPath := filepath.Join(tempDir, packName, relPath)

		// Create directory if needed
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read embedded file
		content, err := EmbeddedPacks.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %q: %w", path, err)
		}

		// Ensure parent directory exists
		parentDir := filepath.Dir(destPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Write file to destination
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %q: %w", destPath, err)
		}

		return nil
	})

	// Handle extraction errors
	if err != nil {
		// Clean up on failure
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to extract pack: %w", err)
	}

	// Return path to extracted pack directory
	return filepath.Join(tempDir, packName), nil
}
