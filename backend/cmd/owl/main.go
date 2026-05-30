package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"owl/internal/api"
	"owl/internal/config"
	"owl/internal/db"
	"owl/internal/extractor"
	"owl/internal/logging"
	"owl/internal/ollama"
	"owl/internal/scanner"
	"owl/internal/store"
)

func main() {
	cfg := config.Init()

	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	logging.Init(cfg)
	ollamaMgr := ollama.Init(cfg)

	database, err := db.Init(filepath.Join(cfg.DataDir, "owl.db"))
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	s := store.New(database)
	s.RecoverStuckFiles()
	sc := scanner.New(s)
	ext := extractor.New(s)
	router := api.NewRouter(s, sc, ext, cfg, ollamaMgr)
	addr := "127.0.0.1:" + cfg.Port
	slog.Info("starting server", "addr", addr, "data_dir", cfg.DataDir)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(addr, router); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down")
	database.Close()
	logging.Close()
	ollamaMgr.Shutdown()
}
