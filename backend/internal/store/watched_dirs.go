package store

import (
	"database/sql"
	"time"
)

type WatchedDir struct {
	ID            int64      `json:"id"`
	Path          string     `json:"path"`
	Recursive     bool       `json:"recursive"`
	Enabled       bool       `json:"enabled"`
	LastScannedAt *time.Time `json:"last_scanned_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (s *Store) ListWatchedDirs() ([]WatchedDir, error) {
	rows, err := s.db.Query(`SELECT id, path, recursive, enabled, last_scanned_at, created_at FROM watched_directories ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dirs []WatchedDir
	for rows.Next() {
		var d WatchedDir
		if err := rows.Scan(&d.ID, &d.Path, &d.Recursive, &d.Enabled, &d.LastScannedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		dirs = append(dirs, d)
	}
	return dirs, rows.Err()
}

func (s *Store) GetWatchedDir(id int64) (*WatchedDir, error) {
	var d WatchedDir
	err := s.db.QueryRow(`SELECT id, path, recursive, enabled, last_scanned_at, created_at FROM watched_directories WHERE id = ?`, id).
		Scan(&d.ID, &d.Path, &d.Recursive, &d.Enabled, &d.LastScannedAt, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

func (s *Store) CreateWatchedDir(path string, recursive bool) (*WatchedDir, error) {
	result, err := s.db.Exec(`INSERT INTO watched_directories (path, recursive) VALUES (?, ?)`, path, recursive)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return s.GetWatchedDir(id)
}

func (s *Store) UpdateWatchedDir(id int64, enabled *bool, recursive *bool) (*WatchedDir, error) {
	if enabled != nil {
		s.db.Exec(`UPDATE watched_directories SET enabled = ? WHERE id = ?`, *enabled, id)
	}
	if recursive != nil {
		s.db.Exec(`UPDATE watched_directories SET recursive = ? WHERE id = ?`, *recursive, id)
	}
	return s.GetWatchedDir(id)
}

func (s *Store) DeleteWatchedDirAndFiles(id int64) error {
	// Get the watched dir path to clean up related guard classifications
	var path string
	err := s.db.QueryRow(`SELECT path FROM watched_directories WHERE id = ?`, id).Scan(&path)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM files_fts WHERE file_id IN (SELECT id FROM files WHERE watched_dir_id = ?)`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM files WHERE watched_dir_id = ?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM folder_guard_classifications WHERE path = ? OR path LIKE ?`, path, path+"/%")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM watched_directories WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) UpdateLastScanned(id int64) error {
	_, err := s.db.Exec(`UPDATE watched_directories SET last_scanned_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}
