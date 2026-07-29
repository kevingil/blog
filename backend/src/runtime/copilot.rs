use std::{
    mem,
    sync::{Arc, Mutex},
    time::Duration,
};

use async_trait::async_trait;
use tokio::{sync::mpsc, task::JoinSet};

use crate::{
    api::{
        agent::{AgentRequestQueue, ChatRequest},
        websocket::{AgentStreamEvent, AgentStreamProvider},
    },
    core::copilot::{CopilotManager, ManagerError},
    error::AppError,
};

pub struct CopilotRuntime {
    manager: Arc<CopilotManager>,
    bridges: Mutex<JoinSet<()>>,
}

impl CopilotRuntime {
    pub fn new(manager: Arc<CopilotManager>) -> Arc<Self> {
        Arc::new(Self {
            manager,
            bridges: Mutex::new(JoinSet::new()),
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
            })
            .await
            .map_err(manager_error)
    }
}

impl AgentStreamProvider for CopilotRuntime {
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.prune_bridges().ok()?;
        let mut source = self.manager.take_response_stream(request_id).ok()?;
        let (sender, receiver) = mpsc::channel(100);
        let manager = self.manager.clone();
        let task_request_id = request_id.to_owned();
        let task = async move {
            while let Some(event) = source.recv().await {
                let value = match serde_json::to_value(event) {
                    Ok(value) => value,
                    Err(error) => {
                        tracing::error!(%error, "failed to serialize copilot stream event");
                        break;
                    }
                };
                let Some(event) = AgentStreamEvent::from_value(value) else {
                    tracing::error!("copilot stream event was not a JSON object");
                    break;
                };
                if sender.send(event).await.is_err() {
                    let _ = manager.cancel_request(&task_request_id);
                    break;
                }
            }
        };
        self.bridges.lock().ok()?.spawn(task);
        Some(receiver)
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
