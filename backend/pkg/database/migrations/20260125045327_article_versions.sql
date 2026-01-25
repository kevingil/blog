-- +goose Up
-- +goose StatementBegin

-- Create article_version table for storing version history
CREATE TABLE article_version (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id UUID NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('draft', 'published')),
    title VARCHAR(500) NOT NULL,
    content TEXT,
    image_url TEXT,
    embedding vector(1536),
    edited_by UUID REFERENCES account(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(article_id, version_number)
);

-- Create index for faster version lookups
CREATE INDEX idx_article_version_article_id ON article_version(article_id);
CREATE INDEX idx_article_version_status ON article_version(status);

-- Add new columns to article table for cached draft content
ALTER TABLE article ADD COLUMN draft_title VARCHAR(500);
ALTER TABLE article ADD COLUMN draft_content TEXT;
ALTER TABLE article ADD COLUMN draft_image_url TEXT;
ALTER TABLE article ADD COLUMN draft_embedding vector(1536);

-- Add new columns for cached published content
ALTER TABLE article ADD COLUMN published_title VARCHAR(500);
ALTER TABLE article ADD COLUMN published_content TEXT;
ALTER TABLE article ADD COLUMN published_image_url TEXT;
ALTER TABLE article ADD COLUMN published_embedding vector(1536);

-- Add version pointer columns
ALTER TABLE article ADD COLUMN current_draft_version_id UUID REFERENCES article_version(id);
ALTER TABLE article ADD COLUMN current_published_version_id UUID REFERENCES article_version(id);

-- Migrate existing data: copy content to draft fields
UPDATE article SET
    draft_title = title,
    draft_content = content,
    draft_image_url = image_url,
    draft_embedding = embedding;

-- For published articles (is_draft = false), also copy to published fields
UPDATE article SET
    published_title = title,
    published_content = content,
    published_image_url = image_url,
    published_embedding = embedding
WHERE is_draft = false;

-- Create initial version record for each article
INSERT INTO article_version (id, article_id, version_number, status, title, content, image_url, embedding, created_at)
SELECT 
    uuid_generate_v4(),
    id,
    1,
    CASE WHEN is_draft THEN 'draft' ELSE 'published' END,
    title,
    content,
    image_url,
    embedding,
    created_at
FROM article;

-- Update version pointers
UPDATE article a SET 
    current_draft_version_id = (
        SELECT id FROM article_version av 
        WHERE av.article_id = a.id 
        ORDER BY version_number DESC 
        LIMIT 1
    );

UPDATE article a SET 
    current_published_version_id = (
        SELECT id FROM article_version av 
        WHERE av.article_id = a.id AND av.status = 'published'
        ORDER BY version_number DESC 
        LIMIT 1
    )
WHERE is_draft = false;

-- Drop deprecated columns
ALTER TABLE article DROP COLUMN title;
ALTER TABLE article DROP COLUMN content;
ALTER TABLE article DROP COLUMN image_url;
ALTER TABLE article DROP COLUMN is_draft;
ALTER TABLE article DROP COLUMN embedding;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Re-add original columns
ALTER TABLE article ADD COLUMN title VARCHAR(500);
ALTER TABLE article ADD COLUMN content TEXT;
ALTER TABLE article ADD COLUMN image_url TEXT;
ALTER TABLE article ADD COLUMN is_draft BOOLEAN DEFAULT true;
ALTER TABLE article ADD COLUMN embedding vector(1536);

-- Restore data from draft fields
UPDATE article SET
    title = draft_title,
    content = draft_content,
    image_url = draft_image_url,
    embedding = draft_embedding,
    is_draft = (published_at IS NULL);

-- Drop new columns
ALTER TABLE article DROP COLUMN draft_title;
ALTER TABLE article DROP COLUMN draft_content;
ALTER TABLE article DROP COLUMN draft_image_url;
ALTER TABLE article DROP COLUMN draft_embedding;
ALTER TABLE article DROP COLUMN published_title;
ALTER TABLE article DROP COLUMN published_content;
ALTER TABLE article DROP COLUMN published_image_url;
ALTER TABLE article DROP COLUMN published_embedding;
ALTER TABLE article DROP COLUMN current_draft_version_id;
ALTER TABLE article DROP COLUMN current_published_version_id;

-- Drop indexes and table
DROP INDEX IF EXISTS idx_article_version_article_id;
DROP INDEX IF EXISTS idx_article_version_status;
DROP TABLE IF EXISTS article_version;

-- +goose StatementEnd
