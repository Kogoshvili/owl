package store

type SearchFileResult struct {
	FileID   int64   `json:"file_id"`
	Name     string  `json:"name"`
	Rank     float64 `json:"rank"`
	Snippet  string  `json:"snippet"`
}

type SearchNoteResult struct {
	NoteID  int64   `json:"note_id"`
	Title   string  `json:"title"`
	Rank    float64 `json:"rank"`
	Snippet string `json:"snippet"`
}

type SearchResults struct {
	Files []SearchFileResult `json:"files"`
	Notes []SearchNoteResult `json:"notes"`
}

func (s *Store) SearchFiles(query string, limit int) ([]SearchFileResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT file_id, name, rank, snippet(files_fts, -1, '...', '...', '...', 32) as snippet
		 FROM files_fts WHERE files_fts MATCH ? ORDER BY rank LIMIT ?`,
		query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchFileResult
	for rows.Next() {
		var r SearchFileResult
		if err := rows.Scan(&r.FileID, &r.Name, &r.Rank, &r.Snippet); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) SearchNotes(query string, limit int) ([]SearchNoteResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT note_id, title, rank, snippet(notes_fts, -1, '...', '...', '...', 32) as snippet
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
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) Search(query string, limit int) (*SearchResults, error) {
	files, err := s.SearchFiles(query, limit)
	if err != nil {
		return nil, err
	}
	notes, err := s.SearchNotes(query, limit)
	if err != nil {
		return nil, err
	}
	return &SearchResults{Files: files, Notes: notes}, nil
}
