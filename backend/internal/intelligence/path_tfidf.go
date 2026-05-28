package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PathTFIDFStrategy struct {
	analyzer *Analyzer
	store    *store.Store
	llm      *llm.Client
}

func NewPathTFIDFStrategy(analyzer *Analyzer, store *store.Store, llmClient *llm.Client) *PathTFIDFStrategy {
	return &PathTFIDFStrategy{
		analyzer: analyzer,
		store:    store,
		llm:      llmClient,
	}
}

func (s *PathTFIDFStrategy) ID() StrategyID       { return StrategyPathTFIDF }
func (s *PathTFIDFStrategy) DisplayName() string   { return "Path TF-IDF" }
func (s *PathTFIDFStrategy) Description() string   { return "Tags from file extension, path segments, and filename keywords. Folders from TF-IDF cosine similarity clustering." }
func (s *PathTFIDFStrategy) Available() bool       { return true }
func (s *PathTFIDFStrategy) SpeedHint() string     { return "~5s for 12K files" }

func (s *PathTFIDFStrategy) SuggestTags(ctx context.Context, fileIDs []int64) ([]TagSuggestion, error) {
	slog.Info("strategy[path_tfidf]: suggesting tags", "files", len(fileIDs))

	fileContents := make(map[int64]string)
	fileData := make(map[int64]*store.File)
	tagCandidates := make(map[string][]int64)

	for i, fileID := range fileIDs {
		file, err := s.store.GetFile(fileID)
		if err != nil || file == nil {
			continue
		}

		content := ""
		if file.ProcessingStatus == "processed" {
			keywords, err := s.analyzer.GetFileKeywords(fileID, 10)
			if err == nil && len(keywords) > 0 {
				terms := make([]string, 0, len(keywords))
				for _, kw := range keywords {
					terms = append(terms, kw.Term)
				}
				content = strings.Join(terms, " ")
			}
		}

		fileData[fileID] = file
		fileContents[fileID] = content

		tagNames := s.collectTagsFromFile(file, content)
		for _, tagName := range tagNames {
			tagCandidates[tagName] = append(tagCandidates[tagName], fileID)
		}

		if (i+1)%100 == 0 || i == len(fileIDs)-1 {
			slog.Info("strategy[path_tfidf]: collecting tag candidates", "progress", i+1, "total", len(fileIDs))
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

	slog.Info("strategy[path_tfidf]: tag suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (s *PathTFIDFStrategy) SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error) {
	minFiles := MinFilesForFolder
	minSimilarity := 0.45

	slog.Info("strategy[path_tfidf]: suggesting folders", "files", len(fileIDs))

	if len(fileIDs) < minFiles {
		return []FolderSuggestion{}, nil
	}

	corpusKeywords, err := s.analyzer.BuildCorpusTFIDF(fileIDs)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	similarityMatrix := make(map[[2]int64]float64)
	for i, id1 := range fileIDs {
		for _, id2 := range fileIDs[i+1:] {
			sim, err := s.analyzer.CalculateFileSimilarity(id1, id2)
			if err != nil {
				continue
			}
			if sim >= minSimilarity {
				similarityMatrix[[2]int64{id1, id2}] = sim
			}
		}
		if (i+1)%50 == 0 || i == len(fileIDs)-1 {
			slog.Info("strategy[path_tfidf]: similarity progress", "files", i+1, "total", len(fileIDs), "similar_pairs", len(similarityMatrix))
		}
	}
	slog.Info("strategy[path_tfidf]: similarity complete", "elapsed", time.Since(start).String(), "similar_pairs", len(similarityMatrix))

	clusters := clusterFiles(fileIDs, similarityMatrix, minSimilarity, minFiles)

	llmAvailable := s.llm != nil && s.llm.IsAvailable(ctx)
	if llmAvailable {
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
	}

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	suggestions := make([]FolderSuggestion, 0)
	for _, cluster := range clusters {
		if len(cluster) < minFiles {
			continue
		}

		name := ""
		description := ""

		if llmAvailable {
			cf, err := s.buildClusterFiles(cluster, corpusKeywords, fileNames)
			if err == nil {
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
						if len(filtered) < minFiles {
							continue
						}
						cluster = filtered
					}
				}
			}
		}

		if name == "" {
			name = s.generateFolderName(cluster, corpusKeywords)
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

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score != suggestions[j].Score {
			return suggestions[i].Score > suggestions[j].Score
		}
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[path_tfidf]: folder suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (s *PathTFIDFStrategy) collectTagsFromFile(file *store.File, content string) []string {
	var tagNames []string
	seen := make(map[string]bool)

	extensionTag := getExtensionTag(file.Extension)
	if extensionTag != "" && !seen[extensionTag] {
		tagNames = append(tagNames, extensionTag)
		seen[extensionTag] = true
	}

	dir := filepath.Dir(file.Path)
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 3 || part == "." || part == ".." {
			continue
		}
		term := NormalizeTerm(part)
		if !IsStopword(term) && !IsNumeric(term) && !seen[term] {
			tagNames = append(tagNames, term)
			seen[term] = true
		}
	}

	if content != "" {
		keywords := ExtractKeywordsFromContent(content, 10)
		for i, kw := range keywords {
			if i >= 5 {
				break
			}
			if !IsStopword(kw.Term) && !IsNumeric(kw.Term) && !seen[kw.Term] {
				tagNames = append(tagNames, kw.Term)
				seen[kw.Term] = true
			}
		}
	}

	return tagNames
}

func (s *PathTFIDFStrategy) buildClusterFiles(fileIDs []int64, corpusKeywords map[int64][]Keyword, fileNames map[int64]string) ([]llm.ClusterFile, error) {
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
	return result, nil
}

func (s *PathTFIDFStrategy) generateFolderName(fileIDs []int64, corpusKeywords map[int64][]Keyword) string {
	termScores := make(map[string]float64)
	for _, fileID := range fileIDs {
		for _, kw := range corpusKeywords[fileID] {
			termScores[kw.Term] += kw.Score
		}
	}

	type ts struct {
		term  string
		score float64
	}
	terms := make([]ts, 0, len(termScores))
	for term, score := range termScores {
		terms = append(terms, ts{term, score})
	}
	sort.Slice(terms, func(i, j int) bool { return terms[i].score > terms[j].score })

	for i := 0; i < len(terms) && i < 3; i++ {
		if !IsStopword(terms[i].term) && !IsNumeric(terms[i].term) {
			return fmt.Sprintf("%s files", terms[i].term)
		}
	}
	return "group"
}

func getExtensionTag(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	return extensionTags[ext]
}
