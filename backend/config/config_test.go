package config

import (
	"path/filepath"
	"testing"

	"taco/backend/models"
)

func TestLoadConfigDefault(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.LLM.Model != "gpt-4o-mini" {
		t.Errorf("Expected default LLM model 'gpt-4o-mini', got '%s'", cfg.LLM.Model)
	}
	if cfg.Image.Model != "gpt-4o-image" {
		t.Errorf("Expected default Image model 'gpt-4o-image', got '%s'", cfg.Image.Model)
	}
	if cfg.Voice.Model != "qwen3-tts-flash" {
		t.Errorf("Expected default Voice model 'qwen3-tts-flash', got '%s'", cfg.Voice.Model)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	testCfg := models.Config{
		NovelFile: "/test/novel.txt",
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: "https://test.com",
			APIKey:  "test-key",
		},
		CharacterCount: 3,
		SceneCount:     5,
	}

	err := SaveConfig(testCfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loadedCfg.NovelFile != testCfg.NovelFile {
		t.Errorf("NovelFile mismatch. Expected '%s', got '%s'", testCfg.NovelFile, loadedCfg.NovelFile)
	}
	if loadedCfg.LLM.Model != testCfg.LLM.Model {
		t.Errorf("LLM Model mismatch. Expected '%s', got '%s'", testCfg.LLM.Model, loadedCfg.LLM.Model)
	}
	if loadedCfg.CharacterCount != testCfg.CharacterCount {
		t.Errorf("CharacterCount mismatch. Expected %d, got %d", testCfg.CharacterCount, loadedCfg.CharacterCount)
	}
}

func TestValidateConfigSuccess(t *testing.T) {
	validCfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "gpt-4",
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
		Image: models.ImageConfig{
			Model:   "dalle",
			BaseURL: "https://api.example.com",
			APIKey:  "image-key",
		},
		Voice: models.VoiceConfig{
			Model:    "tts-1",
			BaseURL:  "https://api.example.com",
			APIKey:   "voice-key",
			Voice:    "alloy",
			Language: "en",
		},
		CharacterCount: 5,
		SceneCount:     10,
	}

	err := ValidateConfig(validCfg)
	if err != nil {
		t.Errorf("ValidateConfig failed for valid config: %v", err)
	}
}

func TestValidateConfigMissingLLMModel(t *testing.T) {
	invalidCfg := models.Config{
		LLM: models.LLMConfig{
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
	}

	err := ValidateConfig(invalidCfg)
	if err == nil {
		t.Error("Expected validation error for missing LLM model")
	}
}

func TestValidateConfigNegativeCharacterCount(t *testing.T) {
	invalidCfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "gpt-4",
			BaseURL: "https://api.example.com",
			APIKey:  "test-key",
		},
		Image: models.ImageConfig{
			Model:   "dalle",
			BaseURL: "https://api.example.com",
			APIKey:  "image-key",
		},
		Voice: models.VoiceConfig{
			Model:    "tts-1",
			BaseURL:  "https://api.example.com",
			APIKey:   "voice-key",
			Voice:    "alloy",
			Language: "en",
		},
		CharacterCount: -1,
	}

	err := ValidateConfig(invalidCfg)
	if err == nil {
		t.Error("Expected validation error for negative character count")
	}
}

func TestLoadCharactersData(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := charactersPath
	charactersPath = filepath.Join(tmpDir, "characters.json")
	defer func() { charactersPath = originalPath }()

	characters, err := LoadCharactersData()
	if err != nil {
		t.Fatalf("LoadCharactersData failed: %v", err)
	}

	if characters == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(characters) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(characters))
	}
}

func TestSaveAndLoadCharactersData(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := charactersPath
	charactersPath = filepath.Join(tmpDir, "characters.json")
	defer func() { charactersPath = originalPath }()

	testCharacters := []models.CharacterProfile{
		{Name: "角色1", Description: "描述1", ImagePath: "/images/1.png"},
		{Name: "角色2", Description: "描述2", ImagePath: "/images/2.png"},
	}

	err := SaveCharactersData(testCharacters)
	if err != nil {
		t.Fatalf("SaveCharactersData failed: %v", err)
	}

	loaded, err := LoadCharactersData()
	if err != nil {
		t.Fatalf("LoadCharactersData failed: %v", err)
	}

	if len(loaded) != len(testCharacters) {
		t.Errorf("Expected %d characters, got %d", len(testCharacters), len(loaded))
	}

	for i := range testCharacters {
		if loaded[i].Name != testCharacters[i].Name {
			t.Errorf("Character %d name mismatch. Expected '%s', got '%s'", i, testCharacters[i].Name, loaded[i].Name)
		}
	}
}

func TestLoadScenesData(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := scenesPath
	scenesPath = filepath.Join(tmpDir, "scenes.json")
	defer func() { scenesPath = originalPath }()

	scenes, err := LoadScenesData()
	if err != nil {
		t.Fatalf("LoadScenesData failed: %v", err)
	}

	if scenes == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(scenes) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(scenes))
	}
}

func TestSaveAndLoadScenesData(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := scenesPath
	scenesPath = filepath.Join(tmpDir, "scenes.json")
	defer func() { scenesPath = originalPath }()

	testScenes := []models.Scene{
		{
			Title:       "场景1",
			Characters:  []string{"角色1", "角色2"},
			Description: "描述1",
			Dialogues:   []string{"对话1"},
			Narration:   "旁白1",
		},
	}

	err := SaveScenesData(testScenes)
	if err != nil {
		t.Fatalf("SaveScenesData failed: %v", err)
	}

	loaded, err := LoadScenesData()
	if err != nil {
		t.Fatalf("LoadScenesData failed: %v", err)
	}

	if len(loaded) != len(testScenes) {
		t.Errorf("Expected %d scenes, got %d", len(testScenes), len(loaded))
	}

	if loaded[0].Title != testScenes[0].Title {
		t.Errorf("Scene title mismatch. Expected '%s', got '%s'", testScenes[0].Title, loaded[0].Title)
	}
}

func TestNormalizeScenes(t *testing.T) {
	scenes := []models.Scene{
		{
			Title:       "  场景1  ",
			Description: "  描述  ",
			Characters:  []string{"  角色1  ", "  角色2  "},
			Dialogues:   []string{"  对话1  "},
			Narration:   "  旁白  ",
		},
	}

	normalized := NormalizeScenes(scenes)

	if normalized[0].Title != "场景1" {
		t.Errorf("Title not trimmed. Expected '场景1', got '%s'", normalized[0].Title)
	}
	if normalized[0].Description != "描述" {
		t.Errorf("Description not trimmed. Expected '描述', got '%s'", normalized[0].Description)
	}
	if normalized[0].Characters[0] != "角色1" {
		t.Errorf("Character not trimmed. Expected '角色1', got '%s'", normalized[0].Characters[0])
	}
	if normalized[0].Dialogues[0] != "对话1" {
		t.Errorf("Dialogue not trimmed. Expected '对话1', got '%s'", normalized[0].Dialogues[0])
	}
	if normalized[0].Narration != "旁白" {
		t.Errorf("Narration not trimmed. Expected '旁白', got '%s'", normalized[0].Narration)
	}
}

func TestNormalizeScenesNil(t *testing.T) {
	normalized := NormalizeScenes(nil)
	if normalized == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(normalized) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(normalized))
	}
}
