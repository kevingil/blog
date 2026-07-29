use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

pub const CHANNEL_WORKER_STATUS: &str = "worker-status";

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SubscribeMessage {
    #[serde(default)]
    pub request_id: String,
    #[serde(default)]
    pub action: String,
    #[serde(default)]
    pub channel: String,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AgentStreamEvent {
    fields: Map<String, Value>,
}

impl AgentStreamEvent {
    pub fn new(fields: Map<String, Value>) -> Self {
        Self { fields }
    }

    pub fn from_value(value: Value) -> Option<Self> {
        value.as_object().cloned().map(Self::new)
    }

    pub fn fields(&self) -> &Map<String, Value> {
        &self.fields
    }

    pub(crate) fn into_wire_message(mut self, request_id: &str) -> (String, bool) {
        self.fields
            .insert("requestId".to_owned(), Value::String(request_id.to_owned()));
        let terminal = self
            .fields
            .get("done")
            .and_then(Value::as_bool)
            .unwrap_or(false)
            || self
                .fields
                .get("error")
                .and_then(Value::as_str)
                .is_some_and(|error| !error.is_empty());
        (Value::Object(self.fields).to_string(), terminal)
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct WorkerStatus {
    pub name: String,
    pub state: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub task_run_id: Option<String>,
    pub progress: i32,
    pub message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub started_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub completed_at: Option<String>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub error: String,
    pub items_total: i32,
    pub items_done: i32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorkerStatusSnapshot {
    pub worker_name: String,
    pub status: WorkerStatus,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorkerStatusUpdate {
    pub worker_name: String,
    pub status: WorkerStatus,
    pub timestamp: DateTime<Utc>,
}

#[derive(Serialize)]
pub(crate) struct WorkerStatusMessage<'a> {
    #[serde(rename = "type")]
    message_type: &'static str,
    worker_name: &'a str,
    status: WorkerStatusWire<'a>,
    #[serde(skip_serializing_if = "Option::is_none")]
    timestamp: Option<DateTime<Utc>>,
}

#[derive(Serialize)]
struct WorkerStatusWire<'a> {
    name: &'a str,
    state: &'a str,
    progress: i32,
    message: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    started_at: Option<&'a str>,
    #[serde(skip_serializing_if = "Option::is_none")]
    completed_at: Option<&'a str>,
    error: &'a String,
    items_total: i32,
    items_done: i32,
}

impl<'a> WorkerStatusMessage<'a> {
    pub(crate) fn initial(snapshot: &'a WorkerStatusSnapshot) -> Self {
        Self {
            message_type: CHANNEL_WORKER_STATUS,
            worker_name: &snapshot.worker_name,
            status: WorkerStatusWire::new(&snapshot.status),
            timestamp: None,
        }
    }

    pub(crate) fn update(update: &'a WorkerStatusUpdate) -> Self {
        Self {
            message_type: CHANNEL_WORKER_STATUS,
            worker_name: &update.worker_name,
            status: WorkerStatusWire::new(&update.status),
            timestamp: Some(update.timestamp),
        }
    }
}

impl<'a> WorkerStatusWire<'a> {
    fn new(status: &'a WorkerStatus) -> Self {
        Self {
            name: &status.name,
            state: &status.state,
            progress: status.progress,
            message: &status.message,
            started_at: status.started_at.as_deref(),
            completed_at: status.completed_at.as_deref(),
            error: &status.error,
            items_total: status.items_total,
            items_done: status.items_done,
        }
    }
}
