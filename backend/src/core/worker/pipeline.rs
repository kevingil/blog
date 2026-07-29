use std::sync::{Arc, Weak};

use async_trait::async_trait;
use serde_json::{Map, Value};

use super::manager::WorkerOutcome;
use super::{
    RunMetadata, StatusService, Worker, WorkerContext, WorkerFailure, WorkerManager, WorkerResult,
    WorkerResultStatus, WorkerState,
};

pub const PIPELINE_WORKER_NAME: &str = "pipeline";

pub struct PipelineWorker {
    manager: Weak<WorkerManager>,
    status: Arc<StatusService>,
}

impl PipelineWorker {
    pub fn new(manager: Weak<WorkerManager>, status: Arc<StatusService>) -> Self {
        Self { manager, status }
    }

    async fn run_step(
        &self,
        context: &WorkerContext,
        worker_name: &str,
        label: &str,
        base_progress: i32,
        progress_span: i32,
    ) -> Result<WorkerResultStatus, WorkerFailure> {
        self.status.update_status(
            PIPELINE_WORKER_NAME,
            WorkerState::Running,
            base_progress,
            label,
        );
        let _ = context
            .task_run()
            .start_step(worker_name, worker_name, Some(label.to_owned()))
            .await;
        let manager = self
            .manager
            .upgrade()
            .ok_or_else(|| WorkerFailure::new("worker manager not configured"))?;
        let metadata = RunMetadata {
            parent_run_id: context.task_run().run_id(),
            trigger_source: "workflow".to_owned(),
            ..RunMetadata::default()
        };
        let child = manager
            .run_and_wait(worker_name, metadata, context.cancellation())
            .await;
        let (child_run_id, outcome) = match child {
            Ok(value) => value,
            Err(error) => {
                let message = error.to_string();
                let _ = context
                    .task_run()
                    .finish_step(
                        worker_name,
                        "failed",
                        Some(format!("{label} failed")),
                        Some(message.clone()),
                        Map::new(),
                    )
                    .await;
                return Err(WorkerFailure::new(message));
            }
        };
        let result: Result<WorkerResultStatus, WorkerFailure> = match outcome {
            WorkerOutcome::Completed(status) => Ok(status),
            WorkerOutcome::Failed(error) => Err(WorkerFailure::new(error)),
            WorkerOutcome::Cancelled => Err(WorkerFailure::new("operation cancelled")),
        };
        match &result {
            Ok(status) => {
                self.status.update_status(
                    PIPELINE_WORKER_NAME,
                    WorkerState::Running,
                    base_progress + progress_span,
                    format!("{label} complete"),
                );
                let summary = if *status == WorkerResultStatus::Warning {
                    format!("{label} completed with warnings")
                } else {
                    format!("{label} complete")
                };
                let mut metrics = Map::new();
                if let Some(child_run_id) = child_run_id {
                    metrics.insert(
                        "child_run_id".to_owned(),
                        Value::String(child_run_id.to_string()),
                    );
                }
                let _ = context
                    .task_run()
                    .finish_step(worker_name, status.as_str(), Some(summary), None, metrics)
                    .await;
            }
            Err(error) => {
                self.status.set_error(
                    PIPELINE_WORKER_NAME,
                    format!("{label} failed: {}", error.message()),
                );
                let _ = context
                    .task_run()
                    .finish_step(
                        worker_name,
                        "failed",
                        Some(format!("{label} failed")),
                        Some(error.to_string()),
                        Map::new(),
                    )
                    .await;
            }
        }
        result
    }
}

#[async_trait]
impl Worker for PipelineWorker {
    fn name(&self) -> &str {
        PIPELINE_WORKER_NAME
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        let crawl = self
            .run_step(&context, "crawl", "Crawling sources", 0, 50)
            .await
            .map_err(|error| WorkerFailure::new(format!("crawl step failed: {error}")))?;
        let insight = self
            .run_step(&context, "insight", "Generating insights", 50, 50)
            .await
            .map_err(|error| WorkerFailure::new(format!("insight step failed: {error}")))?;
        self.status.update_status(
            PIPELINE_WORKER_NAME,
            WorkerState::Running,
            100,
            "Pipeline complete",
        );

        let warning =
            crawl == WorkerResultStatus::Warning || insight == WorkerResultStatus::Warning;
        let mut result = if warning {
            WorkerResult::warning(
                "Pipeline completed with warnings",
                [
                    (crawl == WorkerResultStatus::Warning)
                        .then_some("Crawl completed with warnings".to_owned()),
                    (insight == WorkerResultStatus::Warning)
                        .then_some("Insight generation completed with warnings".to_owned()),
                ]
                .into_iter()
                .flatten()
                .collect(),
            )
        } else {
            WorkerResult::completed("Pipeline completed successfully")
        };
        result.metrics.insert(
            "crawl_status".to_owned(),
            Value::String(crawl.as_str().to_owned()),
        );
        result.metrics.insert(
            "insight_status".to_owned(),
            Value::String(insight.as_str().to_owned()),
        );
        Ok(result)
    }
}
