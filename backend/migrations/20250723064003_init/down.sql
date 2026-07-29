SELECT 'down SQL query';

-- Drop indexes
DROP INDEX IF EXISTS idx_imagen_request_provider;
DROP INDEX IF EXISTS idx_imagen_request_status;
DROP INDEX IF EXISTS idx_file_index_filename;
DROP INDEX IF EXISTS idx_file_index_file_type;
DROP INDEX IF EXISTS idx_file_index_directory_path;
DROP INDEX IF EXISTS idx_article_source_embedding;
DROP INDEX IF EXISTS idx_article_source_article_id;
DROP INDEX IF EXISTS idx_chat_message_article_id;
DROP INDEX IF EXISTS idx_article_tag_ids;
DROP INDEX IF EXISTS idx_article_published_at;
DROP INDEX IF EXISTS idx_article_slug;
DROP INDEX IF EXISTS idx_article_author_id;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS imagen_request;
DROP TABLE IF EXISTS file_index;
DROP TABLE IF EXISTS page;
DROP TABLE IF EXISTS article_source;
DROP TABLE IF EXISTS chat_message;
DROP TABLE IF EXISTS article;
DROP TABLE IF EXISTS tag;
DROP TABLE IF EXISTS account;

-- Drop extensions (optional, might be used by other databases)
-- DROP EXTENSION IF EXISTS "vector";
-- DROP EXTENSION IF EXISTS "uuid-ossp";

