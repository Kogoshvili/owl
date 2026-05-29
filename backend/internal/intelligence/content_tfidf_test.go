package intelligence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCosineSimilarityMaps(t *testing.T) {
	t.Run("identical maps", func(t *testing.T) {
		a := map[string]float64{"term1": 2.0, "term2": 3.0}
		sim := cosineSimilarityMaps(a, a)
		require.InDelta(t, 1.0, sim, 0.0001)
	})

	t.Run("no overlap", func(t *testing.T) {
		a := map[string]float64{"term1": 1.0}
		b := map[string]float64{"term2": 1.0}
		sim := cosineSimilarityMaps(a, b)
		require.InDelta(t, 0.0, sim, 0.0001)
	})

	t.Run("partial overlap", func(t *testing.T) {
		a := map[string]float64{"term1": 1.0, "common": 2.0}
		b := map[string]float64{"term2": 1.0, "common": 2.0}
		sim := cosineSimilarityMaps(a, b)
		// dot=4, normA^2=5, normB^2=5, sim=4/5=0.8
		require.InDelta(t, 0.8, sim, 0.0001)
	})

	t.Run("empty map", func(t *testing.T) {
		a := map[string]float64{}
		b := map[string]float64{"term1": 1.0}
		sim := cosineSimilarityMaps(a, b)
		require.InDelta(t, 0.0, sim, 0.0001)
	})

	t.Run("both empty", func(t *testing.T) {
		sim := cosineSimilarityMaps(map[string]float64{}, map[string]float64{})
		require.InDelta(t, 0.0, sim, 0.0001)
	})
}
