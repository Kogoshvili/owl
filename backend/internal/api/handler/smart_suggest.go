package handler

import (
	"context"
	"log/slog"
	"owl/internal/intelligence"
	"owl/internal/store"
	"time"
)

type SmartSuggestion struct {
	Type        string  `json:"type"` // "new_folder", "add_to_folder", "merge_folders"
	FileIDs     []int64 `json:"file_ids"`
	TargetPath  string  `json:"target_path,omitempty"`  // for add_to_folder
	TargetID    int64   `json:"target_id,omitempty"`     // virtual folder ID
	SourcePaths []string `json:"source_paths,omitempty"` // for merge_folders
	Name        string  `json:"name,omitempty"`          // for new_folder
	Description string  `json:"description,omitempty"`
	Confidence  float64 `json:"confidence"`
	FileCount   int     `json:"file_count"`
	Preview     []string `json:"preview,omitempty"`
}

func (h *IntelligenceHandler) generateSmartSuggestions(ctx context.Context, watchedDirID int64) ([]SmartSuggestion, error) {
	strategy := h.registry.Get(h.folderStrategy)
	if strategy == nil {
		strategy = h.registry.Default()
	}
	if strategy == nil {
		return nil, nil
	}

	wdPath, err := h.store.GetWatchedDirPath(watchedDirID)
	if err != nil {
		return nil, err
	}

	tree, err := h.store.ListPhysicalFolders(watchedDirID)
	if err != nil {
		return nil, err
	}
	if len(tree) == 0 {
		return nil, nil
	}

	root := tree[0]
	rootFiles, _ := h.store.GetFilesInDir(root.Path)
	slog.Info("smart-suggest: starting", "root_path", root.Path, "root_files", len(rootFiles), "total_nodes", len(tree))

	// Get all file IDs in the watched directory for global corpus build
	allFileIDs, err := h.store.ListAllFileIDs(watchedDirID)
	if err != nil {
		return nil, err
	}
	slog.Info("smart-suggest: building global corpus", "files", len(allFileIDs))
	start := time.Now()
	globalCorpus, err := h.analyzer.BuildCorpus(allFileIDs)
	if err != nil {
		return nil, err
	}
	slog.Info("smart-suggest: global corpus built", "elapsed", time.Since(start).String(), "unique_terms", len(globalCorpus.Keywords))

	// Collect all file names for outlier tracking
	fileNames, err := h.store.GetFileNames(allFileIDs)
	if err != nil {
		return nil, err
	}

	// Collect all folders with file IDs
	folderToFileIDs := make(map[string][]int64)
	var collectFileIDs func(f *store.PhysicalFolder)
	collectFileIDs = func(f *store.PhysicalFolder) {
		if f.Path == wdPath {
			files, _ := h.store.GetFilesInDir(f.Path)
			for _, file := range files {
				folderToFileIDs[f.Path] = append(folderToFileIDs[f.Path], file.ID)
			}
		} else if f.FileCount > 0 {
			files, _ := h.store.GetFilesInDir(f.Path)
			fileIDs := make([]int64, 0, len(files))
			for _, file := range files {
				fileIDs = append(fileIDs, file.ID)
			}
			folderToFileIDs[f.Path] = fileIDs
		}
		for _, child := range f.Children {
			collectFileIDs(child)
		}
	}
	collectFileIDs(root)

	// Analyze folder coherence using pre-built corpus
	coherences := make(map[string]*intelligence.FolderCoherence)
	foldersToAnalyze := 0
	for path, fileIDs := range folderToFileIDs {
		if path != wdPath && len(fileIDs) >= intelligence.MinFilesForFolder {
			foldersToAnalyze++
		}
	}
	slog.Info("smart-suggest: analyzing coherence", "folders", foldersToAnalyze)

	analyzedCount := 0
	for path, fileIDs := range folderToFileIDs {
		if path != wdPath && len(fileIDs) >= intelligence.MinFilesForFolder {
			coh, err := intelligence.AnalyzeFolderCoherenceWithCorpus(globalCorpus, fileIDs, fileNames, path)
			if err != nil {
				slog.Warn("smart-suggest: coherence analysis failed", "path", path, "error", err)
			} else {
				coherences[path] = coh
			}
			analyzedCount++
			if analyzedCount%10 == 0 || analyzedCount == foldersToAnalyze {
				slog.Info("smart-suggest: coherence progress", "analyzed", analyzedCount, "total", foldersToAnalyze)
			}
		}
	}
	slog.Info("smart-suggest: coherence analysis complete", "analyzed_folders", analyzedCount, "skipped_trivial", len(folderToFileIDs)-analyzedCount)

	var orphans []int64
	var orphanFiles []store.File
	var coherentFolders []string

	var collectOrphans func(f *store.PhysicalFolder)
	collectOrphans = func(f *store.PhysicalFolder) {
		isRoot := f.Path == wdPath
		coh := coherences[f.Path]

		if isRoot {
			files, err := h.store.GetFilesInDir(f.Path)
			if err == nil {
				for _, file := range files {
					orphans = append(orphans, file.ID)
					orphanFiles = append(orphanFiles, file)
				}
			}
		} else if coh != nil {
			if !coh.IsCoherent {
				files, err := h.store.GetFilesInDir(f.Path)
				if err == nil {
					for _, file := range files {
						orphans = append(orphans, file.ID)
						orphanFiles = append(orphanFiles, file)
					}
				}
			} else {
				coherentFolders = append(coherentFolders, f.Path)
				for _, out := range coh.OutlierFiles {
					orphans = append(orphans, out.ID)
				}
			}
		} else if !isRoot {
			slog.Warn("smart-suggest: skipping folder (no coherence result)", "path", f.Path)
		}

		for _, child := range f.Children {
			collectOrphans(child)
		}
	}
	collectOrphans(root)

	slog.Info("smart-suggest: collected orphans", "total", len(orphans), "coherent_folders", len(coherentFolders))

	var suggestions []SmartSuggestion

	if len(orphans) >= intelligence.MinFilesForFolder {
		slog.Info("smart-suggest: running strategy on orphans", "orphans", len(orphans))
		folderSugs, err := strategy.SuggestFoldersWithCorpus(ctx, orphans, globalCorpus)
		if err != nil {
			slog.Warn("smart-suggest: strategy failed", "error", err)
		} else {
		for _, fs := range folderSugs {
			if len(fs.FileIDs) < intelligence.MinFilesForFolder {
				continue
			}

			matchingFolder, folderID, matchScore := h.findMatchingFolderWithCorpus(globalCorpus, coherentFolders, fs, 0.3)

			if matchScore > 0.3 {
				suggestions = append(suggestions, SmartSuggestion{
					Type:       "add_to_folder",
					FileIDs:    fs.FileIDs,
					Name:       matchingFolder,
					TargetPath: matchingFolder,
					TargetID:   folderID,
					Confidence: matchScore,
					FileCount:  len(fs.FileIDs),
					Preview:    fs.Preview,
				})
			} else {
				suggestions = append(suggestions, SmartSuggestion{
					Type:        "new_folder",
					FileIDs:     fs.FileIDs,
					Name:        fs.Name,
					Description: fs.Description,
					Confidence:  fs.Score,
					FileCount:   len(fs.FileIDs),
					Preview:     fs.Preview,
				})
			}
		}
		}
	}

	siblingGroups := h.findSiblingFolders(root)
	for _, pair := range siblingGroups {
		coh1, cok1 := coherences[pair[0]]
		coh2, cok2 := coherences[pair[1]]
		if cok1 && cok2 && coh1.IsCoherent && coh2.IsCoherent {
			sim := h.interFolderSimilarityWithCorpus(globalCorpus, pair[0], pair[1])
			if sim > 0.5 {
				files1, _ := h.store.GetFilesInDir(pair[0])
				files2, _ := h.store.GetFilesInDir(pair[1])
				var allIDs []int64
				for _, f := range files1 {
					allIDs = append(allIDs, f.ID)
				}
				for _, f := range files2 {
					allIDs = append(allIDs, f.ID)
				}

				suggestions = append(suggestions, SmartSuggestion{
					Type:        "merge_folders",
					FileIDs:     allIDs,
					SourcePaths: pair[:],
					Name:        "merged folder",
					Confidence:  sim,
					FileCount:   len(allIDs),
				})
			}
		}
	}

	slog.Info("smart-suggest: generated suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (h *IntelligenceHandler) findMatchingFolderWithCorpus(corpus *intelligence.Corpus, coherentFolders []string, sug intelligence.FolderSuggestion, minScore float64) (string, int64, float64) {
	var best string
	var bestID int64
	var bestScore float64

	for _, folderPath := range coherentFolders {
		files, err := h.store.GetFilesInDir(folderPath)
		if err != nil || len(files) == 0 {
			continue
		}

		folderFileIDs := make([]int64, len(files))
		for i, f := range files {
			folderFileIDs[i] = f.ID
		}

		// Use pre-built corpus for all files
		var totalSim float64
		var count int
		for _, sugID := range sug.FileIDs {
			sugMap := corpus.KeywordMap[sugID]
			if sugMap == nil {
				continue
			}
			for _, folderID := range folderFileIDs {
				folderMap := corpus.KeywordMap[folderID]
				if folderMap == nil {
					continue
				}
				sim := cosineSimilarityMaps(sugMap, folderMap)
				totalSim += sim
				count++
			}
		}

		if count > 0 {
			avgSim := totalSim / float64(count)
			if avgSim > bestScore {
				bestScore = avgSim
				best = folderPath
			}
		}
	}

	if bestScore < minScore {
		return "", 0, 0
	}

	return best, bestID, bestScore
}

func (h *IntelligenceHandler) findSiblingFolders(root *store.PhysicalFolder) [][2]string {
	var pairs [][2]string

	var walk func(p *store.PhysicalFolder)
	walk = func(p *store.PhysicalFolder) {
		for i := 0; i < len(p.Children); i++ {
			for j := i + 1; j < len(p.Children); j++ {
				pairs = append(pairs, [2]string{p.Children[i].Path, p.Children[j].Path})
			}
			walk(p.Children[i])
		}
	}
	walk(root)

	return pairs
}

func (h *IntelligenceHandler) interFolderSimilarityWithCorpus(corpus *intelligence.Corpus, path1, path2 string) float64 {
	files1, err := h.store.GetFilesInDir(path1)
	if err != nil || len(files1) == 0 {
		return 0
	}
	files2, err := h.store.GetFilesInDir(path2)
	if err != nil || len(files2) == 0 {
		return 0
	}

	fileIDs1 := make([]int64, len(files1))
	for i, f := range files1 {
		fileIDs1[i] = f.ID
	}
	fileIDs2 := make([]int64, len(files2))
	for i, f := range files2 {
		fileIDs2[i] = f.ID
	}

	// Use pre-built corpus for all files
	var totalSim float64
	var count int
	for _, id1 := range fileIDs1 {
		map1 := corpus.KeywordMap[id1]
		if map1 == nil {
			continue
		}
		for _, id2 := range fileIDs2 {
			map2 := corpus.KeywordMap[id2]
			if map2 == nil {
				continue
			}
			sim := cosineSimilarityMaps(map1, map2)
			totalSim += sim
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalSim / float64(count)
}

func (h *IntelligenceHandler) interFolderSimilarity(path1, path2 string) float64 {
	files1, err := h.store.GetFilesInDir(path1)
	if err != nil || len(files1) == 0 {
		return 0
	}
	files2, err := h.store.GetFilesInDir(path2)
	if err != nil || len(files2) == 0 {
		return 0
	}

	var allIDs []int64
	for _, f := range files1 {
		allIDs = append(allIDs, f.ID)
	}
	for _, f := range files2 {
		allIDs = append(allIDs, f.ID)
	}

	corpus, err := h.analyzer.BuildCorpusTFIDF(allIDs)
	if err != nil {
		return 0
	}

	var totalSim float64
	var count int
	for _, f1 := range files1 {
		for _, f2 := range files2 {
			sim := cosineSimilarityMaps(
				keywordsToMap(corpus[f1.ID]),
				keywordsToMap(corpus[f2.ID]),
			)
			totalSim += sim
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return totalSim / float64(count)
}

func keywordsToMap(keywords []intelligence.Keyword) map[string]float64 {
	m := make(map[string]float64)
	for _, kw := range keywords {
		m[kw.Term] = kw.Score
	}
	return m
}

func cosineSimilarityMaps(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for term, scoreA := range a {
		normA += scoreA * scoreA
		if scoreB, ok := b[term]; ok {
			dotProduct += scoreA * scoreB
		}
	}
	for _, scoreB := range b {
		normB += scoreB * scoreB
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
