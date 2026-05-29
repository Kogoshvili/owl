-- Rename virtual_folders → folder_suggestions, drop source/materialized/materialized_path
-- Rename virtual_folder_files → folder_suggestion_files, drop source

ALTER TABLE virtual_folders RENAME TO folder_suggestions_old;
ALTER TABLE virtual_folder_files RENAME TO folder_suggestion_files_old;

CREATE TABLE folder_suggestions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    suggestion_type TEXT NOT NULL DEFAULT 'new_folder',
    confidence REAL NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO folder_suggestions (id, name, description, suggestion_type, confidence, created_at)
SELECT id, name, description, suggestion_type, confidence, created_at
FROM folder_suggestions_old;

CREATE TABLE folder_suggestion_files (
    folder_suggestion_id INTEGER NOT NULL REFERENCES folder_suggestions(id) ON DELETE CASCADE,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (folder_suggestion_id, file_id)
);

INSERT INTO folder_suggestion_files (folder_suggestion_id, file_id, added_at)
SELECT virtual_folder_id, file_id, added_at
FROM folder_suggestion_files_old;

DROP TABLE folder_suggestion_files_old;
DROP TABLE folder_suggestions_old;
