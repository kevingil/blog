use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};

use async_trait::async_trait;
use blog_backend::core::ml::llm::{
    Agent, AgentError, AgentEventType, ContentPart, FinishReason, InMemorySessionStore, LlmMessage,
    MessageRole, Model, Provider, ProviderError, ProviderEvent, ProviderResponse, ReplaceLinesTool,
    SessionStore, TokenUsage, Tool, ToolCall, ToolContext,
};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

struct ScriptedProvider {
    model: Model,
    scripts: Mutex<VecDeque<Vec<ProviderEvent>>>,
}

impl ScriptedProvider {
    fn new(scripts: Vec<Vec<ProviderEvent>>) -> Self {
        Self {
            model: Model::openai("fixture", "fixture", 1_024, true),
            scripts: Mutex::new(scripts.into()),
        }
    }
}

#[async_trait]
impl Provider for ScriptedProvider {
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
                .map_err(|_| ProviderError::Request("fixture receiver dropped".to_owned()))?;
        }
        Ok(receiver)
    }
}

struct BlockingProvider {
    model: Model,
}

#[async_trait]
impl Provider for BlockingProvider {
    fn model(&self) -> Model {
        self.model.clone()
    }

    fn system_message(&self) -> &str {
        "fixture"
    }

    async fn stream_response(
        &self,
        cancellation: CancellationToken,
        _messages: Vec<LlmMessage>,
        _tools: Vec<Arc<dyn Tool>>,
    ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
        cancellation.cancelled().await;
        Err(ProviderError::Cancelled)
    }
}

fn complete(
    content: &str,
    tool_calls: Vec<ToolCall>,
    finish_reason: FinishReason,
) -> ProviderEvent {
    ProviderEvent::complete(ProviderResponse {
        content: content.to_owned(),
        reasoning: String::new(),
        tool_calls,
        usage: TokenUsage {
            input_tokens: 10,
            output_tokens: 5,
            cache_creation_tokens: 0,
            cache_read_tokens: 2,
        },
        finish_reason,
    })
}

#[tokio::test]
async fn agent_streams_deltas_executes_tools_and_preserves_result_order() {
    let provider = Arc::new(ScriptedProvider::new(vec![
        vec![complete(
            "",
            vec![ToolCall {
                id: "call-1".to_owned(),
                name: "replace_lines".to_owned(),
                input: r#"{"start_line":1,"end_line":1,"new_content":"draft","reason":"write"}"#
                    .to_owned(),
                r#type: "function".to_owned(),
                finished: true,
                thought_signature: Vec::new(),
            }],
            FinishReason::ToolUse,
        )],
        vec![
            ProviderEvent::thinking_delta("considering"),
            ProviderEvent::content_delta("Done"),
            complete("Done", Vec::new(), FinishReason::EndTurn),
        ],
    ]));
    let store = Arc::new(InMemorySessionStore::default());
    let session = store.create_session("test").await;
    assert!(session.is_ok());
    let Ok(session) = session else {
        return;
    };
    let agent = Agent::new(
        provider,
        store.clone(),
        vec![Arc::new(ReplaceLinesTool::new(None))],
    );
    let cancellation = CancellationToken::new();
    let context = ToolContext::new(
        &session.id,
        "",
        "request",
        Some(Uuid::new_v4()),
        "",
        "",
        cancellation.clone(),
    );
    let run = agent.start(
        cancellation,
        session.id.clone(),
        "write it".to_owned(),
        Vec::new(),
        context.clone(),
    );
    assert!(run.is_ok());
    let Ok(run) = run else {
        return;
    };
    let mut events = run.events;
    let mut types = Vec::new();
    while let Some(event) = events.recv().await {
        types.push(event.event_type);
    }
    let result = run.handle.await;
    assert!(matches!(result, Ok(Ok(()))));
    assert!(types.contains(&AgentEventType::Thinking));
    assert!(types.contains(&AgentEventType::Tool));
    assert!(types.contains(&AgentEventType::ReasoningDelta));
    assert!(types.contains(&AgentEventType::ContentDelta));
    assert_eq!(context.document_markdown().unwrap_or_default(), "draft");

    let history = store.list_messages(&session.id).await.unwrap_or_default();
    assert_eq!(
        history
            .iter()
            .filter(|message| message.role == MessageRole::Tool)
            .count(),
        1
    );
    assert!(history.iter().any(|message| {
        message.role == MessageRole::Assistant
            && message
                .parts
                .iter()
                .any(|part| matches!(part, ContentPart::Text(text) if text.text == "Done"))
    }));
}

#[tokio::test]
async fn cancellation_is_owned_and_session_busy_is_atomic() {
    let provider = Arc::new(BlockingProvider {
        model: Model::openai("fixture", "fixture", 1_024, false),
    });
    let store = Arc::new(InMemorySessionStore::default());
    let session = store.create_session("test").await;
    assert!(session.is_ok());
    let Ok(session) = session else {
        return;
    };
    let agent = Agent::new(provider, store, Vec::new());
    let root = CancellationToken::new();
    let context = ToolContext::new(&session.id, "", "request", None, "", "", root.clone());
    let run = agent.start(
        root.clone(),
        session.id.clone(),
        "wait".to_owned(),
        Vec::new(),
        context.clone(),
    );
    assert!(run.is_ok());
    let Ok(run) = run else {
        return;
    };
    assert!(matches!(
        agent.start(
            root,
            session.id.clone(),
            "second".to_owned(),
            Vec::new(),
            context,
        ),
        Err(AgentError::SessionBusy)
    ));
    assert!(agent.cancel(&session.id));
    let result = run.handle.await;
    assert!(matches!(result, Ok(Err(AgentError::RequestCancelled))));
    assert!(!agent.is_session_busy(&session.id));
}
