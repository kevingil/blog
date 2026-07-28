-- +goose Up
-- +goose StatementBegin

-- Add user_id to data_source for users without organizations
ALTER TABLE data_source ADD COLUMN user_id UUID REFERENCES account(id) ON DELETE CASCADE;
CREATE INDEX idx_data_source_user ON data_source(user_id);

-- Add subscriber_count for optimization
ALTER TABLE data_source ADD COLUMN subscriber_count INTEGER DEFAULT 1;

-- Make organization_id nullable on insight_topic (allows global topics)
-- Note: organization_id is already nullable in the original migration

-- Make organization_id nullable on insight (allows global insights)
-- Note: organization_id is already nullable in the original migration

-- Create user_insight_status for per-user tracking
CREATE TABLE user_insight_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    insight_id UUID NOT NULL REFERENCES insight(id) ON DELETE CASCADE,
    is_read BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    is_used_in_article BOOLEAN DEFAULT false,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, insight_id)
);

CREATE INDEX idx_user_insight_status_user ON user_insight_status(user_id);
CREATE INDEX idx_user_insight_status_insight ON user_insight_status(insight_id);
CREATE INDEX idx_user_insight_status_unread ON user_insight_status(user_id) WHERE is_read = false;

-- Add constraint: data_source must have either user_id or organization_id
ALTER TABLE data_source ADD CONSTRAINT data_source_owner_check 
    CHECK (user_id IS NOT NULL OR organization_id IS NOT NULL);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

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

-- +goose StatementEnd
