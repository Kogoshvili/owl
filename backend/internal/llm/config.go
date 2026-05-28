package llm

import (
	"log/slog"
	"owl/internal/config"
)

type ClientConfig struct {
	Enabled bool
	BaseURL string
	Model   string
}

func ConfigFromEnv(cfg *config.Config) ClientConfig {
	if cfg == nil || !cfg.LLM.Enabled {
		return ClientConfig{
			Enabled: false,
			BaseURL: "http://localhost:1234/v1",
			Model:   "",
		}
	}

	slog.Info("loading LLM config", "enabled", cfg.LLM.Enabled, "base_url", cfg.LLM.BaseURL, "model", cfg.LLM.Model)
	return ClientConfig{
		Enabled: cfg.LLM.Enabled,
		BaseURL: cfg.LLM.BaseURL,
		Model:   cfg.LLM.Model,
	}
}
