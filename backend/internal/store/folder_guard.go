package store

import (
	"database/sql"
	"path/filepath"
	"strings"
)

type FolderGuardClassification struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Guarded     bool   `json:"guarded"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	ClassifiedAt string `json:"classified_at"`
}

func (s *Store) GetFolderGuard(path string) (guarded bool, source string, err error) {
	var guardedInt int
	err = s.db.QueryRow(`SELECT guarded, source FROM folder_guard_classifications WHERE path = ?`, path).Scan(&guardedInt, &source)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	return guardedInt == 1, source, err
}

func (s *Store) SetFolderGuard(path string, guarded bool, source string, reason string) error {
	guardedInt := 0
	if guarded {
		guardedInt = 1
	}
	_, err := s.db.Exec(`INSERT OR REPLACE INTO folder_guard_classifications (path, guarded, reason, source, classified_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`, path, guardedInt, reason, source)
	return err
}

func (s *Store) ListFolderGuards() ([]FolderGuardClassification, error) {
	rows, err := s.db.Query(`SELECT id, path, guarded, reason, source, classified_at FROM folder_guard_classifications ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guards []FolderGuardClassification
	for rows.Next() {
		var g FolderGuardClassification
		var guardedInt int
		if err := rows.Scan(&g.ID, &g.Path, &guardedInt, &g.Reason, &g.Source, &g.ClassifiedAt); err != nil {
			return nil, err
		}
		g.Guarded = guardedInt == 1
		guards = append(guards, g)
	}
	return guards, rows.Err()
}

func (s *Store) DeleteFolderGuard(path string) error {
	_, err := s.db.Exec(`DELETE FROM folder_guard_classifications WHERE path = ?`, path)
	return err
}

func (s *Store) GetGuardedPaths() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT path FROM folder_guard_classifications WHERE guarded = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	guarded := make(map[string]bool)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		guarded[path] = true
	}
	return guarded, rows.Err()
}

func (s *Store) IsFolderGuarded(path string) (bool, error) {
	path = filepath.ToSlash(path)
	path = strings.TrimRight(path, "/")

	guarded, source, err := s.GetFolderGuard(path)
	if err != nil {
		return false, err
	}

	if source == "user" {
		return guarded, nil
	}

	rows, err := s.db.Query(`SELECT path FROM folder_guard_classifications WHERE guarded = 1`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var guardedPath string
		if err := rows.Scan(&guardedPath); err != nil {
			return false, err
		}
		guardedPath = filepath.ToSlash(guardedPath)
		guardedPath = strings.TrimRight(guardedPath, "/")

		if path == guardedPath {
			return true, nil
		}

		if strings.HasPrefix(path, guardedPath+"/") {
			return true, nil
		}
	}

	return false, nil
}