package intelligence

import (
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
	"txt":  "text",
	"md":   "markdown",
	"rtf":  "document",
	"odt":  "document",
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
}

type Tagger struct {
	analyzer *Analyzer
	store    *store.Store
}

func NewTagger(analyzer *Analyzer, store *store.Store) *Tagger {
	return &Tagger{
		analyzer: analyzer,
		store:    store,
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

	// Apply threshold for single-file auto-tag: only assign tags that
	// already have at least (minTagFileCount - 1) other files.
	// This prevents creating noise from unique path segments/content.
	return t.tagFile(file, content, true)
}

func (t *Tagger) AutoTagFiles(fileIDs []int64) (map[int64][]store.Tag, error) {
	// Collect all tag candidates across all files
	fileContents := make(map[int64]string)
	fileData := make(map[int64]*store.File)
	tagCandidates := make(map[string][]int64) // tagName -> fileIDs

	for _, fileID := range fileIDs {
		file, err := t.store.GetFile(fileID)
		if err != nil {
			return nil, err
		}
		if file == nil {
			continue
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

		fileData[fileID] = file
		fileContents[fileID] = content

		tagNames := t.collectTagsFromFile(file, content)
		for _, tagName := range tagNames {
			tagCandidates[tagName] = append(tagCandidates[tagName], fileID)
		}
	}

	// Filter: only keep tags with >= minTagFileCount files
	for name, ids := range tagCandidates {
		if len(ids) < minTagFileCount {
			delete(tagCandidates, name)
		}
	}

	// Pass 2: write tags that pass threshold
	result := make(map[int64][]store.Tag)
	for tagName, fileIDs := range tagCandidates {
		tag, err := t.store.EnsureTag(tagName, "auto")
		if err != nil {
			continue
		}
		for _, fileID := range fileIDs {
			if err := t.store.AddFileTag(fileID, tag.ID, "auto"); err != nil {
				continue
			}
			result[fileID] = append(result[fileID], *tag)
		}
	}

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

type fileTagCandidate struct {
	fileID    int64
	tagName   string
	tagSource string
}

func (t *Tagger) collectTagsFromFile(file *store.File, content string) []string {
	var tagNames []string
	seen := make(map[string]bool)

	extensionTag := t.getExtensionTag(file.Extension)
	if extensionTag != "" {
		if !seen[extensionTag] {
			tagNames = append(tagNames, extensionTag)
			seen[extensionTag] = true
		}
	}

	pathTags := t.extractTagsFromPath(file.Path)
	for _, tagName := range pathTags {
		if seen[tagName] {
			continue
		}
		tagNames = append(tagNames, tagName)
		seen[tagName] = true
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
			tagNames = append(tagNames, tagName)
			seen[tagName] = true
		}
	}

	return tagNames
}

func (t *Tagger) getExtensionTag(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	return extensionTags[ext]
}

func (t *Tagger) extractTagsFromPath(path string) []string {
	var tags []string

	dir := filepath.Dir(path)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 3 {
			continue
		}
		if part == "." || part == ".." {
			continue
		}
		if IsStopword(NormalizeTerm(part)) {
			continue
		}
		if IsNumeric(part) {
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