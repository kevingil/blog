mod agent;
mod inventory;
mod message;
mod models;
mod prompt;
mod provider;
mod session;
mod tools;

pub use agent::{Agent, AgentError, AgentEvent, AgentEventType, AgentRun};
pub use inventory::{DORMANT_CAPABILITIES, DormantCapability, DormantCapabilityKind};
pub use message::{
    Attachment, BinaryContent, ContentPart, FinishReason, LlmMessage, MessageRole, TextContent,
    ToolCall, ToolResult,
};
pub use models::{Model, ModelId, ModelProvider};
pub use prompt::copilot_prompt;
pub use provider::{
    Provider, ProviderError, ProviderEvent, ProviderEventType, ProviderResponse, TokenUsage,
};
pub use session::{InMemorySessionStore, Session, SessionStore};
pub use tools::{
    AnswerCitation, AnswerResponse, ArtifactHint, AskQuestionTool, DraftSaver,
    GenerateImagePromptTool, GetRelevantSourcesTool, ReadDocumentTool, ReplaceLinesTool,
    ResearchPort, SearchWebSourcesTool, SelectSourcesForEditTool, SourceResource,
    SourceResourcePort, SourceSelection, Tool, ToolCallRequest, ToolContext, ToolInfo,
    ToolResponse, ToolResponseType, WebSearchResponse, WebSearchResult,
};
