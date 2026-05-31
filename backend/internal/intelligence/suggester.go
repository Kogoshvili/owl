package intelligence

import (
	"owl/internal/llm"
	"owl/internal/store"
)

const (
	MinFilesForFolder        = 3
	maxFilesForLLM           = 50
	subClusterThresholdBoost = 0.15
)

type Suggester struct {
	analyzer *Analyzer
	store    *store.Store
}

func NewSuggester(analyzer *Analyzer, store *store.Store) *Suggester {
	return &Suggester{
		analyzer: analyzer,
		store:    store,
	}
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
