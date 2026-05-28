package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type LLMConfig struct {
	Enabled       bool   `json:"enabled"`
	BaseURL       string `json:"base_url"`
	Model         string `json:"model"`
	EmbedBaseURL  string `json:"embed_base_url"`
	EmbedModel    string `json:"embed_model"`
}

type Config struct {
	LLM LLMConfig `json:"llm"`
}

func Load(configPath string) (*Config, error) {
	cfg := &Config{
		LLM: LLMConfig{
			Enabled:      false,
			BaseURL:      "http://localhost:1234/v1",
			Model:        "",
			EmbedBaseURL: "http://localhost:1234/v1",
			EmbedModel:   "",
		},
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	if os.Getenv("LLM_ENABLED") == "true" {
		cfg.LLM.Enabled = true
	}
	if url := os.Getenv("LLM_BASE_URL"); url != "" {
		cfg.LLM.BaseURL = url
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		cfg.LLM.Model = model
	}
	if url := os.Getenv("EMBED_BASE_URL"); url != "" {
		cfg.LLM.EmbedBaseURL = url
	}
	if model := os.Getenv("EMBED_MODEL"); model != "" {
		cfg.LLM.EmbedModel = model
	}

	return cfg, nil
}

func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "owl", "config.json"), nil
}
