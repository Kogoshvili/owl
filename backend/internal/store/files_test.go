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
		recursive INTEGER NOT NULL DEFAULT 1,
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
	CREATE TABLE IF NOT EXISTS virtual_folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		materialized INTEGER NOT NULL DEFAULT 0,
		materialized_path TEXT,
		source TEXT NOT NULL DEFAULT 'manual',
		suggestion_type TEXT NOT NULL DEFAULT 'new_folder',
		confidence REAL NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS virtual_folder_files (
		virtual_folder_id INTEGER NOT NULL REFERENCES virtual_folders(id) ON DELETE CASCADE,
		file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
		source TEXT NOT NULL DEFAULT 'manual',
		PRIMARY KEY (virtual_folder_id, file_id)
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

func TestStore_Tags(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	t.Run("create and get tag", func(t *testing.T) {
		tag, err := s.CreateTag("test-tag", "manual", "a test tag")
		require.NoError(t, err)
		require.NotZero(t, tag.ID)
		require.Equal(t, "test-tag", tag.Name)
	})

	t.Run("get tag by name", func(t *testing.T) {
		tag, err := s.GetTagByName("test-tag")
		require.NoError(t, err)
		require.NotNil(t, tag)
	})

	t.Run("list tags", func(t *testing.T) {
		tags, err := s.ListTags()
		require.NoError(t, err)
		require.NotEmpty(t, tags)
	})

	t.Run("add file tag", func(t *testing.T) {
		wdID := insertWatchedDir(t, db, "/tags")
		fID := insertFile(t, db, struct {
			path, name, ext, parentDir string
			watchedDirID               int64
			processingStatus           string
		}{"/tags/f1.txt", "f1.txt", ".txt", "/tags", wdID, "unprocessed"})

		tag, err := s.EnsureTag("file-tag", "auto")
		require.NoError(t, err)

		err = s.AddFileTag(fID, tag.ID, "auto")
		require.NoError(t, err)

		fileTags, err := s.ListFileTags(fID)
		require.NoError(t, err)
		require.Len(t, fileTags, 1)
		require.Equal(t, "file-tag", fileTags[0].Name)
	})

	t.Run("remove file tag", func(t *testing.T) {
		wdID := insertWatchedDir(t, db, "/tags2")
		fID := insertFile(t, db, struct {
			path, name, ext, parentDir string
			watchedDirID               int64
			processingStatus           string
		}{"/tags2/f2.txt", "f2.txt", ".txt", "/tags2", wdID, "unprocessed"})
		tag, _ := s.EnsureTag("remove-tag", "auto")
		s.AddFileTag(fID, tag.ID, "auto")

		err := s.RemoveFileTag(fID, tag.ID)
		require.NoError(t, err)

		fileTags, _ := s.ListFileTags(fID)
		require.Empty(t, fileTags)
	})

	t.Run("delete tag", func(t *testing.T) {
		tag, _ := s.CreateTag("delete-me", "manual", "")
		err := s.DeleteTag(tag.ID)
		require.NoError(t, err)

		got, _ := s.GetTag(tag.ID)
		require.Nil(t, got)
	})
}

func TestStore_VirtualFolders(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	t.Run("create virtual folder", func(t *testing.T) {
		vf, err := s.CreateVirtualFolder("test-folder", "a test", false)
		require.NoError(t, err)
		require.NotZero(t, vf.ID)
		require.Equal(t, "test-folder", vf.Name)
	})

	t.Run("create auto virtual folder", func(t *testing.T) {
		vf, err := s.CreateVirtualFolderWithType("auto-folder", "auto test", "auto", "new_folder", 0.5)
		require.NoError(t, err)
		require.NotZero(t, vf.ID)
		require.Equal(t, "auto", vf.Source)
	})

	t.Run("get virtual folder detail", func(t *testing.T) {
		wdID := insertWatchedDir(t, db, "/vf")
		fID := insertFile(t, db, struct {
			path, name, ext, parentDir string
			watchedDirID               int64
			processingStatus           string
		}{"/vf/f.txt", "f.txt", ".txt", "/vf", wdID, "unprocessed"})

		vf, _ := s.CreateVirtualFolder("detail-folder", "", false)

		err := s.AddFilesToFolder(vf.ID, []int64{fID}, "manual")
		require.NoError(t, err)

		detail, err := s.GetVirtualFolderDetail(vf.ID)
		require.NoError(t, err)
		require.NotNil(t, detail)
		require.Len(t, detail.Files, 1)
		require.Equal(t, "f.txt", detail.Files[0].Name)
	})

	t.Run("delete virtual folder cascades", func(t *testing.T) {
		vf, _ := s.CreateVirtualFolder("delete-me", "", false)
		err := s.DeleteVirtualFolder(vf.ID)
		require.NoError(t, err)

		got, _ := s.GetVirtualFolder(vf.ID)
		require.Nil(t, got)
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

func TestStore_Search(t *testing.T) {
	db := openTestDB(t)
	s := New(db)

	wdID := insertWatchedDir(t, db, "/search")
	fID := insertFile(t, db, struct {
		path, name, ext, parentDir string
		watchedDirID               int64
		processingStatus           string
	}{"/search/report.txt", "report.txt", ".txt", "/search", wdID, "unprocessed"})

	// Insert FTS content
	err := s.UpsertFileFTS(fID, "report.txt", ".txt", "quarterly financial report")
	require.NoError(t, err)

	t.Run("search by filename", func(t *testing.T) {
		results, err := s.Search("report", "filenames", 10)
		require.NoError(t, err)
		require.NotEmpty(t, results.Files)
	})

	t.Run("search by content", func(t *testing.T) {
		results, err := s.Search("quarterly", "content", 10)
		require.NoError(t, err)
		require.NotEmpty(t, results.Files)
	})

	t.Run("search no results", func(t *testing.T) {
		results, err := s.Search("zzz_nonexistent", "filenames", 10)
		require.NoError(t, err)
		require.Empty(t, results.Files)
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
