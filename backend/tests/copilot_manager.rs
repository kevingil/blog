use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use blog_backend::{
    core::{
        chat::{ChatMessage, MessageMetadata},
        copilot::{
            ArticleDraftService, ChatPersistencePort, ChatRequest, CopilotConfig, CopilotManager,
        },
        ml::llm::{
            Agent, FinishReason, InMemorySessionStore, LlmMessage, Model, Provider, ProviderError,
            ProviderEvent, ProviderResponse, SessionStore, TokenUsage, Tool,
        },
    },
    error::AppError,
};
use chrono::Utc;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

struct FinalProvider {
    model: Model,
}

#[async_trait]
impl Provider for FinalProvider {
    fn model(&self) -> Model {
        self.model.clone()
    }

    fn system_message(&self) -> &str {
        "fixture"
    }

    async fn stream_response(
        &self,
        _cancellation: CancellationToken,
        _messages: Vec<LlmMessage>,
        _tools: Vec<Arc<dyn Tool>>,
    ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
        let (sender, receiver) = mpsc::channel(2);
        sender
            .send(ProviderEvent::content_delta("Finished article"))
            .await
            .map_err(|_| ProviderError::Request("fixture stream dropped".to_owned()))?;
        sender
            .send(ProviderEvent::complete(ProviderResponse {
                content: "Finished article".to_owned(),
                reasoning: String::new(),
                tool_calls: Vec::new(),
                usage: TokenUsage::default(),
                finish_reason: FinishReason::EndTurn,
            }))
            .await
            .map_err(|_| ProviderError::Request("fixture stream dropped".to_owned()))?;
        Ok(receiver)
    }
}

#[derive(Default)]
struct MemoryChat {
    messages: Mutex<Vec<ChatMessage>>,
}

#[async_trait]
impl ChatPersistencePort for MemoryChat {
    async fn save(
        &self,
        article_id: Uuid,
        role: &str,
        content: &str,
        metadata: Option<MessageMetadata>,
    ) -> Result<ChatMessage, AppError> {
        let message = ChatMessage {
            id: Uuid::new_v4(),
            article_id,
            role: role.to_owned(),
            content: content.to_owned(),
            meta_data: metadata
                .map(serde_json::to_value)
                .transpose()
                .map_err(|_| AppError::Internal)?,
            created_at: Some(Utc::now()),
        };
        self.messages
            .lock()
            .map_err(|_| AppError::Internal)?
            .push(message.clone());
        Ok(message)
    }

    async fn history(&self, article_id: Uuid, limit: i64) -> Result<Vec<ChatMessage>, AppError> {
        Ok(self
            .messages
            .lock()
            .map_err(|_| AppError::Internal)?
            .iter()
            .filter(|message| message.article_id == article_id)
            .rev()
            .take(limit as usize)
            .cloned()
            .collect())
    }
}

struct SnapshotService {
    snapshot: Uuid,
}

#[async_trait]
impl ArticleDraftService for SnapshotService {
    async fn create_draft_snapshot(&self, _article_id: Uuid) -> Result<Option<Uuid>, AppError> {
        Ok(Some(self.snapshot))
    }

    async fn update_draft_content(
        &self,
        _article_id: Uuid,
        _content: &str,
    ) -> Result<(), AppError> {
        Ok(())
    }
}

#[tokio::test]
async fn manager_persists_messages_streams_snapshot_and_shuts_down_owned_tasks() {
    let store = Arc::new(InMemorySessionStore::default());
    let agent = Agent::new(
        Arc::new(FinalProvider {
            model: Model::openai("fixture", "fixture", 1_024, false),
        }),
        store.clone(),
        Vec::new(),
    );
    let chat = Arc::new(MemoryChat::default());
    let snapshot = Uuid::new_v4();
    let manager = CopilotManager::new(
        agent,
        store as Arc<dyn SessionStore>,
        chat.clone(),
        None,
        Some(Arc::new(SnapshotService { snapshot })),
        CopilotConfig::new(2, 1, 16, 15).unwrap_or_default(),
        CancellationToken::new(),
    );
    let article_id = Uuid::new_v4();
    let request_id = manager
        .submit(ChatRequest {
            message: "finish it".to_owned(),
            document_content: String::new(),
            document_markdown: "draft".to_owned(),
            article_id: article_id.to_string(),
        })
        .await
        .unwrap_or_default();
    assert!(!request_id.is_empty());
    let stream = manager.take_response_stream(&request_id);
    assert!(stream.is_ok());
    let Ok(mut stream) = stream else {
        return;
    };
    let mut event_types = Vec::new();
    let mut observed_snapshot = None;
    while let Some(event) = stream.recv().await {
        if event.event_type == "turn_started" {
            observed_snapshot = event
                .data
                .and_then(|data| data["snapshot_version_id"].as_str().map(str::to_owned));
        }
        event_types.push(event.event_type);
    }
    assert_eq!(observed_snapshot, Some(snapshot.to_string()));
    assert!(event_types.contains(&"content_delta".to_owned()));
    assert!(event_types.contains(&"text".to_owned()));
    assert!(event_types.contains(&"done".to_owned()));

    let persisted = chat
        .messages
        .lock()
        .map(|messages| messages.clone())
        .unwrap_or_default();
    assert!(persisted.iter().any(|message| message.role == "user"));
    assert!(
        persisted.iter().any(|message| {
            message.role == "assistant" && message.content == "Finished article"
        })
    );
    assert!(
        manager
            .shutdown(std::time::Duration::from_secs(1))
            .await
            .is_ok()
    );
    assert_eq!(manager.active_requests(), 0);
}
