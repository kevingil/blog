use std::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    sync::Arc,
};

use axum::{
    Json, Router,
    body::{Body, Bytes},
    extract::{
        Path, State,
        ws::{Message, WebSocket, WebSocketUpgrade},
    },
    http::{Response, header::CONTENT_TYPE},
    response::IntoResponse,
    routing::{get, post},
};
use serde_json::{Value, json};
use tokio::{net::TcpListener, sync::Mutex};

#[derive(Clone, Default)]
struct FixtureState {
    requests: Arc<Mutex<Vec<Value>>>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let state = FixtureState::default();
    let app = Router::new()
        .route("/v1/embeddings", post(embeddings))
        .route("/v1/responses", post(responses))
        .route("/v1/images/generations", post(images))
        .route("/v1/audio/transcriptions", post(transcriptions))
        .route("/v1/audio/speech", post(speech))
        .route("/v1/live/sessions", get(live_sessions))
        .route("/oauth/token", post(oauth_token))
        .route("/mcp/{server}", post(mcp_http))
        .route("/search", post(exa_search))
        .route("/findSimilar", post(exa_search))
        .route("/answer", post(exa_answer))
        .route("/fixture-image.svg", get(fixture_image))
        .route("/__fixture/requests", get(recorded_requests))
        .with_state(state);
    let address = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 8090);
    let listener = TcpListener::bind(address).await?;
    axum::serve(listener, app).await?;
    Ok(())
}

async fn embeddings(State(state): State<FixtureState>, Json(request): Json<Value>) -> Json<Value> {
    record(&state, "/v1/embeddings", &request).await;
    let dimensions = request
        .get("dimensions")
        .and_then(Value::as_u64)
        .and_then(|value| usize::try_from(value).ok())
        .unwrap_or(1536);
    Json(json!({
        "data": [{"index": 0, "embedding": vec![0.125_f32; dimensions]}],
        "model": request.get("model").cloned().unwrap_or(Value::String("fixture".to_owned())),
        "usage": {"prompt_tokens": 1, "total_tokens": 1}
    }))
}

async fn images(State(state): State<FixtureState>, Json(request): Json<Value>) -> Json<Value> {
    record(&state, "/v1/images/generations", &request).await;
    Json(json!({
        "created": 0,
        "data": [{
            "b64_json": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
        }]
    }))
}

async fn responses(
    State(state): State<FixtureState>,
    Json(request): Json<Value>,
) -> Response<Body> {
    record(&state, "/v1/responses", &request).await;
    let decision = fixture_response(&request);
    if request.get("stream").and_then(Value::as_bool) == Some(true) {
        return sse_response(&decision);
    }
    Json(completed_response(&decision)).into_response()
}

async fn transcriptions(State(state): State<FixtureState>, body: Bytes) -> Json<Value> {
    record(
        &state,
        "/v1/audio/transcriptions",
        &json!({"bytes": body.len()}),
    )
    .await;
    Json(json!({ "text": "Fixture voice note" }))
}

async fn live_sessions(upgrade: WebSocketUpgrade) -> impl IntoResponse {
    upgrade.on_upgrade(handle_live_session)
}

async fn handle_live_session(mut socket: WebSocket) {
    while let Some(Ok(message)) = socket.recv().await {
        let Message::Text(text) = message else {
            continue;
        };
        let Ok(event) = serde_json::from_str::<Value>(text.as_str()) else {
            continue;
        };
        let kind = event.get("type").and_then(Value::as_str).unwrap_or("");
        let reply = match kind {
            "session.start" => json!({
                "type": "session.started",
                "session": {"id": "live_fixture", "model": "gpt-live-1"}
            }),
            "session.input_audio.append" => {
                let _ = send_live(
                    &mut socket,
                    json!({
                        "type": "session.input_transcript.delta",
                        "delta": "tighten the intro"
                    }),
                )
                .await;
                json!({
                    "type": "session.delegation.created",
                    "delegation": {"id": "del_fixture", "target": "client", "offset_ms": 0}
                })
            }
            "session.commentary.append" => {
                let content = event.get("content").and_then(Value::as_str).unwrap_or("");
                let _ = send_live(
                    &mut socket,
                    json!({
                        "type": "session.output_transcript.delta",
                        "delta": content
                    }),
                )
                .await;
                json!({
                    "type": "session.output_audio.delta",
                    "delta": "AAA="
                })
            }
            "session.thinking.append" => json!({
                "type": "session.thinking.appended",
                "client_event_id": event.get("event_id").cloned().unwrap_or(Value::Null)
            }),
            _ => continue,
        };
        if send_live(&mut socket, reply).await.is_err() {
            break;
        }
    }
}

async fn send_live(socket: &mut WebSocket, event: Value) -> Result<(), axum::Error> {
    socket.send(Message::Text(event.to_string().into())).await
}

async fn speech(
    State(state): State<FixtureState>,
    Json(request): Json<Value>,
) -> impl IntoResponse {
    record(&state, "/v1/audio/speech", &request).await;
    (
        [(CONTENT_TYPE, "audio/wav")],
        blog_backend::core::speech::silent_wav(),
    )
}

async fn fixture_image() -> ([(&'static str, &'static str); 1], &'static str) {
    (
        [(CONTENT_TYPE.as_str(), "image/svg+xml")],
        r##"<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630"><rect width="100%" height="100%" fill="#1f2937"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="white" font-size="48">Blog image fixture</text></svg>"##,
    )
}

async fn exa_search(
    State(state): State<FixtureState>,
    uri: axum::http::Uri,
    Json(request): Json<Value>,
) -> Json<Value> {
    record(&state, uri.path(), &request).await;
    let query = request
        .get("query")
        .or_else(|| request.get("url"))
        .and_then(Value::as_str)
        .unwrap_or("fixture");
    Json(json!({
        "requestId": "exa_fixture",
        "resolvedSearchType": "neural",
        "costDollars": {"total": 0.0},
        "results": [{
            "id": "exa_fixture_result",
            "title": format!("Fixture result for {query}"),
            "url": "https://fixture.example.com/article",
            "score": 0.95,
            "text": "Deterministic fixture article content.",
            "highlights": ["Deterministic fixture highlight."],
            "summary": "Deterministic fixture summary.",
            "favicon": "https://fixture.example.com/favicon.ico"
        }]
    }))
}

async fn exa_answer(State(state): State<FixtureState>, Json(request): Json<Value>) -> Json<Value> {
    record(&state, "/answer", &request).await;
    let question = request
        .get("query")
        .and_then(Value::as_str)
        .unwrap_or("fixture");
    Json(json!({
        "answer": format!("Fixture answer for {question}"),
        "citations": [{
            "id": "exa_fixture_citation",
            "title": "Fixture citation",
            "url": "https://fixture.example.com/citation",
            "author": "Fixture Author",
            "publishedDate": "2026-07-27",
            "text": "Deterministic fixture citation content.",
            "favicon": "https://fixture.example.com/favicon.ico"
        }],
        "costDollars": {"total": 0.0}
    }))
}

async fn recorded_requests(State(state): State<FixtureState>) -> Json<Value> {
    Json(Value::Array(state.requests.lock().await.clone()))
}

async fn oauth_token() -> Json<Value> {
    Json(json!({
        "access_token": "fixture",
        "token_type": "Bearer",
        "expires_in": 3600
    }))
}

async fn mcp_http(Path(server): Path<String>, Json(body): Json<Value>) -> Json<Value> {
    if body.get("id").is_none() {
        return Json(json!({}));
    }
    let id = body.get("id").cloned().unwrap_or(Value::Null);
    let method = body.get("method").and_then(Value::as_str).unwrap_or("");
    let result = match method {
        "initialize" => json!({
            "protocolVersion": "2024-11-05",
            "capabilities": {"tools": {}},
            "serverInfo": {"name": server, "version": "fixture"}
        }),
        "tools/list" => json!({ "tools": fixture_tools(&server) }),
        "tools/call" => json!({
            "content": [{
                "type": "text",
                "text": fixture_tool_text(&server, body.get("params"))
            }]
        }),
        _ => json!({}),
    };
    Json(json!({
        "jsonrpc": "2.0",
        "id": id,
        "result": result
    }))
}

fn fixture_tools(server: &str) -> Value {
    match server {
        "notion" => json!([{
            "name": "search",
            "description": "Search pages in the connected Notion workspace",
            "inputSchema": {
                "type": "object",
                "properties": {"query": {"type": "string"}},
                "required": ["query"]
            }
        }]),
        "granola" => json!([{
            "name": "list_meetings",
            "description": "List recent Granola meeting notes",
            "inputSchema": {"type": "object", "properties": {}}
        }]),
        "fireflies" => json!([{
            "name": "list_transcripts",
            "description": "List Fireflies meeting transcripts",
            "inputSchema": {"type": "object", "properties": {}}
        }]),
        _ => json!([{
            "name": "search",
            "description": "Search this connected MCP server",
            "inputSchema": {
                "type": "object",
                "properties": {"query": {"type": "string"}}
            }
        }]),
    }
}

fn fixture_tool_text(server: &str, params: Option<&Value>) -> String {
    let tool = params
        .and_then(|value| value.get("name"))
        .and_then(Value::as_str)
        .unwrap_or("tool");
    match server {
        "notion" => format!("Notion search result for {tool}: product notes and meeting pages."),
        "granola" => format!("Granola meeting notes from {tool}: weekly planning and decisions."),
        "fireflies" => {
            format!("Fireflies transcript from {tool}: discussion summary and action items.")
        }
        _ => format!("{server} returned context from {tool}."),
    }
}

async fn record(state: &FixtureState, path: &str, request: &Value) {
    state.requests.lock().await.push(json!({
        "path": path,
        "body": request,
    }));
}

fn response_input_text(request: &Value) -> String {
    if let Some(input) = request.get("input").and_then(Value::as_str) {
        return input.to_owned();
    }
    request
        .get("input")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
        .filter(|item| item.get("role").and_then(Value::as_str) == Some("user"))
        .flat_map(|item| {
            item.get("content")
                .and_then(Value::as_array)
                .into_iter()
                .flatten()
        })
        .filter(|part| part.get("type").and_then(Value::as_str) == Some("input_text"))
        .filter_map(|part| part.get("text").and_then(Value::as_str))
        .collect::<Vec<_>>()
        .join("\n")
}

struct FixtureDecision {
    text: String,
    calls: Vec<FixtureCall>,
}

struct FixtureCall {
    id: &'static str,
    name: &'static str,
    arguments: String,
}

fn fixture_response(request: &Value) -> FixtureDecision {
    if matches!(
        request.get("model").and_then(Value::as_str),
        Some("openai/gpt-oss-120b") | Some("gpt-6-luna")
    ) {
        return FixtureDecision {
            text: json!({
                "title": "Fixture insight",
                "summary": "A deterministic fixture summary.",
                "content": "Deterministic fixture insight content grounded in the supplied articles.",
                "key_points": [
                    "First fixture takeaway",
                    "Second fixture takeaway",
                    "Third fixture takeaway"
                ]
            })
            .to_string(),
            calls: Vec::new(),
        };
    }
    let input = response_input_text(request);
    let user_text = latest_user_text(request);
    let request_text = chat_request_text(&user_text);
    let calls = if tools_already_ran(request) {
        Vec::new()
    } else {
        plan_editor_tools(request_text)
    };
    let text = if !calls.is_empty() {
        String::new()
    } else if tools_already_ran(request) && editor_edit_requested(request_text) {
        "Updated the article. The title and sources stay in their own fields, outside the body."
            .to_owned()
    } else {
        format!("Fixture response: {input}")
    };
    FixtureDecision { text, calls }
}

fn latest_user_text(request: &Value) -> String {
    if let Some(input) = request.get("input").and_then(Value::as_str) {
        return input.to_owned();
    }
    request
        .get("input")
        .and_then(Value::as_array)
        .and_then(|items| {
            items.iter().rev().find_map(|item| {
                (item.get("role").and_then(Value::as_str) == Some("user")).then(|| {
                    item.get("content")
                        .and_then(Value::as_array)
                        .into_iter()
                        .flatten()
                        .filter(|part| {
                            part.get("type").and_then(Value::as_str) == Some("input_text")
                        })
                        .filter_map(|part| part.get("text").and_then(Value::as_str))
                        .collect::<Vec<_>>()
                        .join("\n")
                })
            })
        })
        .unwrap_or_default()
}

fn chat_request_text(user_text: &str) -> &str {
    user_text
        .split("--- Document Context ---")
        .next()
        .unwrap_or(user_text)
        .trim()
}

fn tools_already_ran(request: &Value) -> bool {
    let Some(items) = request.get("input").and_then(Value::as_array) else {
        return false;
    };
    let Some(last_user) = items
        .iter()
        .rposition(|item| item.get("role").and_then(Value::as_str) == Some("user"))
    else {
        return false;
    };
    items[last_user + 1..]
        .iter()
        .any(|item| item.get("type").and_then(Value::as_str) == Some("function_call_output"))
}

fn editor_edit_requested(text: &str) -> bool {
    requested_title(text).is_some() || !requested_sources(text).is_empty()
}

fn plan_editor_tools(text: &str) -> Vec<FixtureCall> {
    let mut calls = Vec::new();
    if let Some(title) = requested_title(text) {
        calls.push(FixtureCall {
            id: "call_title",
            name: "set_title",
            arguments:
                json!({"title": title, "reason": "The title is stored separately from the body."})
                    .to_string(),
        });
    }
    let sources = requested_sources(text);
    if !sources.is_empty() {
        let sources = sources
            .into_iter()
            .map(|(title, url)| json!({"id": null, "title": title, "url": url, "note": "Added from the editor request."}))
            .collect::<Vec<_>>();
        calls.push(FixtureCall {
            id: "call_sources",
            name: "update_sources",
            arguments: json!({"sources": sources, "remove_ids": []}).to_string(),
        });
    }
    calls
}

fn requested_title(text: &str) -> Option<String> {
    let lower = text.to_ascii_lowercase();
    let marker = [
        "title to ",
        "rename the article to ",
        "rename it to ",
        "set title ",
    ]
    .iter()
    .find_map(|marker| lower.find(marker).map(|index| (index, marker.len())))?;
    let rest = text[marker.0 + marker.1..].trim_start();
    let title = if let Some(stripped) = rest.strip_prefix('"') {
        stripped.split('"').next().unwrap_or("").trim()
    } else if let Some(stripped) = rest.strip_prefix('\'') {
        stripped.split('\'').next().unwrap_or("").trim()
    } else {
        rest.split(" and ").next().unwrap_or(rest).trim()
    };
    let title = title
        .trim_matches(|character: char| matches!(character, '"' | '\'' | '.'))
        .trim();
    (!title.is_empty() && title.len() <= 180).then(|| title.to_owned())
}

fn requested_sources(text: &str) -> Vec<(String, String)> {
    let lower = text.to_ascii_lowercase();
    if !lower.contains("source") && !lower.contains("citation") && !lower.contains("reference") {
        return Vec::new();
    }
    let mut sources = Vec::new();
    let mut index = 0;
    while index < text.len() {
        let rest = &text[index..];
        let Some(relative) = rest.find("https://").or_else(|| rest.find("http://")) else {
            break;
        };
        let start = index + relative;
        let end = text[start..]
            .find(|character: char| {
                character.is_whitespace() || matches!(character, '"' | '\'' | ')' | ',' | '>' | ']')
            })
            .map(|offset| start + offset)
            .unwrap_or(text.len());
        let url = text[start..end].trim_end_matches('.').to_owned();
        if !url.is_empty() {
            let title = quoted_before(&text[..start]).unwrap_or_else(|| host_title(&url));
            sources.push((title, url));
        }
        index = end.max(start + 1);
    }
    if sources.is_empty()
        && let Some(title) = quoted_after_source_label(text)
    {
        sources.push((title, String::new()));
    }
    sources
}

fn quoted_before(text: &str) -> Option<String> {
    let window = text
        .chars()
        .rev()
        .take(160)
        .collect::<Vec<_>>()
        .into_iter()
        .rev()
        .collect::<String>();
    let (before_last, _) = window.rsplit_once('"')?;
    let (_, title) = before_last.rsplit_once('"')?;
    let title = title.trim();
    (!title.is_empty()).then(|| title.to_owned())
}

fn quoted_after_source_label(text: &str) -> Option<String> {
    let lower = text.to_ascii_lowercase();
    let start = ["source titled ", "source called ", "source \""]
        .iter()
        .find_map(|marker| lower.find(marker).map(|index| index + marker.len()))?;
    let rest = text[start..].trim_start_matches('"');
    let title = rest.split(['"', '\n']).next().unwrap_or("").trim();
    (!title.is_empty() && title.len() <= 180).then(|| title.trim_matches('"').to_owned())
}

fn host_title(url: &str) -> String {
    url.trim_start_matches("https://")
        .trim_start_matches("http://")
        .split('/')
        .next()
        .unwrap_or(url)
        .to_owned()
}

fn completed_response(decision: &FixtureDecision) -> Value {
    let output = if decision.calls.is_empty() {
        json!([{
            "type": "message",
            "id": "msg_fixture",
            "role": "assistant",
            "status": "completed",
            "content": [{"type": "output_text", "text": decision.text, "annotations": []}]
        }])
    } else {
        Value::Array(decision.calls.iter().map(function_call_item).collect())
    };
    json!({
        "id": "resp_fixture",
        "object": "response",
        "status": "completed",
        "output": output,
        "usage": {
            "input_tokens": 4,
            "output_tokens": 4,
            "input_tokens_details": {"cached_tokens": 0}
        }
    })
}

fn function_call_item(call: &FixtureCall) -> Value {
    json!({
        "type": "function_call",
        "id": format!("fc_{}", call.id),
        "call_id": call.id,
        "name": call.name,
        "arguments": call.arguments,
        "status": "completed",
    })
}

fn sse_response(decision: &FixtureDecision) -> Response<Body> {
    let mut events = Vec::new();
    if decision.calls.is_empty() {
        events.push(json!({"type": "response.output_text.delta", "delta": decision.text}));
    } else {
        for call in &decision.calls {
            let item = function_call_item(call);
            events.push(json!({"type": "response.output_item.added", "item": item}));
            events.push(json!({
                "type": "response.function_call_arguments.done",
                "item_id": format!("fc_{}", call.id),
                "arguments": call.arguments,
            }));
        }
    }
    events.push(json!({"type": "response.completed", "response": completed_response(decision)}));
    let mut stream = events
        .into_iter()
        .map(|event| format!("data: {event}"))
        .collect::<Vec<_>>()
        .join("\n\n");
    stream.push_str("\n\ndata: [DONE]\n\n");
    Response::builder()
        .header(CONTENT_TYPE, "text/event-stream")
        .body(Body::from(stream))
        .unwrap_or_else(|_| Response::new(Body::empty()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn plans_title_and_sources_from_the_chat_request() {
        let request = json!({
            "input": [{
                "type": "message",
                "role": "user",
                "content": [{
                    "type": "input_text",
                    "text": "Set the title to \"RSI Is a Systems Problem\" and add a source \"GraalVM\" https://www.graalvm.org\n\n--- Document Context ---\nTitle: Old\nTitle is a separate field. Edit it with set_title."
                }]
            }]
        });
        let decision = fixture_response(&request);
        assert_eq!(decision.calls.len(), 2);
        assert_eq!(decision.calls[0].name, "set_title");
        assert!(
            decision.calls[0]
                .arguments
                .contains("RSI Is a Systems Problem")
        );
        assert_eq!(decision.calls[1].name, "update_sources");
        assert!(
            decision.calls[1]
                .arguments
                .contains("https://www.graalvm.org")
        );
        assert!(decision.calls[1].arguments.contains("GraalVM"));
    }

    #[test]
    fn document_context_does_not_trigger_editor_tools() {
        let request = json!({
            "input": "Tighten the intro\n\n--- Document Context ---\nTitle: Old\nTitle is a separate field. Edit it with set_title. Sources are separate objects."
        });
        let decision = fixture_response(&request);
        assert!(decision.calls.is_empty());
        assert!(decision.text.starts_with("Fixture response:"));
    }

    #[test]
    fn confirms_after_the_editor_tools_have_run() {
        let request = json!({
            "input": [
                {
                    "type": "message",
                    "role": "user",
                    "content": [{
                        "type": "input_text",
                        "text": "Set the title to \"RSI Is a Systems Problem\"\n\n--- Document Context ---\nTitle: Old"
                    }]
                },
                {"type": "function_call_output", "call_id": "call_title", "output": "{}"}
            ]
        });
        let decision = fixture_response(&request);
        assert!(decision.calls.is_empty());
        assert!(decision.text.contains("outside the body"));
    }

    #[test]
    fn a_later_request_still_edits_after_an_earlier_tool_turn() {
        let request = json!({
            "input": [
                {
                    "type": "message",
                    "role": "user",
                    "content": [{"type": "input_text", "text": "Set the title to \"First\"\n\n--- Document Context ---\nTitle: Old"}]
                },
                {"type": "function_call_output", "call_id": "call_title", "output": "{}"},
                {
                    "type": "message",
                    "role": "user",
                    "content": [{"type": "input_text", "text": "Set the title to \"Second Title\" and add a source \"OpenJDK\" https://openjdk.org\n\n--- Document Context ---\nTitle: First"}]
                }
            ]
        });
        let decision = fixture_response(&request);
        assert_eq!(decision.calls.len(), 2);
        assert!(decision.calls[0].arguments.contains("Second Title"));
        assert!(decision.calls[1].arguments.contains("https://openjdk.org"));
    }

    #[test]
    fn insight_models_return_a_briefing() {
        for model in ["openai/gpt-oss-120b", "gpt-6-luna"] {
            let decision = fixture_response(&json!({"model": model, "input": "brief the tracker"}));
            assert!(decision.calls.is_empty(), "{model}");
            assert!(decision.text.contains("Fixture insight"), "{model}");
            assert!(decision.text.contains("First fixture takeaway"), "{model}");
        }
    }
}
