use std::sync::Arc;

use crate::{core::taskrun::TaskRunService, error::AppError};

#[derive(Clone)]
pub struct TaskRunState {
    service: Arc<TaskRunService>,
}

impl TaskRunState {
    pub fn new(service: Arc<TaskRunService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> Result<&TaskRunService, AppError> {
        Ok(&self.service)
    }
}
