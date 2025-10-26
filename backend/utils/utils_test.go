package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test", "nested", "dir")

	err := EnsureDir(testPath)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestMustFindProjectRoot(t *testing.T) {
	root := MustFindProjectRoot()
	if root == "" {
		t.Error("MustFindProjectRoot returned empty string")
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		t.Logf("Warning: go.mod not found at %s (this is expected in test environment)", goModPath)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	WriteJSON(w, payload)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "key1") || !strings.Contains(body, "value1") {
		t.Errorf("Response body does not contain expected JSON: %s", body)
	}
}

func TestDownloadToFile(t *testing.T) {
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	ctx := context.Background()
	err := DownloadToFile(ctx, server.URL, targetPath)
	if err != nil {
		t.Fatalf("DownloadToFile failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Downloaded content mismatch. Expected '%s', got '%s'", testContent, string(content))
	}
}

func TestDownloadToFileError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "failed.txt")

	ctx := context.Background()
	err := DownloadToFile(ctx, server.URL, targetPath)
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestSaveBase64ToFile(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "base64.txt")

	base64Data := "SGVsbG8gV29ybGQ="
	err := SaveBase64ToFile(base64Data, targetPath)
	if err != nil {
		t.Fatalf("SaveBase64ToFile failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Hello World" {
		t.Errorf("Decoded content mismatch. Expected 'Hello World', got '%s'", string(content))
	}
}

func TestSaveBase64ToFileWithDataURL(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "base64_data.txt")

	base64Data := "data:text/plain;base64,SGVsbG8gV29ybGQ="
	err := SaveBase64ToFile(base64Data, targetPath)
	if err != nil {
		t.Fatalf("SaveBase64ToFile failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Hello World" {
		t.Errorf("Decoded content mismatch. Expected 'Hello World', got '%s'", string(content))
	}
}

func TestRemoveGeneratedFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	RemoveGeneratedFile("/generated/images/test.txt", "/generated/images/", tmpDir)

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestRemoveGeneratedFileWrongPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	RemoveGeneratedFile("/wrong/prefix/test.txt", "/generated/images/", tmpDir)

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File should not have been deleted with wrong prefix")
	}
}
