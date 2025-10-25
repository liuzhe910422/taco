package utils

import "path/filepath"

const (
	ListenAddr               = ":8080"
	MaxFileSize              = 32 << 20
	GeneratedImagesURLPrefix = "/generated/images/"
	GeneratedAudioURLPrefix  = "/generated/audio/"
)

var (
	ProjectRoot        = MustFindProjectRoot()
	ConfigPath         = filepath.Join(ProjectRoot, "config", "config.json")
	CharactersPath     = filepath.Join(ProjectRoot, "config", "characters.json")
	ScenesPath         = filepath.Join(ProjectRoot, "config", "scenes.json")
	UploadDir          = filepath.Join(ProjectRoot, "uploads")
	GeneratedDir       = filepath.Join(ProjectRoot, "generated")
	GeneratedImagesDir = filepath.Join(GeneratedDir, "images")
	GeneratedAudioDir  = filepath.Join(GeneratedDir, "audio")
	WebDir             = filepath.Join(ProjectRoot, "web")
)
