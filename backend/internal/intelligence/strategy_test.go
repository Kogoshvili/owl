package intelligence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStrategyID(t *testing.T) {
	require.Equal(t, StrategyContentTFIDF, ParseStrategyID("content_tfidf"))
	require.Equal(t, StrategyEmbeddings, ParseStrategyID("embeddings"))
	require.Equal(t, StrategyContentTFIDF, ParseStrategyID(""))
	require.Equal(t, StrategyContentTFIDF, ParseStrategyID("unknown"))
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)

	t.Run("empty registry returns nil", func(t *testing.T) {
		require.Nil(t, r.Get("unknown"))
	})

	t.Run("register and get", func(t *testing.T) {
		s := NewContentTFIDFStrategy(nil, nil, nil)
		r.Register(s)
		got := r.Get(StrategyContentTFIDF)
		require.NotNil(t, got)
		require.Equal(t, StrategyContentTFIDF, got.ID())
	})

	t.Run("default returns first registered", func(t *testing.T) {
		// After registering ContentTFIDF, Default() should return it
		def := r.Default()
		require.NotNil(t, def)
	})

	t.Run("list returns registered strategies", func(t *testing.T) {
		infos := r.List()
		require.NotEmpty(t, infos)
	})
}
