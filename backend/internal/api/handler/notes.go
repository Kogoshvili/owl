package handler

import (
	"encoding/json"
	"net/http"

	"owl/internal/store"
)

type NoteHandler struct {
	store *store.Store
}

func NewNoteHandler(s *store.Store) *NoteHandler {
	return &NoteHandler{store: s}
}

type createNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updateNoteRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

type materializeNoteRequest struct {
	Path string `json:"path"`
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	notes, err := h.store.ListNotes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if notes == nil {
		notes = []store.Note{}
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *NoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	note, err := h.store.GetNote(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if note == nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}

	tags, _ := h.store.ListNoteTags(id)
	folders, _ := h.store.ListNoteFolders(id)

	result := map[string]any{
		"note":    note,
		"tags":    tags,
		"folders": folders,
	}
	if tags == nil {
		result["tags"] = []store.Tag{}
	}
	if folders == nil {
		result["folders"] = []store.VirtualFolder{}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	note, err := h.store.CreateNote(req.Title, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req updateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	note, err := h.store.UpdateNote(id, req.Title, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if note == nil {
		writeError(w, http.StatusNotFound, "note not found")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteNote(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NoteHandler) Materialize(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req materializeNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	if err := h.store.SetNoteMaterialized(id, req.Path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	note, _ := h.store.GetNote(id)
	writeJSON(w, http.StatusOK, note)
}
