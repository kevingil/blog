use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::{
    core::auth::{AccountId, McpAccessToken, NewMcpAccessToken},
    schema::mcp_access_token,
};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = mcp_access_token)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct McpAccessTokenRow {
    pub id: Uuid,
    pub account_id: Uuid,
    pub name: String,
    pub token_hash: String,
    pub token_prefix: String,
    pub created_at: DateTime<Utc>,
    pub last_used_at: Option<DateTime<Utc>>,
    pub revoked_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Insertable)]
#[diesel(table_name = mcp_access_token)]
pub struct NewMcpAccessTokenRow {
    pub id: Uuid,
    pub account_id: Uuid,
    pub name: String,
    pub token_hash: String,
    pub token_prefix: String,
    pub created_at: DateTime<Utc>,
}

impl From<&NewMcpAccessToken> for NewMcpAccessTokenRow {
    fn from(token: &NewMcpAccessToken) -> Self {
        Self {
            id: token.id,
            account_id: token.account_id.0,
            name: token.name.clone(),
            token_hash: token.token_hash.clone(),
            token_prefix: token.token_prefix.clone(),
            created_at: token.created_at,
        }
    }
}

impl From<McpAccessTokenRow> for McpAccessToken {
    fn from(row: McpAccessTokenRow) -> Self {
        Self {
            id: row.id,
            account_id: AccountId(row.account_id),
            name: row.name,
            token_prefix: row.token_prefix,
            created_at: row.created_at,
            last_used_at: row.last_used_at,
            revoked_at: row.revoked_at,
        }
    }
}
