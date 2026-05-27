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
	Reason      string
}

type TagRefinementResult struct {
	Meaningful bool
	BetterName string
	Description string
	Reason      string
}