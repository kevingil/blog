use axum::{
    extract::{State, WebSocketUpgrade},
    response::Response,
};
use futures_util::{SinkExt, StreamExt};
use serde_json::{Value, json};
use tokio::sync::mpsc;

use crate::core::live::{LiveClientCommand, LivePorts};

#[utoipa::path(
    get,
    path = "/agent/live",
    operation_id = "connectLiveSession",
    tag = "agent",
    responses(
        (status = 101, description = "GPT-Live session switching protocols"),
        (status = 400, description = "Invalid WebSocket upgrade headers"),
        (status = 426, description = "Connection cannot be upgraded")
    )
)]
pub async fn live_session(
    State(state): State<super::state::AgentState>,
    upgrade: WebSocketUpgrade,
) -> Response {
    let ports = state.live();
    upgrade.on_upgrade(move |socket| serve_live_session(socket, ports))
}

async fn serve_live_session(socket: axum::extract::ws::WebSocket, ports: LivePorts) {
    use axum::extract::ws::Message;

    let (mut writer, mut reader) = socket.split();
    let (command_tx, command_rx) = mpsc::channel(32);
    let (event_tx, mut event_rx) = mpsc::channel(32);
    let inbound = tokio::spawn(async move {
        while let Some(message) = reader.next().await {
            match message {
                Ok(Message::Text(text)) => {
                    let Some(command) = parse_command(text.as_str()) else {
                        continue;
                    };
                    let stop = matches!(command, LiveClientCommand::Stop);
                    if command_tx.send(command).await.is_err() {
                        break;
                    }
                    if stop {
                        break;
                    }
                }
                Ok(Message::Close(_)) | Err(_) => break,
                Ok(_) => {}
            }
        }
    });
    let outbound = tokio::spawn(async move {
        while let Some(event) = event_rx.recv().await {
            let Ok(text) = serde_json::to_string(&event) else {
                continue;
            };
            if writer.send(Message::Text(text.into())).await.is_err() {
                break;
            }
        }
    });

    let upstream = match ports.upstream().connect().await {
        Ok(upstream) => upstream,
        Err(_) => {
            let _ = event_tx
                .send(json!({
                    "type": "error",
                    "error": "Could not open the GPT-Live session"
                }))
                .await;
            drop(event_tx);
            inbound.abort();
            let _ = outbound.await;
            return;
        }
    };
    let _ =
        crate::core::live::run_live_session(upstream, command_rx, event_tx, ports.harness()).await;
    inbound.abort();
    let _ = outbound.await;
}

fn parse_command(text: &str) -> Option<LiveClientCommand> {
    let value: Value = serde_json::from_str(text).ok()?;
    let kind = value.get("type").and_then(Value::as_str)?;
    let field = |name: &str| {
        value
            .get(name)
            .and_then(Value::as_str)
            .unwrap_or("")
            .to_owned()
    };
    match kind {
        "start" => Some(LiveClientCommand::Start {
            article_id: field("articleId"),
            document_content: field("documentContent"),
            document_markdown: field("documentMarkdown"),
        }),
        "audio" => Some(LiveClientCommand::Audio {
            pcm_base64: field("audio"),
        }),
        "text" => Some(LiveClientCommand::Text {
            text: field("text"),
        }),
        "context" => Some(LiveClientCommand::Context {
            document_content: field("documentContent"),
            document_markdown: field("documentMarkdown"),
        }),
        "stop" => Some(LiveClientCommand::Stop),
        _ => None,
    }
}
