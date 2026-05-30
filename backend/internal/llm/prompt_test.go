package llm

import (
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
		require.Equal(t, []int64{100}, result.RemovedIDs)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := parseClusterResponse(`{bad json}`, nil)
		require.Error(t, err)
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

func TestBuildFolderGuardPromptContentTruncation(t *testing.T) {
	longNames := make([]string, 100)
	for i := range longNames {
		longNames[i] = "aaaaaaaaaa"
	}
	prompt := buildFolderGuardPrompt("test", nil, longNames, "", false)
	require.Contains(t, prompt, `Folder: "test"`)
}
