package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Init(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := runMigrations(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return database, nil
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			extension TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			modified_at DATETIME NOT NULL,
			indexed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);

		CREATE TABLE IF NOT EXISTS file_tags (
			file_id INTEGER NOT NULL REFERENCES files(id),
			tag_id INTEGER NOT NULL REFERENCES tags(id),
			PRIMARY KEY (file_id, tag_id)
		);

		CREATE TABLE IF NOT EXISTS virtual_folders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS virtual_folder_files (
			virtual_folder_id INTEGER NOT NULL REFERENCES virtual_folders(id),
			file_id INTEGER NOT NULL REFERENCES files(id),
			PRIMARY KEY (virtual_folder_id, file_id)
		);

		CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS note_tags (
			note_id INTEGER NOT NULL REFERENCES notes(id),
			tag_id INTEGER NOT NULL REFERENCES tags(id),
			PRIMARY KEY (note_id, tag_id)
		);

		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			virtual_folder_id INTEGER REFERENCES virtual_folders(id),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
