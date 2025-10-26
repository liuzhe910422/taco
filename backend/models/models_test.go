package models

import (
	"testing"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		NovelFile:      "/path/to/novel.txt",
		CharacterCount: 5,
		SceneCount:     10,
		VideoModel:     "test-model",
		AnimeStyle:     "anime",
	}

	if cfg.NovelFile != "/path/to/novel.txt" {
		t.Errorf("Expected NovelFile to be '/path/to/novel.txt', got '%s'", cfg.NovelFile)
	}
	if cfg.CharacterCount != 5 {
		t.Errorf("Expected CharacterCount to be 5, got %d", cfg.CharacterCount)
	}
	if cfg.SceneCount != 10 {
		t.Errorf("Expected SceneCount to be 10, got %d", cfg.SceneCount)
	}
}

func TestLLMConfig(t *testing.T) {
	llm := LLMConfig{
		Model:   "gpt-4",
		BaseURL: "https://api.example.com",
		APIKey:  "test-key",
	}

	if llm.Model != "gpt-4" {
		t.Errorf("Expected Model to be 'gpt-4', got '%s'", llm.Model)
	}
	if llm.BaseURL != "https://api.example.com" {
		t.Errorf("Expected BaseURL to be 'https://api.example.com', got '%s'", llm.BaseURL)
	}
	if llm.APIKey != "test-key" {
		t.Errorf("Expected APIKey to be 'test-key', got '%s'", llm.APIKey)
	}
}

func TestImageConfig(t *testing.T) {
	img := ImageConfig{
		Model:   "dalle",
		BaseURL: "https://api.example.com",
		APIKey:  "image-key",
		Size:    "1024x1024",
		Quality: "hd",
	}

	if img.Size != "1024x1024" {
		t.Errorf("Expected Size to be '1024x1024', got '%s'", img.Size)
	}
	if img.Quality != "hd" {
		t.Errorf("Expected Quality to be 'hd', got '%s'", img.Quality)
	}
}

func TestVoiceConfig(t *testing.T) {
	voice := VoiceConfig{
		Model:     "tts-1",
		BaseURL:   "https://api.example.com",
		APIKey:    "voice-key",
		Voice:     "alloy",
		Language:  "en",
		OutputDir: "/output",
	}

	if voice.Voice != "alloy" {
		t.Errorf("Expected Voice to be 'alloy', got '%s'", voice.Voice)
	}
	if voice.Language != "en" {
		t.Errorf("Expected Language to be 'en', got '%s'", voice.Language)
	}
	if voice.OutputDir != "/output" {
		t.Errorf("Expected OutputDir to be '/output', got '%s'", voice.OutputDir)
	}
}

func TestCharacterProfile(t *testing.T) {
	char := CharacterProfile{
		Name:        "测试角色",
		Description: "一个测试角色",
		ImagePath:   "/images/char.png",
	}

	if char.Name != "测试角色" {
		t.Errorf("Expected Name to be '测试角色', got '%s'", char.Name)
	}
	if char.Description != "一个测试角色" {
		t.Errorf("Expected Description to be '一个测试角色', got '%s'", char.Description)
	}
	if char.ImagePath != "/images/char.png" {
		t.Errorf("Expected ImagePath to be '/images/char.png', got '%s'", char.ImagePath)
	}
}

func TestScene(t *testing.T) {
	scene := Scene{
		Title:       "场景一",
		Characters:  []string{"角色A", "角色B"},
		Description: "场景描述",
		Dialogues:   []string{"对话1", "对话2"},
		Narration:   "旁白",
		ImagePath:   "/images/scene.png",
		AudioPath:   "/audio/scene.mp3",
	}

	if scene.Title != "场景一" {
		t.Errorf("Expected Title to be '场景一', got '%s'", scene.Title)
	}
	if len(scene.Characters) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(scene.Characters))
	}
	if len(scene.Dialogues) != 2 {
		t.Errorf("Expected 2 dialogues, got %d", len(scene.Dialogues))
	}
	if scene.Narration != "旁白" {
		t.Errorf("Expected Narration to be '旁白', got '%s'", scene.Narration)
	}
}

func TestAudioResult(t *testing.T) {
	result := AudioResult{
		Source:    "http://example.com/audio.mp3",
		IsURL:     true,
		Extension: "mp3",
	}

	if !result.IsURL {
		t.Error("Expected IsURL to be true")
	}
	if result.Extension != "mp3" {
		t.Errorf("Expected Extension to be 'mp3', got '%s'", result.Extension)
	}
	if result.Source != "http://example.com/audio.mp3" {
		t.Errorf("Expected Source to be 'http://example.com/audio.mp3', got '%s'", result.Source)
	}
}
