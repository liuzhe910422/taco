package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"taco/backend/models"
	"taco/backend/utils"
)

var imageURLPattern = regexp.MustCompile(`https?://[^\s)]+`)

func GenerateCharacterImage(ctx context.Context, cfg models.Config, character models.CharacterProfile, index int) (string, error) {
	if err := utils.EnsureDir(utils.GeneratedImagesDir); err != nil {
		return "", err
	}

	imageRef, err := requestCharacterImage(ctx, cfg, character)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("character_%02d_%d.png", index+1, time.Now().Unix())
	absPath := filepath.Join(utils.GeneratedImagesDir, filename)

	if character.ImagePath != "" {
		RemoveGeneratedImage(character.ImagePath)
	}

	if strings.HasPrefix(strings.ToLower(imageRef), "http://") || strings.HasPrefix(strings.ToLower(imageRef), "https://") {
		if err := utils.DownloadToFile(ctx, imageRef, absPath); err != nil {
			return "", err
		}
	} else {
		if err := utils.SaveBase64ToFile(imageRef, absPath); err != nil {
			return "", err
		}
	}

	return utils.GeneratedImagesURLPrefix + filename, nil
}

func requestCharacterImage(ctx context.Context, cfg models.Config, character models.CharacterProfile) (string, error) {
	imageCfg := cfg.Image
	if strings.TrimSpace(imageCfg.Model) == "" {
		imageCfg.Model = "gpt-4o-image"
	}
	if strings.TrimSpace(imageCfg.BaseURL) == "" {
		imageCfg.BaseURL = cfg.LLM.BaseURL
	}
	if strings.TrimSpace(imageCfg.APIKey) == "" {
		imageCfg.APIKey = cfg.LLM.APIKey
	}

	if strings.TrimSpace(imageCfg.Model) == "" {
		return "", errors.New("未配置图像模型")
	}
	if strings.TrimSpace(imageCfg.BaseURL) == "" {
		return "", errors.New("未配置图像接口地址")
	}
	if strings.TrimSpace(imageCfg.APIKey) == "" {
		return "", errors.New("未配置图像 API Key")
	}

	base := strings.TrimRight(imageCfg.BaseURL, "/")
	if base == "" {
		return "", errors.New("图像接口地址无效")
	}

	promptBuilder := strings.Builder{}
	if styleDesc := strings.TrimSpace(cfg.AnimeStyle); styleDesc != "" {
		promptBuilder.WriteString("以")
		promptBuilder.WriteString(styleDesc)
		promptBuilder.WriteString("绘制角色立绘，要求：")
	} else {
		promptBuilder.WriteString("以高质量动漫风格绘制角色立绘，要求：")
	}
	promptBuilder.WriteString("角色名称：")
	promptBuilder.WriteString(character.Name)
	promptBuilder.WriteString("。角色特征描述：")
	promptBuilder.WriteString(character.Description)
	promptBuilder.WriteString("。画面需呈现明显的动漫风格、清晰的角色特征、柔和光效与细腻线条，适合作为角色头像或立绘使用。")

	messages := []map[string]any{
		{
			"role":    "user",
			"content": promptBuilder.String(),
		},
	}

	reqBody := map[string]any{
		"model":    imageCfg.Model,
		"messages": messages,
		"n":        1,
	}

	if s := strings.TrimSpace(imageCfg.Size); s != "" {
		reqBody["size"] = s
	}
	if q := strings.TrimSpace(imageCfg.Quality); q != "" {
		reqBody["quality"] = q
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	log.Printf("[图像 API] 请求体大小: %d 字节", len(bodyBytes))

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			waitTime := time.Duration(attempt-1) * 2 * time.Second
			log.Printf("[图像 API] 第 %d 次重试，等待 %v...", attempt, waitTime)
			time.Sleep(waitTime)
		}

		log.Printf("[图像 API] 发起请求 (尝试 %d/%d): %s/v1/chat/completions", attempt, maxRetries, base)

		result, err := doImageRequest(ctx, base, imageCfg.APIKey, bodyBytes)
		if err == nil {
			if attempt > 1 {
				log.Printf("[图像 API] 重试成功！")
			}
			return result, nil
		}

		lastErr = err
		log.Printf("[图像 API] 尝试 %d/%d 失败: %v", attempt, maxRetries, err)
	}

	return "", fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}

func GenerateSceneImage(ctx context.Context, cfg models.Config, scene models.Scene, index int) (string, error) {
	if err := utils.EnsureDir(utils.GeneratedImagesDir); err != nil {
		return "", err
	}

	imageRef, err := requestSceneImage(ctx, cfg, scene)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("scene_%02d_%d.png", index+1, time.Now().Unix())
	absPath := filepath.Join(utils.GeneratedImagesDir, filename)

	if scene.ImagePath != "" {
		RemoveGeneratedImage(scene.ImagePath)
	}

	if strings.HasPrefix(strings.ToLower(imageRef), "http://") || strings.HasPrefix(strings.ToLower(imageRef), "https://") {
		if err := utils.DownloadToFile(ctx, imageRef, absPath); err != nil {
			return "", err
		}
	} else {
		if err := utils.SaveBase64ToFile(imageRef, absPath); err != nil {
			return "", err
		}
	}

	return utils.GeneratedImagesURLPrefix + filename, nil
}

func requestSceneImage(ctx context.Context, cfg models.Config, scene models.Scene) (string, error) {
	imageCfg := cfg.Image
	if strings.TrimSpace(imageCfg.Model) == "" {
		imageCfg.Model = "gpt-4o-image"
	}
	if strings.TrimSpace(imageCfg.BaseURL) == "" {
		imageCfg.BaseURL = cfg.LLM.BaseURL
	}
	if strings.TrimSpace(imageCfg.APIKey) == "" {
		imageCfg.APIKey = cfg.LLM.APIKey
	}

	if strings.TrimSpace(imageCfg.Model) == "" {
		return "", errors.New("未配置图像模型")
	}
	if strings.TrimSpace(imageCfg.BaseURL) == "" {
		return "", errors.New("未配置图像接口地址")
	}
	if strings.TrimSpace(imageCfg.APIKey) == "" {
		return "", errors.New("未配置图像 API Key")
	}

	base := strings.TrimRight(imageCfg.BaseURL, "/")
	if base == "" {
		return "", errors.New("图像接口地址无效")
	}

	characterLine := strings.Join(scene.Characters, "、")
	dialogueSnippet := ""
	if len(scene.Dialogues) > 0 {
		dialogueSnippet = strings.Join(scene.Dialogues, " ")
	}

	promptBuilder := strings.Builder{}
	if styleDesc := strings.TrimSpace(cfg.AnimeStyle); styleDesc != "" {
		promptBuilder.WriteString("以")
		promptBuilder.WriteString(styleDesc)
		promptBuilder.WriteString("绘制以下场景，强调电影级光影、鲜明色彩与角色表情。")
	} else {
		promptBuilder.WriteString("以高质量动漫风格绘制以下场景，强调电影级光影、鲜明色彩与角色表情。")
	}
	promptBuilder.WriteString("场景描述：")
	promptBuilder.WriteString(scene.Description)
	if characterLine != "" {
		promptBuilder.WriteString("。出场角色：")
		promptBuilder.WriteString(characterLine)
	}
	if dialogueSnippet != "" {
		promptBuilder.WriteString("。对话氛围参考：")
		promptBuilder.WriteString(dialogueSnippet)
	}
	promptBuilder.WriteString("。画面需呈现明显的动漫风格、柔和光效与细腻线条。")

	messages := []map[string]any{
		{
			"role":    "user",
			"content": promptBuilder.String(),
		},
	}

	reqBody := map[string]any{
		"model":    imageCfg.Model,
		"messages": messages,
		"n":        1,
	}

	if s := strings.TrimSpace(imageCfg.Size); s != "" {
		reqBody["size"] = s
	}
	if q := strings.TrimSpace(imageCfg.Quality); q != "" {
		reqBody["quality"] = q
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	log.Printf("[图像 API] 请求体大小: %d 字节", len(bodyBytes))

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			waitTime := time.Duration(attempt-1) * 2 * time.Second
			log.Printf("[图像 API] 第 %d 次重试，等待 %v...", attempt, waitTime)
			time.Sleep(waitTime)
		}

		log.Printf("[图像 API] 发起请求 (尝试 %d/%d): %s/v1/chat/completions", attempt, maxRetries, base)

		result, err := doImageRequest(ctx, base, imageCfg.APIKey, bodyBytes)
		if err == nil {
			if attempt > 1 {
				log.Printf("[图像 API] 重试成功！")
			}
			return result, nil
		}

		lastErr = err
		log.Printf("[图像 API] 尝试 %d/%d 失败: %v", attempt, maxRetries, err)
	}

	return "", fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}

func GenerateSceneImageWithCharacters(ctx context.Context, cfg models.Config, scene models.Scene, characters []models.CharacterProfile, index int) (string, error) {
	if err := utils.EnsureDir(utils.GeneratedImagesDir); err != nil {
		return "", err
	}

	imageRef, err := requestSceneImageWithCharacters(ctx, cfg, scene, characters)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("scene_%02d_%d.png", index+1, time.Now().Unix())
	absPath := filepath.Join(utils.GeneratedImagesDir, filename)

	if scene.ImagePath != "" {
		RemoveGeneratedImage(scene.ImagePath)
	}

	if strings.HasPrefix(strings.ToLower(imageRef), "http://") || strings.HasPrefix(strings.ToLower(imageRef), "https://") {
		if err := utils.DownloadToFile(ctx, imageRef, absPath); err != nil {
			return "", err
		}
	} else {
		if err := utils.SaveBase64ToFile(imageRef, absPath); err != nil {
			return "", err
		}
	}

	return utils.GeneratedImagesURLPrefix + filename, nil
}

func requestSceneImageWithCharacters(ctx context.Context, cfg models.Config, scene models.Scene, allCharacters []models.CharacterProfile) (string, error) {
	imageEditCfg := cfg.ImageEdit
	if strings.TrimSpace(imageEditCfg.Model) == "" {
		imageEditCfg.Model = "qwen-image-edit"
	}
	if strings.TrimSpace(imageEditCfg.BaseURL) == "" {
		imageEditCfg.BaseURL = cfg.Image.BaseURL
	}
	if strings.TrimSpace(imageEditCfg.APIKey) == "" {
		imageEditCfg.APIKey = cfg.Image.APIKey
	}

	if strings.TrimSpace(imageEditCfg.Model) == "" {
		return "", errors.New("未配置图像编辑模型")
	}
	if strings.TrimSpace(imageEditCfg.BaseURL) == "" {
		return "", errors.New("未配置图像编辑接口地址")
	}
	if strings.TrimSpace(imageEditCfg.APIKey) == "" {
		return "", errors.New("未配置图像编辑 API Key")
	}

	base := strings.TrimRight(imageEditCfg.BaseURL, "/")
	if base == "" {
		return "", errors.New("图像编辑接口地址无效")
	}

	characterImages := make(map[string]string)
	for _, charName := range scene.Characters {
		for _, char := range allCharacters {
			if char.Name == charName && char.ImagePath != "" {
				var imagePath string
				if strings.HasPrefix(char.ImagePath, utils.GeneratedImagesURLPrefix) {
					filename := strings.TrimPrefix(char.ImagePath, utils.GeneratedImagesURLPrefix)
					imagePath = filepath.Join(utils.GeneratedImagesDir, filename)
				} else if strings.HasPrefix(char.ImagePath, "/") {
					imagePath = char.ImagePath
				} else {
					imagePath = filepath.Join(utils.GeneratedImagesDir, filepath.Base(char.ImagePath))
				}

				log.Printf("[INFO] 读取角色 %s 的图片: %s", charName, imagePath)
				imageData, err := os.ReadFile(imagePath)
				if err != nil {
					log.Printf("[WARNING] 无法读取角色 %s 的图片: %v (路径: %s)", charName, err, imagePath)
					continue
				}
				log.Printf("[INFO] 成功读取角色 %s 的图片，大小: %d 字节", charName, len(imageData))
				base64Image := base64.StdEncoding.EncodeToString(imageData)
				characterImages[charName] = base64Image
				break
			}
		}
	}

	log.Printf("[INFO] 成功加载 %d 个角色的图片", len(characterImages))

	contentArray := []map[string]any{}
	textBuilder := strings.Builder{}

	if len(characterImages) > 0 {
		imageIndex := 1
		for charName, base64Image := range characterImages {
			contentArray = append(contentArray, map[string]any{
				"type": "image_url",
				"image_url": map[string]string{
					"url": fmt.Sprintf("data:image/png;base64,%s", base64Image),
				},
			})
			textBuilder.WriteString(fmt.Sprintf("图%d是%s。", imageIndex, charName))
			imageIndex++
		}
	}

	textBuilder.WriteString("请根据上述人物形象，")
	if styleDesc := strings.TrimSpace(cfg.AnimeStyle); styleDesc != "" {
		textBuilder.WriteString("以")
		textBuilder.WriteString(styleDesc)
		textBuilder.WriteString("绘制以下场景，强调电影级光影、鲜明色彩与角色表情。")
	} else {
		textBuilder.WriteString("以高质量动漫风格绘制以下场景，强调电影级光影、鲜明色彩与角色表情。")
	}
	textBuilder.WriteString("场景描述：")
	textBuilder.WriteString(scene.Description)

	characterLine := strings.Join(scene.Characters, "、")
	if characterLine != "" {
		textBuilder.WriteString("。出场角色：")
		textBuilder.WriteString(characterLine)
	}

	if len(scene.Dialogues) > 0 {
		dialogueSnippet := strings.Join(scene.Dialogues, " ")
		textBuilder.WriteString("。对话氛围参考：")
		textBuilder.WriteString(dialogueSnippet)
	}

	textBuilder.WriteString("。画面需呈现明显的动漫风格、柔和光效与细腻线条。")

	contentArray = append(contentArray, map[string]any{
		"type": "text",
		"text": textBuilder.String(),
	})

	messages := []map[string]any{
		{
			"role":    "user",
			"content": contentArray,
		},
	}

	reqBody := map[string]any{
		"model":    imageEditCfg.Model,
		"messages": messages,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	log.Printf("[图像编辑 API] 请求体大小: %d 字节", len(bodyBytes))
	if len(bodyBytes) <= 1000 {
		log.Printf("[图像编辑 API] 请求体: %s", string(bodyBytes))
	} else {
		log.Printf("[图像编辑 API] 请求体（前1000字符）: %s...", string(bodyBytes[:1000]))
	}

	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			waitTime := time.Duration(attempt-1) * 2 * time.Second
			log.Printf("[图像编辑 API] 第 %d 次重试，等待 %v...", attempt, waitTime)
			time.Sleep(waitTime)
		}

		log.Printf("[图像编辑 API] 发起请求 (尝试 %d/%d): %s/v1/chat/completions", attempt, maxRetries, base)

		result, err := doImageEditRequest(ctx, base, imageEditCfg.APIKey, bodyBytes)
		if err == nil {
			if attempt > 1 {
				log.Printf("[图像编辑 API] 重试成功！")
			}
			return result, nil
		}

		lastErr = err
		log.Printf("[图像编辑 API] 尝试 %d/%d 失败: %v", attempt, maxRetries, err)
	}

	return "", fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
}

func doImageRequest(ctx context.Context, baseURL, apiKey string, bodyBytes []byte) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	apiURL := baseURL + "/v1/chat/completions"
	request, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		log.Printf("[图像 API] 错误响应 (状态码 %d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
		return "", fmt.Errorf("图像服务请求失败 (状态码 %d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", errors.New("图像服务未返回内容")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	imageURL, err := extractImageURL(content)
	if err != nil {
		return "", fmt.Errorf("解析图片链接失败: %w", err)
	}

	return imageURL, nil
}

func doImageEditRequest(ctx context.Context, baseURL, apiKey string, bodyBytes []byte) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	apiURL := baseURL + "/v1/chat/completions"
	request, err := http.NewRequestWithContext(reqCtx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		log.Printf("[图像编辑 API] 错误响应 (状态码 %d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
		return "", fmt.Errorf("图像编辑服务请求失败 (状态码 %d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	responseJSON, _ := json.Marshal(response)
	log.Printf("[图像编辑 API] 响应内容: %s", string(responseJSON))

	if choices, ok := response["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if message, ok := choice["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					imageURL, err := extractImageURL(content)
					if err == nil {
						return imageURL, nil
					}
					if strings.HasPrefix(strings.ToLower(content), "http") {
						return content, nil
					}
				}
			}
		}
	}

	if output, ok := response["output"].(map[string]any); ok {
		if results, ok := output["results"].([]any); ok && len(results) > 0 {
			if result, ok := results[0].(map[string]any); ok {
				if url, ok := result["url"].(string); ok {
					return url, nil
				}
			}
		}
	}

	return "", errors.New("图像编辑服务未返回有效的图片URL")
}

func extractImageURL(content string) (string, error) {
	matches := imageURLPattern.FindAllString(content, -1)
	if len(matches) == 0 {
		return "", errors.New("未找到图片链接")
	}
	for _, candidate := range matches {
		clean := strings.Trim(candidate, "[]()<>\"'`.,")
		lower := strings.ToLower(clean)
		if strings.Contains(lower, ".png") || strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".webp") {
			return clean, nil
		}
	}
	clean := strings.Trim(matches[len(matches)-1], "[]()<>\"'`.,")
	return clean, nil
}

func RemoveGeneratedImage(relPath string) {
	utils.RemoveGeneratedFile(relPath, utils.GeneratedImagesURLPrefix, utils.GeneratedImagesDir)
}
