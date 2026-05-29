package intelligence

import (
	"context"
	"log/slog"
	"owl/internal/llm"
	"owl/internal/store"
	"path/filepath"
	"strings"
)

// minTagFileCount is the minimum number of files a tag must apply to
// during bulk auto-tagging for it to be created/assigned.
// TODO: Make configurable via settings in a future release.
const minTagFileCount = 3

var extensionTags = map[string]string{
	"pdf":  "document",
	"docx": "document",
	"doc":  "document",
	"xlsx": "spreadsheet",
	"pptx": "presentation",
	"txt":  "text",
	"md":   "markdown",
	"rtf":  "document",
	"odt":  "document",
	"log":  "text",
	"jpg":  "image",
	"jpeg": "image",
	"png":  "image",
	"gif":  "image",
	"svg":  "image",
	"bmp":  "image",
	"webp": "image",
	"tiff": "image",
	"mp3":  "audio",
	"wav":  "audio",
	"flac": "audio",
	"ogg":  "audio",
	"aac":  "audio",
	"m4a":  "audio",
	"wma":  "audio",
	"mp4":  "video",
	"avi":  "video",
	"mkv":  "video",
	"mov":  "video",
	"wmv":  "video",
	"flv":  "video",
	"webm": "video",
	"zip":  "archive",
	"tar":  "archive",
	"gz":   "archive",
	"rar":  "archive",
	"7z":   "archive",
	"json": "data",
	"xml":  "data",
	"csv":  "data",
	"yaml": "data",
	"yml":  "data",
	"toml": "data",
	"ini":  "config",
	"cfg":  "config",
	"conf": "config",
	"env":  "config",
	"py":   "code",
	"js":   "code",
	"ts":   "code",
	"go":   "code",
	"rs":   "code",
	"java": "code",
	"c":    "code",
	"cpp":  "code",
	"h":    "code",
	"hpp":  "code",
	"rb":   "code",
	"php":  "code",
	"sh":   "code",
	"bat":  "code",
	"ps1":  "code",
	"sql":  "data",
	"html": "web",
	"htm":  "web",
	"css":  "web",
	"scss": "web",
}

type Tagger struct {
	analyzer  *Analyzer
	store     *store.Store
	llm       *llm.Client
	registry  *Registry
}

func NewTagger(analyzer *Analyzer, store *store.Store, llmClient *llm.Client, registry *Registry) *Tagger {
	return &Tagger{
		analyzer: analyzer,
		store:    store,
		llm:      llmClient,
		registry: registry,
	}
}

func (t *Tagger) AutoTagFile(fileID int64) ([]store.Tag, error) {
	file, err := t.store.GetFile(fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}

	content := ""
	if file.ProcessingStatus == "processed" {
		keywords, err := t.analyzer.GetFileKeywords(fileID, 10)
		if err == nil && len(keywords) > 0 {
			contentKeywords := make([]string, 0, len(keywords))
			for _, kw := range keywords {
				contentKeywords = append(contentKeywords, kw.Term)
			}
			content = strings.Join(contentKeywords, " ")
		}
	}

	return t.tagFile(file, content, true)
}

func (t *Tagger) AutoTagFiles(ctx context.Context, fileIDs []int64, strategyID StrategyID) (map[int64][]store.Tag, error) {
	strategy := t.registry.Get(strategyID)
	if strategy == nil {
		strategy = t.registry.Default()
	}

	slog.Info("auto-tag: starting", "files", len(fileIDs), "strategy", strategyID, "llm_enabled", t.llm != nil && t.llm.IsAvailable(ctx))

	suggestions, err := strategy.SuggestTags(ctx, fileIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]store.Tag)
	createdTags := make(map[string]*store.Tag)

	for _, sug := range suggestions {
		tag, err := t.store.EnsureTag(sug.Name, "auto")
		if err != nil {
			continue
		}
		createdTags[sug.Name] = tag
		for _, fileID := range sug.FileIDs {
			if err := t.store.AddFileTag(fileID, tag.ID, "auto"); err != nil {
				continue
			}
			result[fileID] = append(result[fileID], *tag)
		}
	}

	llmAvailable := t.llm != nil && t.llm.IsAvailable(ctx)
	if llmAvailable {
		slog.Info("auto-tag: LLM refinement", "tags_to_evaluate", len(createdTags))
		for tagName, tag := range createdTags {
			files, err := t.store.GetFilesByTag(tag.ID)
			if err != nil || len(files) == 0 {
				continue
			}

			fileNames := make([]string, 0, len(files))
			for _, f := range files {
				fileNames = append(fileNames, f.Name)
			}

			keywords := t.getTagKeywords(files)

			refinement, err := t.llm.RefineTag(ctx, tagName, fileNames, keywords)
			if err != nil {
				continue
			}

			if !refinement.Keep {
				slog.Info("auto-tag: LLM rejected tag", "name", tagName)
				t.store.DeleteTag(tag.ID)
				for fileID := range result {
					if tags, ok := result[fileID]; ok {
						filtered := []store.Tag{}
						for _, t := range tags {
							if t.ID != tag.ID {
								filtered = append(filtered, t)
							}
						}
						result[fileID] = filtered
					}
				}
				continue
			}

			if refinement.BetterName != "" && refinement.BetterName != tagName {
				slog.Info("auto-tag: LLM renamed tag", "from", tagName, "to", refinement.BetterName)
				_ = t.store.UpdateTagName(tag.ID, refinement.BetterName)
			}

			if refinement.Description != "" {
				slog.Info("auto-tag: LLM added description to tag", "name", tag.Name, "description", refinement.Description)
				_ = t.store.UpdateTagDescription(tag.ID, refinement.Description)
			}
		}
	}

	slog.Info("auto-tag: complete", "files_tagged", len(result), "tags_written", len(createdTags))

	return result, nil
}

func (t *Tagger) tagFile(file *store.File, content string, applyThreshold bool) ([]store.Tag, error) {
	var tags []store.Tag
	seen := make(map[string]bool)

	extensionTag := t.getExtensionTag(file.Extension)
	if extensionTag != "" {
		if shouldApplyTag(extensionTag, file.ID, t.store, applyThreshold) {
			if tag, err := t.store.EnsureTag(extensionTag, "auto"); err == nil {
				tags = append(tags, *tag)
				seen[extensionTag] = true
				if err := t.store.AddFileTag(file.ID, tag.ID, "auto"); err != nil {
					return nil, err
				}
			}
		}
	}

	pathTags := t.extractTagsFromPath(file.Path)
	for _, tagName := range pathTags {
		if seen[tagName] {
			continue
		}
		if shouldApplyTag(tagName, file.ID, t.store, applyThreshold) {
			if tag, err := t.store.EnsureTag(tagName, "auto"); err == nil {
				tags = append(tags, *tag)
				seen[tagName] = true
				if err := t.store.AddFileTag(file.ID, tag.ID, "auto"); err != nil {
					return nil, err
				}
			}
		}
	}

	if content != "" {
		contentTags := t.extractTagsFromContent(content)
		maxContentTags := 5
		for i, tagName := range contentTags {
			if i >= maxContentTags {
				break
			}
			if seen[tagName] {
				continue
			}
			if shouldApplyTag(tagName, file.ID, t.store, applyThreshold) {
				if tag, err := t.store.EnsureTag(tagName, "auto"); err == nil {
					tags = append(tags, *tag)
					seen[tagName] = true
					if err := t.store.AddFileTag(file.ID, tag.ID, "auto"); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return tags, nil
}

// shouldApplyTag checks if a tag should be applied to a file.
// If applyThreshold is true, the tag must have at least (minTagFileCount - 1) other files
// (i.e., including this file, at least minTagFileCount total).
// If the tag doesn't exist yet, we allow creating it (first file with this tag).
func shouldApplyTag(tagName string, fileID int64, st *store.Store, applyThreshold bool) bool {
	if !applyThreshold {
		return true
	}

	// Check if tag exists
	tag, err := st.GetTagByName(tagName)
	if err != nil {
		// Tag doesn't exist, allow creating it (first file)
		return true
	}

	// Count how many files currently have this tag
	count, err := st.CountFilesByTag(tag.ID)
	if err != nil {
		// On error, allow the tag (conservative)
		return true
	}

	// Tag must have at least (minTagFileCount - 1) other files already
	// (so including this file, total >= minTagFileCount)
	return count >= (minTagFileCount - 1)
}



func (t *Tagger) getTagKeywords(files []store.File) []string {
	keywordCount := make(map[string]int)
	for _, f := range files {
		if f.ProcessingStatus == "processed" {
			keywords, err := t.analyzer.GetFileKeywords(f.ID, 5)
			if err != nil {
				continue
			}
			for _, kw := range keywords {
				keywordCount[kw.Term]++
			}
		}
	}

	type kwCount struct {
		keyword string
		count   int
	}
	kws := make([]kwCount, 0, len(keywordCount))
	for k, v := range keywordCount {
		kws = append(kws, kwCount{k, v})
	}

	for i := 0; i < len(kws); i++ {
		for j := i + 1; j < len(kws); j++ {
			if kws[i].count < kws[j].count {
				kws[i], kws[j] = kws[j], kws[i]
			}
		}
	}

	result := []string{}
	for i, kw := range kws {
		if i >= 10 {
			break
		}
		result = append(result, kw.keyword)
	}
	return result
}

func (t *Tagger) getExtensionTag(ext string) string {
	return getExtensionTag(ext)
}

func (t *Tagger) extractTagsFromPath(path string) []string {
	var tags []string
	dir := filepath.Dir(path)
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 3 || part == "." || part == ".." {
			continue
		}
		term := NormalizeTerm(part)
		if !IsStopword(term) && !IsNumeric(term) {
			tags = append(tags, term)
		}
	}
	return tags
}

func (t *Tagger) extractTagsFromContent(content string) []string {
	keywords := ExtractKeywordsFromContent(content, 10)
	tags := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if !IsStopword(kw.Term) && !IsNumeric(kw.Term) {
			tags = append(tags, kw.Term)
		}
	}
	return tags
}

