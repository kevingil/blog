DROP INDEX IF EXISTS idx_insight_topic_next_check;
DROP INDEX IF EXISTS idx_crawled_content_topic_url;

ALTER TABLE crawled_content DROP CONSTRAINT IF EXISTS crawled_content_parent_check;
ALTER TABLE crawled_content DROP COLUMN IF EXISTS topic_id;

DELETE FROM crawled_content WHERE data_source_id IS NULL;
ALTER TABLE crawled_content ALTER COLUMN data_source_id SET NOT NULL;

DROP INDEX IF EXISTS idx_insight_data_source;
ALTER TABLE insight DROP COLUMN IF EXISTS data_source_id;

ALTER TABLE insight_topic
    DROP COLUMN IF EXISTS is_enabled,
    DROP COLUMN IF EXISTS next_check_at,
    DROP COLUMN IF EXISTS check_frequency;
