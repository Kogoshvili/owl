ALTER TABLE files ADD COLUMN processing_status TEXT NOT NULL DEFAULT 'unprocessed'
  CHECK (processing_status IN ('unprocessed', 'queued', 'processing', 'processed', 'stale', 'failed'));
ALTER TABLE files ADD COLUMN processing_error TEXT;
CREATE INDEX IF NOT EXISTS idx_files_processing_status ON files(processing_status);
