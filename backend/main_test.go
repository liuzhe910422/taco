package main

import (
	"os"
	"path/filepath"
	"testing"

	"taco/backend/utils"
)

func TestMainFunction(t *testing.T) {
	t.Skip("Skipping main function test as it runs an HTTP server indefinitely")
}

func TestDirectoryInitialization(t *testing.T) {
	tmpDir := t.TempDir()

	uploadDir := filepath.Join(tmpDir, "uploads")
	imagesDir := filepath.Join(tmpDir, "generated", "images")
	audioDir := filepath.Join(tmpDir, "generated", "audio")

	if err := utils.EnsureDir(uploadDir); err != nil {
		t.Errorf("Failed to create upload directory: %v", err)
	}

	if err := utils.EnsureDir(imagesDir); err != nil {
		t.Errorf("Failed to create images directory: %v", err)
	}

	if err := utils.EnsureDir(audioDir); err != nil {
		t.Errorf("Failed to create audio directory: %v", err)
	}

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		t.Error("Upload directory was not created")
	}

	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		t.Error("Images directory was not created")
	}

	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		t.Error("Audio directory was not created")
	}
}

func TestListenAddr(t *testing.T) {
	if utils.ListenAddr != ":8080" {
		t.Errorf("Expected listen address ':8080', got '%s'", utils.ListenAddr)
	}
}
