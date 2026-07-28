use std::sync::Arc;

use async_trait::async_trait;

use crate::{core::chat::ChatMessageService, error::AppError};

use super::dto::ChatRequest;

#[async_trait]
pub trait AgentRequestQueue: Send + Sync {
    async fn submit(&self, request: ChatRequest) -> Result<String, AppError>;
}

#[derive(Clone)]
pub struct AgentState {
    chat: Arc<ChatMessageService>,
    requests: Arc<dyn AgentRequestQueue>,
}

impl AgentState {
    pub fn new(chat: Arc<ChatMessageService>, requests: Arc<dyn AgentRequestQueue>) -> Self {
        Self { chat, requests }
    }

    pub fn chat(&self) -> Result<&ChatMessageService, AppError> {
        Ok(&self.chat)
    }

    pub fn requests(&self) -> Result<&dyn AgentRequestQueue, AppError> {
        Ok(self.requests.as_ref())
    }
}
