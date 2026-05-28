package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"owl/internal/config"
	"owl/internal/embedding"
	"owl/internal/intelligence"
	"owl/internal/llm"
	"owl/internal/store"
	"strconv"
	"time"
)

type IntelligenceHandler struct {
	store          *store.Store
	analyzer       *intelligence.Analyzer
	tagger         *intelligence.Tagger
	suggester      *intelligence.Suggester
	llm            *llm.Client
	registry       *intelligence.Registry
	folderStrategy intelligence.StrategyID
}

func NewIntelligenceHandler(s *store.Store, llmClient *llm.Client, cfg *config.Config) *IntelligenceHandler {
	analyzer := intelligence.NewAnalyzer(s.Db())
	registry := intelligence.NewRegistry()

	registry.Register(intelligence.NewContentTFIDFStrategy(analyzer, s, llmClient))

	if cfg.LLM.EmbedModel != "" || cfg.LLM.FolderStrategy == "embeddings" {
		embedURL := cfg.LLM.BaseURL
		embedClient := embedding.NewClient(embedURL, cfg.LLM.EmbedModel)
		registry.Register(intelligence.NewEmbeddingsStrategy(analyzer, s, llmClient, embedClient))
	}

	tagger := intelligence.NewTagger(analyzer, s, llmClient, registry)
	suggester := intelligence.NewSuggester(analyzer, s, llmClient, registry)

	folderStrategy := intelligence.ParseStrategyID(cfg.LLM.FolderStrategy)

	return &IntelligenceHandler{
		store:          s,
		analyzer:       analyzer,
		tagger:         tagger,
		suggester:      suggester,
		llm:            llmClient,
		registry:       registry,
		folderStrategy: folderStrategy,
	}
}

func (h *IntelligenceHandler) ListStrategies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.registry.List())
}

func (h *IntelligenceHandler) ListPhysicalFolders(w http.ResponseWriter, r *http.Request) {
	dirIDStr := r.URL.Query().Get("watched_dir_id")
	if dirIDStr == "" {
		writeError(w, http.StatusBadRequest, "watched_dir_id is required")
		return
	}

	dirID, err := strconv.ParseInt(dirIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watched_dir_id")
		return
	}

	tree, err := h.store.ListPhysicalFolders(dirID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tree)
}

func (h *IntelligenceHandler) ListPhysicalFolderFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	files, err := h.store.GetFilesInDir(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":  path,
		"files": files,
		"count": len(files),
	})
}

func (h *IntelligenceHandler) AnalyzeFolderCoherence(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	result, err := intelligence.AnalyzeFolderCoherence(h.analyzer, h.store, path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
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

	strategyID := intelligence.ParseStrategyID(r.URL.Query().Get("strategy"))

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

	slog.Info("auto-tagging files", "count", len(files), "strategy", strategyID)

	fileIDs := make([]int64, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	result, err := h.tagger.AutoTagFiles(r.Context(), fileIDs, strategyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tagCount := 0
	for _, tags := range result {
		tagCount += len(tags)
	}

	slog.Info("auto-tag complete", "files", len(files), "tagged", len(result), "tags_created", tagCount)

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

	strategyID := intelligence.ParseStrategyID(r.URL.Query().Get("strategy"))

	filter := store.FileFilter{
		WatchedDirID: &dirID,
		Limit:        1000,
	}

	files, err := h.store.ListFiles(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("auto-tagging watched dir", "dir_id", dirID, "count", len(files), "strategy", strategyID)

	fileIDs := make([]int64, len(files))
	for i, f := range files {
		fileIDs[i] = f.ID
	}

	result, err := h.tagger.AutoTagFiles(r.Context(), fileIDs, strategyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tagCount := 0
	for _, tags := range result {
		tagCount += len(tags)
	}

	slog.Info("auto-tag dir complete", "dir_id", dirID, "files", len(files), "tagged", len(result), "tags_created", tagCount)

	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(files),
		"tagged":    len(result),
		"tag_count": tagCount,
	})
}

func (h *IntelligenceHandler) ListFolderSuggestions(w http.ResponseWriter, r *http.Request) {
	autoSource := "auto"
	folders, err := h.store.ListVirtualFolders(&autoSource)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	suggestions := make(map[string]any)
	for _, f := range folders {
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

	strategyID := intelligence.ParseStrategyID(r.URL.Query().Get("strategy"))

	minFiles := intelligence.MinFilesForFolder
	if req.MinFiles != nil && *req.MinFiles > 0 {
		minFiles = *req.MinFiles
	}

	minSimilarity := 0.45
	if req.MinSimilarity != nil && *req.MinSimilarity > 0 {
		minSimilarity = *req.MinSimilarity
	}

	slog.Info("generating folder suggestions", "min_files", minFiles, "min_similarity", minSimilarity, "strategy", strategyID)
	start := time.Now()

	suggestions, err := h.suggester.SuggestVirtualFolders(r.Context(), minFiles, minSimilarity, strategyID)
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

	slog.Info("folder suggestions generated", "suggestions", len(suggestions), "created", len(created), "elapsed", time.Since(start).String())

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

func (h *IntelligenceHandler) RefineFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	slog.Info("request method=POST path=/intelligence/refine/folder", "id", id)

	if h.llm == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}

	folder, err := h.store.GetVirtualFolderDetail(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if folder == nil {
		writeError(w, http.StatusNotFound, "folder not found")
		return
	}

	fileIDs := make([]int64, len(folder.Files))
	for i, f := range folder.Files {
		fileIDs[i] = f.ID
	}

	slog.Info("llm: starting refine folder", "id", id, "name", folder.Name, "files", len(fileIDs))

	go func() {
		corpusKeywords, err := h.analyzer.BuildCorpusTFIDF(fileIDs)
		if err != nil {
			slog.Error("llm: refine folder failed to build corpus", "id", id, "error", err)
			return
		}

		fileNames := make(map[int64]string)
		for _, f := range folder.Files {
			fileNames[f.ID] = f.Name
		}

		clusterFiles, err := h.suggester.BuildClusterFiles(fileIDs, corpusKeywords, fileNames)
		if err != nil {
			slog.Error("llm: refine folder failed to build cluster files", "id", id, "error", err)
			return
		}

		refinement, err := h.llm.RefineCluster(context.Background(), clusterFiles, fileIDs, folder.Name)
		if err != nil {
			slog.Error("llm: refine folder failed", "id", id, "name", folder.Name, "error", err)
			return
		}

		if !refinement.Related {
			slog.Info("llm: folder not related, deleting", "id", id, "name", folder.Name)
			h.store.DeleteVirtualFolder(id)
			return
		}

		if refinement.Name != "" && refinement.Name != folder.Name {
			name := refinement.Name
			description := refinement.Description
			if _, err := h.store.UpdateVirtualFolder(id, &name, &description); err != nil {
				slog.Error("llm: refine folder failed to update", "id", id, "error", err)
				return
			}
			slog.Info("llm: folder refined", "id", id, "name", folder.Name, "new_name", refinement.Name, "description", refinement.Description)
		}

		if len(refinement.RemovedIDs) > 0 {
			slog.Info("llm: folder refined, removing files", "id", id, "name", folder.Name, "removed_count", len(refinement.RemovedIDs))
			for _, fileID := range refinement.RemovedIDs {
				h.store.RemoveFileFromFolder(id, fileID)
			}
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "refining", "id": id})
}

func (h *IntelligenceHandler) SmartSuggestFolders(w http.ResponseWriter, r *http.Request) {
	dirIDStr := r.URL.Query().Get("watched_dir_id")
	if dirIDStr == "" {
		writeError(w, http.StatusBadRequest, "watched_dir_id is required")
		return
	}

	dirID, err := strconv.ParseInt(dirIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watched_dir_id")
		return
	}

	slog.Info("smart-suggest: starting generation", "watched_dir_id", dirID)
	start := time.Now()

	suggestions, err := h.generateSmartSuggestions(r.Context(), dirID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("smart-suggest: generation complete", "suggestions", len(suggestions), "elapsed", time.Since(start).String())

	writeJSON(w, http.StatusOK, map[string]any{
		"suggestions": suggestions,
		"count":       len(suggestions),
	})
}

func (h *IntelligenceHandler) AcceptSmartSuggestion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type        string   `json:"type"`
		FileIDs     []int64  `json:"file_ids"`
		Name        string   `json:"name"`
		TargetID    int64    `json:"target_id"`
		SourcePaths []string `json:"source_paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch req.Type {
	case "new_folder":
		desc := "Auto-generated virtual folder"
		folder, err := h.store.CreateVirtualFolder(req.Name, desc, true)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := h.store.AddFilesToFolder(folder.ID, req.FileIDs, "auto"); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, folder)

	case "add_to_folder":
		if req.TargetID > 0 {
			if err := h.store.AddFilesToFolder(req.TargetID, req.FileIDs, "auto"); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		} else {
			desc := "Auto-generated virtual folder from existing files"
			folder, err := h.store.CreateVirtualFolder(req.Name, desc, true)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if err := h.store.AddFilesToFolder(folder.ID, req.FileIDs, "auto"); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, folder)
		}

	case "merge_folders":
		desc := "Merged virtual folder"
		folder, err := h.store.CreateVirtualFolder(req.Name, desc, true)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := h.store.AddFilesToFolder(folder.ID, req.FileIDs, "auto"); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, folder)

	default:
		writeError(w, http.StatusBadRequest, "unknown suggestion type")
	}
}

func (h *IntelligenceHandler) RefineTag(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}

	slog.Info("request method=POST path=/intelligence/refine/tag", "id", id)

	if h.llm == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}

	tag, err := h.store.GetTag(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tag == nil {
		writeError(w, http.StatusNotFound, "tag not found")
		return
	}

	files, err := h.store.GetFilesByTag(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = f.Name
	}

	keywords := []string{}
	for _, f := range files {
		if f.ProcessingStatus == "processed" {
			kws, err := h.analyzer.GetFileKeywords(f.ID, 5)
			if err == nil {
				for _, kw := range kws {
					keywords = append(keywords, kw.Term)
				}
			}
		}
		if len(keywords) >= 10 {
			break
		}
	}

	slog.Info("llm: starting refine tag", "id", id, "name", tag.Name, "files", len(fileNames))

	go func() {
		refinement, err := h.llm.RefineTag(context.Background(), tag.Name, fileNames, keywords)
		if err != nil {
			slog.Error("llm: refine tag failed", "id", id, "name", tag.Name, "error", err)
			return
		}

		if !refinement.Meaningful {
			slog.Info("llm: tag not meaningful, deleting", "id", id, "name", tag.Name)
			h.store.DeleteTag(id)
			return
		}

		if refinement.BetterName != "" && refinement.BetterName != tag.Name {
			if err := h.store.UpdateTagName(id, refinement.BetterName); err != nil {
				slog.Error("llm: refine tag failed to rename", "id", id, "error", err)
				return
			}
			slog.Info("llm: tag refined", "id", id, "old_name", tag.Name, "new_name", refinement.BetterName)
		}

		if refinement.Description != "" {
			if err := h.store.UpdateTagDescription(id, refinement.Description); err != nil {
				slog.Error("llm: refine tag failed to set description", "id", id, "error", err)
				return
			}
			slog.Info("llm: tag refined", "id", id, "name", tag.Name, "description", refinement.Description)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "refining", "id": id})
}

func (h *IntelligenceHandler) GetUnprocessedCount(w http.ResponseWriter, r *http.Request) {
	dirIDStr := r.URL.Query().Get("watched_dir_id")
	if dirIDStr == "" {
		writeError(w, http.StatusBadRequest, "watched_dir_id is required")
		return
	}

	dirID, err := strconv.ParseInt(dirIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watched_dir_id")
		return
	}

	var count int
	err = h.store.Db().QueryRow(`
		SELECT COUNT(*) FROM files 
		WHERE watched_dir_id = ? AND status = 'active' AND processing_status = 'unprocessed'
	`, dirID).Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}