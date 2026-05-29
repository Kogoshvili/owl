package store

import (
	"path/filepath"
	"sort"
	"strings"
)

type PhysicalFolder struct {
	Path     string            `json:"path"`
	Name     string            `json:"name"`
	Depth    int               `json:"depth"`
	FileCount int              `json:"file_count"`
	Children  []*PhysicalFolder `json:"children,omitempty"`
}

func (s *Store) ListPhysicalFolders(watchedDirID int64) ([]*PhysicalFolder, error) {
	rows, err := s.db.Query(`
		SELECT REPLACE(parent_dir, '\', '/') as parent_dir, COUNT(*) as cnt
		FROM files
		WHERE watched_dir_id = ? AND status = 'active'
		GROUP BY REPLACE(parent_dir, '\', '/')
		ORDER BY parent_dir
	`, watchedDirID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type dirInfo struct {
		path      string
		fileCount int
	}
	dirs := make(map[string]int)
	for rows.Next() {
		var dir string
		var cnt int
		if err := rows.Scan(&dir, &cnt); err != nil {
			return nil, err
		}
		dir = filepath.ToSlash(dir)
		dir = strings.TrimRight(dir, "/")
		dirs[dir] = cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var watchedDir *WatchedDir
	wdRows, err := s.db.Query(`SELECT id, path, recursive, enabled, last_scanned_at, created_at FROM watched_directories WHERE id = ?`, watchedDirID)
	if err != nil {
		return nil, err
	}
	defer wdRows.Close()
	if wdRows.Next() {
		var wd WatchedDir
		if err := wdRows.Scan(&wd.ID, &wd.Path, &wd.Recursive, &wd.Enabled, &wd.LastScannedAt, &wd.CreatedAt); err != nil {
			return nil, err
		}
		watchedDir = &wd
	}
	if watchedDir == nil {
		return nil, nil
	}

	rootPath := filepath.ToSlash(watchedDir.Path)
	rootPath = strings.TrimRight(rootPath, "/")
	rootCount := 0
	if cnt, ok := dirs[rootPath]; ok {
		rootCount = cnt
	}

	allPaths := make(map[string]*PhysicalFolder)
	for dir := range dirs {
		rel, err := filepath.Rel(rootPath, dir)
		if err != nil || rel == "." {
			continue
		}
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		for i := 1; i <= len(parts); i++ {
			subPath := strings.Join(parts[:i], "/")
			fullPath := filepath.ToSlash(filepath.Join(rootPath, subPath))
			if _, exists := allPaths[fullPath]; !exists {
				allPaths[fullPath] = &PhysicalFolder{
					Path:  fullPath,
					Name:  parts[i-1],
					Depth: i,
				}
			}
		}
	}

	for dir, cnt := range dirs {
		if dir == rootPath {
			continue
		}
		if folder, ok := allPaths[dir]; ok {
			folder.FileCount = cnt
		}
	}

	childrenMap := make(map[string][]*PhysicalFolder)
	for _, folder := range allPaths {
		parentDir := filepath.ToSlash(filepath.Dir(folder.Path))
		if parentDir == rootPath || parentDir == folder.Path {
			childrenMap[rootPath] = append(childrenMap[rootPath], folder)
		} else {
			childrenMap[parentDir] = append(childrenMap[parentDir], folder)
		}
	}

	for _, children := range childrenMap {
		sort.Slice(children, func(i, j int) bool {
			return children[i].Name < children[j].Name
		})
	}

	var buildChildren func(parentPath string) []*PhysicalFolder
	buildChildren = func(parentPath string) []*PhysicalFolder {
		children := childrenMap[parentPath]
		for _, child := range children {
			child.Children = buildChildren(child.Path)
		}
		return children
	}

	root := &PhysicalFolder{
		Path:      rootPath,
		Name:      filepath.Base(rootPath),
		Depth:     0,
		FileCount: rootCount,
		Children:  buildChildren(rootPath),
	}

	return []*PhysicalFolder{root}, nil
}

func (s *Store) ListPhysicalFoldersAll() ([]*PhysicalFolder, error) {
	wdRows, err := s.db.Query(`SELECT id, path, recursive, enabled, last_scanned_at, created_at FROM watched_directories ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer wdRows.Close()

	var watchedDirs []WatchedDir
	for wdRows.Next() {
		var wd WatchedDir
		if err := wdRows.Scan(&wd.ID, &wd.Path, &wd.Recursive, &wd.Enabled, &wd.LastScannedAt, &wd.CreatedAt); err != nil {
			return nil, err
		}
		watchedDirs = append(watchedDirs, wd)
	}

	var allTrees []*PhysicalFolder
	for _, wd := range watchedDirs {
		tree, err := s.ListPhysicalFolders(wd.ID)
		if err != nil {
			continue
		}
		allTrees = append(allTrees, tree...)
	}

	return allTrees, nil
}

func (s *Store) GetFilesInDir(parentDir string) ([]File, error) {
	parentDir = strings.TrimRight(parentDir, "/")
	source := "active"
	parentDirWin := strings.ReplaceAll(parentDir, "/", "\\")

	query := `
		SELECT ` + fileColumns + `
		FROM files
		WHERE (parent_dir = ? OR parent_dir = ?) AND status = ?
		ORDER BY id DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, parentDir, parentDirWin, source, 10000)
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

func (s *Store) GetFileNames(fileIDs []int64) (map[int64]string, error) {
	if len(fileIDs) == 0 {
		return make(map[int64]string), nil
	}

	placeholders := make([]string, len(fileIDs))
	args := make([]any, len(fileIDs))
	for i, id := range fileIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id, name FROM files WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}
	return result, rows.Err()
}
