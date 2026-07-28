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
        agent::{AgentRequestQueue, ChatRequest},
        article::{ArticleGenerationQueue, GenerationRequest},
        websocket::{AgentStreamEvent, AgentStreamProvider},
    },
    core::{article::ArticleService, chat::ChatMessageService},
    error::AppError,
    integrations::openai::OpenAiClient,
};

const JOB_QUEUE_CAPACITY: usize = 64;
const EVENT_QUEUE_CAPACITY: usize = 64;
const MAX_CONCURRENT_REQUESTS: usize = 8;
const WRITER_INSTRUCTIONS: &str = "You are the blog writing assistant. Respond with useful, publication-ready prose. Preserve the user's intent and do not claim to have used tools that were not provided.";

struct AgentJob {
    request: ChatRequest,
    events: mpsc::Sender<AgentStreamEvent>,
}

#[derive(Clone)]
pub struct RuntimeAgentQueue {
    jobs: mpsc::Sender<AgentJob>,
    streams: Arc<Mutex<HashMap<String, mpsc::Receiver<AgentStreamEvent>>>>,
}

pub struct AgentQueueWorker {
    jobs: mpsc::Receiver<AgentJob>,
    chat: Arc<ChatMessageService>,
    articles: Arc<ArticleService>,
    openai: Arc<OpenAiClient>,
}

impl RuntimeAgentQueue {
    pub fn new(
        chat: Arc<ChatMessageService>,
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
                chat,
                articles,
                openai,
            },
        )
    }

    async fn submit_request(&self, request: ChatRequest) -> Result<String, AppError> {
        let request_id = Uuid::new_v4().to_string();
        let (events, receiver) = mpsc::channel(EVENT_QUEUE_CAPACITY);
        self.streams
            .lock()
            .map_err(|_| AppError::Internal)?
            .insert(request_id.clone(), receiver);
        if self.jobs.send(AgentJob { request, events }).await.is_err() {
            self.streams
                .lock()
                .map_err(|_| AppError::Internal)?
                .remove(&request_id);
            return Err(AppError::External);
        }
        Ok(request_id)
    }
}

#[async_trait]
impl AgentRequestQueue for RuntimeAgentQueue {
    async fn submit(&self, request: ChatRequest) -> Result<String, AppError> {
        self.submit_request(request).await
    }
}

#[async_trait]
impl ArticleGenerationQueue for RuntimeAgentQueue {
    async fn submit(&self, request: GenerationRequest) -> Result<String, AppError> {
        self.submit_request(ChatRequest {
            message: request.message,
            document_content: String::new(),
            document_markdown: String::new(),
            article_id: request.article_id.to_string(),
        })
        .await
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
                        tracing::error!(%error, "agent request task failed");
                    }
                }
                job = self.jobs.recv(), if tasks.len() < MAX_CONCURRENT_REQUESTS => {
                    let Some(job) = job else { break };
                    let chat = self.chat.clone();
                    let articles = self.articles.clone();
                    let openai = self.openai.clone();
                    tasks.spawn(async move {
                        process_job(job, chat, articles, openai).await;
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
                tracing::error!(%error, "agent request task failed during shutdown");
            }
        }
        Ok(())
    }
}

async fn process_job(
    job: AgentJob,
    chat: Arc<ChatMessageService>,
    articles: Arc<ArticleService>,
    openai: Arc<OpenAiClient>,
) {
    let article_id = match Uuid::parse_str(&job.request.article_id) {
        Ok(value) => value,
        Err(_) => {
            send_error(job.events, "Invalid article ID format").await;
            return;
        }
    };
    if chat
        .save_message(article_id, "user", job.request.message.clone(), None)
        .await
        .is_err()
    {
        send_error(job.events, "failed to persist user message").await;
        return;
    }
    let _ = send_event(&job.events, json!({"type": "turn_started"})).await;
    let prompt = if job.request.document_markdown.is_empty() {
        job.request.message.clone()
    } else {
        format!(
            "Current document:\n{}\n\nUser request:\n{}",
            job.request.document_markdown, job.request.message
        )
    };
    match openai.generate_text(WRITER_INSTRUCTIONS, &prompt).await {
        Ok(content) => {
            if articles
                .update_generated_draft(article_id, &content)
                .await
                .is_err()
            {
                send_error(job.events, "failed to persist generated article").await;
                return;
            }
            if chat
                .save_message(article_id, "assistant", content.clone(), None)
                .await
                .is_err()
            {
                send_error(job.events, "failed to persist assistant message").await;
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
    let fields: Map<String, Value> = value.as_object().cloned().ok_or(AppError::Internal)?;
    events
        .send(AgentStreamEvent::new(fields))
        .await
        .map_err(|_| AppError::External)
}
