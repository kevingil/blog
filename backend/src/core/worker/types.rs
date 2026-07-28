use std::{collections::BTreeMap, fmt, sync::Arc};

use chrono::{DateTime, SecondsFormat, Utc};
use serde_json::Value;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::core::taskrun::TaskRunContext;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WorkerState {
    Idle,
    Running,
    Completed,
    Failed,
}

impl WorkerState {
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Idle => "idle",
            Self::Running => "running",
            Self::Completed => "completed",
            Self::Failed => "failed",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorkerStatus {
    pub name: String,
    pub state: WorkerState,
    pub task_run_id: Option<Uuid>,
    pub progress: i32,
    pub message: String,
    pub started_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
    pub error: String,
    pub items_total: i32,
    pub items_done: i32,
}

impl WorkerStatus {
    pub fn idle(name: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            state: WorkerState::Idle,
            task_run_id: None,
            progress: 0,
            message: String::new(),
            started_at: None,
            completed_at: None,
            error: String::new(),
            items_total: 0,
            items_done: 0,
        }
    }

    pub fn started_at_string(&self) -> Option<String> {
        self.started_at
            .map(|value| value.to_rfc3339_opts(SecondsFormat::Secs, true))
    }

    pub fn completed_at_string(&self) -> Option<String> {
        self.completed_at
            .map(|value| value.to_rfc3339_opts(SecondsFormat::Secs, true))
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorkerStatusUpdate {
    pub worker_name: String,
    pub status: WorkerStatus,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WorkerResultStatus {
    Completed,
    Warning,
}

impl WorkerResultStatus {
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Completed => "completed",
            Self::Warning => "warning",
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct WorkerResult {
    pub status: WorkerResultStatus,
    pub summary: String,
    pub output_summary: BTreeMap<String, Value>,
    pub metrics: BTreeMap<String, Value>,
    pub warnings: Vec<String>,
}

impl WorkerResult {
    pub fn completed(summary: impl Into<String>) -> Self {
        Self {
            status: WorkerResultStatus::Completed,
            summary: summary.into(),
            output_summary: BTreeMap::new(),
            metrics: BTreeMap::new(),
            warnings: Vec::new(),
        }
    }

    pub fn warning(summary: impl Into<String>, warnings: Vec<String>) -> Self {
        Self {
            status: WorkerResultStatus::Warning,
            summary: summary.into(),
            output_summary: BTreeMap::new(),
            metrics: BTreeMap::new(),
            warnings,
        }
    }
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RunMetadata {
    pub user_id: Option<Uuid>,
    pub organization_id: Option<Uuid>,
    pub triggered_by_user_id: Option<Uuid>,
    pub parent_run_id: Option<Uuid>,
    pub trigger_source: String,
}

#[derive(Clone)]
pub struct WorkerContext {
    cancellation: CancellationToken,
    task_run: TaskRunContext,
}

impl WorkerContext {
    pub fn new(cancellation: CancellationToken, task_run: TaskRunContext) -> Self {
        Self {
            cancellation,
            task_run,
        }
    }

    pub fn cancellation(&self) -> &CancellationToken {
        &self.cancellation
    }

    pub fn task_run(&self) -> &TaskRunContext {
        &self.task_run
    }

    pub async fn cancelled(&self) {
        self.cancellation.cancelled().await;
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorkerFailure {
    message: Arc<str>,
}

impl WorkerFailure {
    pub fn new(message: impl Into<Arc<str>>) -> Self {
        Self {
            message: message.into(),
        }
    }

    pub fn message(&self) -> &str {
        &self.message
    }
}

impl fmt::Display for WorkerFailure {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(&self.message)
    }
}

impl std::error::Error for WorkerFailure {}
