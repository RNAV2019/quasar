package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type NotebookConfig struct {
	DefaultNotebook string `yaml:"default_notebook"`
}

func (c *Config) NotebooksConfigPath() string {
	return filepath.Join(c.CacheDir, "notebooks.yaml")
}

func (c *Config) LoadNotebookConfig() (*NotebookConfig, error) {
	path := c.NotebooksConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &NotebookConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read notebook config: %w", err)
	}

	var cfg NotebookConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse notebook config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) SaveNotebookConfig(cfg *NotebookConfig) error {
	path := c.NotebooksConfigPath()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal notebook config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write notebook config: %w", err)
	}

	return nil
}

func (c *Config) SetDefaultNotebook(name string) error {
	cfg, err := c.LoadNotebookConfig()
	if err != nil {
		return err
	}
	cfg.DefaultNotebook = name
	return c.SaveNotebookConfig(cfg)
}

func (c *Config) GetDefaultNotebook() (string, error) {
	cfg, err := c.LoadNotebookConfig()
	if err != nil {
		return "", err
	}
	return cfg.DefaultNotebook, nil
}

func (c *Config) NotebookPath(name string) string {
	return filepath.Join(c.NotesDir, name)
}

func (c *Config) NotebookExists(name string) bool {
	info, err := os.Stat(c.NotebookPath(name))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (c *Config) ListNotebooks() ([]string, error) {
	entries, err := os.ReadDir(c.NotesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	var notebooks []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			notebooks = append(notebooks, entry.Name())
		}
	}

	return notebooks, nil
}

func (c *Config) HasNotebooks() (bool, error) {
	notebooks, err := c.ListNotebooks()
	if err != nil {
		return false, err
	}
	return len(notebooks) > 0, nil
}
