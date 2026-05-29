CREATE TABLE IF NOT EXISTS watched_directories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_scanned_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    extension TEXT NOT NULL DEFAULT '',
    mime_type TEXT NOT NULL DEFAULT '',
    size INTEGER NOT NULL DEFAULT 0,
    parent_dir TEXT NOT NULL,
    watched_dir_id INTEGER REFERENCES watched_directories(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'missing', 'deleted')),
    modified_at DATETIME NOT NULL,
    indexed_at DATETIME,
    content_indexed_at DATETIME,
    processing_status TEXT NOT NULL DEFAULT 'unprocessed' CHECK (processing_status IN ('unprocessed', 'queued', 'processing', 'processed', 'stale', 'failed')),
    processing_error TEXT,
    file_metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_files_parent_dir ON files(parent_dir);
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_extension ON files(extension);
CREATE INDEX IF NOT EXISTS idx_files_watched_dir_id ON files(watched_dir_id);
CREATE INDEX IF NOT EXISTS idx_files_processing_status ON files(processing_status);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL UNIQUE REFERENCES files(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS file_embeddings (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    embedding BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (file_id, model)
);

CREATE TABLE IF NOT EXISTS folder_guard_classifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    guarded INTEGER NOT NULL DEFAULT 0,
    reason TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'llm' CHECK (source IN ('llm', 'user')),
    classified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_folder_guard_path ON folder_guard_classifications(path);
CREATE INDEX IF NOT EXISTS idx_folder_guard_guarded ON folder_guard_classifications(guarded);
CREATE INDEX IF NOT EXISTS idx_folder_guard_source ON folder_guard_classifications(source);

CREATE TABLE IF NOT EXISTS folder_suggestions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    suggestion_type TEXT NOT NULL DEFAULT 'new_folder',
    confidence REAL NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS folder_suggestion_files (
    folder_suggestion_id INTEGER NOT NULL REFERENCES folder_suggestions(id) ON DELETE CASCADE,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (folder_suggestion_id, file_id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
    file_id UNINDEXED,
    name,
    extension,
    content
);
