package llm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildClusterPrompt(t *testing.T) {
	files := []ClusterFile{
		{ID: 1, Name: "report", Extension: ".pdf", ParentDir: "/downloads", Keywords: []string{"financial", "report"}},
		{ID: 2, Name: "budget", Extension: ".xlsx", ParentDir: "/downloads", Keywords: []string{"budget"}},
	}
	prompt := buildClusterPrompt(files, "group")
	require.Contains(t, prompt, "report.pdf")
	require.Contains(t, prompt, "budget.xlsx")
	require.Contains(t, prompt, "financial, report")
	require.Contains(t, prompt, "/downloads")
	require.Contains(t, prompt, `"related": true`)
	require.Contains(t, prompt, `"name": "Specific Name"`)
}

func TestBuildTagPrompt(t *testing.T) {
	prompt := buildTagPrompt("data", []string{"file1.txt", "file2.csv"}, []string{"data", "report"})
	require.Contains(t, prompt, `Tag: "data"`)
	require.Contains(t, prompt, "file1.txt")
	require.Contains(t, prompt, "file2.csv")
	require.Contains(t, prompt, "Keywords: data, report")
	require.Contains(t, prompt, `"keep": true`)
}

func TestBuildFolderGuardPrompt(t *testing.T) {
	t.Run("with parent context", func(t *testing.T) {
		prompt := buildFolderGuardPrompt("AxelChat", []string{"src", "docs"}, []string{"main.go", "readme.md"}, "Downloads", false)
		require.Contains(t, prompt, `Folder: "AxelChat"`)
		require.Contains(t, prompt, `Parent: "Downloads" (unrelated files)`)
		require.Contains(t, prompt, "Subfolders: src, docs")
		require.Contains(t, prompt, "- main.go")
		require.Contains(t, prompt, `"related": true/false`)
	})

	t.Run("no subfolders", func(t *testing.T) {
		prompt := buildFolderGuardPrompt("stuff", nil, []string{"a.txt"}, "", false)
		require.Contains(t, prompt, "Subfolders: (none)")
	})

	t.Run("no files", func(t *testing.T) {
		prompt := buildFolderGuardPrompt("stuff", []string{"sub"}, nil, "", false)
		require.Contains(t, prompt, "Files:\n- (none)")
	})

	t.Run("guarded parent", func(t *testing.T) {
		prompt := buildFolderGuardPrompt("sub", nil, nil, "Parent", true)
		require.Contains(t, prompt, `Parent: "Parent" (related files)`)
	})
}

func TestBuildKeywordExtractionPrompt(t *testing.T) {
	files := []struct {
		ID      int64
		Name    string
		Content string
	}{
		{ID: 1, Name: "doc.txt", Content: "the quick brown fox"},
	}
	prompt := buildKeywordExtractionPrompt(files)
	require.Contains(t, prompt, "[id:1]")
	require.Contains(t, prompt, "doc.txt")
	require.Contains(t, prompt, "quick brown fox")
	require.Contains(t, prompt, `"keywords": ["keyword1"`)
}

func TestParseClusterResponse(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		raw := `{"related": true, "removed": [2], "name": "Finance", "description": "Financial documents"}`
		result, err := parseClusterResponse(raw, []int64{100, 200, 300})
		require.NoError(t, err)
		require.True(t, result.Related)
		require.Equal(t, "Finance", result.Name)
		require.Equal(t, "Financial documents", result.Description)
		require.Equal(t, []int64{200}, result.RemovedIDs)
	})

	t.Run("with markdown fences", func(t *testing.T) {
		raw := "```json\n{\"related\": false, \"removed\": [], \"name\": \"\", \"description\": \"\"}\n```"
		result, err := parseClusterResponse(raw, []int64{1, 2})
		require.NoError(t, err)
		require.False(t, result.Related)
	})

	t.Run("removed index out of bounds", func(t *testing.T) {
		raw := `{"related": true, "removed": [1, 99], "name": "", "description": ""}`
		result, err := parseClusterResponse(raw, []int64{100})
		require.NoError(t, err)
		require.Equal(t, []int64{100}, result.RemovedIDs) // 99 is out of bounds, skipped
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := parseClusterResponse(`{bad json}`, nil)
		require.Error(t, err)
	})
}

func TestParseTagResponse(t *testing.T) {
	t.Run("keep true", func(t *testing.T) {
		raw := `{"keep": "true", "better_name": "reports", "description": "desc"}`
		result, err := parseTagResponse(raw)
		require.NoError(t, err)
		require.True(t, result.Keep)
		require.Equal(t, "reports", result.BetterName)
	})

	t.Run("keep false", func(t *testing.T) {
		raw := `{"keep": "false", "better_name": "", "description": ""}`
		result, err := parseTagResponse(raw)
		require.NoError(t, err)
		require.False(t, result.Keep)
	})

	t.Run("with markdown fences", func(t *testing.T) {
		raw := "```\n{\"keep\": \"true\"}\n```"
		result, err := parseTagResponse(raw)
		require.NoError(t, err)
		require.True(t, result.Keep)
	})
}

func TestParseFolderGuardResponse(t *testing.T) {
	t.Run("related true", func(t *testing.T) {
		raw := `{"related": true, "reason": "part of same app"}`
		result, err := parseFolderGuardResponse(raw)
		require.NoError(t, err)
		require.True(t, result.Related)
		require.Equal(t, "part of same app", result.Reason)
	})

	t.Run("related false", func(t *testing.T) {
		raw := `{"related": false, "reason": "random files"}`
		result, err := parseFolderGuardResponse(raw)
		require.NoError(t, err)
		require.False(t, result.Related)
	})

	t.Run("with markdown fences", func(t *testing.T) {
		raw := "```json\n{\"related\": true, \"reason\": \"test\"}\n```"
		result, err := parseFolderGuardResponse(raw)
		require.NoError(t, err)
		require.True(t, result.Related)
	})
}

func TestParseKeywordExtractionResponse(t *testing.T) {
	t.Run("valid JSON array", func(t *testing.T) {
		raw := `[{"id": 1, "keywords": ["kw1", "kw2"]}, {"id": 2, "keywords": ["kw3"]}]`
		result, err := parseKeywordExtractionResponse(raw)
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.Equal(t, int64(1), result[0].FileID)
		require.Equal(t, []string{"kw1", "kw2"}, result[0].Keywords)
		require.Equal(t, int64(2), result[1].FileID)
		require.Equal(t, []string{"kw3"}, result[1].Keywords)
	})

	t.Run("with markdown fences", func(t *testing.T) {
		raw := "```\n[{\"id\": 1, \"keywords\": [\"test\"]}]\n```"
		result, err := parseKeywordExtractionResponse(raw)
		require.NoError(t, err)
		require.Len(t, result, 1)
	})

	t.Run("empty array", func(t *testing.T) {
		result, err := parseKeywordExtractionResponse(`[]`)
		require.NoError(t, err)
		require.Empty(t, result)
	})
}

func TestBuildFolderGuardPromptContentTruncation(t *testing.T) {
	longNames := make([]string, 100)
	for i := range longNames {
		longNames[i] = strings.Repeat("a", 10)
	}
	prompt := buildFolderGuardPrompt("test", nil, longNames, "", false)
	// Should not crash, should contain file names
	require.Contains(t, prompt, `Folder: "test"`)
}

func TestBuildKeywordExtractionPromptContentTruncation(t *testing.T) {
	files := []struct {
		ID      int64
		Name    string
		Content string
	}{
		{ID: 1, Name: "long.txt", Content: strings.Repeat("x", 1000)},
	}
	prompt := buildKeywordExtractionPrompt(files)
	// Content should be truncated to 500 chars
	require.Contains(t, prompt, strings.Repeat("x", 500))
	require.NotContains(t, prompt, strings.Repeat("x", 501))
}
