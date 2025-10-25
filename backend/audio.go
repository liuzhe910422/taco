package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var audioDataURLPattern = regexp.MustCompile(`^data:audio/([^;]+);base64,`)

func generateSceneAudio(ctx context.Context, cfg Config, scene Scene, index int) (string, error) {
	if err := ensureDir(generatedAudioDir); err != nil {
		return "", err
	}

	result, err := requestSceneAudio(ctx, cfg, scene)
	if err != nil {
		return "", err
	}

	ext := ".mp3"
	if result.Extension != "" {
		ext = "." + strings.TrimPrefix(result.Extension, ".")
	}
	filename := fmt.Sprintf("scene_%02d_%d%s", index+1, time.Now().Unix(), ext)
	absPath := filepath.Join(generatedAudioDir, filename)

	if scene.AudioPath != "" {
		removeGeneratedAudio(scene.AudioPath)
	}

	if result.IsURL {
		if err := downloadToFile(ctx, result.Source, absPath); err != nil {
			return "", err
		}
	} else {
		if err := saveBase64ToFile(result.Source, absPath); err != nil {
			return "", err
		}
	}

	return generatedAudioURLPrefix + filename, nil
}

func requestSceneAudio(ctx context.Context, cfg Config, scene Scene) (audioResult, error) {
	voiceCfg := cfg.Voice
	if strings.TrimSpace(voiceCfg.Model) == "" {
		return audioResult{}, errors.New("未配置语音模型")
	}
	if strings.TrimSpace(voiceCfg.BaseURL) == "" {
		return audioResult{}, errors.New("未配置语音接口地址")
	}
	if strings.TrimSpace(voiceCfg.APIKey) == "" {
		return audioResult{}, errors.New("未配置语音 API Key")
	}

	text := buildSceneSpeechText(scene)
	if text == "" {
		return audioResult{}, errors.New("缺少可用于生成语音的文本")
	}

	base := strings.TrimRight(voiceCfg.BaseURL, "/")
	if base == "" {
		return audioResult{}, errors.New("语音接口地址无效")
	}

	input := map[string]any{
		"text": text,
	}
	if strings.TrimSpace(voiceCfg.Voice) != "" {
		input["voice"] = voiceCfg.Voice
	}
	if strings.TrimSpace(voiceCfg.Language) != "" {
		input["language_type"] = voiceCfg.Language
	}

	reqBody := map[string]any{
		"model": voiceCfg.Model,
		"input": input,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return audioResult{}, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 240*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 2400 * time.Second}
	apiURL := base + "/v1/chat/completions"
	request, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return audioResult{}, err
	}
	request.Header.Set("Authorization", "Bearer "+voiceCfg.APIKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return audioResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return audioResult{}, fmt.Errorf("语音服务请求失败: %s", strings.TrimSpace(string(errBody)))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return audioResult{}, err
	}

	result, err := parseDashscopeAudio(payload)
	if err != nil {
		return audioResult{}, err
	}
	return result, nil
}

func buildSceneSpeechText(scene Scene) string {
	if txt := strings.TrimSpace(scene.Narration); txt != "" {
		return txt
	}
	if len(scene.Dialogues) > 0 {
		return strings.Join(scene.Dialogues, " ")
	}
	if txt := strings.TrimSpace(scene.Description); txt != "" {
		return txt
	}
	return strings.TrimSpace(scene.Title)
}

func parseDashscopeAudio(payload map[string]any) (audioResult, error) {
	if payload == nil {
		return audioResult{}, errors.New("语音服务未返回音频数据")
	}

	if output, ok := payload["output"].(map[string]any); ok {
		if audioObj, ok := output["audio"].(map[string]any); ok {
			if res, ok := extractAudioFromMap(audioObj); ok {
				return res, nil
			}
		}
		if res, ok := extractAudioFromMap(output); ok {
			return res, nil
		}
		if results, ok := output["results"].([]any); ok {
			for _, item := range results {
				if m, ok := item.(map[string]any); ok {
					if res, ok := extractAudioFromMap(m); ok {
						return res, nil
					}
				}
			}
		}
	}

	if data, ok := payload["data"].(map[string]any); ok {
		if res, ok := extractAudioFromMap(data); ok {
			return res, nil
		}
	}

	if results, ok := payload["results"].([]any); ok {
		for _, item := range results {
			if m, ok := item.(map[string]any); ok {
				if res, ok := extractAudioFromMap(m); ok {
					return res, nil
				}
			}
		}
	}

	return audioResult{}, errors.New("语音服务未返回音频数据")
}

func extractAudioFromMap(m map[string]any) (audioResult, bool) {
	if m == nil {
		return audioResult{}, false
	}

	result := audioResult{Extension: "mp3"}

	if ext, ok := stringFromMap(m, "format", "audio_format", "audioExt"); ok && ext != "" {
		result.Extension = strings.TrimPrefix(ext, ".")
	}

	if audioStr, ok := stringFromMap(m, "audio", "audio_data"); ok && strings.TrimSpace(audioStr) != "" {
		result.Source = strings.TrimSpace(audioStr)
		if strings.HasPrefix(strings.ToLower(result.Source), "http://") || strings.HasPrefix(strings.ToLower(result.Source), "https://") {
			result.IsURL = true
			result.Extension = inferAudioExtensionFromURL(result.Source, result.Extension)
			return result, true
		}
		result.IsURL = false
		result.Extension = inferAudioExtensionFromData(result.Source, result.Extension)
		return result, true
	}

	if urlStr, ok := stringFromMap(m, "audio_url", "url"); ok && strings.TrimSpace(urlStr) != "" {
		result.Source = strings.TrimSpace(urlStr)
		result.IsURL = true
		result.Extension = inferAudioExtensionFromURL(result.Source, result.Extension)
		return result, true
	}

	return audioResult{}, false
}

func stringFromMap(m map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case string:
				return v, true
			case fmt.Stringer:
				return v.String(), true
			}
		}
	}
	return "", false
}

func inferAudioExtensionFromData(data string, fallback string) string {
	fallback = strings.TrimPrefix(fallback, ".")
	if matches := audioDataURLPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(data))); len(matches) == 2 {
		mime := matches[1]
		if mime != "" {
			return mime
		}
	}
	if fallback != "" {
		return fallback
	}
	return "mp3"
}

func inferAudioExtensionFromURL(url string, fallback string) string {
	urlLower := strings.ToLower(url)
	known := []string{".mp3", ".wav", ".ogg", ".m4a", ".aac"}
	for _, ext := range known {
		if strings.Contains(urlLower, ext) {
			return strings.TrimPrefix(ext, ".")
		}
	}
	if fallback != "" {
		return strings.TrimPrefix(fallback, ".")
	}
	return "mp3"
}

func removeGeneratedAudio(relPath string) {
	removeGeneratedFile(relPath, generatedAudioURLPrefix, generatedAudioDir)
}
