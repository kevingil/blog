-- +goose Up
-- +goose StatementBegin
ALTER TABLE project
  ADD COLUMN IF NOT EXISTS content TEXT,
  ADD COLUMN IF NOT EXISTS tag_ids INTEGER[] DEFAULT '{}';

-- Create index for tag_ids for faster search/filtering
CREATE INDEX IF NOT EXISTS idx_project_tag_ids ON project USING GIN(tag_ids);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop index first, then columns
DROP INDEX IF EXISTS idx_project_tag_ids;

ALTER TABLE project
  DROP COLUMN IF EXISTS tag_ids,
  DROP COLUMN IF EXISTS content;
-- +goose StatementEnd


