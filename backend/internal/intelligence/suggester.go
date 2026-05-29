package intelligence

import (
	"context"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
)

const (
	MinFilesForFolder           = 3
	maxFilesForLLM              = 50
	subClusterThresholdBoost    = 0.15
)

type Suggestion struct {
	Name        string
	Description string
	FileIDs     []int64
	Score       float64
	Preview     []string
}

type Suggester struct {
	analyzer *Analyzer
	store    *store.Store
	llm      *llm.Client
	registry *Registry
}

func NewSuggester(analyzer *Analyzer, store *store.Store, llmClient *llm.Client, registry *Registry) *Suggester {
	return &Suggester{
		analyzer: analyzer,
		store:    store,
		llm:      llmClient,
		registry: registry,
	}
}

func (s *Suggester) SuggestVirtualFolders(ctx context.Context, minFiles int, minSimilarity float64, strategyID StrategyID) ([]Suggestion, error) {
	if minFiles <= 0 {
		minFiles = MinFilesForFolder
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.45
	}

	strategy := s.registry.Get(strategyID)
	if strategy == nil {
		strategy = s.registry.Default()
	}

	slog.Info("suggester: starting", "strategy", strategyID, "min_files", minFiles, "min_similarity", minSimilarity)

	fileIDs, err := s.analyzer.GetProcessedFiles(0)
	if err != nil {
		return nil, err
	}

	if len(fileIDs) < minFiles {
		slog.Info("suggester: not enough processed files", "count", len(fileIDs), "min_required", minFiles)
		return []Suggestion{}, nil
	}

	folderSuggestions, err := strategy.SuggestFolders(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	suggestions := make([]Suggestion, 0, len(folderSuggestions))
	for _, fs := range folderSuggestions {
		suggestions = append(suggestions, Suggestion{
			Name:        fs.Name,
			Description: fs.Description,
			FileIDs:     fs.FileIDs,
			Score:       fs.Score,
			Preview:     fs.Preview,
		})
	}

	slog.Info("suggester: complete", "suggestions", len(suggestions))
	return suggestions, nil
}

func (s *Suggester) BuildClusterFiles(fileIDs []int64, corpusKeywords map[int64][]Keyword, fileNames map[int64]string) ([]llm.ClusterFile, error) {
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

func (s *Suggester) CreateFolderFromSuggestion(suggestion Suggestion) (*store.VirtualFolder, error) {
	folder, err := s.store.CreateVirtualFolder(suggestion.Name, suggestion.Description, true)
	if err != nil {
		return nil, err
	}

	err = s.store.AddFilesToFolder(folder.ID, suggestion.FileIDs, "auto")
	if err != nil {
		return nil, err
	}

	slog.Info("suggester: created folder from suggestion", "name", suggestion.Name, "files", len(suggestion.FileIDs))
	return folder, nil
}