ALTER TABLE files ADD COLUMN watched_dir_id INTEGER REFERENCES watched_directories(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_files_watched_dir_id ON files(watched_dir_id);
