use std::{
    collections::{BTreeMap, HashMap},
    sync::{Arc, Mutex, Weak},
    time::{Duration, Instant},
};

use async_trait::async_trait;
use chrono::{SecondsFormat, Utc};
use futures_util::future::join_all;
use serde_json::{Map, Value, json};
use thiserror::Error;
use tokio::{sync::mpsc, task::JoinHandle};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{
    core::{
        chat::{
            ArtifactInfo, ChainOfThoughtStep, ChatMessage, ChatMessageService, MessageContext,
            MessageMetadata, ThinkingBlock, ToolExecution, ToolStepInfo,
        },
        ml::llm::{
            Agent, AgentError, AgentEvent, AgentEventType, ContentPart, LlmMessage, MessageRole,
            SessionStore, TextContent, ToolContext, ToolResult,
        },
        source::{Source, SourceService},
    },
    error::AppError,
};

use super::{
    ArticleDraftService, ChatRequest, CopilotConfig, FullMessagePayload, ReasoningStep,
    StreamResponse, ToolCallPayload, ToolGroupPayload, ToolStatusPayload, TurnStep,
    metadata::{
        ARTIFACT_STATUS_PENDING, ARTIFACT_TYPE_CODE_EDIT, MetadataBuilder, message_context,
    },
};

#[derive(Debug, Error)]
pub enum ManagerError {
    #[error("message is required")]
    MessageRequired,
    #[error("articleId is required")]
    ArticleRequired,
    #[error("invalid articleId")]
    InvalidArticle,
    #[error("maximum concurrent requests reached ({0})")]
    ConcurrencyLimit(usize),
    #[error("request not found")]
    RequestNotFound,
    #[error("request stream already taken")]
    StreamAlreadyTaken,
    #[error("copilot dependency failed")]
    Dependency,
    #[error("shutdown timeout: {0} requests still active")]
    ShutdownTimeout(usize),
}

#[async_trait]
pub trait ChatPersistencePort: Send + Sync {
    async fn save(
        &self,
        article_id: Uuid,
        role: &str,
        content: &str,
        metadata: Option<MessageMetadata>,
    ) -> Result<ChatMessage, AppError>;
    async fn history(&self, article_id: Uuid, limit: i64) -> Result<Vec<ChatMessage>, AppError>;
}

#[async_trait]
impl ChatPersistencePort for ChatMessageService {
    async fn save(
        &self,
        article_id: Uuid,
        role: &str,
        content: &str,
        metadata: Option<MessageMetadata>,
    ) -> Result<ChatMessage, AppError> {
        self.save_message(article_id, role, content, metadata).await
    }

    async fn history(&self, article_id: Uuid, limit: i64) -> Result<Vec<ChatMessage>, AppError> {
        self.conversation_history(article_id, limit).await
    }
}

#[async_trait]
pub trait SourceContextPort: Send + Sync {
    async fn list_for_article(&self, article_id: Uuid) -> Result<Vec<Source>, AppError>;
}

#[async_trait]
impl SourceContextPort for SourceService {
    async fn list_for_article(&self, article_id: Uuid) -> Result<Vec<Source>, AppError> {
        self.get_by_article_id(article_id).await
    }
}

struct RequestEntry {
    cancellation: CancellationToken,
    receiver: Option<mpsc::Receiver<StreamResponse>>,
    handle: Option<JoinHandle<Result<(), ManagerError>>>,
    created_at: Instant,
}

pub struct CopilotManager {
    agent: Arc<Agent>,
    store: Arc<dyn SessionStore>,
    chat: Arc<dyn ChatPersistencePort>,
    sources: Option<Arc<dyn SourceContextPort>>,
    drafts: Option<Arc<dyn ArticleDraftService>>,
    config: CopilotConfig,
    root_cancellation: CancellationToken,
    requests: Mutex<HashMap<String, RequestEntry>>,
}

impl CopilotManager {
    pub fn new(
        agent: Arc<Agent>,
        store: Arc<dyn SessionStore>,
        chat: Arc<dyn ChatPersistencePort>,
        sources: Option<Arc<dyn SourceContextPort>>,
        drafts: Option<Arc<dyn ArticleDraftService>>,
        config: CopilotConfig,
        root_cancellation: CancellationToken,
    ) -> Arc<Self> {
        Arc::new(Self {
            agent,
            store,
            chat,
            sources,
            drafts,
            config,
            root_cancellation,
            requests: Mutex::new(HashMap::new()),
        })
    }

    pub async fn submit(self: &Arc<Self>, request: ChatRequest) -> Result<String, ManagerError> {
        self.prune_completed();
        if request.message.is_empty() {
            return Err(ManagerError::MessageRequired);
        }
        if request.article_id.is_empty() {
            return Err(ManagerError::ArticleRequired);
        }
        let article_id =
            Uuid::parse_str(&request.article_id).map_err(|_| ManagerError::InvalidArticle)?;
        if self.active_requests() >= self.config.max_concurrent_requests.get() {
            return Err(ManagerError::ConcurrencyLimit(
                self.config.max_concurrent_requests.get(),
            ));
        }

        let request_id = Uuid::new_v4().to_string();
        let session = self
            .store
            .create_session("Writing Copilot Session")
            .await
            .map_err(|_| ManagerError::Dependency)?;
        let context = message_context(
            request.article_id.clone(),
            session.id.clone(),
            request_id.clone(),
            "",
        );
        let user_message = self
            .chat
            .save(
                article_id,
                "user",
                &request.message,
                Some(MetadataBuilder::new().with_context(context).build()),
            )
            .await
            .map_err(|_| ManagerError::Dependency)?;

        self.load_history(article_id, &session.id, Some(user_message.id))
            .await;

        let cancellation = self.root_cancellation.child_token();
        let tool_context = ToolContext::new(
            session.id.clone(),
            "",
            request_id.clone(),
            Some(article_id),
            request.document_content.clone(),
            request.document_markdown.clone(),
            cancellation.clone(),
        );
        let prompt = self.build_prompt(article_id, &request).await;
        let run = self
            .agent
            .start(
                cancellation.clone(),
                session.id.clone(),
                prompt,
                Vec::new(),
                tool_context,
            )
            .map_err(|_| ManagerError::Dependency)?;
        let (sender, receiver) = mpsc::channel(self.config.channel_buffer.get());
        let manager = Arc::downgrade(self);
        let task_request_id = request_id.clone();
        let task_request = request.clone();
        let request_timeout = self.config.request_timeout;
        let run_cancellation = run.cancellation.clone();
        let handle = tokio::spawn(async move {
            process_run(
                manager,
                task_request_id,
                task_request,
                session.id,
                article_id,
                run.events,
                run.handle,
                run_cancellation,
                sender,
                request_timeout,
            )
            .await
        });

        self.requests
            .lock()
            .map_err(|_| ManagerError::Dependency)?
            .insert(
                request_id.clone(),
                RequestEntry {
                    cancellation,
                    receiver: Some(receiver),
                    handle: Some(handle),
                    created_at: Instant::now(),
                },
            );
        Ok(request_id)
    }

    pub fn take_response_stream(
        &self,
        request_id: &str,
    ) -> Result<mpsc::Receiver<StreamResponse>, ManagerError> {
        let mut requests = self.requests.lock().map_err(|_| ManagerError::Dependency)?;
        let request = requests
            .get_mut(request_id)
            .ok_or(ManagerError::RequestNotFound)?;
        request
            .receiver
            .take()
            .ok_or(ManagerError::StreamAlreadyTaken)
    }

    pub fn cancel_request(&self, request_id: &str) -> Result<(), ManagerError> {
        let requests = self.requests.lock().map_err(|_| ManagerError::Dependency)?;
        let request = requests
            .get(request_id)
            .ok_or(ManagerError::RequestNotFound)?;
        request.cancellation.cancel();
        Ok(())
    }

    pub fn active_requests(&self) -> usize {
        self.requests
            .lock()
            .map(|requests| {
                requests
                    .values()
                    .filter(|request| {
                        request
                            .handle
                            .as_ref()
                            .is_some_and(|handle| !handle.is_finished())
                    })
                    .count()
            })
            .unwrap_or(usize::MAX)
    }

    pub async fn shutdown(&self, timeout: Duration) -> Result<(), ManagerError> {
        self.root_cancellation.cancel();
        let mut handles = {
            let mut requests = self.requests.lock().map_err(|_| ManagerError::Dependency)?;
            for request in requests.values() {
                request.cancellation.cancel();
            }
            requests
                .values_mut()
                .filter_map(|request| request.handle.take())
                .collect::<Vec<_>>()
        };
        let joined = tokio::time::timeout(timeout, join_all(handles.iter_mut())).await;
        if joined.is_err() {
            let active = handles
                .iter()
                .filter(|handle| !handle.is_finished())
                .count();
            for handle in &handles {
                handle.abort();
            }
            let _ = join_all(handles).await;
            return Err(ManagerError::ShutdownTimeout(active));
        }
        Ok(())
    }

    fn prune_completed(&self) {
        let Ok(mut requests) = self.requests.lock() else {
            return;
        };
        let cleanup_delay = self.config.cleanup_delay;
        requests.retain(|_, request| {
            let finished = request.handle.as_ref().is_none_or(JoinHandle::is_finished);
            !(finished && request.created_at.elapsed() >= cleanup_delay)
        });
    }

    async fn load_history(&self, article_id: Uuid, session_id: &str, exclude_id: Option<Uuid>) {
        let Ok(messages) = self.chat.history(article_id, 30).await else {
            return;
        };
        let mut reconstructed = messages
            .into_iter()
            .filter(|message| Some(message.id) != exclude_id)
            .flat_map(reconstruct_messages)
            .collect::<Vec<_>>();
        sanitize_history(&mut reconstructed);
        for message in reconstructed {
            let _ = self
                .store
                .create_message(session_id, message.role, message.parts, "loaded")
                .await;
        }
    }

    async fn build_prompt(&self, article_id: Uuid, request: &ChatRequest) -> String {
        let document = if request.document_markdown.is_empty() {
            &request.document_content
        } else {
            &request.document_markdown
        };
        let mut prompt = format!(
            "{}\n\n{}",
            request.message,
            generate_document_context(document)
        );
        if let Some(sources) = &self.sources
            && let Ok(sources) = sources.list_for_article(article_id).await
        {
            let source_context = format_source_context(sources);
            if !source_context.is_empty() {
                prompt.push_str("\n\n");
                prompt.push_str(&source_context);
            }
        }
        prompt
    }
}

#[allow(clippy::too_many_arguments)]
async fn process_run(
    manager: Weak<CopilotManager>,
    request_id: String,
    request: ChatRequest,
    session_id: String,
    article_id: Uuid,
    mut agent_events: mpsc::Receiver<AgentEvent>,
    agent_handle: JoinHandle<Result<(), AgentError>>,
    cancellation: CancellationToken,
    sender: mpsc::Sender<StreamResponse>,
    request_timeout: Duration,
) -> Result<(), ManagerError> {
    let Some(manager_ref) = manager.upgrade() else {
        cancellation.cancel();
        let _ = agent_handle.await;
        return Ok(());
    };
    if let Some(drafts) = &manager_ref.drafts
        && let Ok(Some(snapshot_id)) = drafts.create_draft_snapshot(article_id).await
    {
        let mut event = StreamResponse::new(&request_id, "turn_started");
        event.data = Some(json!({"snapshot_version_id": snapshot_id}));
        send_stream(&sender, &cancellation, event).await?;
    }
    drop(manager_ref);

    let deadline = tokio::time::sleep(request_timeout);
    tokio::pin!(deadline);
    let mut iteration = 1;
    let mut steps: Vec<TurnStep> = Vec::new();
    let mut current_step: Option<usize> = None;

    loop {
        let event = tokio::select! {
            biased;
            () = cancellation.cancelled() => {
                break;
            }
            () = &mut deadline => {
                cancellation.cancel();
                let error = StreamResponse::terminal_error(&request_id, "request timed out");
                let _ = sender.send(error).await;
                break;
            }
            event = agent_events.recv() => event,
        };
        let Some(event) = event else {
            break;
        };
        let event_done = event.done;
        match event.event_type {
            AgentEventType::Thinking => {
                let mut stream = StreamResponse::new(&request_id, "thinking");
                stream.thinking_message = event.thinking_message;
                stream.iteration = event.iteration;
                send_stream(&sender, &cancellation, stream).await?;
            }
            AgentEventType::ReasoningDelta => {
                let index = if current_step.is_some_and(|index| {
                    steps
                        .get(index)
                        .is_some_and(|step| step.step_type == "reasoning")
                }) {
                    current_step.unwrap_or_default()
                } else {
                    steps.push(TurnStep {
                        step_type: "reasoning".to_owned(),
                        reasoning: Some(ReasoningStep {
                            content: String::new(),
                            duration_ms: 0,
                            visible: true,
                        }),
                        tool: None,
                        content: String::new(),
                    });
                    steps.len() - 1
                };
                if let Some(reasoning) = steps[index].reasoning.as_mut() {
                    reasoning.content.push_str(&event.reasoning_delta);
                }
                current_step = Some(index);
                let mut stream = StreamResponse::new(&request_id, "reasoning_delta");
                stream.thinking_content = event.reasoning_delta;
                stream.iteration = iteration;
                stream.step_index = index;
                send_stream(&sender, &cancellation, stream).await?;
            }
            AgentEventType::ContentDelta => {
                let index = if current_step.is_some_and(|index| {
                    steps
                        .get(index)
                        .is_some_and(|step| step.step_type == "content")
                }) {
                    current_step.unwrap_or_default()
                } else {
                    steps.push(TurnStep {
                        step_type: "content".to_owned(),
                        reasoning: None,
                        tool: None,
                        content: String::new(),
                    });
                    steps.len() - 1
                };
                steps[index].content.push_str(&event.content_delta);
                current_step = Some(index);
                let mut stream = StreamResponse::new(&request_id, "content_delta");
                stream.content = event.content_delta;
                stream.iteration = iteration;
                stream.step_index = index;
                send_stream(&sender, &cancellation, stream).await?;
            }
            AgentEventType::Response => {
                iteration += 1;
                if let Some(message) = event.message {
                    let calls = message.tool_calls();
                    if calls.is_empty() {
                        persist_assistant(
                            &manager,
                            article_id,
                            &request,
                            &request_id,
                            &session_id,
                            &message,
                            &steps,
                        )
                        .await;
                        let mut stream = StreamResponse::new(&request_id, "text");
                        stream.content = message.text();
                        stream.iteration = iteration;
                        stream.step_index = current_step.unwrap_or_default();
                        send_stream(&sender, &cancellation, stream).await?;
                        steps.clear();
                        current_step = None;
                    } else {
                        for call in calls {
                            let input = serde_json::from_str(&call.input)
                                .unwrap_or_else(|_| json!({"raw": call.input}));
                            let tool = ToolStatusPayload {
                                group_id: String::new(),
                                tool_id: call.id.clone(),
                                name: call.name.clone(),
                                status: "running".to_owned(),
                                result: BTreeMap::new(),
                                error: String::new(),
                                completed_at: String::new(),
                                duration_ms: 0,
                            };
                            steps.push(TurnStep {
                                step_type: "tool".to_owned(),
                                reasoning: None,
                                tool: Some(tool),
                                content: String::new(),
                            });
                            current_step = Some(steps.len() - 1);
                            let mut stream = StreamResponse::new(&request_id, "tool_use");
                            stream.iteration = iteration;
                            stream.step_index = steps.len() - 1;
                            stream.tool_id = call.id;
                            stream.tool_name = call.name;
                            stream.tool_input = Some(input);
                            send_stream(&sender, &cancellation, stream).await?;
                        }
                    }
                }
            }
            AgentEventType::Tool => {
                if let Some(message) = event.message {
                    stream_tool_results(
                        &manager,
                        &sender,
                        &cancellation,
                        &request_id,
                        article_id,
                        &request,
                        &session_id,
                        iteration,
                        &mut steps,
                        message.tool_results(),
                    )
                    .await?;
                }
            }
            AgentEventType::Error => {
                let error = event
                    .error
                    .map(|error| error.to_string())
                    .unwrap_or_else(|| "agent request failed".to_owned());
                send_stream(
                    &sender,
                    &cancellation,
                    StreamResponse::terminal_error(&request_id, error),
                )
                .await?;
                break;
            }
        }
        if event_done {
            let mut done = StreamResponse::new(&request_id, "done");
            done.done = true;
            send_stream(&sender, &cancellation, done).await?;
            break;
        }
    }

    cancellation.cancel();
    if !agent_handle.is_finished() {
        agent_handle.abort();
    }
    let _ = agent_handle.await;
    Ok(())
}

async fn persist_assistant(
    manager: &Weak<CopilotManager>,
    article_id: Uuid,
    request: &ChatRequest,
    request_id: &str,
    session_id: &str,
    message: &LlmMessage,
    steps: &[TurnStep],
) {
    let Some(manager) = manager.upgrade() else {
        return;
    };
    let context = message_context(&request.article_id, session_id, request_id, "");
    let converted = steps.iter().map(convert_step).collect::<Vec<_>>();
    let mut builder = MetadataBuilder::new()
        .with_context(context)
        .with_steps(converted);
    let reasoning = steps
        .iter()
        .filter_map(|step| step.reasoning.as_ref())
        .map(|reasoning| reasoning.content.as_str())
        .collect::<String>();
    if !reasoning.is_empty() {
        builder = builder.with_thinking(ThinkingBlock {
            content: reasoning,
            duration_ms: 0,
            visible: true,
        });
    }
    let _ = manager
        .chat
        .save(
            article_id,
            "assistant",
            &message.text(),
            Some(builder.build()),
        )
        .await;
}

#[allow(clippy::too_many_arguments)]
async fn stream_tool_results(
    manager: &Weak<CopilotManager>,
    sender: &mpsc::Sender<StreamResponse>,
    cancellation: &CancellationToken,
    request_id: &str,
    article_id: Uuid,
    request: &ChatRequest,
    session_id: &str,
    iteration: usize,
    steps: &mut [TurnStep],
    results: Vec<ToolResult>,
) -> Result<(), ManagerError> {
    let group_id = Uuid::new_v4().to_string();
    let mut calls = Vec::with_capacity(results.len());
    for result in &results {
        let parsed: Map<String, Value> =
            serde_json::from_str(&result.content).unwrap_or_else(|_| {
                Map::from_iter([
                    ("content".to_owned(), Value::String(result.content.clone())),
                    ("is_error".to_owned(), Value::Bool(result.is_error)),
                ])
            });
        let tool_name = parsed
            .get("tool_name")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned();
        let status = if result.is_error {
            "error"
        } else {
            "completed"
        };
        let completed_at = Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true);
        let record = ToolCallPayload {
            id: result.tool_call_id.clone(),
            name: tool_name.clone(),
            input: BTreeMap::new(),
            status: status.to_owned(),
            result: parsed.clone().into_iter().collect(),
            error: if result.is_error {
                result.content.clone()
            } else {
                String::new()
            },
            started_at: String::new(),
            completed_at: completed_at.clone(),
            duration_ms: 0,
        };
        for step in steps.iter_mut() {
            if let Some(tool) = step.tool.as_mut()
                && tool.tool_id == result.tool_call_id
            {
                tool.status = status.to_owned();
                tool.name = tool_name.clone();
                tool.result = parsed.clone().into_iter().collect();
                tool.error = record.error.clone();
                tool.completed_at = completed_at.clone();
            }
        }
        calls.push(record);

        if let Some(saved) = persist_tool_result(
            manager, article_id, request, request_id, session_id, result, &tool_name, &parsed,
        )
        .await
        {
            let meta_data = saved
                .meta_data
                .as_ref()
                .and_then(Value::as_object)
                .cloned()
                .unwrap_or_default()
                .into_iter()
                .collect();
            let mut stream = StreamResponse::new(request_id, "full_message");
            stream.iteration = iteration;
            stream.full_message = Some(FullMessagePayload {
                id: saved.id.to_string(),
                article_id: saved.article_id.to_string(),
                role: saved.role,
                content: saved.content,
                meta_data,
                created_at: saved
                    .created_at
                    .unwrap_or_else(Utc::now)
                    .to_rfc3339_opts(SecondsFormat::Secs, true),
            });
            send_stream(sender, cancellation, stream).await?;
        }
    }

    let mut complete = StreamResponse::new(request_id, "tool_group_complete");
    complete.iteration = iteration;
    complete.tool_group = Some(ToolGroupPayload {
        group_id,
        status: "completed".to_owned(),
        calls,
    });
    send_stream(sender, cancellation, complete).await?;

    for result in results {
        let parsed: Map<String, Value> = serde_json::from_str(&result.content).unwrap_or_default();
        let name = parsed
            .get("tool_name")
            .and_then(Value::as_str)
            .unwrap_or_default();
        let mut stream = StreamResponse::new(request_id, "tool_result");
        stream.iteration = iteration;
        stream.tool_id = result.tool_call_id;
        stream.tool_name = name.to_owned();
        stream.tool_result = Some(json!({
            "content": result.content,
            "metadata": result.metadata,
            "is_error": result.is_error,
            "is_search": name == "search_web_sources" || name == "ask_question",
            "tool_name": name,
        }));
        send_stream(sender, cancellation, stream).await?;
    }
    Ok(())
}

#[allow(clippy::too_many_arguments)]
async fn persist_tool_result(
    manager: &Weak<CopilotManager>,
    article_id: Uuid,
    request: &ChatRequest,
    request_id: &str,
    session_id: &str,
    result: &ToolResult,
    tool_name: &str,
    parsed: &Map<String, Value>,
) -> Option<ChatMessage> {
    let manager = manager.upgrade()?;
    let execution = ToolExecution {
        tool_name: tool_name.to_owned(),
        tool_id: result.tool_call_id.clone(),
        input: Value::Null,
        output: Value::Object(parsed.clone()),
        error: if result.is_error {
            result.content.clone()
        } else {
            String::new()
        },
        duration_ms: 0,
        executed_at: Utc::now(),
        success: !result.is_error,
    };
    let mut builder = MetadataBuilder::new()
        .with_context(MessageContext {
            article_id: request.article_id.clone(),
            session_id: session_id.to_owned(),
            request_id: request_id.to_owned(),
            document_version: String::new(),
            document_hash: String::new(),
            user_id: String::new(),
        })
        .with_tool_execution(execution);
    let content = match tool_name {
        "replace_lines" => {
            let proposed = parsed
                .get("new_str")
                .and_then(Value::as_str)
                .unwrap_or_default();
            let original = parsed
                .get("old_str")
                .and_then(Value::as_str)
                .unwrap_or_default();
            let reason = parsed
                .get("reason")
                .and_then(Value::as_str)
                .unwrap_or_default();
            builder = builder.with_artifact(ArtifactInfo {
                id: Uuid::new_v4().to_string(),
                artifact_type: ARTIFACT_TYPE_CODE_EDIT.to_owned(),
                status: if result.is_error {
                    "error".to_owned()
                } else {
                    ARTIFACT_STATUS_PENDING.to_owned()
                },
                content: proposed.to_owned(),
                diff_preview: format!(
                    "Old: {}\nNew: {}",
                    truncate_chars(original, 50),
                    truncate_chars(proposed, 50)
                ),
                title: "replace_lines result".to_owned(),
                description: if result.is_error {
                    result.content.clone()
                } else {
                    reason.to_owned()
                },
                applied_at: None,
            });
            String::new()
        }
        "search_web_sources" => format!(
            "🔍 Web search completed: Found {} results, created {} sources",
            parsed
                .get("total_found")
                .and_then(Value::as_i64)
                .unwrap_or(0),
            parsed
                .get("sources_successful")
                .and_then(Value::as_i64)
                .unwrap_or(0)
        ),
        "get_relevant_sources" => format!(
            "📚 Retrieved {} relevant source excerpts",
            parsed
                .get("total_found")
                .and_then(Value::as_i64)
                .unwrap_or(0)
        ),
        "ask_question" => format!(
            "❓ Question answered with {} citations",
            parsed
                .get("citations")
                .and_then(Value::as_array)
                .map(Vec::len)
                .unwrap_or(0)
        ),
        "select_sources_for_edit" => format!(
            "🧠 Selected {} sources for edit context",
            parsed
                .get("selected_count")
                .and_then(Value::as_i64)
                .unwrap_or(0)
        ),
        _ => return None,
    };
    manager
        .chat
        .save(article_id, "assistant", &content, Some(builder.build()))
        .await
        .ok()
}

fn reconstruct_messages(message: ChatMessage) -> Vec<LlmMessage> {
    let role = match message.role.as_str() {
        "assistant" => MessageRole::Assistant,
        "tool" => MessageRole::Tool,
        _ => MessageRole::User,
    };
    let created_at = message.created_at.unwrap_or_else(Utc::now);
    let mut parts = vec![ContentPart::Text(TextContent {
        text: message.content,
    })];
    let mut tool_result = None;
    if let Some(metadata) = message
        .meta_data
        .and_then(|value| serde_json::from_value::<MessageMetadata>(value).ok())
        && let Some(execution) = metadata.tool_execution
    {
        parts.push(ContentPart::ToolCall(crate::core::ml::llm::ToolCall {
            id: execution.tool_id.clone(),
            name: execution.tool_name,
            input: serde_json::to_string(&execution.input).unwrap_or_default(),
            r#type: String::new(),
            finished: execution.success,
            thought_signature: Vec::new(),
        }));
        if metadata.artifact.is_some() {
            tool_result = Some(ToolResult {
                tool_call_id: execution.tool_id,
                content: serde_json::to_string(&execution.output).unwrap_or_default(),
                metadata: String::new(),
                is_error: !execution.success,
            });
        }
    }
    let message = LlmMessage {
        id: message.id.to_string(),
        session_id: String::new(),
        role,
        parts,
        model: String::new(),
        created_at,
        updated_at: created_at,
    };
    if let Some(result) = tool_result {
        vec![
            message,
            LlmMessage {
                id: Uuid::new_v4().to_string(),
                session_id: String::new(),
                role: MessageRole::Tool,
                parts: vec![ContentPart::ToolResult(result)],
                model: String::new(),
                created_at,
                updated_at: created_at,
            },
        ]
    } else {
        vec![message]
    }
}

fn sanitize_history(messages: &mut Vec<LlmMessage>) {
    let mut sanitized: Vec<LlmMessage> = Vec::with_capacity(messages.len());
    for message in messages.drain(..) {
        if let Some(previous) = sanitized.last() {
            if message.role == previous.role && message.role != MessageRole::Tool {
                if let Some(previous) = sanitized.last_mut() {
                    *previous = message;
                }
                continue;
            }
            if message.role == MessageRole::Tool && previous.role != MessageRole::Assistant {
                continue;
            }
        }
        sanitized.push(message);
    }
    let first_user = sanitized
        .iter()
        .position(|message| message.role == MessageRole::User)
        .unwrap_or(sanitized.len());
    sanitized.drain(..first_user);
    *messages = sanitized;
}

fn convert_step(step: &TurnStep) -> ChainOfThoughtStep {
    ChainOfThoughtStep {
        step_type: step.step_type.clone(),
        reasoning: step.reasoning.as_ref().map(|reasoning| ThinkingBlock {
            content: reasoning.content.clone(),
            duration_ms: reasoning.duration_ms,
            visible: reasoning.visible,
        }),
        tool: step.tool.as_ref().map(|tool| ToolStepInfo {
            tool_id: tool.tool_id.clone(),
            tool_name: tool.name.clone(),
            input: Map::new(),
            output: tool.result.clone().into_iter().collect(),
            status: tool.status.clone(),
            error: tool.error.clone(),
            started_at: String::new(),
            completed_at: tool.completed_at.clone(),
            duration_ms: tool.duration_ms,
        }),
        content: step.content.clone(),
    }
}

fn generate_document_context(markdown: &str) -> String {
    if markdown.trim().is_empty() {
        return "--- Document Context ---\nTotal: 0 lines, 0 chars, 0 paragraphs\n(empty document)\n---".to_owned();
    }
    let lines = markdown.lines().collect::<Vec<_>>();
    let paragraphs = lines
        .iter()
        .fold((0, true), |(count, previous_empty), line| {
            let empty = line.trim().is_empty();
            (count + usize::from(!empty && previous_empty), empty)
        })
        .0;
    let sections = lines
        .iter()
        .enumerate()
        .filter_map(|(index, line)| {
            let trimmed = line.trim();
            let level = trimmed
                .chars()
                .take_while(|character| *character == '#')
                .count();
            (level > 0 && level <= 6 && !trimmed[level..].trim().is_empty()).then_some((
                index + 1,
                level,
                trimmed,
            ))
        })
        .collect::<Vec<_>>();
    if sections.is_empty() {
        return format!(
            "--- Document Context ---\nTotal: {} lines, {} chars, {} paragraphs\n(no headings found)\n---",
            lines.len(),
            markdown.len(),
            paragraphs
        );
    }
    let mut outline = String::new();
    for (index, (line, level, heading)) in sections.iter().enumerate() {
        let next_line = sections
            .get(index + 1)
            .map(|section| section.0)
            .unwrap_or(lines.len() + 1);
        outline.push_str(&format!(
            "{}{:4}| {} ({} lines)\n",
            "  ".repeat(level.saturating_sub(2)),
            line,
            heading,
            next_line - line
        ));
    }
    format!(
        "--- Document Context ---\nTotal: {} lines, {} chars, {} paragraphs\nSections:\n{}---",
        lines.len(),
        markdown.len(),
        paragraphs,
        outline
    )
}

fn format_source_context(mut sources: Vec<Source>) -> String {
    sources.sort_by(|left, right| right.created_at.cmp(&left.created_at));
    let mut output = String::from("Available Sources:\n");
    for source in sources {
        output.push_str(&format!("- [{}] {}", source.id, source.title));
        if !source.url.is_empty() {
            output.push_str(&format!(" | {}", source.url));
        }
        if !source.source_type.is_empty() {
            output.push_str(&format!(" | type={}", source.source_type));
        }
        output.push('\n');
        if !source.content.trim().is_empty() {
            output.push_str("  preview: ");
            output.push_str(&truncate_chars(source.content.trim(), 220));
            output.push('\n');
        }
    }
    output.trim().to_owned()
}

fn truncate_chars(value: &str, limit: usize) -> String {
    let mut characters = value.chars();
    let prefix = characters.by_ref().take(limit).collect::<String>();
    if characters.next().is_some() {
        format!("{prefix}...")
    } else {
        prefix
    }
}

async fn send_stream(
    sender: &mpsc::Sender<StreamResponse>,
    cancellation: &CancellationToken,
    event: StreamResponse,
) -> Result<(), ManagerError> {
    tokio::select! {
        biased;
        () = cancellation.cancelled() => Err(ManagerError::Dependency),
        result = sender.send(event) => result.map_err(|_| ManagerError::Dependency),
    }
}
