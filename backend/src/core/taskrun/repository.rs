use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{TaskRun, TaskRunEvent, TaskRunStep};

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct TaskRunFilter {
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub task_name: String,
    pub status: String,
    pub kind: String,
    pub limit: i64,
}

#[async_trait]
pub trait TaskRunRepository: Send + Sync {
    async fn create_run(&self, run: &mut TaskRun) -> Result<(), AppError>;
    async fn update_run(&self, run: &TaskRun) -> Result<(), AppError>;
    async fn find_run_by_id(&self, id: Uuid) -> Result<TaskRun, AppError>;
    async fn list_runs(&self, filter: TaskRunFilter) -> Result<Vec<TaskRun>, AppError>;
    async fn create_step(&self, step: &mut TaskRunStep) -> Result<(), AppError>;
    async fn update_step(&self, step: &TaskRunStep) -> Result<(), AppError>;
    async fn find_step_by_run_and_key(
        &self,
        run_id: Uuid,
        step_key: &str,
    ) -> Result<TaskRunStep, AppError>;
    async fn list_steps_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunStep>, AppError>;
    async fn create_event(&self, event: &mut TaskRunEvent) -> Result<(), AppError>;
    async fn list_events_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunEvent>, AppError>;
}
