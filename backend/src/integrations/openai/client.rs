use std::{collections::HashMap, sync::Arc, time::Duration};

use async_trait::async_trait;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use futures_util::StreamExt;
use reqwest::Client;
use secrecy::{ExposeSecret, SecretString};
use serde::{Deserialize, Serialize};

use crate::{
    core::{
        article::{Article, ArticleContextWriter, ArticleEmbeddingProvider},
        insight,
        ml::{
            EmbeddingPort as MlEmbeddingPort, TextGenerationPort,
            llm::{
                ContentPart, FinishReason, LlmMessage, MessageRole, Model, Provider, ProviderError,
                ProviderEvent, ProviderEventType, ProviderResponse, TokenUsage, Tool, ToolCall,
                copilot_prompt,
            },
        },
        source,
    },
    error::AppError,
};

const DEFAULT_BASE_URL: &str = "https://api.openai.com/v1";
const DEFAULT_EMBEDDING_MODEL: &str = "text-embedding-3-small";
const DEFAULT_GENERATION_MODEL: &str = "gpt-5-2025-08-07";
const COPILOT_MODEL_ID: &str = "gpt-5.4-mini";
const COPILOT_API_MODEL: &str = "gpt-5.4-mini-2026-03-17";
const COPILOT_MAX_OUTPUT_TOKENS: i64 = 16_384;
const DEFAULT_IMAGE_MODEL: &str = "gpt-image-1";
const EMBEDDING_DIMENSIONS: usize = 1536;
const REQUEST_TIMEOUT: Duration = Duration::from_secs(30);

#[derive(Clone)]
pub struct OpenAiClient {
    client: Client,
    api_key: SecretString,
    base_url: String,
    embedding_model: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GeneratedImage {
    Url(String),
    Bytes(Vec<u8>),
}

impl OpenAiClient {
    pub fn new(api_key: impl Into<String>) -> Result<Self, AppError> {
        Self::with_base_url(api_key, DEFAULT_BASE_URL)
    }

    pub fn with_base_url(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
    ) -> Result<Self, AppError> {
        let api_key = api_key.into();
        let base_url = base_url.into().trim_end_matches('/').to_owned();
        if base_url.is_empty() {
            return Err(AppError::InvalidInput(
                "OpenAI base URL must not be empty".to_owned(),
            ));
        }
        Ok(Self {
            client: Client::builder()
                .timeout(REQUEST_TIMEOUT)
                .build()
                .map_err(|_| AppError::Internal)?,
            api_key: SecretString::from(api_key),
            base_url,
            embedding_model: DEFAULT_EMBEDDING_MODEL.to_owned(),
        })
    }

    pub fn with_embedding_model(mut self, model: impl Into<String>) -> Result<Self, AppError> {
        let model = model.into();
        if model.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "OpenAI embedding model must not be empty".to_owned(),
            ));
        }
        self.embedding_model = model;
        Ok(self)
    }

    pub fn is_configured(&self) -> bool {
        !self.api_key.expose_secret().is_empty()
    }

    pub async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        if text.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "embedding input must not be empty".to_owned(),
            ));
        }
        let response = self
            .client
            .post(format!("{}/embeddings", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .json(&EmbeddingRequest {
                input: text,
                model: &self.embedding_model,
                dimensions: EMBEDDING_DIMENSIONS,
            })
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let body: EmbeddingResponse = response.json().await.map_err(|_| AppError::External)?;
        let embedding = body
            .data
            .into_iter()
            .min_by_key(|item| item.index)
            .ok_or(AppError::External)?
            .embedding;
        if embedding.len() != EMBEDDING_DIMENSIONS {
            return Err(AppError::External);
        }
        Ok(embedding)
    }

    pub async fn generate_text(&self, instructions: &str, input: &str) -> Result<String, AppError> {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        if input.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "response input must not be empty".to_owned(),
            ));
        }
        let response = self
            .client
            .post(format!("{}/responses", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .json(&ResponseRequest::text(instructions, input))
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let body: ResponseBody = response.json().await.map_err(|_| AppError::External)?;
        let text = body
            .output
            .into_iter()
            .flat_map(|item| item.content)
            .filter_map(|content| content.text)
            .collect::<Vec<_>>()
            .join("");
        if text.is_empty() {
            Err(AppError::External)
        } else {
            Ok(text)
        }
    }

    pub async fn generate_image(&self, prompt: &str) -> Result<GeneratedImage, AppError> {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        if prompt.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "image prompt must not be empty".to_owned(),
            ));
        }
        let response = self
            .client
            .post(format!("{}/images/generations", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .json(&ImageRequest {
                model: DEFAULT_IMAGE_MODEL,
                prompt,
                size: "1536x1024",
            })
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let body: ImageResponse = response.json().await.map_err(|_| AppError::External)?;
        let image = body.data.into_iter().next().ok_or(AppError::External)?;
        if let Some(url) = image.url.filter(|value| !value.is_empty()) {
            return Ok(GeneratedImage::Url(url));
        }
        let encoded = image
            .b64_json
            .filter(|value| !value.is_empty())
            .ok_or(AppError::External)?;
        STANDARD
            .decode(encoded)
            .map(GeneratedImage::Bytes)
            .map_err(|_| AppError::External)
    }
}

#[derive(Serialize)]
struct EmbeddingRequest<'a> {
    input: &'a str,
    model: &'a str,
    dimensions: usize,
}

#[derive(Deserialize)]
struct EmbeddingResponse {
    data: Vec<EmbeddingItem>,
}

#[derive(Deserialize)]
struct EmbeddingItem {
    index: usize,
    embedding: Vec<f32>,
}

#[derive(Serialize)]
struct ResponseRequest<'a> {
    model: &'a str,
    instructions: String,
    input: serde_json::Value,
    store: bool,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    stream: bool,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    tools: Vec<ResponseTool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    max_output_tokens: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    reasoning: Option<ResponseReasoning>,
}

impl<'a> ResponseRequest<'a> {
    fn text(instructions: &str, input: &str) -> Self {
        Self {
            model: DEFAULT_GENERATION_MODEL,
            instructions: instructions.to_owned(),
            input: serde_json::Value::String(input.to_owned()),
            store: false,
            stream: false,
            tools: Vec::new(),
            max_output_tokens: None,
            reasoning: None,
        }
    }
}

#[derive(Serialize)]
struct ResponseReasoning {
    effort: &'static str,
}

#[derive(Deserialize)]
struct ResponseBody {
    #[serde(default)]
    output: Vec<ResponseOutput>,
    #[serde(default)]
    status: String,
    #[serde(default)]
    usage: ResponseUsage,
}

#[derive(Deserialize)]
struct ResponseOutput {
    #[serde(rename = "type", default)]
    output_type: String,
    #[serde(default)]
    content: Vec<ResponseContent>,
    #[serde(default)]
    call_id: String,
    #[serde(default)]
    name: String,
    #[serde(default)]
    arguments: String,
}

#[derive(Deserialize)]
struct ResponseContent {
    #[serde(rename = "type", default)]
    content_type: String,
    #[serde(default)]
    text: Option<String>,
}

#[derive(Debug, Default, Deserialize)]
struct ResponseUsage {
    #[serde(default)]
    input_tokens: i64,
    #[serde(default)]
    output_tokens: i64,
    #[serde(default)]
    input_tokens_details: ResponseInputTokenDetails,
}

#[derive(Debug, Default, Deserialize)]
struct ResponseInputTokenDetails {
    #[serde(default)]
    cached_tokens: i64,
}

#[derive(Serialize)]
struct ResponseTool {
    r#type: &'static str,
    name: String,
    description: String,
    parameters: serde_json::Value,
    strict: bool,
}

#[derive(Serialize)]
struct ImageRequest<'a> {
    model: &'a str,
    prompt: &'a str,
    size: &'a str,
}

#[derive(Deserialize)]
struct ImageResponse {
    data: Vec<ImageItem>,
}

#[derive(Deserialize)]
struct ImageItem {
    url: Option<String>,
    b64_json: Option<String>,
}

#[async_trait]
impl ArticleEmbeddingProvider for OpenAiClient {
    async fn generate_embedding(&self, content: &str) -> Result<Vec<f32>, AppError> {
        OpenAiClient::generate_embedding(self, content).await
    }
}

#[async_trait]
impl ArticleContextWriter for OpenAiClient {
    async fn update_with_context(&self, article: &Article) -> Result<String, AppError> {
        self.generate_text(
            "Revise the blog draft using its existing context. Return only the complete updated article body.",
            &article.draft_content,
        )
        .await
    }
}

#[async_trait]
impl insight::EmbeddingPort for OpenAiClient {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        OpenAiClient::generate_embedding(self, text).await
    }
}

#[async_trait]
impl source::EmbeddingPort for OpenAiClient {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        OpenAiClient::generate_embedding(self, text).await
    }
}

#[async_trait]
impl MlEmbeddingPort for OpenAiClient {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        OpenAiClient::generate_embedding(self, text).await
    }
}

#[async_trait]
impl TextGenerationPort for OpenAiClient {
    async fn generate_text(&self, instructions: &str, input: &str) -> Result<String, AppError> {
        OpenAiClient::generate_text(self, instructions, input).await
    }
}

#[async_trait]
impl Provider for OpenAiClient {
    fn model(&self) -> Model {
        Model::openai(
            COPILOT_MODEL_ID,
            COPILOT_API_MODEL,
            COPILOT_MAX_OUTPUT_TOKENS,
            true,
        )
    }

    fn system_message(&self) -> &str {
        "You are the blog writing copilot."
    }

    async fn stream_response(
        &self,
        cancellation: tokio_util::sync::CancellationToken,
        messages: Vec<LlmMessage>,
        tools: Vec<Arc<dyn Tool>>,
    ) -> Result<tokio::sync::mpsc::Receiver<ProviderEvent>, ProviderError> {
        if !self.is_configured() {
            return Err(ProviderError::Request(
                "OpenAI API key is not configured".to_owned(),
            ));
        }
        let tool_names = tools
            .iter()
            .map(|tool| tool.info().name)
            .collect::<Vec<_>>();
        let instructions = copilot_prompt(&tool_names);
        let input = response_input(&messages);
        let response_tools = tools
            .iter()
            .map(|tool| {
                let info = tool.info();
                ResponseTool {
                    r#type: "function",
                    name: info.name,
                    description: info.description,
                    parameters: serde_json::json!({
                        "type": "object",
                        "properties": info.parameters,
                        "required": info.required,
                        "additionalProperties": false,
                    }),
                    strict: true,
                }
            })
            .collect();
        let request = self
            .client
            .post(format!("{}/responses", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .json(&ResponseRequest {
                model: COPILOT_API_MODEL,
                instructions,
                input,
                store: false,
                stream: true,
                tools: response_tools,
                max_output_tokens: Some(COPILOT_MAX_OUTPUT_TOKENS),
                reasoning: Some(ResponseReasoning { effort: "medium" }),
            })
            .send();
        let response = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(ProviderError::Cancelled),
            response = request => response.map_err(|error| ProviderError::Request(error.to_string()))?,
        };
        if !response.status().is_success() {
            return Err(ProviderError::Request(format!(
                "OpenAI returned HTTP {}",
                response.status()
            )));
        }
        let (sender, receiver) = tokio::sync::mpsc::channel(100);
        tokio::spawn(run_response_stream(response, sender, cancellation));
        Ok(receiver)
    }
}

fn response_input(messages: &[LlmMessage]) -> serde_json::Value {
    let mut items = Vec::new();
    for (index, message) in messages.iter().enumerate() {
        match message.role {
            MessageRole::User | MessageRole::System => {
                let mut content = Vec::new();
                for part in &message.parts {
                    match part {
                        ContentPart::Text(text) if !text.text.is_empty() => {
                            content.push(serde_json::json!({
                                "type": "input_text",
                                "text": text.text,
                            }));
                        }
                        ContentPart::Binary(binary) => {
                            content.push(serde_json::json!({
                                "type": "input_image",
                                "image_url": format!(
                                    "data:{};base64,{}",
                                    binary.mime_type,
                                    STANDARD.encode(&binary.data)
                                ),
                                "detail": "auto",
                            }));
                        }
                        _ => {}
                    }
                }
                if !content.is_empty() {
                    let role = if message.role == MessageRole::System {
                        "system"
                    } else {
                        "user"
                    };
                    items.push(serde_json::json!({
                        "type": "message",
                        "role": role,
                        "content": content,
                    }));
                }
            }
            MessageRole::Assistant => {
                let text = message.text();
                if !text.is_empty() {
                    items.push(serde_json::json!({
                        "type": "message",
                        "id": output_message_id(message, index),
                        "role": "assistant",
                        "status": "completed",
                        "content": [{
                            "type": "output_text",
                            "text": text,
                            "annotations": [],
                        }],
                    }));
                }
                for call in message.tool_calls() {
                    if !call.name.is_empty() {
                        items.push(serde_json::json!({
                            "type": "function_call",
                            "call_id": call.id,
                            "name": call.name,
                            "arguments": call.input,
                        }));
                    }
                }
            }
            MessageRole::Tool => {
                for result in message.tool_results() {
                    items.push(serde_json::json!({
                        "type": "function_call_output",
                        "call_id": result.tool_call_id,
                        "output": result.content,
                    }));
                }
            }
        }
    }
    serde_json::Value::Array(items)
}

fn output_message_id(message: &LlmMessage, index: usize) -> String {
    if message.id.starts_with("msg_") {
        message.id.clone()
    } else if message.id.is_empty() {
        format!("msg_history_{index}")
    } else {
        format!("msg_{}", message.id.replace('-', "_"))
    }
}

struct StreamAccumulator {
    content: String,
    reasoning: String,
    pending_calls: HashMap<String, ToolCall>,
    tool_calls: Vec<ToolCall>,
    completed: bool,
}

impl StreamAccumulator {
    fn new() -> Self {
        Self {
            content: String::new(),
            reasoning: String::new(),
            pending_calls: HashMap::new(),
            tool_calls: Vec::new(),
            completed: false,
        }
    }
}

async fn run_response_stream(
    response: reqwest::Response,
    sender: tokio::sync::mpsc::Sender<ProviderEvent>,
    cancellation: tokio_util::sync::CancellationToken,
) {
    let mut bytes = response.bytes_stream();
    let mut buffer = Vec::new();
    let mut state = StreamAccumulator::new();
    loop {
        let chunk = tokio::select! {
            biased;
            () = cancellation.cancelled() => return,
            () = sender.closed() => return,
            chunk = bytes.next() => chunk,
        };
        let Some(chunk) = chunk else {
            break;
        };
        let chunk = match chunk {
            Ok(chunk) => chunk,
            Err(error) => {
                let _ = send_provider_event(
                    &sender,
                    &cancellation,
                    provider_error_event(error.to_string()),
                )
                .await;
                return;
            }
        };
        buffer.extend_from_slice(&chunk);
        while let Some(boundary) = buffer.windows(2).position(|window| window == b"\n\n") {
            let record = buffer.drain(..boundary + 2).collect::<Vec<_>>();
            let Ok(record) = std::str::from_utf8(&record) else {
                let _ = send_provider_event(
                    &sender,
                    &cancellation,
                    provider_error_event("OpenAI returned invalid UTF-8 SSE data"),
                )
                .await;
                return;
            };
            let data = record
                .lines()
                .filter_map(|line| line.strip_prefix("data:"))
                .map(str::trim_start)
                .collect::<Vec<_>>()
                .join("\n");
            if data.is_empty() || data == "[DONE]" {
                continue;
            }
            let value: serde_json::Value = match serde_json::from_str(&data) {
                Ok(value) => value,
                Err(error) => {
                    let _ = send_provider_event(
                        &sender,
                        &cancellation,
                        provider_error_event(error.to_string()),
                    )
                    .await;
                    return;
                }
            };
            if process_response_event(&sender, &cancellation, &mut state, &value)
                .await
                .is_err()
            {
                return;
            }
            if state.completed {
                return;
            }
        }
    }
    if !state.completed {
        let _ = send_provider_event(
            &sender,
            &cancellation,
            provider_error_event(ProviderError::MissingCompletion.to_string()),
        )
        .await;
    }
}

async fn process_response_event(
    sender: &tokio::sync::mpsc::Sender<ProviderEvent>,
    cancellation: &tokio_util::sync::CancellationToken,
    state: &mut StreamAccumulator,
    event: &serde_json::Value,
) -> Result<(), ()> {
    let event_type = event
        .get("type")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    match event_type {
        "response.output_text.delta" => {
            let delta = string_field(event, "delta");
            state.content.push_str(&delta);
            if !delta.is_empty() {
                send_provider_event(sender, cancellation, ProviderEvent::content_delta(delta))
                    .await?;
            }
        }
        "response.reasoning_summary_text.delta"
        | "response.reasoning_text.delta"
        | "response.reasoning.delta" => {
            let delta = event
                .get("delta")
                .or_else(|| event.get("text"))
                .and_then(serde_json::Value::as_str)
                .unwrap_or_default()
                .to_owned();
            state.reasoning.push_str(&delta);
            if !delta.is_empty() {
                send_provider_event(sender, cancellation, ProviderEvent::thinking_delta(delta))
                    .await?;
            }
        }
        "response.output_item.added" => {
            let item = &event["item"];
            if item.get("type").and_then(serde_json::Value::as_str) == Some("function_call") {
                let item_id = string_field(item, "id");
                let call = ToolCall {
                    id: string_field(item, "call_id"),
                    name: string_field(item, "name"),
                    input: string_field(item, "arguments"),
                    r#type: "function".to_owned(),
                    finished: false,
                    thought_signature: Vec::new(),
                };
                state.pending_calls.insert(item_id, call.clone());
                send_provider_event(
                    sender,
                    cancellation,
                    ProviderEvent {
                        event_type: ProviderEventType::ToolUseStart,
                        content: String::new(),
                        thinking: String::new(),
                        response: None,
                        tool_call: Some(call),
                        error: None,
                    },
                )
                .await?;
            }
        }
        "response.function_call_arguments.delta" => {
            let item_id = string_field(event, "item_id");
            if let Some(call) = state.pending_calls.get_mut(&item_id) {
                call.input.push_str(&string_field(event, "delta"));
                send_provider_event(
                    sender,
                    cancellation,
                    ProviderEvent {
                        event_type: ProviderEventType::ToolUseDelta,
                        content: String::new(),
                        thinking: String::new(),
                        response: None,
                        tool_call: Some(call.clone()),
                        error: None,
                    },
                )
                .await?;
            }
        }
        "response.function_call_arguments.done" => {
            let item_id = string_field(event, "item_id");
            if let Some(mut call) = state.pending_calls.remove(&item_id) {
                call.input = string_field(event, "arguments");
                call.finished = true;
                state.tool_calls.push(call.clone());
                send_provider_event(
                    sender,
                    cancellation,
                    ProviderEvent {
                        event_type: ProviderEventType::ToolUseStop,
                        content: String::new(),
                        thinking: String::new(),
                        response: None,
                        tool_call: Some(call),
                        error: None,
                    },
                )
                .await?;
            }
        }
        "response.completed" => {
            complete_response(sender, cancellation, state, &event["response"]).await?;
        }
        "response.failed" | "error" => {
            let error = event
                .pointer("/response/error/message")
                .or_else(|| event.get("message"))
                .and_then(serde_json::Value::as_str)
                .unwrap_or("OpenAI response failed");
            send_provider_event(sender, cancellation, provider_error_event(error)).await?;
            state.completed = true;
        }
        _ => {}
    }
    Ok(())
}

async fn complete_response(
    sender: &tokio::sync::mpsc::Sender<ProviderEvent>,
    cancellation: &tokio_util::sync::CancellationToken,
    state: &mut StreamAccumulator,
    response: &serde_json::Value,
) -> Result<(), ()> {
    let body: ResponseBody = serde_json::from_value(response.clone()).unwrap_or(ResponseBody {
        output: Vec::new(),
        status: String::new(),
        usage: ResponseUsage::default(),
    });
    for output in body.output {
        match output.output_type.as_str() {
            "function_call" => {
                if !state
                    .tool_calls
                    .iter()
                    .any(|call| call.id == output.call_id)
                {
                    state.tool_calls.push(ToolCall {
                        id: output.call_id,
                        name: output.name,
                        input: output.arguments,
                        r#type: "function".to_owned(),
                        finished: true,
                        thought_signature: Vec::new(),
                    });
                }
            }
            "reasoning" => {
                for part in output.content {
                    if part.content_type.contains("reasoning")
                        && let Some(text) = part.text
                        && !state.reasoning.contains(&text)
                    {
                        state.reasoning.push_str(&text);
                    }
                }
            }
            _ if state.content.is_empty() => {
                state.content.push_str(
                    &output
                        .content
                        .into_iter()
                        .filter_map(|part| part.text)
                        .collect::<String>(),
                );
            }
            _ => {}
        }
    }
    let cached = body.usage.input_tokens_details.cached_tokens;
    let usage = TokenUsage {
        input_tokens: body.usage.input_tokens.saturating_sub(cached),
        output_tokens: body.usage.output_tokens,
        cache_creation_tokens: 0,
        cache_read_tokens: cached,
    };
    let finish_reason = if !state.tool_calls.is_empty() {
        FinishReason::ToolUse
    } else if body.status == "incomplete" {
        FinishReason::MaxTokens
    } else {
        FinishReason::EndTurn
    };
    send_provider_event(
        sender,
        cancellation,
        ProviderEvent::complete(ProviderResponse {
            content: state.content.clone(),
            reasoning: state.reasoning.clone(),
            tool_calls: state.tool_calls.clone(),
            usage,
            finish_reason,
        }),
    )
    .await?;
    state.completed = true;
    Ok(())
}

fn string_field(value: &serde_json::Value, field: &str) -> String {
    value
        .get(field)
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default()
        .to_owned()
}

fn provider_error_event(error: impl Into<String>) -> ProviderEvent {
    ProviderEvent {
        event_type: ProviderEventType::Error,
        content: String::new(),
        thinking: String::new(),
        response: None,
        tool_call: None,
        error: Some(ProviderError::Request(error.into())),
    }
}

async fn send_provider_event(
    sender: &tokio::sync::mpsc::Sender<ProviderEvent>,
    cancellation: &tokio_util::sync::CancellationToken,
    event: ProviderEvent,
) -> Result<(), ()> {
    tokio::select! {
        biased;
        () = cancellation.cancelled() => Err(()),
        result = sender.send(event) => result.map_err(|_| ()),
    }
}
