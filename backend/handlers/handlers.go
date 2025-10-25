package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"taco/backend/config"
	"taco/backend/models"
	"taco/backend/services/audio"
	"taco/backend/services/image"
	"taco/backend/services/llm"
	"taco/backend/utils"
)

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Printf("[ERROR] 读取配置失败: %v", err)
			http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[SUCCESS] 成功读取配置")
		utils.WriteJSON(w, cfg)

	case http.MethodPost:
		var cfg models.Config
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&cfg); err != nil {
			log.Printf("[ERROR] 请求数据无效: %v", err)
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if err := config.ValidateConfig(cfg); err != nil {
			log.Printf("[ERROR] 配置验证失败: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := config.SaveConfig(cfg); err != nil {
			log.Printf("[ERROR] 保存配置失败: %v", err)
			http.Error(w, fmt.Sprintf("保存配置失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[SUCCESS] 成功保存配置")
		utils.WriteJSON(w, cfg)

	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(utils.MaxFileSize); err != nil {
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

	if err := os.MkdirAll(utils.UploadDir, 0o755); err != nil {
		http.Error(w, "创建上传目录失败", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(header.Filename)
	if filename == "" {
		filename = "novel.txt"
	}
	targetName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
	targetPath := filepath.Join(utils.UploadDir, targetName)

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
	utils.WriteJSON(w, map[string]string{
		"filePath": targetPath,
	})
}

func CharactersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		characters, err := config.LoadCharactersData()
		if err != nil {
			http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, characters)
	case http.MethodPost:
		var characters []models.CharacterProfile
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&characters); err != nil {
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if characters == nil {
			characters = []models.CharacterProfile{}
		}
		if err := config.SaveCharactersData(characters); err != nil {
			http.Error(w, fmt.Sprintf("保存角色失败: %v", err), http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, characters)
	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func ScenesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	switch r.Method {
	case http.MethodGet:
		scenes, err := config.LoadScenesData()
		if err != nil {
			http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, scenes)
	case http.MethodPost:
		var scenes []models.Scene
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&scenes); err != nil {
			http.Error(w, "请求数据无效", http.StatusBadRequest)
			return
		}
		if scenes == nil {
			scenes = []models.Scene{}
		}
		if err := config.SaveScenesData(scenes); err != nil {
			http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, scenes)
	default:
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
	}
}

func ExtractCharactersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := config.LoadConfig()
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

	characters, err := llm.CallLLMForCharacters(ctx, cfg, string(novelData))
	if err != nil {
		log.Printf("[ERROR] 分析角色失败: %v", err)
		http.Error(w, fmt.Sprintf("分析角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功提取 %d 个角色", len(characters))
	if err := config.SaveCharactersData(characters); err != nil {
		log.Printf("[ERROR] 保存角色信息失败: %v", err)
		http.Error(w, fmt.Sprintf("保存角色信息失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, characters)
}

func ExtractScenesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := config.LoadConfig()
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

	characters, err := config.LoadCharactersData()
	if err != nil {
		log.Printf("[ERROR] 读取角色失败: %v", err)
		http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] 开始提取场景，小说文件大小: %d 字节，角色数: %d", len(novelData), len(characters))

	ctx, cancel := context.WithTimeout(r.Context(), 900*time.Second)
	defer cancel()

	scenes, err := llm.CallLLMForScenes(ctx, cfg, string(novelData), characters)
	if err != nil {
		log.Printf("[ERROR] 分析场景失败: %v", err)
		http.Error(w, fmt.Sprintf("分析场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功提取 %d 个场景", len(scenes))
	if err := config.SaveScenesData(scenes); err != nil {
		log.Printf("[ERROR] 保存场景失败: %v", err)
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, scenes)
}

func GenerateCharacterImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "角色索引无效", http.StatusBadRequest)
		return
	}

	characters, err := config.LoadCharactersData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
		return
	}
	if payload.Index >= len(characters) {
		http.Error(w, "角色索引超出范围", http.StatusBadRequest)
		return
	}

	character := characters[payload.Index]
	if strings.TrimSpace(character.Description) == "" {
		http.Error(w, "角色描述为空，无法生成图片", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] 开始生成角色 %d 的图片，角色名称: %s", payload.Index, character.Name)

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	imagePath, err := image.GenerateCharacterImage(ctx, cfg, character, payload.Index)
	if err != nil {
		log.Printf("[ERROR] 生成图片失败: %v", err)
		http.Error(w, fmt.Sprintf("生成图片失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功生成角色图片: %s", imagePath)
	character.ImagePath = imagePath
	characters[payload.Index] = character
	if err := config.SaveCharactersData(characters); err != nil {
		http.Error(w, fmt.Sprintf("保存角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, character)
}

func GenerateSceneImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "场景索引无效", http.StatusBadRequest)
		return
	}

	scenes, err := config.LoadScenesData()
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

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	imagePath, err := image.GenerateSceneImage(ctx, cfg, scene, payload.Index)
	if err != nil {
		log.Printf("[ERROR] 生成图片失败: %v", err)
		http.Error(w, fmt.Sprintf("生成图片失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功生成场景图片: %s", imagePath)
	scene.ImagePath = imagePath
	scenes[payload.Index] = scene
	if err := config.SaveScenesData(scenes); err != nil {
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, scene)
}

func GenerateSceneImageWithCharactersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "场景索引无效", http.StatusBadRequest)
		return
	}

	scenes, err := config.LoadScenesData()
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

	log.Printf("[INFO] 开始使用人物图片生成场景 %d 的图片，场景标题: %s", payload.Index, scene.Title)

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	characters, err := config.LoadCharactersData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	imagePath, err := image.GenerateSceneImageWithCharacters(ctx, cfg, scene, characters, payload.Index)
	if err != nil {
		log.Printf("[ERROR] 生成图片失败: %v", err)
		http.Error(w, fmt.Sprintf("生成图片失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[SUCCESS] 成功生成场景图片: %s", imagePath)
	scene.ImagePath = imagePath
	scenes[payload.Index] = scene
	if err := config.SaveScenesData(scenes); err != nil {
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, scene)
}

func GenerateSceneAudioHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Index int `json:"index"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		http.Error(w, "请求数据无效", http.StatusBadRequest)
		return
	}
	if payload.Index < 0 {
		http.Error(w, "场景索引无效", http.StatusBadRequest)
		return
	}

	scenes, err := config.LoadScenesData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
		return
	}
	if payload.Index >= len(scenes) {
		http.Error(w, "场景索引超出范围", http.StatusBadRequest)
		return
	}

	scene := scenes[payload.Index]
	if text := audio.BuildSceneSpeechText(scene); text == "" {
		http.Error(w, "场景缺少可用于生成语音的文本", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(cfg.Voice.APIKey) == "" {
		http.Error(w, "请先在配置中填写语音 API Key", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
	defer cancel()

	audioPath, err := audio.GenerateSceneAudio(ctx, cfg, scene, payload.Index)
	if err != nil {
		log.Printf("[ERROR] 生成语音失败: %v", err)
		http.Error(w, fmt.Sprintf("生成语音失败: %v", err), http.StatusInternalServerError)
		return
	}

	scene.AudioPath = audioPath
	scenes[payload.Index] = scene
	if err := config.SaveScenesData(scenes); err != nil {
		http.Error(w, fmt.Sprintf("保存场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	utils.WriteJSON(w, scene)
}

func GenerateAllCharacterImagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	characters, err := config.LoadCharactersData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取角色失败: %v", err), http.StatusInternalServerError)
		return
	}

	if len(characters) == 0 {
		http.Error(w, "暂无角色信息", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	sendProgress := func(current, total int, status string, success bool) {
		data := map[string]interface{}{
			"current": current,
			"total":   total,
			"status":  status,
			"success": success,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	total := len(characters)
	for i := range characters {
		if strings.TrimSpace(characters[i].Description) == "" {
			sendProgress(i+1, total, fmt.Sprintf("角色 %d (%s) 描述为空，跳过", i+1, characters[i].Name), false)
			continue
		}

		sendProgress(i+1, total, fmt.Sprintf("正在生成角色 %d/%d: %s", i+1, total, characters[i].Name), false)

		ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
		imagePath, err := image.GenerateCharacterImage(ctx, cfg, characters[i], i)
		cancel()

		if err != nil {
			log.Printf("[ERROR] 生成角色 %d 图片失败: %v", i, err)
			sendProgress(i+1, total, fmt.Sprintf("角色 %d (%s) 生成失败: %v", i+1, characters[i].Name, err), false)
			continue
		}

		characters[i].ImagePath = imagePath
		if err := config.SaveCharactersData(characters); err != nil {
			log.Printf("[ERROR] 保存角色 %d 失败: %v", i, err)
		}

		sendProgress(i+1, total, fmt.Sprintf("角色 %d/%d (%s) 生成成功", i+1, total, characters[i].Name), true)
	}

	sendProgress(total, total, "所有角色图片生成完成", true)
}

func GenerateAllSceneImagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s - 来自 %s", r.Method, r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodPost {
		http.Error(w, "不支持的请求方法", http.StatusMethodNotAllowed)
		return
	}

	scenes, err := config.LoadScenesData()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取场景失败: %v", err), http.StatusInternalServerError)
		return
	}

	if len(scenes) == 0 {
		http.Error(w, "暂无场景信息", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("读取配置失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "不支持流式响应", http.StatusInternalServerError)
		return
	}

	sendProgress := func(current, total int, status string, success bool) {
		data := map[string]interface{}{
			"current": current,
			"total":   total,
			"status":  status,
			"success": success,
		}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	total := len(scenes)
	for i := range scenes {
		if strings.TrimSpace(scenes[i].Description) == "" {
			sendProgress(i+1, total, fmt.Sprintf("场景 %d (%s) 描述为空，跳过", i+1, scenes[i].Title), false)
			continue
		}

		sendProgress(i+1, total, fmt.Sprintf("正在生成场景 %d/%d: %s", i+1, total, scenes[i].Title), false)

		ctx, cancel := context.WithTimeout(r.Context(), 300*time.Second)
		imagePath, err := image.GenerateSceneImage(ctx, cfg, scenes[i], i)
		cancel()

		if err != nil {
			log.Printf("[ERROR] 生成场景 %d 图片失败: %v", i, err)
			sendProgress(i+1, total, fmt.Sprintf("场景 %d (%s) 生成失败: %v", i+1, scenes[i].Title, err), false)
			continue
		}

		scenes[i].ImagePath = imagePath
		if err := config.SaveScenesData(scenes); err != nil {
			log.Printf("[ERROR] 保存场景 %d 失败: %v", i, err)
		}

		sendProgress(i+1, total, fmt.Sprintf("场景 %d/%d (%s) 生成成功", i+1, total, scenes[i].Title), true)
	}

	sendProgress(total, total, "所有场景图片生成完成", true)
}
