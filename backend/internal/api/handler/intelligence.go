package handler

import (
	"encoding/json"
	"net/http"
	"owl/internal/intelligence"
	"owl/internal/store"
	"strconv"
)

type IntelligenceHandler struct {
	store      *store.Store
	analyzer   *intelligence.Analyzer
	tagger     *intelligence.Tagger
	suggester  *intelligence.Suggester
}

func NewIntelligenceHandler(s *store.Store) *IntelligenceHandler {
	analyzer := intelligence.NewAnalyzer(s.Db())
	return &IntelligenceHandler{
		store:     s,
		analyzer:  analyzer,
		tagger:    intelligence.NewTagger(analyzer, s),
		suggester: intelligence.NewSuggester(analyzer, s),
	}
}

func (h *IntelligenceHandler) TagFile(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	tags, err := h.tagger.AutoTagFile(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tags": tags})
}

type tagFilesRequest struct {
	WatchedDirID     *int64  `json:"watched_dir_id"`
	Extension        *string `json:"extension"`
	ProcessingStatus *string `json:"processing_status"`
	Limit            *int    `json:"limit"`
}

func (h *IntelligenceHandler) TagFiles(w http.ResponseWriter, r *http.Request) {
	var req tagFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	filter := store.FileFilter{
		Extension:        req.Extension,
		ProcessingStatus: req.ProcessingStatus,
		WatchedDirID:     req.WatchedDirID,
	}

	limit := 100
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}
	filter.Limit = limit

	files, err := h.store.ListFiles(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileIDs := make([]int64, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	result, err := h.tagger.AutoTagFiles(fileIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tagCount := 0
	for _, tags := range result {
		tagCount += len(tags)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(files),
		"tagged":    len(result),
		"tag_count": tagCount,
	})
}

func (h *IntelligenceHandler) TagWatchedDir(w http.ResponseWriter, r *http.Request) {
	dirID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	filter := store.FileFilter{
		WatchedDirID: &dirID,
		Limit:        1000,
	}

	files, err := h.store.ListFiles(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileIDs := make([]int64, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	result, err := h.tagger.AutoTagFiles(fileIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tagCount := 0
	for _, tags := range result {
		tagCount += len(tags)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(files),
		"tagged":    len(result),
		"tag_count": tagCount,
	})
}

func (h *IntelligenceHandler) ListFolderSuggestions(w http.ResponseWriter, r *http.Request) {
	folders, err := h.store.ListVirtualFolders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	suggestions := make(map[string]any)
	for _, f := range folders {
		if f.Source == "auto" {
			detail, err := h.store.GetVirtualFolderDetail(f.ID)
			if err != nil {
				continue
			}
			preview := make([]string, 0, 5)
			for i, file := range detail.Files {
				if i >= 5 {
					break
				}
				preview = append(preview, file.Name)
			}

			suggestions[strconv.FormatInt(f.ID, 10)] = map[string]any{
				"id":          f.ID,
				"name":        f.Name,
				"description": f.Description,
				"file_count":  len(detail.Files),
				"preview":     preview,
				"created_at":  f.CreatedAt,
			}
		}
	}

	writeJSON(w, http.StatusOK, suggestions)
}

type generateSuggestionsRequest struct {
	Name            *string  `json:"name"`
	Description     *string  `json:"description"`
	MinFiles        *int     `json:"min_files"`
	MinSimilarity   *float64 `json:"min_similarity"`
}

func (h *IntelligenceHandler) GenerateSuggestions(w http.ResponseWriter, r *http.Request) {
	var req generateSuggestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Use the default threshold from the suggester package
	minFiles := intelligence.MinFilesForFolder
	if req.MinFiles != nil && *req.MinFiles > 0 {
		minFiles = *req.MinFiles
	}

	minSimilarity := 0.3
	if req.MinSimilarity != nil && *req.MinSimilarity > 0 {
		minSimilarity = *req.MinSimilarity
	}

	suggestions, err := h.suggester.SuggestVirtualFolders(minFiles, minSimilarity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	created := []store.VirtualFolder{}
	for _, s := range suggestions {
		name := s.Name
		description := s.Description
		if req.Name != nil {
			name = *req.Name
		}
		if req.Description != nil {
			description = *req.Description
		}

		updatedSuggestion := intelligence.Suggestion{
			Name:        name,
			Description: description,
			FileIDs:     s.FileIDs,
			Score:       s.Score,
			Preview:     s.Preview,
		}

		folder, err := h.suggester.CreateFolderFromSuggestion(updatedSuggestion)
		if err != nil {
			continue
		}
		created = append(created, *folder)
	}

	writeJSON(w, http.StatusOK, map[string]any{"created": created})
}

func (h *IntelligenceHandler) AcceptFolderSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	folder, err := h.store.UpdateVirtualFolderSource(id, "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, folder)
}

func (h *IntelligenceHandler) DismissFolderSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteVirtualFolder(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *IntelligenceHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	var sourcePtr *string
	if source == "auto" || source == "manual" {
		sourcePtr = &source
	}

	tags, err := h.store.ListTagsWithCount(sourcePtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tags)
}

func (h *IntelligenceHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tag, err := h.store.EnsureTag(req.Name, "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tag)
}

func (h *IntelligenceHandler) ListTagFiles(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	filter := store.FileFilter{
		SortBy:    r.URL.Query().Get("sort"),
		SortOrder: r.URL.Query().Get("order"),
	}

	if ext := r.URL.Query().Get("extension"); ext != "" {
		filter.Extension = &ext
	}
	if status := r.URL.Query().Get("processing_status"); status != "" {
		filter.ProcessingStatus = &status
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	filter.Limit = limit

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	filter.Offset = offset

	total, files, err := h.store.ListFilesByTag(id, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"files": files,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

func (h *IntelligenceHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	if err := h.store.DeleteTag(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *IntelligenceHandler) AcceptTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	tag, err := h.store.UpdateTagSource(id, "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tag)
}