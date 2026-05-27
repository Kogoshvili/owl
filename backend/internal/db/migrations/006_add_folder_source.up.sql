-- Add source column to virtual_folders table to align with tags table
ALTER TABLE virtual_folders ADD COLUMN source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('auto', 'manual'));

-- Migrate existing data: auto_generated=1 -> source='auto', auto_generated=0 -> source='manual'
UPDATE virtual_folders SET source = 'auto' WHERE auto_generated = 1;
UPDATE virtual_folders SET source = 'manual' WHERE auto_generated = 0;

-- Note: auto_generated column is kept for backwards compatibility
-- New code should use source column exclusively