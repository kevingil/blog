use std::sync::Arc;

use serde_json::{Value, json};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::error::AppError;

use super::{
    ArtifactInfo, ChatMessage, ChatMessageRepository, MessageMetadata, UserAction,
    types::{
        ARTIFACT_STATUS_ACCEPTED, ARTIFACT_STATUS_APPLIED, ARTIFACT_STATUS_REJECTED,
        ARTIFACT_TYPE_CODE_EDIT, ARTIFACT_TYPE_CONTENT_GENERATION, ARTIFACT_TYPE_IMAGE_PROMPT,
        ARTIFACT_TYPE_REWRITE, ARTIFACT_TYPE_SUGGESTION, USER_ACTION_ACCEPT, USER_ACTION_MODIFY,
        USER_ACTION_REJECT,
    },
};

const DEFAULT_HISTORY_LIMIT: i64 = 50;
const MAX_HISTORY_LIMIT: i64 = 200;

#[derive(Clone)]
pub struct ChatMessageService {
    repository: Arc<dyn ChatMessageRepository>,
    cancellation: CancellationToken,
}

impl ChatMessageService {
    pub fn new(
        repository: Arc<dyn ChatMessageRepository>,
        cancellation: CancellationToken,
    ) -> Self {
        Self {
            repository,
            cancellation,
        }
    }

    pub async fn save_message(
        &self,
        article_id: Uuid,
        role: impl Into<String>,
        content: impl Into<String>,
        metadata: Option<MessageMetadata>,
    ) -> Result<ChatMessage, AppError> {
        validate_metadata(metadata.as_ref())?;
        let meta_data = match metadata {
            Some(value) => serde_json::to_value(value).map_err(|_| AppError::Internal)?,
            None => json!({}),
        };
        let mut message = ChatMessage {
            id: Uuid::nil(),
            article_id,
            role: role.into(),
            content: content.into(),
            meta_data: Some(meta_data),
            created_at: None,
        };
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.create(&mut message) => {
                result.map_err(|_| AppError::Internal)?;
                Ok(message)
            }
        }
    }

    pub async fn conversation_history(
        &self,
        article_id: Uuid,
        limit: i64,
    ) -> Result<Vec<ChatMessage>, AppError> {
        let limit = match limit {
            ..=0 => DEFAULT_HISTORY_LIMIT,
            value if value > MAX_HISTORY_LIMIT => MAX_HISTORY_LIMIT,
            value => value,
        };
        let mut messages = tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.list_by_article(article_id, limit) => {
                result.map_err(|_| AppError::Internal)?
            }
        };
        messages.reverse();
        Ok(messages)
    }

    pub async fn clear_conversation_history(&self, article_id: Uuid) -> Result<(), AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.delete_by_article(article_id) => {
                result.map(|_| ()).map_err(|_| AppError::Internal)
            }
        }
    }

    pub async fn update_message_metadata(
        &self,
        message_id: Uuid,
        metadata: Option<MessageMetadata>,
    ) -> Result<(), AppError> {
        validate_metadata(metadata.as_ref())?;
        let value = serde_json::to_value(metadata).map_err(|_| AppError::Internal)?;
        let rows = tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.repository.update_metadata(message_id, value) => {
                result.map_err(|_| AppError::Internal)?
            }
        };
        if rows == 0 {
            return Err(AppError::NotFound);
        }
        Ok(())
    }

    pub async fn update_artifact_status(
        &self,
        message_id: Uuid,
        artifact_id: &str,
        status: &str,
    ) -> Result<(), AppError> {
        let message = self.get_message_by_id(message_id).await?;
        let mut metadata = parse_metadata(message.meta_data)?;
        let artifact = metadata.artifact.as_mut().ok_or(AppError::NotFound)?;
        if !artifact_id.is_empty() && artifact.id != artifact_id {
            return Err(AppError::NotFound);
        }
        artifact.status = status.to_owned();
        if matches!(status, ARTIFACT_STATUS_ACCEPTED | ARTIFACT_STATUS_APPLIED) {
            artifact.applied_at = Some(chrono::Utc::now());
        }
        self.update_message_metadata(message_id, Some(metadata))
            .await
    }

    pub async fn get_message_by_id(&self, message_id: Uuid) -> Result<ChatMessage, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.find_by_id(message_id) => {
                match result {
                    Ok(message) => Ok(message),
                    Err(AppError::NotFound) => Err(AppError::NotFound),
                    Err(_) => Err(AppError::Internal),
                }
            }
        }
    }

    pub async fn pending_artifacts(&self, article_id: Uuid) -> Result<Vec<ChatMessage>, AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.repository.list_pending_artifacts(article_id) => {
                result.map_err(|_| AppError::Internal)
            }
        }
    }

    pub async fn accept_artifact(
        &self,
        message_id: Uuid,
        feedback: impl Into<String>,
    ) -> Result<(), AppError> {
        self.update_artifact_status(message_id, "", ARTIFACT_STATUS_ACCEPTED)
            .await?;
        let message = self.get_message_by_id(message_id).await?;
        let mut metadata = parse_metadata(message.meta_data)?;
        metadata.user_action = Some(UserAction {
            action: USER_ACTION_ACCEPT.to_owned(),
            timestamp: chrono::Utc::now(),
            artifact_id: metadata
                .artifact
                .as_ref()
                .map(|artifact| artifact.id.clone())
                .unwrap_or_default(),
            feedback: feedback.into(),
            reason: String::new(),
        });
        self.update_message_metadata(message_id, Some(metadata))
            .await
    }

    pub async fn reject_artifact(
        &self,
        message_id: Uuid,
        reason: impl Into<String>,
    ) -> Result<(), AppError> {
        self.update_artifact_status(message_id, "", ARTIFACT_STATUS_REJECTED)
            .await?;
        let message = self.get_message_by_id(message_id).await?;
        let mut metadata = parse_metadata(message.meta_data)?;
        metadata.user_action = Some(UserAction {
            action: USER_ACTION_REJECT.to_owned(),
            timestamp: chrono::Utc::now(),
            artifact_id: metadata
                .artifact
                .as_ref()
                .map(|artifact| artifact.id.clone())
                .unwrap_or_default(),
            feedback: String::new(),
            reason: reason.into(),
        });
        self.update_message_metadata(message_id, Some(metadata))
            .await
    }

    pub async fn artifact_content(&self, message_id: Uuid) -> Result<String, AppError> {
        let message = self.get_message_by_id(message_id).await?;
        parse_metadata(message.meta_data)?
            .artifact
            .map(|artifact| artifact.content)
            .ok_or(AppError::NotFound)
    }

    pub async fn mark_artifact_as_applied(&self, message_id: Uuid) -> Result<(), AppError> {
        let message = self.get_message_by_id(message_id).await?;
        let mut metadata = parse_metadata(message.meta_data)?;
        let artifact = metadata.artifact.as_mut().ok_or(AppError::NotFound)?;
        artifact.status = ARTIFACT_STATUS_APPLIED.to_owned();
        artifact.applied_at = Some(chrono::Utc::now());
        self.update_message_metadata(message_id, Some(metadata))
            .await
    }
}

fn parse_metadata(value: Option<Value>) -> Result<MessageMetadata, AppError> {
    match value {
        None | Some(Value::Null) => Ok(MessageMetadata::default()),
        Some(Value::Object(ref object)) if object.is_empty() => Ok(MessageMetadata::default()),
        Some(value) => serde_json::from_value(value).map_err(|_| AppError::Internal),
    }
}

fn validate_metadata(metadata: Option<&MessageMetadata>) -> Result<(), AppError> {
    let Some(metadata) = metadata else {
        return Ok(());
    };
    if let Some(artifact) = metadata.artifact.as_ref() {
        validate_artifact(artifact)?;
    }
    if let Some(action) = metadata.user_action.as_ref() {
        if action.action.is_empty() {
            return Err(AppError::InvalidInput(
                "Invalid metadata: invalid user action: action is required".to_owned(),
            ));
        }
        if !matches!(
            action.action.as_str(),
            USER_ACTION_ACCEPT | USER_ACTION_REJECT | USER_ACTION_MODIFY
        ) {
            return Err(AppError::InvalidInput(format!(
                "Invalid metadata: invalid user action: invalid action: {}",
                action.action
            )));
        }
    }
    Ok(())
}

fn validate_artifact(artifact: &ArtifactInfo) -> Result<(), AppError> {
    if artifact.id.is_empty() {
        return Err(AppError::InvalidInput(
            "Invalid metadata: invalid artifact: artifact ID is required".to_owned(),
        ));
    }
    if artifact.artifact_type.is_empty() {
        return Err(AppError::InvalidInput(
            "Invalid metadata: invalid artifact: artifact type is required".to_owned(),
        ));
    }
    if !matches!(
        artifact.artifact_type.as_str(),
        ARTIFACT_TYPE_CODE_EDIT
            | ARTIFACT_TYPE_REWRITE
            | ARTIFACT_TYPE_SUGGESTION
            | ARTIFACT_TYPE_CONTENT_GENERATION
            | ARTIFACT_TYPE_IMAGE_PROMPT
    ) {
        return Err(AppError::InvalidInput(format!(
            "Invalid metadata: invalid artifact: invalid artifact type: {}",
            artifact.artifact_type
        )));
    }
    if !artifact.status.is_empty()
        && !matches!(
            artifact.status.as_str(),
            super::types::ARTIFACT_STATUS_PENDING
                | ARTIFACT_STATUS_ACCEPTED
                | ARTIFACT_STATUS_REJECTED
                | ARTIFACT_STATUS_APPLIED
        )
    {
        return Err(AppError::InvalidInput(format!(
            "Invalid metadata: invalid artifact: invalid artifact status: {}",
            artifact.status
        )));
    }
    Ok(())
}
