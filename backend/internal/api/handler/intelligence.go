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
		tree, err := h.store.ListPhysicalFoldersAll()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tree)
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
			"id":               f.ID,
			"name":             f.Name,
			"description":      f.Description,
			"suggestion_type":  f.SuggestionType,
			"confidence":       f.Confidence,
			"file_count":       len(detail.Files),
			"preview":          preview,
			"created_at":       f.CreatedAt,
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

	slog.Info("generating folder suggestions (async)", "min_files", minFiles, "min_similarity", minSimilarity, "strategy", strategyID)

	go h.generateSuggestionsAsync(r.Context(), minFiles, minSimilarity, strategyID)

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "generating", "message": "Suggestions are being generated in the background. Poll GET /intelligence/folders/suggestions for progress."})
}

func (h *IntelligenceHandler) generateSuggestionsAsync(ctx context.Context, minFiles int, minSimilarity float64, strategyID intelligence.StrategyID) {
	slog.Info("async generation started", "min_files", minFiles, "min_similarity", minSimilarity, "strategy", strategyID)
	start := time.Now()

	if err := h.store.DeleteVirtualFoldersBySource("auto"); err != nil {
		slog.Error("failed to clear old auto suggestions", "error", err)
		return
	}
	slog.Info("cleared old auto suggestions")

	allFileIDs, err := h.store.ListAllFileIDsAll()
	if err != nil {
		slog.Error("failed to get all file IDs", "error", err)
		return
	}
	slog.Info("got all file IDs", "count", len(allFileIDs))

	if len(allFileIDs) < minFiles {
		slog.Info("not enough processed files across all watched dirs", "count", len(allFileIDs), "min_required", minFiles)
		return
	}

	slog.Info("building global corpus")
	corpusStart := time.Now()
	globalCorpus, err := h.analyzer.BuildCorpus(allFileIDs)
	if err != nil {
		slog.Error("failed to build global corpus", "error", err)
		return
	}
	slog.Info("global corpus built", "elapsed", time.Since(corpusStart).String(), "unique_terms", len(globalCorpus.Keywords))

	fileNames, err := h.store.GetFileNames(allFileIDs)
	if err != nil {
		slog.Error("failed to get file names", "error", err)
		return
	}

	trees, err := h.store.ListPhysicalFoldersAll()
	if err != nil {
		slog.Error("failed to list all physical folders", "error", err)
		return
	}
	if len(trees) == 0 {
		slog.Info("no physical folders found")
		return
	}

	folderToFileIDs := make(map[string][]int64)
	for _, tree := range trees {
		h.collectFileIDs(tree, folderToFileIDs)
	}

	var orphans []int64
	var coherentFolders []string

	slog.Info("analyzing coherence for all folders")
	coh := h.analyzeCoherenceForAllFolders(globalCorpus, fileNames, folderToFileIDs)

	for path, coherence := range coh {
		if coherence != nil {
			if coherence.IsCoherent {
				coherentFolders = append(coherentFolders, path)
				for _, out := range coherence.OutlierFiles {
					orphans = append(orphans, out.ID)
				}
			} else {
				files, _ := h.store.GetFilesInDir(path)
				for _, f := range files {
					orphans = append(orphans, f.ID)
				}
			}
		}
	}

	for _, tree := range trees {
		files, _ := h.store.GetFilesInDir(tree.Path)
		for _, f := range files {
			alreadyOrphan := false
			for _, oid := range orphans {
				if oid == f.ID {
					alreadyOrphan = true
					break
				}
			}
			if !alreadyOrphan {
				orphans = append(orphans, f.ID)
			}
		}
	}

	slog.Info("collected orphans", "total", len(orphans))

	if len(orphans) < minFiles {
		slog.Info("not enough orphan files", "orphans", len(orphans), "min_required", minFiles)
		return
	}

	strategy := h.registry.Get(strategyID)
	if strategy == nil {
		strategy = h.registry.Default()
	}

	slog.Info("running strategy on orphans", "orphans", len(orphans))
	folderSugs, err := strategy.SuggestFoldersWithCorpus(ctx, orphans, globalCorpus)
	if err != nil {
		slog.Error("strategy failed", "error", err)
		return
	}

	slog.Info("saving suggestions to DB", "suggestions", len(folderSugs))
	saved := 0
	for _, fs := range folderSugs {
		if len(fs.FileIDs) < minFiles {
			continue
		}

		folder, err := h.store.CreateVirtualFolderWithType(fs.Name, fs.Description, "auto", "new_folder", fs.Score)
		if err != nil {
			slog.Warn("failed to create virtual folder", "name", fs.Name, "error", err)
			continue
		}

		if err := h.store.AddFilesToFolder(folder.ID, fs.FileIDs, "auto"); err != nil {
			slog.Warn("failed to add files to folder", "id", folder.ID, "error", err)
			h.store.DeleteVirtualFolder(folder.ID)
			continue
		}

		saved++
		slog.Debug("saved suggestion", "id", folder.ID, "name", fs.Name, "files", len(fs.FileIDs))
	}

	slog.Info("generation complete", "saved", saved, "elapsed", time.Since(start).String())
}

func (h *IntelligenceHandler) collectFileIDs(f *store.PhysicalFolder, folderToFileIDs map[string][]int64) {
	if f.FileCount > 0 {
		files, _ := h.store.GetFilesInDir(f.Path)
		fileIDs := make([]int64, 0, len(files))
		for _, file := range files {
			fileIDs = append(fileIDs, file.ID)
		}
		folderToFileIDs[f.Path] = fileIDs
	}
	for _, child := range f.Children {
		h.collectFileIDs(child, folderToFileIDs)
	}
}

func (h *IntelligenceHandler) analyzeCoherenceForAllFolders(corpus *intelligence.Corpus, fileNames map[int64]string, folderToFileIDs map[string][]int64) map[string]*intelligence.FolderCoherence {
	coherences := make(map[string]*intelligence.FolderCoherence)
	analyzedCount := 0
	totalFolders := 0

	for _, fileIDs := range folderToFileIDs {
		if len(fileIDs) >= intelligence.MinFilesForFolder {
			totalFolders++
		}
	}

	slog.Info("analyzing coherence", "folders", totalFolders)

	for path, fileIDs := range folderToFileIDs {
		if len(fileIDs) >= intelligence.MinFilesForFolder {
			coh, err := intelligence.AnalyzeFolderCoherenceWithCorpus(corpus, fileIDs, fileNames, path)
			if err != nil {
				slog.Warn("coherence analysis failed", "path", path, "error", err)
			} else {
				coherences[path] = coh
			}
			analyzedCount++
			if analyzedCount%10 == 0 || analyzedCount == totalFolders {
				slog.Info("coherence progress", "analyzed", analyzedCount, "total", totalFolders)
			}
		}
	}

	slog.Info("coherence analysis complete", "analyzed_folders", analyzedCount, "skipped_trivial", len(folderToFileIDs)-analyzedCount)

	return coherences
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

func (h *IntelligenceHandler) RefineAllSuggestions(w http.ResponseWriter, r *http.Request) {
	slog.Info("request method=POST path=/intelligence/folders/suggestions/refine-all")

	if h.llm == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}

	autoSource := "auto"
	folders, err := h.store.ListVirtualFolders(&autoSource)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("refine-all: starting", "folders", len(folders))

	for _, f := range folders {
		go h.refineFolderAsync(f.ID)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "refining", "count": len(folders)})
}

func (h *IntelligenceHandler) refineFolderAsync(id int64) {
	folder, err := h.store.GetVirtualFolderDetail(id)
	if err != nil {
		slog.Error("refine-all: failed to get folder", "id", id, "error", err)
		return
	}

	if folder == nil {
		return
	}

	fileIDs := make([]int64, len(folder.Files))
	for i, f := range folder.Files {
		fileIDs[i] = f.ID
	}

	slog.Info("refine-all: starting refine folder", "id", id, "name", folder.Name, "files", len(fileIDs))

	corpusKeywords, err := h.analyzer.BuildCorpusTFIDF(fileIDs)
	if err != nil {
		slog.Error("refine-all: failed to build corpus", "id", id, "error", err)
		return
	}

	fileNames := make(map[int64]string)
	for _, f := range folder.Files {
		fileNames[f.ID] = f.Name
	}

	clusterFiles, err := h.suggester.BuildClusterFiles(fileIDs, corpusKeywords, fileNames)
	if err != nil {
		slog.Error("refine-all: failed to build cluster files", "id", id, "error", err)
		return
	}

	refinement, err := h.llm.RefineCluster(context.Background(), clusterFiles, fileIDs, folder.Name)
	if err != nil {
		slog.Error("refine-all: failed to refine folder", "id", id, "name", folder.Name, "error", err)
		return
	}

	if !refinement.Related {
		slog.Info("refine-all: folder not related, deleting", "id", id, "name", folder.Name)
		h.store.DeleteVirtualFolder(id)
		return
	}

	if refinement.Name != "" && refinement.Name != folder.Name {
		name := refinement.Name
		description := refinement.Description
		if _, err := h.store.UpdateVirtualFolder(id, &name, &description); err != nil {
			slog.Error("refine-all: failed to update folder", "id", id, "error", err)
			return
		}
		slog.Info("refine-all: folder refined", "id", id, "name", folder.Name, "new_name", refinement.Name, "description", refinement.Description)
	}

	if len(refinement.RemovedIDs) > 0 {
		slog.Info("refine-all: folder refined, removing files", "id", id, "name", folder.Name, "removed_count", len(refinement.RemovedIDs))
		for _, fileID := range refinement.RemovedIDs {
			h.store.RemoveFileFromFolder(id, fileID)
		}
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
	
	var count int
	var err error
	
	if dirIDStr == "" {
		err = h.store.Db().QueryRow(`
			SELECT COUNT(*) FROM files 
			WHERE status = 'active' AND processing_status = 'unprocessed'
		`).Scan(&count)
	} else {
		dirID, err := strconv.ParseInt(dirIDStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid watched_dir_id")
			return
		}
		err = h.store.Db().QueryRow(`
			SELECT COUNT(*) FROM files 
			WHERE watched_dir_id = ? AND status = 'active' AND processing_status = 'unprocessed'
		`, dirID).Scan(&count)
	}
	
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}