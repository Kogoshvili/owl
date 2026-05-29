package llm

type ClusterFile struct {
	ID        int64
	Name      string
	Extension string
	ParentDir  string
	Keywords   []string
}

type RefinementResult struct {
	Related     bool
	RemovedIDs  []int64
	Name        string
	Description string
}

type KeywordExtraction struct {
	FileID   int64
	Keywords []string
}

type FolderClassification struct {
	Related bool
	Reason  string
}