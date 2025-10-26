package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"taco/backend/models"
	"taco/backend/utils"
)

var audioDataURLPattern = regexp.MustCompile(`^data:audio/([^;]+);base64,`)

func GenerateSceneAudio(ctx context.Context, cfg models.Config, scene models.Scene, index int) (string, error) {
	if err := utils.EnsureDir(utils.GeneratedAudioDir); err != nil {
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
	absPath := filepath.Join(utils.GeneratedAudioDir, filename)

	if scene.AudioPath != "" {
		RemoveGeneratedAudio(scene.AudioPath)
	}

	if result.IsURL {
		if err := utils.DownloadToFile(ctx, result.Source, absPath); err != nil {
			return "", err
		}
	} else {
		if err := utils.SaveBase64ToFile(result.Source, absPath); err != nil {
			return "", err
		}
	}

	return utils.GeneratedAudioURLPrefix + filename, nil
}

func requestSceneAudio(ctx context.Context, cfg models.Config, scene models.Scene) (models.AudioResult, error) {
	voiceCfg := cfg.Voice
	if strings.TrimSpace(voiceCfg.Model) == "" {
		return models.AudioResult{}, errors.New("未配置语音模型")
	}
	if strings.TrimSpace(voiceCfg.BaseURL) == "" {
		return models.AudioResult{}, errors.New("未配置语音接口地址")
	}
	if strings.TrimSpace(voiceCfg.APIKey) == "" {
		return models.AudioResult{}, errors.New("未配置语音 API Key")
	}

	text := BuildSceneSpeechText(scene)
	if text == "" {
		return models.AudioResult{}, errors.New("缺少可用于生成语音的文本")
	}

	base := strings.TrimRight(voiceCfg.BaseURL, "/")
	if base == "" {
		return models.AudioResult{}, errors.New("语音接口地址无效")
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
		return models.AudioResult{}, err
	}

	apiURL := base + "/api/v1/services/aigc/multimodal-generation/generation"

	// 创建 HTTP 请求
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return models.AudioResult{}, err
	}
	request.Header.Set("Authorization", "Bearer "+voiceCfg.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connection", "keep-alive")
	request.ContentLength = int64(len(bodyBytes))

	// 配置 HTTP Transport 以处理大请求体
	transport := &http.Transport{
		DisableKeepAlives:   false,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		// 关键：设置足够长的响应头超时和期望继续超时
		ExpectContinueTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 600 * time.Second,
		// 添加 DialContext 以设置连接超时
		DialContext: (&net.Dialer{
			Timeout:   120 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,  // Keep-Alive 探测间隔
		}).DialContext,
	}

	// 使用标准 HTTP 客户端，设置合理的超时时间
	client := &http.Client{
		Timeout:   600 * time.Second,
		Transport: transport,
	}

	resp, err := client.Do(request)
	if err != nil {
		return models.AudioResult{}, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		errMsg := strings.TrimSpace(string(errBody))
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return models.AudioResult{}, fmt.Errorf("语音服务请求失败 (状态码: %d): %s", resp.StatusCode, errMsg)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return models.AudioResult{}, err
	}

	result, err := parseDashscopeAudio(payload)
	if err != nil {
		return models.AudioResult{}, err
	}
	return result, nil
}

func BuildSceneSpeechText(scene models.Scene) string {
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

func parseDashscopeAudio(payload map[string]any) (models.AudioResult, error) {
	if payload == nil {
		return models.AudioResult{}, errors.New("语音服务未返回音频数据")
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

	return models.AudioResult{}, errors.New("语音服务未返回音频数据")
}

func extractAudioFromMap(m map[string]any) (models.AudioResult, bool) {
	if m == nil {
		return models.AudioResult{}, false
	}

	result := models.AudioResult{Extension: "mp3"}

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

	return models.AudioResult{}, false
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

func RemoveGeneratedAudio(relPath string) {
	utils.RemoveGeneratedFile(relPath, utils.GeneratedAudioURLPrefix, utils.GeneratedAudioDir)
}
