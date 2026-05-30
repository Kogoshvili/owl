package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"owl/internal/extractor"
	"owl/internal/store"
)

type FileHandler struct {
	store     *store.Store
	extractor *extractor.Extractor
}

func NewFileHandler(s *store.Store, ext *extractor.Extractor) *FileHandler {
	return &FileHandler{store: s, extractor: ext}
}

type fileResponse struct {
	store.File
	Processable *bool `json:"processable,omitempty"`
}

func addProcessable(files []store.File) []fileResponse {
	result := make([]fileResponse, len(files))
	for i, f := range files {
		result[i] = fileResponse{File: f}
		if f.ProcessingStatus == "unprocessed" {
			p := extractor.IsSupported(f.Extension)
			result[i].Processable = &p
		}
	}
	return result
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
	if v := r.URL.Query().Get("processing_status"); v != "" {
		f.ProcessingStatus = &v
	}
	if v := r.URL.Query().Get("supported"); v != "" {
		b := v == "true"
		f.Supported = &b
	}
	if v := r.URL.Query().Get("sort"); v != "" {
		f.SortBy = v
	}
	if v := r.URL.Query().Get("order"); v != "" {
		f.SortOrder = v
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

	total, err := h.store.CountFiles(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	files, err := h.store.ListFiles(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []store.File{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"files":  addProcessable(files),
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

func (h *FileHandler) Extensions(w http.ResponseWriter, r *http.Request) {
	extensions, err := h.store.ListExtensions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if extensions == nil {
		extensions = []string{}
	}
	writeJSON(w, http.StatusOK, extensions)
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
	extractedContent, _ := h.store.GetFileContent(id)

	resp := addProcessable([]store.File{*file})

	result := map[string]any{
		"file":              resp[0],
		"comment":           comment,
		"extracted_content": extractedContent,
	}
	if extractedContent == "" {
		result["extracted_content"] = nil
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *FileHandler) Raw(w http.ResponseWriter, r *http.Request) {
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

	if _, err := os.Stat(file.Path); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "file no longer exists on disk")
		return
	}

	http.ServeFile(w, r, file.Path)
}

func (h *FileHandler) ListByDir(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	f := store.FileFilter{Limit: 50}
	f.WatchedDirID = &id

	if v := r.URL.Query().Get("extension"); v != "" {
		f.Extension = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		f.Status = &v
	}
	if v := r.URL.Query().Get("processing_status"); v != "" {
		f.ProcessingStatus = &v
	}
	if v := r.URL.Query().Get("supported"); v != "" {
		b := v == "true"
		f.Supported = &b
	}
	if v := r.URL.Query().Get("sort"); v != "" {
		f.SortBy = v
	}
	if v := r.URL.Query().Get("order"); v != "" {
		f.SortOrder = v
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

	total, err := h.store.CountFiles(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	files, err := h.store.ListFiles(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []store.File{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"files":  addProcessable(files),
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

func (h *FileHandler) Extract(w http.ResponseWriter, r *http.Request) {
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

	if !extractor.IsSupported(file.Extension) {
		writeError(w, http.StatusBadRequest, "unsupported file type")
		return
	}

	if err := h.store.QueueFileForExtraction(id); err != nil {
		writeError(w, http.StatusBadRequest, "file cannot be queued for extraction")
		return
	}

	slog.Info("starting background extraction", "file_id", id)
	go h.extractor.ProcessAll(context.Background(), nil)

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
