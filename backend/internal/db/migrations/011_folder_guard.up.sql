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