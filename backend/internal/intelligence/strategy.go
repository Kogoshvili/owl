package intelligence

import (
	"context"
)

type StrategyID string

const (
	StrategyPathTFIDF    StrategyID = "path_tfidf"
	StrategyContentTFIDF StrategyID = "content_tfidf"
	StrategyEmbeddings   StrategyID = "embeddings"
	StrategyLLMKeywords  StrategyID = "llm_keywords"
	StrategyHybrid       StrategyID = "hybrid"
)

type StrategyInfo struct {
	ID          StrategyID `json:"id"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Available   bool       `json:"available"`
	SpeedHint   string     `json:"speed_hint"`
}

type TagSuggestion struct {
	Name       string
	FileIDs    []int64
	Confidence float64
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

	SuggestTags(ctx context.Context, fileIDs []int64) ([]TagSuggestion, error)
	SuggestFolders(ctx context.Context, fileIDs []int64) ([]FolderSuggestion, error)
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
	return r.strategies[StrategyPathTFIDF]
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
	case StrategyPathTFIDF, StrategyContentTFIDF, StrategyEmbeddings,
		StrategyLLMKeywords, StrategyHybrid:
		return StrategyID(s)
	default:
		return StrategyPathTFIDF
	}
}
