use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};

use crate::core::chat::ChatMessage;

#[derive(Debug, Clone, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ChatRequest {
    pub message: String,
    #[serde(default)]
    pub document_content: String,
    #[serde(default)]
    pub document_markdown: String,
    pub article_id: String,
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

fn timestamp_or_zero(timestamp: Option<DateTime<Utc>>) -> String {
    timestamp.map_or_else(
        || "0001-01-01T00:00:00Z".to_owned(),
        |value| value.to_rfc3339_opts(SecondsFormat::AutoSi, true),
    )
}
