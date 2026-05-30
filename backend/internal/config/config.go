package config

import (
	"flag"
	"os"
)

type LLMConfig struct {
	Enabled        bool
	BaseURL        string
	Model          string
	EmbedModel     string
	FolderStrategy string
}

type Config struct {
	LLM      LLMConfig
	Port     string
	DataDir  string
	LogLevel string
}

func Init() *Config {
	port := flag.String("port", "3721", "HTTP server port")
	dataDir := flag.String("data-dir", "data", "Data directory for database and logs")
	flag.Parse()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		Port:     *port,
		DataDir:  *dataDir,
		LogLevel: logLevel,
		LLM: LLMConfig{
			Enabled:        true,
			BaseURL:        "http://127.0.0.1:11434",
			Model:          "deepseek-r1:1.5b",
			EmbedModel:     "nomic-embed-text:latest",
			FolderStrategy: "content_tfidf",
		},
	}
}
