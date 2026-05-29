package intelligence

import (
	"sort"
	"strings"
)

func clusterFiles(fileIDs []int64, similarityMatrix map[[2]int64]float64, minSim float64, minFiles int) [][]int64 {
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

func getFilePreview(fileIDs []int64, fileNames map[int64]string) []string {
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

func calculateClusterScore(fileIDs []int64, similarityMatrix map[[2]int64]float64) float64 {
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

func topTerms(keywords map[int64][]Keyword, fileIDs []int64, n int) []string {
	termScores := make(map[string]float64)
	for _, fileID := range fileIDs {
		for _, kw := range keywords[fileID] {
			termScores[kw.Term] += kw.Score
		}
	}

	type ts struct {
		term  string
		score float64
	}
	terms := make([]ts, 0, len(termScores))
	for term, score := range termScores {
		terms = append(terms, ts{term, score})
	}
	sort.Slice(terms, func(i, j int) bool { return terms[i].score > terms[j].score })

	result := make([]string, 0, n)
	for i := 0; i < len(terms) && len(result) < n; i++ {
		if !IsStopword(terms[i].term) && !IsNumeric(terms[i].term) {
			result = append(result, terms[i].term)
		}
	}
	return result
}

var extensionTags = map[string]string{
	"pdf":  "document",
	"docx": "document",
	"doc":  "document",
	"xlsx": "spreadsheet",
	"pptx": "presentation",
	"txt":  "text",
	"md":   "markdown",
	"rtf":  "document",
	"odt":  "document",
	"log":  "text",
	"jpg":  "image",
	"jpeg": "image",
	"png":  "image",
	"gif":  "image",
	"svg":  "image",
	"bmp":  "image",
	"webp": "image",
	"tiff": "image",
	"mp3":  "audio",
	"wav":  "audio",
	"flac": "audio",
	"ogg":  "audio",
	"aac":  "audio",
	"m4a":  "audio",
	"wma":  "audio",
	"mp4":  "video",
	"avi":  "video",
	"mkv":  "video",
	"mov":  "video",
	"wmv":  "video",
	"flv":  "video",
	"webm": "video",
	"zip":  "archive",
	"tar":  "archive",
	"gz":   "archive",
	"rar":  "archive",
	"7z":   "archive",
	"json": "data",
	"xml":  "data",
	"csv":  "data",
	"yaml": "data",
	"yml":  "data",
	"toml": "data",
	"ini":  "config",
	"cfg":  "config",
	"conf": "config",
	"env":  "config",
	"py":   "code",
	"js":   "code",
	"ts":   "code",
	"go":   "code",
	"rs":   "code",
	"java": "code",
	"c":    "code",
	"cpp":  "code",
	"h":    "code",
	"hpp":  "code",
	"rb":   "code",
	"php":  "code",
	"sh":   "code",
	"bat":  "code",
	"ps1":  "code",
	"sql":  "data",
	"html": "web",
	"htm":  "web",
	"css":  "web",
	"scss": "web",
}

func getExtensionTag(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	return extensionTags[ext]
}
