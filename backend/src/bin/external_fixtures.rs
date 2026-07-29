use std::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    sync::Arc,
};

use axum::{
    Json, Router,
    body::Body,
    extract::State,
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
    let input = response_input_text(&request);
    let output_text = if request.get("model").and_then(Value::as_str) == Some("openai/gpt-oss-120b")
    {
        json!({
            "title": "Fixture insight",
            "summary": "A deterministic fixture summary.",
            "content": "Deterministic fixture insight content grounded in the supplied articles.",
            "key_points": [
                "First fixture takeaway",
                "Second fixture takeaway",
                "Third fixture takeaway"
            ]
        })
        .to_string()
    } else {
        format!("Fixture response: {input}")
    };
    if request.get("stream").and_then(Value::as_bool) == Some(true) {
        let completed = json!({
            "id": "resp_fixture",
            "object": "response",
            "status": "completed",
            "output": [{
                "type": "message",
                "id": "msg_fixture",
                "role": "assistant",
                "status": "completed",
                "content": [{"type": "output_text", "text": output_text, "annotations": []}]
            }],
            "usage": {
                "input_tokens": 4,
                "output_tokens": 4,
                "input_tokens_details": {"cached_tokens": 0}
            }
        });
        let stream = [
            format!(
                "data: {}",
                json!({"type": "response.output_text.delta", "delta": output_text})
            ),
            format!(
                "data: {}",
                json!({"type": "response.completed", "response": completed})
            ),
            "data: [DONE]".to_owned(),
        ]
        .join("\n\n");
        return Response::builder()
            .header(CONTENT_TYPE, "text/event-stream")
            .body(Body::from(format!("{stream}\n\n")))
            .unwrap_or_else(|_| Response::new(Body::empty()));
    }
    Json(json!({
        "id": "resp_fixture",
        "status": "completed",
        "output": [{
            "type": "message",
            "content": [{
                "type": "output_text",
                "text": output_text
            }]
        }],
        "usage": {"input_tokens": 4, "output_tokens": 4}
    }))
    .into_response()
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
