package llm

import "owl/internal/config"

type ClientConfig struct {
	Enabled bool
	BaseURL string
	Model   string
}

func ConfigFromEnv(cfg *config.Config) ClientConfig {
	if cfg == nil || !cfg.LLM.Enabled {
		return ClientConfig{
			Enabled: false,
			BaseURL: "http://localhost:11434",
			Model:   "",
		}
	}

	return ClientConfig{
		Enabled: cfg.LLM.Enabled,
		BaseURL: cfg.LLM.BaseURL,
		Model:   cfg.LLM.Model,
	}
}
