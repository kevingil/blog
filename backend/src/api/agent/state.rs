use std::sync::Arc;

use async_trait::async_trait;

use crate::{
    core::{
        chat::ChatMessageService, conversation::ConversationService, mcp::McpConnectorService,
        ml::llm::ToolRegistry,
    },
    error::AppError,
};

use super::dto::ChatRequest;

#[async_trait]
pub trait AgentRequestQueue: Send + Sync {
    async fn submit(&self, request: ChatRequest) -> Result<String, AppError>;
}

#[derive(Clone)]
pub struct AgentState {
    chat: Arc<ChatMessageService>,
    requests: Arc<dyn AgentRequestQueue>,
    connectors: Arc<McpConnectorService>,
    conversation: Arc<ConversationService>,
    registry: Arc<ToolRegistry>,
}

impl AgentState {
    pub fn new(
        chat: Arc<ChatMessageService>,
        requests: Arc<dyn AgentRequestQueue>,
        connectors: Arc<McpConnectorService>,
        conversation: Arc<ConversationService>,
        registry: Arc<ToolRegistry>,
    ) -> Self {
        Self {
            chat,
            requests,
            connectors,
            conversation,
            registry,
        }
    }

    pub fn chat(&self) -> Result<&ChatMessageService, AppError> {
        Ok(&self.chat)
    }

    pub fn requests(&self) -> Result<&dyn AgentRequestQueue, AppError> {
        Ok(self.requests.as_ref())
    }

    pub fn connectors(&self) -> Result<&McpConnectorService, AppError> {
        Ok(&self.connectors)
    }

    pub fn conversation(&self) -> Result<&ConversationService, AppError> {
        Ok(&self.conversation)
    }

    pub fn registry(&self) -> Result<&ToolRegistry, AppError> {
        Ok(&self.registry)
    }
}
