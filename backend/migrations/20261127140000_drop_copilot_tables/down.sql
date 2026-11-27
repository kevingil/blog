-- Recreate the structures removed by up.sql, copied from the migrations that
-- introduced them: 20250723064003_init, 20260926000000_agent_harness, and
-- 20260926000001_agent_skills.
ALTER TABLE article ADD COLUMN session_memory JSONB DEFAULT '{}';

CREATE TABLE chat_message (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id UUID NOT NULL REFERENCES article(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    meta_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_chat_message_article_id ON chat_message(article_id, created_at);

CREATE TABLE mcp_connector (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    transport VARCHAR(20) NOT NULL,
    command TEXT,
    args JSONB NOT NULL DEFAULT '[]',
    url TEXT,
    headers JSONB NOT NULL DEFAULT '{}',
    env JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_mcp_connector_enabled ON mcp_connector (enabled);

CREATE TABLE agent_skill (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    instructions TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_agent_skill_name ON agent_skill (lower(name));
CREATE INDEX idx_agent_skill_enabled ON agent_skill (enabled);
