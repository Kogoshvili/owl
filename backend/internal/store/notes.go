package store

import (
	"database/sql"
	"time"
)

type Note struct {
	ID               int64      `json:"id"`
	Title            string     `json:"title"`
	Content          string     `json:"content"`
	Materialized     bool       `json:"materialized"`
	MaterializedPath *string    `json:"materialized_path"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s *Store) ListNotes() ([]Note, error) {
	rows, err := s.db.Query(`SELECT id, title, content, materialized, materialized_path, created_at, updated_at FROM notes ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Materialized, &n.MaterializedPath, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) GetNote(id int64) (*Note, error) {
	var n Note
	err := s.db.QueryRow(`SELECT id, title, content, materialized, materialized_path, created_at, updated_at FROM notes WHERE id = ?`, id).
		Scan(&n.ID, &n.Title, &n.Content, &n.Materialized, &n.MaterializedPath, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &n, err
}

func (s *Store) CreateNote(title, content string) (*Note, error) {
	result, err := s.db.Exec(`INSERT INTO notes (title, content) VALUES (?, ?)`, title, content)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetNote(id)
}

func (s *Store) UpdateNote(id int64, title *string, content *string) (*Note, error) {
	if title != nil {
		s.db.Exec(`UPDATE notes SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *title, id)
	}
	if content != nil {
		s.db.Exec(`UPDATE notes SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *content, id)
	}
	return s.GetNote(id)
}

func (s *Store) DeleteNote(id int64) error {
	_, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

func (s *Store) SetNoteMaterialized(id int64, path string) error {
	_, err := s.db.Exec(`UPDATE notes SET materialized = 1, materialized_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, path, id)
	return err
}

func (s *Store) AttachNoteToFolder(noteID, folderID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO note_virtual_folders (note_id, virtual_folder_id) VALUES (?, ?)`, noteID, folderID)
	return err
}

func (s *Store) DetachNoteFromFolder(noteID, folderID int64) error {
	_, err := s.db.Exec(`DELETE FROM note_virtual_folders WHERE note_id = ? AND virtual_folder_id = ?`, noteID, folderID)
	return err
}

func (s *Store) ListFolderNotes(folderID int64) ([]Note, error) {
	rows, err := s.db.Query(
		`SELECT n.id, n.title, n.content, n.materialized, n.materialized_path, n.created_at, n.updated_at
		 FROM notes n JOIN note_virtual_folders nvf ON n.id = nvf.note_id WHERE nvf.virtual_folder_id = ? ORDER BY n.updated_at DESC`,
		folderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Materialized, &n.MaterializedPath, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) ListNoteFolders(noteID int64) ([]VirtualFolder, error) {
	rows, err := s.db.Query(
		`SELECT vf.id, vf.name, vf.description, vf.source, vf.materialized, vf.materialized_path, vf.created_at
		 FROM virtual_folders vf JOIN note_virtual_folders nvf ON vf.id = nvf.virtual_folder_id WHERE nvf.note_id = ?`,
		noteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []VirtualFolder
	for rows.Next() {
		var f VirtualFolder
		if err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Source, &f.Materialized, &f.MaterializedPath, &f.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}
