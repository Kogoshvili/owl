package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"owl/internal/extractor"
	"owl/internal/scanner"
	"owl/internal/store"
)

type WatchedDirHandler struct {
	store          *store.Store
	scanner        *scanner.Scanner
	extractor      *extractor.Extractor
	scanTracker    opTracker
	extractTracker opTracker
}

func NewWatchedDirHandler(s *store.Store, sc *scanner.Scanner, ext *extractor.Extractor) *WatchedDirHandler {
	return &WatchedDirHandler{store: s, scanner: sc, extractor: ext}
}

type createWatchedDirRequest struct {
	Path string `json:"path"`
}

type updateWatchedDirRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *WatchedDirHandler) List(w http.ResponseWriter, r *http.Request) {
	dirs, err := h.store.ListWatchedDirs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dirs == nil {
		dirs = []store.WatchedDir{}
	}
	writeJSON(w, http.StatusOK, dirs)
}

func (h *WatchedDirHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createWatchedDirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	id, err := h.store.CreateWatchedDir(req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go scan(h, req.Path, id)

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *WatchedDirHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req updateWatchedDirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := h.store.UpdateWatchedDir(id, req.Enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "watched directory not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (h *WatchedDirHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteWatchedDirAndFiles(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WatchedDirHandler) Scan(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	dir, err := h.store.GetWatchedDir(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dir == nil {
		writeError(w, http.StatusNotFound, "watched directory not found")
		return
	}

	go scan(h, dir.Path, dir.ID)

	writeJSON(w, http.StatusAccepted, dir)
}

func (h *WatchedDirHandler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scanTracker.get()
	if status == nil {
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *WatchedDirHandler) Extract(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	dir, err := h.store.GetWatchedDir(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dir == nil {
		writeError(w, http.StatusNotFound, "watched directory not found")
		return
	}

	queued, err := h.store.QueueFilesForExtraction(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if queued > 0 {
		go runExtraction(h.extractor, &h.extractTracker)
	}

	writeJSON(w, http.StatusAccepted, map[string]int64{"queued": queued})
}

func scan(h *WatchedDirHandler, path string, id int64) {
	slog.Info("starting background scan", "dir", path, "dir_id", id)
	h.scanTracker.clear()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("scan panicked", "dir", path, "panic", r)
			h.scanTracker.error(fmt.Sprintf("Scan panicked: %v", r))
		}
	}()
	h.scanTracker.update("scanning", "Scanning directory", 0, 0)
	if err := h.scanner.Scan(context.Background(), path, id); err != nil {
		h.scanTracker.error("Scan failed: " + err.Error())
		return
	}
	h.scanTracker.complete("Scan complete")
}
