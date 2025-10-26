package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"taco/backend/config"
	"taco/backend/models"
	"taco/backend/utils"
)

func TestConfigHandlerGet(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ConfigPath = filepath.Join(tmpDir, "config.json")
	defer func() {
		utils.ConfigPath = filepath.Join(utils.ProjectRoot, "config", "config.json")
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	ConfigHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var cfg models.Config
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestConfigHandlerPost(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ConfigPath = filepath.Join(tmpDir, "config.json")
	defer func() {
		utils.ConfigPath = filepath.Join(utils.ProjectRoot, "config", "config.json")
	}()

	testCfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: "https://test.com",
			APIKey:  "test-key",
		},
		Image: models.ImageConfig{
			Model:   "test-image",
			BaseURL: "https://test.com",
			APIKey:  "test-key",
		},
		Voice: models.VoiceConfig{
			Model:    "test-voice",
			BaseURL:  "https://test.com",
			APIKey:   "test-key",
			Voice:    "test",
			Language: "en",
		},
		CharacterCount: 5,
		SceneCount:     10,
	}

	body, _ := json.Marshal(testCfg)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ConfigHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestConfigHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/config", nil)
	w := httptest.NewRecorder()

	ConfigHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestUploadHandler(t *testing.T) {
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir
	defer func() {
		utils.UploadDir = filepath.Join(utils.ProjectRoot, "uploads")
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("novel", "test.txt")
	part.Write([]byte("test novel content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	UploadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["filePath"] == "" {
		t.Error("Expected non-empty file path")
	}
}

func TestUploadHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/upload", nil)
	w := httptest.NewRecorder()

	UploadHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestCharactersHandlerGet(t *testing.T) {
	tmpDir := t.TempDir()
	utils.CharactersPath = filepath.Join(tmpDir, "characters.json")
	defer func() {
		utils.CharactersPath = filepath.Join(utils.ProjectRoot, "config", "characters.json")
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/characters", nil)
	w := httptest.NewRecorder()

	CharactersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var characters []models.CharacterProfile
	if err := json.NewDecoder(w.Body).Decode(&characters); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestCharactersHandlerPost(t *testing.T) {
	tmpDir := t.TempDir()
	utils.CharactersPath = filepath.Join(tmpDir, "characters.json")
	defer func() {
		utils.CharactersPath = filepath.Join(utils.ProjectRoot, "config", "characters.json")
	}()

	characters := []models.CharacterProfile{
		{Name: "角色1", Description: "描述1"},
		{Name: "角色2", Description: "描述2"},
	}

	body, _ := json.Marshal(characters)
	req := httptest.NewRequest(http.MethodPost, "/api/characters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CharactersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestScenesHandlerGet(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ScenesPath = filepath.Join(tmpDir, "scenes.json")
	defer func() {
		utils.ScenesPath = filepath.Join(utils.ProjectRoot, "config", "scenes.json")
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/scenes", nil)
	w := httptest.NewRecorder()

	ScenesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var scenes []models.Scene
	if err := json.NewDecoder(w.Body).Decode(&scenes); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestScenesHandlerPost(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ScenesPath = filepath.Join(tmpDir, "scenes.json")
	defer func() {
		utils.ScenesPath = filepath.Join(utils.ProjectRoot, "config", "scenes.json")
	}()

	scenes := []models.Scene{
		{
			Title:       "场景1",
			Characters:  []string{"角色1"},
			Description: "描述1",
			Dialogues:   []string{"对话1"},
			Narration:   "旁白1",
		},
	}

	body, _ := json.Marshal(scenes)
	req := httptest.NewRequest(http.MethodPost, "/api/scenes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ScenesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestExtractCharactersHandlerNoNovel(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ConfigPath = filepath.Join(tmpDir, "config.json")
	defer func() {
		utils.ConfigPath = filepath.Join(utils.ProjectRoot, "config", "config.json")
	}()

	cfg := models.Config{
		NovelFile: "",
	}
	config.SaveConfig(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/characters/extract", nil)
	w := httptest.NewRecorder()

	ExtractCharactersHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "尚未上传小说文件") {
		t.Error("Expected error message about missing novel file")
	}
}

func TestExtractScenesHandlerNoNovel(t *testing.T) {
	tmpDir := t.TempDir()
	utils.ConfigPath = filepath.Join(tmpDir, "config.json")
	defer func() {
		utils.ConfigPath = filepath.Join(utils.ProjectRoot, "config", "config.json")
	}()

	cfg := models.Config{
		NovelFile: "",
	}
	config.SaveConfig(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/scenes/extract", nil)
	w := httptest.NewRecorder()

	ExtractScenesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUploadCharacterImageHandler(t *testing.T) {
	tmpDir := t.TempDir()
	utils.CharactersPath = filepath.Join(tmpDir, "characters.json")
	utils.GeneratedImagesDir = tmpDir
	defer func() {
		utils.CharactersPath = filepath.Join(utils.ProjectRoot, "config", "characters.json")
		utils.GeneratedImagesDir = filepath.Join(utils.ProjectRoot, "generated", "images")
	}()

	characters := []models.CharacterProfile{
		{Name: "角色1", Description: "描述1"},
	}
	config.SaveCharactersData(characters)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("index", "0")
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write([]byte("fake image data"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/characters/upload-image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	UploadCharacterImageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGenerateCharacterImageHandlerInvalidIndex(t *testing.T) {
	payload := map[string]int{"index": -1}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/characters/generate-image", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	GenerateCharacterImageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGenerateSceneImageHandlerInvalidIndex(t *testing.T) {
	payload := map[string]int{"index": -1}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/scenes/generate-image", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	GenerateSceneImageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGenerateSceneAudioHandlerInvalidIndex(t *testing.T) {
	payload := map[string]int{"index": -1}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/scenes/generate-audio", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	GenerateSceneAudioHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
