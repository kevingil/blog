use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

pub const TRANSPORT_STDIO: &str = "stdio";
pub const TRANSPORT_SSE: &str = "sse";
pub const TRANSPORT_HTTP: &str = "http";

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct McpConnector {
    pub id: Uuid,
    pub name: String,
    pub transport: String,
    pub command: String,
    pub args: Vec<String>,
    pub url: String,
    pub headers: BTreeMap<String, String>,
    pub env: BTreeMap<String, String>,
    pub enabled: bool,
    pub last_error: String,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct CreateMcpConnector {
    pub name: String,
    pub transport: String,
    #[serde(default)]
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub headers: BTreeMap<String, String>,
    #[serde(default)]
    pub env: BTreeMap<String, String>,
    #[serde(default = "default_enabled")]
    pub enabled: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct UpdateMcpConnector {
    pub name: Option<String>,
    pub transport: Option<String>,
    pub command: Option<String>,
    pub args: Option<Vec<String>>,
    pub url: Option<String>,
    pub headers: Option<BTreeMap<String, String>>,
    pub env: Option<BTreeMap<String, String>>,
    pub enabled: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct McpToolDescriptor {
    pub name: String,
    pub description: String,
    #[serde(default)]
    pub input_schema: Value,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ConnectorRefresh {
    pub connector_id: Uuid,
    pub tool_names: Vec<String>,
}

fn default_enabled() -> bool {
    true
}

pub fn sanitize_tool_prefix(name: &str) -> String {
    let mut prefix = String::new();
    for character in name.chars() {
        if character.is_ascii_alphanumeric() {
            prefix.push(character.to_ascii_lowercase());
        } else if !prefix.ends_with('_') {
            prefix.push('_');
        }
    }
    let prefix = prefix.trim_matches('_').to_owned();
    if prefix.is_empty() {
        "mcp".to_owned()
    } else {
        prefix
    }
}
