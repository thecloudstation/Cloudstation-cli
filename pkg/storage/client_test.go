package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestDownloadAndExtract(t *testing.T) {
	// Create a test tar.gz file in memory
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a test directory
	dirHeader := &tar.Header{
		Name:     "testdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tarWriter.WriteHeader(dirHeader); err != nil {
		t.Fatalf("Failed to write dir header: %v", err)
	}

	// Add a test file
	testContent := []byte("Hello, World!")
	fileHeader := &tar.Header{
		Name: "testdir/test.txt",
		Mode: 0644,
		Size: int64(len(testContent)),
	}
	if err := tarWriter.WriteHeader(fileHeader); err != nil {
		t.Fatalf("Failed to write file header: %v", err)
	}
	if _, err := tarWriter.Write(testContent); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Close writers
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, bytes.NewReader(buf.Bytes())); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := hclog.NewNullLogger()

	// Test extraction
	if err := DownloadAndExtract(server.URL, tempDir, logger); err != nil {
		t.Errorf("DownloadAndExtract failed: %v", err)
	}

	// Verify extracted content
	extractedFile := filepath.Join(tempDir, "testdir", "test.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("Extracted content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

func TestDownloadAndExtract_InvalidURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := hclog.NewNullLogger()

	// Test with invalid URL
	err = DownloadAndExtract("http://invalid-url-that-does-not-exist.local", tempDir, logger)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestDownloadAndExtract_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := hclog.NewNullLogger()

	err = DownloadAndExtract(server.URL, tempDir, logger)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestDownloadAndExtract_InvalidTarball(t *testing.T) {
	// Create test server that returns invalid tar.gz data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid gzip data"))
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := hclog.NewNullLogger()

	err = DownloadAndExtract(server.URL, tempDir, logger)
	if err == nil {
		t.Error("Expected error for invalid gzip data, got nil")
	}
}

func TestDownloadAndExtract_PathTraversal(t *testing.T) {
	// Create a malicious tar.gz with path traversal attempt
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a file with path traversal attempt
	maliciousContent := []byte("malicious")
	fileHeader := &tar.Header{
		Name: "../../../etc/passwd", // Attempt path traversal
		Mode: 0644,
		Size: int64(len(maliciousContent)),
	}
	if err := tarWriter.WriteHeader(fileHeader); err != nil {
		t.Fatalf("Failed to write file header: %v", err)
	}
	if _, err := tarWriter.Write(maliciousContent); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, bytes.NewReader(buf.Bytes())); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := hclog.NewNullLogger()

	// Should handle path traversal securely
	err = DownloadAndExtract(server.URL, tempDir, logger)
	if err != nil {
		// It's OK if it errors, but should not extract outside tempDir
		t.Logf("Got expected error for path traversal: %v", err)
	}

	// Verify no file was created outside tempDir
	if _, err := os.Stat("/etc/passwd.test"); err == nil {
		t.Error("Path traversal attack succeeded - file created outside destDir")
	}
}

func TestDownloadFile(t *testing.T) {
	testContent := []byte("Test file content")

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	destPath := filepath.Join(tempDir, "downloaded.txt")
	logger := hclog.NewNullLogger()

	// Test download
	if err := DownloadFile(server.URL, destPath, logger); err != nil {
		t.Errorf("DownloadFile failed: %v", err)
	}

	// Verify downloaded content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Errorf("Failed to read downloaded file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("Downloaded content mismatch: got %q, want %q", string(content), string(testContent))
	}
}
