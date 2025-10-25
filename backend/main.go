package main

import (
	"log"
	"net/http"

	"taco/backend/config"
	"taco/backend/handlers"
	"taco/backend/utils"
)

func main() {
	if _, err := config.LoadConfig(); err != nil {
		log.Fatalf("initialise config: %v", err)
	}

	if err := utils.EnsureDir(utils.UploadDir); err != nil {
		log.Fatalf("ensure upload dir: %v", err)
	}
	if err := utils.EnsureDir(utils.GeneratedImagesDir); err != nil {
		log.Fatalf("ensure generated dir: %v", err)
	}
	if err := utils.EnsureDir(utils.GeneratedAudioDir); err != nil {
		log.Fatalf("ensure audio dir: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(utils.WebDir)))
	mux.Handle("/generated/", http.StripPrefix("/generated/", http.FileServer(http.Dir(utils.GeneratedDir))))
	mux.HandleFunc("/api/config", handlers.ConfigHandler)
	mux.HandleFunc("/api/upload", handlers.UploadHandler)
	mux.HandleFunc("/api/characters", handlers.CharactersHandler)
	mux.HandleFunc("/api/characters/extract", handlers.ExtractCharactersHandler)
	mux.HandleFunc("/api/characters/generate-image", handlers.GenerateCharacterImageHandler)
	mux.HandleFunc("/api/characters/generate-all", handlers.GenerateAllCharacterImagesHandler)
	mux.HandleFunc("/api/scenes", handlers.ScenesHandler)
	mux.HandleFunc("/api/scenes/extract", handlers.ExtractScenesHandler)
	mux.HandleFunc("/api/scenes/generate-image", handlers.GenerateSceneImageHandler)
	mux.HandleFunc("/api/scenes/generate-image-with-characters", handlers.GenerateSceneImageWithCharactersHandler)
	mux.HandleFunc("/api/scenes/generate-audio", handlers.GenerateSceneAudioHandler)
	mux.HandleFunc("/api/scenes/generate-all", handlers.GenerateAllSceneImagesHandler)

	log.Printf("Server listening on %s", utils.ListenAddr)
	if err := http.ListenAndServe(utils.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}
