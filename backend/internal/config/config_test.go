package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Init()
	require.NotNil(t, cfg)
	require.Equal(t, "3721", cfg.Port)
	require.Equal(t, "data", cfg.DataDir)
	require.Equal(t, "info", cfg.LogLevel)
	require.True(t, cfg.LLM.Enabled)
	require.Equal(t, "http://127.0.0.1:11434", cfg.LLM.BaseURL)
	require.Equal(t, "deepseek-r1:1.5b", cfg.LLM.Model)
	require.Equal(t, "nomic-embed-text:latest", cfg.LLM.EmbedModel)
	require.Equal(t, "content_tfidf", cfg.LLM.FolderStrategy)
}
