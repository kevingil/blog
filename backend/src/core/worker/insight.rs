use std::sync::Arc;

use async_trait::async_trait;
use serde_json::{Map, json};
use tokio_util::sync::CancellationToken;

use crate::core::{datasource::DataSource, insight::InsightTopic};

use super::{StatusService, Worker, WorkerContext, WorkerFailure, WorkerResult, WorkerState};

const INSIGHT_WORKER_NAME: &str = "insight";

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum InsightTopicResult {
    Created,
    SkippedInsufficient,
    SkippedRecent,
}

#[async_trait]
pub trait InsightGenerationPort: Send + Sync {
    fn is_configured(&self) -> bool;

    async fn topics(&self) -> Result<Vec<InsightTopic>, WorkerFailure>;

    async fn sources(&self) -> Result<Vec<DataSource>, WorkerFailure>;

    async fn generate_for_source(
        &self,
        source: &DataSource,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure>;

    async fn generate_for_topic(
        &self,
        topic: &InsightTopic,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure>;
}

pub struct InsightWorker {
    status: Arc<StatusService>,
    generator: Option<Arc<dyn InsightGenerationPort>>,
}

impl InsightWorker {
    pub fn new(
        status: Arc<StatusService>,
        generator: Option<Arc<dyn InsightGenerationPort>>,
    ) -> Result<Self, WorkerFailure> {
        if generator
            .as_ref()
            .is_some_and(|generator| !generator.is_configured())
        {
            return Err(WorkerFailure::new(
                "configured insight provider failed validation",
            ));
        }
        Ok(Self { status, generator })
    }
}

#[async_trait]
impl Worker for InsightWorker {
    fn name(&self) -> &str {
        INSIGHT_WORKER_NAME
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        let Some(generator) = self.generator.as_ref() else {
            self.status.update_status(
                self.name(),
                WorkerState::Running,
                100,
                "LLM not configured, skipping",
            );
            return Ok(WorkerResult::warning(
                "LLM provider not configured, insight generation skipped",
                vec!["LLM provider not configured".to_owned()],
            ));
        };
        self.status
            .update_status(self.name(), WorkerState::Running, 0, "Fetching trackers...");
        let sources = tokio::select! {
            biased;
            () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = generator.sources() => result
                .map_err(|error| WorkerFailure::new(format!("failed to get sources: {error}")))?,
        };
        let topics = tokio::select! {
            biased;
            () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = generator.topics() => result
                .map_err(|error| WorkerFailure::new(format!("failed to get topics: {error}")))?,
        };
        if sources.is_empty() && topics.is_empty() {
            self.status
                .update_status(self.name(), WorkerState::Running, 100, "No trackers due");
            return Ok(WorkerResult::warning(
                "No trackers due for insight generation",
                vec!["No trackers were due".to_owned()],
            ));
        }
        let total = i32::try_from(sources.len() + topics.len())
            .map_err(|_| WorkerFailure::new("too many topics"))?;
        self.status.set_progress(
            self.name(),
            0,
            total,
            format!("Found {total} trackers to check"),
        );
        let mut skipped_insufficient = 0_i32;
        let mut skipped_recent = 0_i32;
        let mut failed = 0_i32;
        let mut created = 0_i32;
        let mut index = 0_usize;
        for source in sources {
            if context.cancellation().is_cancelled() {
                return Err(WorkerFailure::new("operation cancelled"));
            }
            let done = i32::try_from(index).map_err(|_| WorkerFailure::new("too many topics"))?;
            index += 1;
            self.status.set_progress(
                self.name(),
                done,
                total,
                format!("Checking domain: {}", source.name),
            );
            let generated = tokio::select! {
                biased;
                () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
                result = generator.generate_for_source(&source, context.cancellation()) => result,
            };
            match generated {
                Ok(InsightTopicResult::Created) => created += 1,
                Ok(InsightTopicResult::SkippedInsufficient) => skipped_insufficient += 1,
                Ok(InsightTopicResult::SkippedRecent) => skipped_recent += 1,
                Err(error) => {
                    failed += 1;
                    let mut meta_data = Map::new();
                    meta_data.insert("data_source_id".to_owned(), json!(source.id));
                    meta_data.insert("error".to_owned(), json!(error.to_string()));
                    let _ = context
                        .task_run()
                        .record_event(
                            None,
                            "topic_failed",
                            "warning",
                            format!("Domain {} failed", source.name),
                            meta_data,
                        )
                        .await;
                }
            }
        }
        for topic in topics {
            if context.cancellation().is_cancelled() {
                return Err(WorkerFailure::new("operation cancelled"));
            }
            let done = i32::try_from(index).map_err(|_| WorkerFailure::new("too many topics"))?;
            index += 1;
            self.status.set_progress(
                self.name(),
                done,
                total,
                format!("Generating insight for: {}", topic.name),
            );
            let generated = tokio::select! {
                biased;
                () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
                result = generator.generate_for_topic(&topic, context.cancellation()) => result,
            };
            match generated {
                Ok(InsightTopicResult::Created) => created += 1,
                Ok(InsightTopicResult::SkippedInsufficient) => skipped_insufficient += 1,
                Ok(InsightTopicResult::SkippedRecent) => skipped_recent += 1,
                Err(error) => {
                    failed += 1;
                    let mut meta_data = Map::new();
                    meta_data.insert("topic_id".to_owned(), json!(topic.id));
                    meta_data.insert("error".to_owned(), json!(error.to_string()));
                    let _ = context
                        .task_run()
                        .record_event(
                            None,
                            "topic_failed",
                            "warning",
                            format!("Topic {} failed", topic.name),
                            meta_data,
                        )
                        .await;
                }
            }
        }
        self.status.set_progress(
            self.name(),
            total,
            total,
            format!("Completed processing {total} topics"),
        );
        let mut result = if created == 0 || failed > 0 {
            let mut warnings = Vec::new();
            if created == 0 {
                warnings.push("No insights were created".to_owned());
            }
            if failed > 0 {
                warnings.push(format!("{failed} topics failed during generation"));
            }
            WorkerResult::warning(
                format!("Processed {total} topics and created {created} insights"),
                warnings,
            )
        } else {
            WorkerResult::completed(format!("Created {created} insights from {total} topics"))
        };
        result
            .metrics
            .insert("topics_considered".to_owned(), json!(total));
        result.metrics.insert(
            "topics_skipped_insufficient_content".to_owned(),
            json!(skipped_insufficient),
        );
        result.metrics.insert(
            "topics_skipped_recent_insight".to_owned(),
            json!(skipped_recent),
        );
        result
            .metrics
            .insert("topics_failed".to_owned(), json!(failed));
        result
            .metrics
            .insert("insights_created".to_owned(), json!(created));
        result
            .output_summary
            .insert("insights_created".to_owned(), json!(created));
        Ok(result)
    }
}
