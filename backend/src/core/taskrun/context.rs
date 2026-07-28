use std::sync::Arc;

use uuid::Uuid;

use crate::error::AppError;

use super::{
    FinishStepInput, JsonObject, RecordEventInput, StartStepInput, TaskRunEventLevel,
    TaskRunService, TaskRunStatus,
};

#[derive(Clone)]
pub struct TaskRunTracker {
    service: Arc<TaskRunService>,
    run_id: Uuid,
}

impl TaskRunTracker {
    pub fn new(service: Arc<TaskRunService>, run_id: Uuid) -> Option<Self> {
        (!run_id.is_nil()).then_some(Self { service, run_id })
    }

    pub fn run_id(&self) -> Uuid {
        self.run_id
    }

    pub async fn start_step(
        &self,
        step_key: impl Into<String>,
        step_name: impl Into<String>,
        summary: Option<String>,
    ) -> Result<(), AppError> {
        self.service
            .start_step(StartStepInput {
                run_id: self.run_id,
                step_key: step_key.into(),
                step_name: step_name.into(),
                summary,
            })
            .await
            .map(|_| ())
    }

    pub async fn finish_step(
        &self,
        step_key: impl Into<String>,
        status: &str,
        summary: Option<String>,
        error_summary: Option<String>,
        metrics: JsonObject,
    ) -> Result<(), AppError> {
        self.service
            .finish_step(FinishStepInput {
                run_id: self.run_id,
                step_key: step_key.into(),
                status: parse_status(status),
                summary,
                error_summary,
                metrics,
            })
            .await
    }

    pub async fn record_event(
        &self,
        step_key: Option<String>,
        event_type: impl Into<String>,
        level: &str,
        message: impl Into<String>,
        meta_data: JsonObject,
    ) -> Result<(), AppError> {
        self.service
            .record_event(RecordEventInput {
                run_id: self.run_id,
                step_key,
                event_type: event_type.into(),
                level: parse_level(level),
                message: message.into(),
                meta_data,
            })
            .await
    }
}

#[derive(Clone, Default)]
pub struct TaskRunContext {
    tracker: Option<TaskRunTracker>,
}

impl TaskRunContext {
    pub fn new(tracker: Option<TaskRunTracker>) -> Self {
        Self { tracker }
    }

    pub fn tracker(&self) -> Option<&TaskRunTracker> {
        self.tracker.as_ref()
    }

    pub fn run_id(&self) -> Option<Uuid> {
        self.tracker.as_ref().map(TaskRunTracker::run_id)
    }

    pub async fn start_step(
        &self,
        step_key: impl Into<String>,
        step_name: impl Into<String>,
        summary: Option<String>,
    ) -> Result<(), AppError> {
        let Some(tracker) = self.tracker.as_ref() else {
            return Ok(());
        };
        tracker.start_step(step_key, step_name, summary).await
    }

    pub async fn finish_step(
        &self,
        step_key: impl Into<String>,
        status: &str,
        summary: Option<String>,
        error_summary: Option<String>,
        metrics: JsonObject,
    ) -> Result<(), AppError> {
        let Some(tracker) = self.tracker.as_ref() else {
            return Ok(());
        };
        tracker
            .finish_step(step_key, status, summary, error_summary, metrics)
            .await
    }

    pub async fn record_event(
        &self,
        step_key: Option<String>,
        event_type: impl Into<String>,
        level: &str,
        message: impl Into<String>,
        meta_data: JsonObject,
    ) -> Result<(), AppError> {
        let Some(tracker) = self.tracker.as_ref() else {
            return Ok(());
        };
        tracker
            .record_event(step_key, event_type, level, message, meta_data)
            .await
    }
}

fn parse_status(status: &str) -> TaskRunStatus {
    match status {
        "warning" => TaskRunStatus::Warning,
        "failed" => TaskRunStatus::Failed,
        "cancelled" => TaskRunStatus::Cancelled,
        _ => TaskRunStatus::Completed,
    }
}

fn parse_level(level: &str) -> TaskRunEventLevel {
    match level {
        "warning" => TaskRunEventLevel::Warning,
        "error" => TaskRunEventLevel::Error,
        _ => TaskRunEventLevel::Info,
    }
}
