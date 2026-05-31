package intelligence

type FolderCoherence struct {
	Path          string  `json:"path"`
	FileCount     int     `json:"file_count"`
	AvgSimilarity float64 `json:"avg_similarity"`
	IsCoherent    bool    `json:"is_coherent"`
	OutlierFiles  []OutlierFile `json:"outlier_files,omitempty"`
}

type OutlierFile struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Similarity float64 `json:"avg_similarity_to_others"`
}

const coherenceThreshold = 0.15

func AnalyzeFolderCoherenceWithCorpus(corpus *Corpus, folderFileIDs []int64, fileNames map[int64]string, folderPath string) (*FolderCoherence, error) {
	if len(folderFileIDs) < 2 {
		return &FolderCoherence{
			Path:          folderPath,
			FileCount:     len(folderFileIDs),
			AvgSimilarity: 1.0,
			IsCoherent:    true,
		}, nil
	}

	keywordMap := make(map[int64]map[string]float64)
	for _, id := range folderFileIDs {
		if km, ok := corpus.KeywordMap[id]; ok {
			keywordMap[id] = km
		}
	}

	var totalSim float64
	var pairCount int

	fileAvgSim := make(map[int64]float64)
	filePairCount := make(map[int64]int)

	for i, id1 := range folderFileIDs {
		for _, id2 := range folderFileIDs[i+1:] {
			sim := cosineSimilarityMaps(keywordMap[id1], keywordMap[id2])
			totalSim += sim
			pairCount++

			fileAvgSim[id1] += sim
			filePairCount[id1]++
			fileAvgSim[id2] += sim
			filePairCount[id2]++
		}
	}

	var avgSim float64
	if pairCount > 0 {
		avgSim = totalSim / float64(pairCount)
	}

	for id := range fileAvgSim {
		if filePairCount[id] > 0 {
			fileAvgSim[id] /= float64(filePairCount[id])
		}
	}

	isCoherent := avgSim >= coherenceThreshold

	var outliers []OutlierFile
	if !isCoherent {
		for _, fileID := range folderFileIDs {
			avg := fileAvgSim[fileID]
			if avg < coherenceThreshold {
				name := ""
				if n, ok := fileNames[fileID]; ok {
					name = n
				}
				outliers = append(outliers, OutlierFile{
					ID:         fileID,
					Name:       name,
					Similarity: avg,
				})
			}
		}
	}

	return &FolderCoherence{
		Path:          folderPath,
		FileCount:     len(folderFileIDs),
		AvgSimilarity: avgSim,
		IsCoherent:    isCoherent,
		OutlierFiles:  outliers,
	}, nil
}


