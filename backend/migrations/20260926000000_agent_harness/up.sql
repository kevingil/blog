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
