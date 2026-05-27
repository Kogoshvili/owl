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

	scopes := r.URL.Query().Get("scopes")

	results, err := h.store.Search(query, scopes, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results)
}
