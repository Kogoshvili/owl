package intelligence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterFiles(t *testing.T) {
	t.Run("no pairs in matrix", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		matrix := map[[2]int64]float64{}
		clusters := clusterFiles(ids, matrix, 0.5, 2)
		require.Empty(t, clusters)
	})

	t.Run("single cluster of 3", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		matrix := map[[2]int64]float64{
			{1, 2}: 0.8,
			{1, 3}: 0.7,
			{2, 3}: 0.9,
		}
		clusters := clusterFiles(ids, matrix, 0.5, 2)
		require.Len(t, clusters, 1)
		require.ElementsMatch(t, []int64{1, 2, 3}, clusters[0])
	})

	t.Run("two disconnected clusters", func(t *testing.T) {
		ids := []int64{1, 2, 3, 4, 5, 6}
		matrix := map[[2]int64]float64{
			{1, 2}: 0.8,
			{1, 3}: 0.7,
			{2, 3}: 0.9,
			{4, 5}: 0.8,
			{4, 6}: 0.7,
			{5, 6}: 0.9,
		}
		clusters := clusterFiles(ids, matrix, 0.5, 2)
		require.Len(t, clusters, 2)
	})

	t.Run("transitive connection via intermediate", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		matrix := map[[2]int64]float64{
			{1, 2}: 0.8,
			{2, 3}: 0.7,
			// 1 and 3 are NOT directly connected, but should cluster via 2
		}
		clusters := clusterFiles(ids, matrix, 0.5, 2)
		require.Len(t, clusters, 1)
		require.ElementsMatch(t, []int64{1, 2, 3}, clusters[0])
	})

	t.Run("minFiles filter", func(t *testing.T) {
		ids := []int64{1, 2}
		matrix := map[[2]int64]float64{
			{1, 2}: 0.9,
		}
		clusters := clusterFiles(ids, matrix, 0.5, 3)
		require.Empty(t, clusters)
	})

	t.Run("below threshold pair excluded", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		matrix := map[[2]int64]float64{
			{1, 2}: 0.9,
			// {1,3} and {2,3} not present
		}
		clusters := clusterFiles(ids, matrix, 0.5, 2)
		require.Len(t, clusters, 1)
		require.ElementsMatch(t, []int64{1, 2}, clusters[0])
	})
}

func TestGetExtensionTag(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".pdf", "document"},
		{".jpg", "image"},
		{".png", "image"},
		{".mp3", "audio"},
		{".zip", "archive"},
		{".mp4", "video"},
		{".txt", "text"},
		{".json", "data"},
		{".ini", "config"},
		{".unknown", ""},
		{"", ""},
		{".docx", "document"},
		{".unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := getExtensionTag(tt.ext)
		require.Equal(t, tt.want, got, "getExtensionTag(%q)", tt.ext)
	}
}

func TestCalculateClusterScore(t *testing.T) {
	ids := []int64{1, 2, 3}
	matrix := map[[2]int64]float64{
		{1, 2}: 0.8,
		{1, 3}: 0.6,
		{2, 3}: 0.7,
	}
	score := calculateClusterScore(ids, matrix)
	// (0.8 + 0.6 + 0.7) / 3 = 0.7
	require.InDelta(t, 0.7, score, 0.0001)
}

func TestGetFilePreview(t *testing.T) {
	names := map[int64]string{1: "a.pdf", 2: "b.pdf", 3: "c.pdf", 4: "d.pdf", 5: "e.pdf", 6: "f.pdf"}
	t.Run("up to 5 files", func(t *testing.T) {
		preview := getFilePreview([]int64{1, 2, 3, 4, 5, 6}, names)
		require.Len(t, preview, 5)
	})
	t.Run("fewer than 5", func(t *testing.T) {
		preview := getFilePreview([]int64{1, 2}, names)
		require.Len(t, preview, 2)
	})
}

func TestTopTerms(t *testing.T) {
	keywords := map[int64][]Keyword{
		1: {{Term: "report", Score: 10}, {Term: "financial", Score: 5}, {Term: "the", Score: 100}},
		2: {{Term: "report", Score: 8}, {Term: "budget", Score: 7}},
	}
	terms := topTerms(keywords, []int64{1, 2}, 3)
	// "report" = 18, "budget" = 7, "financial" = 5
	// "the" is a stopword, filtered
	require.Equal(t, []string{"report", "budget", "financial"}, terms)
}
