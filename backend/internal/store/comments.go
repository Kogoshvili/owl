package store

import (
	"database/sql"
	"time"
)

type Comment struct {
	ID        int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) GetComment(fileID int64) (*Comment, error) {
	var c Comment
	err := s.db.QueryRow(`SELECT id, file_id, content, created_at, updated_at FROM comments WHERE file_id = ?`, fileID).
		Scan(&c.ID, &c.FileID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (s *Store) UpsertComment(fileID int64, content string) (*Comment, error) {
	_, err := s.db.Exec(
		`INSERT INTO comments (file_id, content) VALUES (?, ?) ON CONFLICT(file_id) DO UPDATE SET content = excluded.content, updated_at = CURRENT_TIMESTAMP`,
		fileID, content,
	)
	if err != nil {
		return nil, err
	}
	return s.GetComment(fileID)
}

func (s *Store) DeleteComment(fileID int64) error {
	_, err := s.db.Exec(`DELETE FROM comments WHERE file_id = ?`, fileID)
	return err
}
