package intelligence

type StrategyID string

const (
	StrategyContentTFIDF StrategyID = "content_tfidf"
	StrategyEmbeddings   StrategyID = "embeddings"
)

type FolderSuggestion struct {
	Name        string
	Description string
	FileIDs     []int64
	Score       float64
	Preview     []string
}
