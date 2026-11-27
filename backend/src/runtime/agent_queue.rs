use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use serde_json::{Map, Value, json};
use tokio::{sync::mpsc, task::JoinSet};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{
    api::{
        article::{ArticleGenerationQueue, GenerationRequest},
        websocket::{AgentStreamEvent, AgentStreamProvider},
    },
    core::article::ArticleService,
    error::AppError,
    integrations::openai::OpenAiClient,
};

const JOB_QUEUE_CAPACITY: usize = 64;
const EVENT_QUEUE_CAPACITY: usize = 64;
const MAX_CONCURRENT_REQUESTS: usize = 8;
const WRITER_INSTRUCTIONS: &str = "You are the blog writing assistant. Respond with useful, publication-ready prose. Preserve the user's intent and do not claim to have used tools that were not provided.";

struct AgentJob {
    message: String,
    article_id: Uuid,
    events: mpsc::Sender<AgentStreamEvent>,
}

#[derive(Clone)]
pub struct RuntimeAgentQueue {
    jobs: mpsc::Sender<AgentJob>,
    streams: Arc<Mutex<HashMap<String, mpsc::Receiver<AgentStreamEvent>>>>,
}

pub struct AgentQueueWorker {
    jobs: mpsc::Receiver<AgentJob>,
    articles: Arc<ArticleService>,
    openai: Arc<OpenAiClient>,
}

impl RuntimeAgentQueue {
    pub fn new(
        articles: Arc<ArticleService>,
        openai: Arc<OpenAiClient>,
    ) -> (Arc<Self>, AgentQueueWorker) {
        let (jobs, receiver) = mpsc::channel(JOB_QUEUE_CAPACITY);
        let queue = Arc::new(Self {
            jobs,
            streams: Arc::new(Mutex::new(HashMap::new())),
        });
        (
            queue,
            AgentQueueWorker {
                jobs: receiver,
                articles,
                openai,
            },
        )
    }
}

#[async_trait]
impl ArticleGenerationQueue for RuntimeAgentQueue {
    async fn submit(&self, request: GenerationRequest) -> Result<String, AppError> {
        let request_id = Uuid::new_v4().to_string();
        let (events, receiver) = mpsc::channel(EVENT_QUEUE_CAPACITY);
        self.streams
            .lock()
            .map_err(AppError::internal)?
            .insert(request_id.clone(), receiver);
        let job = AgentJob {
            message: request.message,
            article_id: request.article_id,
            events,
        };
        if self.jobs.send(job).await.is_err() {
            self.streams
                .lock()
                .map_err(AppError::internal)?
                .remove(&request_id);
            return Err(AppError::external("article generation queue is closed"));
        }
        Ok(request_id)
    }
}

impl AgentStreamProvider for RuntimeAgentQueue {
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.streams.lock().ok()?.remove(request_id)
    }
}

impl AgentQueueWorker {
    pub async fn run(mut self, cancellation: CancellationToken) -> anyhow::Result<()> {
        let mut tasks = JoinSet::new();
        loop {
            tokio::select! {
                biased;
                () = cancellation.cancelled() => break,
                completed = tasks.join_next(), if !tasks.is_empty() => {
                    if let Some(Err(error)) = completed {
                        tracing::error!(%error, "article generation task failed");
                    }
                }
                job = self.jobs.recv(), if tasks.len() < MAX_CONCURRENT_REQUESTS => {
                    let Some(job) = job else { break };
                    let articles = self.articles.clone();
                    let openai = self.openai.clone();
                    tasks.spawn(async move {
                        process_job(job, articles, openai).await;
                    });
                }
            }
        }
        self.jobs.close();
        while let Some(job) = self.jobs.recv().await {
            send_error(job.events, "application is shutting down").await;
        }
        while let Some(result) = tasks.join_next().await {
            if let Err(error) = result {
                tracing::error!(%error, "article generation task failed during shutdown");
            }
        }
        Ok(())
    }
}

async fn process_job(job: AgentJob, articles: Arc<ArticleService>, openai: Arc<OpenAiClient>) {
    let _ = send_event(&job.events, json!({"type": "turn_started"})).await;
    match openai
        .generate_text(WRITER_INSTRUCTIONS, &job.message)
        .await
    {
        Ok(content) => {
            if articles
                .update_generated_draft(job.article_id, &content)
                .await
                .is_err()
            {
                send_error(job.events, "failed to persist generated article").await;
                return;
            }
            let _ = send_event(
                &job.events,
                json!({"type": "content_delta", "content": content}),
            )
            .await;
            let _ = send_event(&job.events, json!({"type": "done", "done": true})).await;
        }
        Err(_) => send_error(job.events, "assistant request failed").await,
    }
}

async fn send_error(events: mpsc::Sender<AgentStreamEvent>, error: &str) {
    let _ = send_event(
        &events,
        json!({"type": "error", "error": error, "done": true}),
    )
    .await;
}

async fn send_event(events: &mpsc::Sender<AgentStreamEvent>, value: Value) -> Result<(), AppError> {
    let fields: Map<String, Value> = value
        .as_object()
        .cloned()
        .ok_or(AppError::internal("agent event payload was not an object"))?;
    events
        .send(AgentStreamEvent::new(fields))
        .await
        .map_err(AppError::external)
}
