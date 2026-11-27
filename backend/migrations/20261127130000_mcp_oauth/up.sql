-- OAuth clients and authorization codes for MCP sign-in. Access tokens and
-- refresh-token hashes live on mcp_access_token. Plaintext secrets are not stored.
CREATE TABLE oauth_client (
    client_id VARCHAR(36) PRIMARY KEY,
    client_name VARCHAR(200) NOT NULL,
    redirect_uris TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_authorization_code (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code_hash VARCHAR(64) NOT NULL UNIQUE,
    client_id VARCHAR(36) NOT NULL REFERENCES oauth_client(client_id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE INDEX idx_oauth_authorization_code_account_id
    ON oauth_authorization_code (account_id);

ALTER TABLE mcp_access_token
    ADD COLUMN oauth_client_id VARCHAR(36) REFERENCES oauth_client(client_id) ON DELETE SET NULL,
    ADD COLUMN client_name VARCHAR(200),
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN refresh_token_hash VARCHAR(64) UNIQUE,
    ADD COLUMN refresh_expires_at TIMESTAMPTZ;
