package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
	"sort"
	"strings"
)

type LLMKeywordStrategy struct {
	analyzer *Analyzer
	store    *store.Store
	llm      *llm.Client
}

func NewLLMKeywordStrategy(analyzer *Analyzer, store *store.Store, llmClient *llm.Client) *LLMKeywordStrategy {
	return &LLMKeywordStrategy{
		analyzer: analyzer,
		store:    store,
		llm:      llmClient,
	}
}

func (s *LLMKeywordStrategy) ID() StrategyID       { return StrategyLLMKeywords }
func (s *LLMKeywordStrategy) DisplayName() string   { return "LLM Keywords" }
func (s *LLMKeywordStrategy) Description() string   { return "Sends file content to LLM in batches to extract semantic keywords. Groups files by keyword overlap using Jaccard similarity." }
func (s *LLMKeywordStrategy) Available() bool {
	return s.llm != nil
}
func (s *LLMKeywordStrategy) SpeedHint() string     { return "~2-6h for 12K files" }

func (s *LLMKeywordStrategy) SuggestTags(ctx context.Context, fileIDs []int64) ([]TagSuggestion, error) {
	slog.Info("strategy[llm_keywords]: suggesting tags", "files", len(fileIDs))

	fileKeywords, err := s.extractAllKeywords(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	tagCandidates := make(map[string][]int64)
	for fileID, keywords := range fileKeywords {
		for _, kw := range keywords {
			kw = strings.ToLower(strings.TrimSpace(kw))
			if len(kw) < 3 || IsStopword(kw) || IsNumeric(kw) {
				continue
			}
			tagCandidates[kw] = append(tagCandidates[kw], fileID)
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

	slog.Info("strategy[llm_keywords]: tag suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (s *LLMKeywordStrategy) SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error) {
	slog.Info("strategy[llm_keywords]: suggesting folders", "files", len(fileIDs))

	fileKeywords, err := s.extractAllKeywords(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	keywordSets := make(map[int64]map[string]bool)
	for fileID, keywords := range fileKeywords {
		set := make(map[string]bool)
		for _, kw := range keywords {
			kw = strings.ToLower(strings.TrimSpace(kw))
			if len(kw) >= 3 && !IsStopword(kw) && !IsNumeric(kw) {
				set[kw] = true
			}
		}
		keywordSets[fileID] = set
	}

	minJaccard := 0.3
	similarityMatrix := make(map[[2]int64]float64)
	for i, id1 := range fileIDs {
		for _, id2 := range fileIDs[i+1:] {
			jaccard := jaccardSimilarity(keywordSets[id1], keywordSets[id2])
			if jaccard >= minJaccard {
				similarityMatrix[[2]int64{id1, id2}] = jaccard
			}
		}
	}

	clusters := clusterFiles(fileIDs, similarityMatrix, minJaccard, MinFilesForFolder)

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	llmAvailable := s.llm != nil && s.llm.IsAvailable(ctx)
	corpusKeywords, _ := s.analyzer.BuildCorpusTFIDF(fileIDs)

	suggestions := make([]FolderSuggestion, 0)
	for _, cluster := range clusters {
		if len(cluster) < MinFilesForFolder {
			continue
		}

		name := ""
		description := ""

		if llmAvailable {
			cf := s.buildClusterFiles(cluster, corpusKeywords, fileNames)
			refinement, err := s.llm.RefineCluster(ctx, cf, cluster, name)
			if err == nil && refinement != nil {
				if !refinement.Related {
					continue
				}
				name = refinement.Name
				description = refinement.Description
				if len(refinement.RemovedIDs) > 0 {
					removed := make(map[int64]bool)
					for _, id := range refinement.RemovedIDs {
						removed[id] = true
					}
					filtered := []int64{}
					for _, id := range cluster {
						if !removed[id] {
							filtered = append(filtered, id)
						}
					}
					if len(filtered) < MinFilesForFolder {
						continue
					}
					cluster = filtered
				}
			}
		}

		if name == "" {
			terms := topTerms(corpusKeywords, cluster, 3)
			if len(terms) > 0 {
				name = fmt.Sprintf("%s files", terms[0])
			} else {
				name = "group"
			}
		}
		if description == "" {
			description = fmt.Sprintf("Auto-generated from %d related files (LLM keywords)", len(cluster))
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

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score != suggestions[j].Score {
			return suggestions[i].Score > suggestions[j].Score
		}
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[llm_keywords]: folder suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (s *LLMKeywordStrategy) extractAllKeywords(ctx context.Context, fileIDs []int64) (map[int64][]string, error) {
	result := make(map[int64][]string)
	batchSize := 10

	for i := 0; i < len(fileIDs); i += batchSize {
		end := i + batchSize
		if end > len(fileIDs) {
			end = len(fileIDs)
		}
		batch := fileIDs[i:end]

		files := make([]struct{ ID int64; Name string; Content string }, len(batch))
		for j, id := range batch {
			file, err := s.store.GetFile(id)
			if err != nil || file == nil {
				continue
			}
			content, _ := s.store.GetFileContent(id)
			if len(content) > 500 {
				content = content[:500]
			}
			files[j] = struct{ ID int64; Name string; Content string }{
				ID:      id,
				Name:    file.Name,
				Content: content,
			}
		}

		extractions, err := s.llm.ExtractKeywords(ctx, files)
		if err != nil {
			slog.Warn("strategy[llm_keywords]: batch failed", "batch", i/batchSize, "error", err)
			continue
		}

		for _, ext := range extractions {
			result[ext.FileID] = ext.Keywords
		}

		if (i+batchSize)%50 == 0 || end == len(fileIDs) {
			slog.Info("strategy[llm_keywords]: progress", "processed", end, "total", len(fileIDs))
		}
	}

	return result, nil
}

func (s *LLMKeywordStrategy) buildClusterFiles(fileIDs []int64, corpusKeywords map[int64][]Keyword, fileNames map[int64]string) []llm.ClusterFile {
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

func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
