-- Note: This migration is irreversible as it would lose data about which folders were auto vs manual
-- If rollback is needed, the source column can be dropped but auto_generated will not be restored

-- DROP COLUMN is not directly supported in SQLite, need to recreate table
-- For now, leave the source column in place