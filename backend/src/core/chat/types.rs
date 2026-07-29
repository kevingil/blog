use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ChatMessage {
    pub id: Uuid,
    pub article_id: Uuid,
    pub role: String,
    pub content: String,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct MessageMetadata {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifact: Option<ArtifactInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_execution: Option<ToolExecution>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub context: Option<MessageContext>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_action: Option<UserAction>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thinking: Option<ThinkingBlock>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub steps: Vec<ChainOfThoughtStep>,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ArtifactInfo {
    pub id: String,
    #[serde(rename = "type")]
    pub artifact_type: String,
    pub status: String,
    pub content: String,
    pub diff_preview: String,
    pub title: String,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub applied_at: Option<DateTime<Utc>>,
}

pub const ARTIFACT_STATUS_PENDING: &str = "pending";
pub const ARTIFACT_STATUS_ACCEPTED: &str = "accepted";
pub const ARTIFACT_STATUS_REJECTED: &str = "rejected";
pub const ARTIFACT_STATUS_APPLIED: &str = "applied";

pub const ARTIFACT_TYPE_CODE_EDIT: &str = "code_edit";
pub const ARTIFACT_TYPE_REWRITE: &str = "rewrite";
pub const ARTIFACT_TYPE_SUGGESTION: &str = "suggestion";
pub const ARTIFACT_TYPE_CONTENT_GENERATION: &str = "content_generation";
pub const ARTIFACT_TYPE_IMAGE_PROMPT: &str = "image_prompt";

pub const USER_ACTION_ACCEPT: &str = "accept";
pub const USER_ACTION_REJECT: &str = "reject";
pub const USER_ACTION_MODIFY: &str = "modify";

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ToolExecution {
    pub tool_name: String,
    pub tool_id: String,
    pub input: Value,
    pub output: Value,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    pub duration_ms: i64,
    pub executed_at: DateTime<Utc>,
    pub success: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default)]
pub struct MessageContext {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub article_id: String,
    pub session_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub request_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub document_version: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub document_hash: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub user_id: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct UserAction {
    pub action: String,
    pub timestamp: DateTime<Utc>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub artifact_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub feedback: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reason: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct TurnMetadata {
    pub turn_id: String,
    pub turn_sequence: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thinking: Option<ThinkingBlock>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_group: Option<ToolGroup>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub artifacts: Vec<Artifact>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub context: Option<MessageContext>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default)]
pub struct ThinkingBlock {
    pub content: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
    pub visible: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ChainOfThoughtStep {
    #[serde(rename = "type")]
    pub step_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reasoning: Option<ThinkingBlock>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool: Option<ToolStepInfo>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub content: String,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ToolStepInfo {
    pub tool_id: String,
    pub tool_name: String,
    #[serde(default, skip_serializing_if = "Map::is_empty")]
    pub input: Map<String, Value>,
    #[serde(default, skip_serializing_if = "Map::is_empty")]
    pub output: Map<String, Value>,
    pub status: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub started_at: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub completed_at: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ToolGroup {
    pub group_id: String,
    pub status: ToolGroupStatus,
    pub calls: Option<Vec<ToolCallRecord>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ToolGroupStatus {
    Pending,
    Running,
    Completed,
    Error,
    Other(String),
}

impl Default for ToolGroupStatus {
    fn default() -> Self {
        Self::Other(String::new())
    }
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct ToolCallRecord {
    pub id: String,
    pub name: String,
    pub input: Option<Map<String, Value>>,
    pub status: ToolCallStatus,
    #[serde(default, skip_serializing_if = "Map::is_empty")]
    pub result: Map<String, Value>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
    pub started_at: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub completed_at: String,
    #[serde(default, skip_serializing_if = "is_zero_i64")]
    pub duration_ms: i64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ToolCallStatus {
    Pending,
    Running,
    Completed,
    Error,
    Other(String),
}

impl Default for ToolCallStatus {
    fn default() -> Self {
        Self::Other(String::new())
    }
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct Artifact {
    pub id: String,
    #[serde(rename = "type")]
    pub artifact_type: ArtifactType,
    pub status: ArtifactStatus,
    pub data: Option<Map<String, Value>>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ArtifactType {
    Diff,
    Sources,
    Answer,
    Content,
    ImagePrompt,
    Other(String),
}

impl Default for ArtifactType {
    fn default() -> Self {
        Self::Other(String::new())
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ArtifactStatus {
    Pending,
    Accepted,
    Rejected,
    Other(String),
}

impl Default for ArtifactStatus {
    fn default() -> Self {
        Self::Other(String::new())
    }
}

const fn is_zero_i64(value: &i64) -> bool {
    *value == 0
}

macro_rules! impl_string_enum {
    ($type_name:ty, {$($variant:path => $value:literal),+ $(,)?}) => {
        impl $type_name {
            pub fn as_str(&self) -> &str {
                match self {
                    $($variant => $value,)+
                    Self::Other(value) => value,
                }
            }
        }

        impl From<String> for $type_name {
            fn from(value: String) -> Self {
                match value.as_str() {
                    $($value => $variant,)+
                    _ => Self::Other(value),
                }
            }
        }

        impl Serialize for $type_name {
            fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
            where
                S: serde::Serializer,
            {
                serializer.serialize_str(self.as_str())
            }
        }

        impl<'de> Deserialize<'de> for $type_name {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where
                D: serde::Deserializer<'de>,
            {
                String::deserialize(deserializer).map(Self::from)
            }
        }
    };
}

impl_string_enum!(ToolGroupStatus, {
    Self::Pending => "pending",
    Self::Running => "running",
    Self::Completed => "completed",
    Self::Error => "error",
});
impl_string_enum!(ToolCallStatus, {
    Self::Pending => "pending",
    Self::Running => "running",
    Self::Completed => "completed",
    Self::Error => "error",
});
impl_string_enum!(ArtifactType, {
    Self::Diff => "diff",
    Self::Sources => "sources",
    Self::Answer => "answer",
    Self::Content => "content_generation",
    Self::ImagePrompt => "image_prompt",
});
impl_string_enum!(ArtifactStatus, {
    Self::Pending => "pending",
    Self::Accepted => "accepted",
    Self::Rejected => "rejected",
});
