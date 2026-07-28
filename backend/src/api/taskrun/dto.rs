use std::collections::HashMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::core::taskrun::{TaskRun, TaskRunEvent, TaskRunStep};

#[derive(Debug, Clone, Default, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct TaskRunListQuery {
    pub task_name: Option<String>,
    pub status: Option<String>,
    pub kind: Option<String>,
    pub limit: Option<String>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunResponse {
    pub id: String,
    pub kind: String,
    pub task_name: String,
    pub status: String,
    pub trigger_source: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error_summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub started_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub completed_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration_ms: Option<i64>,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub output_summary: Map<String, Value>,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub metrics: Map<String, Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_run_id: Option<String>,
}

impl From<TaskRun> for TaskRunResponse {
    fn from(run: TaskRun) -> Self {
        let duration_ms = run
            .started_at
            .zip(run.completed_at)
            .map(|(started, completed)| (completed - started).num_milliseconds());
        Self {
            id: run.id.to_string(),
            kind: run.kind.as_str().to_owned(),
            task_name: run.task_name,
            status: run.status.as_str().to_owned(),
            trigger_source: run.trigger_source,
            summary: run.summary,
            error_summary: run.error_summary,
            started_at: run.started_at.map(timestamp),
            completed_at: run.completed_at.map(timestamp),
            duration_ms,
            output_summary: run.output_summary,
            metrics: run.metrics,
            parent_run_id: run.parent_run_id.map(|id| id.to_string()),
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunStepResponse {
    pub id: String,
    pub step_key: String,
    pub step_name: String,
    pub status: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error_summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub started_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub completed_at: Option<String>,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub metrics: Map<String, Value>,
}

impl From<TaskRunStep> for TaskRunStepResponse {
    fn from(step: TaskRunStep) -> Self {
        Self {
            id: step.id.to_string(),
            step_key: step.step_key,
            step_name: step.step_name,
            status: step.status.as_str().to_owned(),
            summary: step.summary,
            error_summary: step.error_summary,
            started_at: step.started_at.map(timestamp),
            completed_at: step.completed_at.map(timestamp),
            metrics: step.metrics,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunEventResponse {
    pub id: String,
    pub event_type: String,
    pub level: String,
    pub message: String,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub step_key: Option<String>,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub meta_data: Map<String, Value>,
}

impl TaskRunEventResponse {
    pub fn new(event: TaskRunEvent, step_keys: &HashMap<Uuid, String>) -> Self {
        Self {
            id: event.id.to_string(),
            event_type: event.event_type,
            level: event.level.as_str().to_owned(),
            message: event.message,
            created_at: event.created_at.map_or_else(zero_timestamp, timestamp),
            step_key: event
                .task_run_step_id
                .and_then(|id| step_keys.get(&id).cloned()),
            meta_data: event.meta_data,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunListResponse {
    pub runs: Vec<TaskRunResponse>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunDetailResponse {
    pub run: TaskRunResponse,
    pub steps: Vec<TaskRunStepResponse>,
    pub events: Vec<TaskRunEventResponse>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct TaskRunEventsResponse {
    pub events: Vec<TaskRunEventResponse>,
}

pub fn step_keys(steps: &[TaskRunStep]) -> HashMap<Uuid, String> {
    steps
        .iter()
        .map(|step| (step.id, step.step_key.clone()))
        .collect()
}

fn timestamp(value: DateTime<Utc>) -> String {
    value.to_rfc3339_opts(SecondsFormat::Secs, true)
}

fn zero_timestamp() -> String {
    "0001-01-01T00:00:00Z".to_owned()
}
