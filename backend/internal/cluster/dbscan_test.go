package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDBSCAN(t *testing.T) {
	t.Run("no points", func(t *testing.T) {
		clusters := DBSCAN(nil, 0.5, 3)
		require.Nil(t, clusters)
	})

	t.Run("all noise below minPts", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{10, 10}},
			{ID: 3, Vec: []float32{20, 20}},
		}
		clusters := DBSCAN(points, 1.0, 3)
		require.Empty(t, clusters)
	})

	t.Run("single cluster", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{0.1, 0.1}},
			{ID: 3, Vec: []float32{0.2, 0.2}},
			{ID: 4, Vec: []float32{10, 10}},
		}
		clusters := DBSCAN(points, 1.0, 2)
		require.Len(t, clusters, 1)
		require.Len(t, clusters[0].Points, 3)
	})

	t.Run("noise points excluded", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{0.1, 0.1}},
			{ID: 3, Vec: []float32{10, 10}}, // noise
		}
		clusters := DBSCAN(points, 1.0, 2)
		require.Len(t, clusters, 1)
		for _, p := range clusters[0].Points {
			require.NotEqual(t, int64(3), p.ID)
		}
	})

	t.Run("two separate clusters with noise", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{0.1, 0.1}},
			{ID: 3, Vec: []float32{0.2, 0.2}},
			{ID: 4, Vec: []float32{10, 10}},
			{ID: 5, Vec: []float32{10.1, 10.1}},
			{ID: 6, Vec: []float32{10.2, 10.2}},
			{ID: 7, Vec: []float32{100, 100}}, // noise
		}
		clusters := DBSCAN(points, 1.0, 2)
		require.Len(t, clusters, 2)
		require.Len(t, clusters[0].Points, 3)
		require.Len(t, clusters[1].Points, 3)
	})

	t.Run("minPts filter", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{0.1, 0.1}}, // only 2 close, needs 3
		}
		clusters := DBSCAN(points, 1.0, 3)
		require.Empty(t, clusters)
	})

	t.Run("dense chain connects through intermediate", func(t *testing.T) {
		points := []Point{
			{ID: 1, Vec: []float32{0, 0}},
			{ID: 2, Vec: []float32{0.5, 0.5}},
			{ID: 3, Vec: []float32{1.0, 1.0}},
			{ID: 4, Vec: []float32{1.5, 1.5}},
		}
		// eps=1.0, each point is ~0.7 from its neighbor
		clusters := DBSCAN(points, 1.0, 2)
		require.Len(t, clusters, 1)
		require.Len(t, clusters[0].Points, 4)
	})
}

func TestEuclideanDist(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{3, 4, 0}
	d := euclideanDist(a, b)
	require.InDelta(t, float32(5), d, 0.0001)
}
