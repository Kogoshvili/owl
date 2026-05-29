package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"owl/internal/store"
)

type Scanner struct {
	store *store.Store
}

func New(s *store.Store) *Scanner {
	return &Scanner{store: s}
}

func (sc *Scanner) Scan(ctx context.Context, dirPath string, watchedDirID int64) error {
	start := time.Now()
	slog.Info("scanner: starting", "dir", dirPath)

	var seenPaths []string
	var parentDirs = map[string]bool{}
	var fileCount int

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if d.IsDir() {
			if path != dirPath && isHidden(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		if isHidden(d.Name()) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		parentDir := filepath.Dir(path)
		parentDirs[parentDir] = true

		ext := strings.ToLower(filepath.Ext(path))
		name := filepath.Base(path)
		mimeType := detectMIME(name, ext)

		f := &store.File{
			Path:         path,
			Name:         name,
			Extension:    ext,
			MimeType:     mimeType,
			Size:         info.Size(),
			ParentDir:    parentDir,
			WatchedDirID: &watchedDirID,
			Status:       "active",
			ModifiedAt:   info.ModTime(),
		}

		if _, err := sc.store.UpsertFile(f); err != nil {
			slog.Warn("scanner: failed to upsert file", "path", path, "error", err)
			return nil
		}

		seenPaths = append(seenPaths, path)
		fileCount++
		if fileCount%500 == 0 {
			slog.Info("scanner: progress", "files", fileCount)
		}
		return nil
	})

	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	var dirList []string
	for dir := range parentDirs {
		dirList = append(dirList, dir)
	}

	if err := sc.store.MarkMissingInDirs(dirList, seenPaths); err != nil {
		slog.Warn("scanner: failed to mark missing files", "error", err)
	}

	if err := sc.store.SetScanned(seenPaths); err != nil {
		slog.Warn("scanner: failed to set scanned", "error", err)
	}

	if err := sc.store.UpdateLastScanned(watchedDirID); err != nil {
		slog.Warn("scanner: failed to update last_scanned_at", "error", err)
	}

	slog.Info("scanner: complete", "dir", dirPath, "files", fileCount, "elapsed", time.Since(start).String())
	return nil
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func detectMIME(name, ext string) string {
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".txt", ".md", ".log":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".exe":
		return "application/x-msdownload"
	default:
		return "application/octet-stream"
	}
}
