package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"owl/internal/api"
	"owl/internal/db"
	"owl/internal/extractor"
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
	router := api.NewRouter(s, sc, ext)
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
