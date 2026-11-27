ALTER TABLE mcp_access_token
    DROP COLUMN IF EXISTS refresh_expires_at,
    DROP COLUMN IF EXISTS refresh_token_hash,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS client_name,
    DROP COLUMN IF EXISTS oauth_client_id;

DROP TABLE IF EXISTS oauth_authorization_code;
DROP TABLE IF EXISTS oauth_client;
