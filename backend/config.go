package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func loadConfig() (Config, error) {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaultCfg := Config{
				NovelFile: "",
				LLM: LLMConfig{
					Model:   "gpt-4o-mini",
					BaseURL: "https://apiqik.apifox.cn/7311242m0",
				},
				Image: ImageConfig{
					Model:   "gpt-4o-image",
					BaseURL: "https://apiqik.apifox.cn/7311242m0",
					Size:    "1024x1024",
					Quality: "standard",
				},
				Voice: VoiceConfig{
					Model:     "qwen3-tts-flash",
					BaseURL:   "https://dashscope.aliyuncs.com",
					APIKey:    "",
					Voice:     "Cherry",
					Language:  "Chinese",
					OutputDir: "generated/audio",
				},
				VideoModel:     "pika-labs",
				CharacterCount: 2,
				SceneCount:     5,
			}
			if err := saveConfig(defaultCfg); err != nil {
				return Config{}, err
			}
			if err := saveCharactersData([]CharacterProfile{}); err != nil {
				return Config{}, err
			}
			if err := saveScenesData([]Scene{}); err != nil {
				return Config{}, err
			}
			return defaultCfg, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	var legacy struct {
		LLMModel     string             `json:"llmModel"`
		LLMBaseURL   string             `json:"llmBaseUrl"`
		LLMAPIKey    string             `json:"llmApiKey"`
		ImageModel   string             `json:"imageModel"`
		ImageBaseURL string             `json:"imageBaseUrl"`
		ImageAPIKey  string             `json:"imageApiKey"`
		ImageSize    string             `json:"imageSize"`
		ImageQuality string             `json:"imageQuality"`
		Characters   []CharacterProfile `json:"characters"`
		Scenes       []Scene            `json:"scenes"`
	}
	if err := json.Unmarshal(data, &legacy); err == nil {
		migrated := false
		if cfg.LLM.Model == "" && legacy.LLMModel != "" {
			cfg.LLM.Model = legacy.LLMModel
		}
		if cfg.LLM.BaseURL == "" && legacy.LLMBaseURL != "" {
			cfg.LLM.BaseURL = legacy.LLMBaseURL
		}
		if cfg.LLM.APIKey == "" && legacy.LLMAPIKey != "" {
			cfg.LLM.APIKey = legacy.LLMAPIKey
		}
		if cfg.Image.Model == "" {
			if legacy.ImageModel != "" {
				cfg.Image.Model = legacy.ImageModel
			} else {
				cfg.Image.Model = "gpt-4o-image"
			}
		}
		if cfg.Image.BaseURL == "" {
			if legacy.ImageBaseURL != "" {
				cfg.Image.BaseURL = legacy.ImageBaseURL
			} else if cfg.LLM.BaseURL != "" {
				cfg.Image.BaseURL = cfg.LLM.BaseURL
			}
		}
		if cfg.Image.APIKey == "" && legacy.ImageAPIKey != "" {
			cfg.Image.APIKey = legacy.ImageAPIKey
		}
		if cfg.Image.Size == "" && legacy.ImageSize != "" {
			cfg.Image.Size = legacy.ImageSize
		}
		if cfg.Image.Quality == "" && legacy.ImageQuality != "" {
			cfg.Image.Quality = legacy.ImageQuality
		}
		if characters := legacy.Characters; len(characters) > 0 {
			if err := saveCharactersData(characters); err != nil {
				log.Printf("迁移旧角色数据失败: %v", err)
			} else {
				migrated = true
			}
		}
		if scenes := legacy.Scenes; len(scenes) > 0 {
			if err := saveScenesData(scenes); err != nil {
				log.Printf("迁移旧场景数据失败: %v", err)
			} else {
				migrated = true
			}
		}
		if migrated {
			if err := saveConfig(cfg); err != nil {
				log.Printf("更新配置文件失败: %v", err)
			}
		}
	}

	if strings.TrimSpace(cfg.Image.Model) == "" {
		cfg.Image.Model = "gpt-4o-image"
	}
	if strings.TrimSpace(cfg.Image.BaseURL) == "" {
		cfg.Image.BaseURL = cfg.LLM.BaseURL
	}
	if strings.TrimSpace(cfg.Image.APIKey) == "" {
		cfg.Image.APIKey = cfg.LLM.APIKey
	}
	if strings.TrimSpace(cfg.Image.Size) == "" {
		cfg.Image.Size = "1024x1024"
	}
	if strings.TrimSpace(cfg.Image.Quality) == "" {
		cfg.Image.Quality = "standard"
	}
	if strings.TrimSpace(cfg.Voice.Model) == "" {
		cfg.Voice.Model = "qwen3-tts-flash"
	}
	if strings.TrimSpace(cfg.Voice.BaseURL) == "" {
		cfg.Voice.BaseURL = "https://dashscope.aliyuncs.com"
	}
	if strings.TrimSpace(cfg.Voice.OutputDir) == "" {
		cfg.Voice.OutputDir = "generated/audio"
	}
	if strings.TrimSpace(cfg.Voice.Voice) == "" {
		cfg.Voice.Voice = "Cherry"
	}
	if strings.TrimSpace(cfg.Voice.Language) == "" {
		cfg.Voice.Language = "Chinese"
	}
	if cfg.Voice.APIKey == "" {
		cfg.Voice.APIKey = cfg.LLM.APIKey
	}

	return cfg, nil
}

func saveConfig(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	tmpPath := configPath + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return err
	}

	return os.Rename(tmpPath, configPath)
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.LLM.Model) == "" {
		return errors.New("LLM 模型不能为空")
	}
	if strings.TrimSpace(cfg.LLM.BaseURL) == "" {
		return errors.New("LLM 接口地址不能为空")
	}
	if strings.TrimSpace(cfg.Image.Model) == "" {
		return errors.New("图像模型不能为空")
	}
	if strings.TrimSpace(cfg.Image.BaseURL) == "" && strings.TrimSpace(cfg.LLM.BaseURL) == "" {
		return errors.New("图像接口地址不能为空")
	}
	if strings.TrimSpace(cfg.Image.APIKey) == "" && strings.TrimSpace(cfg.LLM.APIKey) == "" {
		return errors.New("请填写 LLM 或图像 API Key")
	}
	if strings.TrimSpace(cfg.Voice.Model) == "" {
		return errors.New("语音模型不能为空")
	}
	if strings.TrimSpace(cfg.Voice.BaseURL) == "" {
		return errors.New("语音接口地址不能为空")
	}
	if strings.TrimSpace(cfg.Voice.APIKey) == "" {
		return errors.New("语音 API Key 不能为空")
	}
	if strings.TrimSpace(cfg.Voice.Voice) == "" {
		return errors.New("发音人不能为空")
	}
	if strings.TrimSpace(cfg.Voice.Language) == "" {
		return errors.New("语言类型不能为空")
	}
	if cfg.CharacterCount < 0 {
		return errors.New("人物数必须是非负整数")
	}
	if cfg.SceneCount < 0 {
		return errors.New("场景数必须是非负整数")
	}
	return nil
}

func loadCharactersData() ([]CharacterProfile, error) {
	if err := os.MkdirAll(filepath.Dir(charactersPath), 0o755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(charactersPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []CharacterProfile{}, nil
		}
		return nil, err
	}
	var characters []CharacterProfile
	if err := json.Unmarshal(data, &characters); err != nil {
		return nil, err
	}
	if characters == nil {
		return []CharacterProfile{}, nil
	}
	return characters, nil
}

func saveCharactersData(characters []CharacterProfile) error {
	if characters == nil {
		characters = []CharacterProfile{}
	}
	if err := os.MkdirAll(filepath.Dir(charactersPath), 0o755); err != nil {
		return err
	}
	tmpPath := charactersPath + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(characters); err != nil {
		return err
	}
	return os.Rename(tmpPath, charactersPath)
}

func loadScenesData() ([]Scene, error) {
	if err := os.MkdirAll(filepath.Dir(scenesPath), 0o755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(scenesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Scene{}, nil
		}
		return nil, err
	}
	var scenes []Scene
	if err := json.Unmarshal(data, &scenes); err != nil {
		return nil, err
	}
	if scenes == nil {
		return []Scene{}, nil
	}
	return normalizeScenes(scenes), nil
}

func saveScenesData(scenes []Scene) error {
	if scenes == nil {
		scenes = []Scene{}
	}
	scenes = normalizeScenes(scenes)
	if err := os.MkdirAll(filepath.Dir(scenesPath), 0o755); err != nil {
		return err
	}
	tmpPath := scenesPath + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(scenes); err != nil {
		return err
	}
	return os.Rename(tmpPath, scenesPath)
}
