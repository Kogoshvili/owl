-- Add suggestion_type and confidence columns to virtual_folders table
ALTER TABLE virtual_folders ADD COLUMN suggestion_type TEXT NOT NULL DEFAULT 'new_folder';
ALTER TABLE virtual_folders ADD COLUMN confidence REAL NOT NULL DEFAULT 0;