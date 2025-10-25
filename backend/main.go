package main

import (
	"log"
	"net/http"
	"path/filepath"
)

const (
	listenAddr               = ":8080"
	maxFileSize              = 32 << 20
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
	mux.HandleFunc("/api/characters/generate-image", generateCharacterImageHandler)
	mux.HandleFunc("/api/scenes", scenesHandler)
	mux.HandleFunc("/api/scenes/extract", extractScenesHandler)
	mux.HandleFunc("/api/scenes/generate-image", generateSceneImageHandler)
	mux.HandleFunc("/api/scenes/generate-image-with-characters", generateSceneImageWithCharactersHandler)
	mux.HandleFunc("/api/scenes/generate-audio", generateSceneAudioHandler)

	log.Printf("Server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
