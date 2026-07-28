use std::{collections::HashMap, sync::Arc, time::Duration};

use async_trait::async_trait;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use futures_util::StreamExt;
use reqwest::{Client, RequestBuilder};
use secrecy::{ExposeSecret, SecretString};
use serde_json::{Value, json};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use crate::{
    core::ml::llm::{
        ContentPart, FinishReason, LlmMessage, MessageRole, Model, ModelProvider, Provider,
        ProviderError, ProviderEvent, ProviderEventType, ProviderResponse, TokenUsage, Tool,
        ToolCall,
    },
    error::AppError,
    integrations::openai::OpenAiClient,
};

const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);
const STREAM_BUFFER: usize = 100;
const ANTHROPIC_BASE_URL: &str = "https://api.anthropic.com/v1";
const GEMINI_BASE_URL: &str = "https://generativelanguage.googleapis.com/v1beta";
const GROQ_BASE_URL: &str = "https://api.groq.com/openai/v1";

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Protocol {
    Anthropic,
    GeminiApiKey,
    VertexBearer,
}

#[derive(Clone)]
struct JsonStreamClient {
    client: Client,
    credential: SecretString,
    base_url: String,
    model: Model,
    system_message: String,
    protocol: Protocol,
}

#[derive(Clone)]
pub struct AnthropicClient(JsonStreamClient);

#[derive(Clone)]
pub struct GeminiClient(JsonStreamClient);

#[derive(Clone)]
pub struct VertexAiClient(JsonStreamClient);

#[derive(Clone)]
pub struct GroqClient(OpenAiClient);

impl AnthropicClient {
    pub fn new(
        api_key: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
    ) -> Result<Self, ProviderError> {
        Self::with_base_url(api_key, ANTHROPIC_BASE_URL, model, system_message)
    }

    pub fn with_base_url(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
    ) -> Result<Self, ProviderError> {
        JsonStreamClient::new(
            api_key,
            base_url,
            model,
            system_message,
            Protocol::Anthropic,
        )
        .map(Self)
    }
}

impl GeminiClient {
    pub fn new(
        api_key: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
    ) -> Result<Self, ProviderError> {
        Self::with_base_url(api_key, GEMINI_BASE_URL, model, system_message)
    }

    pub fn with_base_url(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
    ) -> Result<Self, ProviderError> {
        JsonStreamClient::new(
            api_key,
            base_url,
            model,
            system_message,
            Protocol::GeminiApiKey,
        )
        .map(Self)
    }
}

impl VertexAiClient {
    pub fn with_base_url(
        access_token: impl Into<String>,
        publisher_base_url: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
    ) -> Result<Self, ProviderError> {
        JsonStreamClient::new(
            access_token,
            publisher_base_url,
            model,
            system_message,
            Protocol::VertexBearer,
        )
        .map(Self)
    }
}

impl GroqClient {
    pub fn new(
        api_key: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
        reasoning_effort: Option<String>,
    ) -> Result<Self, ProviderError> {
        Self::with_base_url(
            api_key,
            GROQ_BASE_URL,
            model,
            system_message,
            reasoning_effort,
        )
    }

    pub fn with_base_url(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
        reasoning_effort: Option<String>,
    ) -> Result<Self, ProviderError> {
        if model.provider.0 != ModelProvider::GROQ {
            return Err(ProviderError::Request(
                "Groq client requires a groq model".to_owned(),
            ));
        }
        OpenAiClient::with_base_url(api_key, base_url)
            .and_then(|client| client.with_provider_model(model, system_message, reasoning_effort))
            .map(Self)
            .map_err(|error| ProviderError::Request(error.to_string()))
    }

    pub fn is_configured(&self) -> bool {
        self.0.is_configured()
    }

    pub async fn generate_text(&self, input: &str) -> Result<String, AppError> {
        self.0.generate_provider_text(input).await
    }
}

impl JsonStreamClient {
    fn new(
        credential: impl Into<String>,
        base_url: impl Into<String>,
        model: Model,
        system_message: impl Into<String>,
        protocol: Protocol,
    ) -> Result<Self, ProviderError> {
        let base_url = base_url.into().trim_end_matches('/').to_owned();
        if base_url.is_empty() {
            return Err(ProviderError::Request(
                "provider base URL must not be empty".to_owned(),
            ));
        }
        if model.api_model.trim().is_empty() {
            return Err(ProviderError::Request(
                "provider API model must not be empty".to_owned(),
            ));
        }
        if model.default_max_tokens <= 0 {
            return Err(ProviderError::Request(
                "provider max tokens must be greater than zero".to_owned(),
            ));
        }
        let expected_provider = match protocol {
            Protocol::Anthropic => ModelProvider::ANTHROPIC,
            Protocol::GeminiApiKey => ModelProvider::GEMINI,
            Protocol::VertexBearer => ModelProvider::VERTEX_AI,
        };
        if model.provider.0 != expected_provider {
            return Err(ProviderError::Request(format!(
                "{} client requires a {expected_provider} model",
                model.provider.0
            )));
        }
        let system_message = system_message.into();
        if system_message.trim().is_empty() {
            return Err(ProviderError::Request(
                "provider system message must not be empty".to_owned(),
            ));
        }
        let client = Client::builder()
            .timeout(REQUEST_TIMEOUT)
            .build()
            .map_err(|error| ProviderError::Request(error.to_string()))?;
        Ok(Self {
            client,
            credential: SecretString::from(credential.into()),
            base_url,
            model,
            system_message,
            protocol,
        })
    }

    fn request(&self, messages: &[LlmMessage], tools: &[Arc<dyn Tool>]) -> RequestBuilder {
        match self.protocol {
            Protocol::Anthropic => self
                .client
                .post(format!("{}/messages", self.base_url))
                .header("x-api-key", self.credential.expose_secret())
                .header("anthropic-version", "2023-06-01")
                .json(&anthropic_request(self, messages, tools)),
            Protocol::GeminiApiKey => self
                .client
                .post(format!(
                    "{}/models/{}:streamGenerateContent",
                    self.base_url, self.model.api_model
                ))
                .query(&[("alt", "sse"), ("key", self.credential.expose_secret())])
                .json(&gemini_request(self, messages, tools)),
            Protocol::VertexBearer => self
                .client
                .post(format!(
                    "{}/models/{}:streamGenerateContent",
                    self.base_url, self.model.api_model
                ))
                .query(&[("alt", "sse")])
                .bearer_auth(self.credential.expose_secret())
                .json(&gemini_request(self, messages, tools)),
        }
    }
}

macro_rules! impl_provider {
    ($provider:ty) => {
        #[async_trait]
        impl Provider for $provider {
            fn model(&self) -> Model {
                self.0.model.clone()
            }

            fn system_message(&self) -> &str {
                &self.0.system_message
            }

            async fn stream_response(
                &self,
                cancellation: CancellationToken,
                messages: Vec<LlmMessage>,
                tools: Vec<Arc<dyn Tool>>,
            ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
                stream_json_provider(&self.0, cancellation, messages, tools).await
            }
        }
    };
}

impl_provider!(AnthropicClient);
impl_provider!(GeminiClient);
impl_provider!(VertexAiClient);

#[async_trait]
impl Provider for GroqClient {
    fn model(&self) -> Model {
        self.0.model()
    }

    fn system_message(&self) -> &str {
        self.0.system_message()
    }

    async fn stream_response(
        &self,
        cancellation: CancellationToken,
        messages: Vec<LlmMessage>,
        tools: Vec<Arc<dyn Tool>>,
    ) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
        self.0.stream_response(cancellation, messages, tools).await
    }
}

async fn stream_json_provider(
    provider: &JsonStreamClient,
    cancellation: CancellationToken,
    messages: Vec<LlmMessage>,
    tools: Vec<Arc<dyn Tool>>,
) -> Result<mpsc::Receiver<ProviderEvent>, ProviderError> {
    if provider.credential.expose_secret().is_empty() {
        return Err(ProviderError::Request(format!(
            "{} API credential is not configured",
            provider.model.provider.0
        )));
    }
    let response = tokio::select! {
        biased;
        () = cancellation.cancelled() => return Err(ProviderError::Cancelled),
        response = provider.request(&messages, &tools).send() => {
            response.map_err(|error| ProviderError::Request(error.to_string()))?
        },
    };
    if !response.status().is_success() {
        return Err(ProviderError::Request(format!(
            "{} returned HTTP {}",
            provider.model.provider.0,
            response.status()
        )));
    }
    let (sender, receiver) = mpsc::channel(STREAM_BUFFER);
    let protocol = provider.protocol;
    tokio::spawn(async move {
        run_sse(response, sender, cancellation, protocol).await;
    });
    Ok(receiver)
}

fn anthropic_request(
    provider: &JsonStreamClient,
    messages: &[LlmMessage],
    tools: &[Arc<dyn Tool>],
) -> Value {
    let messages = messages
        .iter()
        .filter_map(anthropic_message)
        .collect::<Vec<_>>();
    let tools = tools
        .iter()
        .map(|tool| {
            let info = tool.info();
            json!({
                "name": info.name,
                "description": info.description,
                "input_schema": {
                    "type": "object",
                    "properties": info.parameters,
                    "required": info.required,
                },
            })
        })
        .collect::<Vec<_>>();
    let mut body = json!({
        "model": provider.model.api_model,
        "max_tokens": provider.model.default_max_tokens,
        "temperature": 0,
        "system": provider.system_message,
        "messages": messages,
        "stream": true,
    });
    if !tools.is_empty() {
        body["tools"] = Value::Array(tools);
    }
    if provider.model.can_reason {
        body["thinking"] = json!({
            "type": "enabled",
            "budget_tokens": (provider.model.default_max_tokens * 8 / 10).max(1),
        });
        body["temperature"] = json!(1);
    }
    body
}

fn anthropic_message(message: &LlmMessage) -> Option<Value> {
    let role = match message.role {
        MessageRole::User | MessageRole::Tool => "user",
        MessageRole::Assistant => "assistant",
        MessageRole::System => return None,
    };
    let mut blocks = Vec::new();
    for part in &message.parts {
        match part {
            ContentPart::Text(text) if !text.text.is_empty() => {
                blocks.push(json!({"type": "text", "text": text.text}));
            }
            ContentPart::Binary(binary) if message.role == MessageRole::User => {
                blocks.push(json!({
                    "type": "image",
                    "source": {
                        "type": "base64",
                        "media_type": binary.mime_type,
                        "data": STANDARD.encode(&binary.data),
                    },
                }));
            }
            ContentPart::ToolCall(call) if message.role == MessageRole::Assistant => {
                blocks.push(json!({
                    "type": "tool_use",
                    "id": call.id,
                    "name": call.name,
                    "input": parse_json_or_string(&call.input),
                }));
            }
            ContentPart::ToolResult(result) if message.role == MessageRole::Tool => {
                blocks.push(json!({
                    "type": "tool_result",
                    "tool_use_id": result.tool_call_id,
                    "content": result.content,
                    "is_error": result.is_error,
                }));
            }
            _ => {}
        }
    }
    (!blocks.is_empty()).then(|| json!({"role": role, "content": blocks}))
}

fn gemini_request(
    provider: &JsonStreamClient,
    messages: &[LlmMessage],
    tools: &[Arc<dyn Tool>],
) -> Value {
    let tool_names = messages
        .iter()
        .flat_map(LlmMessage::tool_calls)
        .map(|call| (call.id, call.name))
        .collect::<HashMap<_, _>>();
    let contents = messages
        .iter()
        .filter_map(|message| gemini_message(message, &tool_names))
        .collect::<Vec<_>>();
    let declarations = tools
        .iter()
        .map(|tool| {
            let info = tool.info();
            json!({
                "name": info.name,
                "description": info.description,
                "parameters": {
                    "type": "OBJECT",
                    "properties": uppercase_schema_types(Value::Object(
                        info.parameters.into_iter().collect()
                    )),
                    "required": info.required,
                },
            })
        })
        .collect::<Vec<_>>();
    let mut generation_config = json!({
        "maxOutputTokens": provider.model.default_max_tokens,
    });
    if provider.model.can_reason {
        generation_config["thinkingConfig"] = json!({"thinkingLevel": "MEDIUM"});
    }
    let mut body = json!({
        "systemInstruction": {
            "parts": [{"text": provider.system_message}],
        },
        "contents": contents,
        "generationConfig": generation_config,
    });
    if !declarations.is_empty() {
        body["tools"] = json!([{"functionDeclarations": declarations}]);
    }
    body
}

fn gemini_message(message: &LlmMessage, tool_names: &HashMap<String, String>) -> Option<Value> {
    let role = match message.role {
        MessageRole::User | MessageRole::Tool => "user",
        MessageRole::Assistant => "model",
        MessageRole::System => return None,
    };
    let mut parts = Vec::new();
    for part in &message.parts {
        match part {
            ContentPart::Text(text) if !text.text.is_empty() => {
                parts.push(json!({"text": text.text}));
            }
            ContentPart::Binary(binary) if message.role == MessageRole::User => {
                parts.push(json!({
                    "inlineData": {
                        "mimeType": binary.mime_type,
                        "data": STANDARD.encode(&binary.data),
                    },
                }));
            }
            ContentPart::ToolCall(call) if message.role == MessageRole::Assistant => {
                let mut function_call = json!({
                    "name": call.name,
                    "args": parse_json_or_string(&call.input),
                });
                if !call.thought_signature.is_empty() {
                    function_call["thoughtSignature"] =
                        json!(STANDARD.encode(&call.thought_signature));
                }
                parts.push(json!({"functionCall": function_call}));
            }
            ContentPart::ToolResult(result) if message.role == MessageRole::Tool => {
                parts.push(json!({
                    "functionResponse": {
                        "name": tool_names
                            .get(&result.tool_call_id)
                            .cloned()
                            .unwrap_or_else(|| result.tool_call_id.clone()),
                        "response": parse_json_or_result(&result.content),
                    },
                }));
            }
            _ => {}
        }
    }
    (!parts.is_empty()).then(|| json!({"role": role, "parts": parts}))
}

fn parse_json_or_string(value: &str) -> Value {
    serde_json::from_str(value).unwrap_or_else(|_| json!({"value": value}))
}

fn parse_json_or_result(value: &str) -> Value {
    serde_json::from_str(value).unwrap_or_else(|_| json!({"result": value}))
}

fn uppercase_schema_types(value: Value) -> Value {
    match value {
        Value::Array(values) => {
            Value::Array(values.into_iter().map(uppercase_schema_types).collect())
        }
        Value::Object(values) => Value::Object(
            values
                .into_iter()
                .map(|(key, value)| {
                    if key == "type" {
                        let value = match value {
                            Value::String(value) => Value::String(value.to_uppercase()),
                            other => uppercase_schema_types(other),
                        };
                        (key, value)
                    } else {
                        (key, uppercase_schema_types(value))
                    }
                })
                .collect(),
        ),
        other => other,
    }
}

#[derive(Default)]
struct StreamState {
    content: String,
    reasoning: String,
    pending_calls: HashMap<String, ToolCall>,
    tool_calls: Vec<ToolCall>,
    usage: TokenUsage,
    finish_reason: FinishReason,
    completed: bool,
    next_call_id: usize,
}

async fn run_sse(
    response: reqwest::Response,
    sender: mpsc::Sender<ProviderEvent>,
    cancellation: CancellationToken,
    protocol: Protocol,
) {
    let mut stream = response.bytes_stream();
    let mut buffer = Vec::new();
    let mut state = StreamState::default();
    loop {
        let chunk = tokio::select! {
            biased;
            () = cancellation.cancelled() => return,
            () = sender.closed() => return,
            chunk = stream.next() => chunk,
        };
        let Some(chunk) = chunk else {
            break;
        };
        let chunk = match chunk {
            Ok(chunk) => chunk,
            Err(error) => {
                let _ = send_event(&sender, &cancellation, error_event(error.to_string())).await;
                return;
            }
        };
        buffer.extend_from_slice(&chunk);
        while let Some(boundary) = sse_boundary(&buffer) {
            let record = buffer.drain(..boundary).collect::<Vec<_>>();
            drain_sse_separator(&mut buffer);
            let Ok(record) = std::str::from_utf8(&record) else {
                let _ = send_event(
                    &sender,
                    &cancellation,
                    error_event("provider returned invalid UTF-8 SSE data"),
                )
                .await;
                return;
            };
            let data = record
                .lines()
                .filter_map(|line| line.trim_end_matches('\r').strip_prefix("data:"))
                .map(str::trim_start)
                .collect::<Vec<_>>()
                .join("\n");
            if data.is_empty() || data == "[DONE]" {
                continue;
            }
            let event: Value = match serde_json::from_str(&data) {
                Ok(event) => event,
                Err(error) => {
                    let _ =
                        send_event(&sender, &cancellation, error_event(error.to_string())).await;
                    return;
                }
            };
            let result = match protocol {
                Protocol::Anthropic => {
                    process_anthropic_event(&sender, &cancellation, &mut state, &event).await
                }
                Protocol::GeminiApiKey | Protocol::VertexBearer => {
                    process_gemini_event(&sender, &cancellation, &mut state, &event).await
                }
            };
            if result.is_err() || state.completed {
                return;
            }
        }
    }
    if !state.completed {
        let _ = send_event(
            &sender,
            &cancellation,
            error_event(ProviderError::MissingCompletion.to_string()),
        )
        .await;
    }
}

fn sse_boundary(buffer: &[u8]) -> Option<usize> {
    buffer
        .windows(2)
        .position(|window| window == b"\n\n")
        .or_else(|| buffer.windows(4).position(|window| window == b"\r\n\r\n"))
}

fn drain_sse_separator(buffer: &mut Vec<u8>) {
    if buffer.starts_with(b"\r\n\r\n") {
        buffer.drain(..4);
    } else if buffer.starts_with(b"\n\n") {
        buffer.drain(..2);
    }
}

async fn process_anthropic_event(
    sender: &mpsc::Sender<ProviderEvent>,
    cancellation: &CancellationToken,
    state: &mut StreamState,
    event: &Value,
) -> Result<(), ()> {
    match string_at(event, &["type"]) {
        "message_start" => {
            state.usage.input_tokens = i64_at(event, &["message", "usage", "input_tokens"]);
            state.usage.cache_creation_tokens =
                i64_at(event, &["message", "usage", "cache_creation_input_tokens"]);
            state.usage.cache_read_tokens =
                i64_at(event, &["message", "usage", "cache_read_input_tokens"]);
        }
        "content_block_start" => {
            let block = event.get("content_block").unwrap_or(&Value::Null);
            match string_at(block, &["type"]) {
                "text" => {
                    send_event(
                        sender,
                        cancellation,
                        typed_event(ProviderEventType::ContentStart),
                    )
                    .await?;
                }
                "thinking" => {}
                "tool_use" => {
                    let id = string_at(block, &["id"]).to_owned();
                    let index = i64_at(event, &["index"]).to_string();
                    let call = ToolCall {
                        id: id.clone(),
                        name: string_at(block, &["name"]).to_owned(),
                        input: String::new(),
                        r#type: "function".to_owned(),
                        finished: false,
                        thought_signature: Vec::new(),
                    };
                    state.pending_calls.insert(index, call.clone());
                    send_event(
                        sender,
                        cancellation,
                        tool_event(ProviderEventType::ToolUseStart, call),
                    )
                    .await?;
                }
                _ => {}
            }
        }
        "content_block_delta" => {
            let delta = event.get("delta").unwrap_or(&Value::Null);
            match string_at(delta, &["type"]) {
                "text_delta" => {
                    let text = string_at(delta, &["text"]).to_owned();
                    state.content.push_str(&text);
                    if !text.is_empty() {
                        send_event(sender, cancellation, ProviderEvent::content_delta(text))
                            .await?;
                    }
                }
                "thinking_delta" => {
                    let thinking = string_at(delta, &["thinking"]).to_owned();
                    state.reasoning.push_str(&thinking);
                    if !thinking.is_empty() {
                        send_event(
                            sender,
                            cancellation,
                            ProviderEvent::thinking_delta(thinking),
                        )
                        .await?;
                    }
                }
                "input_json_delta" => {
                    let fragment = string_at(delta, &["partial_json"]).to_owned();
                    let index = i64_at(event, &["index"]).to_string();
                    if let Some(call) = state.pending_calls.get_mut(&index) {
                        call.input.push_str(&fragment);
                        send_event(
                            sender,
                            cancellation,
                            tool_event(ProviderEventType::ToolUseDelta, call.clone()),
                        )
                        .await?;
                    }
                }
                _ => {}
            }
        }
        "content_block_stop" => {
            let index = i64_at(event, &["index"]).to_string();
            if let Some(mut call) = state.pending_calls.remove(&index) {
                call.finished = true;
                state.tool_calls.push(call.clone());
                send_event(
                    sender,
                    cancellation,
                    tool_event(ProviderEventType::ToolUseStop, call),
                )
                .await?;
            } else {
                send_event(
                    sender,
                    cancellation,
                    typed_event(ProviderEventType::ContentStop),
                )
                .await?;
            }
        }
        "message_delta" => {
            state.usage.output_tokens = i64_at(event, &["usage", "output_tokens"]);
            state.finish_reason =
                anthropic_finish_reason(string_at(event, &["delta", "stop_reason"]));
        }
        "message_stop" => complete(sender, cancellation, state).await?,
        "error" => {
            let message = string_at(event, &["error", "message"]);
            send_event(sender, cancellation, error_event(message)).await?;
        }
        _ => {}
    }
    Ok(())
}

async fn process_gemini_event(
    sender: &mpsc::Sender<ProviderEvent>,
    cancellation: &CancellationToken,
    state: &mut StreamState,
    event: &Value,
) -> Result<(), ()> {
    if event.get("error").is_some() {
        return send_event(
            sender,
            cancellation,
            error_event(string_at(event, &["error", "message"])),
        )
        .await;
    }
    state.usage.input_tokens =
        i64_at(event, &["usageMetadata", "promptTokenCount"]).max(state.usage.input_tokens);
    state.usage.output_tokens =
        i64_at(event, &["usageMetadata", "candidatesTokenCount"]).max(state.usage.output_tokens);
    state.usage.cache_read_tokens = i64_at(event, &["usageMetadata", "cachedContentTokenCount"])
        .max(state.usage.cache_read_tokens);

    if let Some(parts) = event
        .pointer("/candidates/0/content/parts")
        .and_then(Value::as_array)
    {
        for part in parts {
            if let Some(text) = part.get("text").and_then(Value::as_str) {
                if part.get("thought").and_then(Value::as_bool) == Some(true) {
                    state.reasoning.push_str(text);
                    send_event(
                        sender,
                        cancellation,
                        ProviderEvent::thinking_delta(text.to_owned()),
                    )
                    .await?;
                } else {
                    state.content.push_str(text);
                    send_event(
                        sender,
                        cancellation,
                        ProviderEvent::content_delta(text.to_owned()),
                    )
                    .await?;
                }
            }
            if let Some(function_call) = part.get("functionCall") {
                state.next_call_id += 1;
                let call = ToolCall {
                    id: function_call
                        .get("id")
                        .and_then(Value::as_str)
                        .map(ToOwned::to_owned)
                        .unwrap_or_else(|| format!("gemini_call_{}", state.next_call_id)),
                    name: string_at(function_call, &["name"]).to_owned(),
                    input: function_call
                        .get("args")
                        .map(Value::to_string)
                        .unwrap_or_else(|| "{}".to_owned()),
                    r#type: "function".to_owned(),
                    finished: true,
                    thought_signature: part
                        .get("thoughtSignature")
                        .and_then(Value::as_str)
                        .and_then(|value| STANDARD.decode(value).ok())
                        .unwrap_or_default(),
                };
                send_event(
                    sender,
                    cancellation,
                    tool_event(ProviderEventType::ToolUseStart, call.clone()),
                )
                .await?;
                send_event(
                    sender,
                    cancellation,
                    tool_event(ProviderEventType::ToolUseStop, call.clone()),
                )
                .await?;
                state.tool_calls.push(call);
            }
        }
    }
    let finish_reason = string_at(event, &["candidates", "0", "finishReason"]);
    if !finish_reason.is_empty() {
        state.finish_reason = gemini_finish_reason(finish_reason);
        complete(sender, cancellation, state).await?;
    }
    Ok(())
}

async fn complete(
    sender: &mpsc::Sender<ProviderEvent>,
    cancellation: &CancellationToken,
    state: &mut StreamState,
) -> Result<(), ()> {
    if state.completed {
        return Ok(());
    }
    if !state.tool_calls.is_empty() {
        state.finish_reason = FinishReason::ToolUse;
    } else if state.finish_reason == FinishReason::Unknown {
        state.finish_reason = FinishReason::EndTurn;
    }
    state.completed = true;
    send_event(
        sender,
        cancellation,
        ProviderEvent::complete(ProviderResponse {
            content: state.content.clone(),
            reasoning: state.reasoning.clone(),
            tool_calls: state.tool_calls.clone(),
            usage: state.usage,
            finish_reason: state.finish_reason,
        }),
    )
    .await
}

fn anthropic_finish_reason(reason: &str) -> FinishReason {
    match reason {
        "end_turn" | "stop_sequence" => FinishReason::EndTurn,
        "max_tokens" => FinishReason::MaxTokens,
        "tool_use" => FinishReason::ToolUse,
        _ => FinishReason::Unknown,
    }
}

fn gemini_finish_reason(reason: &str) -> FinishReason {
    match reason {
        "STOP" => FinishReason::EndTurn,
        "MAX_TOKENS" => FinishReason::MaxTokens,
        _ => FinishReason::Unknown,
    }
}

fn typed_event(event_type: ProviderEventType) -> ProviderEvent {
    ProviderEvent {
        event_type,
        content: String::new(),
        thinking: String::new(),
        response: None,
        tool_call: None,
        error: None,
    }
}

fn tool_event(event_type: ProviderEventType, tool_call: ToolCall) -> ProviderEvent {
    ProviderEvent {
        tool_call: Some(tool_call),
        ..typed_event(event_type)
    }
}

fn error_event(error: impl Into<String>) -> ProviderEvent {
    ProviderEvent {
        event_type: ProviderEventType::Error,
        content: String::new(),
        thinking: String::new(),
        response: None,
        tool_call: None,
        error: Some(ProviderError::Request(error.into())),
    }
}

async fn send_event(
    sender: &mpsc::Sender<ProviderEvent>,
    cancellation: &CancellationToken,
    event: ProviderEvent,
) -> Result<(), ()> {
    tokio::select! {
        biased;
        () = cancellation.cancelled() => Err(()),
        result = sender.send(event) => result.map_err(|_| ()),
    }
}

fn string_at<'a>(value: &'a Value, path: &[&str]) -> &'a str {
    let mut current = value;
    for segment in path {
        current = match current {
            Value::Array(values) => segment
                .parse::<usize>()
                .ok()
                .and_then(|index| values.get(index))
                .unwrap_or(&Value::Null),
            Value::Object(values) => values.get(*segment).unwrap_or(&Value::Null),
            _ => &Value::Null,
        };
    }
    current.as_str().unwrap_or_default()
}

fn i64_at(value: &Value, path: &[&str]) -> i64 {
    let mut current = value;
    for segment in path {
        current = match current {
            Value::Array(values) => segment
                .parse::<usize>()
                .ok()
                .and_then(|index| values.get(index))
                .unwrap_or(&Value::Null),
            Value::Object(values) => values.get(*segment).unwrap_or(&Value::Null),
            _ => &Value::Null,
        };
    }
    current.as_i64().unwrap_or_default()
}
