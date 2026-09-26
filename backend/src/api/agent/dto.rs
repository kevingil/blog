use std::collections::BTreeMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};

use crate::core::{
    chat::ChatMessage,
    mcp::{CreateMcpConnector, McpConnector, UpdateMcpConnector},
    ml::llm::RegisteredTool,
};

#[derive(Debug, Clone, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ChatRequest {
    pub message: String,
    #[serde(default)]
    pub document_content: String,
    #[serde(default)]
    pub document_markdown: String,
    pub article_id: String,
    #[serde(default)]
    pub channel: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ChatRequestResponse {
    pub request_id: String,
    pub status: String,
}

#[derive(Debug, Clone, Default, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct ConversationQuery {
    pub limit: Option<String>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct ChatMessageResponse {
    pub id: String,
    pub article_id: String,
    pub role: String,
    pub content: String,
    pub meta_data: Option<Value>,
    pub created_at: String,
}

impl From<ChatMessage> for ChatMessageResponse {
    fn from(message: ChatMessage) -> Self {
        Self {
            id: message.id.to_string(),
            article_id: message.article_id.to_string(),
            role: message.role,
            content: message.content,
            meta_data: message.meta_data,
            created_at: timestamp_or_zero(message.created_at),
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct ConversationHistoryResponse {
    pub messages: Vec<ChatMessageResponse>,
    pub article_id: String,
    pub total: usize,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct PendingArtifactsResponse {
    pub artifacts: Vec<ChatMessageResponse>,
}

#[derive(Debug, Clone, Default, Deserialize, ToSchema)]
pub struct ArtifactFeedbackRequest {
    #[serde(default)]
    pub feedback: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct SuccessFlagResponse {
    pub success: bool,
}

#[derive(Debug, Clone, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConversationTurnRequest {
    pub article_id: String,
    #[serde(default)]
    pub document_content: String,
    #[serde(default)]
    pub document_markdown: String,
    #[serde(default)]
    pub message: String,
    #[serde(default)]
    pub audio_base64: String,
    #[serde(default)]
    pub mime_type: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConversationTurnResponse {
    pub request_id: String,
    pub status: String,
    pub transcript: String,
    pub channel: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConnectorResponse {
    pub id: String,
    pub name: String,
    pub transport: String,
    pub command: String,
    pub args: Vec<String>,
    pub url: String,
    pub headers: BTreeMap<String, String>,
    pub env: BTreeMap<String, String>,
    pub enabled: bool,
    pub last_error: String,
    pub created_at: String,
    pub updated_at: String,
}

impl From<McpConnector> for ConnectorResponse {
    fn from(connector: McpConnector) -> Self {
        Self {
            id: connector.id.to_string(),
            name: connector.name,
            transport: connector.transport,
            command: connector.command,
            args: connector.args,
            url: connector.url,
            headers: connector.headers,
            env: connector.env,
            enabled: connector.enabled,
            last_error: connector.last_error,
            created_at: timestamp_or_zero(connector.created_at),
            updated_at: timestamp_or_zero(connector.updated_at),
        }
    }
}

#[derive(Debug, Clone, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConnectorWriteRequest {
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
    #[serde(default = "default_true")]
    pub enabled: bool,
}

impl From<ConnectorWriteRequest> for CreateMcpConnector {
    fn from(request: ConnectorWriteRequest) -> Self {
        Self {
            name: request.name,
            transport: request.transport,
            command: request.command,
            args: request.args,
            url: request.url,
            headers: request.headers,
            env: request.env,
            enabled: request.enabled,
        }
    }
}

#[derive(Debug, Clone, Default, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConnectorUpdateRequest {
    pub name: Option<String>,
    pub transport: Option<String>,
    pub command: Option<String>,
    pub args: Option<Vec<String>>,
    pub url: Option<String>,
    pub headers: Option<BTreeMap<String, String>>,
    pub env: Option<BTreeMap<String, String>>,
    pub enabled: Option<bool>,
}

impl From<ConnectorUpdateRequest> for UpdateMcpConnector {
    fn from(request: ConnectorUpdateRequest) -> Self {
        Self {
            name: request.name,
            transport: request.transport,
            command: request.command,
            args: request.args,
            url: request.url,
            headers: request.headers,
            env: request.env,
            enabled: request.enabled,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConnectorListResponse {
    pub connectors: Vec<ConnectorResponse>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ConnectorRefreshResponse {
    pub connector_id: String,
    pub tool_names: Vec<String>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct AgentToolResponse {
    pub name: String,
    pub description: String,
    pub source: String,
}

impl From<RegisteredTool> for AgentToolResponse {
    fn from(tool: RegisteredTool) -> Self {
        Self {
            name: tool.name,
            description: tool.description,
            source: tool.source,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct AgentToolListResponse {
    pub tools: Vec<AgentToolResponse>,
}

fn default_true() -> bool {
    true
}

fn timestamp_or_zero(timestamp: Option<DateTime<Utc>>) -> String {
    timestamp.map_or_else(
        || "0001-01-01T00:00:00Z".to_owned(),
        |value| value.to_rfc3339_opts(SecondsFormat::AutoSi, true),
    )
}
