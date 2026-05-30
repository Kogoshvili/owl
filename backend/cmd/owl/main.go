package main

import (
	"context"
	"flag"
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
	port := flag.String("port", "3721", "HTTP server port")
	dataDir := flag.String("data-dir", "data", "Data directory for database and logs")
	flag.Parse()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	logging.Init(logLevel, *dataDir)

	cfg := config.Default()

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

	database, err := db.Init(filepath.Join(*dataDir, "owl.db"))
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	defer logging.Close()

	s := store.New(database)
	s.RecoverStuckFiles()
	sc := scanner.New(s)
	ext := extractor.New(s)
	router := api.NewRouter(s, sc, ext, llmClient, cfg)
	addr := "127.0.0.1:" + *port
	slog.Info("starting server", "addr", addr, "data_dir", *dataDir)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
