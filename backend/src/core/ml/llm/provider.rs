use async_trait::async_trait;
use thiserror::Error;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use super::{LlmMessage, Model, Tool, ToolCall};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProviderEventType {
    ContentStart,
    ToolUseStart,
    ToolUseDelta,
    ToolUseStop,
    ContentDelta,
    ThinkingDelta,
    ContentStop,
    Complete,
    Error,
    Warning,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct TokenUsage {
    pub input_tokens: i64,
    pub output_tokens: i64,
    pub cache_creation_tokens: i64,
    pub cache_read_tokens: i64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ProviderResponse {
    pub content: String,
    pub reasoning: String,
    pub tool_calls: Vec<ToolCall>,
    pub usage: TokenUsage,
    pub finish_reason: super::FinishReason,
}

#[derive(Debug, Clone)]
pub struct ProviderEvent {
    pub event_type: ProviderEventType,
    pub content: String,
    pub thinking: String,
    pub response: Option<ProviderResponse>,
    pub tool_call: Option<ToolCall>,
    pub error: Option<ProviderError>,
}

impl ProviderEvent {
    pub fn content_delta(content: impl Into<String>) -> Self {
        Self {
            event_type: ProviderEventType::ContentDelta,
            content: content.into(),
            thinking: String::new(),
            response: None,
            tool_call: None,
            error: None,
        }
    }

    pub fn thinking_delta(thinking: impl Into<String>) -> Self {
        Self {
            event_type: ProviderEventType::ThinkingDelta,
            content: String::new(),
            thinking: thinking.into(),
            response: None,
            tool_call: None,
            error: None,
        }
    }

    pub fn complete(response: ProviderResponse) -> Self {
        Self {
            event_type: ProviderEventType::Complete,
            content: String::new(),
            thinking: String::new(),
            response: Some(response),
            tool_call: None,
            error: None,
        }
    }
}

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum ProviderError {
    #[error("request cancelled")]
    Cancelled,
    #[error("provider request failed: {0}")]
    Request(String),
    #[error("provider stream ended without a completion event")]
    MissingCompletion,
}

#[async_trait]
pub trait Provider: Send + Sync {
    fn model(&self) -> Model;
    fn system_message(&self) -> &str;

    async fn stream_response(
        &self,
        cancellation: CancellationToken,
        messages: Vec<LlmMessage>,
        tools: Vec<std::sync::Arc<dyn Tool>>,
    ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError>;
}
