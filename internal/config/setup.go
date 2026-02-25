package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	CacheDir string
	NotesDir string
}

func SetupEnvironment() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get home directory: %w", err)
	}

	cachePath := filepath.Join(home, ".cache", "quasar")
	notesPath := filepath.Join(home, "Documents", "quasar")

	dirs := []string{cachePath, notesPath}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("Failed to create directory %s: %w", dir, err)
		}
	}
	return &Config{
		CacheDir: cachePath,
		NotesDir: notesPath,
	}, nil
}
