use std::sync::Arc;

use crate::{
    api::websocket::{
        WorkerStatus as WebSocketWorkerStatus, WorkerStatusProvider, WorkerStatusSnapshot,
        WorkerStatusUpdate as WebSocketWorkerStatusUpdate,
    },
    core::worker::{StatusService, WorkerStatus},
};

pub struct WorkerStatusAdapter {
    status: Arc<StatusService>,
}

impl WorkerStatusAdapter {
    pub fn new(status: Arc<StatusService>) -> Self {
        Self { status }
    }
}

impl WorkerStatusProvider for WorkerStatusAdapter {
    fn snapshot(&self) -> Vec<WorkerStatusSnapshot> {
        self.status
            .snapshot()
            .into_iter()
            .map(|(worker_name, status)| WorkerStatusSnapshot {
                worker_name,
                status: websocket_status(status),
            })
            .collect()
    }

    fn subscribe(&self) -> tokio::sync::mpsc::Receiver<WebSocketWorkerStatusUpdate> {
        self.status
            .subscribe_mapped(|update| WebSocketWorkerStatusUpdate {
                worker_name: update.worker_name.clone(),
                status: websocket_status(update.status.clone()),
                timestamp: update.timestamp,
            })
    }
}

fn websocket_status(status: WorkerStatus) -> WebSocketWorkerStatus {
    let started_at = status.started_at_string();
    let completed_at = status.completed_at_string();
    WebSocketWorkerStatus {
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
