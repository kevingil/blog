use std::sync::Arc;

use crate::core::worker::{StatusService, WorkerManager};

#[derive(Clone)]
pub struct WorkerState {
    manager: Arc<WorkerManager>,
    status: Arc<StatusService>,
}

impl WorkerState {
    pub fn new(manager: Arc<WorkerManager>, status: Arc<StatusService>) -> Self {
        Self { manager, status }
    }

    pub fn manager(&self) -> &Arc<WorkerManager> {
        &self.manager
    }

    pub fn status(&self) -> &StatusService {
        &self.status
    }
}
