ALTER TABLE insight_topic
    ADD COLUMN check_frequency VARCHAR(50) NOT NULL DEFAULT 'daily',
    ADD COLUMN next_check_at TIMESTAMPTZ,
    ADD COLUMN is_enabled BOOLEAN NOT NULL DEFAULT true;

UPDATE insight_topic
SET next_check_at = NOW()
WHERE next_check_at IS NULL;

ALTER TABLE insight
    ADD COLUMN data_source_id UUID REFERENCES data_source(id) ON DELETE SET NULL;

CREATE INDEX idx_insight_data_source ON insight(data_source_id);

ALTER TABLE crawled_content
    ALTER COLUMN data_source_id DROP NOT NULL,
    ADD COLUMN topic_id UUID REFERENCES insight_topic(id) ON DELETE CASCADE;

ALTER TABLE crawled_content
    ADD CONSTRAINT crawled_content_parent_check
    CHECK (data_source_id IS NOT NULL OR topic_id IS NOT NULL);

CREATE UNIQUE INDEX idx_crawled_content_topic_url
    ON crawled_content(topic_id, url)
    WHERE topic_id IS NOT NULL;

CREATE INDEX idx_insight_topic_next_check
    ON insight_topic(next_check_at)
    WHERE is_enabled = true;
