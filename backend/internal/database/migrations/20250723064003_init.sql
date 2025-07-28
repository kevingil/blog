-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Users table
CREATE TABLE account (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tags table
CREATE TABLE tag (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Articles table with tag_ids array and chat functionality
CREATE TABLE article (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    image_url TEXT,
    author_id UUID NOT NULL REFERENCES account(id),
    tag_ids INTEGER[] DEFAULT '{}',
    is_draft BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    imagen_request_id UUID,
    embedding vector(1536),
    session_memory JSONB DEFAULT '{}'
);

-- Sources table for citations with embeddings
CREATE TABLE article_source (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id UUID NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    title VARCHAR(500),
    content TEXT NOT NULL,
    url TEXT,
    source_type VARCHAR(50) DEFAULT 'web',
    embedding vector(1536),
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chat messages table for article copilot
CREATE TABLE chat_message (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id UUID NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Pages table
CREATE TABLE page (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT,
    description TEXT,
    image_url TEXT,
    meta_data JSONB DEFAULT '{}',
    is_published BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(slug)
);

-- File index table for S3 object tracking
CREATE TABLE file_index (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    s3_key TEXT NOT NULL UNIQUE,
    filename TEXT NOT NULL,
    directory_path TEXT DEFAULT '',
    file_type VARCHAR(100),
    file_size BIGINT,
    content_type VARCHAR(100),
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Image generation requests table
CREATE TABLE imagen_request (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    prompt TEXT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    request_id VARCHAR(255) UNIQUE,
    status VARCHAR(20) DEFAULT 'pending',
    output_url TEXT,
    file_index_id UUID REFERENCES file_index(id),
    error_message TEXT,
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Projects table
CREATE TABLE project (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    image_url TEXT,
    url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_article_author_id ON article(author_id);
CREATE INDEX idx_article_slug ON article(slug);
CREATE INDEX idx_article_published_at ON article(published_at);
CREATE INDEX idx_article_tag_ids ON article USING GIN(tag_ids);
CREATE INDEX idx_chat_message_article_id ON chat_message(article_id, created_at);
CREATE INDEX idx_article_source_article_id ON article_source(article_id);
CREATE INDEX idx_article_source_embedding ON article_source USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_file_index_directory_path ON file_index(directory_path);
CREATE INDEX idx_file_index_file_type ON file_index(file_type);
CREATE INDEX idx_file_index_filename ON file_index(filename);
CREATE INDEX idx_imagen_request_status ON imagen_request(status);
CREATE INDEX idx_imagen_request_provider ON imagen_request(provider);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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

-- +goose StatementEnd
