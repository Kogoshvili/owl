package store

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS watched_directories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		enabled INTEGER NOT NULL DEFAULT 1,
		last_scanned_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		extension TEXT NOT NULL DEFAULT '',
		mime_type TEXT NOT NULL DEFAULT '',
		size INTEGER NOT NULL DEFAULT 0,
		parent_dir TEXT NOT NULL,
		watched_dir_id INTEGER REFERENCES watched_directories(id) ON DELETE CASCADE,
		status TEXT NOT NULL DEFAULT 'active',
		modified_at DATETIME NOT NULL,
		indexed_at DATETIME,
		content_indexed_at DATETIME,
		processing_status TEXT NOT NULL DEFAULT 'unprocessed',
		processing_error TEXT,
		file_metadata TEXT
	);
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		source TEXT NOT NULL DEFAULT 'manual',
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS file_tags (
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		source TEXT NOT NULL DEFAULT 'manual',
		PRIMARY KEY (file_id, tag_id)
	);
	CREATE TABLE IF NOT EXISTS folder_suggestions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		suggestion_type TEXT NOT NULL DEFAULT 'new_folder',
		confidence REAL NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS folder_suggestion_files (
		folder_suggestion_id INTEGER NOT NULL REFERENCES folder_suggestions(id) ON DELETE CASCADE,
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		PRIMARY KEY (folder_suggestion_id, file_id)
	);
	CREATE TABLE IF NOT EXISTS folder_guard_classifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		guarded INTEGER NOT NULL DEFAULT 0,
		reason TEXT,
		source TEXT NOT NULL DEFAULT 'llm',
		classified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
		file_id UNINDEXED, name, extension, content
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)
	return db
}

func insertWatchedDir(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO watched_directories (path) VALUES (?)`, path)
	require.NoError(t, err)
	id, _ := result.LastInsertId()
	return id
}

func insertFile(t *testing.T, db *sql.DB, f struct {
	path, name, ext, parentDir string
	watchedDirID               int64
	processingStatus           string
}) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO files (path, name, extension, parent_dir, watched_dir_id, status, modified_at, processing_status)
		VALUES (?, ?, ?, ?, ?, 'active', ?, ?)
	`, f.path, f.name, f.ext, f.parentDir, f.watchedDirID, time.Now(), f.processingStatus)
	require.NoError(t, err)
	id, _ := result.LastInsertId()
	return id
}

func TestStore_FileCRUD(t *testing.T) {
	db := openTestDB(t)
	s := New(db)
	wdID := insertWatchedDir(t, db, "/test")

	t.Run("list files — empty", func(t *testing.T) {
		files, err := s.ListFiles(FileFilter{})
		require.NoError(t, err)
		require.Empty(t, files)
	})

	t.Run("create and get file", func(t *testing.T) {
		f := File{
			Path:             "/test/a.txt",
			Name:             "a.txt",
			Extension:        ".txt",
			MimeType:         "text/plain",
			Size:             100,
			ParentDir:        "/test",
			WatchedDirID:     &wdID,
			Status:           "active",
			ModifiedAt:       time.Now(),
			ProcessingStatus: "unprocessed",
		}
		created, err := s.UpsertFile(&f)
		require.NoError(t, err)
		require.NotZero(t, created.ID)

		got, err := s.GetFile(created.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "a.txt", got.Name)
	})

	t.Run("list files with filter", func(t *testing.T) {
		files, err := s.ListFiles(FileFilter{Extension: strPtr(".txt")})
		require.NoError(t, err)
		require.Len(t, files, 1)
	})

	t.Run("count files", func(t *testing.T) {
		count, err := s.CountFiles(FileFilter{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("buildFileWhere with Supported filter", func(t *testing.T) {
		insertFile(t, db, struct {
			path, name, ext, parentDir string
			watchedDirID               int64
			processingStatus           string
		}{"/test/b.json", "b.json", ".json", "/test", wdID, "unprocessed"})

		// Supported = true should find .json
		trueVal := true
		cond, args := s.buildFileWhere(FileFilter{Supported: &trueVal})
		require.Contains(t, cond, "LOWER(extension) IN")
		files, err := s.ListFiles(FileFilter{Supported: &trueVal})
		require.NoError(t, err)
		require.NotEmpty(t, files)

		// Supported = false should return empty (both .txt and .json are supported)
		falseVal := false
		files2, err := s.ListFiles(FileFilter{Supported: &falseVal})
		require.NoError(t, err)
		require.Empty(t, files2)
		_ = cond
		_ = args
	})
}

func TestStore_Suggestions(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	t.Run("create suggestion", func(t *testing.T) {
		sug, err := s.CreateSuggestion("test-folder", "a test", "new_folder", 0)
		require.NoError(t, err)
		require.NotZero(t, sug.ID)
		require.Equal(t, "test-folder", sug.Name)
	})

	t.Run("create suggestion with type and confidence", func(t *testing.T) {
		sug, err := s.CreateSuggestion("auto-folder", "auto test", "new_folder", 0.5)
		require.NoError(t, err)
		require.NotZero(t, sug.ID)
		require.Equal(t, 0.5, sug.Confidence)
	})

	t.Run("get suggestion detail", func(t *testing.T) {
		wdID := insertWatchedDir(t, db, "/vf")
		fID := insertFile(t, db, struct {
			path, name, ext, parentDir string
			watchedDirID               int64
			processingStatus           string
		}{"/vf/f.txt", "f.txt", ".txt", "/vf", wdID, "unprocessed"})

		sug, _ := s.CreateSuggestion("detail-folder", "", "new_folder", 0)

		err := s.AddFilesToSuggestion(sug.ID, []int64{fID})
		require.NoError(t, err)

		detail, err := s.GetSuggestionDetail(sug.ID)
		require.NoError(t, err)
		require.NotNil(t, detail)
		require.Len(t, detail.Files, 1)
		require.Equal(t, "f.txt", detail.Files[0].Name)
	})

	t.Run("delete suggestion cascades", func(t *testing.T) {
		sug, _ := s.CreateSuggestion("delete-me", "", "new_folder", 0)
		err := s.DeleteSuggestion(sug.ID)
		require.NoError(t, err)

		got, _ := s.GetSuggestion(sug.ID)
		require.Nil(t, got)
	})

	t.Run("list suggestions", func(t *testing.T) {
		s.CreateSuggestion("sug1", "", "new_folder", 0.3)
		s.CreateSuggestion("sug2", "", "new_folder", 0.7)
		suggestions, err := s.ListSuggestions()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(suggestions), 2)
	})
}

func TestStore_FolderGuard(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	t.Run("set and get guard", func(t *testing.T) {
		err := s.SetFolderGuard("/test/path", true, "llm", "related files")
		require.NoError(t, err)

		guarded, source, err := s.GetFolderGuard("/test/path")
		require.NoError(t, err)
		require.True(t, guarded)
		require.Equal(t, "llm", source)
	})

	t.Run("list guards", func(t *testing.T) {
		s.SetFolderGuard("/test/other", false, "user", "")
		guards, err := s.ListFolderGuards()
		require.NoError(t, err)
		require.NotEmpty(t, guards)
	})

	t.Run("get guarded paths", func(t *testing.T) {
		paths, err := s.GetGuardedPaths()
		require.NoError(t, err)
		require.True(t, paths["/test/path"])
		require.False(t, paths["/test/other"])
	})

	t.Run("is folder guarded with ancestor", func(t *testing.T) {
		s.SetFolderGuard("/parent", true, "llm", "")
		guarded, err := s.IsFolderGuarded("/parent/child")
		require.NoError(t, err)
		require.True(t, guarded)
	})

	t.Run("delete guard", func(t *testing.T) {
		err := s.DeleteFolderGuard("/test/path")
		require.NoError(t, err)
		guarded, _, _ := s.GetFolderGuard("/test/path")
		require.False(t, guarded)
	})
}

func TestStore_PhysicalFolders(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	wdID := insertWatchedDir(t, db, "/root")
	insertFile(t, db, struct {
		path, name, ext, parentDir string
		watchedDirID               int64
		processingStatus           string
	}{"/root/a.txt", "a.txt", ".txt", "/root", wdID, "unprocessed"})
	insertFile(t, db, struct {
		path, name, ext, parentDir string
		watchedDirID               int64
		processingStatus           string
	}{"/root/sub/b.txt", "b.txt", ".txt", "/root/sub", wdID, "unprocessed"})

	t.Run("list physical folders", func(t *testing.T) {
		trees, err := s.ListPhysicalFoldersAll()
		require.NoError(t, err)
		require.NotEmpty(t, trees)
	})
}

func strPtr(s string) *string { return &s }
