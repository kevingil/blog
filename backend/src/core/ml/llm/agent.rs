use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use futures_util::future::join_all;
use thiserror::Error;
use tokio::{sync::mpsc, task::JoinHandle};
use tokio_util::sync::CancellationToken;

use super::{
    Attachment, ContentPart, FinishReason, LlmMessage, MessageRole, Model, Provider, ProviderError,
    ProviderEventType, SessionStore, TextContent, TokenUsage, Tool, ToolCallRequest, ToolContext,
    ToolResult,
};

const MAX_ITERATIONS: usize = 25;
const EVENT_CHANNEL_CAPACITY: usize = 100;

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum AgentError {
    #[error("request cancelled by user")]
    RequestCancelled,
    #[error("session is currently processing another request")]
    SessionBusy,
    #[error("provider failed: {0}")]
    Provider(String),
    #[error("session store failed: {0}")]
    Store(String),
    #[error("agent event receiver was dropped")]
    ReceiverDropped,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AgentEventType {
    Error,
    Response,
    Tool,
    Thinking,
    ContentDelta,
    ReasoningDelta,
}

#[derive(Debug, Clone)]
pub struct AgentEvent {
    pub event_type: AgentEventType,
    pub message: Option<LlmMessage>,
    pub error: Option<AgentError>,
    pub thinking_message: String,
    pub iteration: usize,
    pub content_delta: String,
    pub reasoning_delta: String,
    pub done: bool,
}

impl AgentEvent {
    fn thinking(iteration: usize) -> Self {
        Self {
            event_type: AgentEventType::Thinking,
            message: None,
            error: None,
            thinking_message: "Thinking...".to_owned(),
            iteration,
            content_delta: String::new(),
            reasoning_delta: String::new(),
            done: false,
        }
    }

    fn error(error: AgentError) -> Self {
        Self {
            event_type: AgentEventType::Error,
            message: None,
            error: Some(error),
            thinking_message: String::new(),
            iteration: 0,
            content_delta: String::new(),
            reasoning_delta: String::new(),
            done: true,
        }
    }
}

pub struct AgentRun {
    pub events: mpsc::Receiver<AgentEvent>,
    pub cancellation: CancellationToken,
    pub handle: JoinHandle<Result<(), AgentError>>,
}

pub struct Agent {
    provider: Arc<dyn Provider>,
    store: Arc<dyn SessionStore>,
    tools: Vec<Arc<dyn Tool>>,
    active: Mutex<HashMap<String, CancellationToken>>,
}

impl Agent {
    pub fn new(
        provider: Arc<dyn Provider>,
        store: Arc<dyn SessionStore>,
        tools: Vec<Arc<dyn Tool>>,
    ) -> Arc<Self> {
        Arc::new(Self {
            provider,
            store,
            tools,
            active: Mutex::new(HashMap::new()),
        })
    }

    pub fn model(&self) -> Model {
        self.provider.model()
    }

    pub fn is_session_busy(&self, session_id: &str) -> bool {
        self.active
            .lock()
            .map(|active| active.contains_key(session_id))
            .unwrap_or(true)
    }

    pub fn is_busy(&self) -> bool {
        self.active
            .lock()
            .map(|active| !active.is_empty())
            .unwrap_or(true)
    }

    pub fn cancel(&self, session_id: &str) -> bool {
        let cancellation = self
            .active
            .lock()
            .ok()
            .and_then(|active| active.get(session_id).cloned());
        if let Some(cancellation) = cancellation {
            cancellation.cancel();
            true
        } else {
            false
        }
    }

    pub fn start(
        self: &Arc<Self>,
        parent_cancellation: CancellationToken,
        session_id: String,
        content: String,
        attachments: Vec<Attachment>,
        context: ToolContext,
    ) -> Result<AgentRun, AgentError> {
        let cancellation = parent_cancellation.child_token();
        {
            let mut active = self
                .active
                .lock()
                .map_err(|_| AgentError::Store("active request lock poisoned".to_owned()))?;
            if active.contains_key(&session_id) {
                return Err(AgentError::SessionBusy);
            }
            active.insert(session_id.clone(), cancellation.clone());
        }

        let (sender, events) = mpsc::channel(EVENT_CHANNEL_CAPACITY);
        let agent = self.clone();
        let task_cancellation = cancellation.clone();
        let task_session = session_id.clone();
        let handle = tokio::spawn(async move {
            let result = agent
                .run_loop(
                    task_cancellation.clone(),
                    task_session.clone(),
                    content,
                    attachments,
                    context,
                    sender.clone(),
                )
                .await;
            if let Err(error) = &result {
                let _ = send_event(
                    &sender,
                    &task_cancellation,
                    AgentEvent::error(error.clone()),
                )
                .await;
            }
            if let Ok(mut active) = agent.active.lock() {
                active.remove(&task_session);
            }
            result
        });
        Ok(AgentRun {
            events,
            cancellation,
            handle,
        })
    }

    async fn run_loop(
        &self,
        cancellation: CancellationToken,
        session_id: String,
        content: String,
        attachments: Vec<Attachment>,
        context: ToolContext,
        events: mpsc::Sender<AgentEvent>,
    ) -> Result<(), AgentError> {
        let mut history = self
            .store
            .list_messages(&session_id)
            .await
            .map_err(store_error)?;
        let mut parts = vec![ContentPart::Text(TextContent { text: content })];
        if self.provider.model().supports_attachments {
            parts.extend(attachments.into_iter().map(|attachment| {
                ContentPart::Binary(super::BinaryContent {
                    path: attachment.file_path,
                    mime_type: attachment.mime_type,
                    data: attachment.content,
                })
            }));
        }
        let user_message = self
            .store
            .create_message(&session_id, MessageRole::User, parts, "")
            .await
            .map_err(store_error)?;
        history.push(user_message);

        for iteration in 1..=MAX_ITERATIONS {
            ensure_not_cancelled(&cancellation)?;
            send_event(&events, &cancellation, AgentEvent::thinking(iteration)).await?;

            let mut provider_events = self
                .provider
                .stream_response(cancellation.clone(), history.clone(), self.tools.clone())
                .await
                .map_err(provider_error)?;
            let mut assistant = self
                .store
                .create_message(
                    &session_id,
                    MessageRole::Assistant,
                    Vec::new(),
                    &self.provider.model().id.0,
                )
                .await
                .map_err(store_error)?;
            let mut completed = None;

            loop {
                let provider_event = tokio::select! {
                    biased;
                    () = cancellation.cancelled() => {
                        assistant.finish(FinishReason::Canceled);
                        let _ = self.store.update_message(assistant).await;
                        return Err(AgentError::RequestCancelled);
                    }
                    event = provider_events.recv() => event,
                };
                let Some(provider_event) = provider_event else {
                    break;
                };
                match provider_event.event_type {
                    ProviderEventType::ThinkingDelta => {
                        send_event(
                            &events,
                            &cancellation,
                            AgentEvent {
                                event_type: AgentEventType::ReasoningDelta,
                                message: None,
                                error: None,
                                thinking_message: String::new(),
                                iteration,
                                content_delta: String::new(),
                                reasoning_delta: provider_event.thinking,
                                done: false,
                            },
                        )
                        .await?;
                    }
                    ProviderEventType::ContentDelta => {
                        assistant.append_text(&provider_event.content);
                        self.store
                            .update_message(assistant.clone())
                            .await
                            .map_err(store_error)?;
                        send_event(
                            &events,
                            &cancellation,
                            AgentEvent {
                                event_type: AgentEventType::ContentDelta,
                                message: None,
                                error: None,
                                thinking_message: String::new(),
                                iteration,
                                content_delta: provider_event.content,
                                reasoning_delta: String::new(),
                                done: false,
                            },
                        )
                        .await?;
                    }
                    ProviderEventType::Error => {
                        return Err(provider_event.error.map(provider_error).unwrap_or_else(
                            || AgentError::Provider("provider stream failed".to_owned()),
                        ));
                    }
                    ProviderEventType::Complete => {
                        completed = provider_event.response;
                        break;
                    }
                    ProviderEventType::ContentStart
                    | ProviderEventType::ContentStop
                    | ProviderEventType::ToolUseStart
                    | ProviderEventType::ToolUseDelta
                    | ProviderEventType::ToolUseStop
                    | ProviderEventType::Warning => {}
                }
            }

            let response =
                completed.ok_or_else(|| provider_error(ProviderError::MissingCompletion))?;
            assistant.set_tool_calls(response.tool_calls.clone());
            assistant.finish(response.finish_reason);
            self.store
                .update_message(assistant.clone())
                .await
                .map_err(store_error)?;
            self.track_usage(&session_id, response.usage).await?;

            if response.finish_reason == FinishReason::ToolUse && !response.tool_calls.is_empty() {
                send_event(
                    &events,
                    &cancellation,
                    AgentEvent {
                        event_type: AgentEventType::Response,
                        message: Some(assistant.clone()),
                        error: None,
                        thinking_message: String::new(),
                        iteration,
                        content_delta: String::new(),
                        reasoning_delta: String::new(),
                        done: false,
                    },
                )
                .await?;
                let tool_context = context.with_message_id(assistant.id.clone());
                let results = self
                    .execute_tools(cancellation.clone(), tool_context, response.tool_calls)
                    .await;
                let tool_message = self
                    .store
                    .create_message(
                        &session_id,
                        MessageRole::Tool,
                        results.into_iter().map(ContentPart::ToolResult).collect(),
                        "",
                    )
                    .await
                    .map_err(store_error)?;
                send_event(
                    &events,
                    &cancellation,
                    AgentEvent {
                        event_type: AgentEventType::Tool,
                        message: Some(tool_message.clone()),
                        error: None,
                        thinking_message: String::new(),
                        iteration,
                        content_delta: String::new(),
                        reasoning_delta: String::new(),
                        done: false,
                    },
                )
                .await?;
                history.push(assistant);
                history.push(tool_message);
                continue;
            }

            send_event(
                &events,
                &cancellation,
                AgentEvent {
                    event_type: AgentEventType::Response,
                    message: Some(assistant),
                    error: None,
                    thinking_message: String::new(),
                    iteration,
                    content_delta: String::new(),
                    reasoning_delta: String::new(),
                    done: true,
                },
            )
            .await?;
            return Ok(());
        }

        let mut final_message = self
            .store
            .create_message(
                &session_id,
                MessageRole::Assistant,
                vec![ContentPart::Text(TextContent {
                    text: "I've made several edits to your document. Let me know if you'd like me to continue with additional changes.".to_owned(),
                })],
                &self.provider.model().id.0,
            )
            .await
            .map_err(store_error)?;
        final_message.finish(FinishReason::EndTurn);
        self.store
            .update_message(final_message.clone())
            .await
            .map_err(store_error)?;
        send_event(
            &events,
            &cancellation,
            AgentEvent {
                event_type: AgentEventType::Response,
                message: Some(final_message),
                error: None,
                thinking_message: String::new(),
                iteration: MAX_ITERATIONS + 1,
                content_delta: String::new(),
                reasoning_delta: String::new(),
                done: true,
            },
        )
        .await
    }

    async fn execute_tools(
        &self,
        cancellation: CancellationToken,
        context: ToolContext,
        calls: Vec<super::ToolCall>,
    ) -> Vec<ToolResult> {
        let resolved = calls
            .iter()
            .map(|call| {
                (
                    call.clone(),
                    self.tools
                        .iter()
                        .find(|tool| tool.info().name == call.name)
                        .cloned(),
                )
            })
            .collect::<Vec<_>>();
        let can_run_parallel = resolved.len() > 1
            && resolved
                .iter()
                .all(|(_, tool)| tool.as_ref().is_some_and(|tool| tool.info().parallel_safe));

        if can_run_parallel {
            join_all(
                resolved.into_iter().map(|(call, tool)| {
                    run_tool(cancellation.clone(), context.clone(), call, tool)
                }),
            )
            .await
        } else {
            let mut results = Vec::with_capacity(resolved.len());
            for (call, tool) in resolved {
                results.push(run_tool(cancellation.clone(), context.clone(), call, tool).await);
            }
            results
        }
    }

    async fn track_usage(&self, session_id: &str, usage: TokenUsage) -> Result<(), AgentError> {
        let mut session = self
            .store
            .get_session(session_id)
            .await
            .map_err(store_error)?;
        let model = self.provider.model();
        session.cost += model.cost_per_1m_in_cached / 1e6 * usage.cache_creation_tokens as f64
            + model.cost_per_1m_out_cached / 1e6 * usage.cache_read_tokens as f64
            + model.cost_per_1m_in / 1e6 * usage.input_tokens as f64
            + model.cost_per_1m_out / 1e6 * usage.output_tokens as f64;
        session.completion_tokens = usage.output_tokens + usage.cache_read_tokens;
        session.prompt_tokens = usage.input_tokens + usage.cache_creation_tokens;
        self.store
            .save_session(session)
            .await
            .map_err(store_error)?;
        Ok(())
    }
}

async fn run_tool(
    cancellation: CancellationToken,
    context: ToolContext,
    call: super::ToolCall,
    tool: Option<Arc<dyn Tool>>,
) -> ToolResult {
    if cancellation.is_cancelled() {
        return ToolResult {
            tool_call_id: call.id,
            content: "Tool execution canceled by user".to_owned(),
            metadata: String::new(),
            is_error: true,
        };
    }
    let Some(tool) = tool else {
        return ToolResult {
            tool_call_id: call.id,
            content: format!("Tool not found: {}", call.name),
            metadata: String::new(),
            is_error: true,
        };
    };
    match tool
        .run(
            context,
            ToolCallRequest {
                id: call.id.clone(),
                name: call.name,
                input: call.input,
            },
        )
        .await
    {
        Ok(response) => ToolResult {
            tool_call_id: call.id,
            content: response.content,
            metadata: response.metadata,
            is_error: response.is_error,
        },
        Err(error) => ToolResult {
            tool_call_id: call.id,
            content: format!("Tool execution error: {error}"),
            metadata: String::new(),
            is_error: true,
        },
    }
}

async fn send_event(
    sender: &mpsc::Sender<AgentEvent>,
    cancellation: &CancellationToken,
    event: AgentEvent,
) -> Result<(), AgentError> {
    tokio::select! {
        biased;
        () = cancellation.cancelled() => Err(AgentError::RequestCancelled),
        result = sender.send(event) => result.map_err(|_| AgentError::ReceiverDropped),
    }
}

fn ensure_not_cancelled(cancellation: &CancellationToken) -> Result<(), AgentError> {
    if cancellation.is_cancelled() {
        Err(AgentError::RequestCancelled)
    } else {
        Ok(())
    }
}

fn provider_error(error: ProviderError) -> AgentError {
    match error {
        ProviderError::Cancelled => AgentError::RequestCancelled,
        other => AgentError::Provider(other.to_string()),
    }
}

fn store_error(error: impl std::fmt::Display) -> AgentError {
    AgentError::Store(error.to_string())
}
