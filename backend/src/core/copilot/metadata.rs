use chrono::Utc;
use serde_json::Value;
use sha2::{Digest, Sha256};

use crate::{
    core::chat::{
        ArtifactInfo, ChainOfThoughtStep, MessageContext, MessageMetadata, ThinkingBlock,
        ToolExecution, UserAction,
    },
    error::AppError,
};

pub const ARTIFACT_STATUS_PENDING: &str = "pending";
pub const ARTIFACT_STATUS_ACCEPTED: &str = "accepted";
pub const ARTIFACT_STATUS_REJECTED: &str = "rejected";
pub const ARTIFACT_STATUS_APPLIED: &str = "applied";
pub const ARTIFACT_TYPE_CODE_EDIT: &str = "code_edit";
pub const ARTIFACT_TYPE_REWRITE: &str = "rewrite";
pub const ARTIFACT_TYPE_SUGGESTION: &str = "suggestion";
pub const ARTIFACT_TYPE_CONTENT_GENERATION: &str = "content_generation";
pub const ARTIFACT_TYPE_IMAGE_PROMPT: &str = "image_prompt";
pub const USER_ACTION_ACCEPT: &str = "accept";
pub const USER_ACTION_REJECT: &str = "reject";
pub const USER_ACTION_MODIFY: &str = "modify";

pub struct MetadataBuilder {
    metadata: MessageMetadata,
}

impl Default for MetadataBuilder {
    fn default() -> Self {
        Self::new()
    }
}

impl MetadataBuilder {
    pub fn new() -> Self {
        Self {
            metadata: MessageMetadata::default(),
        }
    }

    pub fn with_artifact(mut self, artifact: ArtifactInfo) -> Self {
        self.metadata.artifact = Some(artifact);
        self
    }

    pub fn with_tool_execution(mut self, execution: ToolExecution) -> Self {
        self.metadata.tool_execution = Some(execution);
        self
    }

    pub fn with_context(mut self, context: MessageContext) -> Self {
        self.metadata.context = Some(context);
        self
    }

    pub fn with_user_action(mut self, action: UserAction) -> Self {
        self.metadata.user_action = Some(action);
        self
    }

    pub fn with_thinking(mut self, thinking: ThinkingBlock) -> Self {
        self.metadata.thinking = Some(thinking);
        self
    }

    pub fn with_steps(mut self, steps: Vec<ChainOfThoughtStep>) -> Self {
        self.metadata.steps = steps;
        self
    }

    pub fn build(self) -> MessageMetadata {
        self.metadata
    }
}

pub fn message_context(
    article_id: impl Into<String>,
    session_id: impl Into<String>,
    request_id: impl Into<String>,
    user_id: impl Into<String>,
) -> MessageContext {
    MessageContext {
        article_id: article_id.into(),
        session_id: session_id.into(),
        request_id: request_id.into(),
        document_version: String::new(),
        document_hash: String::new(),
        user_id: user_id.into(),
    }
}

pub fn with_document_hash(mut context: MessageContext, content: &str) -> MessageContext {
    context.document_hash = hex::encode(Sha256::digest(content.as_bytes()));
    context
}

pub fn tool_execution(
    tool_name: impl Into<String>,
    tool_id: impl Into<String>,
    input: Value,
    output: Value,
    duration_ms: i64,
    error: Option<&str>,
) -> ToolExecution {
    ToolExecution {
        tool_name: tool_name.into(),
        tool_id: tool_id.into(),
        input,
        output,
        error: error.unwrap_or_default().to_owned(),
        duration_ms,
        executed_at: Utc::now(),
        success: error.is_none(),
    }
}

pub fn user_action(
    action: impl Into<String>,
    artifact_id: impl Into<String>,
    feedback: impl Into<String>,
    reason: impl Into<String>,
) -> UserAction {
    UserAction {
        action: action.into(),
        timestamp: Utc::now(),
        artifact_id: artifact_id.into(),
        feedback: feedback.into(),
        reason: reason.into(),
    }
}

pub fn validate(metadata: Option<&MessageMetadata>) -> Result<(), AppError> {
    let Some(metadata) = metadata else {
        return Ok(());
    };
    if let Some(artifact) = &metadata.artifact {
        if artifact.id.is_empty() {
            return invalid("invalid artifact: artifact ID is required");
        }
        if artifact.artifact_type.is_empty() {
            return invalid("invalid artifact: artifact type is required");
        }
        if !matches!(
            artifact.artifact_type.as_str(),
            ARTIFACT_TYPE_CODE_EDIT
                | ARTIFACT_TYPE_REWRITE
                | ARTIFACT_TYPE_SUGGESTION
                | ARTIFACT_TYPE_CONTENT_GENERATION
                | ARTIFACT_TYPE_IMAGE_PROMPT
        ) {
            return invalid(&format!(
                "invalid artifact: invalid artifact type: {}",
                artifact.artifact_type
            ));
        }
        if !artifact.status.is_empty()
            && !matches!(
                artifact.status.as_str(),
                ARTIFACT_STATUS_PENDING
                    | ARTIFACT_STATUS_ACCEPTED
                    | ARTIFACT_STATUS_REJECTED
                    | ARTIFACT_STATUS_APPLIED
            )
        {
            return invalid(&format!(
                "invalid artifact: invalid artifact status: {}",
                artifact.status
            ));
        }
    }
    if let Some(action) = &metadata.user_action {
        if action.action.is_empty() {
            return invalid("invalid user action: action is required");
        }
        if !matches!(
            action.action.as_str(),
            USER_ACTION_ACCEPT | USER_ACTION_REJECT | USER_ACTION_MODIFY
        ) {
            return invalid(&format!(
                "invalid user action: invalid action: {}",
                action.action
            ));
        }
    }
    Ok(())
}

fn invalid<T>(message: &str) -> Result<T, AppError> {
    Err(AppError::InvalidInput(format!(
        "Invalid metadata: {message}"
    )))
}
