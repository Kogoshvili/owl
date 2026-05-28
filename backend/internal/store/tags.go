package store

import (
	"database/sql"
	"time"
)

type Tag struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TagWithCount struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	FileCount   int64     `json:"file_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type FileTag struct {
	FileID int64  `json:"file_id"`
	TagID  int64  `json:"tag_id"`
	Source string `json:"source"`
}

func (s *Store) UpdateTagSource(id int64, source string) (*Tag, error) {
	_, err := s.db.Exec(`UPDATE tags SET source = ? WHERE id = ?`, source, id)
	if err != nil {
		return nil, err
	}
	return s.GetTag(id)
}

func (s *Store) UpdateTagDescription(id int64, description string) error {
	_, err := s.db.Exec(`UPDATE tags SET description = ? WHERE id = ?`, description, id)
	return err
}

func (s *Store) ListTagsWithCount(source *string) ([]TagWithCount, error) {
	var query string
	var args []any

	if source != nil {
		query = `SELECT t.id, t.name, t.source, t.description, COUNT(ft.file_id) as file_count, t.created_at FROM tags t LEFT JOIN file_tags ft ON t.id = ft.tag_id WHERE t.source = ? GROUP BY t.id ORDER BY file_count DESC`
		args = []any{*source}
	} else {
		query = `SELECT t.id, t.name, t.source, t.description, COUNT(ft.file_id) as file_count, t.created_at FROM tags t LEFT JOIN file_tags ft ON t.id = ft.tag_id GROUP BY t.id ORDER BY file_count DESC`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagWithCount
	for rows.Next() {
		var t TagWithCount
		if err := rows.Scan(&t.ID, &t.Name, &t.Source, &t.Description, &t.FileCount, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) ListTags() ([]Tag, error) {
	rows, err := s.db.Query(`SELECT id, name, source, description, created_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Source, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *Store) GetTag(id int64) (*Tag, error) {
	var t Tag
	err := s.db.QueryRow(`SELECT id, name, source, description, created_at FROM tags WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Source, &t.Description, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s *Store) GetTagByName(name string) (*Tag, error) {
	var t Tag
	err := s.db.QueryRow(`SELECT id, name, source, description, created_at FROM tags WHERE name = ?`, name).
		Scan(&t.ID, &t.Name, &t.Source, &t.Description, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

func (s *Store) CreateTag(name string, source string, description string) (*Tag, error) {
	result, err := s.db.Exec(`INSERT INTO tags (name, source, description) VALUES (?, ?, ?)`, name, source, description)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetTag(id)
}

func (s *Store) EnsureTag(name string, source string) (*Tag, error) {
	t, err := s.GetTagByName(name)
	if err != nil {
		return nil, err
	}
	if t != nil {
		return t, nil
	}
	return s.CreateTag(name, source, "")
}

func (s *Store) DeleteTag(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	return err
}

func (s *Store) UpdateTagName(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *Store) GetFilesByTag(tagID int64) ([]File, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.path, f.name, f.extension, f.mime_type, f.size, f.parent_dir, f.watched_dir_id, f.status, f.modified_at, f.indexed_at, f.content_indexed_at, f.processing_status, f.processing_error, f.file_metadata
		FROM files f
		JOIN file_tags ft ON f.id = ft.file_id
		WHERE ft.tag_id = ?
	`, tagID)
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

func (s *Store) AddFileTag(fileID, tagID int64, source string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO file_tags (file_id, tag_id, source) VALUES (?, ?, ?)`, fileID, tagID, source)
	return err
}

func (s *Store) CountFilesByTag(tagID int64) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM file_tags WHERE tag_id = ?`, tagID).Scan(&count)
	return count, err
}

func (s *Store) RemoveFileTag(fileID, tagID int64) error {
	_, err := s.db.Exec(`DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?`, fileID, tagID)
	return err
}

func (s *Store) ListFileTags(fileID int64) ([]Tag, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.source, t.description, t.created_at FROM tags t JOIN file_tags ft ON t.id = ft.tag_id WHERE ft.file_id = ? ORDER BY t.name`,
		fileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Source, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
