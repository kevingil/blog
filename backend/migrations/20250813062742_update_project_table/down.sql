-- Drop index first, then columns
DROP INDEX IF EXISTS idx_project_tag_ids;

ALTER TABLE project
  DROP COLUMN IF EXISTS tag_ids,
  DROP COLUMN IF EXISTS content;
