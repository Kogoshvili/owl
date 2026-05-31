package intelligence

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"owl/internal/cluster"
	"owl/internal/embedding"
	"owl/internal/store"
	"owl/internal/vector"
	"sort"
	"strings"
)

type EmbeddingsStrategy struct {
	analyzer   *Analyzer
	store      *store.Store
	embeddings *embedding.Client
}

func NewEmbeddingsStrategy(analyzer *Analyzer, store *store.Store, embedClient *embedding.Client) *EmbeddingsStrategy {
	return &EmbeddingsStrategy{
		analyzer:   analyzer,
		store:      store,
		embeddings: embedClient,
	}
}

func (s *EmbeddingsStrategy) SuggestFoldersWithCorpus(ctx context.Context, fileIDs []int64, corpus *Corpus) ([]FolderSuggestion, error) {
	slog.Info("strategy[embeddings]: suggesting folders", "files", len(fileIDs))

	embeddings, err := s.computeEmbeddings(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	points := make([]cluster.Point, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		if emb, ok := embeddings[fileID]; ok {
			points = append(points, cluster.Point{ID: fileID, Vec: emb})
		}
	}

	clusters := cluster.DBSCAN(points, 0.4, MinFilesForFolder)

	{
		compSizes := map[int]int{}
		for _, cl := range clusters {
			compSizes[len(cl.Points)]++
		}
		slog.Info("strategy[embeddings]: dbscan clusters", "clusters", len(clusters), "points", len(points), "size_distribution", fmt.Sprintf("%v", compSizes))
	}

	fileNames, err := s.analyzer.GetFileNames(fileIDs)
	if err != nil {
		return nil, err
	}

	corpusKeywords, _ := s.analyzer.BuildCorpusTFIDFWithFallback(fileIDs)

	suggestions := make([]FolderSuggestion, 0)
	for _, cl := range clusters {
		if len(cl.Points) < MinFilesForFolder {
			continue
		}

		clusterFileIDs := make([]int64, len(cl.Points))
		for i, p := range cl.Points {
			clusterFileIDs[i] = p.ID
		}

		// Use TF-IDF top terms for naming (LLM refinement happens only when user clicks "Refine")
		name := ""
		description := ""
		terms := topTerms(corpusKeywords, clusterFileIDs, 3)
		if len(terms) > 0 {
			name = fmt.Sprintf("%s files", terms[0])
		} else {
			name = "group"
		}
		if description == "" {
			description = fmt.Sprintf("Auto-generated from %d related files (embeddings)", len(clusterFileIDs))
		}

		avgSim := s.avgIntraClusterSimilarity(cl.Points, embeddings)
		preview := getFilePreview(clusterFileIDs, fileNames)

		suggestions = append(suggestions, FolderSuggestion{
			Name:        name,
			Description: description,
			FileIDs:     clusterFileIDs,
			Score:       float64(avgSim),
			Preview:     preview,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score != suggestions[j].Score {
			return suggestions[i].Score > suggestions[j].Score
		}
		return len(suggestions[i].FileIDs) > len(suggestions[j].FileIDs)
	})

	slog.Info("strategy[embeddings]: folder suggestions", "count", len(suggestions))
	return suggestions, nil
}

func (s *EmbeddingsStrategy) computeEmbeddings(ctx context.Context, fileIDs []int64) (map[int64][]float32, error) {
	result := make(map[int64][]float32)

	cached, err := s.loadCachedEmbeddings(fileIDs)
	if err != nil {
		slog.Warn("strategy[embeddings]: failed to load cache", "error", err)
		cached = make(map[int64][]float32)
	}

	remaining := []int64{}
	for _, id := range fileIDs {
		if _, ok := cached[id]; ok {
			result[id] = cached[id]
		} else {
			remaining = append(remaining, id)
		}
	}

	slog.Info("strategy[embeddings]: embedding status", "total", len(fileIDs), "cached", len(cached), "to_compute", len(remaining))

	if len(remaining) == 0 {
		return result, nil
	}

	batchSize := 20
	for i := 0; i < len(remaining); i += batchSize {
		end := i + batchSize
		if end > len(remaining) {
			end = len(remaining)
		}
		batch := remaining[i:end]

		texts := make([]string, len(batch))
		for j, id := range batch {
			texts[j] = s.buildEmbeddingText(id)
		}

		embeddings, err := s.embeddings.Embed(ctx, texts)
		if err != nil {
			slog.Error("strategy[embeddings]: batch failed", "batch", i/batchSize, "error", err)
			continue
		}

		for j, id := range batch {
			if j < len(embeddings) {
				result[id] = embeddings[j]
				go s.cacheEmbedding(id, embeddings[j])
			}
		}

		if (i+batchSize)%100 == 0 || end == len(remaining) {
			slog.Info("strategy[embeddings]: progress", "computed", end, "total", len(remaining))
		}
	}

	return result, nil
}

func (s *EmbeddingsStrategy) buildEmbeddingText(fileID int64) string {
	file, err := s.store.GetFile(fileID)
	if err != nil || file == nil {
		return ""
	}

	var parts []string
	parts = append(parts, file.Name)
	parts = append(parts, file.Extension)
	parts = append(parts, file.ParentDir)

	content, err := s.store.GetFileContent(fileID)
	if err == nil && content != "" {
		if len(content) > 2000 {
			content = content[:2000]
		}
		parts = append(parts, content)
	}

	return strings.Join(parts, " ")
}

func (s *EmbeddingsStrategy) avgIntraClusterSimilarity(points []cluster.Point, embeddings map[int64][]float32) float32 {
	if len(points) < 2 {
		return 0
	}

	var totalSim float32
	var count int

	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			a, okA := embeddings[points[i].ID]
			b, okB := embeddings[points[j].ID]
			if okA && okB {
				totalSim += vector.CosineSimilarity(a, b)
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}
	return totalSim / float32(count)
}

func (s *EmbeddingsStrategy) loadCachedEmbeddings(fileIDs []int64) (map[int64][]float32, error) {
	result := make(map[int64][]float32)
	model := s.embeddings.Model()

	for _, id := range fileIDs {
		var blob []byte
		err := s.store.Db().QueryRow(
			`SELECT embedding FROM file_embeddings WHERE file_id = ? AND model = ?`,
			id, model,
		).Scan(&blob)
		if err != nil {
			continue
		}
		emb := blobToFloat32(blob)
		if len(emb) > 0 {
			result[id] = emb
		}
	}

	return result, nil
}

func (s *EmbeddingsStrategy) cacheEmbedding(fileID int64, embedding []float32) {
	blob := float32ToBlob(embedding)
	model := s.embeddings.Model()

	_, err := s.store.Db().Exec(
		`INSERT OR REPLACE INTO file_embeddings (file_id, model, embedding) VALUES (?, ?, ?)`,
		fileID, model, blob,
	)
	if err != nil {
		slog.Warn("strategy[embeddings]: failed to cache embedding", "file_id", fileID, "error", err)
	}
}

func float32ToBlob(v []float32) []byte {
	blob := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(f))
	}
	return blob
}

func blobToFloat32(blob []byte) []float32 {
	if len(blob)%4 != 0 {
		return nil
	}
	n := len(blob) / 4
	v := make([]float32, n)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return v
}
