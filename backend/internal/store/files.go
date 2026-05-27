package store

import (
	"database/sql"
	"time"
)

type File struct {
	ID               int64      `json:"id"`
	Path             string     `json:"path"`
	Name             string     `json:"name"`
	Extension        string     `json:"extension"`
	MimeType         string     `json:"mime_type"`
	Size             int64      `json:"size"`
	ParentDir        string     `json:"parent_dir"`
	WatchedDirID     *int64     `json:"watched_dir_id"`
	Status           string     `json:"status"`
	ModifiedAt       time.Time  `json:"modified_at"`
	IndexedAt        *time.Time `json:"indexed_at"`
	ContentIndexedAt *time.Time `json:"content_indexed_at"`
}

const fileColumns = `id, path, name, extension, mime_type, size, parent_dir, watched_dir_id, status, modified_at, indexed_at, content_indexed_at`

func scanFile(scanner interface{ Scan(...any) error }, f *File) error {
	return scanner.Scan(&f.ID, &f.Path, &f.Name, &f.Extension, &f.MimeType, &f.Size, &f.ParentDir, &f.WatchedDirID, &f.Status, &f.ModifiedAt, &f.IndexedAt, &f.ContentIndexedAt)
}

type FileFilter struct {
	Extension    *string
	Status       *string
	TagID        *int64
	ParentDir    *string
	WatchedDirID *int64
	Limit        int
	Offset       int
}

func (s *Store) ListFiles(f FileFilter) ([]File, error) {
	query := `SELECT ` + fileColumns + ` FROM files`
	var args []any
	var conditions []string

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

	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	query += " ORDER BY indexed_at DESC"

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
	err := s.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE id = ?`, id).Scan(
		&f.ID, &f.Path, &f.Name, &f.Extension, &f.MimeType, &f.Size, &f.ParentDir, &f.WatchedDirID, &f.Status, &f.ModifiedAt, &f.IndexedAt, &f.ContentIndexedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func (s *Store) GetFileByPath(path string) (*File, error) {
	var f File
	err := s.db.QueryRow(`SELECT `+fileColumns+` FROM files WHERE path = ?`, path).Scan(
		&f.ID, &f.Path, &f.Name, &f.Extension, &f.MimeType, &f.Size, &f.ParentDir, &f.WatchedDirID, &f.Status, &f.ModifiedAt, &f.IndexedAt, &f.ContentIndexedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
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
			`UPDATE files SET name = ?, extension = ?, mime_type = ?, size = ?, parent_dir = ?, watched_dir_id = ?, status = 'active', modified_at = ?, content_indexed_at = NULL WHERE id = ?`,
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

func (s *Store) ListFilesByDir(dirID int64) ([]File, error) {
	rows, err := s.db.Query(`SELECT `+fileColumns+` FROM files WHERE watched_dir_id = ? ORDER BY name`, dirID)
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

func (s *Store) DeleteFilesByDir(dirID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM files_fts WHERE file_id IN (SELECT id FROM files WHERE watched_dir_id = ?)`, dirID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM files WHERE watched_dir_id = ?`, dirID)
	if err != nil {
		return err
	}

	return tx.Commit()
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
