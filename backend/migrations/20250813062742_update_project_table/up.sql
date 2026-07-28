ALTER TABLE project
  ADD COLUMN IF NOT EXISTS content TEXT,
  ADD COLUMN IF NOT EXISTS tag_ids INTEGER[] DEFAULT '{}';

-- Create index for tag_ids for faster search/filtering
CREATE INDEX IF NOT EXISTS idx_project_tag_ids ON project USING GIN(tag_ids);
