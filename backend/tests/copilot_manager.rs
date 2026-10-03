use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        chat::{ChatMessage, MessageMetadata},
        copilot::{
            ArticleDraftService, ChatPersistencePort, ChatRequest, CopilotConfig, CopilotManager,
        },
        ml::llm::{
            Agent, ApplyPatchTool, FinishReason, InMemorySessionStore, LlmMessage, MessageRole,
            Model, Provider, ProviderError, ProviderEvent, ProviderResponse, SessionStore,
            TokenUsage, Tool, ToolCall,
        },
        speech::{SpeechAudio, SpeechPort, silent_wav},
    },
    error::AppError,
};
use chrono::Utc;
use serde_json::Value;
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
                hosted: Vec::new(),
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
        let messages = self
            .messages
            .lock()
            .map_err(|_| AppError::Internal)?
            .iter()
            .filter(|message| message.article_id == article_id)
            .cloned()
            .collect::<Vec<_>>();
        let limit = usize::try_from(limit).unwrap_or(0);
        let start = messages.len().saturating_sub(limit);
        Ok(messages[start..].to_vec())
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
        None,
        None,
    );
    let article_id = Uuid::new_v4();
    let request_id = manager
        .submit(ChatRequest {
            message: "finish it".to_owned(),
            document_content: String::new(),
            document_markdown: "draft".to_owned(),
            document_title: String::new(),
            article_id: article_id.to_string(),
            channel: "text".to_owned(),
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

struct FixtureSpeech;

#[async_trait]
impl SpeechPort for FixtureSpeech {
    async fn transcribe(&self, _audio: &[u8], _mime_type: &str) -> Result<String, AppError> {
        Ok("tighten the intro".to_owned())
    }

    async fn synthesize(&self, text: &str) -> Result<SpeechAudio, AppError> {
        Ok(SpeechAudio {
            bytes: silent_wav(),
            mime_type: format!("audio/wav;text={text}"),
        })
    }
}

#[tokio::test]
async fn voice_turn_emits_transcript_and_speech_into_the_same_session() {
    let store = Arc::new(InMemorySessionStore::default());
    let agent = Agent::new(
        Arc::new(FinalProvider {
            model: Model::openai("fixture", "fixture", 1_024, false),
        }),
        store.clone(),
        Vec::new(),
    );
    let chat = Arc::new(MemoryChat::default());
    let manager = CopilotManager::new(
        agent,
        store as Arc<dyn SessionStore>,
        chat.clone(),
        None,
        None,
        CopilotConfig::new(2, 1, 16, 15).unwrap_or_default(),
        CancellationToken::new(),
        Some(Arc::new(FixtureSpeech)),
        None,
    );
    let article_id = Uuid::new_v4();
    let request_id = manager
        .submit(ChatRequest {
            message: "tighten the intro".to_owned(),
            document_content: String::new(),
            document_markdown: "draft".to_owned(),
            document_title: String::new(),
            article_id: article_id.to_string(),
            channel: "voice".to_owned(),
        })
        .await
        .unwrap_or_default();
    let mut stream = manager
        .take_response_stream(&request_id)
        .expect("voice stream");
    let mut event_types = Vec::new();
    let mut spoke = false;
    while let Some(event) = stream.recv().await {
        if event.event_type == "speech" {
            spoke = event
                .data
                .as_ref()
                .and_then(|data| data.get("audioBase64"))
                .and_then(Value::as_str)
                .is_some_and(|value| !value.is_empty());
        }
        event_types.push(event.event_type);
    }
    assert!(event_types.contains(&"transcript".to_owned()));
    assert!(event_types.contains(&"speech".to_owned()));
    assert!(spoke);
    let persisted = chat
        .messages
        .lock()
        .map(|messages| messages.clone())
        .unwrap_or_default();
    assert!(persisted.iter().any(|message| {
        message.role == "user"
            && message.meta_data.as_ref().is_some_and(|value| {
                value.get("input_channel").and_then(|item| item.as_str()) == Some("voice")
            })
    }));
}

#[tokio::test]
async fn live_turn_keeps_the_transcript_and_skips_speech_synthesis() {
    let store = Arc::new(InMemorySessionStore::default());
    let agent = Agent::new(
        Arc::new(FinalProvider {
            model: Model::openai("fixture", "fixture", 1_024, false),
        }),
        store.clone(),
        Vec::new(),
    );
    let chat = Arc::new(MemoryChat::default());
    let manager = CopilotManager::new(
        agent,
        store as Arc<dyn SessionStore>,
        chat.clone(),
        None,
        None,
        CopilotConfig::new(2, 1, 16, 15).unwrap_or_default(),
        CancellationToken::new(),
        Some(Arc::new(FixtureSpeech)),
        None,
    );
    let request_id = manager
        .submit(ChatRequest {
            message: "tighten the intro".to_owned(),
            document_content: String::new(),
            document_markdown: "draft".to_owned(),
            document_title: String::new(),
            article_id: Uuid::new_v4().to_string(),
            channel: "live".to_owned(),
        })
        .await
        .unwrap_or_default();
    let mut stream = manager
        .take_response_stream(&request_id)
        .expect("live stream");
    let mut event_types = Vec::new();
    while let Some(event) = stream.recv().await {
        event_types.push(event.event_type);
    }
    assert!(event_types.contains(&"transcript".to_owned()));
    assert!(!event_types.contains(&"speech".to_owned()));
    let persisted = chat
        .messages
        .lock()
        .map(|messages| messages.clone())
        .unwrap_or_default();
    assert!(persisted.iter().any(|message| {
        message.role == "user"
            && message.meta_data.as_ref().is_some_and(|value| {
                value.get("input_channel").and_then(|item| item.as_str()) == Some("voice")
            })
    }));
}

struct RecordingProvider {
    model: Model,
    scripts: Mutex<VecDeque<Vec<ProviderEvent>>>,
    seen: Mutex<Vec<Vec<LlmMessage>>>,
}

impl RecordingProvider {
    fn new(scripts: Vec<Vec<ProviderEvent>>) -> Self {
        Self {
            model: Model::openai("fixture", "fixture", 1_024, false),
            scripts: Mutex::new(scripts.into()),
            seen: Mutex::new(Vec::new()),
        }
    }
}

#[async_trait]
impl Provider for RecordingProvider {
    fn model(&self) -> Model {
        self.model.clone()
    }

    fn system_message(&self) -> &str {
        "fixture"
    }

    async fn stream_response(
        &self,
        _cancellation: CancellationToken,
        messages: Vec<LlmMessage>,
        _tools: Vec<Arc<dyn Tool>>,
    ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
        self.seen
            .lock()
            .map_err(|_| ProviderError::Request("seen lock".to_owned()))?
            .push(messages);
        let script = self
            .scripts
            .lock()
            .map_err(|_| ProviderError::Request("script lock".to_owned()))?
            .pop_front()
            .ok_or_else(|| ProviderError::Request("missing script".to_owned()))?;
        let (sender, receiver) = mpsc::channel(script.len().max(1));
        for event in script {
            sender
                .send(event)
                .await
                .map_err(|_| ProviderError::Request("fixture stream dropped".to_owned()))?;
        }
        Ok(receiver)
    }
}

fn scripted_complete(content: &str, calls: Vec<ToolCall>, finish: FinishReason) -> ProviderEvent {
    ProviderEvent::complete(ProviderResponse {
        content: content.to_owned(),
        reasoning: String::new(),
        tool_calls: calls,
        hosted: Vec::new(),
        usage: TokenUsage::default(),
        finish_reason: finish,
    })
}

struct StoredDraft;

#[async_trait]
impl ArticleDraftService for StoredDraft {
    async fn create_draft_snapshot(&self, _article_id: Uuid) -> Result<Option<Uuid>, AppError> {
        Ok(None)
    }

    async fn update_draft_content(
        &self,
        _article_id: Uuid,
        _content: &str,
    ) -> Result<(), AppError> {
        Ok(())
    }

    async fn load_draft_content(&self, _article_id: Uuid) -> Result<Option<String>, AppError> {
        Ok(Some("Stored opening".to_owned()))
    }
}

#[tokio::test]
async fn backend_edit_streams_the_saved_draft_and_replays_tool_execution() {
    let provider = Arc::new(RecordingProvider::new(vec![
        vec![scripted_complete(
            "",
            vec![ToolCall {
                id: "call-patch".to_owned(),
                name: "apply_patch".to_owned(),
                input: r#"{"patch":"","old_str":"","new_str":"Hello draft","reason":"write"}"#
                    .to_owned(),
                r#type: "function".to_owned(),
                finished: true,
                thought_signature: Vec::new(),
            }],
            FinishReason::ToolUse,
        )],
        vec![scripted_complete("Done", Vec::new(), FinishReason::EndTurn)],
        vec![scripted_complete("Next", Vec::new(), FinishReason::EndTurn)],
    ]));
    let store = Arc::new(InMemorySessionStore::default());
    let agent = Agent::new(
        provider.clone(),
        store.clone(),
        vec![Arc::new(ApplyPatchTool::new(None))],
    );
    let chat = Arc::new(MemoryChat::default());
    let manager = CopilotManager::new(
        agent,
        store as Arc<dyn SessionStore>,
        chat.clone(),
        None,
        None,
        CopilotConfig::new(2, 1, 16, 15).unwrap_or_default(),
        CancellationToken::new(),
        None,
        None,
    );
    let article_id = Uuid::new_v4();
    let request_id = manager
        .submit(ChatRequest {
            message: "write it".to_owned(),
            document_content: String::new(),
            document_markdown: String::new(),
            document_title: String::new(),
            article_id: article_id.to_string(),
            channel: "text".to_owned(),
        })
        .await
        .unwrap_or_default();
    let mut stream = manager
        .take_response_stream(&request_id)
        .expect("edit stream");
    let mut updated = String::new();
    while let Some(event) = stream.recv().await {
        if event.event_type == "document_update" {
            updated = event.content;
        }
    }
    assert_eq!(updated, "Hello draft");

    let follow_up = manager
        .submit(ChatRequest {
            message: "continue".to_owned(),
            document_content: String::new(),
            document_markdown: "Hello draft".to_owned(),
            document_title: String::new(),
            article_id: article_id.to_string(),
            channel: "text".to_owned(),
        })
        .await
        .unwrap_or_default();
    let mut follow_stream = manager
        .take_response_stream(&follow_up)
        .expect("follow-up stream");
    while follow_stream.recv().await.is_some() {}

    let seen = provider
        .seen
        .lock()
        .map(|seen| seen.clone())
        .unwrap_or_default();
    let follow_messages = seen.last().cloned().unwrap_or_default();
    assert!(follow_messages.iter().any(|message| {
        message.role == MessageRole::Assistant
            && message
                .tool_calls()
                .iter()
                .any(|call| call.id == "call-patch" && call.name == "apply_patch")
    }));
    assert!(follow_messages.iter().any(|message| {
        message.role == MessageRole::Tool
            && message.tool_results().iter().any(|result| {
                result.tool_call_id == "call-patch" && result.content.contains("Hello draft")
            })
    }));
}

#[tokio::test]
async fn empty_client_document_uses_the_stored_article() {
    let provider = Arc::new(RecordingProvider::new(vec![vec![scripted_complete(
        "Ready",
        Vec::new(),
        FinishReason::EndTurn,
    )]]));
    let store = Arc::new(InMemorySessionStore::default());
    let agent = Agent::new(provider.clone(), store.clone(), Vec::new());
    let manager = CopilotManager::new(
        agent,
        store as Arc<dyn SessionStore>,
        Arc::new(MemoryChat::default()),
        None,
        Some(Arc::new(StoredDraft)),
        CopilotConfig::new(2, 1, 16, 15).unwrap_or_default(),
        CancellationToken::new(),
        None,
        None,
    );
    let request_id = manager
        .submit(ChatRequest {
            message: "look at the draft".to_owned(),
            document_content: String::new(),
            document_markdown: String::new(),
            document_title: String::new(),
            article_id: Uuid::new_v4().to_string(),
            channel: "text".to_owned(),
        })
        .await
        .unwrap_or_default();
    let mut stream = manager
        .take_response_stream(&request_id)
        .expect("stored draft stream");
    while stream.recv().await.is_some() {}
    let seen = provider
        .seen
        .lock()
        .map(|seen| seen.clone())
        .unwrap_or_default();
    let prompt = seen
        .first()
        .and_then(|messages| {
            messages
                .iter()
                .find(|message| message.role == MessageRole::User)
        })
        .map(|message| message.text())
        .unwrap_or_default();
    assert!(
        prompt.contains("Total: 1 lines, 14 chars, 1 paragraphs"),
        "{prompt}"
    );
    assert!(!prompt.contains("empty document"), "{prompt}");
}
