package storage

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// DownloadAndExtract downloads a tarball from the given URL and extracts it to the destination directory
func DownloadAndExtract(url, destDir string, logger hclog.Logger) error {
	logger.Info("Downloading source tarball", "url", url, "destDir", destDir)

	// Download the tarball
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download tarball: HTTP %d", resp.StatusCode)
	}

	logger.Info("Download complete, extracting tarball")

	// Create a gzip reader
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create a tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	fileCount := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Clean the file path to prevent path traversal attacks
		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			// Create the file
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			// Copy file contents
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			file.Close()
			fileCount++

		case tar.TypeSymlink:
			// Validate symlink target doesn't use absolute path
			if filepath.IsAbs(header.Linkname) {
				return fmt.Errorf("refusing to create absolute symlink: %s -> %s", header.Name, header.Linkname)
			}

			// Validate symlink target doesn't traverse outside destDir
			linkTarget := filepath.Clean(filepath.Join(filepath.Dir(target), header.Linkname))
			if !strings.HasPrefix(linkTarget, filepath.Clean(destDir)+string(os.PathSeparator)) &&
				linkTarget != filepath.Clean(destDir) {
				return fmt.Errorf("symlink target escapes destination directory: %s -> %s", header.Name, header.Linkname)
			}

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink %s: %w", target, err)
			}

			// Create symbolic link
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink %s: %w", target, err)
			}
		}
	}

	logger.Info("Extraction complete", "fileCount", fileCount)
	return nil
}

// DownloadFile downloads a file from the given URL and saves it to the destination path
func DownloadFile(url, destPath string, logger hclog.Logger) error {
	logger.Info("Downloading file", "url", url, "destPath", destPath)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create/overwrite destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Download complete", "destPath", destPath)
	return nil
}
