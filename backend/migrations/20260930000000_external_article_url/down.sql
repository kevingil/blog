DROP INDEX IF EXISTS article_external_url_key;

ALTER TABLE article DROP COLUMN IF EXISTS external_url;
