use std::{sync::Arc, time::Duration};

use async_trait::async_trait;
use blog_backend::core::live::{
    LiveClientCommand, LiveConnection, LiveHarness, LiveTurn, LiveTurnHandle, live_websocket_url,
    run_live_session,
};
use blog_backend::error::AppError;
use serde_json::{Value, json};
use tokio::sync::{Mutex, mpsc};

#[test]
fn live_websocket_url_maps_http_schemes() {
    assert_eq!(
        live_websocket_url("https://api.openai.com/v1"),
        "wss://api.openai.com/v1/live/sessions"
    );
    assert_eq!(
        live_websocket_url("http://external-fixtures:8090/v1"),
        "ws://external-fixtures:8090/v1/live/sessions"
    );
}

struct ScriptedConnection {
    inbound: mpsc::Receiver<Value>,
    outbound: mpsc::Sender<Value>,
}

#[async_trait]
impl LiveConnection for ScriptedConnection {
    async fn send_event(&mut self, event: Value) -> Result<(), AppError> {
        self.outbound
            .send(event)
            .await
            .map_err(|_| AppError::External)
    }

    async fn recv_event(&mut self) -> Result<Option<Value>, AppError> {
        Ok(self.inbound.recv().await)
    }
}

#[derive(Default)]
struct RecordingHarness {
    messages: Mutex<Vec<String>>,
}

#[async_trait]
impl LiveHarness for RecordingHarness {
    async fn run_turn(&self, turn: LiveTurn) -> Result<LiveTurnHandle, AppError> {
        self.messages.lock().await.push(turn.message);
        let (sender, events) = mpsc::channel(4);
        sender
            .send(json!({"type": "text", "content": "Intro tightened."}))
            .await
            .map_err(|_| AppError::External)?;
        Ok(LiveTurnHandle {
            request_id: "req-live".to_owned(),
            events,
        })
    }
}

async fn next_event(receiver: &mut mpsc::Receiver<Value>) -> Value {
    tokio::time::timeout(Duration::from_secs(2), receiver.recv())
        .await
        .ok()
        .flatten()
        .unwrap_or(Value::Null)
}

async fn next_matching(receiver: &mut mpsc::Receiver<Value>, event_type: &str) -> Value {
    for _ in 0..8 {
        let event = next_event(receiver).await;
        if event.get("type").and_then(Value::as_str) == Some(event_type) {
            return event;
        }
    }
    Value::Null
}

#[tokio::test]
async fn delegation_runs_the_harness_and_speaks_through_gpt_live() {
    let (inbound_tx, inbound_rx) = mpsc::channel(8);
    let (outbound_tx, mut outbound_rx) = mpsc::channel(8);
    let harness = Arc::new(RecordingHarness::default());
    let harness_for_session = harness.clone();
    let (command_tx, command_rx) = mpsc::channel(8);
    let (browser_tx, mut browser_rx) = mpsc::channel(8);
    let session = tokio::spawn(async move {
        run_live_session(
            Box::new(ScriptedConnection {
                inbound: inbound_rx,
                outbound: outbound_tx,
            }),
            command_rx,
            browser_tx,
            harness_for_session,
        )
        .await
    });

    command_tx
        .send(LiveClientCommand::Start {
            article_id: "article-1".to_owned(),
            document_content: String::new(),
            document_markdown: String::new(),
        })
        .await
        .ok();
    let start = next_matching(&mut outbound_rx, "session.start").await;
    assert_eq!(start["session"]["model"], "gpt-live-1");
    assert_eq!(start["session"]["delegation"]["type"], "client");

    inbound_tx
        .send(json!({"type": "session.started", "session": {"id": "s1"}}))
        .await
        .ok();
    let started = next_matching(&mut browser_rx, "started").await;
    assert_eq!(started["model"], "gpt-live-1");

    command_tx
        .send(LiveClientCommand::Audio {
            pcm_base64: "AAAA".to_owned(),
        })
        .await
        .ok();
    let audio = next_matching(&mut outbound_rx, "session.input_audio.append").await;
    assert_eq!(audio["audio"], "AAAA");

    inbound_tx
        .send(json!({
            "type": "session.input_transcript.delta",
            "delta": "tighten the intro"
        }))
        .await
        .ok();
    inbound_tx
        .send(json!({
            "type": "session.delegation.created",
            "delegation": {"id": "del_1", "target": "client"}
        }))
        .await
        .ok();

    let commentary = next_matching(&mut outbound_rx, "session.commentary.append").await;
    assert_eq!(commentary["content"], "Let me take care of that.");
    assert_eq!(commentary["delegation_id"], "del_1");
    let delegation = next_matching(&mut browser_rx, "delegation").await;
    assert_eq!(delegation["requestId"], "req-live");
    assert_eq!(delegation["message"], "tighten the intro");

    let summary = next_matching(&mut outbound_rx, "session.commentary.append").await;
    assert_eq!(summary["content"], "Intro tightened.");
    let messages = harness.messages.lock().await.clone();
    assert_eq!(messages, vec!["tighten the intro".to_owned()]);

    command_tx.send(LiveClientCommand::Stop).await.ok();
    let _ = tokio::time::timeout(Duration::from_secs(2), session).await;
}

#[tokio::test]
async fn typed_link_is_included_in_the_same_live_turn() {
    let (inbound_tx, inbound_rx) = mpsc::channel(8);
    let (outbound_tx, mut outbound_rx) = mpsc::channel(8);
    let harness = Arc::new(RecordingHarness::default());
    let harness_for_session = harness.clone();
    let (command_tx, command_rx) = mpsc::channel(8);
    let (browser_tx, mut browser_rx) = mpsc::channel(8);
    tokio::spawn(async move {
        let _ = run_live_session(
            Box::new(ScriptedConnection {
                inbound: inbound_rx,
                outbound: outbound_tx,
            }),
            command_rx,
            browser_tx,
            harness_for_session,
        )
        .await;
    });

    command_tx
        .send(LiveClientCommand::Start {
            article_id: "article-1".to_owned(),
            document_content: String::new(),
            document_markdown: String::new(),
        })
        .await
        .ok();
    let _ = next_matching(&mut outbound_rx, "session.start").await;
    inbound_tx
        .send(json!({"type": "session.started", "session": {"id": "s1"}}))
        .await
        .ok();
    let _ = next_matching(&mut browser_rx, "started").await;
    inbound_tx
        .send(json!({"type": "session.input_transcript.delta", "delta": "check this "}))
        .await
        .ok();
    let _ = next_matching(&mut browser_rx, "transcript").await;
    command_tx
        .send(LiveClientCommand::Text {
            text: "https://example.com/notes".to_owned(),
        })
        .await
        .ok();

    let thinking = next_matching(&mut outbound_rx, "session.thinking.append").await;
    let thinking_text = thinking["content"].as_str().unwrap_or("");
    assert!(thinking_text.contains("https://example.com/notes"));
    let commentary = next_matching(&mut outbound_rx, "session.commentary.append").await;
    assert_eq!(commentary["content"], "Let me take care of that.");
    let delegation = next_matching(&mut browser_rx, "delegation").await;
    assert_eq!(
        delegation["message"],
        "check this\n\nhttps://example.com/notes"
    );
    let messages = harness.messages.lock().await.clone();
    assert_eq!(
        messages,
        vec!["check this\n\nhttps://example.com/notes".to_owned()]
    );
}
