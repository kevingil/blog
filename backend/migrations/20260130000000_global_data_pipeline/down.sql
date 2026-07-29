
-- Remove constraint
ALTER TABLE data_source DROP CONSTRAINT IF EXISTS data_source_owner_check;

-- Drop user_insight_status table and indexes
DROP INDEX IF EXISTS idx_user_insight_status_unread;
DROP INDEX IF EXISTS idx_user_insight_status_insight;
DROP INDEX IF EXISTS idx_user_insight_status_user;
DROP TABLE IF EXISTS user_insight_status;

-- Remove columns from data_source
ALTER TABLE data_source DROP COLUMN IF EXISTS subscriber_count;
DROP INDEX IF EXISTS idx_data_source_user;
ALTER TABLE data_source DROP COLUMN IF EXISTS user_id;

