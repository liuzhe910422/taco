package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"taco/backend/models"
)

func TestInvokeLLMSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected path '/v1/chat/completions', got '%s'", r.URL.Path)
		}

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "Test response",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
	}

	messages := []map[string]string{
		{"role": "user", "content": "test message"},
	}

	ctx := context.Background()
	result, err := InvokeLLM(ctx, cfg, messages, 0.7)
	if err != nil {
		t.Fatalf("InvokeLLM failed: %v", err)
	}

	if result != "Test response" {
		t.Errorf("Expected 'Test response', got '%s'", result)
	}
}

func TestInvokeLLMMissingAPIKey(t *testing.T) {
	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: "https://api.example.com",
			APIKey:  "",
		},
	}

	messages := []map[string]string{
		{"role": "user", "content": "test"},
	}

	ctx := context.Background()
	_, err := InvokeLLM(ctx, cfg, messages, 0.7)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestInvokeLLMEmptyBaseURL(t *testing.T) {
	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: "",
			APIKey:  "test-key",
		},
	}

	messages := []map[string]string{
		{"role": "user", "content": "test"},
	}

	ctx := context.Background()
	_, err := InvokeLLM(ctx, cfg, messages, 0.7)
	if err == nil {
		t.Error("Expected error for empty base URL")
	}
}

func TestCallLLMForCharacters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		characters := []models.CharacterProfile{
			{Name: "角色1", Description: "描述1"},
			{Name: "角色2", Description: "描述2"},
		}
		jsonData, _ := json.Marshal(characters)

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": string(jsonData),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
		CharacterCount: 5,
	}

	ctx := context.Background()
	characters, err := CallLLMForCharacters(ctx, cfg, "小说内容")
	if err != nil {
		t.Fatalf("CallLLMForCharacters failed: %v", err)
	}

	if len(characters) != 2 {
		t.Errorf("Expected 2 characters, got %d", len(characters))
	}

	if characters[0].Name != "角色1" {
		t.Errorf("Expected first character name '角色1', got '%s'", characters[0].Name)
	}
}

func TestCallLLMForCharactersLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		characters := []models.CharacterProfile{
			{Name: "角色1", Description: "描述1"},
			{Name: "角色2", Description: "描述2"},
			{Name: "角色3", Description: "描述3"},
		}
		jsonData, _ := json.Marshal(characters)

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": string(jsonData),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
		CharacterCount: 2,
	}

	ctx := context.Background()
	characters, err := CallLLMForCharacters(ctx, cfg, "小说内容")
	if err != nil {
		t.Fatalf("CallLLMForCharacters failed: %v", err)
	}

	if len(characters) != 2 {
		t.Errorf("Expected character limit of 2, got %d", len(characters))
	}
}

func TestCallLLMForScenes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scenes := []models.Scene{
			{
				Title:       "场景1",
				Characters:  []string{"角色1"},
				Description: "描述1",
				Dialogues:   []string{"对话1"},
				Narration:   "旁白1",
			},
		}
		jsonData, _ := json.Marshal(scenes)

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": string(jsonData),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		LLM: models.LLMConfig{
			Model:   "test-model",
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
		SceneCount: 10,
	}

	characters := []models.CharacterProfile{
		{Name: "角色1", Description: "描述1"},
	}

	ctx := context.Background()
	scenes, err := CallLLMForScenes(ctx, cfg, "小说内容", characters)
	if err != nil {
		t.Fatalf("CallLLMForScenes failed: %v", err)
	}

	if len(scenes) != 1 {
		t.Errorf("Expected 1 scene, got %d", len(scenes))
	}

	if scenes[0].Title != "场景1" {
		t.Errorf("Expected scene title '场景1', got '%s'", scenes[0].Title)
	}
}

func TestParseScenesJSON(t *testing.T) {
	jsonStr := `[
		{
			"title": "场景1",
			"characters": ["角色1"],
			"description": "描述",
			"dialogues": ["对话"],
			"narration": "旁白"
		}
	]`

	scenes, err := ParseScenesJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseScenesJSON failed: %v", err)
	}

	if len(scenes) != 1 {
		t.Errorf("Expected 1 scene, got %d", len(scenes))
	}

	if scenes[0].Title != "场景1" {
		t.Errorf("Expected title '场景1', got '%s'", scenes[0].Title)
	}
}

func TestParseScenesJSONWrapper(t *testing.T) {
	jsonStr := `{
		"scenes": [
			{
				"title": "场景1",
				"characters": ["角色1"],
				"description": "描述",
				"dialogues": ["对话"],
				"narration": "旁白"
			}
		]
	}`

	scenes, err := ParseScenesJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseScenesJSON failed: %v", err)
	}

	if len(scenes) != 1 {
		t.Errorf("Expected 1 scene, got %d", len(scenes))
	}
}

func TestParseScenesJSONEmpty(t *testing.T) {
	scenes, err := ParseScenesJSON("")
	if err != nil {
		t.Fatalf("ParseScenesJSON failed for empty string: %v", err)
	}

	if len(scenes) != 0 {
		t.Errorf("Expected 0 scenes for empty string, got %d", len(scenes))
	}
}

func TestNormalizeScenes(t *testing.T) {
	scenes := []models.Scene{
		{
			Title:       "  场景1  ",
			Description: "  描述  ",
			Characters:  []string{"  角色1  "},
			Dialogues:   []string{"  对话  "},
			Narration:   "  旁白  ",
		},
	}

	normalized := NormalizeScenes(scenes)

	if normalized[0].Title != "场景1" {
		t.Errorf("Title not trimmed properly")
	}
	if normalized[0].Description != "描述" {
		t.Errorf("Description not trimmed properly")
	}
	if normalized[0].Characters[0] != "角色1" {
		t.Errorf("Character not trimmed properly")
	}
}

func TestNormalizeScenesNil(t *testing.T) {
	normalized := NormalizeScenes(nil)
	if normalized == nil {
		t.Error("Expected empty slice, got nil")
	}
	if len(normalized) != 0 {
		t.Errorf("Expected 0 scenes, got %d", len(normalized))
	}
}
