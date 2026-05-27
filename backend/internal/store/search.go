package store

import (
	"fmt"
	"strings"
)

type SearchFileResult struct {
	FileID       int64    `json:"file_id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Extension    string   `json:"extension"`
	Rank         float64  `json:"rank"`
	Snippet      string   `json:"snippet"`
	MatchSources []string `json:"match_sources"`
}

type SearchNoteResult struct {
	NoteID       int64    `json:"note_id"`
	Title        string   `json:"title"`
	Rank         float64  `json:"rank"`
	Snippet      string   `json:"snippet"`
	MatchSources []string `json:"match_sources"`
}

type SearchResults struct {
	Files []SearchFileResult `json:"files"`
	Notes []SearchNoteResult `json:"notes"`
}

var allScopes = []string{"filenames", "content", "comments", "tags", "notes"}

func parseScopes(scopes string) map[string]bool {
	if scopes == "" {
		result := make(map[string]bool, len(allScopes))
		for _, s := range allScopes {
			result[s] = true
		}
		return result
	}
	result := make(map[string]bool)
	for _, s := range strings.Split(scopes, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s != "" {
			result[s] = true
		}
	}
	return result
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

func (s *Store) searchFilenames(query string, limit int) ([]SearchFileResult, error) {
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, name, path, extension FROM files WHERE name LIKE ? OR path LIKE ? ORDER BY name LIMIT ?`,
		pattern, pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchFileResult
	for rows.Next() {
		var r SearchFileResult
		if err := rows.Scan(&r.FileID, &r.Name, &r.Path, &r.Extension); err != nil {
			return nil, err
		}
		r.MatchSources = []string{"filename"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchFTS(query string, limit int) ([]SearchFileResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT fts.file_id, fts.name, f.path, fts.extension, rank, snippet(files_fts, -1, '...', '...', '...', 32)
		 FROM files_fts fts JOIN files f ON f.id = fts.file_id
		 WHERE files_fts MATCH ? ORDER BY rank LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchFileResult
	for rows.Next() {
		var r SearchFileResult
		if err := rows.Scan(&r.FileID, &r.Name, &r.Path, &r.Extension, &r.Rank, &r.Snippet); err != nil {
			return nil, err
		}
		r.MatchSources = []string{"content"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchComments(query string, limit int) ([]SearchFileResult, error) {
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT f.id, f.name, f.path, f.extension, c.content
		 FROM files f JOIN comments c ON f.id = c.file_id
		 WHERE c.content LIKE ? ORDER BY f.name LIMIT ?`,
		pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchFileResult
	for rows.Next() {
		var r SearchFileResult
		var commentContent string
		if err := rows.Scan(&r.FileID, &r.Name, &r.Path, &r.Extension, &commentContent); err != nil {
			return nil, err
		}
		r.Snippet = truncateStr(commentContent, 200)
		r.MatchSources = []string{"comment"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchFilesByTag(query string, limit int) ([]SearchFileResult, error) {
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT DISTINCT f.id, f.name, f.path, f.extension, t.name
		 FROM files f
		 JOIN file_tags ft ON f.id = ft.file_id
		 JOIN tags t ON ft.tag_id = t.id
		 WHERE t.name LIKE ?
		 ORDER BY f.name LIMIT ?`,
		pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchFileResult
	for rows.Next() {
		var r SearchFileResult
		var tagName string
		if err := rows.Scan(&r.FileID, &r.Name, &r.Path, &r.Extension, &tagName); err != nil {
			return nil, err
		}
		r.Snippet = fmt.Sprintf("tag: %s", tagName)
		r.MatchSources = []string{"tag"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchNotesFTS(query string, limit int) ([]SearchNoteResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT note_id, title, rank, snippet(notes_fts, -1, '...', '...', '...', 32)
		 FROM notes_fts WHERE notes_fts MATCH ? ORDER BY rank LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchNoteResult
	for rows.Next() {
		var r SearchNoteResult
		if err := rows.Scan(&r.NoteID, &r.Title, &r.Rank, &r.Snippet); err != nil {
			return nil, err
		}
		r.MatchSources = []string{"content"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) searchNotesByTag(query string, limit int) ([]SearchNoteResult, error) {
	if limit <= 0 {
		limit = 20
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT DISTINCT n.id, n.title, t.name
		 FROM notes n
		 JOIN note_tags nt ON n.id = nt.note_id
		 JOIN tags t ON nt.tag_id = t.id
		 WHERE t.name LIKE ?
		 ORDER BY n.title LIMIT ?`,
		pattern, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchNoteResult
	for rows.Next() {
		var r SearchNoteResult
		var tagName string
		if err := rows.Scan(&r.NoteID, &r.Title, &tagName); err != nil {
			return nil, err
		}
		r.Snippet = fmt.Sprintf("tag: %s", tagName)
		r.MatchSources = []string{"tag"}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) Search(query string, scopes string, limit int) (*SearchResults, error) {
	if limit <= 0 {
		limit = 20
	}
	active := parseScopes(scopes)

	var allFiles []SearchFileResult
	var allNotes []SearchNoteResult

	if active["filenames"] {
		results, err := s.searchFilenames(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search filenames: %w", err)
		}
		allFiles = append(allFiles, results...)
	}

	if active["content"] {
		results, err := s.searchFTS(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search content: %w", err)
		}
		allFiles = append(allFiles, results...)
	}

	if active["comments"] {
		results, err := s.searchComments(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search comments: %w", err)
		}
		allFiles = append(allFiles, results...)
	}

	if active["tags"] {
		results, err := s.searchFilesByTag(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search file tags: %w", err)
		}
		allFiles = append(allFiles, results...)

		noteResults, err := s.searchNotesByTag(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search note tags: %w", err)
		}
		allNotes = append(allNotes, noteResults...)
	}

	if active["notes"] {
		results, err := s.searchNotesFTS(query, limit)
		if err != nil {
			return nil, fmt.Errorf("search notes: %w", err)
		}
		allNotes = append(allNotes, results...)
	}

	dedupedFiles := dedupeFileResults(allFiles)
	dedupedNotes := dedupeNoteResults(allNotes)

	if dedupedFiles == nil {
		dedupedFiles = []SearchFileResult{}
	}
	if dedupedNotes == nil {
		dedupedNotes = []SearchNoteResult{}
	}

	return &SearchResults{Files: dedupedFiles, Notes: dedupedNotes}, nil
}

func dedupeFileResults(results []SearchFileResult) []SearchFileResult {
	seen := make(map[int64]int)
	var deduped []SearchFileResult
	for _, r := range results {
		if idx, ok := seen[r.FileID]; ok {
			existing := &deduped[idx]
			existing.MatchSources = mergeSources(existing.MatchSources, r.MatchSources)
			if r.Snippet != "" && existing.Snippet == "" {
				existing.Snippet = r.Snippet
			}
			if r.Rank < existing.Rank {
				existing.Rank = r.Rank
			}
		} else {
			seen[r.FileID] = len(deduped)
			deduped = append(deduped, r)
		}
	}
	return deduped
}

func dedupeNoteResults(results []SearchNoteResult) []SearchNoteResult {
	seen := make(map[int64]int)
	var deduped []SearchNoteResult
	for _, r := range results {
		if idx, ok := seen[r.NoteID]; ok {
			existing := &deduped[idx]
			existing.MatchSources = mergeSources(existing.MatchSources, r.MatchSources)
			if r.Snippet != "" && existing.Snippet == "" {
				existing.Snippet = r.Snippet
			}
			if r.Rank < existing.Rank {
				existing.Rank = r.Rank
			}
		} else {
			seen[r.NoteID] = len(deduped)
			deduped = append(deduped, r)
		}
	}
	return deduped
}

func mergeSources(existing, incoming []string) []string {
	set := make(map[string]bool, len(existing))
	for _, s := range existing {
		set[s] = true
	}
	for _, s := range incoming {
		if !set[s] {
			existing = append(existing, s)
			set[s] = true
		}
	}
	return existing
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
