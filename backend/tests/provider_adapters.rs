use std::{collections::BTreeMap, error::Error, sync::Arc};

use async_trait::async_trait;
use axum::{
    Json, Router,
    body::Body,
    extract::{OriginalUri, State},
    http::{HeaderMap, Response, header},
};
use blog_backend::{
    core::ml::llm::{
        ContentPart, FinishReason, LlmMessage, MessageRole, Model, ModelProvider, Provider,
        ProviderEvent, ProviderEventType, ProviderResponse, TextContent, Tool, ToolCall,
        ToolCallRequest, ToolContext, ToolInfo, ToolResponse, ToolResult,
    },
    error::AppError,
    integrations::llm::{AnthropicClient, GeminiClient, GroqClient, VertexAiClient},
};
use serde_json::{Value, json};
use tokio::{
    net::TcpListener,
    sync::{Mutex, mpsc},
};
use tokio_util::sync::CancellationToken;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Debug)]
struct CapturedRequest {
    uri: String,
    headers: HeaderMap,
    body: Value,
}

#[derive(Clone)]
struct FixtureState {
    requests: Arc<Mutex<Vec<CapturedRequest>>>,
    content_type: Arc<String>,
    response_body: Arc<String>,
}

async fn provider_fixture(
    State(state): State<FixtureState>,
    OriginalUri(uri): OriginalUri,
    headers: HeaderMap,
    Json(body): Json<Value>,
) -> Response<Body> {
    state.requests.lock().await.push(CapturedRequest {
        uri: uri.to_string(),
        headers,
        body,
    });
    Response::builder()
        .header(header::CONTENT_TYPE, state.content_type.as_str())
        .body(Body::from(state.response_body.as_str().to_owned()))
        .unwrap_or_else(|_| Response::new(Body::empty()))
}

async fn spawn_fixture(
    records: &[&str],
) -> TestResult<(
    String,
    Arc<Mutex<Vec<CapturedRequest>>>,
    tokio::task::JoinHandle<Result<(), std::io::Error>>,
)> {
    spawn_response_fixture("text/event-stream", format!("{}\n\n", records.join("\n\n"))).await
}

async fn spawn_json_fixture(
    response: Value,
) -> TestResult<(
    String,
    Arc<Mutex<Vec<CapturedRequest>>>,
    tokio::task::JoinHandle<Result<(), std::io::Error>>,
)> {
    spawn_response_fixture("application/json", response.to_string()).await
}

async fn spawn_response_fixture(
    content_type: &str,
    response_body: String,
) -> TestResult<(
    String,
    Arc<Mutex<Vec<CapturedRequest>>>,
    tokio::task::JoinHandle<Result<(), std::io::Error>>,
)> {
    let requests = Arc::new(Mutex::new(Vec::new()));
    let app = Router::new()
        .fallback(provider_fixture)
        .with_state(FixtureState {
            requests: requests.clone(),
            content_type: Arc::new(content_type.to_owned()),
            response_body: Arc::new(response_body),
        });
    let listener = TcpListener::bind("127.0.0.1:0").await?;
    let address = listener.local_addr()?;
    let server = tokio::spawn(async move { axum::serve(listener, app).await });
    Ok((format!("http://{address}"), requests, server))
}

fn model(provider: &str, id: &str, can_reason: bool) -> Model {
    Model::new(id, provider, id, 4_096, can_reason, true)
}

fn messages() -> Vec<LlmMessage> {
    vec![LlmMessage::new(
        "session",
        MessageRole::User,
        vec![ContentPart::Text(TextContent {
            text: "Write with sources".to_owned(),
        })],
        "",
    )]
}

#[derive(Debug)]
struct FixtureTool;

#[async_trait]
impl Tool for FixtureTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "search_sources".to_owned(),
            description: "Search sources".to_owned(),
            parameters: BTreeMap::from([(
                "query".to_owned(),
                json!({"type": "string", "description": "search query"}),
            )]),
            required: vec!["query".to_owned()],
            parallel_safe: true,
        }
    }

    async fn run(
        &self,
        _context: ToolContext,
        _call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        Ok(ToolResponse::text("unused"))
    }
}

async fn collect(
    mut events: mpsc::Receiver<ProviderEvent>,
) -> TestResult<(Vec<ProviderEventType>, ProviderResponse)> {
    let mut event_types = Vec::new();
    let mut completed = None;
    while let Some(event) = events.recv().await {
        event_types.push(event.event_type);
        if event.event_type == ProviderEventType::Complete {
            completed = event.response;
        }
    }
    Ok((
        event_types,
        completed.ok_or("provider did not emit completion")?,
    ))
}

#[tokio::test]
async fn anthropic_adapter_preserves_messages_tools_stream_and_usage() -> TestResult {
    let records = [
        r#"data: {"type":"message_start","message":{"usage":{"input_tokens":12,"cache_creation_input_tokens":2,"cache_read_input_tokens":3}}}"#,
        r#"data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}"#,
        r#"data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"draft "}}"#,
        r#"data: {"type":"content_block_stop","index":0}"#,
        r#"data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool_1","name":"search_sources","input":{}}}"#,
        r#"data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"rust\"}"}}"#,
        r#"data: {"type":"content_block_stop","index":1}"#,
        r#"data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":7}}"#,
        r#"data: {"type":"message_stop"}"#,
    ];
    let (base_url, requests, server) = spawn_fixture(&records).await?;
    let provider = AnthropicClient::with_base_url(
        "anthropic-key",
        format!("{base_url}/v1"),
        model(ModelProvider::ANTHROPIC, "claude-test", false),
        "Anthropic system",
    )?;
    let events = provider
        .stream_response(
            CancellationToken::new(),
            messages(),
            vec![Arc::new(FixtureTool)],
        )
        .await?;
    let (event_types, completed) = collect(events).await?;
    assert_eq!(completed.content, "draft ");
    assert_eq!(completed.finish_reason, FinishReason::ToolUse);
    assert_eq!(completed.tool_calls[0].id, "tool_1");
    assert_eq!(completed.tool_calls[0].input, r#"{"query":"rust"}"#);
    assert_eq!(completed.usage.input_tokens, 12);
    assert_eq!(completed.usage.cache_creation_tokens, 2);
    assert_eq!(completed.usage.cache_read_tokens, 3);
    assert_eq!(completed.usage.output_tokens, 7);
    assert!(event_types.contains(&ProviderEventType::ToolUseDelta));

    let requests = requests.lock().await;
    let request = requests.first().ok_or("fixture did not receive request")?;
    assert_eq!(request.uri, "/v1/messages");
    assert_eq!(request.headers["x-api-key"], "anthropic-key");
    assert_eq!(request.headers["anthropic-version"], "2023-06-01");
    assert_eq!(request.body["model"], "claude-test");
    assert_eq!(request.body["system"], "Anthropic system");
    assert_eq!(request.body["messages"][0]["role"], "user");
    assert_eq!(request.body["tools"][0]["name"], "search_sources");
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn gemini_adapter_preserves_schema_thinking_stream_and_usage() -> TestResult {
    let records = [
        r#"data: {"candidates":[{"content":{"parts":[{"text":"thinking ","thought":true},{"text":"answer "}]}}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":4,"cachedContentTokenCount":2}}"#,
        r#"data: {"candidates":[{"content":{"parts":[{"functionCall":{"id":"call_1","name":"search_sources","args":{"query":"rust"}},"thoughtSignature":"c2ln"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":6,"cachedContentTokenCount":2}}"#,
    ];
    let (base_url, requests, server) = spawn_fixture(&records).await?;
    let provider = GeminiClient::with_base_url(
        "gemini-key",
        format!("{base_url}/v1beta"),
        model(ModelProvider::GEMINI, "gemini-test", true),
        "Gemini system",
    )?;
    let mut history = messages();
    history.push(LlmMessage::new(
        "session",
        MessageRole::Assistant,
        vec![ContentPart::ToolCall(ToolCall {
            id: "previous_call".to_owned(),
            name: "search_sources".to_owned(),
            input: r#"{"query":"rust"}"#.to_owned(),
            r#type: "function".to_owned(),
            finished: true,
            thought_signature: b"previous-signature".to_vec(),
        })],
        "gemini-test",
    ));
    history.push(LlmMessage::new(
        "session",
        MessageRole::Tool,
        vec![ContentPart::ToolResult(ToolResult {
            tool_call_id: "previous_call".to_owned(),
            content: r#"{"results":1}"#.to_owned(),
            metadata: String::new(),
            is_error: false,
        })],
        "gemini-test",
    ));
    let events = provider
        .stream_response(
            CancellationToken::new(),
            history,
            vec![Arc::new(FixtureTool)],
        )
        .await?;
    let (event_types, completed) = collect(events).await?;
    assert_eq!(completed.content, "answer ");
    assert_eq!(completed.reasoning, "thinking ");
    assert_eq!(completed.finish_reason, FinishReason::ToolUse);
    assert_eq!(completed.tool_calls[0].thought_signature, b"sig");
    assert_eq!(completed.usage.input_tokens, 9);
    assert_eq!(completed.usage.output_tokens, 6);
    assert_eq!(completed.usage.cache_read_tokens, 2);
    assert!(event_types.contains(&ProviderEventType::ThinkingDelta));

    let requests = requests.lock().await;
    let request = requests.first().ok_or("fixture did not receive request")?;
    assert_eq!(
        request.uri,
        "/v1beta/models/gemini-test:streamGenerateContent?alt=sse&key=gemini-key"
    );
    assert_eq!(
        request.body["systemInstruction"]["parts"][0]["text"],
        "Gemini system"
    );
    assert_eq!(
        request.body["tools"][0]["functionDeclarations"][0]["parameters"]["properties"]["query"]["type"],
        "STRING"
    );
    assert_eq!(
        request.body["generationConfig"]["thinkingConfig"]["thinkingLevel"],
        "MEDIUM"
    );
    assert_eq!(
        request.body["contents"][2]["parts"][0]["functionResponse"]["name"],
        "search_sources"
    );
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn vertex_adapter_uses_bearer_auth_and_vertex_publisher_path() -> TestResult {
    let records = [
        r#"data: {"candidates":[{"content":{"parts":[{"text":"vertex"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":1}}"#,
    ];
    let (base_url, requests, server) = spawn_fixture(&records).await?;
    let provider = VertexAiClient::with_base_url(
        "vertex-token",
        format!("{base_url}/v1/projects/test/locations/us-central1/publishers/google"),
        model(ModelProvider::VERTEX_AI, "gemini-vertex-test", false),
        "Vertex system",
    )?;
    let events = provider
        .stream_response(CancellationToken::new(), messages(), Vec::new())
        .await?;
    let (_, completed) = collect(events).await?;
    assert_eq!(completed.content, "vertex");
    assert_eq!(completed.finish_reason, FinishReason::EndTurn);

    let requests = requests.lock().await;
    let request = requests.first().ok_or("fixture did not receive request")?;
    assert_eq!(
        request.uri,
        "/v1/projects/test/locations/us-central1/publishers/google/models/gemini-vertex-test:streamGenerateContent?alt=sse"
    );
    assert_eq!(request.headers["authorization"], "Bearer vertex-token");
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn groq_adapter_reuses_the_fixed_responses_event_contract() -> TestResult {
    let records = [
        r#"data: {"type":"response.output_item.added","item":{"type":"reasoning","content":[{"type":"reasoning_text","text":"considering"}]}}"#,
        r#"data: {"type":"response.output_text.delta","delta":"groq"}"#,
        r#"data: {"type":"response.completed","response":{"status":"completed","output":[],"usage":{"input_tokens":5,"output_tokens":2,"input_tokens_details":{"cached_tokens":1}}}}"#,
        "data: [DONE]",
    ];
    let (base_url, requests, server) = spawn_fixture(&records).await?;
    let provider = GroqClient::with_base_url(
        "groq-key",
        format!("{base_url}/openai/v1"),
        model(ModelProvider::GROQ, "groq-test", true),
        "Groq system",
        Some("medium".to_owned()),
    )?;
    let events = provider
        .stream_response(CancellationToken::new(), messages(), Vec::new())
        .await?;
    let (_, completed) = collect(events).await?;
    assert_eq!(completed.content, "groq");
    assert_eq!(completed.reasoning, "considering");
    assert_eq!(completed.finish_reason, FinishReason::EndTurn);
    assert_eq!(completed.usage.input_tokens, 4);
    assert_eq!(completed.usage.cache_read_tokens, 1);
    assert_eq!(completed.usage.output_tokens, 2);

    let requests = requests.lock().await;
    let request = requests.first().ok_or("fixture did not receive request")?;
    assert_eq!(request.uri, "/openai/v1/responses");
    assert_eq!(request.headers["authorization"], "Bearer groq-key");
    assert_eq!(request.body["model"], "groq-test");
    assert_eq!(request.body["instructions"], "Groq system");
    assert_eq!(request.body["reasoning"]["effort"], "medium");
    assert!(request.body.get("store").is_none());
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn groq_text_generation_uses_the_active_insight_contract() -> TestResult {
    let generated = json!({
        "title": "Fixture insight",
        "summary": "Fixture summary",
        "content": "Fixture content",
        "key_points": ["one", "two", "three"]
    })
    .to_string();
    let (base_url, requests, server) = spawn_json_fixture(json!({
        "status": "completed",
        "output": [{
            "type": "message",
            "content": [{"type": "output_text", "text": generated.clone()}]
        }],
        "usage": {
            "input_tokens": 10,
            "output_tokens": 8,
            "input_tokens_details": {"cached_tokens": 0}
        }
    }))
    .await?;
    let provider = GroqClient::with_base_url(
        "groq-key",
        format!("{base_url}/openai/v1"),
        Model::new(
            "openai/gpt-oss-120b",
            ModelProvider::GROQ,
            "openai/gpt-oss-120b",
            2_000,
            true,
            true,
        ),
        "Return strict insight JSON.",
        Some("medium".to_owned()),
    )?;

    assert_eq!(
        provider.generate_text("typed insight input").await?,
        generated
    );
    let requests = requests.lock().await;
    let request = requests.first().ok_or("fixture did not receive request")?;
    assert_eq!(request.body["model"], "openai/gpt-oss-120b");
    assert_eq!(request.body["max_output_tokens"], 2_000);
    assert_eq!(request.body["input"], "typed insight input");
    assert_eq!(request.body["reasoning"]["effort"], "medium");
    assert!(request.body.get("stream").is_none());
    assert!(request.body.get("store").is_none());
    drop(requests);

    server.abort();
    let _ = server.await;
    Ok(())
}

#[tokio::test]
async fn provider_adapters_reject_mismatched_models_and_cancel_before_io() -> TestResult {
    assert!(
        AnthropicClient::new(
            "key",
            model(ModelProvider::GEMINI, "wrong", false),
            "system",
        )
        .is_err()
    );
    assert!(
        GroqClient::new(
            "key",
            model(ModelProvider::OPENAI, "wrong", false),
            "system",
            None,
        )
        .is_err()
    );

    let provider = GeminiClient::with_base_url(
        "key",
        "http://127.0.0.1:1/v1beta",
        model(ModelProvider::GEMINI, "gemini-test", false),
        "system",
    )?;
    let cancellation = CancellationToken::new();
    cancellation.cancel();
    let result = provider
        .stream_response(cancellation, messages(), Vec::new())
        .await;
    assert!(matches!(
        result,
        Err(blog_backend::core::ml::llm::ProviderError::Cancelled)
    ));
    Ok(())
}
