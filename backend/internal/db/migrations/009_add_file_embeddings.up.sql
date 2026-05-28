CREATE TABLE IF NOT EXISTS file_embeddings (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    embedding BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (file_id, model)
);
