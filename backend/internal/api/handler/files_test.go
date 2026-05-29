package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/require"

	"owl/internal/store"
)

func setupFileHandler(t *testing.T) (*FileHandler, *sql.DB, int64) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec(`
		CREATE TABLE watched_directories (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, recursive INTEGER DEFAULT 1, enabled INTEGER DEFAULT 1, last_scanned_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, name TEXT NOT NULL, extension TEXT NOT NULL DEFAULT '', mime_type TEXT NOT NULL DEFAULT '', size INTEGER DEFAULT 0, parent_dir TEXT NOT NULL, watched_dir_id INTEGER REFERENCES watched_directories(id) ON DELETE CASCADE, status TEXT DEFAULT 'active', modified_at DATETIME, indexed_at DATETIME, content_indexed_at DATETIME, processing_status TEXT DEFAULT 'unprocessed', processing_error TEXT, file_metadata TEXT);
		CREATE TABLE comments (file_id INTEGER PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE, content TEXT NOT NULL DEFAULT '', updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
	`)
	s := store.New(db)

	result, err := db.Exec(`INSERT INTO watched_directories (path) VALUES ('/test')`)
	require.NoError(t, err)
	wdID, _ := result.LastInsertId()

	for i := 0; i < 3; i++ {
		names := []string{"a.txt", "b.txt", "c.json"}
		exts := []string{".txt", ".txt", ".json"}
		db.Exec(`INSERT INTO files (path, name, extension, parent_dir, watched_dir_id, status, modified_at, processing_status) VALUES (?, ?, ?, ?, ?, 'active', ?, 'unprocessed')`,
			"/test/"+names[i], names[i], exts[i], "/test", wdID, time.Now())
	}

	h := NewFileHandler(s, nil)
	return h, db, wdID
}

func TestFileHandler_List(t *testing.T) {
	h, _, _ := setupFileHandler(t)

	t.Run("list all files", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files", nil)
		w := httptest.NewRecorder()
		h.List(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		data := resp["data"].(map[string]any)
		files := data["files"].([]any)
		require.Len(t, files, 3)
		require.Equal(t, float64(3), data["total"])
	})

	t.Run("filter by extension", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files?extension=.json", nil)
		w := httptest.NewRecorder()
		h.List(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		files := data["files"].([]any)
		require.Len(t, files, 1)
	})

	t.Run("filter by supported", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files?supported=true", nil)
		w := httptest.NewRecorder()
		h.List(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		files := data["files"].([]any)
		// .txt is supported, .json is supported
		require.Len(t, files, 3)
	})

	t.Run("filter by supported=false", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files?supported=false", nil)
		w := httptest.NewRecorder()
		h.List(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		files := data["files"].([]any)
		// no files are unsupported
		require.Empty(t, files)
	})
}

func TestFileHandler_Extensions(t *testing.T) {
	h, _, _ := setupFileHandler(t)

	r := httptest.NewRequest("GET", "/files/extensions", nil)
	w := httptest.NewRecorder()
	h.Extensions(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	exts := resp["data"].([]any)
	require.ElementsMatch(t, []any{".txt", ".json"}, exts)
}

func TestFileHandler_Get(t *testing.T) {
	h, db, _ := setupFileHandler(t)

	var id int64
	db.QueryRow(`SELECT id FROM files LIMIT 1`).Scan(&id)

	t.Run("get existing file", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files/1", nil)
		r.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		h.Get(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		file := data["file"].(map[string]any)
		require.Equal(t, "a.txt", file["name"])
	})

	t.Run("get nonexistent file", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files/999", nil)
		r.SetPathValue("id", "999")
		w := httptest.NewRecorder()
		h.Get(w, r)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get with invalid id", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/files/abc", nil)
		r.SetPathValue("id", "abc")
		w := httptest.NewRecorder()
		h.Get(w, r)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})
	_ = id
}
