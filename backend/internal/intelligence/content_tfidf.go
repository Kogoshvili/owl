package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
	"sort"
	"time"
)

type ContentTFIDFStrategy struct {
	analyzer *Analyzer
	store    *store.Store
	llm      *llm.Client
}

func NewContentTFIDFStrategy(analyzer *Analyzer, store *store.Store, llmClient *llm.Client) *ContentTFIDFStrategy {
	return &ContentTFIDFStrategy{
		analyzer: analyzer,
		store:    store,
		llm:      llmClient,
	}
}

func (s *ContentTFIDFStrategy) ID() StrategyID       { return StrategyContentTFIDF }
func (s *ContentTFIDFStrategy) DisplayName() string   { return "Content TF-IDF" }
func (s *ContentTFIDFStrategy) Description() string   { return "Tags and folders from TF-IDF analysis of extracted file content. Much richer signal than path-based analysis." }
func (s *ContentTFIDFStrategy) Available() bool       { return true }
func (s *ContentTFIDFStrategy) SpeedHint() string     { return "~30s for 12K files" }

func (s *ContentTFIDFStrategy) SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error) {
	return s.SuggestFoldersWithCorpus(ctx, fileIDs, nil)
}

func (s *ContentTFIDFStrategy) SuggestFoldersWithCorpus(ctx context.Context, fileIDs []int64, corpus *Corpus) ([]FolderSuggestion, error) {
	minFiles := MinFilesForFolder
	minSimilarity := 0.45

	slog.Info("strategy[content_tfidf]: suggesting folders", "files", len(fileIDs))

	if len(fileIDs) < minFiles {
		return []FolderSuggestion{}, nil
	}

	slog.Info("strategy[content_tfidf]: building corpus", "files", len(fileIDs))
	start := time.Now()
	corpusKeywords, err := s.analyzer.BuildCorpusTFIDFWithFallback(fileIDs)
	if err != nil {
		return nil, err
	}
	slog.Info("strategy[content_tfidf]: corpus built", "elapsed", time.Since(start).String())

	fileKeywords := make(map[int64]map[string]float64)
	if corpus != nil {
		// Use pre-built corpus
		for _, id := range fileIDs {
			if km, ok := corpus.KeywordMap[id]; ok {
				fileKeywords[id] = km
			}
		}
		slog.Info("strategy[content_tfidf]: using pre-built corpus", "files", len(fileIDs))
	} else {
		// Build corpus from scratch (legacy path)
		slog.Info("strategy[content_tfidf]: building corpus from scratch", "files", len(fileIDs))
		for _, id := range fileIDs {
			kwMap := make(map[string]float64)
			for _, kw := range corpusKeywords[id] {
				kwMap[kw.Term] = kw.Score
			}
			fileKeywords[id] = kwMap
		}
	}

	slog.Info("strategy[content_tfidf]: computing similarity matrix", "files", len(fileIDs))
	start = time.Now()
	similarityMatrix := make(map[[2]int64]float64)
	for i, id1 := range fileIDs {
		for _, id2 := range fileIDs[i+1:] {
			sim := cosineSimilarityMaps(fileKeywords[id1], fileKeywords[id2])
			if sim >= minSimilarity {
				if id1 < id2 {
					similarityMatrix[[2]int64{id1, id2}] = sim
				} else {
					similarityMatrix[[2]int64{id2, id1}] = sim
				}
			}
		}
		if (i+1)%50 == 0 || i == len(fileIDs)-1 {
			slog.Info("strategy[content_tfidf]: similarity progress", "files", i+1, "total", len(fileIDs), "similar_pairs", len(similarityMatrix))
		}
	}
	slog.Info("strategy[content_tfidf]: similarity complete", "elapsed", time.Since(start).String(), "similar_pairs", len(similarityMatrix))

	emptyKW := 0
	minKW, maxKW, totalKW := 0, 0, 0
	for _, id := range fileIDs {
		n := len(fileKeywords[id])
		totalKW += n
		if n == 0 {
			emptyKW++
		}
		if minKW == 0 || n < minKW {
			minKW = n
		}
		if n > maxKW {
			maxKW = n
		}
	}
	slog.Info("strategy[content_tfidf]: keyword stats", "files", len(fileIDs), "empty", emptyKW, "min_keywords", minKW, "max_keywords", maxKW, "avg_keywords", float64(totalKW)/float64(len(fileIDs)))

	slog.Info("strategy[content_tfidf]: clustering", "files", len(fileIDs))
	start = time.Now()
	clusters := clusterFiles(fileIDs, similarityMatrix, minSimilarity, minFiles)
	slog.Info("strategy[content_tfidf]: clustering complete", "clusters", len(clusters), "elapsed", time.Since(start).String())

	if len(clusters) == 0 {
		// Debug: find connected components to understand why
		assigned := make(map[int64]bool)
		compSizes := map[int]int{}
		for _, id := range fileIDs {
			if assigned[id] {
				continue
			}
			comp := []int64{id}
			assigned[id] = true
			changed := true
			for changed {
				changed = false
				for _, cid := range comp {
					for _, fid := range fileIDs {
						if assigned[fid] {
							continue
						}
						pair := [2]int64{cid, fid}
						if cid > fid {
							pair = [2]int64{fid, cid}
						}
						if _, exists := similarityMatrix[pair]; exists {
							comp = append(comp, fid)
							assigned[fid] = true
							changed = true
						}
					}
				}
			}
			compSizes[len(comp)]++
		}
		slog.Info("strategy[content_tfidf]: component size distribution (0 clusters)", "distribution", fmt.Sprintf("%v", compSizes), "total_components", len(compSizes))
	}

	slog.Info("strategy[content_tfidf]: refining and generating suggestions", "clusters", len(clusters))

	// Always sub-cluster large clusters (regardless of LLM availability)
	subClustered := [][]int64{}
	for _, cluster := range clusters {
		if len(cluster) > maxFilesForLLM {
			subs := clusterFiles(cluster, similarityMatrix, minSimilarity+subClusterThresholdBoost, minFiles)
			subClustered = append(subClustered, subs...)
		} else {
			subClustered = append(subClustered, cluster)
		}
	}
	clusters = subClustered

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	suggestions := make([]FolderSuggestion, 0)
	for _, cluster := range clusters {
		if len(cluster) < minFiles {
			continue
		}

		// Use TF-IDF top terms for naming (LLM refinement happens only when user clicks "Refine")
		name := ""
		description := ""
		terms := topTerms(corpusKeywords, cluster, 3)
		if len(terms) > 0 {
			name = fmt.Sprintf("%s files", terms[0])
		} else {
			name = "group"
		}
		if description == "" {
			description = fmt.Sprintf("Auto-generated from %d related files", len(cluster))
		}

		preview := getFilePreview(cluster, fileNames)
		score := calculateClusterScore(cluster, similarityMatrix)

		suggestions = append(suggestions, FolderSuggestion{
			Name:        name,
			Description: description,
			FileIDs:     cluster,
			Score:       score,
			Preview:     preview,
		})
	}

	slog.Info("strategy[content_tfidf]: folder suggestions complete", "suggestions", len(suggestions))

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score != suggestions[j].Score {
			return suggestions[i].Score > suggestions[j].Score
		}
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[content_tfidf]: folder suggestions", "count", len(suggestions))
	return suggestions, nil
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
