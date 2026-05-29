package store

import (
	"database/sql"
	"strings"
	"time"
)

type FolderSuggestion struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	SuggestionType string    `json:"suggestion_type"`
	Confidence     float64   `json:"confidence"`
	CreatedAt      time.Time `json:"created_at"`
}

type FolderSuggestionDetail struct {
	FolderSuggestion
	Files []File `json:"files"`
}

func (s *Store) ListSuggestions() ([]FolderSuggestion, error) {
	rows, err := s.db.Query(`SELECT id, name, description, suggestion_type, confidence, created_at FROM folder_suggestions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []FolderSuggestion
	for rows.Next() {
		var sug FolderSuggestion
		if err := rows.Scan(&sug.ID, &sug.Name, &sug.Description, &sug.SuggestionType, &sug.Confidence, &sug.CreatedAt); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, sug)
	}
	return suggestions, rows.Err()
}

func (s *Store) GetSuggestion(id int64) (*FolderSuggestion, error) {
	var sug FolderSuggestion
	err := s.db.QueryRow(`SELECT id, name, description, suggestion_type, confidence, created_at FROM folder_suggestions WHERE id = ?`, id).
		Scan(&sug.ID, &sug.Name, &sug.Description, &sug.SuggestionType, &sug.Confidence, &sug.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sug, err
}

func (s *Store) CreateSuggestion(name, description, suggestionType string, confidence float64) (*FolderSuggestion, error) {
	result, err := s.db.Exec(`INSERT INTO folder_suggestions (name, description, suggestion_type, confidence) VALUES (?, ?, ?, ?)`, name, description, suggestionType, confidence)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetSuggestion(id)
}

func (s *Store) UpdateSuggestion(id int64, name *string, description *string) (*FolderSuggestion, error) {
	if name != nil {
		s.db.Exec(`UPDATE folder_suggestions SET name = ? WHERE id = ?`, *name, id)
	}
	if description != nil {
		s.db.Exec(`UPDATE folder_suggestions SET description = ? WHERE id = ?`, *description, id)
	}
	return s.GetSuggestion(id)
}

func (s *Store) DeleteSuggestion(id int64) error {
	_, err := s.db.Exec(`DELETE FROM folder_suggestions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteAllSuggestions() error {
	_, err := s.db.Exec(`DELETE FROM folder_suggestions`)
	return err
}

func (s *Store) AddFileToSuggestion(suggestionID, fileID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO folder_suggestion_files (folder_suggestion_id, file_id) VALUES (?, ?)`, suggestionID, fileID)
	return err
}

func (s *Store) AddFilesToSuggestion(suggestionID int64, fileIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO folder_suggestion_files (folder_suggestion_id, file_id) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, fileID := range fileIDs {
		if _, err := stmt.Exec(suggestionID, fileID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RemoveFileFromSuggestion(suggestionID, fileID int64) error {
	_, err := s.db.Exec(`DELETE FROM folder_suggestion_files WHERE folder_suggestion_id = ? AND file_id = ?`, suggestionID, fileID)
	return err
}

func (s *Store) ListSuggestionFiles(suggestionID int64) ([]File, error) {
	rows, err := s.db.Query(
		`SELECT f.`+fileColumns+`
		 FROM files f JOIN folder_suggestion_files fsf ON f.id = fsf.file_id WHERE fsf.folder_suggestion_id = ? ORDER BY f.name`,
		suggestionID,
	)
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

func (s *Store) GetSuggestionDetail(id int64) (*FolderSuggestionDetail, error) {
	suggestion, err := s.GetSuggestion(id)
	if err != nil || suggestion == nil {
		return nil, err
	}

	files, err := s.ListSuggestionFiles(id)
	if err != nil {
		return nil, err
	}

	return &FolderSuggestionDetail{
		FolderSuggestion: *suggestion,
		Files:            files,
	}, nil
}

func joinConditions(conditions []string, sep string) string {
	return strings.Join(conditions, sep)
}
