package main

type Config struct {
	NovelFile      string      `json:"novelFile"`
	LLM            LLMConfig   `json:"llm"`
	Image          ImageConfig `json:"image"`
	ImageEdit      ImageConfig `json:"imageEdit"`
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
	ImagePath   string `json:"imagePath,omitempty"`
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
