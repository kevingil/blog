mod config;
mod draft_service;
mod manager;
pub mod metadata;
pub mod prompts;
mod types;
mod writer;

pub use config::{CopilotConfig, CopilotConfigError};
pub use draft_service::{ArticleDraftAdapter, ArticleDraftService};
pub use manager::{ChatPersistencePort, CopilotManager, ManagerError, SourceContextPort};
pub use types::{
    ArtifactPayload, ChatRequest, ChatRequestResponse, FullMessagePayload, ReasoningStep,
    StreamResponse, ToolCallPayload, ToolGroupPayload, ToolStatusPayload, TurnStep,
};
pub use writer::{GeneratedArticle, WriterAgent};
