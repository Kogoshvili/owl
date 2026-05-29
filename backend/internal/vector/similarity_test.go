package vector

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := []float32{1, 2, 3}
		sim := CosineSimilarity(a, a)
		require.InDelta(t, 1.0, sim, 0.0001)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{0, 1, 0}
		sim := CosineSimilarity(a, b)
		require.InDelta(t, 0, sim, 0.0001)
	})

	t.Run("opposite vectors", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{-1, -2, -3}
		sim := CosineSimilarity(a, b)
		require.InDelta(t, -1.0, sim, 0.0001)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		sim := CosineSimilarity([]float32{1, 2}, []float32{1, 2, 3})
		require.Equal(t, float32(0), sim)
	})

	t.Run("empty vectors", func(t *testing.T) {
		sim := CosineSimilarity([]float32{}, []float32{})
		require.Equal(t, float32(0), sim)
	})

	t.Run("zero vector", func(t *testing.T) {
		a := []float32{0, 0, 0}
		b := []float32{1, 2, 3}
		sim := CosineSimilarity(a, b)
		require.Equal(t, float32(0), sim)
	})

	t.Run("partial overlap", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{1, 2, 1}
		sim := CosineSimilarity(a, b)
		// dot=1+4+3=8, normA^2=14, normB^2=6
		// sim=8/sqrt(84) ≈ 0.8729
		expected := float32(8) / (float32(math.Sqrt(14)) * float32(math.Sqrt(6)))
		require.InDelta(t, expected, sim, 0.0001)
	})
}

func TestEuclideanDistance(t *testing.T) {
	t.Run("identical points", func(t *testing.T) {
		a := []float32{1, 2, 3}
		d := EuclideanDistance(a, a)
		require.InDelta(t, 0, d, 0.0001)
	})

	t.Run("standard distance", func(t *testing.T) {
		a := []float32{0, 0, 0}
		b := []float32{3, 4, 0}
		d := EuclideanDistance(a, b)
		require.InDelta(t, 5, d, 0.0001)
	})

	t.Run("mismatched lengths", func(t *testing.T) {
		d := EuclideanDistance([]float32{1, 2}, []float32{1})
		require.Equal(t, float32(math.MaxFloat32), d)
	})

	t.Run("empty vectors", func(t *testing.T) {
		d := EuclideanDistance([]float32{}, []float32{})
		require.Equal(t, float32(math.MaxFloat32), d)
	})
}

func TestNormalize(t *testing.T) {
	t.Run("unit vector stays same", func(t *testing.T) {
		v := Normalize([]float32{1, 0, 0})
		require.InDelta(t, float32(1), v[0], 0.0001)
		require.InDelta(t, float32(0), v[1], 0.0001)
	})

	t.Run("non-unit vector", func(t *testing.T) {
		v := Normalize([]float32{3, 4})
		// length = 5, expect [0.6, 0.8]
		require.InDelta(t, float32(0.6), v[0], 0.0001)
		require.InDelta(t, float32(0.8), v[1], 0.0001)
	})

	t.Run("zero vector", func(t *testing.T) {
		v := Normalize([]float32{0, 0, 0})
		require.Equal(t, []float32{0, 0, 0}, v)
	})

	t.Run("preserves length", func(t *testing.T) {
		v := Normalize([]float32{1, 2, 3, 4, 5})
		var norm float32
		for _, x := range v {
			norm += x * x
		}
		require.InDelta(t, float32(1), float32(math.Sqrt(float64(norm))), 0.0001)
	})
}
