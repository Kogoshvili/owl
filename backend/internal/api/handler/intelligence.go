package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"owl/internal/config"
	"owl/internal/embedding"
	"owl/internal/extractor"
	"owl/internal/intelligence"
	"owl/internal/llm"
	"owl/internal/store"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IntelligenceHandler struct {
	store            *store.Store
	analyzer         *intelligence.Analyzer
	suggester        *intelligence.Suggester
	llm              *llm.Client
	extractor        *extractor.Extractor
	registry         *intelligence.Registry
	folderStrategy   intelligence.StrategyID
	generationMu     sync.Mutex
	generationStatus *generationStatus
	maxGuardDepth    int
}

type generationStatus struct {
	Stage        string `json:"stage"`
	Message      string `json:"message"`
	Progress     int    `json:"progress"`
	Total        int    `json:"total"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
}

func NewIntelligenceHandler(s *store.Store, llmClient *llm.Client, ext *extractor.Extractor, cfg *config.Config) *IntelligenceHandler {
	analyzer := intelligence.NewAnalyzer(s.Db())
	registry := intelligence.NewRegistry()

	registry.Register(intelligence.NewContentTFIDFStrategy(analyzer, s, llmClient))

	if cfg.LLM.EmbedModel != "" || cfg.LLM.FolderStrategy == "embeddings" {
		embedURL := cfg.LLM.BaseURL
		embedClient := embedding.NewClient(embedURL, cfg.LLM.EmbedModel)
		registry.Register(intelligence.NewEmbeddingsStrategy(analyzer, s, llmClient, embedClient))
	}

	suggester := intelligence.NewSuggester(analyzer, s, llmClient, registry)

	folderStrategy := intelligence.ParseStrategyID(cfg.LLM.FolderStrategy)

	return &IntelligenceHandler{
		store:            s,
		analyzer:         analyzer,
		suggester:        suggester,
		llm:              llmClient,
		extractor:        ext,
		registry:         registry,
		folderStrategy:   folderStrategy,
		generationStatus: nil,
		maxGuardDepth:    3,
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

func (h *IntelligenceHandler) ListFolderSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions, err := h.store.ListSuggestions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make(map[string]any)
	for _, s := range suggestions {
		detail, err := h.store.GetSuggestionDetail(s.ID)
		if err != nil || detail == nil {
			continue
		}
		preview := make([]string, 0, 5)
		for i, file := range detail.Files {
			if i >= 5 {
				break
			}
			preview = append(preview, file.Name)
		}

		result[strconv.FormatInt(s.ID, 10)] = map[string]any{
			"id":               s.ID,
			"name":             s.Name,
			"description":      s.Description,
			"suggestion_type":  s.SuggestionType,
			"confidence":       s.Confidence,
			"file_count":       len(detail.Files),
			"preview":          preview,
			"created_at":       s.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, result)
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

	strategyID := h.folderStrategy

	minFiles := intelligence.MinFilesForFolder
	if req.MinFiles != nil && *req.MinFiles > 0 {
		minFiles = *req.MinFiles
	}

	minSimilarity := 0.45
	if req.MinSimilarity != nil && *req.MinSimilarity > 0 {
		minSimilarity = *req.MinSimilarity
	}

	slog.Info("generating folder suggestions (async)", "min_files", minFiles, "min_similarity", minSimilarity, "strategy", strategyID)

	h.clearGenerationStatus()
	go h.generateSuggestionsAsync(context.Background(), minFiles, minSimilarity, strategyID)

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "generating", "message": "Suggestions are being generated in the background. Poll GET /intelligence/folders/suggestions for progress."})
}

func (h *IntelligenceHandler) generateSuggestionsAsync(ctx context.Context, minFiles int, minSimilarity float64, strategyID intelligence.StrategyID) {
	slog.Info("async generation started", "min_files", minFiles, "min_similarity", minSimilarity, "strategy", strategyID)
	start := time.Now()
	h.updateGenerationStatus("initializing", "Clearing old suggestions", 0, 1)

	if err := h.store.DeleteAllSuggestions(); err != nil {
		slog.Error("failed to clear old suggestions", "error", err)
		h.updateGenerationStatus("error", "Failed to clear old suggestions", 0, 1)
		return
	}
	slog.Info("cleared old auto suggestions")

	trees, err := h.store.ListPhysicalFoldersAll()
	if err != nil {
		slog.Error("failed to list all physical folders", "error", err)
		h.updateGenerationStatus("error", "Failed to list physical folders", 0, 1)
		return
	}
	if len(trees) == 0 {
		slog.Info("no physical folders found")
		h.updateGenerationStatus("complete", "No physical folders found", 1, 1)
		return
	}

	folderToFileIDs := make(map[string][]int64)
	for _, tree := range trees {
		h.collectFileIDs(tree, folderToFileIDs)
	}

	h.updateGenerationStatus("classifying_folders", "Classifying folders with LLM guard", 0, len(folderToFileIDs))
	slog.Info("classifying folders with LLM guard", "folders", len(folderToFileIDs))
	guardedPaths, err := h.classifyFoldersWithGuard(ctx, folderToFileIDs, trees)
	if err != nil {
		slog.Error("failed to classify folders with guard", "error", err)
		h.updateGenerationStatus("error", "Failed to classify folders", 0, 1)
		return
	}
	slog.Info("folder guard classification complete", "guarded_count", len(guardedPaths))

	h.updateGenerationStatus("filtering_folders", "Filtering out guarded folders", 0, len(folderToFileIDs))
	slog.Info("filtering out guarded folders", "folder_count", len(folderToFileIDs), "guarded_paths", len(guardedPaths))
	filteredCount := 0
	for path := range folderToFileIDs {
		isGuarded := false
		for guardedPath := range guardedPaths {
			if guardedPaths[guardedPath] && (path == guardedPath || strings.HasPrefix(path, guardedPath+"/")) {
				isGuarded = true
				break
			}
		}
		if isGuarded {
			slog.Debug("skipping guarded folder", "path", path)
			delete(folderToFileIDs, path)
			filteredCount++
		}
	}

	allFileIDs := make([]int64, 0)
	for _, ids := range folderToFileIDs {
		allFileIDs = append(allFileIDs, ids...)
	}
	slog.Info("got all file IDs (unguarded)", "count", len(allFileIDs))

	if len(allFileIDs) < minFiles {
		slog.Info("not enough processed files across unguarded folders", "count", len(allFileIDs), "min_required", minFiles)
		h.updateGenerationStatus("complete", "Not enough processed files", 1, 1)
		return
	}

	h.updateGenerationStatus("building_corpus", "Building global corpus", 0, 1)
	slog.Info("building global corpus")
	corpusStart := time.Now()
	globalCorpus, err := h.analyzer.BuildCorpus(allFileIDs)
	if err != nil {
		slog.Error("failed to build global corpus", "error", err)
		h.updateGenerationStatus("error", "Failed to build global corpus", 0, 1)
		return
	}
	slog.Info("global corpus built", "elapsed", time.Since(corpusStart).String(), "unique_terms", len(globalCorpus.Keywords))

	fileNames, err := h.store.GetFileNames(allFileIDs)
	if err != nil {
		slog.Error("failed to get file names", "error", err)
		h.updateGenerationStatus("error", "Failed to get file names", 0, 1)
		return
	}

	var orphans []int64
	var coherentFolders []string

	h.updateGenerationStatus("analyzing_coherence", "Analyzing folder coherence", 0, len(folderToFileIDs))
	slog.Info("analyzing coherence for all unguarded folders")
	coh := h.analyzeCoherenceForAllFolders(globalCorpus, fileNames, folderToFileIDs)

	analyzed := 0
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
		analyzed++
		if analyzed%10 == 0 {
			h.updateGenerationStatus("analyzing_coherence", "Analyzing folder coherence", analyzed, len(folderToFileIDs))
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
		h.updateGenerationStatus("complete", "Not enough orphan files", 1, 1)
		return
	}

	h.updateGenerationStatus("clustering", "Running strategy clustering", 0, 1)
	strategy := h.registry.Get(strategyID)
	if strategy == nil {
		strategy = h.registry.Default()
	}

	slog.Info("running strategy on orphans", "orphans", len(orphans))
	folderSugs, err := strategy.SuggestFoldersWithCorpus(ctx, orphans, globalCorpus)
	if err != nil {
		slog.Error("strategy failed", "error", err)
		h.updateGenerationStatus("error", "Strategy failed", 0, 1)
		return
	}

	h.updateGenerationStatus("saving", "Saving suggestions to DB", 0, len(folderSugs))
	slog.Info("saving suggestions to DB", "suggestions", len(folderSugs))
	saved := 0
	for _, fs := range folderSugs {
		if len(fs.FileIDs) < minFiles {
			continue
		}

		suggestion, err := h.store.CreateSuggestion(fs.Name, fs.Description, "new_folder", fs.Score)
		if err != nil {
			slog.Warn("failed to create suggestion", "name", fs.Name, "error", err)
			continue
		}

		if err := h.store.AddFilesToSuggestion(suggestion.ID, fs.FileIDs); err != nil {
			slog.Warn("failed to add files to suggestion", "id", suggestion.ID, "error", err)
			h.store.DeleteSuggestion(suggestion.ID)
			continue
		}

		saved++
		slog.Debug("saved suggestion", "id", suggestion.ID, "name", fs.Name, "files", len(fs.FileIDs))
		h.updateGenerationStatus("saving", "Saving suggestions to DB", saved, len(folderSugs))
	}

	slog.Info("generation complete", "saved", saved, "elapsed", time.Since(start).String())
	h.updateGenerationStatus("complete", fmt.Sprintf("Complete: %d suggestions generated", saved), 1, 1)
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

func (h *IntelligenceHandler) escalateGuards(trees []*store.PhysicalFolder, guardMap map[string]bool) {
	var allNodes []*store.PhysicalFolder
	var collect func(f *store.PhysicalFolder)
	collect = func(f *store.PhysicalFolder) {
		allNodes = append(allNodes, f)
		for _, child := range f.Children {
			collect(child)
		}
	}
	for _, tree := range trees {
		collect(tree)
	}

	sort.Slice(allNodes, func(i, j int) bool {
		return allNodes[i].Depth > allNodes[j].Depth
	})

	escalated := 0
	for _, node := range allNodes {
		if node.FileCount > 0 {
			continue
		}
		if len(node.Children) == 0 {
			continue
		}
		if guardMap[node.Path] {
			continue
		}
		allGuarded := true
		for _, child := range node.Children {
			if !guardMap[child.Path] {
				allGuarded = false
				break
			}
		}
		if allGuarded {
			guardMap[node.Path] = true
			if err := h.store.SetFolderGuard(node.Path, true, "llm", "All subfolders are guarded"); err != nil {
				slog.Warn("failed to escalate guard", "path", node.Path, "error", err)
			} else {
				escalated++
				slog.Info("guard escalated", "path", node.Path)
			}
		}
	}

	if escalated > 0 {
		slog.Info("guard escalation complete", "escalated", escalated)
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

func (h *IntelligenceHandler) classifyFoldersWithGuard(ctx context.Context, folderToFileIDs map[string][]int64, trees []*store.PhysicalFolder) (map[string]bool, error) {
	existingGuards, err := h.store.GetGuardedPaths()
	if err != nil {
		slog.Error("failed to load existing guard classifications", "error", err)
		return nil, err
	}

	slog.Info("folder guard classification", "existing_guards", len(existingGuards))

	if h.llm == nil {
		slog.Info("LLM not available, treating all folders as open")
		return existingGuards, nil
	}

	type queueItem struct {
		folder       *store.PhysicalFolder
		parentName   string
		parentGuarded bool
		depth        int
	}

	queue := make([]queueItem, 0)
	for _, tree := range trees {
		for _, child := range tree.Children {
			queue = append(queue, queueItem{
				folder:       child,
				parentName:   tree.Name,
				parentGuarded: false,
				depth:        1,
			})
		}
	}

	classifiedCount := 0
	skippedCount := 0
	totalToProcess := 0
	for _, tree := range trees {
		totalToProcess += countFolders(tree)
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		folder := item.folder

		if item.depth > h.maxGuardDepth {
			slog.Debug("folder exceeds max guard depth, treating as guarded", "path", folder.Path, "depth", item.depth, "max_depth", h.maxGuardDepth)
			existingGuards[folder.Path] = true
			skippedCount++
			continue
		}

		if existingGuarded, exists := existingGuards[folder.Path]; exists {
			if existingGuarded {
				slog.Debug("folder already classified as guarded, skipping subtree", "path", folder.Path)
				skippedCount += countFoldersInSubtree(folder)
				h.cleanDescendantGuards(folder.Path)
			} else {
				for _, child := range folder.Children {
					queue = append(queue, queueItem{
						folder:       child,
						parentName:   folder.Name,
						parentGuarded: false,
						depth:        item.depth + 1,
					})
				}
			}
			continue
		}

		isGuardedByAncestor := false
		for guardedPath := range existingGuards {
			if existingGuards[guardedPath] {
				if folder.Path == guardedPath || strings.HasPrefix(folder.Path, guardedPath+"/") {
					isGuardedByAncestor = true
					slog.Debug("folder guarded by ancestor", "path", folder.Path, "ancestor", guardedPath)
					break
				}
			}
		}

		if isGuardedByAncestor {
			skippedCount += countFoldersInSubtree(folder)
			existingGuards[folder.Path] = true
			continue
		}

		if !h.llm.IsAvailable(ctx) {
			slog.Warn("LLM became unavailable during classification, treating remaining folders as open")
			existingGuards[folder.Path] = false
			for _, child := range folder.Children {
				queue = append(queue, queueItem{
					folder:       child,
					parentName:   folder.Name,
					parentGuarded: false,
					depth:        item.depth + 1,
				})
			}
			continue
		}

		files, err := h.store.GetFilesInDir(folder.Path)
		if err != nil {
			slog.Warn("failed to get files for folder, treating as open", "path", folder.Path, "error", err)
			existingGuards[folder.Path] = false
			for _, child := range folder.Children {
				queue = append(queue, queueItem{
					folder:       child,
					parentName:   folder.Name,
					parentGuarded: false,
					depth:        item.depth + 1,
				})
			}
			continue
		}

		fileNames := make([]string, 0, len(files))
		for _, file := range files {
			fileNames = append(fileNames, file.Name)
		}

		subfolderNames := make([]string, 0, len(folder.Children))
		for _, child := range folder.Children {
			subfolderNames = append(subfolderNames, child.Name)
		}

		classification, err := h.llm.ClassifyFolder(ctx, filepath.Base(folder.Path), subfolderNames, fileNames, item.parentName, item.parentGuarded)
		if err != nil {
			slog.Warn("LLM classification failed, treating folder as open", "path", folder.Path, "error", err)
			existingGuards[folder.Path] = false
			for _, child := range folder.Children {
				queue = append(queue, queueItem{
					folder:       child,
					parentName:   folder.Name,
					parentGuarded: false,
					depth:        item.depth + 1,
				})
			}
			continue
		}

		guarded := classification.Related
		source := "llm"
		if err := h.store.SetFolderGuard(folder.Path, guarded, source, classification.Reason); err != nil {
			slog.Warn("failed to save guard classification", "path", folder.Path, "error", err)
		}

		existingGuards[folder.Path] = guarded
		classifiedCount++

		if guarded {
			slog.Info("folder classified as guarded, skipping subtree", "path", folder.Path, "reason", classification.Reason, "children_skipped", len(folder.Children))
			skippedCount += countFoldersInSubtree(folder)
			h.cleanDescendantGuards(folder.Path)
		} else {
			slog.Debug("folder classified as open, will process children", "path", folder.Path, "reason", classification.Reason)
			for _, child := range folder.Children {
				queue = append(queue, queueItem{
					folder:       child,
					parentName:   folder.Name,
					parentGuarded: false,
					depth:        item.depth + 1,
				})
			}
		}

		if classifiedCount%5 == 0 {
			h.updateGenerationStatus("classifying_folders", "Classifying folders with LLM guard", classifiedCount, totalToProcess)
			slog.Info("folder guard progress", "classified", classifiedCount, "skipped", skippedCount)
		}
	}

	totalGuarded := 0
	for _, guarded := range existingGuards {
		if guarded {
			totalGuarded++
		}
	}

	slog.Info("folder guard classification complete", "classified", classifiedCount, "skipped", skippedCount, "total_guarded", totalGuarded, "max_depth", h.maxGuardDepth)

	return existingGuards, nil
}

func (h *IntelligenceHandler) cleanDescendantGuards(parentPath string) {
	guards, err := h.store.ListFolderGuards()
	if err != nil {
		slog.Warn("failed to list guard classifications to clean descendants", "path", parentPath, "error", err)
		return
	}

	for _, guard := range guards {
		if strings.HasPrefix(guard.Path, parentPath+"/") {
			if err := h.store.DeleteFolderGuard(guard.Path); err != nil {
				slog.Warn("failed to delete descendant guard classification", "path", guard.Path, "error", err)
			} else {
				slog.Debug("cleaned descendant guard classification", "path", guard.Path)
			}
		}
	}
}

func countFolders(f *store.PhysicalFolder) int {
	count := len(f.Children)
	for _, child := range f.Children {
		count += countFolders(child)
	}
	return count
}

func countFoldersInSubtree(f *store.PhysicalFolder) int {
	count := 1
	for _, child := range f.Children {
		count += countFoldersInSubtree(child)
	}
	return count
}

func (h *IntelligenceHandler) updateGenerationStatus(stage, message string, progress, total int) {
	h.generationMu.Lock()
	defer h.generationMu.Unlock()

	if h.generationStatus == nil {
		h.generationStatus = &generationStatus{
			StartedAt: time.Now().Format(time.RFC3339),
		}
	}

	h.generationStatus.Stage = stage
	h.generationStatus.Message = message
	h.generationStatus.Progress = progress
	h.generationStatus.Total = total

	if stage == "complete" || stage == "error" {
		h.generationStatus.CompletedAt = time.Now().Format(time.RFC3339)
	}
}

func (h *IntelligenceHandler) getGenerationStatus() *generationStatus {
	h.generationMu.Lock()
	defer h.generationMu.Unlock()
	return h.generationStatus
}

func (h *IntelligenceHandler) clearGenerationStatus() {
	h.generationMu.Lock()
	defer h.generationMu.Unlock()
	h.generationStatus = nil
}

func (h *IntelligenceHandler) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	status := h.getGenerationStatus()
	if status == nil {
		writeJSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}

	result := map[string]any{
		"active":      true,
		"stage":       status.Stage,
		"message":     status.Message,
		"progress":    status.Progress,
		"total":       status.Total,
		"started_at":  status.StartedAt,
	}
	if status.CompletedAt != "" {
		result["completed_at"] = status.CompletedAt
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *IntelligenceHandler) RunGuard(w http.ResponseWriter, r *http.Request) {
	slog.Info("request method=POST path=/intelligence/guard/run")

	go func() {
		trees, err := h.store.ListPhysicalFoldersAll()
		if err != nil {
			slog.Error("guard: failed to list physical folders", "error", err)
			return
		}

		folderToFileIDs := make(map[string][]int64)
		for _, tree := range trees {
			h.collectFileIDs(tree, folderToFileIDs)
		}

		guardMap, err := h.classifyFoldersWithGuard(context.Background(), folderToFileIDs, trees)
		if err != nil {
			slog.Error("guard: classification failed", "error", err)
			return
		}

		h.escalateGuards(trees, guardMap)
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "guard started"})
}

func (h *IntelligenceHandler) ExtractOrphans(w http.ResponseWriter, r *http.Request) {
	slog.Info("request method=POST path=/intelligence/files/extract-orphans")

	guardedPaths, err := h.store.GetGuardedPaths()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go func() {
		trees, err := h.store.ListPhysicalFoldersAll()
		if err != nil {
			slog.Error("extract orphans: failed to list physical folders", "error", err)
			return
		}

		queued := 0
		for _, tree := range trees {
			queued += h.extractOrphansInTree(tree, guardedPaths)
		}

		if queued > 0 && h.extractor != nil {
			slog.Info("extract orphans: starting processing", "queued", queued)
			h.extractor.ProcessAll(context.Background())
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "extraction started"})
}

func (h *IntelligenceHandler) extractOrphansInTree(folder *store.PhysicalFolder, guardedPaths map[string]bool) int {
	isGuarded := false
	for path := range guardedPaths {
		if guardedPaths[path] && (folder.Path == path || strings.HasPrefix(folder.Path, path+"/")) {
			isGuarded = true
			break
		}
	}

	if isGuarded {
		slog.Debug("extract orphans: skipping guarded folder", "path", folder.Path)
		return 0
	}

	queued := 0

	if folder.FileCount > 0 {
		files, err := h.store.GetFilesInDir(folder.Path)
		if err != nil {
			slog.Warn("extract orphans: failed to get files", "path", folder.Path, "error", err)
		} else {
			for _, f := range files {
				if f.ProcessingStatus == "unprocessed" || f.ProcessingStatus == "stale" || f.ProcessingStatus == "failed" {
					if err := h.store.QueueFileForExtraction(f.ID); err != nil {
						slog.Debug("extract orphans: skipped file", "id", f.ID, "reason", err)
					} else {
						queued++
					}
				}
			}
		}
	}

	for _, child := range folder.Children {
		queued += h.extractOrphansInTree(child, guardedPaths)
	}

	return queued
}

func (h *IntelligenceHandler) DismissFolderSuggestion(w http.ResponseWriter, r *http.Request) {
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

	suggestion, err := h.store.GetSuggestionDetail(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if suggestion == nil {
		writeError(w, http.StatusNotFound, "suggestion not found")
		return
	}

	fileIDs := make([]int64, len(suggestion.Files))
	for i, f := range suggestion.Files {
		fileIDs[i] = f.ID
	}

	slog.Info("llm: starting refine suggestion", "id", id, "name", suggestion.Name, "files", len(fileIDs))

	go func() {
		corpusKeywords, err := h.analyzer.BuildCorpusTFIDF(fileIDs)
		if err != nil {
			slog.Error("llm: refine suggestion failed to build corpus", "id", id, "error", err)
			return
		}

		fileNames := make(map[int64]string)
		for _, f := range suggestion.Files {
			fileNames[f.ID] = f.Name
		}

		clusterFiles, err := h.suggester.BuildClusterFiles(fileIDs, corpusKeywords, fileNames)
		if err != nil {
			slog.Error("llm: refine suggestion failed to build cluster files", "id", id, "error", err)
			return
		}

		refinement, err := h.llm.RefineCluster(context.Background(), clusterFiles, fileIDs, suggestion.Name)
		if err != nil {
			slog.Error("llm: refine suggestion failed", "id", id, "name", suggestion.Name, "error", err)
			return
		}

		if !refinement.Related {
			slog.Info("llm: suggestion not related, deleting", "id", id, "name", suggestion.Name)
			h.store.DeleteSuggestion(id)
			return
		}

		if refinement.Name != "" && refinement.Name != suggestion.Name {
			name := refinement.Name
			description := refinement.Description
			if _, err := h.store.UpdateSuggestion(id, &name, &description); err != nil {
				slog.Error("llm: refine suggestion failed to update", "id", id, "error", err)
				return
			}
			slog.Info("llm: suggestion refined", "id", id, "name", suggestion.Name, "new_name", refinement.Name, "description", refinement.Description)
		}

		if len(refinement.RemovedIDs) > 0 {
			slog.Info("llm: suggestion refined, removing files", "id", id, "name", suggestion.Name, "removed_count", len(refinement.RemovedIDs))
			for _, fileID := range refinement.RemovedIDs {
				h.store.RemoveFileFromSuggestion(id, fileID)
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

	suggestions, err := h.store.ListSuggestions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	slog.Info("refine-all: starting", "suggestions", len(suggestions))

	for _, s := range suggestions {
		go h.refineSuggestionAsync(s.ID)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"status": "refining", "count": len(suggestions)})
}

func (h *IntelligenceHandler) refineSuggestionAsync(id int64) {
	suggestion, err := h.store.GetSuggestionDetail(id)
	if err != nil {
		slog.Error("refine-all: failed to get suggestion", "id", id, "error", err)
		return
	}

	if suggestion == nil {
		return
	}

	fileIDs := make([]int64, len(suggestion.Files))
	for i, f := range suggestion.Files {
		fileIDs[i] = f.ID
	}

	slog.Info("refine-all: starting refine suggestion", "id", id, "name", suggestion.Name, "files", len(fileIDs))

	corpusKeywords, err := h.analyzer.BuildCorpusTFIDF(fileIDs)
	if err != nil {
		slog.Error("refine-all: failed to build corpus", "id", id, "error", err)
		return
	}

	fileNames := make(map[int64]string)
	for _, f := range suggestion.Files {
		fileNames[f.ID] = f.Name
	}

	clusterFiles, err := h.suggester.BuildClusterFiles(fileIDs, corpusKeywords, fileNames)
	if err != nil {
		slog.Error("refine-all: failed to build cluster files", "id", id, "error", err)
		return
	}

	refinement, err := h.llm.RefineCluster(context.Background(), clusterFiles, fileIDs, suggestion.Name)
	if err != nil {
		slog.Error("refine-all: failed to refine suggestion", "id", id, "name", suggestion.Name, "error", err)
		return
	}

	if !refinement.Related {
		slog.Info("refine-all: suggestion not related, deleting", "id", id, "name", suggestion.Name)
		h.store.DeleteSuggestion(id)
		return
	}

	if refinement.Name != "" && refinement.Name != suggestion.Name {
		name := refinement.Name
		description := refinement.Description
		if _, err := h.store.UpdateSuggestion(id, &name, &description); err != nil {
			slog.Error("refine-all: failed to update suggestion", "id", id, "error", err)
			return
		}
		slog.Info("refine-all: suggestion refined", "id", id, "name", suggestion.Name, "new_name", refinement.Name, "description", refinement.Description)
	}

	if len(refinement.RemovedIDs) > 0 {
		slog.Info("refine-all: suggestion refined, removing files", "id", id, "name", suggestion.Name, "removed_count", len(refinement.RemovedIDs))
		for _, fileID := range refinement.RemovedIDs {
			h.store.RemoveFileFromSuggestion(id, fileID)
		}
	}
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

type processingStats struct {
	Guarded     int `json:"guarded"`
	Open        int `json:"open"`
	Extractable int `json:"extractable"`
	Queued      int `json:"queued"`
	Processing  int `json:"processing"`
	Processed   int `json:"processed"`
	Failed      int `json:"failed"`
}

func (h *IntelligenceHandler) GetProcessingStats(w http.ResponseWriter, r *http.Request) {
	var stats processingStats

	h.store.Db().QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'active' AND processing_status = 'unprocessed' AND LOWER(extension) IN `+store.SupportedExtsSQL).Scan(&stats.Extractable)
	h.store.Db().QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'active' AND processing_status = 'queued'`).Scan(&stats.Queued)
	h.store.Db().QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'active' AND processing_status = 'processing'`).Scan(&stats.Processing)
	h.store.Db().QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'active' AND processing_status = 'processed'`).Scan(&stats.Processed)
	h.store.Db().QueryRow(`SELECT COUNT(*) FROM files WHERE status = 'active' AND processing_status = 'failed'`).Scan(&stats.Failed)

	allGuards, err := h.store.ListFolderGuards()
	if err == nil {
		for _, g := range allGuards {
			if g.Guarded {
				stats.Guarded++
			} else {
				stats.Open++
			}
		}
	}

	writeJSON(w, http.StatusOK, stats)
}

func (h *IntelligenceHandler) ListFolderGuards(w http.ResponseWriter, r *http.Request) {
	guards, err := h.store.ListFolderGuards()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, guards)
}

type setFolderGuardRequest struct {
	Path    string `json:"path"`
	Guarded bool   `json:"guarded"`
}

func (h *IntelligenceHandler) SetFolderGuard(w http.ResponseWriter, r *http.Request) {
	var req setFolderGuardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	reason := "User manually set this folder"
	if !req.Guarded {
		reason = "User manually unguarded this folder"
	}

	if err := h.store.SetFolderGuard(req.Path, req.Guarded, "user", reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if req.Guarded {
		h.cleanDescendantGuards(req.Path)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}