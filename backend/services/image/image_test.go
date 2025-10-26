package image

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

func TestGenerateCharacterImageSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalImagesDir := utils.GeneratedImagesDir
	utils.GeneratedImagesDir = tmpDir
	defer func() { utils.GeneratedImagesDir = originalImagesDir }()

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake image data"))
	}))
	defer imageServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": imageServer.URL + "/image.png",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		Image: models.ImageConfig{
			Model:   "test-image",
			BaseURL: server.URL,
			APIKey:  "test-key",
			Size:    "1024x1024",
			Quality: "standard",
		},
	}

	character := models.CharacterProfile{
		Name:        "测试角色",
		Description: "角色描述",
	}

	ctx := context.Background()
	imagePath, err := GenerateCharacterImage(ctx, cfg, character, 0)
	if err != nil {
		t.Fatalf("GenerateCharacterImage failed: %v", err)
	}

	if imagePath == "" {
		t.Error("Expected non-empty image path")
	}
}

func TestGenerateCharacterImageMissingConfig(t *testing.T) {
	cfg := models.Config{
		Image: models.ImageConfig{
			Model:   "",
			BaseURL: "",
			APIKey:  "",
		},
	}

	character := models.CharacterProfile{
		Name:        "测试角色",
		Description: "角色描述",
	}

	ctx := context.Background()
	_, err := requestCharacterImage(ctx, cfg, character)
	if err == nil {
		t.Error("Expected error for missing image config")
	}
}

func TestGenerateSceneImageSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalImagesDir := utils.GeneratedImagesDir
	utils.GeneratedImagesDir = tmpDir
	defer func() { utils.GeneratedImagesDir = originalImagesDir }()

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake scene image data"))
	}))
	defer imageServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": imageServer.URL + "/scene.png",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := models.Config{
		Image: models.ImageConfig{
			Model:   "test-image",
			BaseURL: server.URL,
			APIKey:  "test-key",
		},
	}

	scene := models.Scene{
		Title:       "测试场景",
		Description: "场景描述",
		Characters:  []string{"角色1", "角色2"},
		Dialogues:   []string{"对话1"},
	}

	ctx := context.Background()
	imagePath, err := GenerateSceneImage(ctx, cfg, scene, 0)
	if err != nil {
		t.Fatalf("GenerateSceneImage failed: %v", err)
	}

	if imagePath == "" {
		t.Error("Expected non-empty image path")
	}
}

func TestExtractImageURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "Valid PNG URL",
			content: "https://example.com/image.png",
			wantErr: false,
		},
		{
			name:    "Valid JPG URL",
			content: "https://example.com/image.jpg",
			wantErr: false,
		},
		{
			name:    "URL in text",
			content: "Here is your image: https://example.com/image.png!",
			wantErr: false,
		},
		{
			name:    "Multiple URLs",
			content: "First: http://example.com/1.png Second: https://example.com/2.jpg",
			wantErr: false,
		},
		{
			name:    "No URL",
			content: "No image here",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := extractImageURL(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractImageURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && url == "" {
				t.Error("Expected non-empty URL")
			}
		})
	}
}

func TestDoImageRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Missing or incorrect Authorization header")
		}

		response := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "https://example.com/image.png",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	bodyBytes := []byte(`{"model":"test","messages":[]}`)
	ctx := context.Background()

	imageURL, err := doImageRequest(ctx, server.URL, "test-key", bodyBytes)
	if err != nil {
		t.Fatalf("doImageRequest failed: %v", err)
	}

	if imageURL == "" {
		t.Error("Expected non-empty image URL")
	}
}

func TestDoImageRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	bodyBytes := []byte(`{"model":"test","messages":[]}`)
	ctx := context.Background()

	_, err := doImageRequest(ctx, server.URL, "test-key", bodyBytes)
	if err == nil {
		t.Error("Expected error for bad request")
	}
}

func TestDoImageRequestNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"choices": []map[string]any{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	bodyBytes := []byte(`{"model":"test","messages":[]}`)
	ctx := context.Background()

	_, err := doImageRequest(ctx, server.URL, "test-key", bodyBytes)
	if err == nil {
		t.Error("Expected error for empty choices")
	}
}

func TestRemoveGeneratedImage(t *testing.T) {
	tmpDir := t.TempDir()
	originalImagesDir := utils.GeneratedImagesDir
	utils.GeneratedImagesDir = tmpDir
	defer func() { utils.GeneratedImagesDir = originalImagesDir }()

	testFile := filepath.Join(tmpDir, "test.png")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	RemoveGeneratedImage("/generated/images/test.png")

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func TestDoImageEditRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"output": map[string]any{
				"choices": []any{
					map[string]any{
						"message": map[string]any{
							"content": []any{
								map[string]any{
									"image": "https://example.com/edited.png",
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	bodyBytes := []byte(`{"model":"test"}`)
	ctx := context.Background()

	imageURL, err := doImageEditRequest(ctx, server.URL, "test-key", bodyBytes)
	if err != nil {
		t.Fatalf("doImageEditRequest failed: %v", err)
	}

	if imageURL == "" {
		t.Error("Expected non-empty image URL")
	}
}

func TestDoImageEditRequestNoImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"output": map[string]any{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	bodyBytes := []byte(`{"model":"test"}`)
	ctx := context.Background()

	_, err := doImageEditRequest(ctx, server.URL, "test-key", bodyBytes)
	if err == nil {
		t.Error("Expected error for missing image in response")
	}
}
