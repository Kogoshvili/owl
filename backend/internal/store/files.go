package store

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
)

type File struct {
	ID               int64             `json:"id"`
	Path             string            `json:"path"`
	Name             string            `json:"name"`
	Extension        string            `json:"extension"`
	MimeType         string            `json:"mime_type"`
	Size             int64             `json:"size"`
	ParentDir        string            `json:"parent_dir"`
	WatchedDirID     *int64            `json:"watched_dir_id"`
	Status           string            `json:"status"`
	ModifiedAt       time.Time         `json:"modified_at"`
	IndexedAt        *time.Time        `json:"indexed_at"`
	ContentIndexedAt *time.Time        `json:"content_indexed_at"`
	ProcessingStatus string            `json:"processing_status"`
	ProcessingError  *string           `json:"processing_error"`
	FileMetadata     map[string]any    `json:"file_metadata"`
}

const fileColumns = `id, path, name, extension, mime_type, size, parent_dir, watched_dir_id, status, modified_at, indexed_at, content_indexed_at, processing_status, processing_error, file_metadata`

const SupportedExtsSQL = "('.txt','.md','.log','.csv','.json','.xml','.yaml','.yml','.toml','.ini','.cfg','.conf','.sh','.bat','.ps1','.py','.js','.ts','.go','.rs','.java','.c','.cpp','.h','.hpp','.rb','.php','.sql','.env','.gitignore','.html','.htm','.svg','.css','.scss','.pdf','.docx','.xlsx','.pptx')"

func scanFile(scanner interface{ Scan(...any) error }, f *File) error {
	var metadataJSON sql.NullString
	err := scanner.Scan(&f.ID, &f.Path, &f.Name, &f.Extension, &f.MimeType, &f.Size, &f.ParentDir, &f.WatchedDirID, &f.Status, &f.ModifiedAt, &f.IndexedAt, &f.ContentIndexedAt, &f.ProcessingStatus, &f.ProcessingError, &metadataJSON)
	if err != nil {
		return err
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		json.Unmarshal([]byte(metadataJSON.String), &f.FileMetadata)
	}
	return nil
}

type FileFilter struct {
	Extension        *string
	Status           *string
	TagID            *int64
	ParentDir        *string
	WatchedDirID     *int64
	ProcessingStatus *string
	Supported        *bool
	SortBy           string
	SortOrder        string
	Limit            int
	Offset           int
}

var allowedSortColumns = map[string]string{
	"name":              "name",
	"extension":         "extension",
	"size":              "size",
	"modified_at":       "modified_at",
	"indexed_at":        "indexed_at",
	"processing_status": "processing_status",
}

func buildOrderBy(sortBy, sortOrder string) string {
	col, ok := allowedSortColumns[sortBy]
	if !ok {
		col = "indexed_at"
	}
	dir := "DESC"
	if strings.EqualFold(sortOrder, "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

func (s *Store) buildFileWhere(f FileFilter) (string, []any) {
	var conditions []string
	var args []any

	if f.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *f.Status)
	}
	if f.Extension != nil {
		conditions = append(conditions, "extension = ?")
		args = append(args, *f.Extension)
	}
	if f.ParentDir != nil {
		conditions = append(conditions, "parent_dir = ?")
		args = append(args, *f.ParentDir)
	}
	if f.TagID != nil {
		conditions = append(conditions, "id IN (SELECT file_id FROM file_tags WHERE tag_id = ?)")
		args = append(args, *f.TagID)
	}
	if f.WatchedDirID != nil {
		conditions = append(conditions, "watched_dir_id = ?")
		args = append(args, *f.WatchedDirID)
	}
	if f.ProcessingStatus != nil {
		conditions = append(conditions, "processing_status = ?")
		args = append(args, *f.ProcessingStatus)
	}
	if f.Supported != nil {
		if *f.Supported {
			conditions = append(conditions, "LOWER(extension) IN "+SupportedExtsSQL)
		} else {
			conditions = append(conditions, "LOWER(extension) NOT IN "+SupportedExtsSQL)
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + joinConditions(conditions, " AND ")
	}
	return where, args
}

func (s *Store) CountFiles(f FileFilter) (int, error) {
	where, args := s.buildFileWhere(f)
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM files"+where, args...).Scan(&count)
	return count, err
}

func (s *Store) ListFiles(f FileFilter) ([]File, error) {
	where, args := s.buildFileWhere(f)
	query := `SELECT ` + fileColumns + ` FROM files` + where + " ORDER BY " + buildOrderBy(f.SortBy, f.SortOrder)

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := scanFile(rows, &f); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Store) GetFile(id int64) (*File, error) {
	var f File
	err := scanFile(s.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE id = ?`, id), &f)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) GetFileByPath(path string) (*File, error) {
	var f File
	err := scanFile(s.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE path = ?`, path), &f)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) CreateFile(f *File) (*File, error) {
	result, err := s.db.Exec(
		`INSERT INTO files (path, name, extension, mime_type, size, parent_dir, watched_dir_id, status, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.Path, f.Name, f.Extension, f.MimeType, f.Size, f.ParentDir, f.WatchedDirID, f.Status, f.ModifiedAt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetFile(id)
}

func (s *Store) UpdateFileStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE files SET status = ? WHERE id = ?`, status, id)
	return err
}

func (s *Store) UpsertFile(f *File) (*File, error) {
	existing, err := s.GetFileByPath(f.Path)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return s.CreateFile(f)
	}

	if !existing.ModifiedAt.Equal(f.ModifiedAt) {
		_, err = s.db.Exec(
			`UPDATE files SET name = ?, extension = ?, mime_type = ?, size = ?, parent_dir = ?, watched_dir_id = ?, status = 'active', modified_at = ?, processing_status = 'stale', processing_error = NULL, content_indexed_at = NULL WHERE id = ?`,
			f.Name, f.Extension, f.MimeType, f.Size, f.ParentDir, f.WatchedDirID, f.ModifiedAt, existing.ID,
		)
		if err != nil {
			return nil, err
		}
	} else if existing.Status != "active" {
		s.db.Exec(`UPDATE files SET status = 'active' WHERE id = ?`, existing.ID)
	}

	return s.GetFile(existing.ID)
}

func (s *Store) MarkMissingInDirs(parentDirs []string, seenPaths []string) error {
	if len(parentDirs) == 0 {
		return nil
	}

	placeholders := ""
	args := []any{}
	for i, dir := range parentDirs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, dir)
	}

	query := `UPDATE files SET status = 'missing' WHERE parent_dir IN (` + placeholders + `) AND status = 'active'`

	if len(seenPaths) > 0 {
		notPlaceholders := ""
		for i, p := range seenPaths {
			if i > 0 {
				notPlaceholders += ","
			}
			notPlaceholders += "?"
			args = append(args, p)
		}
		query += ` AND path NOT IN (` + notPlaceholders + `)`
	}

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *Store) SetScanned(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE files SET indexed_at = CURRENT_TIMESTAMP WHERE path = ? AND indexed_at IS NULL`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range paths {
		stmt.Exec(p)
	}
	return tx.Commit()
}

func (s *Store) QueueFilesForExtraction(dirID int64) (int64, error) {
	result, err := s.db.Exec(
		`UPDATE files SET processing_status = 'queued', processing_error = NULL WHERE watched_dir_id = ? AND processing_status IN ('unprocessed', 'stale', 'failed') AND LOWER(extension) IN `+SupportedExtsSQL,
		dirID,
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Info("queued files for extraction", "dir_id", dirID, "count", n)
	}
	return n, nil
}

func (s *Store) QueueFileForExtraction(fileID int64) error {
	result, err := s.db.Exec(
		`UPDATE files SET processing_status = 'queued', processing_error = NULL WHERE id = ? AND processing_status IN ('unprocessed', 'stale', 'failed') AND LOWER(extension) IN `+SupportedExtsSQL,
		fileID,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) GetNextQueuedFile() (*File, error) {
	var f File
	err := scanFile(s.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE processing_status = 'queued' ORDER BY size ASC LIMIT 1`), &f)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) SetFileProcessing(fileID int64) error {
	_, err := s.db.Exec(`UPDATE files SET processing_status = 'processing' WHERE id = ?`, fileID)
	return err
}

func (s *Store) SetFileProcessed(fileID int64) error {
	_, err := s.db.Exec(
		`UPDATE files SET processing_status = 'processed', content_indexed_at = CURRENT_TIMESTAMP, processing_error = NULL WHERE id = ?`,
		fileID,
	)
	return err
}

func (s *Store) SetFileFailed(fileID int64, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE files SET processing_status = 'failed', processing_error = ? WHERE id = ?`,
		errMsg, fileID,
	)
	return err
}

func (s *Store) RecoverStuckFiles() {
	result, err := s.db.Exec(
		`UPDATE files SET processing_status = 'failed', processing_error = 'Interrupted: server restarted while processing' WHERE processing_status IN ('queued', 'processing')`,
	)
	if err != nil {
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		slog.Warn("recovered stuck files", "count", n)
	}
}

func (s *Store) SetFileMetadata(fileID int64, metadata map[string]any) error {
	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE files SET file_metadata = ? WHERE id = ?`, string(jsonBytes), fileID)
	return err
}

func (s *Store) GetFileContent(fileID int64) (string, error) {
	var content string
	err := s.db.QueryRow(`SELECT content FROM files_fts WHERE file_id = ?`, fileID).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return content, err
}

func (s *Store) UpsertFileFTS(fileID int64, name, extension, content string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM files_fts WHERE file_id = ?`, fileID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO files_fts (file_id, name, extension, content) VALUES (?, ?, ?, ?)`,
		fileID, name, extension, content,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListExtensions() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT extension FROM files WHERE extension != '' ORDER BY extension`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exts []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		exts = append(exts, e)
	}
	return exts, rows.Err()
}


