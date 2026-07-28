-- +goose Up
-- +goose StatementBegin

-- Data Source table - User's preferred websites to crawl
CREATE TABLE data_source (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    feed_url TEXT,
    source_type VARCHAR(50) DEFAULT 'blog',
    crawl_frequency VARCHAR(50) DEFAULT 'daily',
    is_enabled BOOLEAN DEFAULT true,
    is_discovered BOOLEAN DEFAULT false,
    discovered_from_id UUID REFERENCES data_source(id) ON DELETE SET NULL,
    last_crawled_at TIMESTAMPTZ,
    next_crawl_at TIMESTAMPTZ,
    crawl_status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,
    content_count INTEGER DEFAULT 0,
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_data_source_org ON data_source(organization_id);
CREATE INDEX idx_data_source_next_crawl ON data_source(next_crawl_at) WHERE is_enabled = true;
CREATE INDEX idx_data_source_status ON data_source(crawl_status);
CREATE INDEX idx_data_source_url ON data_source(url);

-- Insight Topic table - Topic categories with embeddings for semantic matching
CREATE TABLE insight_topic (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    keywords JSONB DEFAULT '[]',
    embedding vector(1536),
    is_auto_generated BOOLEAN DEFAULT false,
    content_count INTEGER DEFAULT 0,
    last_insight_at TIMESTAMPTZ,
    color VARCHAR(20),
    icon VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_insight_topic_embedding ON insight_topic 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_insight_topic_org ON insight_topic(organization_id);

-- Crawled Content table - Content from crawled sources
CREATE TABLE crawled_content (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_source_id UUID NOT NULL REFERENCES data_source(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    title VARCHAR(500),
    content TEXT NOT NULL,
    summary TEXT,
    author VARCHAR(255),
    published_at TIMESTAMPTZ,
    embedding vector(1536),
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(data_source_id, url)
);

CREATE INDEX idx_crawled_content_embedding ON crawled_content 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_crawled_content_data_source ON crawled_content(data_source_id);
CREATE INDEX idx_crawled_content_created ON crawled_content(created_at DESC);
CREATE INDEX idx_crawled_content_published ON crawled_content(published_at DESC);

-- Content Topic Match table - Junction table with similarity scores
CREATE TABLE content_topic_match (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES crawled_content(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES insight_topic(id) ON DELETE CASCADE,
    similarity_score FLOAT NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(content_id, topic_id)
);

CREATE INDEX idx_content_topic_content ON content_topic_match(content_id);
CREATE INDEX idx_content_topic_topic ON content_topic_match(topic_id);
CREATE INDEX idx_content_topic_primary ON content_topic_match(topic_id) WHERE is_primary = true;

-- Insight table - Generated insights/mini-blogs
CREATE TABLE insight (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    topic_id UUID REFERENCES insight_topic(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    summary TEXT NOT NULL,
    content TEXT,
    key_points JSONB DEFAULT '[]',
    source_content_ids UUID[] DEFAULT '{}',
    embedding vector(1536),
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    is_read BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    is_used_in_article BOOLEAN DEFAULT false,
    meta_data JSONB DEFAULT '{}'
);

CREATE INDEX idx_insight_embedding ON insight 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_insight_topic ON insight(topic_id);
CREATE INDEX idx_insight_generated ON insight(generated_at DESC);
CREATE INDEX idx_insight_org ON insight(organization_id);
CREATE INDEX idx_insight_unread ON insight(organization_id) WHERE is_read = false;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

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

-- +goose StatementEnd
