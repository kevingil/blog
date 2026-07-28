use std::sync::{Arc, Mutex, MutexGuard};

use async_trait::async_trait;
use blog_backend::{
    core::chat::{
        ArtifactInfo, ChatMessage, ChatMessageRepository, ChatMessageService, MessageContext,
        MessageMetadata, ThinkingBlock, ToolCallRecord, ToolCallStatus, ToolExecution, ToolGroup,
        ToolGroupStatus, ToolStepInfo, UserAction,
    },
    error::AppError,
};
use chrono::{TimeZone, Utc};
use serde_json::{Map, Value, json};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

#[derive(Default)]
struct ChatState {
    messages: Vec<ChatMessage>,
    requested_limits: Vec<i64>,
    metadata_updates: usize,
    fail_metadata_update: Option<usize>,
}

#[derive(Default)]
struct MemoryChatRepository {
    state: Mutex<ChatState>,
}

impl MemoryChatRepository {
    fn state(&self) -> MutexGuard<'_, ChatState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl ChatMessageRepository for MemoryChatRepository {
    async fn create(&self, message: &mut ChatMessage) -> Result<(), AppError> {
        if message.id.is_nil() {
            message.id = Uuid::new_v4();
        }
        message.created_at = Some(Utc::now());
        self.state().messages.push(message.clone());
        Ok(())
    }

    async fn find_by_id(&self, id: Uuid) -> Result<ChatMessage, AppError> {
        self.state()
            .messages
            .iter()
            .find(|message| message.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list_by_article(
        &self,
        article_id: Uuid,
        limit: i64,
    ) -> Result<Vec<ChatMessage>, AppError> {
        let mut state = self.state();
        state.requested_limits.push(limit);
        let mut messages = state
            .messages
            .iter()
            .filter(|message| message.article_id == article_id)
            .cloned()
            .collect::<Vec<_>>();
        messages.sort_by(|left, right| right.created_at.cmp(&left.created_at));
        messages.truncate(usize::try_from(limit).map_err(|_| AppError::Internal)?);
        Ok(messages)
    }

    async fn list_pending_artifacts(&self, article_id: Uuid) -> Result<Vec<ChatMessage>, AppError> {
        Ok(self
            .state()
            .messages
            .iter()
            .filter(|message| {
                message.article_id == article_id
                    && message
                        .meta_data
                        .as_ref()
                        .and_then(|value| value.pointer("/artifact/status"))
                        == Some(&Value::String("pending".to_owned()))
            })
            .cloned()
            .collect())
    }

    async fn update(&self, message: &ChatMessage) -> Result<(), AppError> {
        let mut state = self.state();
        let stored = state
            .messages
            .iter_mut()
            .find(|stored| stored.id == message.id)
            .ok_or(AppError::NotFound)?;
        *stored = message.clone();
        Ok(())
    }

    async fn update_metadata(&self, id: Uuid, metadata: Value) -> Result<u64, AppError> {
        let mut state = self.state();
        state.metadata_updates += 1;
        if state.fail_metadata_update == Some(state.metadata_updates) {
            return Err(AppError::Database);
        }
        let Some(message) = state.messages.iter_mut().find(|message| message.id == id) else {
            return Ok(0);
        };
        message.meta_data = Some(metadata);
        Ok(1)
    }

    async fn delete_by_article(&self, article_id: Uuid) -> Result<u64, AppError> {
        let mut state = self.state();
        let before = state.messages.len();
        state
            .messages
            .retain(|message| message.article_id != article_id);
        u64::try_from(before - state.messages.len()).map_err(|_| AppError::Internal)
    }
}

fn artifact(status: &str) -> ArtifactInfo {
    ArtifactInfo {
        id: "artifact-1".to_owned(),
        artifact_type: "rewrite".to_owned(),
        status: status.to_owned(),
        content: "replacement".to_owned(),
        diff_preview: "-old\n+new".to_owned(),
        title: "Rewrite".to_owned(),
        description: "Replace the draft".to_owned(),
        applied_at: None,
    }
}

#[tokio::test]
async fn save_and_history_preserve_metadata_defaults_limits_and_order() {
    let repository = Arc::new(MemoryChatRepository::default());
    let service = ChatMessageService::new(repository.clone(), CancellationToken::new());
    let article_id = Uuid::new_v4();

    let first = service
        .save_message(article_id, "user", "first", None)
        .await;
    assert!(first.is_ok());
    {
        let mut state = repository.state();
        if let Some(message) = state.messages.first_mut() {
            message.created_at = Utc.timestamp_opt(10, 0).single();
        }
    }
    let second = service
        .save_message(article_id, "assistant", "second", None)
        .await;
    assert!(second.is_ok());
    {
        let mut state = repository.state();
        if let Some(message) = state.messages.get_mut(1) {
            message.created_at = Utc.timestamp_opt(20, 0).single();
        }
    }

    let history = service.conversation_history(article_id, 0).await;
    assert!(history.is_ok());
    let history = history.unwrap_or_default();
    assert_eq!(
        history
            .iter()
            .map(|message| message.content.as_str())
            .collect::<Vec<_>>(),
        vec!["first", "second"]
    );
    assert!(
        history
            .iter()
            .all(|message| message.meta_data == Some(json!({})))
    );

    let capped = service.conversation_history(article_id, 500).await;
    assert!(capped.is_ok());
    assert_eq!(repository.state().requested_limits, vec![50, 200]);
}

#[tokio::test]
async fn metadata_json_keeps_go_field_names_and_omission_rules() {
    let timestamp = Utc.timestamp_opt(42, 123_000_000).single();
    assert!(timestamp.is_some());
    let timestamp = timestamp.unwrap_or_else(Utc::now);
    let metadata = MessageMetadata {
        artifact: Some(artifact("pending")),
        tool_execution: Some(ToolExecution {
            tool_name: "edit".to_owned(),
            tool_id: "call-1".to_owned(),
            input: json!({"title": "x"}),
            output: Value::Null,
            error: String::new(),
            duration_ms: 12,
            executed_at: timestamp,
            success: true,
        }),
        context: Some(MessageContext {
            article_id: String::new(),
            session_id: "session-1".to_owned(),
            request_id: String::new(),
            document_version: String::new(),
            document_hash: String::new(),
            user_id: String::new(),
        }),
        user_action: Some(UserAction {
            action: "accept".to_owned(),
            timestamp,
            artifact_id: String::new(),
            feedback: String::new(),
            reason: String::new(),
        }),
        thinking: Some(ThinkingBlock {
            content: "thought".to_owned(),
            duration_ms: 0,
            visible: false,
        }),
        steps: vec![],
    };
    let encoded = serde_json::to_value(metadata);
    assert!(encoded.is_ok());
    let encoded = encoded.unwrap_or(Value::Null);
    assert_eq!(encoded.pointer("/artifact/type"), Some(&json!("rewrite")));
    assert_eq!(
        encoded.pointer("/tool_execution/duration_ms"),
        Some(&json!(12))
    );
    assert!(encoded.pointer("/tool_execution/error").is_none());
    assert!(encoded.pointer("/context/article_id").is_none());
    assert!(encoded.pointer("/user_action/reason").is_none());
    assert!(encoded.pointer("/thinking/duration_ms").is_none());
    assert!(encoded.pointer("/steps").is_none());

    let tool_group = ToolGroup {
        group_id: "group-1".to_owned(),
        status: ToolGroupStatus::Completed,
        calls: Some(vec![ToolCallRecord {
            id: "call-1".to_owned(),
            name: "edit".to_owned(),
            input: Some(Map::new()),
            status: ToolCallStatus::Completed,
            result: Map::new(),
            error: String::new(),
            started_at: "start".to_owned(),
            completed_at: String::new(),
            duration_ms: 0,
        }]),
    };
    let tool_step = ToolStepInfo {
        tool_id: "call-1".to_owned(),
        tool_name: "edit".to_owned(),
        input: Map::new(),
        output: Map::new(),
        status: "completed".to_owned(),
        error: String::new(),
        started_at: String::new(),
        completed_at: String::new(),
        duration_ms: 0,
    };
    let group_json = serde_json::to_value(tool_group);
    let step_json = serde_json::to_value(tool_step);
    assert!(group_json.is_ok());
    assert!(step_json.is_ok());
    assert_eq!(
        group_json.unwrap_or(Value::Null).pointer("/status"),
        Some(&json!("completed"))
    );
    assert_eq!(
        step_json.unwrap_or(Value::Null),
        json!({"tool_id":"call-1","tool_name":"edit","status":"completed"})
    );

    let nullable_group = ToolGroup {
        group_id: "group-2".to_owned(),
        status: ToolGroupStatus::Other("provider-specific".to_owned()),
        calls: None,
    };
    assert_eq!(
        serde_json::to_value(nullable_group).unwrap_or(Value::Null),
        json!({
            "group_id": "group-2",
            "status": "provider-specific",
            "calls": null
        })
    );
}

#[tokio::test]
async fn legacy_partial_artifact_metadata_zero_fills_like_go_json_unmarshal() {
    let repository = Arc::new(MemoryChatRepository::default());
    let service = ChatMessageService::new(repository.clone(), CancellationToken::new());
    let article_id = Uuid::new_v4();
    let message_id = Uuid::new_v4();
    repository.state().messages.push(ChatMessage {
        id: message_id,
        article_id,
        role: "assistant".to_owned(),
        content: String::new(),
        meta_data: Some(json!({
            "artifact": {
                "id": "legacy-artifact",
                "type": "rewrite",
                "status": "pending",
                "content": "legacy replacement"
            },
            "context": {}
        })),
        created_at: Some(Utc::now()),
    });

    let content = service.artifact_content(message_id).await;
    assert_eq!(content.unwrap_or_default(), "legacy replacement");
    let pending = service.pending_artifacts(article_id).await;
    assert!(pending.is_ok());
    let pending = pending.unwrap_or_default();
    assert_eq!(pending.len(), 1);
    let metadata = pending[0]
        .meta_data
        .clone()
        .and_then(|value| serde_json::from_value::<MessageMetadata>(value).ok())
        .unwrap_or_default();
    let artifact = metadata.artifact.unwrap_or_default();
    assert_eq!(artifact.diff_preview, "");
    assert_eq!(artifact.title, "");
    assert_eq!(artifact.description, "");
}

#[tokio::test]
async fn artifact_accept_keeps_first_write_when_user_action_write_fails() {
    let repository = Arc::new(MemoryChatRepository::default());
    let service = ChatMessageService::new(repository.clone(), CancellationToken::new());
    let message = service
        .save_message(
            Uuid::new_v4(),
            "assistant",
            "proposal",
            Some(MessageMetadata {
                artifact: Some(artifact("pending")),
                ..MessageMetadata::default()
            }),
        )
        .await;
    assert!(message.is_ok());
    let message_id = message
        .map(|message| message.id)
        .unwrap_or_else(|_| Uuid::nil());
    repository.state().fail_metadata_update = Some(2);

    let accepted = service.accept_artifact(message_id, "looks good").await;
    assert!(matches!(accepted, Err(AppError::Internal)));
    let stored = repository.state().messages.first().cloned();
    assert!(stored.is_some());
    let metadata = stored
        .and_then(|message| message.meta_data)
        .and_then(|value| serde_json::from_value::<MessageMetadata>(value).ok());
    assert!(metadata.is_some());
    let metadata = metadata.unwrap_or_default();
    assert_eq!(
        metadata
            .artifact
            .as_ref()
            .map(|artifact| artifact.status.as_str()),
        Some("accepted")
    );
    assert!(
        metadata
            .artifact
            .as_ref()
            .and_then(|artifact| artifact.applied_at)
            .is_some()
    );
    assert!(metadata.user_action.is_none());
}

#[tokio::test]
async fn invalid_artifact_is_rejected_before_repository_write_and_cancellation_stops_admission() {
    let repository = Arc::new(MemoryChatRepository::default());
    let token = CancellationToken::new();
    let service = ChatMessageService::new(repository.clone(), token.clone());
    let invalid = service
        .save_message(
            Uuid::new_v4(),
            "assistant",
            "proposal",
            Some(MessageMetadata {
                artifact: Some(ArtifactInfo {
                    artifact_type: "unknown".to_owned(),
                    ..artifact("pending")
                }),
                ..MessageMetadata::default()
            }),
        )
        .await;
    assert!(matches!(invalid, Err(AppError::InvalidInput(_))));
    assert!(repository.state().messages.is_empty());

    token.cancel();
    let cancelled = service
        .save_message(Uuid::new_v4(), "user", "message", None)
        .await;
    assert!(matches!(cancelled, Err(AppError::Internal)));
    assert!(repository.state().messages.is_empty());
}
