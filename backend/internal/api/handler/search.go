package handler

import (
	"net/http"
	"strconv"

	"owl/internal/store"
)

type SearchHandler struct {
	store *store.Store
}

func NewSearchHandler(s *store.Store) *SearchHandler {
	return &SearchHandler{store: s}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	searchType := r.URL.Query().Get("type")

	switch searchType {
	case "files":
		results, err := h.store.SearchFiles(query, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []store.SearchFileResult{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"files": results})

	case "notes":
		results, err := h.store.SearchNotes(query, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []store.SearchNoteResult{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"notes": results})

	default:
		results, err := h.store.Search(query, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results.Files == nil {
			results.Files = []store.SearchFileResult{}
		}
		if results.Notes == nil {
			results.Notes = []store.SearchNoteResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}
