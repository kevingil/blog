
-- Drop indexes
DROP INDEX IF EXISTS idx_insight_unread;
DROP INDEX IF EXISTS idx_insight_org;
DROP INDEX IF EXISTS idx_insight_generated;
DROP INDEX IF EXISTS idx_insight_topic;
DROP INDEX IF EXISTS idx_insight_embedding;

DROP INDEX IF EXISTS idx_content_topic_primary;
DROP INDEX IF EXISTS idx_content_topic_topic;
DROP INDEX IF EXISTS idx_content_topic_content;

DROP INDEX IF EXISTS idx_crawled_content_published;
DROP INDEX IF EXISTS idx_crawled_content_created;
DROP INDEX IF EXISTS idx_crawled_content_data_source;
DROP INDEX IF EXISTS idx_crawled_content_embedding;

DROP INDEX IF EXISTS idx_insight_topic_org;
DROP INDEX IF EXISTS idx_insight_topic_embedding;

DROP INDEX IF EXISTS idx_data_source_url;
DROP INDEX IF EXISTS idx_data_source_status;
DROP INDEX IF EXISTS idx_data_source_next_crawl;
DROP INDEX IF EXISTS idx_data_source_org;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS insight;
DROP TABLE IF EXISTS content_topic_match;
DROP TABLE IF EXISTS crawled_content;
DROP TABLE IF EXISTS insight_topic;
DROP TABLE IF EXISTS data_source;

