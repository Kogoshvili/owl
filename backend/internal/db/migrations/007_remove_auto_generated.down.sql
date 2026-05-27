ALTER TABLE virtual_folders ADD COLUMN auto_generated INTEGER NOT NULL DEFAULT 0;
UPDATE virtual_folders SET auto_generated = CASE WHEN source = 'auto' THEN 1 ELSE 0 END;
