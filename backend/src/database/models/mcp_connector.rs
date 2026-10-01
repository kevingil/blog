use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::mcp_connector;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = mcp_connector)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct McpConnectorRow {
    pub id: Uuid,
    pub name: String,
    pub transport: String,
    pub command: Option<String>,
    pub args: Value,
    pub url: Option<String>,
    pub headers: Value,
    pub env: Value,
    pub enabled: bool,
    pub last_error: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = mcp_connector)]
pub struct NewMcpConnectorRow {
    pub id: Uuid,
    pub name: String,
    pub transport: String,
    pub command: Option<String>,
    pub args: Value,
    pub url: Option<String>,
    pub headers: Value,
    pub env: Value,
    pub enabled: bool,
    pub last_error: Option<String>,
}
