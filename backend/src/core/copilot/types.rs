use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::ToSchema;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ChatRequest {
    pub message: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub document_content: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub document_markdown: String,
    pub article_id: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct ChatRequestResponse {
    pub request_id: String,
    pub status: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct StreamResponse {
    #[serde(
        rename = "requestId",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    pub request_id: String,
    #[serde(rename = "type")]
    pub event_type: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub content: String,
    #[serde(default, skip_serializing_if = "is_zero_usize")]
    pub iteration: usize,
    #[serde(default, skip_serializing_if = "is_zero_usize")]
    pub step_index: usize,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_input: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_group: Option<ToolGroupPayload>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_status: Option<ToolStatusPayload>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifact: Option<ArtifactPayload>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub thinking_content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub full_message: Option<FullMessagePayload>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub thinking_message: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub role: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<Value>,
    #[serde(default, skip_serializing_if = "is_false")]
    pub done: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
}

impl StreamResponse {
    pub fn new(request_id: impl Into<String>, event_type: impl Into<String>) -> Self {
        Self {
            request_id: request_id.into(),
            event_type: event_type.into(),
            content: String::new(),
            iteration: 0,
            step_index: 0,
            tool_id: String::new(),
            tool_name: String::new(),
            tool_input: None,
            tool_result: None,
            tool_group: None,
            tool_status: None,
            artifact: None,
            thinking_content: String::new(),
            full_message: None,
            thinking_message: String::new(),
            role: String::new(),
            data: None,
            done: false,
            error: String::new(),
        }
    }

    pub fn terminal_error(request_id: impl Into<String>, error: impl Into<String>) -> Self {
        let mut event = Self::new(request_id, "error");
        event.error = error.into();
        event.done = true;
        event
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolGroupPayload {
    pub group_id: String,
    pub status: String,
    pub calls: Vec<ToolCallPayload>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolCallPayload {
    pub id: String,
    pub name: String,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub input: BTreeMap<String, Value>,
    pub status: String,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub result: BTreeMap<String, Value>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub started_at: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub completed_at: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolStatusPayload {
    pub group_id: String,
    pub tool_id: String,
    pub name: String,
    pub status: String,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub result: BTreeMap<String, Value>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub completed_at: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ArtifactPayload {
    pub id: String,
    #[serde(rename = "type")]
    pub artifact_type: String,
    pub status: String,
    pub data: BTreeMap<String, Value>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct FullMessagePayload {
    pub id: String,
    pub article_id: String,
    pub role: String,
    pub content: String,
    pub meta_data: BTreeMap<String, Value>,
    pub created_at: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct TurnStep {
    #[serde(rename = "type")]
    pub step_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reasoning: Option<ReasoningStep>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool: Option<ToolStatusPayload>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub content: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ReasoningStep {
    pub content: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
    #[serde(default, skip_serializing_if = "is_false")]
    pub visible: bool,
}

const fn is_zero_usize(value: &usize) -> bool {
    *value == 0
}

const fn is_zero_i64(value: &i64) -> bool {
    *value == 0
}

const fn is_false(value: &bool) -> bool {
    !*value
}
