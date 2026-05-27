package handler

import (
	"encoding/json"
	"net/http"

	"owl/internal/store"
)

type WatchedDirHandler struct {
	store *store.Store
}

func NewWatchedDirHandler(s *store.Store) *WatchedDirHandler {
	return &WatchedDirHandler{store: s}
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

	if err := h.store.DeleteWatchedDir(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
