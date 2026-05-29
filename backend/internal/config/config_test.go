package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.False(t, cfg.LLM.Enabled)
	require.Equal(t, "http://localhost:1234/v1", cfg.LLM.BaseURL)
	require.Empty(t, cfg.LLM.Model)
	require.Empty(t, cfg.LLM.EmbedModel)
	require.Equal(t, "content_tfidf", cfg.LLM.FolderStrategy)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	err := os.WriteFile(configPath, []byte(`{"llm": {"enabled": true, "model": "test-model"}}`), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.True(t, cfg.LLM.Enabled)
	require.Equal(t, "test-model", cfg.LLM.Model)
	// Should keep defaults for fields not in file
	require.Equal(t, "http://localhost:1234/v1", cfg.LLM.BaseURL)
	require.Equal(t, "content_tfidf", cfg.LLM.FolderStrategy)
}

func TestLoadEnvOverrides(t *testing.T) {
	os.Setenv("LLM_ENABLED", "true")
	os.Setenv("LLM_BASE_URL", "http://test:9999/v1")
	os.Setenv("LLM_MODEL", "env-model")
	os.Setenv("EMBED_MODEL", "env-embed")
	os.Setenv("FOLDER_STRATEGY", "embeddings")
	defer func() {
		os.Unsetenv("LLM_ENABLED")
		os.Unsetenv("LLM_BASE_URL")
		os.Unsetenv("LLM_MODEL")
		os.Unsetenv("EMBED_MODEL")
		os.Unsetenv("FOLDER_STRATEGY")
	}()

	cfg, err := Load("")
	require.NoError(t, err)
	require.True(t, cfg.LLM.Enabled)
	require.Equal(t, "http://test:9999/v1", cfg.LLM.BaseURL)
	require.Equal(t, "env-model", cfg.LLM.Model)
	require.Equal(t, "env-embed", cfg.LLM.EmbedModel)
	require.Equal(t, "embeddings", cfg.LLM.FolderStrategy)
}

func TestLoadFileWithEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	err := os.WriteFile(configPath, []byte(`{"llm": {"base_url": "http://file:8888/v1", "model": "file-model"}}`), 0644)
	require.NoError(t, err)

	os.Setenv("LLM_MODEL", "env-override")
	defer os.Unsetenv("LLM_MODEL")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	// File value for base_url
	require.Equal(t, "http://file:8888/v1", cfg.LLM.BaseURL)
	// Env var overrides file value
	require.Equal(t, "env-override", cfg.LLM.Model)
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.False(t, cfg.LLM.Enabled)
}
