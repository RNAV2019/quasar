package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// NotebookConfig holds persistent notebook preferences.
type NotebookConfig struct {
	DefaultNotebook string `yaml:"default_notebook"`
}

// NotebooksConfigPath returns the path to the notebooks YAML config file.
func (c *Config) NotebooksConfigPath() string {
	return filepath.Join(c.CacheDir, "notebooks.yaml")
}

// LoadNotebookConfig reads and parses the notebooks config file.
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

// SaveNotebookConfig writes the notebooks config to disk.
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

// SetDefaultNotebook persists name as the default notebook.
func (c *Config) SetDefaultNotebook(name string) error {
	cfg, err := c.LoadNotebookConfig()
	if err != nil {
		return err
	}
	cfg.DefaultNotebook = name
	return c.SaveNotebookConfig(cfg)
}

// GetDefaultNotebook returns the name of the default notebook.
func (c *Config) GetDefaultNotebook() (string, error) {
	cfg, err := c.LoadNotebookConfig()
	if err != nil {
		return "", err
	}
	return cfg.DefaultNotebook, nil
}

