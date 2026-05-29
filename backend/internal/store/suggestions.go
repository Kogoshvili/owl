package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const suggestionColumns = `id, name, description, suggestion_type, confidence, materialized_at, materialized_path, created_at`

type FolderSuggestion struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	SuggestionType   string     `json:"suggestion_type"`
	Confidence       float64    `json:"confidence"`
	MaterializedAt   *time.Time `json:"materialized_at"`
	MaterializedPath string     `json:"materialized_path"`
	CreatedAt        time.Time  `json:"created_at"`
}

type FolderSuggestionDetail struct {
	FolderSuggestion
	Files []File `json:"files"`
}

func (s *Store) ListSuggestions() ([]FolderSuggestion, error) {
	rows, err := s.db.Query(`SELECT ` + suggestionColumns + ` FROM folder_suggestions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []FolderSuggestion
	for rows.Next() {
		var sug FolderSuggestion
		if err := scanSuggestion(rows, &sug); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, sug)
	}
	return suggestions, rows.Err()
}

func (s *Store) GetSuggestion(id int64) (*FolderSuggestion, error) {
	var sug FolderSuggestion
	err := s.db.QueryRow(`SELECT `+suggestionColumns+` FROM folder_suggestions WHERE id = ?`, id).
		Scan(&sug.ID, &sug.Name, &sug.Description, &sug.SuggestionType, &sug.Confidence, &sug.MaterializedAt, &sug.MaterializedPath, &sug.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &sug, err
}

func scanSuggestion(row interface{ Scan(...any) error }, sug *FolderSuggestion) error {
	return row.Scan(&sug.ID, &sug.Name, &sug.Description, &sug.SuggestionType, &sug.Confidence, &sug.MaterializedAt, &sug.MaterializedPath, &sug.CreatedAt)
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

type MaterializeResult struct {
	SuggestionID int64    `json:"suggestion_id"`
	FolderPath   string   `json:"folder_path"`
	Moved        int      `json:"moved"`
	Failed       []string `json:"failed,omitempty"`
}

func (s *Store) MaterializeSuggestion(id int64, destBase string) (*MaterializeResult, error) {
	sug, err := s.GetSuggestion(id)
	if err != nil {
		return nil, fmt.Errorf("get suggestion: %w", err)
	}
	if sug == nil {
		return nil, fmt.Errorf("suggestion not found")
	}
	if sug.MaterializedAt != nil {
		return nil, fmt.Errorf("suggestion already materialized at %s", sug.MaterializedPath)
	}

	files, err := s.ListSuggestionFiles(id)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to materialize")
	}

	folderPath := filepath.Join(destBase, sanitizeDirName(sug.Name))
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	result := &MaterializeResult{
		SuggestionID: id,
		FolderPath:   folderPath,
	}

	for _, f := range files {
		src := f.Path
		dst := filepath.Join(folderPath, f.Name)

		// handle name collisions
		if _, err := os.Stat(dst); err == nil {
			ext := filepath.Ext(f.Name)
			base := strings.TrimSuffix(f.Name, ext)
			for i := 1; ; i++ {
				dst = filepath.Join(folderPath, fmt.Sprintf("%s_%d%s", base, i, ext))
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					break
				}
			}
		}

		if err := os.Rename(src, dst); err != nil {
			result.Failed = append(result.Failed, f.Name)
			continue
		}
		result.Moved++
	}

	now := time.Now()
	if _, err := s.db.Exec(`UPDATE folder_suggestions SET materialized_at = ?, materialized_path = ? WHERE id = ?`,
		now, folderPath, id); err != nil {
		return nil, fmt.Errorf("update suggestion: %w", err)
	}

	return result, nil
}

func sanitizeDirName(name string) string {
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return strings.TrimSpace(r.Replace(name))
}

func joinConditions(conditions []string, sep string) string {
	return strings.Join(conditions, sep)
}
