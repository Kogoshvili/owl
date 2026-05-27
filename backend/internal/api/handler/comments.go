package handler

import (
	"encoding/json"
	"net/http"

	"owl/internal/store"
)

type CommentHandler struct {
	store *store.Store
}

func NewCommentHandler(s *store.Store) *CommentHandler {
	return &CommentHandler{store: s}
}

type upsertCommentRequest struct {
	Content string `json:"content"`
}

func (h *CommentHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	var req upsertCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	file, err := h.store.GetFile(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if file == nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	comment, err := h.store.UpsertComment(fileID, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comment)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	fileID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteComment(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
