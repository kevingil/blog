use std::sync::Arc;

use chrono::Utc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    JsonObject, TaskRun, TaskRunEvent, TaskRunEventLevel, TaskRunFilter, TaskRunKind,
    TaskRunRepository, TaskRunStatus, TaskRunStep,
};

#[derive(Clone)]
pub struct TaskRunService {
    repository: Arc<dyn TaskRunRepository>,
    cancellation: CancellationToken,
}

#[derive(Debug, Clone)]
pub struct StartRunInput {
    pub kind: TaskRunKind,
    pub task_name: String,
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub triggered_by_user_id: Option<Uuid>,
    pub trigger_source: String,
    pub parent_run_id: Option<Uuid>,
    pub input_payload: JsonObject,
    pub summary: Option<String>,
}

#[derive(Debug, Clone)]
pub struct FinishRunInput {
    pub run_id: Uuid,
    pub status: TaskRunStatus,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub output_summary: JsonObject,
    pub metrics: JsonObject,
}

#[derive(Debug, Clone)]
pub struct StartStepInput {
    pub run_id: Uuid,
    pub step_key: String,
    pub step_name: String,
    pub summary: Option<String>,
}

#[derive(Debug, Clone)]
pub struct FinishStepInput {
    pub run_id: Uuid,
    pub step_key: String,
    pub status: TaskRunStatus,
    pub summary: Option<String>,
    pub error_summary: Option<String>,
    pub metrics: JsonObject,
}

#[derive(Debug, Clone)]
pub struct RecordEventInput {
    pub run_id: Uuid,
    pub step_key: Option<String>,
    pub event_type: String,
    pub level: TaskRunEventLevel,
    pub message: String,
    pub meta_data: JsonObject,
}

impl TaskRunService {
    pub fn new(repository: Arc<dyn TaskRunRepository>, cancellation: CancellationToken) -> Self {
        Self {
            repository,
            cancellation,
        }
    }

    pub async fn start_run(&self, input: StartRunInput) -> Result<TaskRun, AppError> {
        let now = Utc::now();
        let mut run = TaskRun {
            id: Uuid::nil(),
            kind: input.kind,
            task_name: input.task_name,
            status: TaskRunStatus::Running,
            organization_id: input.organization_id,
            user_id: input.user_id,
            triggered_by_user_id: input.triggered_by_user_id,
            trigger_source: if input.trigger_source.is_empty() {
                "manual".to_owned()
            } else {
                input.trigger_source
            },
            parent_run_id: input.parent_run_id,
            summary: input.summary,
            error_summary: None,
            input_payload: input.input_payload,
            output_summary: JsonObject::new(),
            metrics: JsonObject::new(),
            started_at: Some(now),
            completed_at: None,
            created_at: None,
            updated_at: None,
        };
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.create_run(&mut run) => result?,
        }
        self.record_event(RecordEventInput {
            run_id: run.id,
            step_key: None,
            event_type: "run_started".to_owned(),
            level: TaskRunEventLevel::Info,
            message: "Run started".to_owned(),
            meta_data: JsonObject::new(),
        })
        .await?;
        Ok(run)
    }

    pub async fn finish_run(&self, input: FinishRunInput) -> Result<(), AppError> {
        let mut run = self.get_run(input.run_id).await?;
        run.status = input.status.clone();
        run.summary = input.summary;
        run.error_summary = input.error_summary;
        run.output_summary = input.output_summary;
        run.metrics = input.metrics;
        run.completed_at = Some(Utc::now());
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.update_run(&run) => result?,
        }

        let (event_type, level, default_message) = match input.status {
            TaskRunStatus::Warning => (
                "run_warning",
                TaskRunEventLevel::Warning,
                "Run completed with warnings",
            ),
            TaskRunStatus::Failed => ("run_failed", TaskRunEventLevel::Error, "Run failed"),
            TaskRunStatus::Cancelled => {
                ("run_cancelled", TaskRunEventLevel::Warning, "Run cancelled")
            }
            _ => ("run_completed", TaskRunEventLevel::Info, "Run completed"),
        };
        let message = run
            .summary
            .as_ref()
            .filter(|summary| !summary.is_empty())
            .cloned()
            .unwrap_or_else(|| default_message.to_owned());
        let mut meta_data = JsonObject::new();
        meta_data.insert(
            "status".to_owned(),
            serde_json::Value::String(input.status.as_str().to_owned()),
        );
        self.record_event(RecordEventInput {
            run_id: run.id,
            step_key: None,
            event_type: event_type.to_owned(),
            level,
            message,
            meta_data,
        })
        .await
    }

    pub async fn start_step(&self, input: StartStepInput) -> Result<TaskRunStep, AppError> {
        let mut step = TaskRunStep {
            id: Uuid::nil(),
            task_run_id: input.run_id,
            step_key: input.step_key.clone(),
            step_name: input.step_name.clone(),
            status: TaskRunStatus::Running,
            summary: input.summary,
            error_summary: None,
            metrics: JsonObject::new(),
            started_at: Some(Utc::now()),
            completed_at: None,
            created_at: None,
            updated_at: None,
        };
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.create_step(&mut step) => result?,
        }
        self.record_event(RecordEventInput {
            run_id: input.run_id,
            step_key: Some(input.step_key),
            event_type: "step_started".to_owned(),
            level: TaskRunEventLevel::Info,
            message: input.step_name,
            meta_data: JsonObject::new(),
        })
        .await?;
        Ok(step)
    }

    pub async fn finish_step(&self, input: FinishStepInput) -> Result<(), AppError> {
        let mut step = self
            .find_step_by_run_and_key(input.run_id, &input.step_key)
            .await?;
        step.status = input.status.clone();
        step.summary = input.summary.clone();
        step.error_summary = input.error_summary;
        step.metrics = input.metrics.clone();
        step.completed_at = Some(Utc::now());
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.update_step(&step) => result?,
        }

        let (event_type, level, default_message) = match input.status {
            TaskRunStatus::Warning => (
                "step_warning",
                TaskRunEventLevel::Warning,
                format!("{} completed with warnings", step.step_name),
            ),
            TaskRunStatus::Failed => (
                "step_failed",
                TaskRunEventLevel::Error,
                format!("{} failed", step.step_name),
            ),
            _ => (
                "step_completed",
                TaskRunEventLevel::Info,
                step.step_name.clone(),
            ),
        };
        let message = input
            .summary
            .filter(|summary| !summary.is_empty())
            .unwrap_or(default_message);
        self.record_event(RecordEventInput {
            run_id: input.run_id,
            step_key: Some(input.step_key),
            event_type: event_type.to_owned(),
            level,
            message,
            meta_data: input.metrics,
        })
        .await
    }

    pub async fn record_event(&self, input: RecordEventInput) -> Result<(), AppError> {
        let step_id = if let Some(step_key) = input.step_key.as_deref() {
            self.find_step_by_run_and_key(input.run_id, step_key)
                .await
                .ok()
                .map(|step| step.id)
        } else {
            None
        };
        let mut event = TaskRunEvent {
            id: Uuid::nil(),
            task_run_id: input.run_id,
            task_run_step_id: step_id,
            event_type: input.event_type,
            level: input.level,
            message: input.message,
            meta_data: input.meta_data,
            created_at: None,
        };
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.create_event(&mut event) => result,
        }
    }

    pub async fn get_run(&self, id: Uuid) -> Result<TaskRun, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.find_run_by_id(id) => result,
        }
    }

    pub async fn list_runs(&self, filter: TaskRunFilter) -> Result<Vec<TaskRun>, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.list_runs(filter) => result,
        }
    }

    pub async fn list_steps_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunStep>, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.list_steps_by_run_id(run_id) => result,
        }
    }

    pub async fn list_events_by_run_id(&self, run_id: Uuid) -> Result<Vec<TaskRunEvent>, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.list_events_by_run_id(run_id) => result,
        }
    }

    async fn find_step_by_run_and_key(
        &self,
        run_id: Uuid,
        step_key: &str,
    ) -> Result<TaskRunStep, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.find_step_by_run_and_key(run_id, step_key) => result,
        }
    }
}
