package utils

import (
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	if ListenAddr != ":8080" {
		t.Errorf("Expected ListenAddr to be ':8080', got '%s'", ListenAddr)
	}

	if MaxFileSize != 32<<20 {
		t.Errorf("Expected MaxFileSize to be %d, got %d", 32<<20, MaxFileSize)
	}

	if GeneratedImagesURLPrefix != "/generated/images/" {
		t.Errorf("Expected GeneratedImagesURLPrefix to be '/generated/images/', got '%s'", GeneratedImagesURLPrefix)
	}

	if GeneratedAudioURLPrefix != "/generated/audio/" {
		t.Errorf("Expected GeneratedAudioURLPrefix to be '/generated/audio/', got '%s'", GeneratedAudioURLPrefix)
	}
}

func TestGlobalPaths(t *testing.T) {
	if ProjectRoot == "" {
		t.Error("ProjectRoot should not be empty")
	}

	if !strings.Contains(ConfigPath, "config.json") {
		t.Errorf("ConfigPath should contain 'config.json', got '%s'", ConfigPath)
	}

	if !strings.Contains(CharactersPath, "characters.json") {
		t.Errorf("CharactersPath should contain 'characters.json', got '%s'", CharactersPath)
	}

	if !strings.Contains(ScenesPath, "scenes.json") {
		t.Errorf("ScenesPath should contain 'scenes.json', got '%s'", ScenesPath)
	}

	if !strings.Contains(GeneratedImagesDir, "images") {
		t.Errorf("GeneratedImagesDir should contain 'images', got '%s'", GeneratedImagesDir)
	}

	if !strings.Contains(GeneratedAudioDir, "audio") {
		t.Errorf("GeneratedAudioDir should contain 'audio', got '%s'", GeneratedAudioDir)
	}

	if !strings.Contains(WebDir, "web") {
		t.Errorf("WebDir should contain 'web', got '%s'", WebDir)
	}
}
