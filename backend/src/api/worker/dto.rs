use serde::Serialize;
use utoipa::ToSchema;

use crate::core::worker::WorkerStatus;

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct WorkerStatusResponse {
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

impl From<WorkerStatus> for WorkerStatusResponse {
    fn from(status: WorkerStatus) -> Self {
        let started_at = status.started_at_string();
        let completed_at = status.completed_at_string();
        Self {
            name: status.name,
            state: status.state.as_str().to_owned(),
            task_run_id: status.task_run_id.map(|id| id.to_string()),
            progress: status.progress,
            message: status.message,
            started_at,
            completed_at,
            error: status.error,
            items_total: status.items_total,
            items_done: status.items_done,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct AllWorkersStatusResponse {
    pub workers: Vec<WorkerStatusResponse>,
    pub is_running: bool,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct RunningWorkersResponse {
    pub workers: Vec<String>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct RunWorkerResponse {
    pub started: bool,
    pub message: String,
    pub task_run_id: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct StopWorkerResponse {
    pub stopped: bool,
    pub message: String,
}
