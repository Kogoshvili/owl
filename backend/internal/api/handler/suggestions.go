package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"owl/internal/store"
)

type SuggestionHandler struct {
	store *store.Store
}

func NewSuggestionHandler(s *store.Store) *SuggestionHandler {
	return &SuggestionHandler{store: s}
}

type createSuggestionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateSuggestionRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type addFilesToSuggestionRequest struct {
	FileIDs []int64 `json:"file_ids"`
}

func (h *SuggestionHandler) List(w http.ResponseWriter, r *http.Request) {
	suggestions, err := h.store.ListSuggestions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if suggestions == nil {
		suggestions = []store.FolderSuggestion{}
	}
	writeJSON(w, http.StatusOK, suggestions)
}

func (h *SuggestionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	detail, err := h.store.GetSuggestionDetail(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		writeError(w, http.StatusNotFound, "suggestion not found")
		return
	}
	if detail.Files == nil {
		detail.Files = []store.File{}
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *SuggestionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	suggestion, err := h.store.CreateSuggestion(req.Name, req.Description, "new_folder", 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, suggestion)
}

func (h *SuggestionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req updateSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	suggestion, err := h.store.UpdateSuggestion(id, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if suggestion == nil {
		writeError(w, http.StatusNotFound, "suggestion not found")
		return
	}
	writeJSON(w, http.StatusOK, suggestion)
}

func (h *SuggestionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteSuggestion(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SuggestionHandler) AddFiles(w http.ResponseWriter, r *http.Request) {
	suggestionID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req addFilesToSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.AddFilesToSuggestion(suggestionID, req.FileIDs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *SuggestionHandler) RemoveFile(w http.ResponseWriter, r *http.Request) {
	suggestionID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	fileID, ok := parsePathID(w, r, "fileId")
	if !ok {
		return
	}

	if err := h.store.RemoveFileFromSuggestion(suggestionID, fileID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type materializeRequest struct {
	Destination string `json:"destination"`
}

func (h *SuggestionHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req materializeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Destination == "" {
		home, _ := os.UserHomeDir()
		req.Destination = filepath.Join(home, "Owl-organized")
	}

	result, err := h.store.MaterializeSuggestion(id, req.Destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
