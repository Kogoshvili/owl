package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"owl/internal/extractor"
	"owl/internal/scanner"
	"owl/internal/store"
)

type WatchedDirHandler struct {
	store     *store.Store
	scanner   *scanner.Scanner
	extractor *extractor.Extractor
}

func NewWatchedDirHandler(s *store.Store, sc *scanner.Scanner, ext *extractor.Extractor) *WatchedDirHandler {
	return &WatchedDirHandler{store: s, scanner: sc, extractor: ext}
}

type createWatchedDirRequest struct {
	Path      string `json:"path"`
	Recursive *bool  `json:"recursive"`
}

type updateWatchedDirRequest struct {
	Enabled   *bool `json:"enabled"`
	Recursive *bool `json:"recursive"`
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
	recursive := true
	if req.Recursive != nil {
		recursive = *req.Recursive
	}

	dir, err := h.store.CreateWatchedDir(req.Path, recursive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("starting background scan", "dir", dir.Path, "dir_id", dir.ID)
	go h.scanner.Scan(context.Background(), dir.Path, dir.Recursive, dir.ID)

	writeJSON(w, http.StatusCreated, dir)
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

	dir, err := h.store.UpdateWatchedDir(id, req.Enabled, req.Recursive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if dir == nil {
		writeError(w, http.StatusNotFound, "watched directory not found")
		return
	}
	writeJSON(w, http.StatusOK, dir)
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

	slog.Info("starting background scan", "dir", dir.Path, "dir_id", dir.ID)
	go h.scanner.Scan(context.Background(), dir.Path, dir.Recursive, dir.ID)

	writeJSON(w, http.StatusAccepted, dir)
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
		slog.Info("starting background extraction", "dir_id", id, "queued", queued)
		go h.extractor.ProcessAll(context.Background())
	}

	writeJSON(w, http.StatusAccepted, map[string]int64{"queued": queued})
}
