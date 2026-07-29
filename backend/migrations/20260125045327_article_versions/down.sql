
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

