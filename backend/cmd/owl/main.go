package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"owl/internal/api"
	"owl/internal/db"
	"owl/internal/store"
)

func main() {
	dataDir, err := ensureDataDir()
	if err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	database, err := db.Init(filepath.Join(dataDir, "owl.db"))
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	s := store.New(database)
	router := api.NewRouter(s)
	addr := ":3721"
	fmt.Printf("Owl backend listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

func ensureDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dataDir := filepath.Join(configDir, "owl")
	return dataDir, os.MkdirAll(dataDir, 0755)
}
