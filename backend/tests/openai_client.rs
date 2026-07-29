use std::{error::Error, sync::Arc};

use axum::{
    Json, Router,
    body::Body,
    extract::State,
    http::{HeaderMap, Response, header},
    response::IntoResponse,
    routing::post,
};
use blog_backend::{
    core::ml::llm::{
        ContentPart, FinishReason, LlmMessage, MessageRole, Provider, ProviderEventType,
        ReadDocumentTool, TextContent, Tool, ToolCall, ToolResult,
    },
    error::AppError,
    integrations::openai::{GeneratedImage, OpenAiClient},
};
use serde_json::{Value, json};
use tokio::{net::TcpListener, sync::Mutex};

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone, Default)]
struct Received {
    request: Arc<Mutex<Option<(HeaderMap, Value)>>>,
    responses: Arc<Mutex<Vec<Value>>>,
}

async fn embedding_fixture(
    State(received): State<Received>,
    headers: HeaderMap,
    Json(body): Json<Value>,
) -> Json<Value> {
    *received.request.lock().await = Some((headers, body));
    Json(json!({
        "data": [{"index": 0, "embedding": vec![0.25_f32; 1536]}],
        "model": "text-embedding-3-small",
        "usage": {"prompt_tokens": 1, "total_tokens": 1}
    }))
}

async fn response_fixture(
    State(received): State<Received>,
    Json(body): Json<Value>,
) -> Response<Body> {
    received.responses.lock().await.push(body.clone());
    if body["stream"] == true {
        let stream = [
            r#"data: {"type":"response.output_text.delta","delta":"draft "}"#,
            r#"data: {"type":"response.output_item.added","item":{"type":"function_call","id":"fc_1","call_id":"call_2","name":"read_document","arguments":""}}"#,
            r#"data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{}"}"#,
            r#"data: {"type":"response.function_call_arguments.done","item_id":"fc_1","arguments":"{}"}"#,
            r#"data: {"type":"response.completed","response":{"status":"completed","output":[],"usage":{"input_tokens":11,"output_tokens":7,"input_tokens_details":{"cached_tokens":3}}}}"#,
            "data: [DONE]",
        ]
        .join("\n\n");
        return Response::builder()
            .header(header::CONTENT_TYPE, "text/event-stream")
            .body(Body::from(format!("{stream}\n\n")))
            .unwrap_or_else(|_| Response::new(Body::empty()));
    }
    Json(json!({
        "output": [{
            "type": "message",
            "content": [{
                "type": "output_text",
                "text": format!("response:{}", body["input"].as_str().unwrap_or_default())
            }]
        }]
    }))
    .into_response()
}

async fn image_fixture() -> Json<Value> {
    Json(json!({"data": [{"b64_json": "aW1hZ2UtYnl0ZXM="}]}))
}

#[tokio::test]
async fn openai_provider_stream_preserves_structured_tool_history_and_usage() -> TestResult {
    let received = Received::default();
    let app = Router::new()
        .route("/v1/responses", post(response_fixture))
        .with_state(received.clone());
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move { axum::serve(listener, app).await });
    let client = OpenAiClient::with_base_url("test-key", format!("http://{address}/v1"))?;
    let session = "session";
    let messages = vec![
        LlmMessage::new(
            session,
            MessageRole::User,
            vec![ContentPart::Text(TextContent {
                text: "Use the tool".to_owned(),
            })],
            "",
        ),
        LlmMessage::new(
            session,
            MessageRole::Assistant,
            vec![ContentPart::ToolCall(ToolCall {
                id: "call_1".to_owned(),
                name: "read_document".to_owned(),
                input: "{}".to_owned(),
                r#type: "function".to_owned(),
                finished: true,
                thought_signature: Vec::new(),
            })],
            "gpt-5.2",
        ),
        LlmMessage::new(
            session,
            MessageRole::Tool,
            vec![ContentPart::ToolResult(ToolResult {
                tool_call_id: "call_1".to_owned(),
                content: "document".to_owned(),
                metadata: String::new(),
                is_error: false,
            })],
            "",
        ),
    ];
    let tools: Vec<Arc<dyn Tool>> = vec![Arc::new(ReadDocumentTool)];
    let mut events = client
        .stream_response(tokio_util::sync::CancellationToken::new(), messages, tools)
        .await?;
    let mut content = String::new();
    let mut completed = None;
    while let Some(event) = events.recv().await {
        if event.event_type == ProviderEventType::ContentDelta {
            content.push_str(&event.content);
        }
        if event.event_type == ProviderEventType::Complete {
            completed = event.response;
        }
    }
    let completed = completed.ok_or("missing completion event")?;
    assert_eq!(content, "draft ");
    assert_eq!(completed.content, "draft ");
    assert_eq!(completed.finish_reason, FinishReason::ToolUse);
    assert_eq!(completed.tool_calls.len(), 1);
    assert_eq!(completed.tool_calls[0].id, "call_2");
    assert_eq!(completed.tool_calls[0].input, "{}");
    assert_eq!(completed.usage.input_tokens, 8);
    assert_eq!(completed.usage.cache_read_tokens, 3);
    assert_eq!(completed.usage.output_tokens, 7);

    let requests = received.responses.lock().await;
    let request = requests.last().ok_or("fixture did not receive request")?;
    assert_eq!(request["stream"], true);
    assert_eq!(request["store"], false);
    assert_eq!(request["model"], "gpt-5.4-mini-2026-03-17");
    assert_eq!(request["max_output_tokens"], 16_384);
    assert_eq!(request["reasoning"]["effort"], "medium");
    assert!(
        request["input"]
            .as_array()
            .is_some_and(|items| items.iter().any(|item| {
                item["type"] == "function_call_output"
                    && item["call_id"] == "call_1"
                    && item["output"] == "document"
            }))
    );
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn openai_embedding_adapter_preserves_request_and_vector_contract() -> TestResult {
    let received = Received::default();
    let app = Router::new()
        .route("/v1/embeddings", post(embedding_fixture))
        .route("/v1/responses", post(response_fixture))
        .route("/v1/images/generations", post(image_fixture))
        .with_state(received.clone());
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move { axum::serve(listener, app).await });
    let client = OpenAiClient::with_base_url("test-key", format!("http://{address}/v1"))?;

    let embedding = client.generate_embedding("contract input").await?;
    assert_eq!(embedding, vec![0.25; 1536]);

    let request = received
        .request
        .lock()
        .await
        .take()
        .ok_or("fixture did not receive request")?;
    assert_eq!(request.0["authorization"], "Bearer test-key");
    assert_eq!(request.1["model"], "text-embedding-3-small");
    assert_eq!(request.1["input"], "contract input");
    assert_eq!(request.1["dimensions"], 1536);
    assert_eq!(
        client
            .generate_text("Respond precisely.", "fixture input")
            .await?,
        "response:fixture input"
    );
    assert_eq!(
        client.generate_image("fixture image").await?,
        GeneratedImage::Bytes(b"image-bytes".to_vec())
    );

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn openai_embedding_adapter_rejects_invalid_configuration_and_input() -> TestResult {
    let unconfigured = OpenAiClient::new("")?;
    assert!(!unconfigured.is_configured());
    assert!(matches!(
        unconfigured.generate_embedding("input").await,
        Err(AppError::External)
    ));
    assert!(matches!(
        OpenAiClient::with_base_url("key", ""),
        Err(AppError::InvalidInput(_))
    ));
    let client = OpenAiClient::new("test-key")?;
    assert!(matches!(
        client.generate_embedding("  ").await,
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}
