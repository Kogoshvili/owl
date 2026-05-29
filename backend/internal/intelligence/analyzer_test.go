package intelligence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"hello, world!", []string{"hello", "world"}},
		{"  spaced  out  ", []string{"spaced", "out"}},
		{"", nil},
		{"123 abc", []string{"123", "abc"}},
	}
	for _, tt := range tests {
		got := Tokenize(tt.input)
		require.Equal(t, tt.want, got, "Tokenize(%q)", tt.input)
	}
}

func TestNormalizeTerm(t *testing.T) {
	require.Equal(t, "hello", NormalizeTerm("Hello"))
	require.Equal(t, "hello", NormalizeTerm("  Hello  "))
}

func TestIsStopword(t *testing.T) {
	require.True(t, IsStopword("the"))
	require.True(t, IsStopword("and"))
	require.True(t, IsStopword("i"))
	require.False(t, IsStopword("report"))
	require.False(t, IsStopword("financial"))
}

func TestIsNumeric(t *testing.T) {
	require.True(t, IsNumeric("123"))
	require.True(t, IsNumeric("0"))
	require.False(t, IsNumeric("123abc"))
	require.False(t, IsNumeric(""))
	require.False(t, IsNumeric("abc"))
}

func TestTopKeywords(t *testing.T) {
	t.Run("returns top N by score", func(t *testing.T) {
		kws := []Keyword{
			{Term: "a", Score: 1},
			{Term: "b", Score: 5},
			{Term: "c", Score: 3},
		}
		result := topKeywords(kws, 2)
		require.Len(t, result, 2)
		require.Equal(t, "b", result[0].Term)
	})

	t.Run("returns all if N >= len", func(t *testing.T) {
		kws := []Keyword{{Term: "a", Score: 1}, {Term: "b", Score: 2}}
		result := topKeywords(kws, 5)
		require.Len(t, result, 2)
	})

	t.Run("returns all if N <= 0", func(t *testing.T) {
		kws := []Keyword{{Term: "a", Score: 1}}
		result := topKeywords(kws, 0)
		require.Len(t, result, 1)
	})
}

func TestExtractKeywordsFromContent(t *testing.T) {
	t.Run("extracts and filters content", func(t *testing.T) {
		content := "the quick brown fox jumps over the lazy dog 123"
		kws := ExtractKeywordsFromContent(content, 10)
		terms := make([]string, len(kws))
		for i, kw := range kws {
			terms[i] = kw.Term
		}
		// "the", "123" filtered (stopword, numeric)
		// remaining: quick, brown, fox, jumps, over, lazy, dog
		require.ElementsMatch(t, []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"}, terms)
	})

	t.Run("empty content", func(t *testing.T) {
		kws := ExtractKeywordsFromContent("", 10)
		require.Empty(t, kws)
	})
}
