package store

import "database/sql"

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Db() *sql.DB {
	return s.db
}
