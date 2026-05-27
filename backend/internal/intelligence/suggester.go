package intelligence

import (
	"fmt"
	"owl/internal/store"
	"sort"
)

// MinFilesForFolder is the minimum number of files a virtual folder must contain
// to be suggested. Matches minTagFileCount for consistency.
// TODO: Make configurable via settings in a future release.
const MinFilesForFolder = 3

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
}

func NewSuggester(analyzer *Analyzer, store *store.Store) *Suggester {
	return &Suggester{
		analyzer: analyzer,
		store:    store,
	}
}

func (s *Suggester) SuggestVirtualFolders(minFiles int, minSimilarity float64) ([]Suggestion, error) {
	if minFiles <= 0 {
		minFiles = MinFilesForFolder
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.3
	}

	fileIDs, err := s.analyzer.GetProcessedFiles(500)
	if err != nil {
		return nil, err
	}

	if len(fileIDs) < minFiles {
		return []Suggestion{}, nil
	}

	corpusKeywords, err := s.analyzer.BuildCorpusTFIDF(fileIDs)
	if err != nil {
		return nil, err
	}

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
	}

	clusters := s.clusterFiles(fileIDs, similarityMatrix, minSimilarity, minFiles)

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	suggestions := make([]Suggestion, 0)
	for _, cluster := range clusters {
		if len(cluster) < minFiles {
			continue
		}

		name, err := s.generateFolderName(cluster, corpusKeywords)
		if err != nil {
			continue
		}

		preview := s.getFilePreview(cluster, fileNames)

		score := s.calculateClusterScore(cluster, similarityMatrix)

		suggestions = append(suggestions, Suggestion{
			Name:        name,
			Description: fmt.Sprintf("Auto-generated from %d related files", len(cluster)),
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

	return suggestions, nil
}

func (s *Suggester) clusterFiles(fileIDs []int64, similarityMatrix map[[2]int64]float64, minSim float64, minFiles int) [][]int64 {
	clusters := [][]int64{}
	assigned := make(map[int64]bool)

	for _, id := range fileIDs {
		if assigned[id] {
			continue
		}

		cluster := []int64{id}
		assigned[id] = true

		changed := true
		for changed {
			changed = false
			for _, clusterID := range cluster {
				for _, fileID := range fileIDs {
					if assigned[fileID] {
						continue
					}

					pair := [2]int64{clusterID, fileID}
					if clusterID > fileID {
						pair = [2]int64{fileID, clusterID}
					}

					if sim, exists := similarityMatrix[pair]; exists && sim >= minSim {
						cluster = append(cluster, fileID)
						assigned[fileID] = true
						changed = true
					}
				}
			}
		}

		if len(cluster) >= minFiles {
			clusters = append(clusters, cluster)
		}
	}

	return clusters
}

func (s *Suggester) generateFolderName(fileIDs []int64, corpusKeywords map[int64][]Keyword) (string, error) {
	termScores := make(map[string]float64)

	for _, fileID := range fileIDs {
		keywords := corpusKeywords[fileID]
		for _, kw := range keywords {
			termScores[kw.Term] += kw.Score
		}
	}

	type termScore struct {
		term  string
		score float64
	}

	terms := make([]termScore, 0, len(termScores))
	for term, score := range termScores {
		terms = append(terms, termScore{term, score})
	}

	sort.Slice(terms, func(i, j int) bool {
		return terms[i].score > terms[j].score
	})

	nameTerms := []string{}
	for i := 0; i < len(terms) && i < 3; i++ {
		if !IsStopword(terms[i].term) && !IsNumeric(terms[i].term) {
			nameTerms = append(nameTerms, terms[i].term)
		}
	}

	if len(nameTerms) == 0 {
		return "group", nil
	}

	return fmt.Sprintf("%s files", nameTerms[0]), nil
}

func (s *Suggester) getFilePreview(fileIDs []int64, fileNames map[int64]string) []string {
	preview := make([]string, 0, 5)
	for i, id := range fileIDs {
		if i >= 5 {
			break
		}
		if name, exists := fileNames[id]; exists {
			preview = append(preview, name)
		}
	}
	return preview
}

func (s *Suggester) calculateClusterScore(fileIDs []int64, similarityMatrix map[[2]int64]float64) float64 {
	totalScore := 0.0
	count := 0

	for i, id1 := range fileIDs {
		for _, id2 := range fileIDs[i+1:] {
			pair := [2]int64{id1, id2}
			if id1 > id2 {
				pair = [2]int64{id2, id1}
			}

			if sim, exists := similarityMatrix[pair]; exists {
				totalScore += sim
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return totalScore / float64(count)
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

	return folder, nil
}