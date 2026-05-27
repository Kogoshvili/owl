package handler

import (
	"encoding/json"
	"net/http"

	"owl/internal/store"
)

type TagHandler struct {
	store *store.Store
}

func NewTagHandler(s *store.Store) *TagHandler {
	return &TagHandler{store: s}
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.store.ListTags()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tags == nil {
		tags = []store.Tag{}
	}
	writeJSON(w, http.StatusOK, tags)
}

type addTagRequest struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

func (h *TagHandler) AddFileTag(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req addTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source := req.Source
	if source == "" {
		source = "manual"
	}

	tag, err := h.store.EnsureTag(req.Name, source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.store.AddFileTag(fileID, tag.ID, source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (h *TagHandler) RemoveFileTag(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	tagID, ok := parsePathID(w, r, "tagId")
	if !ok {
		return
	}

	if err := h.store.RemoveFileTag(fileID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TagHandler) AddNoteTag(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req addTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	source := req.Source
	if source == "" {
		source = "manual"
	}

	tag, err := h.store.EnsureTag(req.Name, source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := h.store.AddNoteTag(noteID, tag.ID, source); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tag)
}

func (h *TagHandler) RemoveNoteTag(w http.ResponseWriter, r *http.Request) {
	noteID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	tagID, ok := parsePathID(w, r, "tagId")
	if !ok {
		return
	}

	if err := h.store.RemoveNoteTag(noteID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
