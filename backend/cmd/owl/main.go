package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"owl/internal/api"
	"owl/internal/config"
	"owl/internal/db"
	"owl/internal/extractor"
	"owl/internal/llm"
	"owl/internal/logging"
	"owl/internal/scanner"
	"owl/internal/store"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logging.Init(logLevel)

	configPath := "../config.json" //os.Getenv("OWL_CONFIG")
	if configPath == "" {
		var err error
		configPath, err = config.DefaultConfigPath()
		if err != nil {
			slog.Warn("could not determine default config path", "error", err)
		} else {
			slog.Info("using default config", "path", configPath)
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Warn("failed to load config file", "error", err)
		cfg = &config.Config{}
	}

	llmCfg := llm.ConfigFromEnv(cfg)
	var llmClient *llm.Client
	if llmCfg.Enabled {
		llmClient = llm.NewClient(llmCfg)
		if llmClient != nil {
			if llmClient.IsAvailable(context.Background()) {
				slog.Info("LLM refinement enabled", "url", llmCfg.BaseURL, "model", llmCfg.Model)
			} else {
				slog.Info("LLM refinement enabled but unavailable at startup", "url", llmCfg.BaseURL)
			}
		}
	} else {
		slog.Info("LLM refinement disabled")
	}

	dataDir, err := ensureDataDir()
	if err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	database, err := db.Init(filepath.Join(dataDir, "owl.db"))
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	s := store.New(database)
	s.RecoverStuckFiles()
	sc := scanner.New(s)
	ext := extractor.New(s)
	router := api.NewRouter(s, sc, ext, llmClient, cfg)
	addr := ":3721"
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func ensureDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(configDir, "owl")
	return dataDir, os.MkdirAll(dataDir, 0755)
}
