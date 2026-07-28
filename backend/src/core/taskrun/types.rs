use std::fmt;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde_json::{Map, Value};
use uuid::Uuid;

pub type JsonObject = Map<String, Value>;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TaskRunKind {
    Worker,
    Workflow,
    Agent,
    Other(String),
}

impl TaskRunKind {
    pub fn as_str(&self) -> &str {
        match self {
            Self::Worker => "worker",
            Self::Workflow => "workflow",
            Self::Agent => "agent",
            Self::Other(value) => value,
        }
    }
}

impl From<String> for TaskRunKind {
    fn from(value: String) -> Self {
        match value.as_str() {
            "worker" => Self::Worker,
            "workflow" => Self::Workflow,
            "agent" => Self::Agent,
            _ => Self::Other(value),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TaskRunStatus {
    Queued,
    Running,
    Completed,
    Warning,
    Failed,
    Cancelled,
    Other(String),
}

impl TaskRunStatus {
    pub fn as_str(&self) -> &str {
        match self {
            Self::Queued => "queued",
            Self::Running => "running",
            Self::Completed => "completed",
            Self::Warning => "warning",
            Self::Failed => "failed",
            Self::Cancelled => "cancelled",
            Self::Other(value) => value,
        }
    }
}

impl From<String> for TaskRunStatus {
    fn from(value: String) -> Self {
        match value.as_str() {
            "queued" => Self::Queued,
            "running" => Self::Running,
            "completed" => Self::Completed,
            "warning" => Self::Warning,
            "failed" => Self::Failed,
            "cancelled" => Self::Cancelled,
            _ => Self::Other(value),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TaskRunEventLevel {
    Info,
    Warning,
    Error,
    Other(String),
}

impl TaskRunEventLevel {
    pub fn as_str(&self) -> &str {
        match self {
            Self::Info => "info",
            Self::Warning => "warning",
            Self::Error => "error",
            Self::Other(value) => value,
        }
    }
}

impl From<String> for TaskRunEventLevel {
    fn from(value: String) -> Self {
        match value.as_str() {
            "info" => Self::Info,
            "warning" => Self::Warning,
            "error" => Self::Error,
            _ => Self::Other(value),
        }
    }
}

macro_rules! impl_string_enum {
    ($type_name:ty) => {
        impl fmt::Display for $type_name {
            fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                formatter.write_str(self.as_str())
            }
        }

        impl Serialize for $type_name {
            fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
            where
                S: Serializer,
            {
                serializer.serialize_str(self.as_str())
            }
        }

        impl<'de> Deserialize<'de> for $type_name {
            fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
            where
                D: Deserializer<'de>,
            {
                String::deserialize(deserializer).map(Self::from)
            }
        }
    };
}

impl_string_enum!(TaskRunKind);
impl_string_enum!(TaskRunStatus);
impl_string_enum!(TaskRunEventLevel);

#[derive(Debug, Clone, PartialEq)]
pub struct TaskRun {
    pub id: Uuid,
    pub kind: TaskRunKind,
    pub task_name: String,
    pub status: TaskRunStatus,
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub triggered_by_user_id: Option<Uuid>,
    pub trigger_source: String,
    pub parent_run_id: Option<Uuid>,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub input_payload: JsonObject,
    pub output_summary: JsonObject,
    pub metrics: JsonObject,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct TaskRunStep {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub step_key: String,
    pub step_name: String,
    pub status: TaskRunStatus,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub metrics: JsonObject,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct TaskRunEvent {
    pub id: Uuid,
    pub task_run_id: Uuid,
    pub task_run_step_id: Option<Uuid>,
    pub event_type: String,
    pub level: TaskRunEventLevel,
    pub message: String,
    pub meta_data: JsonObject,
    pub created_at: Option<DateTime<Utc>>,
}
