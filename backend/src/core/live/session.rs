use std::sync::Arc;

use async_trait::async_trait;
use serde_json::{Value, json};
use tokio::{sync::mpsc, task::JoinSet};
use uuid::Uuid;

use crate::error::AppError;

pub const LIVE_MODEL: &str = "gpt-live-1";
const COMMENTARY_LIMIT: usize = 1_500;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LiveClientCommand {
    Start {
        article_id: String,
        document_content: String,
        document_markdown: String,
    },
    Audio {
        pcm_base64: String,
    },
    Text {
        text: String,
    },
    Context {
        document_content: String,
        document_markdown: String,
    },
    Stop,
}

#[derive(Debug, Clone)]
pub struct LiveTurn {
    pub article_id: String,
    pub message: String,
    pub document_content: String,
    pub document_markdown: String,
}

pub struct LiveTurnHandle {
    pub request_id: String,
    pub events: mpsc::Receiver<Value>,
}

#[async_trait]
pub trait LiveHarness: Send + Sync {
    async fn run_turn(&self, turn: LiveTurn) -> Result<LiveTurnHandle, AppError>;
}

#[async_trait]
pub trait LiveConnection: Send {
    async fn send_event(&mut self, event: Value) -> Result<(), AppError>;
    async fn recv_event(&mut self) -> Result<Option<Value>, AppError>;
}

#[async_trait]
pub trait LiveUpstream: Send + Sync {
    async fn connect(&self) -> Result<Box<dyn LiveConnection>, AppError>;
}

enum SessionNote {
    Upstream(Value),
    Browser(Value),
    Finished,
}

struct SessionState {
    article_id: String,
    document_content: String,
    document_markdown: String,
    user_transcript: String,
    typed: Vec<String>,
    in_flight: bool,
    pending: String,
}

pub fn live_websocket_url(base_url: &str) -> String {
    let trimmed = base_url.trim().trim_end_matches('/');
    let with_scheme = if let Some(rest) = trimmed.strip_prefix("https://") {
        format!("wss://{rest}")
    } else if let Some(rest) = trimmed.strip_prefix("http://") {
        format!("ws://{rest}")
    } else if trimmed.starts_with("wss://") || trimmed.starts_with("ws://") {
        trimmed.to_owned()
    } else {
        format!("wss://{trimmed}")
    };
    format!("{with_scheme}/live/sessions")
}

pub fn session_start_event(instructions: &str) -> Value {
    json!({
        "type": "session.start",
        "event_id": format!("start_{}", Uuid::new_v4().simple()),
        "session": {
            "model": LIVE_MODEL,
            "instructions": instructions,
            "audio": {
                "format": {"type": "audio/pcm", "rate": 24000},
                "output": {"voice": "marin"}
            },
            "delegation": {"type": "client"}
        }
    })
}

pub async fn run_live_session(
    mut upstream: Box<dyn LiveConnection>,
    mut commands: mpsc::Receiver<LiveClientCommand>,
    browser: mpsc::Sender<Value>,
    harness: Arc<dyn LiveHarness>,
) -> Result<(), AppError> {
    let Some(start) = commands.recv().await else {
        return Ok(());
    };
    let LiveClientCommand::Start {
        article_id,
        document_content,
        document_markdown,
    } = start
    else {
        let _ = browser
            .send(json!({"type": "error", "error": "live session must start first"}))
            .await;
        return Err(AppError::InvalidInput(
            "live session must start first".to_owned(),
        ));
    };
    if article_id.trim().is_empty() {
        let _ = browser
            .send(json!({"type": "error", "error": "articleId is required"}))
            .await;
        return Err(AppError::InvalidInput("articleId is required".to_owned()));
    }

    upstream
        .send_event(session_start_event(LIVE_INSTRUCTIONS))
        .await?;
    let mut state = SessionState {
        article_id,
        document_content,
        document_markdown,
        user_transcript: String::new(),
        typed: Vec::new(),
        in_flight: false,
        pending: String::new(),
    };
    let (notes_tx, mut notes) = mpsc::channel(64);
    let mut tasks = JoinSet::new();

    loop {
        tokio::select! {
            incoming = upstream.recv_event() => {
                match incoming? {
                    None => break,
                    Some(event) => {
                        handle_upstream_event(event, &mut state, &browser, &notes_tx, &harness, &mut tasks).await?;
                    }
                }
            }
            command = commands.recv() => {
                let Some(command) = command else { break };
                if matches!(command, LiveClientCommand::Stop) {
                    break;
                }
                handle_client_command(command, &mut state, &mut upstream, &browser, &notes_tx, &harness, &mut tasks).await?;
            }
            note = notes.recv() => {
                match note {
                    Some(SessionNote::Upstream(event)) => upstream.send_event(event).await?,
                    Some(SessionNote::Browser(event)) => {
                        if browser.send(event).await.is_err() {
                            break;
                        }
                    }
                    Some(SessionNote::Finished) => {
                        state.in_flight = false;
                        if !state.pending.trim().is_empty() {
                            let message = std::mem::take(&mut state.pending);
                            begin_turn(&mut state, message, &browser, &notes_tx, &harness, &mut tasks).await?;
                        }
                    }
                    None => {}
                }
            }
        }
    }
    tasks.abort_all();
    Ok(())
}

async fn handle_client_command(
    command: LiveClientCommand,
    state: &mut SessionState,
    upstream: &mut Box<dyn LiveConnection>,
    browser: &mpsc::Sender<Value>,
    notes: &mpsc::Sender<SessionNote>,
    harness: &Arc<dyn LiveHarness>,
    tasks: &mut JoinSet<()>,
) -> Result<(), AppError> {
    match command {
        LiveClientCommand::Audio { pcm_base64 } => {
            if pcm_base64.trim().is_empty() {
                return Ok(());
            }
            upstream
                .send_event(json!({
                    "type": "session.input_audio.append",
                    "event_id": event_id("audio"),
                    "audio": pcm_base64,
                }))
                .await
        }
        LiveClientCommand::Text { text } => {
            let text = text.trim().to_owned();
            if text.is_empty() {
                return Ok(());
            }
            state.typed.push(text.clone());
            upstream
                .send_event(thinking_append(&format!(
                    "The user typed this while the voice session is open. Treat it as part of the same conversation: {text}"
                )))
                .await?;
            let _ = browser
                .send(
                    json!({"type": "transcript", "role": "user", "text": combined_message(state)}),
                )
                .await;
            let message = combined_message(state);
            state.user_transcript.clear();
            state.typed.clear();
            notes
                .send(SessionNote::Upstream(commentary_append(
                    None,
                    "Let me take care of that.",
                )))
                .await
                .map_err(|_| AppError::External)?;
            if state.in_flight {
                state.pending = message;
            } else {
                begin_turn(state, message, browser, notes, harness, tasks).await?;
            }
            Ok(())
        }
        LiveClientCommand::Context {
            document_content,
            document_markdown,
        } => {
            state.document_content = document_content;
            state.document_markdown = document_markdown;
            Ok(())
        }
        LiveClientCommand::Start { .. } | LiveClientCommand::Stop => Ok(()),
    }
}

async fn handle_upstream_event(
    event: Value,
    state: &mut SessionState,
    browser: &mpsc::Sender<Value>,
    notes: &mpsc::Sender<SessionNote>,
    harness: &Arc<dyn LiveHarness>,
    tasks: &mut JoinSet<()>,
) -> Result<(), AppError> {
    let event_type = event.get("type").and_then(Value::as_str).unwrap_or("");
    match event_type {
        "session.started" => {
            let _ = browser
                .send(json!({
                    "type": "started",
                    "model": LIVE_MODEL,
                    "sessionId": event.get("session").and_then(|session| session.get("id")).cloned().unwrap_or(Value::Null),
                }))
                .await;
            let excerpt = truncate_chars(
                if state.document_markdown.is_empty() {
                    state.document_content.as_str()
                } else {
                    state.document_markdown.as_str()
                },
                800,
            );
            if !excerpt.is_empty() {
                let note = thinking_append(&format!(
                    "The user is editing this blog article. Current document excerpt: {excerpt}"
                ));
                notes
                    .send(SessionNote::Upstream(note))
                    .await
                    .map_err(|_| AppError::External)?;
            }
        }
        "session.input_transcript.delta" => {
            if let Some(delta) = event.get("delta").and_then(Value::as_str) {
                state.user_transcript.push_str(delta);
                let _ = browser
                    .send(json!({
                        "type": "transcript",
                        "role": "user",
                        "text": combined_message(state),
                    }))
                    .await;
            }
        }
        "session.output_transcript.delta" => {
            if let Some(delta) = event.get("delta").and_then(Value::as_str) {
                let _ = browser
                    .send(json!({"type": "transcript", "role": "assistant", "delta": delta}))
                    .await;
            }
        }
        "session.output_audio.delta" => {
            let audio = event
                .get("delta")
                .or_else(|| event.get("audio"))
                .and_then(Value::as_str)
                .unwrap_or("");
            if !audio.is_empty() {
                let _ = browser
                    .send(json!({"type": "audio", "audio": audio, "sampleRate": 24000}))
                    .await;
            }
        }
        "session.delegation.created" => {
            let delegation_id = event
                .pointer("/delegation/id")
                .and_then(Value::as_str)
                .unwrap_or("")
                .to_owned();
            let message = combined_message(state);
            state.user_transcript.clear();
            state.typed.clear();
            let acknowledge = commentary_append(
                if delegation_id.is_empty() {
                    None
                } else {
                    Some(delegation_id)
                },
                "Let me take care of that.",
            );
            notes
                .send(SessionNote::Upstream(acknowledge))
                .await
                .map_err(|_| AppError::External)?;
            if message.trim().is_empty() {
                return Ok(());
            }
            if state.in_flight {
                state.pending = message;
            } else {
                begin_turn(state, message, browser, notes, harness, tasks).await?;
            }
        }
        "error" => {
            let message = event
                .get("error")
                .and_then(|value| value.get("message").or(Some(value)))
                .map(|value| value.to_string())
                .unwrap_or_else(|| "live session failed".to_owned());
            let _ = browser
                .send(json!({"type": "error", "error": message}))
                .await;
        }
        _ => {}
    }
    Ok(())
}

async fn begin_turn(
    state: &mut SessionState,
    message: String,
    browser: &mpsc::Sender<Value>,
    notes: &mpsc::Sender<SessionNote>,
    harness: &Arc<dyn LiveHarness>,
    tasks: &mut JoinSet<()>,
) -> Result<(), AppError> {
    state.in_flight = true;
    let turn = LiveTurn {
        article_id: state.article_id.clone(),
        message: message.clone(),
        document_content: state.document_content.clone(),
        document_markdown: state.document_markdown.clone(),
    };
    let handle = harness.run_turn(turn).await?;
    let _ = browser
        .send(json!({
            "type": "delegation",
            "requestId": handle.request_id,
            "message": message,
        }))
        .await;
    let notes = notes.clone();
    tasks.spawn(async move {
        forward_harness(handle, notes).await;
    });
    Ok(())
}

async fn forward_harness(mut handle: LiveTurnHandle, notes: mpsc::Sender<SessionNote>) {
    let mut spoken = String::new();
    while let Some(event) = handle.events.recv().await {
        let event_type = event.get("type").and_then(Value::as_str);
        if matches!(event_type, Some("text") | Some("content_delta"))
            && let Some(content) = event.get("content").and_then(Value::as_str)
        {
            spoken.push_str(content);
        }
        if notes
            .send(SessionNote::Browser(
                json!({"type": "harness", "event": event}),
            ))
            .await
            .is_err()
        {
            let _ = notes.send(SessionNote::Finished).await;
            return;
        }
    }
    let summary = if spoken.trim().is_empty() {
        "The task is finished.".to_owned()
    } else {
        truncate_chars(spoken.trim(), COMMENTARY_LIMIT)
    };
    let _ = notes
        .send(SessionNote::Upstream(commentary_append(None, &summary)))
        .await;
    let _ = notes.send(SessionNote::Finished).await;
}

fn combined_message(state: &SessionState) -> String {
    let transcript = state.user_transcript.trim();
    let typed = state
        .typed
        .iter()
        .map(|text| text.trim())
        .filter(|text| !text.is_empty())
        .collect::<Vec<_>>()
        .join("\n\n");
    match (transcript.is_empty(), typed.is_empty()) {
        (true, true) => String::new(),
        (false, true) => transcript.to_owned(),
        (true, false) => typed,
        (false, false) => format!("{transcript}\n\n{typed}"),
    }
}

fn thinking_append(content: &str) -> Value {
    json!({
        "type": "session.thinking.append",
        "event_id": event_id("thinking"),
        "delegation_id": null,
        "content": truncate_chars(content, COMMENTARY_LIMIT),
    })
}

fn commentary_append(delegation_id: Option<String>, content: &str) -> Value {
    json!({
        "type": "session.commentary.append",
        "event_id": event_id("commentary"),
        "delegation_id": delegation_id,
        "content": truncate_chars(content, COMMENTARY_LIMIT),
    })
}

fn event_id(prefix: &str) -> String {
    format!("{prefix}_{}", Uuid::new_v4().simple())
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

const LIVE_INSTRUCTIONS: &str = "You are the spoken side of a blog writing copilot in a live, full-duplex conversation. Keep replies short and natural. When the user wants an edit, research, a link checked, or any change to the article, delegate to the backend and tell them you are taking care of it. After the backend returns a result, say briefly that it is done. Typed notes and links from the chat are part of the same conversation. Do not claim a change is finished until the backend says so.";
