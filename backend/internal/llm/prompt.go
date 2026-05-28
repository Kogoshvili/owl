package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildClusterPrompt(files []ClusterFile, currentName string) string {
	var fileStrings []string
	for i, f := range files {
		keywords := strings.Join(f.Keywords, ", ")
		fileStrings = append(fileStrings, fmt.Sprintf("%d. %s%s (in %s) — keywords: %s", i+1, f.Name, f.Extension, f.ParentDir, keywords))
	}

	return `You are a file organization assistant. Given a group of files clustered by keyword similarity, evaluate and improve the grouping.

For each file you see: filename, extension, folder location, and top keywords extracted from its content.

Determine:
1. Are these files genuinely related? (not just sharing common words like "data" or "report")
2. Which files don't belong? (list by number, e.g., "2, 5")
3. Suggest a concise folder name (2-4 words, specific not generic)
4. Write a one-sentence description of what these files share

Files:
` + strings.Join(fileStrings, "\n") + `

Respond ONLY with valid JSON (no markdown):
{"related": true, "removed": [2,5], "name": "Specific Name", "description": "One sentence description"}`
}

func buildTagPrompt(tagName string, fileNames []string, keywords []string) string {
	fileList := strings.Join(fileNames, "\n- ")
	keywordList := strings.Join(keywords, ", ")

	return `You are a file organization assistant. Evaluate whether an auto-generated tag is meaningful and suggest a better name if possible.

Current tag: "` + tagName + `"

Files with this tag:
- ` + fileList + `

Shared keywords: ` + keywordList + `

Determine:
1. Is this tag meaningful? (tags like "data", "file", "self" are not meaningful)
2. Suggest a better, more specific tag name (2-3 words) if the current one is vague
3. Write a one-sentence description of what this tag represents

Respond ONLY with valid JSON (no markdown):
{"meaningful": true, "better_name": "specific name", "description": "One sentence description"}`
}

func parseClusterResponse(raw string, fileIDs []int64) (*RefinementResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")

	var result struct {
		Related     bool   `json:"related"`
		Removed     []int  `json:"removed"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}

	removedIDs := []int64{}
	for _, idx := range result.Removed {
		if idx < 1 || idx > len(fileIDs) {
			continue
		}
		removedIDs = append(removedIDs, fileIDs[idx-1])
	}

	return &RefinementResult{
		Related:     result.Related,
		RemovedIDs:  removedIDs,
		Name:        result.Name,
		Description: result.Description,
	}, nil
}

func parseTagResponse(raw string) (*TagRefinementResult, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")

	var result TagRefinementResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func buildKeywordExtractionPrompt(files []struct{ ID int64; Name string; Content string }) string {
	var fileStrings []string
	for i, f := range files {
		content := f.Content
		if len(content) > 500 {
			content = content[:500]
		}
		fileStrings = append(fileStrings, fmt.Sprintf("%d. [id:%d] %s\nContent preview: %s", i+1, f.ID, f.Name, content))
	}

	return `You are a file analysis assistant. For each file below, extract 5-10 keywords that best describe its content and purpose.

Rules:
- Extract specific, meaningful keywords (not generic words like "file", "data", "document")
- Include topic, technology, domain, or subject keywords
- Keywords should be lowercase, 1-3 words each
- Return exactly the same number of entries as files provided

Files:
` + strings.Join(fileStrings, "\n\n") + `

Respond ONLY with valid JSON array (no markdown):
[{"id": 1, "keywords": ["keyword1", "keyword2", ...]}, ...]`
}

func parseKeywordExtractionResponse(raw string) ([]KeywordExtraction, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")

	var result []struct {
		ID       int64    `json:"id"`
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}

	extractions := make([]KeywordExtraction, len(result))
	for i, r := range result {
		extractions[i] = KeywordExtraction{
			FileID:   r.ID,
			Keywords: r.Keywords,
		}
	}
	return extractions, nil
}