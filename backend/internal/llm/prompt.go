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

	return `These files were grouped by keyword similarity. Are they genuinely related?

Files:
` + strings.Join(fileStrings, "\n") + `

Respond ONLY with valid JSON (no markdown):
{"related": true, "removed": [], "name": "Specific Name", "description": "One sentence description"}`
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

func buildFolderGuardPrompt(folderName string, subfolders []string, fileNames []string, parentName string, parentGuarded bool) string {
	subfolderList := strings.Join(subfolders, ", ")
	if subfolderList == "" {
		subfolderList = "(none)"
	}

	fileList := strings.Join(fileNames, "\n- ")
	if fileList == "" {
		fileList = "(none)"
	}

	var parentHint string
	if parentName != "" {
		if parentGuarded {
			parentHint = fmt.Sprintf(`Parent: "%s" (related files)`, parentName)
		} else {
			parentHint = fmt.Sprintf(`Parent: "%s" (unrelated files)`, parentName)
		}
	}

	return fmt.Sprintf(`Folder: "%s"
%s
Subfolders: %s
Files:
- %s

Are the files in this folder related to each other (part of the same app/project/game)?
Answer YES if these files belong together.
Answer NO if these files are unrelated/random.

Respond ONLY with valid JSON (no markdown):
{"related": true/false, "reason": "brief explanation"}`, folderName, parentHint, subfolderList, fileList)
}

func parseFolderGuardResponse(raw string) (*FolderClassification, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")

	var result struct {
		Related bool   `json:"related"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}

	return &FolderClassification{
		Related: result.Related,
		Reason:  result.Reason,
	}, nil
}
