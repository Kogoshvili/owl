package extractor

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"owl/internal/store"
)

const (
	maxFileSize   = 50 * 1024 * 1024
	maxTextLength = 500 * 1024
)

type Extractor struct {
	store *store.Store
}

func New(s *store.Store) *Extractor {
	return &Extractor{store: s}
}

var supportedExtensions = map[string]bool{
	".txt":  true, ".md": true, ".log": true,
	".csv": true, ".json": true, ".xml": true,
	".yaml": true, ".yml": true, ".toml": true,
	".ini": true, ".cfg": true, ".conf": true,
	".sh": true, ".bat": true, ".ps1": true,
	".py": true, ".js": true, ".ts": true,
	".go": true, ".rs": true, ".java": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".rb": true, ".php": true, ".sql": true,
	".env": true, ".gitignore": true,
	".html": true, ".htm": true, ".svg": true,
	".css": true, ".scss": true,
	".pdf": true,
	".docx": true, ".xlsx": true, ".pptx": true,
}

func IsSupported(ext string) bool {
	return supportedExtensions[strings.ToLower(ext)]
}

type ProgressFn func(processed, total int)

func (e *Extractor) ProcessAll(ctx context.Context, progress ProgressFn) error {
	start := time.Now()
	processed := 0
	slog.Info("extractor: starting ProcessAll")

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		didProcess, err := e.ProcessNext(ctx)
		if err != nil {
			return err
		}
		if !didProcess {
			slog.Info("extractor: complete", "processed", processed, "elapsed", time.Since(start).String())
			return nil
		}
		processed++
		if progress != nil {
			progress(processed, 0)
		}
		if processed%100 == 0 {
			slog.Info("extractor: progress", "processed", processed)
		}
	}
}

func (e *Extractor) ProcessNext(ctx context.Context) (bool, error) {
	file, err := e.store.GetNextQueuedFile()
	if err != nil {
		return false, err
	}
	if file == nil {
		return false, nil
	}

	if err := e.store.SetFileProcessing(file.ID); err != nil {
		return false, err
	}

	if file.Size > maxFileSize {
		e.store.SetFileFailed(file.ID, "file too large")
		return true, nil
	}

	text, err := e.extract(ctx, file)
	if err != nil {
		slog.Warn("extractor: failed to extract", "path", file.Path, "error", err)
		e.store.SetFileFailed(file.ID, truncateString(err.Error(), 500))
		return true, nil
	}

	metadata := extractMetadata(file)
	if len(metadata) > 0 {
		if err := e.store.SetFileMetadata(file.ID, metadata); err != nil {
			slog.Warn("extractor: failed to set metadata", "path", file.Path, "error", err)
		}
	}

	if text == "" {
		e.store.SetFileProcessed(file.ID)
		return true, nil
	}

	if err := e.store.UpsertFileFTS(file.ID, file.Name, file.Extension, text); err != nil {
		slog.Warn("extractor: failed to upsert FTS", "path", file.Path, "error", err)
		e.store.SetFileFailed(file.ID, truncateString(err.Error(), 500))
		return true, nil
	}

	if err := e.store.SetFileProcessed(file.ID); err != nil {
		slog.Warn("extractor: failed to set processed", "path", file.Path, "error", err)
	}

	slog.Debug("extractor: processed file", "path", file.Path, "name", file.Name, "size", file.Size)
	return true, nil
}

func (e *Extractor) extract(ctx context.Context, f *store.File) (string, error) {
	ext := strings.ToLower(f.Extension)

	switch {
	case isPlainText(ext):
		return extractText(f.Path)
	case ext == ".pdf":
		return extractPDF(f.Path)
	case ext == ".docx":
		return extractDOCX(f.Path)
	case ext == ".xlsx":
		return extractXLSX(f.Path)
	case ext == ".pptx":
		return extractPPTX(f.Path)
	default:
		return "", nil
	}
}

func isPlainText(ext string) bool {
	switch ext {
	case ".txt", ".md", ".log", ".csv", ".json", ".xml",
		".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf",
		".sh", ".bat", ".ps1", ".py", ".js", ".ts",
		".go", ".rs", ".java", ".c", ".cpp", ".h", ".hpp",
		".rb", ".php", ".sql", ".env", ".gitignore",
		".html", ".htm", ".svg", ".css", ".scss":
		return true
	}
	return false
}

func extractText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return truncateToString(data), nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func truncateToString(data []byte) string {
	if len(data) > maxTextLength {
		data = data[:maxTextLength]
	}
	return string(bytes.ToValidUTF8(data, []byte("")))
}
