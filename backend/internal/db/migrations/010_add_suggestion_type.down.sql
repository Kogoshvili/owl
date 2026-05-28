-- Remove suggestion_type and confidence columns from virtual_folders table
ALTER TABLE virtual_folders DROP COLUMN suggestion_type;
ALTER TABLE virtual_folders DROP COLUMN confidence;