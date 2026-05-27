package handler

import (
	"net/http"
	"strconv"

	"owl/internal/store"
)

type FileHandler struct {
	store *store.Store
}

func NewFileHandler(s *store.Store) *FileHandler {
	return &FileHandler{store: s}
}

func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	f := store.FileFilter{Limit: 50}

	if v := r.URL.Query().Get("extension"); v != "" {
		f.Extension = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		f.Status = &v
	}
	if v := r.URL.Query().Get("parent_dir"); v != "" {
		f.ParentDir = &v
	}
	if v := r.URL.Query().Get("tag_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.TagID = &id
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Offset = n
		}
	}

	files, err := h.store.ListFiles(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []store.File{}
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *FileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	file, err := h.store.GetFile(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if file == nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	comment, _ := h.store.GetComment(id)
	tags, _ := h.store.ListFileTags(id)

	result := map[string]any{
		"file":    file,
		"comment": comment,
		"tags":    tags,
	}
	if tags == nil {
		result["tags"] = []store.Tag{}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) ListByDir(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	files, err := h.store.ListFilesByDir(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []store.File{}
	}
	writeJSON(w, http.StatusOK, files)
}
