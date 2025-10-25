package main

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
	"sync"
	"time"
)

const (
	listenAddr               = ":8080"
	maxFileSize              = 32 << 20 // 32 MB
	generatedImagesURLPrefix = "/generated/images/"
	generatedAudioURLPrefix  = "/generated/audio/"
)

var (
	projectRoot        = mustFindProjectRoot()
	configPath         = filepath.Join(projectRoot, "config", "config.json")
	charactersPath     = filepath.Join(projectRoot, "config", "characters.json")
	scenesPath         = filepath.Join(projectRoot, "config", "scenes.json")
	uploadDir          = filepath.Join(projectRoot, "uploads")
	generatedDir       = filepath.Join(projectRoot, "generated")
	generatedImagesDir = filepath.Join(generatedDir, "images")
	generatedAudioDir  = filepath.Join(generatedDir, "audio")
	webDir             = filepath.Join(projectRoot, "web")
)

// Config represents the user configurable options for the generator.
type Config struct {
	NovelFile      string      `json:"novelFile"`
	LLM            LLMConfig   `json:"llm"`
	Image          ImageConfig `json:"image"`
	Voice          VoiceConfig `json:"voice"`
	VideoModel     string      `json:"videoModel"`
	CharacterCount int         `json:"characterCount"`
	SceneCount     int         `json:"sceneCount"`
}

type LLMConfig struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
}

type ImageConfig struct {
	Model   string `json:"model"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
}

type VoiceConfig struct {
	Model     string `json:"model"`
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey"`
	Voice     string `json:"voice"`
	Language  string `json:"language"`
	OutputDir string `json:"outputDir"`
}

type CharacterProfile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Scene struct {
	Title       string   `json:"title"`
	Characters  []string `json:"characters"`
	Description string   `json:"description"`
	Dialogues   []string `json:"dialogues"`
	Narration   string   `json:"narration"`
	ImagePath   string   `json:"imagePath"`
	AudioPath   string   `json:"audioPath"`
}

type audioResult struct {
	Source    string
	IsURL     bool
	Extension string
}

type ProgressUpdate struct {
	TaskID    string `json:"taskId"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Progress  int    `json:"progress"`
	Completed bool   `json:"completed"`
	Error     string `json:"error,omitempty"`
}

type progressTracker struct {
	mu       sync.RWMutex
	channels map[string][]chan ProgressUpdate
}

var tracker = &progressTracker{
	channels: make(map[string][]chan ProgressUpdate),
}

func (pt *progressTracker) subscribe(taskID string) chan ProgressUpdate {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	ch := make(chan ProgressUpdate, 10)
	pt.channels[taskID] = append(pt.channels[taskID], ch)
	return ch
}

func (pt *progressTracker) unsubscribe(taskID string, ch chan ProgressUpdate) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	channels := pt.channels[taskID]
	for i, c := range channels {
		if c == ch {
			pt.channels[taskID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(pt.channels[taskID]) == 0 {
		delete(pt.channels, taskID)
	}
}

func (pt *progressTracker) publish(update ProgressUpdate) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	for _, ch := range pt.channels[update.TaskID] {
		select {
		case ch <- update:
		default:
		}
	}
}

func main() {
	if _, err := loadConfig(); err != nil {
		log.Fatalf("initialise config: %v", err)
	}

	if err := ensureDir(uploadDir); err != nil {
		log.Fatalf("ensure upload dir: %v", err)
	}
	if err := ensureDir(generatedImagesDir); err != nil {
		log.Fatalf("ensure generated dir: %v", err)
	}
	if err := ensureDir(generatedAudioDir); err != nil {
		log.Fatalf("ensure audio dir: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	mux.Handle("/generated/", http.StripPrefix("/generated/", http.FileServer(http.Dir(generatedDir))))
	mux.HandleFunc("/api/config", configHandler)
	mux.HandleFunc("/api/upload", uploadHandler)
	mux.HandleFunc("/api/characters", charactersHandler)
	mux.HandleFunc("/api/characters/extract", extractCharactersHandler)
	mux.HandleFunc("/api/scenes", scenesHandler)
	mux.HandleFunc("/api/scenes/extract", extractScenesHandler)
	mux.HandleFunc("/api/scenes/generate-image", generateSceneImageHandler)
	mux.HandleFunc("/api/scenes/generate-audio", generateSceneAudioHandler)
	mux.HandleFunc("/api/progress/", progressHandler)

	log.Printf("Server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

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

	// Backward compatibility for legacy flat keys.
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

func progressHandler(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	if taskID == "" {
		http.Error(w, "缺少任务ID", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式传输", http.StatusInternalServerError)
		return
	}

	ch := tracker.subscribe(taskID)
	defer tracker.unsubscribe(taskID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(update)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			if update.Completed || update.Error != "" {
				return
			}
		}
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		cfg, err := loadConfig()
		if err != nil {
			log.Printf("[ERROR] 读取配置失败: %v", err)
			http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[SUCCESS] 成功读取配置")
		writeJSON(w, cfg)

	case http.MethodPost:
		var cfg Config
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&cfg); err != nil {
			log.Printf("[ERROR] 请求数据无效: %v", err)
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if err := validateConfig(cfg); err != nil {
			log.Printf("[ERROR] 配置验证失败: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := saveConfig(cfg); err != nil {
			log.Printf("[ERROR] 保存配置失败: %v", err)
			http.Error(w, fmt.Sprintf("保存配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[SUCCESS] 成功保存配置")
		writeJSON(w, cfg)

	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		log.Printf("[ERROR] 无法解析上传文件: %v", err)
		http.Error(w, "无法解析上传文件", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("novel")
	if err != nil {
		log.Printf("[ERROR] 未选择小说文件: %v", err)
		http.Error(w, "未选择小说文件", http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Printf("[INFO] 接收到文件上传: %s, 大小: %d 字节", header.Filename, header.Size)

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		http.Error(w, "创建上传目录失败", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(header.Filename)
	if filename == "" {
		filename = "novel.txt"
	}
	targetName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
	targetPath := filepath.Join(uploadDir, targetName)

	dst, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, "保存文件失败", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("[ERROR] 写入文件失败: %v", err)
		http.Error(w, "写入文件失败", http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 文件上传成功: %s", targetPath)
	writeJSON(w, map[string]string{
		"filePath": targetPath,
	})
}

func charactersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		characters, err := loadCharactersData()
		if err != nil {
			http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, characters)
	case http.MethodPost:
		var characters []CharacterProfile
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&characters); err != nil {
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if characters == nil {
			characters = []CharacterProfile{}
		}
		if err := saveCharactersData(characters); err != nil {
			http.Error(w, fmt.Sprintf("保存角色失败: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, characters)
	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func scenesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		scenes, err := loadScenesData()
		if err != nil {
			http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, scenes)
	case http.MethodPost:
		var scenes []Scene
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&scenes); err != nil {
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if scenes == nil {
			scenes = []Scene{}
		}
		if err := saveScenesData(scenes); err != nil {
			http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, scenes)
	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func extractCharactersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	if cfg.NovelFile == "" {
		http.Error(w, "尚未上传小说文件", http.StatusBadRequest)
		return
	}

	novelData, err := os.ReadFile(cfg.NovelFile)
	if err != nil {
		log.Printf("[ERROR] 读取小说文件失败: %v", err)
		http.Error(w, fmt.Sprintf("读取小说文件失败: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[INFO] 开始提取角色，小说文件大小: %d 字节", len(novelData))

	ctx, cancel := context.WithTimeout(r.Context(), 600*time.Second)
	defer cancel()

	characters, err := callLLMForCharacters(ctx, cfg, string(novelData))
	if err != nil {
		log.Printf("[ERROR] 分析角色失败: %v", err)
		http.Error(w, fmt.Sprintf("分析角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功提取 %d 个角色", len(characters))
	if err := saveCharactersData(characters); err != nil {
		log.Printf("[ERROR] 保存角色信息失败: %v", err)
		http.Error(w, fmt.Sprintf("保存角色信息失败: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, characters)
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

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json: %v", err)
	}
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

func generateSceneImage(ctx context.Context, cfg Config, scene Scene, index int, taskID string) (string, error) {
	if err := ensureDir(generatedImagesDir); err != nil {
		return "", err
	}

	tracker.publish(ProgressUpdate{
		TaskID:   taskID,
		Stage:    "generating_image",
		Message:  "正在生成图片...",
		Progress: 30,
	})

	imageRef, err := requestSceneImage(ctx, cfg, scene)
	if err != nil {
		return "", err
	}

	tracker.publish(ProgressUpdate{
		TaskID:   taskID,
		Stage:    "downloading_image",
		Message:  "正在保存图片...",
		Progress: 70,
	})

	filename := fmt.Sprintf("scene_%02d_%d.png", index+1, time.Now().Unix())
	absPath := filepath.Join(generatedImagesDir, filename)

	if scene.ImagePath != "" {
		removeGeneratedImage(scene.ImagePath)
	}

	if strings.HasPrefix(strings.ToLower(imageRef), "http://") || strings.HasPrefix(strings.ToLower(imageRef), "https://") {
		if err := downloadToFile(ctx, imageRef, absPath); err != nil {
			return "", err
		}
	} else {
		if err := saveBase64ToFile(imageRef, absPath); err != nil {
			return "", err
		}
	}

	return generatedImagesURLPrefix + filename, nil
}

func requestSceneImage(ctx context.Context, cfg Config, scene Scene) (string, error) {
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
	promptBuilder.WriteString("以高质量动漫风格绘制以下场景，强调电影级光影、鲜明色彩与角色表情。")
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

	// 使用重试机制
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// 指数退避：第2次重试等待2秒，第3次等待4秒
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

func generateSceneAudio(ctx context.Context, cfg Config, scene Scene, index int, taskID string) (string, error) {
	if err := ensureDir(generatedAudioDir); err != nil {
		return "", err
	}

	tracker.publish(ProgressUpdate{
		TaskID:   taskID,
		Stage:    "generating_audio",
		Message:  "正在生成语音...",
		Progress: 30,
	})

	result, err := requestSceneAudio(ctx, cfg, scene)
	if err != nil {
		return "", err
	}

	tracker.publish(ProgressUpdate{
		TaskID:   taskID,
		Stage:    "downloading_audio",
		Message:  "正在保存音频...",
		Progress: 70,
	})

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
	apiURL := base + "/api/v1/services/aigc/multimodal-generation/generation"
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

var imageURLPattern = regexp.MustCompile(`https?://[^\s)]+`)

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

func downloadToFile(ctx context.Context, fileURL, targetPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}

	transport := &http.Transport{DisableKeepAlives: true, ForceAttemptHTTP2: false}
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("下载失败: %s", strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return os.WriteFile(targetPath, data, 0o644)
}

func saveBase64ToFile(encoded, targetPath string) error {
	encoded = strings.TrimSpace(encoded)
	if strings.HasPrefix(encoded, "data:") {
		if idx := strings.Index(encoded, ","); idx != -1 {
			encoded = encoded[idx+1:]
		}
	}
	encoded = strings.NewReplacer("\n", "", "\r", "").Replace(encoded)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("解码 Base64 数据失败: %w", err)
	}
	return os.WriteFile(targetPath, data, 0o644)
}

func parseDashscopeAudio(payload map[string]any) (audioResult, error) {
	if payload == nil {
		return audioResult{}, errors.New("语音服务未返回音频数据")
	}

	if output, ok := payload["output"].(map[string]any); ok {
		// 处理嵌套的 audio 对象 (官方格式: output.audio.{url, data})
		if audioObj, ok := output["audio"].(map[string]any); ok {
			if res, ok := extractAudioFromMap(audioObj); ok {
				return res, nil
			}
		}
		// 尝试直接从 output 提取
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

var audioDataURLPattern = regexp.MustCompile(`^data:audio/([^;]+);base64,`)

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

func removeGeneratedImage(relPath string) {
	removeGeneratedFile(relPath, generatedImagesURLPrefix, generatedImagesDir)
}

func removeGeneratedAudio(relPath string) {
	removeGeneratedFile(relPath, generatedAudioURLPrefix, generatedAudioDir)
}

func removeGeneratedFile(relPath, prefix, dir string) {
	if !strings.HasPrefix(relPath, prefix) {
		return
	}
	filename := strings.TrimPrefix(relPath, prefix)
	if filename == "" {
		return
	}
	absPath := filepath.Join(dir, filename)
	if err := os.Remove(absPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("删除生成文件失败: %v", err)
	}
}

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

func extractScenesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	if cfg.NovelFile == "" {
		http.Error(w, "尚未上传小说文件", http.StatusBadRequest)
		return
	}

	novelData, err := os.ReadFile(cfg.NovelFile)
	if err != nil {
		log.Printf("[ERROR] 读取小说文件失败: %v", err)
		http.Error(w, fmt.Sprintf("读取小说文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	characters, err := loadCharactersData()
	if err != nil {
		log.Printf("[ERROR] 读取角色失败: %v", err)
		http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] 开始提取场景，小说文件大小: %d 字节，角色数: %d", len(novelData), len(characters))

	ctx, cancel := context.WithTimeout(r.Context(), 900*time.Second)
	defer cancel()

	scenes, err := callLLMForScenes(ctx, cfg, string(novelData), characters)
	if err != nil {
		log.Printf("[ERROR] 分析场景失败: %v", err)
		http.Error(w, fmt.Sprintf("分析场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功提取 %d 个场景", len(scenes))
	if err := saveScenesData(scenes); err != nil {
		log.Printf("[ERROR] 保存场景失败: %v", err)
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, scenes)
}

func generateSceneImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index  int    `json:"index"`
		TaskID string `json:"taskId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "场景索引无效", http.StatusBadRequest)
		return
	}
	if payload.TaskID == "" {
		payload.TaskID = fmt.Sprintf("image-%d-%d", payload.Index, time.Now().UnixNano())
	}

	scenes, err := loadScenesData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
		return
	}
	if payload.Index >= len(scenes) {
		http.Error(w, "场景索引超出范围", http.StatusBadRequest)
		return
	}

	scene := scenes[payload.Index]
	if strings.TrimSpace(scene.Description) == "" {
		http.Error(w, "场景描述为空，无法生成图片", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] 开始生成场景 %d 的图片，场景标题: %s", payload.Index, scene.Title)

	tracker.publish(ProgressUpdate{
		TaskID:   payload.TaskID,
		Stage:    "preparing",
		Message:  "准备生成图片...",
		Progress: 10,
	})

	cfg, err := loadConfig()
	if err != nil {
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "读取配置失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	imagePath, err := generateSceneImage(ctx, cfg, scene, payload.Index, payload.TaskID)
	if err != nil {
		log.Printf("[ERROR] 生成图片失败: %v", err)
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "生成图片失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("生成图片失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功生成场景图片: %s", imagePath)
	scene.ImagePath = imagePath
	scenes[payload.Index] = scene
	if err := saveScenesData(scenes); err != nil {
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "保存场景失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	tracker.publish(ProgressUpdate{
		TaskID:    payload.TaskID,
		Stage:     "completed",
		Message:   "图片生成完成",
		Progress:  100,
		Completed: true,
	})

	writeJSON(w, map[string]any{
		"scene":  scene,
		"taskId": payload.TaskID,
	})
}

func generateSceneAudioHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index  int    `json:"index"`
		TaskID string `json:"taskId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "场景索引无效", http.StatusBadRequest)
		return
	}
	if payload.TaskID == "" {
		payload.TaskID = fmt.Sprintf("audio-%d-%d", payload.Index, time.Now().UnixNano())
	}

	scenes, err := loadScenesData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
		return
	}
	if payload.Index >= len(scenes) {
		http.Error(w, "场景索引超出范围", http.StatusBadRequest)
		return
	}

	scene := scenes[payload.Index]
	if text := buildSceneSpeechText(scene); text == "" {
		http.Error(w, "场景缺少可用于生成语音的文本", http.StatusBadRequest)
		return
	}

	tracker.publish(ProgressUpdate{
		TaskID:   payload.TaskID,
		Stage:    "preparing",
		Message:  "准备生成语音...",
		Progress: 10,
	})

	cfg, err := loadConfig()
	if err != nil {
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "读取配置失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(cfg.Voice.APIKey) == "" {
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "未配置语音 API Key",
			Error:     "请先在配置中填写语音 API Key",
			Completed: true,
		})
		http.Error(w, "请先在配置中填写语音 API Key", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	audioPath, err := generateSceneAudio(ctx, cfg, scene, payload.Index, payload.TaskID)
	if err != nil {
		log.Printf("[ERROR] 生成语音失败: %v", err)
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "生成语音失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("生成语音失败: %v", err), http.StatusInternalServerError)
		return
	}

	scene.AudioPath = audioPath
	scenes[payload.Index] = scene
	if err := saveScenesData(scenes); err != nil {
		tracker.publish(ProgressUpdate{
			TaskID:    payload.TaskID,
			Stage:     "error",
			Message:   "保存场景失败",
			Error:     err.Error(),
			Completed: true,
		})
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	tracker.publish(ProgressUpdate{
		TaskID:    payload.TaskID,
		Stage:     "completed",
		Message:   "语音生成完成",
		Progress:  100,
		Completed: true,
	})

	writeJSON(w, map[string]any{
		"scene":  scene,
		"taskId": payload.TaskID,
	})
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

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func mustFindProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作目录失败: %v", err)
	}
	seen := map[string]bool{}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if seen[dir] {
			break
		}
		seen[dir] = true
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	log.Printf("未找到 go.mod，默认使用当前目录: %s", dir)
	return dir
}
