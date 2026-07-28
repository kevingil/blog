use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum MessageRole {
    User,
    Assistant,
    System,
    Tool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FinishReason {
    EndTurn,
    ToolUse,
    Canceled,
    PermissionDenied,
    Unknown,
    MaxTokens,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct TextContent {
    pub text: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BinaryContent {
    pub path: String,
    pub mime_type: String,
    pub data: Vec<u8>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ToolCall {
    pub id: String,
    pub name: String,
    pub input: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub r#type: String,
    #[serde(default, skip_serializing_if = "is_false")]
    pub finished: bool,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub thought_signature: Vec<u8>,
}

const fn is_false(value: &bool) -> bool {
    !*value
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ToolResult {
    pub tool_call_id: String,
    pub content: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub metadata: String,
    pub is_error: bool,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum ContentPart {
    Text(TextContent),
    Binary(BinaryContent),
    ToolCall(ToolCall),
    ToolResult(ToolResult),
    Finish {
        reason: FinishReason,
        at: DateTime<Utc>,
    },
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct LlmMessage {
    pub id: String,
    pub session_id: String,
    pub role: MessageRole,
    pub parts: Vec<ContentPart>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub model: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl LlmMessage {
    pub fn new(
        session_id: impl Into<String>,
        role: MessageRole,
        parts: Vec<ContentPart>,
        model: impl Into<String>,
    ) -> Self {
        let now = Utc::now();
        Self {
            id: Uuid::new_v4().to_string(),
            session_id: session_id.into(),
            role,
            parts,
            model: model.into(),
            created_at: now,
            updated_at: now,
        }
    }

    pub fn text(&self) -> String {
        self.parts
            .iter()
            .filter_map(|part| match part {
                ContentPart::Text(text) => Some(text.text.as_str()),
                _ => None,
            })
            .collect()
    }

    pub fn append_text(&mut self, delta: &str) {
        if let Some(ContentPart::Text(text)) = self
            .parts
            .iter_mut()
            .find(|part| matches!(part, ContentPart::Text(_)))
        {
            text.text.push_str(delta);
        } else {
            self.parts.push(ContentPart::Text(TextContent {
                text: delta.to_owned(),
            }));
        }
        self.updated_at = Utc::now();
    }

    pub fn tool_calls(&self) -> Vec<ToolCall> {
        self.parts
            .iter()
            .filter_map(|part| match part {
                ContentPart::ToolCall(call) => Some(call.clone()),
                _ => None,
            })
            .collect()
    }

    pub fn set_tool_calls(&mut self, calls: Vec<ToolCall>) {
        self.parts
            .retain(|part| !matches!(part, ContentPart::ToolCall(_)));
        self.parts
            .extend(calls.into_iter().map(ContentPart::ToolCall));
        self.updated_at = Utc::now();
    }

    pub fn tool_results(&self) -> Vec<ToolResult> {
        self.parts
            .iter()
            .filter_map(|part| match part {
                ContentPart::ToolResult(result) => Some(result.clone()),
                _ => None,
            })
            .collect()
    }

    pub fn finish(&mut self, reason: FinishReason) {
        self.parts.push(ContentPart::Finish {
            reason,
            at: Utc::now(),
        });
        self.updated_at = Utc::now();
    }

    pub fn finish_reason(&self) -> Option<FinishReason> {
        self.parts.iter().rev().find_map(|part| match part {
            ContentPart::Finish { reason, .. } => Some(*reason),
            _ => None,
        })
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Attachment {
    pub file_path: String,
    pub mime_type: String,
    pub content: Vec<u8>,
}
