package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
	"sort"
	"strings"
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

func (s *ContentTFIDFStrategy) SuggestTags(ctx context.Context, fileIDs []int64) ([]TagSuggestion, error) {
	slog.Info("strategy[content_tfidf]: suggesting tags", "files", len(fileIDs))

	corpusKeywords, err := s.analyzer.BuildCorpusTFIDFWithFallback(fileIDs)
	if err != nil {
		return nil, err
	}

	tagCandidates := make(map[string][]int64)
	for _, fileID := range fileIDs {
		keywords, ok := corpusKeywords[fileID]
		if !ok {
			continue
		}

		file, err := s.store.GetFile(fileID)
		if err != nil || file == nil {
			continue
		}

		seen := make(map[string]bool)

		extTag := getExtensionTag(file.Extension)
		if extTag != "" && !seen[extTag] {
			tagCandidates[extTag] = append(tagCandidates[extTag], fileID)
			seen[extTag] = true
		}

		maxContentTags := 10
		for i, kw := range keywords {
			if i >= maxContentTags {
				break
			}
			if !seen[kw.Term] {
				tagCandidates[kw.Term] = append(tagCandidates[kw.Term], fileID)
				seen[kw.Term] = true
			}
		}
	}

	for name, ids := range tagCandidates {
		if len(ids) < minTagFileCount {
			delete(tagCandidates, name)
		}
	}

	suggestions := make([]TagSuggestion, 0, len(tagCandidates))
	for tagName, ids := range tagCandidates {
		suggestions = append(suggestions, TagSuggestion{
			Name:       tagName,
			FileIDs:    ids,
			Confidence: float64(len(ids)) / float64(len(fileIDs)),
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[content_tfidf]: tag suggestions", "count", len(suggestions))
	return suggestions, nil
}

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
				similarityMatrix[[2]int64{id1, id2}] = sim
			}
		}
		if (i+1)%50 == 0 || i == len(fileIDs)-1 {
			slog.Info("strategy[content_tfidf]: similarity progress", "files", i+1, "total", len(fileIDs), "similar_pairs", len(similarityMatrix))
		}
	}
	slog.Info("strategy[content_tfidf]: similarity complete", "elapsed", time.Since(start).String(), "similar_pairs", len(similarityMatrix))

	slog.Info("strategy[content_tfidf]: clustering", "files", len(fileIDs))
	start = time.Now()
	clusters := clusterFiles(fileIDs, similarityMatrix, minSimilarity, minFiles)
	slog.Info("strategy[content_tfidf]: clustering complete", "clusters", len(clusters), "elapsed", time.Since(start).String())

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

func (s *ContentTFIDFStrategy) buildClusterFiles(fileIDs []int64, corpusKeywords map[int64][]Keyword, fileNames map[int64]string) []llm.ClusterFile {
	result := []llm.ClusterFile{}
	for _, fileID := range fileIDs {
		name, ok := fileNames[fileID]
		if !ok {
			continue
		}
		file, err := s.store.GetFile(fileID)
		if err != nil || file == nil {
			continue
		}
		keywords := []string{}
		if kw, ok := corpusKeywords[fileID]; ok {
			for i, k := range kw {
				if i >= 5 {
					break
				}
				keywords = append(keywords, k.Term)
			}
		}
		result = append(result, llm.ClusterFile{
			ID:        fileID,
			Name:      name,
			Extension: file.Extension,
			ParentDir: file.ParentDir,
			Keywords:  keywords,
		})
	}
	return result
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

func (s *ContentTFIDFStrategy) collectContentTags(file *store.File, keywords []Keyword, seen map[string]bool) []string {
	var tags []string

	extTag := getExtensionTag(file.Extension)
	if extTag != "" && !seen[extTag] {
		tags = append(tags, extTag)
		seen[extTag] = true
	}

	for i, kw := range keywords {
		if i >= 10 {
			break
		}
		if !seen[kw.Term] && !IsStopword(kw.Term) && !IsNumeric(kw.Term) && len(kw.Term) >= 3 {
			tags = append(tags, kw.Term)
			seen[kw.Term] = true
		}
	}

	return tags
}

func joinKeywords(keywords []Keyword, max int) string {
	terms := make([]string, 0, max)
	for i, kw := range keywords {
		if i >= max {
			break
		}
		terms = append(terms, kw.Term)
	}
	return strings.Join(terms, " ")
}
