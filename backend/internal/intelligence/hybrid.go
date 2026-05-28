package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"owl/internal/cluster"
	"owl/internal/embedding"
	"owl/internal/llm"
	"owl/internal/store"
	"owl/internal/vector"
	"sort"
	"strings"
)

type HybridStrategy struct {
	analyzer   *Analyzer
	store      *store.Store
	llm        *llm.Client
	embeddings *embedding.Client
}

func NewHybridStrategy(analyzer *Analyzer, store *store.Store, llmClient *llm.Client, embedClient *embedding.Client) *HybridStrategy {
	return &HybridStrategy{
		analyzer:   analyzer,
		store:      store,
		llm:        llmClient,
		embeddings: embedClient,
	}
}

func (s *HybridStrategy) ID() StrategyID       { return StrategyHybrid }
func (s *HybridStrategy) DisplayName() string   { return "Hybrid LLM+Embed" }
func (s *HybridStrategy) Description() string   { return "Best quality: LLM extracts keywords from file content, then keywords are embedded and clustered with DBSCAN. Combines LLM understanding with efficient vector clustering." }
func (s *HybridStrategy) Available() bool {
	return s.llm != nil && s.embeddings != nil
}
func (s *HybridStrategy) SpeedHint() string     { return "~30-60min for 12K files" }

func (s *HybridStrategy) SuggestTags(ctx context.Context, fileIDs []int64) ([]TagSuggestion, error) {
	slog.Info("strategy[hybrid]: suggesting tags", "files", len(fileIDs))

	llmStrategy := NewLLMKeywordStrategy(s.analyzer, s.store, s.llm)
	return llmStrategy.SuggestTags(ctx, fileIDs)
}

func (s *HybridStrategy) SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error) {
	slog.Info("strategy[hybrid]: suggesting folders", "files", len(fileIDs))

	llmStrategy := NewLLMKeywordStrategy(s.analyzer, s.store, s.llm)
	fileKeywords, err := llmStrategy.extractAllKeywords(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	if len(fileKeywords) == 0 {
		return []FolderSuggestion{}, nil
	}

	keywordTexts := make(map[int64]string)
	for fileID, keywords := range fileKeywords {
		keywordTexts[fileID] = strings.Join(keywords, " ")
	}

	var idsToEmbed []int64
	var texts []string
	for _, id := range fileIDs {
		if text, ok := keywordTexts[id]; ok && text != "" {
			idsToEmbed = append(idsToEmbed, id)
			texts = append(texts, text)
		}
	}

	slog.Info("strategy[hybrid]: embedding keywords", "files", len(texts))

	embeddingMap := make(map[int64][]float32)
	batchSize := 20
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		embeddings, err := s.embeddings.Embed(ctx, texts[i:end])
		if err != nil {
			slog.Warn("strategy[hybrid]: embedding batch failed", "batch", i/batchSize, "error", err)
			continue
		}

		for j, emb := range embeddings {
			if i+j < len(idsToEmbed) {
				embeddingMap[idsToEmbed[i+j]] = emb
			}
		}
	}

	if len(embeddingMap) < MinFilesForFolder {
		return []FolderSuggestion{}, nil
	}

	points := make([]cluster.Point, 0, len(embeddingMap))
	for id, emb := range embeddingMap {
		points = append(points, cluster.Point{ID: id, Vec: emb})
	}

	clusters := cluster.DBSCAN(points, 0.4, MinFilesForFolder)

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	corpusKeywords, _ := s.analyzer.BuildCorpusTFIDF(fileIDs)
	llmAvailable := s.llm != nil && s.llm.IsAvailable(ctx)

	suggestions := make([]FolderSuggestion, 0)
	for _, cl := range clusters {
		if len(cl.Points) < MinFilesForFolder {
			continue
		}

		clusterFileIDs := make([]int64, len(cl.Points))
		for i, p := range cl.Points {
			clusterFileIDs[i] = p.ID
		}

		name := ""
		description := ""

		if llmAvailable {
			cf := buildLLMClusterFiles(s.store, clusterFileIDs, corpusKeywords, fileNames)
			refinement, err := s.llm.RefineCluster(ctx, cf, clusterFileIDs, name)
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
					for _, id := range clusterFileIDs {
						if !removed[id] {
							filtered = append(filtered, id)
						}
					}
					if len(filtered) < MinFilesForFolder {
						continue
					}
					clusterFileIDs = filtered
				}
			}
		}

		if name == "" {
			terms := topTerms(corpusKeywords, clusterFileIDs, 3)
			if len(terms) > 0 {
				name = fmt.Sprintf("%s files", terms[0])
			} else {
				name = "group"
			}
		}
		if description == "" {
			description = fmt.Sprintf("Auto-generated from %d related files (hybrid)", len(clusterFileIDs))
		}

		avgSim := avgSimilarity(cl.Points, embeddingMap)
		preview := getFilePreview(clusterFileIDs, fileNames)

		suggestions = append(suggestions, FolderSuggestion{
			Name:        name,
			Description: description,
			FileIDs:     clusterFileIDs,
			Score:       float64(avgSim),
			Preview:     preview,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score != suggestions[j].Score {
			return suggestions[i].Score > suggestions[j].Score
		}
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[hybrid]: folder suggestions", "count", len(suggestions))
	return suggestions, nil
}

func buildLLMClusterFiles(st *store.Store, fileIDs []int64, corpusKeywords map[int64][]Keyword, fileNames map[int64]string) []llm.ClusterFile {
	result := []llm.ClusterFile{}
	for _, fileID := range fileIDs {
		name, ok := fileNames[fileID]
		if !ok {
			continue
		}
		file, err := st.GetFile(fileID)
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

func avgSimilarity(points []cluster.Point, embeddings map[int64][]float32) float32 {
	if len(points) < 2 {
		return 0
	}

	var total float32
	var count int

	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			a, okA := embeddings[points[i].ID]
			b, okB := embeddings[points[j].ID]
			if okA && okB {
				total += vector.CosineSimilarity(a, b)
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}
	return total / float32(count)
}
