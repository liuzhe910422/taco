package audio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"taco/backend/models"
	"taco/backend/utils"
)

func TestBuildSceneSpeechText(t *testing.T) {
	tests := []struct {
		name     string
		scene    models.Scene
		expected string
	}{
		{
			name: "Narration priority",
			scene: models.Scene{
				Narration:   "旁白文本",
				Dialogues:   []string{"对话1", "对话2"},
				Description: "描述",
				Title:       "标题",
			},
			expected: "旁白文本",
		},
		{
			name: "Dialogues when no narration",
			scene: models.Scene{
				Narration:   "",
				Dialogues:   []string{"对话1", "对话2"},
				Description: "描述",
				Title:       "标题",
			},
			expected: "对话1 对话2",
		},
		{
			name: "Description when no narration or dialogues",
			scene: models.Scene{
				Narration:   "",
				Dialogues:   []string{},
				Description: "描述",
				Title:       "标题",
			},
			expected: "描述",
		},
		{
			name: "Title as fallback",
			scene: models.Scene{
				Narration:   "",
				Dialogues:   []string{},
				Description: "",
				Title:       "标题",
			},
			expected: "标题",
		},
		{
			name: "Empty scene",
			scene: models.Scene{
				Narration:   "",
				Dialogues:   []string{},
				Description: "",
				Title:       "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSceneSpeechText(tt.scene)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGenerateSceneAudioSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalAudioDir := utils.GeneratedAudioDir
	utils.GeneratedAudioDir = tmpDir
	defer func() { utils.GeneratedAudioDir = originalAudioDir }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"output": map[string]any{
				"audio": "data:audio/mp3;base64,SGVsbG8gV29ybGQ=",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		Voice: models.VoiceConfig{
			Model:    "test-tts",
			BaseURL:  server.URL,
			APIKey:   "test-key",
			Voice:    "test-voice",
			Language: "Chinese",
		},
	}

	scene := models.Scene{
		Narration: "测试旁白",
	}

	ctx := context.Background()
	audioPath, err := GenerateSceneAudio(ctx, cfg, scene, 0)
	if err != nil {
		t.Fatalf("GenerateSceneAudio failed: %v", err)
	}

	if audioPath == "" {
		t.Error("Expected non-empty audio path")
	}

	if filepath.Ext(audioPath) != ".mp3" {
		t.Errorf("Expected .mp3 extension, got '%s'", filepath.Ext(audioPath))
	}
}

func TestGenerateSceneAudioMissingConfig(t *testing.T) {
	cfg := models.Config{
		Voice: models.VoiceConfig{
			Model:   "",
			BaseURL: "",
			APIKey:  "",
		},
	}

	scene := models.Scene{
		Narration: "测试旁白",
	}

	ctx := context.Background()
	_, err := GenerateSceneAudio(ctx, cfg, scene, 0)
	if err == nil {
		t.Error("Expected error for missing voice config")
	}
}

func TestGenerateSceneAudioEmptyText(t *testing.T) {
	cfg := models.Config{
		Voice: models.VoiceConfig{
			Model:    "test-tts",
			BaseURL:  "https://api.example.com",
			APIKey:   "test-key",
			Voice:    "test-voice",
			Language: "Chinese",
		},
	}

	scene := models.Scene{}

	ctx := context.Background()
	_, err := GenerateSceneAudio(ctx, cfg, scene, 0)
	if err == nil {
		t.Error("Expected error for empty scene text")
	}
}

func TestParseDashscopeAudio(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]any
		wantURL  bool
		wantErr  bool
	}{
		{
			name: "Valid audio in output.audio",
			payload: map[string]any{
				"output": map[string]any{
					"audio": "http://example.com/audio.mp3",
				},
			},
			wantURL: true,
			wantErr: false,
		},
		{
			name: "Valid base64 audio",
			payload: map[string]any{
				"output": map[string]any{
					"audio": "data:audio/mp3;base64,SGVsbG8=",
				},
			},
			wantURL: false,
			wantErr: false,
		},
		{
			name: "Audio URL in results",
			payload: map[string]any{
				"output": map[string]any{
					"results": []any{
						map[string]any{
							"audio_url": "http://example.com/audio.wav",
						},
					},
				},
			},
			wantURL: true,
			wantErr: false,
		},
		{
			name:    "Empty payload",
			payload: nil,
			wantErr: true,
		},
		{
			name:    "No audio data",
			payload: map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDashscopeAudio(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDashscopeAudio() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.IsURL != tt.wantURL {
				t.Errorf("parseDashscopeAudio() IsURL = %v, want %v", result.IsURL, tt.wantURL)
			}
		})
	}
}

func TestInferAudioExtensionFromURL(t *testing.T) {
	tests := []struct {
		url      string
		fallback string
		expected string
	}{
		{"http://example.com/audio.mp3", "", "mp3"},
		{"http://example.com/audio.wav", "", "wav"},
		{"http://example.com/audio.ogg", "", "ogg"},
		{"http://example.com/audio", "m4a", "m4a"},
		{"http://example.com/test", "", "mp3"},
	}

	for _, tt := range tests {
		result := inferAudioExtensionFromURL(tt.url, tt.fallback)
		if result != tt.expected {
			t.Errorf("inferAudioExtensionFromURL(%s, %s) = %s, expected %s",
				tt.url, tt.fallback, result, tt.expected)
		}
	}
}

func TestInferAudioExtensionFromData(t *testing.T) {
	tests := []struct {
		data     string
		fallback string
		expected string
	}{
		{"data:audio/mp3;base64,SGVsbG8=", "", "mp3"},
		{"data:audio/wav;base64,SGVsbG8=", "", "wav"},
		{"SGVsbG8=", "ogg", "ogg"},
		{"SGVsbG8=", "", "mp3"},
	}

	for _, tt := range tests {
		result := inferAudioExtensionFromData(tt.data, tt.fallback)
		if result != tt.expected {
			t.Errorf("inferAudioExtensionFromData(%s, %s) = %s, expected %s",
				tt.data, tt.fallback, result, tt.expected)
		}
	}
}

func TestRemoveGeneratedAudio(t *testing.T) {
	tmpDir := t.TempDir()
	originalAudioDir := utils.GeneratedAudioDir
	utils.GeneratedAudioDir = tmpDir
	defer func() { utils.GeneratedAudioDir = originalAudioDir }()

	testFile := filepath.Join(tmpDir, "test.mp3")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	RemoveGeneratedAudio("/generated/audio/test.mp3")

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}
