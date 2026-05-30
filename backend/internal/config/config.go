package config

type LLMConfig struct {
	Enabled        bool
	BaseURL        string
	Model          string
	EmbedModel     string
	FolderStrategy string
}

type Config struct {
	LLM LLMConfig
}

func Default() *Config {
	return &Config{
		LLM: LLMConfig{
			Enabled:        true,
			BaseURL:        "http://127.0.0.1:11434",
			Model:          "deepseek-r1:1.5b",
			EmbedModel:     "nomic-embed-text:latest",
			FolderStrategy: "content_tfidf",
		},
	}
}
