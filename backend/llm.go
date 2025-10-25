package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func invokeLLM(ctx context.Context, cfg Config, messages []map[string]string, temperature float64) (string, error) {
	if strings.TrimSpace(cfg.LLM.APIKey) == "" {
		return "", errors.New("请先在配置中填写 LLM API Key")
	}

	base := strings.TrimRight(cfg.LLM.BaseURL, "/")
	if base == "" {
		return "", errors.New("LLM Base URL 未设置")
	}

	reqBody := map[string]any{
		"model":       cfg.LLM.Model,
		"messages":    messages,
		"temperature": temperature,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	apiURL := base + "/v1/chat/completions"
	log.Printf("[LLM API] 发起请求: %s, 模型: %s, 消息数: %d", apiURL, cfg.LLM.Model, len(messages))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+cfg.LLM.APIKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("[LLM API] 请求失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("[LLM API] 收到响应: HTTP %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		log.Printf("[LLM API] 错误响应: %s", strings.TrimSpace(string(errBody)))
		return "", fmt.Errorf("LLM 请求失败: %s", strings.TrimSpace(string(errBody)))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", errors.New("LLM 未返回结果")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	log.Printf("[LLM API] 成功获取响应，内容长度: %d 字节", len(content))
	return content, nil
}

func callLLMForCharacters(ctx context.Context, cfg Config, novel string) ([]CharacterProfile, error) {
	prompt := fmt.Sprintf(`请阅读以下小说内容，从中提取主要人物及其关键特征。请输出 JSON 数组，每个元素包含字段 "name" (人物名) 和 "description" (特征描述)。仅返回有效的 JSON，不要附加额外文字。

小说内容：
%s`, novel)

	content, err := invokeLLM(ctx, cfg, []map[string]string{
		{
			"role":    "system",
			"content": "你是一名擅长从小说中抽取人物信息的助手。",
		},
		{
			"role":    "user",
			"content": prompt,
		},
	}, 0.3)
	if err != nil {
		return nil, err
	}

	var characters []CharacterProfile
	if err := json.Unmarshal([]byte(content), &characters); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if characters == nil {
		characters = []CharacterProfile{}
	}

	if limit := cfg.CharacterCount; limit > 0 && len(characters) > limit {
		characters = characters[:limit]
	}
	return characters, nil
}

func callLLMForScenes(ctx context.Context, cfg Config, novel string, characters []CharacterProfile) ([]Scene, error) {
	charactersJSON, err := json.Marshal(characters)
	if err != nil {
		return nil, fmt.Errorf("序列化角色信息失败: %w", err)
	}

	prompt := fmt.Sprintf(`请基于以下小说内容和现有的角色信息拆分出适合制作动漫的关键场景。请输出 JSON 数组，每个元素为一个场景对象，包含字段:
- "title": 场景名称
- "characters": 出场人物名称数组
- "description": 场景的视觉/剧情描述
- "dialogues": 关键对话数组，每个元素是一句话
- "narration": 旁白或解说词

仅返回可被 JSON 解析的数组，不要添加额外说明。

小说内容：
%s

角色信息 (JSON):
%s`, novel, string(charactersJSON))

	content, err := invokeLLM(ctx, cfg, []map[string]string{
		{
			"role":    "system",
			"content": "你是一名资深的分镜师，擅长把小说拆分成动漫场景。",
		},
		{
			"role":    "user",
			"content": prompt,
		},
	}, 0.2)
	if err != nil {
		return nil, err
	}

	scenes, err := parseScenesJSON(content)
	if err != nil {
		return nil, err
	}

	if limit := cfg.SceneCount; limit > 0 && len(scenes) > limit {
		scenes = scenes[:limit]
	}
	return scenes, nil
}

func parseScenesJSON(content string) ([]Scene, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return []Scene{}, nil
	}

	var scenes []Scene
	err := json.Unmarshal([]byte(content), &scenes)
	if err == nil {
		return normalizeScenes(scenes), nil
	}

	var wrapper struct {
		Scenes []Scene `json:"scenes"`
	}
	if errWrapper := json.Unmarshal([]byte(content), &wrapper); errWrapper == nil {
		return normalizeScenes(wrapper.Scenes), nil
	}

	return nil, fmt.Errorf("解析 LLM 场景响应失败: %w", err)
}

func normalizeScenes(scenes []Scene) []Scene {
	if scenes == nil {
		return []Scene{}
	}

	normalized := make([]Scene, len(scenes))
	for i, scene := range scenes {
		scene.Title = strings.TrimSpace(scene.Title)
		scene.Description = strings.TrimSpace(scene.Description)
		scene.Narration = strings.TrimSpace(scene.Narration)
		scene.ImagePath = strings.TrimSpace(scene.ImagePath)
		scene.AudioPath = strings.TrimSpace(scene.AudioPath)

		if scene.Characters == nil {
			scene.Characters = []string{}
		} else {
			for idx, name := range scene.Characters {
				scene.Characters[idx] = strings.TrimSpace(name)
			}
		}

		if scene.Dialogues == nil {
			scene.Dialogues = []string{}
		} else {
			for idx, line := range scene.Dialogues {
				scene.Dialogues[idx] = strings.TrimSpace(line)
			}
		}

		normalized[i] = scene
	}
	return normalized
}
