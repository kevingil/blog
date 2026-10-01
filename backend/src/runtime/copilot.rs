use std::{
    collections::HashMap,
    mem,
    sync::{Arc, Mutex},
    time::Duration,
};

use async_trait::async_trait;
use serde_json::Value;
use tokio::{sync::mpsc, task::JoinSet};

use crate::{
    api::{
        agent::{AgentRequestQueue, ChatRequest},
        websocket::{AgentStreamEvent, AgentStreamProvider},
    },
    core::{
        copilot::{CopilotManager, ManagerError},
        live::{LiveHarness, LiveTurn, LiveTurnHandle},
    },
    error::AppError,
};

pub struct CopilotRuntime {
    manager: Arc<CopilotManager>,
    bridges: Mutex<JoinSet<()>>,
    shared: Mutex<HashMap<String, Arc<Mutex<SharedTurn>>>>,
}

struct SharedTurn {
    history: Vec<AgentStreamEvent>,
    subscribers: Vec<mpsc::UnboundedSender<AgentStreamEvent>>,
    finished: bool,
}

impl CopilotRuntime {
    pub fn new(manager: Arc<CopilotManager>) -> Arc<Self> {
        Arc::new(Self {
            manager,
            bridges: Mutex::new(JoinSet::new()),
            shared: Mutex::new(HashMap::new()),
        })
    }

    pub async fn shutdown(&self, shutdown_timeout: Duration) -> Result<(), ManagerError> {
        self.manager.shutdown(shutdown_timeout).await?;
        let mut bridges = self
            .bridges
            .lock()
            .map(|mut bridges| mem::take(&mut *bridges))
            .map_err(|_| ManagerError::Dependency)?;
        if tokio::time::timeout(shutdown_timeout, async {
            while bridges.join_next().await.is_some() {}
        })
        .await
        .is_err()
        {
            bridges.abort_all();
            while bridges.join_next().await.is_some() {}
        }
        Ok(())
    }

    fn prune_bridges(&self) -> Result<(), ManagerError> {
        let mut bridges = self.bridges.lock().map_err(|_| ManagerError::Dependency)?;
        while let Some(result) = bridges.try_join_next() {
            if let Err(error) = result {
                tracing::warn!(%error, "copilot stream bridge failed");
            }
        }
        Ok(())
    }
}

#[async_trait]
impl AgentRequestQueue for CopilotRuntime {
    async fn submit(&self, request: ChatRequest) -> Result<String, AppError> {
        self.manager
            .submit(crate::core::copilot::ChatRequest {
                message: request.message,
                document_content: request.document_content,
                document_markdown: request.document_markdown,
                article_id: request.article_id,
                channel: request.channel,
            })
            .await
            .map_err(manager_error)
    }
}

impl AgentStreamProvider for CopilotRuntime {
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.forward_shared(request_id)
    }
}

impl CopilotRuntime {
    fn forward_shared(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.prune_bridges().ok()?;
        let shared = self.shared_turn(request_id)?;
        let mut source = attach_subscriber(&shared)?;
        let (sender, receiver) = mpsc::channel(100);
        let manager = self.manager.clone();
        let task_request_id = request_id.to_owned();
        let task = async move {
            while let Some(event) = source.recv().await {
                if sender.send(event).await.is_err() {
                    let others = shared
                        .lock()
                        .map(|turn| turn.subscribers.len())
                        .unwrap_or(0);
                    if others <= 1 {
                        let _ = manager.cancel_request(&task_request_id);
                    }
                    break;
                }
            }
        };
        self.bridges.lock().ok()?.spawn(task);
        Some(receiver)
    }

    fn shared_turn(&self, request_id: &str) -> Option<Arc<Mutex<SharedTurn>>> {
        let mut turns = self.shared.lock().ok()?;
        if let Some(existing) = turns.get(request_id) {
            return Some(existing.clone());
        }
        let mut source = self.manager.take_response_stream(request_id).ok()?;
        let shared = Arc::new(Mutex::new(SharedTurn {
            history: Vec::new(),
            subscribers: Vec::new(),
            finished: false,
        }));
        turns.insert(request_id.to_owned(), shared.clone());
        drop(turns);
        let publisher = shared.clone();
        let task = async move {
            while let Some(event) = source.recv().await {
                let value = match serde_json::to_value(&event) {
                    Ok(value) => value,
                    Err(error) => {
                        tracing::error!(%error, "failed to serialize copilot stream event");
                        break;
                    }
                };
                let Some(agent_event) = AgentStreamEvent::from_value(value) else {
                    tracing::error!("copilot stream event was not a JSON object");
                    break;
                };
                let Ok(mut turn) = publisher.lock() else {
                    break;
                };
                turn.history.push(agent_event.clone());
                turn.subscribers
                    .retain(|subscriber| subscriber.send(agent_event.clone()).is_ok());
            }
            if let Ok(mut turn) = publisher.lock() {
                turn.finished = true;
                turn.subscribers.clear();
            }
        };
        self.bridges.lock().ok()?.spawn(task);
        Some(shared)
    }
}

fn attach_subscriber(
    shared: &Arc<Mutex<SharedTurn>>,
) -> Option<mpsc::UnboundedReceiver<AgentStreamEvent>> {
    let (sender, receiver) = mpsc::unbounded_channel();
    let mut turn = shared.lock().ok()?;
    for event in &turn.history {
        let _ = sender.send(event.clone());
    }
    if !turn.finished {
        turn.subscribers.push(sender);
    }
    Some(receiver)
}

pub struct CopilotLiveHarness {
    runtime: Arc<CopilotRuntime>,
}

impl CopilotLiveHarness {
    pub fn new(runtime: Arc<CopilotRuntime>) -> Arc<Self> {
        Arc::new(Self { runtime })
    }
}

#[async_trait]
impl LiveHarness for CopilotLiveHarness {
    async fn run_turn(&self, turn: LiveTurn) -> Result<LiveTurnHandle, AppError> {
        let request_id = self
            .runtime
            .submit(ChatRequest {
                message: turn.message,
                document_content: turn.document_content,
                document_markdown: turn.document_markdown,
                article_id: turn.article_id,
                channel: "live".to_owned(),
            })
            .await?;
        let events = self
            .runtime
            .forward_shared(&request_id)
            .ok_or_else(|| AppError::Conflict("request stream already taken".to_owned()))?;
        let (sender, receiver) = mpsc::channel(64);
        tokio::spawn(async move {
            let mut events = events;
            while let Some(event) = events.recv().await {
                let value = Value::Object(event.fields().clone());
                if sender.send(value).await.is_err() {
                    break;
                }
            }
        });
        Ok(LiveTurnHandle {
            request_id,
            events: receiver,
        })
    }
}

pub struct CombinedAgentStreamProvider {
    copilot: Arc<CopilotRuntime>,
    article_generation: Arc<dyn AgentStreamProvider>,
}

impl CombinedAgentStreamProvider {
    pub fn new(
        copilot: Arc<CopilotRuntime>,
        article_generation: Arc<dyn AgentStreamProvider>,
    ) -> Arc<Self> {
        Arc::new(Self {
            copilot,
            article_generation,
        })
    }
}

impl AgentStreamProvider for CombinedAgentStreamProvider {
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.copilot
            .take_response_stream(request_id)
            .or_else(|| self.article_generation.take_response_stream(request_id))
    }
}

fn manager_error(error: ManagerError) -> AppError {
    match error {
        ManagerError::MessageRequired => {
            AppError::InvalidInput("message is a required field".to_owned())
        }
        ManagerError::ArticleRequired => {
            AppError::InvalidInput("articleId is a required field".to_owned())
        }
        ManagerError::InvalidArticle => {
            AppError::InvalidInput("Invalid article ID format".to_owned())
        }
        ManagerError::ConcurrencyLimit(limit) => {
            AppError::Conflict(format!("maximum concurrent requests reached ({limit})"))
        }
        ManagerError::RequestNotFound => AppError::NotFound,
        ManagerError::StreamAlreadyTaken => {
            AppError::Conflict("request stream already taken".to_owned())
        }
        ManagerError::Dependency | ManagerError::ShutdownTimeout(_) => AppError::Internal,
    }
}
