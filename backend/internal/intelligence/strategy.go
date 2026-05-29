package intelligence

import (
	"context"
)

type StrategyID string

const (
	StrategyContentTFIDF StrategyID = "content_tfidf"
	StrategyEmbeddings   StrategyID = "embeddings"
)

type StrategyInfo struct {
	ID          StrategyID `json:"id"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Available   bool       `json:"available"`
	SpeedHint   string     `json:"speed_hint"`
}

type FolderSuggestion struct {
	Name        string
	Description string
	FileIDs     []int64
	Score       float64
	Preview     []string
}

type ProgressFunc func(stage string, current, total int)

type Strategy interface {
	ID() StrategyID
	DisplayName() string
	Description() string
	Available() bool
	SpeedHint() string

	SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error)
	SuggestFoldersWithCorpus(ctx context.Context, fileIDs []int64, corpus *Corpus) ([]FolderSuggestion, error)
}

type Registry struct {
	strategies map[StrategyID]Strategy
	order      []StrategyID
}

func NewRegistry() *Registry {
	return &Registry{
		strategies: make(map[StrategyID]Strategy),
	}
}

func (r *Registry) Register(s Strategy) {
	r.strategies[s.ID()] = s
	r.order = append(r.order, s.ID())
}

func (r *Registry) Get(id StrategyID) Strategy {
	return r.strategies[id]
}

func (r *Registry) Default() Strategy {
	if s := r.strategies[StrategyContentTFIDF]; s != nil {
		return s
	}
	if s := r.strategies[StrategyEmbeddings]; s != nil {
		return s
	}
	return nil
}

func (r *Registry) List() []StrategyInfo {
	result := make([]StrategyInfo, 0, len(r.order))
	for _, id := range r.order {
		s := r.strategies[id]
		result = append(result, StrategyInfo{
			ID:          s.ID(),
			DisplayName: s.DisplayName(),
			Description: s.Description(),
			Available:   s.Available(),
			SpeedHint:   s.SpeedHint(),
		})
	}
	return result
}

func ParseStrategyID(s string) StrategyID {
	switch StrategyID(s) {
	case StrategyContentTFIDF, StrategyEmbeddings:
		return StrategyID(s)
	default:
		return StrategyContentTFIDF
	}
}
