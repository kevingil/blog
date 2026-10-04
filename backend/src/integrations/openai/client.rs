use std::{
    collections::{HashMap, HashSet},
    sync::Arc,
    time::Duration,
};

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
                ContentPart, FinishReason, HostedToolCall, LlmMessage, MessageRole, Model,
                Provider, ProviderError, ProviderEvent, ProviderEventType, ProviderResponse,
                TokenUsage, Tool, ToolCall, ToolResult, copilot_prompt,
            },
        },
        source,
    },
    error::AppError,
};

const DEFAULT_BASE_URL: &str = "https://api.openai.com/v1";
const DEFAULT_EMBEDDING_MODEL: &str = "text-embedding-3-small";
const DEFAULT_GENERATION_MODEL: &str = "gpt-5-2025-08-07";
const COPILOT_MODEL_ID: &str = "gpt-6.1-sol";
const COPILOT_API_MODEL: &str = "gpt-6.1-sol";
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
    provider_model: Model,
    system_message: Option<String>,
    max_output_tokens: i64,
    reasoning_effort: Option<String>,
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
            provider_model: Model::openai(
                COPILOT_MODEL_ID,
                COPILOT_API_MODEL,
                COPILOT_MAX_OUTPUT_TOKENS,
                true,
            ),
            system_message: None,
            max_output_tokens: COPILOT_MAX_OUTPUT_TOKENS,
            reasoning_effort: Some("medium".to_owned()),
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

    pub fn with_provider_model(
        mut self,
        model: Model,
        system_message: impl Into<String>,
        reasoning_effort: Option<String>,
    ) -> Result<Self, AppError> {
        if model.api_model.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "provider API model must not be empty".to_owned(),
            ));
        }
        if model.default_max_tokens <= 0 {
            return Err(AppError::InvalidInput(
                "provider max tokens must be greater than zero".to_owned(),
            ));
        }
        let system_message = system_message.into();
        if system_message.trim().is_empty() {
            return Err(AppError::InvalidInput(
                "provider system message must not be empty".to_owned(),
            ));
        }
        self.max_output_tokens = model.default_max_tokens;
        self.provider_model = model;
        self.system_message = Some(system_message);
        self.reasoning_effort = reasoning_effort;
        Ok(self)
    }

    pub fn is_configured(&self) -> bool {
        !self.api_key.expose_secret().is_empty()
    }

    pub(crate) fn live_credentials(&self) -> (&str, &str) {
        (&self.base_url, self.api_key.expose_secret())
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

    pub async fn generate_provider_text(&self, input: &str) -> Result<String, AppError> {
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
            .json(&ResponseRequest {
                model: &self.provider_model.api_model,
                instructions: self.system_message().to_owned(),
                input: serde_json::Value::String(input.to_owned()),
                store: (self.provider_model.provider.0
                    != crate::core::ml::llm::ModelProvider::GROQ)
                    .then_some(false),
                stream: false,
                tools: Vec::new(),
                max_output_tokens: Some(self.max_output_tokens),
                reasoning: self
                    .reasoning_effort
                    .clone()
                    .map(|effort| ResponseReasoning { effort }),
            })
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
        if text.trim().is_empty() {
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

    pub async fn transcribe_audio(
        &self,
        audio: &[u8],
        mime_type: &str,
    ) -> Result<String, AppError> {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        if audio.is_empty() {
            return Err(AppError::InvalidInput("audio is empty".to_owned()));
        }
        let filename = audio_filename(mime_type);
        let part = reqwest::multipart::Part::bytes(audio.to_vec())
            .file_name(filename)
            .mime_str(if mime_type.trim().is_empty() {
                "application/octet-stream"
            } else {
                mime_type
            })
            .map_err(|_| AppError::InvalidInput("invalid audio mime type".to_owned()))?;
        let form = reqwest::multipart::Form::new()
            .text("model", "whisper-1")
            .part("file", part);
        let response = self
            .client
            .post(format!("{}/audio/transcriptions", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .multipart(form)
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let body: TranscriptionResponse = response.json().await.map_err(|_| AppError::External)?;
        if body.text.trim().is_empty() {
            Err(AppError::External)
        } else {
            Ok(body.text)
        }
    }

    pub async fn synthesize_speech(
        &self,
        text: &str,
    ) -> Result<crate::core::speech::SpeechAudio, AppError> {
        if !self.is_configured() {
            return Err(AppError::External);
        }
        if text.trim().is_empty() {
            return Err(AppError::InvalidInput("speech text is empty".to_owned()));
        }
        let response = self
            .client
            .post(format!("{}/audio/speech", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .json(&SpeechRequest {
                model: "gpt-4o-mini-tts",
                input: text,
                voice: "alloy",
                format: "wav",
            })
            .send()
            .await
            .map_err(|_| AppError::External)?;
        if !response.status().is_success() {
            return Err(AppError::External);
        }
        let bytes = response.bytes().await.map_err(|_| AppError::External)?;
        if bytes.is_empty() {
            return Err(AppError::External);
        }
        Ok(crate::core::speech::SpeechAudio {
            bytes: bytes.to_vec(),
            mime_type: "audio/wav".to_owned(),
        })
    }
}

#[async_trait]
impl crate::core::speech::SpeechPort for OpenAiClient {
    async fn transcribe(&self, audio: &[u8], mime_type: &str) -> Result<String, AppError> {
        self.transcribe_audio(audio, mime_type).await
    }

    async fn synthesize(&self, text: &str) -> Result<crate::core::speech::SpeechAudio, AppError> {
        self.synthesize_speech(text).await
    }
}

fn audio_filename(mime_type: &str) -> &'static str {
    if mime_type.contains("wav") {
        "speech.wav"
    } else if mime_type.contains("mpeg") || mime_type.contains("mp3") {
        "speech.mp3"
    } else if mime_type.contains("ogg") {
        "speech.ogg"
    } else {
        "speech.webm"
    }
}

#[derive(Deserialize)]
struct TranscriptionResponse {
    text: String,
}

#[derive(Serialize)]
struct SpeechRequest<'a> {
    model: &'a str,
    input: &'a str,
    voice: &'a str,
    format: &'a str,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    store: Option<bool>,
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    stream: bool,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    tools: Vec<serde_json::Value>,
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
            store: Some(false),
            stream: false,
            tools: Vec::new(),
            max_output_tokens: None,
            reasoning: None,
        }
    }
}

#[derive(Serialize)]
struct ResponseReasoning {
    effort: String,
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
        self.provider_model.clone()
    }

    fn system_message(&self) -> &str {
        self.system_message
            .as_deref()
            .unwrap_or("You are the blog writing copilot.")
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
        let hosted = hosted_tools_enabled(&self.base_url);
        let mut tool_names = tools
            .iter()
            .map(|tool| tool.info().name)
            .collect::<Vec<_>>();
        if !tool_names.iter().any(|existing| existing == "web_search") {
            tool_names.push("web_search".to_owned());
        }
        if hosted && !tool_names.iter().any(|existing| existing == "sandbox") {
            tool_names.push("sandbox".to_owned());
        }
        let instructions = self
            .system_message
            .clone()
            .unwrap_or_else(|| copilot_prompt(&tool_names));
        let input = response_input(&messages);
        let response_tools = response_tools(&tools, hosted);
        let request = self
            .client
            .post(format!("{}/responses", self.base_url))
            .bearer_auth(self.api_key.expose_secret())
            .timeout(Duration::from_secs(300))
            .json(&ResponseRequest {
                model: &self.provider_model.api_model,
                instructions,
                input,
                store: (self.provider_model.provider.0
                    != crate::core::ml::llm::ModelProvider::GROQ)
                    .then_some(false),
                stream: true,
                tools: response_tools,
                max_output_tokens: Some(self.max_output_tokens),
                reasoning: self
                    .reasoning_effort
                    .clone()
                    .map(|effort| ResponseReasoning { effort }),
            })
            .send();
        let response = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(ProviderError::Cancelled),
            response = request => response.map_err(|error| ProviderError::Request(error.to_string()))?,
        };
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(ProviderError::Request(provider_status_error(
                "OpenAI",
                status.as_u16(),
                &body,
            )));
        }
        let (sender, receiver) = tokio::sync::mpsc::channel(100);
        tokio::spawn(run_response_stream(response, sender, cancellation));
        Ok(receiver)
    }
}

fn hosted_tools_enabled(base_url: &str) -> bool {
    base_url.to_ascii_lowercase().contains("api.openai.com")
}

fn response_tools(tools: &[Arc<dyn Tool>], hosted: bool) -> Vec<serde_json::Value> {
    let mut items = Vec::new();
    // Ordinary lookup uses OpenAI web search. Code interpreter stays on the hosted OpenAI API.
    items.push(serde_json::json!({"type": "web_search"}));
    if hosted {
        items.push(serde_json::json!({
            "type": "code_interpreter",
            "container": {"type": "auto"}
        }));
    }
    for tool in tools {
        let info = tool.info();
        if matches!(
            info.name.as_str(),
            "web_search" | "sandbox" | "code_interpreter"
        ) {
            continue;
        }
        items.push(serde_json::json!({
            "type": "function",
            "name": info.name,
            "description": info.description,
            "parameters": strict_parameters(&info.parameters, &info.required),
            "strict": true,
        }));
    }
    items
}

fn strict_parameters(
    parameters: &std::collections::BTreeMap<String, serde_json::Value>,
    required: &[String],
) -> serde_json::Value {
    let mut properties = serde_json::Map::new();
    for (name, schema) in parameters {
        properties.insert(name.clone(), schema.clone());
    }
    strict_schema(&serde_json::json!({
        "type": "object",
        "properties": properties,
        "required": required,
    }))
}

/// OpenAI strict function schemas reject a request with HTTP 400 unless every
/// object lists all of its properties in `required` and sets
/// `additionalProperties` to false. Optional fields stay optional by accepting
/// null.
fn strict_schema(schema: &serde_json::Value) -> serde_json::Value {
    let Some(object) = schema.as_object() else {
        return schema.clone();
    };
    let mut object = object.clone();
    if let Some(items) = object.get("items").cloned() {
        object.insert("items".to_owned(), strict_schema(&items));
    }
    for key in ["anyOf", "oneOf", "allOf"] {
        if let Some(options) = object.get(key).and_then(serde_json::Value::as_array) {
            let next = options.iter().map(strict_schema).collect();
            object.insert(key.to_owned(), serde_json::Value::Array(next));
        }
    }
    if schema_is_object(&object) {
        let properties = object
            .get("properties")
            .and_then(serde_json::Value::as_object)
            .cloned()
            .unwrap_or_default();
        let mut required = object
            .get("required")
            .and_then(serde_json::Value::as_array)
            .into_iter()
            .flatten()
            .filter_map(|item| item.as_str().map(ToOwned::to_owned))
            .filter(|name| properties.contains_key(name))
            .collect::<Vec<_>>();
        let mut next_properties = serde_json::Map::new();
        for (name, child) in properties {
            let child = if required.iter().any(|existing| existing == &name) {
                child
            } else {
                required.push(name.clone());
                make_nullable(child)
            };
            next_properties.insert(name, strict_schema(&child));
        }
        object.insert(
            "properties".to_owned(),
            serde_json::Value::Object(next_properties),
        );
        object.insert(
            "required".to_owned(),
            serde_json::Value::Array(
                required
                    .into_iter()
                    .map(serde_json::Value::String)
                    .collect(),
            ),
        );
        object.insert(
            "additionalProperties".to_owned(),
            serde_json::Value::Bool(false),
        );
        if !object.contains_key("type") {
            object.insert("type".to_owned(), serde_json::json!("object"));
        }
    }
    serde_json::Value::Object(object)
}

fn schema_is_object(object: &serde_json::Map<String, serde_json::Value>) -> bool {
    if object.contains_key("properties") || object.contains_key("additionalProperties") {
        return true;
    }
    match object.get("type") {
        Some(serde_json::Value::String(name)) => name == "object",
        Some(serde_json::Value::Array(types)) => {
            types.iter().any(|item| item.as_str() == Some("object"))
        }
        _ => false,
    }
}

fn make_nullable(schema: serde_json::Value) -> serde_json::Value {
    if schema_allows_null(&schema) {
        return schema;
    }
    let Some(object) = schema.as_object() else {
        return schema;
    };
    let mut object = object.clone();
    match object.get("type").cloned() {
        Some(serde_json::Value::String(name)) => {
            object.insert("type".to_owned(), serde_json::json!([name, "null"]));
            serde_json::Value::Object(object)
        }
        Some(serde_json::Value::Array(mut types)) => {
            types.push(serde_json::json!("null"));
            object.insert("type".to_owned(), serde_json::Value::Array(types));
            serde_json::Value::Object(object)
        }
        _ => serde_json::json!({
            "anyOf": [schema, {"type": "null"}]
        }),
    }
}

fn schema_allows_null(schema: &serde_json::Value) -> bool {
    match schema.get("type") {
        Some(serde_json::Value::String(name)) => name == "null",
        Some(serde_json::Value::Array(types)) => {
            types.iter().any(|item| item.as_str() == Some("null"))
        }
        _ => schema
            .get("anyOf")
            .and_then(serde_json::Value::as_array)
            .is_some_and(|options| {
                options.iter().any(|option| {
                    option.get("type").and_then(serde_json::Value::as_str) == Some("null")
                })
            }),
    }
}

fn provider_status_error(provider: &str, status: u16, body: &str) -> String {
    let detail = body.split_whitespace().collect::<Vec<_>>().join(" ");
    let detail = truncate_detail(&detail, 400);
    if detail.is_empty() {
        format!("{provider} returned HTTP {status}")
    } else {
        format!("{provider} returned HTTP {status}: {detail}")
    }
}

fn truncate_detail(value: &str, limit: usize) -> String {
    let mut characters = value.chars();
    let prefix = characters.by_ref().take(limit).collect::<String>();
    if characters.next().is_some() {
        format!("{prefix}...")
    } else {
        prefix
    }
}

fn response_input(messages: &[LlmMessage]) -> serde_json::Value {
    let answered = answered_tool_call_ids(messages);
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
                    if call.r#type == "hosted" || call.name.is_empty() {
                        continue;
                    }
                    if answered.contains(&call.id) {
                        let arguments = if call.input.trim().is_empty() || call.input == "null" {
                            "{}"
                        } else {
                            call.input.as_str()
                        };
                        items.push(serde_json::json!({
                            "type": "function_call",
                            "call_id": call.id,
                            "name": call.name,
                            "arguments": arguments,
                        }));
                    }
                }
            }
            MessageRole::Tool => {
                for result in message.tool_results() {
                    if let Some(replay) = hosted_replay(&result.content) {
                        items.push(replay);
                        continue;
                    }
                    if !answered.contains(&result.tool_call_id) {
                        continue;
                    }
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

fn hosted_replay(content: &str) -> Option<serde_json::Value> {
    let value: serde_json::Value = serde_json::from_str(content).ok()?;
    if value.get("hosted").and_then(serde_json::Value::as_bool) != Some(true) {
        return None;
    }
    let replay = value.get("replay")?.clone();
    replay.as_object().is_some().then_some(replay)
}

fn hosted_exchanges(items: &[serde_json::Value]) -> Vec<HostedToolCall> {
    let citations = citation_results(items);
    let search_count = items
        .iter()
        .filter(|item| {
            item.get("type").and_then(serde_json::Value::as_str) == Some("web_search_call")
        })
        .count();
    let mut exchanges = Vec::new();
    for item in items {
        let Some(mut exchange) = hosted_exchange(item) else {
            continue;
        };
        if exchange.call.name == "web_search" && search_count == 1 && citations_needed(&exchange) {
            attach_citations(&mut exchange, &citations);
        }
        exchanges.push(exchange);
    }
    exchanges
}

fn citations_needed(exchange: &HostedToolCall) -> bool {
    serde_json::from_str::<serde_json::Value>(&exchange.result.content)
        .ok()
        .and_then(|value| {
            value
                .get("search_results")
                .and_then(|results| results.as_array())
                .cloned()
        })
        .is_some_and(|results| results.is_empty())
}

fn attach_citations(exchange: &mut HostedToolCall, citations: &[serde_json::Value]) {
    if citations.is_empty() {
        return;
    }
    let Ok(mut value) = serde_json::from_str::<serde_json::Value>(&exchange.result.content) else {
        return;
    };
    let Some(object) = value.as_object_mut() else {
        return;
    };
    object.insert(
        "search_results".to_owned(),
        serde_json::Value::Array(citations.to_vec()),
    );
    object.insert(
        "total_found".to_owned(),
        serde_json::Value::from(citations.len()),
    );
    object.insert(
        "message".to_owned(),
        serde_json::Value::String(format!("Found {} results", citations.len())),
    );
    if let Ok(content) = serde_json::to_string(&value) {
        exchange.result.content = content;
    }
}

fn hosted_exchange(item: &serde_json::Value) -> Option<HostedToolCall> {
    let kind = item.get("type").and_then(serde_json::Value::as_str)?;
    let name = match kind {
        "web_search_call" => "web_search",
        "code_interpreter_call" => "sandbox",
        _ => return None,
    };
    let id = ["id", "call_id"]
        .iter()
        .find_map(|field| {
            item.get(field)
                .and_then(serde_json::Value::as_str)
                .filter(|value| !value.is_empty())
        })?
        .to_owned();
    let status = item
        .get("status")
        .and_then(serde_json::Value::as_str)
        .unwrap_or("completed");
    let failed = matches!(status, "failed" | "incomplete");
    let mut summary = match name {
        "web_search" => web_search_summary(item),
        _ => sandbox_summary(item),
    };
    if let Some(object) = summary.as_object_mut() {
        object.insert("hosted".to_owned(), serde_json::Value::Bool(true));
        object.insert("replay".to_owned(), item.clone());
        object.insert(
            "tool_name".to_owned(),
            serde_json::Value::String(name.to_owned()),
        );
    }
    let input = serde_json::json!({
        "query": summary.get("query").cloned().unwrap_or(serde_json::Value::Null),
        "code": summary.get("code").cloned().unwrap_or(serde_json::Value::Null),
    })
    .to_string();
    Some(HostedToolCall {
        call: ToolCall {
            id: id.clone(),
            name: name.to_owned(),
            input,
            r#type: "hosted".to_owned(),
            finished: true,
            thought_signature: Vec::new(),
        },
        result: ToolResult {
            tool_call_id: id,
            content: serde_json::to_string(&summary).unwrap_or_else(|_| "{}".to_owned()),
            metadata: String::new(),
            is_error: failed,
        },
    })
}

fn web_search_summary(item: &serde_json::Value) -> serde_json::Value {
    let query = item
        .pointer("/action/query")
        .and_then(serde_json::Value::as_str)
        .or_else(|| {
            item.pointer("/action/queries/0")
                .and_then(serde_json::Value::as_str)
        })
        .unwrap_or_default()
        .to_owned();
    let mut results = Vec::new();
    for key in ["sources", "results", "search_results"] {
        push_search_entries(&mut results, item.get(key));
    }
    push_search_entries(&mut results, item.pointer("/action/sources"));
    serde_json::json!({
        "query": query,
        "search_results": results,
        "total_found": results.len(),
        "message": format!("Found {} results", results.len()),
    })
}

fn sandbox_summary(item: &serde_json::Value) -> serde_json::Value {
    let code = item
        .get("code")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    let mut stdout = String::new();
    let mut stderr = String::new();
    if let Some(outputs) = item.get("outputs").and_then(serde_json::Value::as_array) {
        for output in outputs {
            let kind = output
                .get("type")
                .and_then(serde_json::Value::as_str)
                .unwrap_or_default();
            let logs = output
                .get("logs")
                .and_then(serde_json::Value::as_str)
                .or_else(|| output.get("text").and_then(serde_json::Value::as_str))
                .unwrap_or_default();
            match kind {
                "logs" | "log" | "stdout" => stdout.push_str(logs),
                "error" | "stderr" => {
                    if logs.is_empty() {
                        stderr.push_str(
                            output
                                .get("message")
                                .and_then(serde_json::Value::as_str)
                                .unwrap_or_default(),
                        );
                    } else {
                        stderr.push_str(logs);
                    }
                }
                _ if !logs.is_empty() && stdout.is_empty() => stdout.push_str(logs),
                _ => {}
            }
        }
    }
    serde_json::json!({
        "code": code,
        "stdout": stdout,
        "stderr": stderr,
        "exit_code": i32::from(!stderr.is_empty()),
    })
}

fn citation_results(items: &[serde_json::Value]) -> Vec<serde_json::Value> {
    let mut results = Vec::new();
    for item in items {
        let Some(content) = item.get("content").and_then(serde_json::Value::as_array) else {
            continue;
        };
        for part in content {
            let Some(annotations) = part
                .get("annotations")
                .and_then(serde_json::Value::as_array)
            else {
                continue;
            };
            for annotation in annotations {
                if annotation.get("type").and_then(serde_json::Value::as_str)
                    != Some("url_citation")
                {
                    continue;
                }
                push_search_entries(&mut results, Some(annotation));
            }
        }
    }
    results
}

fn push_search_entries(results: &mut Vec<serde_json::Value>, value: Option<&serde_json::Value>) {
    let Some(value) = value else {
        return;
    };
    let entries = value
        .as_array()
        .map(|entries| entries.as_slice())
        .unwrap_or(std::slice::from_ref(value));
    for entry in entries {
        if !entry.is_object() {
            continue;
        }
        let url = entry
            .get("url")
            .or_else(|| entry.get("link"))
            .and_then(serde_json::Value::as_str)
            .unwrap_or_default();
        if url.is_empty() {
            continue;
        }
        let title = entry
            .get("title")
            .or_else(|| entry.get("name"))
            .and_then(serde_json::Value::as_str)
            .unwrap_or(url);
        let summary = entry
            .get("snippet")
            .or_else(|| entry.get("summary"))
            .or_else(|| entry.get("text"))
            .and_then(serde_json::Value::as_str)
            .unwrap_or_default();
        results.push(serde_json::json!({
            "title": title,
            "url": url,
            "summary": summary,
        }));
    }
}

fn answered_tool_call_ids(messages: &[LlmMessage]) -> HashSet<String> {
    let mut calls = HashSet::new();
    let mut results = HashSet::new();
    for message in messages {
        for call in message.tool_calls() {
            if !call.id.is_empty() && !call.name.is_empty() {
                calls.insert(call.id);
            }
        }
        for result in message.tool_results() {
            if !result.tool_call_id.is_empty() {
                results.insert(result.tool_call_id);
            }
        }
    }
    calls.intersection(&results).cloned().collect()
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
    hosted: Vec<HostedToolCall>,
    completed: bool,
}

impl StreamAccumulator {
    fn new() -> Self {
        Self {
            content: String::new(),
            reasoning: String::new(),
            pending_calls: HashMap::new(),
            tool_calls: Vec::new(),
            hosted: Vec::new(),
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
            } else if item.get("type").and_then(serde_json::Value::as_str) == Some("reasoning") {
                for part in item
                    .get("content")
                    .and_then(serde_json::Value::as_array)
                    .into_iter()
                    .flatten()
                {
                    if part.get("type").and_then(serde_json::Value::as_str)
                        == Some("reasoning_text")
                    {
                        let text = part
                            .get("text")
                            .and_then(serde_json::Value::as_str)
                            .unwrap_or_default();
                        if !text.is_empty() {
                            state.reasoning.push_str(text);
                            send_provider_event(
                                sender,
                                cancellation,
                                ProviderEvent::thinking_delta(text.to_owned()),
                            )
                            .await?;
                        }
                    }
                }
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
    if let Some(items) = response.get("output").and_then(|value| value.as_array()) {
        state.hosted = hosted_exchanges(items);
    }
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
            hosted: state.hosted.clone(),
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

#[cfg(test)]
mod tests {
    use super::{provider_status_error, response_tools, strict_schema};
    use crate::core::ml::llm::{ReadDocumentTool, ReplaceLinesTool, Tool};
    use std::sync::Arc;

    fn assert_strict(schema: &serde_json::Value) -> Result<(), String> {
        if let Some(items) = schema.get("items") {
            assert_strict(items)?;
        }
        for key in ["anyOf", "oneOf", "allOf"] {
            if let Some(options) = schema.get(key).and_then(serde_json::Value::as_array) {
                for option in options {
                    assert_strict(option)?;
                }
            }
        }
        let is_object = schema.get("type").and_then(serde_json::Value::as_str) == Some("object")
            || schema
                .get("type")
                .and_then(serde_json::Value::as_array)
                .is_some_and(|types| types.iter().any(|item| item.as_str() == Some("object")))
            || schema.get("properties").is_some();
        if !is_object {
            return Ok(());
        }
        if schema.get("additionalProperties") != Some(&serde_json::Value::Bool(false)) {
            return Err(format!("missing additionalProperties: false in {schema}"));
        }
        let properties = schema
            .get("properties")
            .and_then(serde_json::Value::as_object)
            .ok_or_else(|| format!("object schema missing properties: {schema}"))?;
        let required = schema
            .get("required")
            .and_then(serde_json::Value::as_array)
            .ok_or_else(|| format!("object schema missing required: {schema}"))?;
        for name in properties.keys() {
            if !required.iter().any(|item| item.as_str() == Some(name)) {
                return Err(format!("property {name} is not required in {schema}"));
            }
            assert_strict(&properties[name])?;
        }
        Ok(())
    }

    #[test]
    fn strict_tool_schemas_list_every_property_including_nested_objects() {
        let tools: Vec<Arc<dyn Tool>> = vec![
            Arc::new(ReadDocumentTool),
            Arc::new(ReplaceLinesTool::new(None)),
        ];
        let schemas = response_tools(&tools, false);
        let replace = schemas
            .iter()
            .find(|tool| tool["name"] == "replace_lines")
            .expect("replace_lines schema");
        assert_eq!(replace["strict"], true);
        assert_strict(&replace["parameters"]).expect("replace_lines is strict");
        let required = replace["parameters"]["required"]
            .as_array()
            .expect("required");
        assert!(required.iter().any(|item| item == "new_content"));
        assert_eq!(
            replace["parameters"]["properties"]["new_content"]["type"],
            "string"
        );
        let hosted = response_tools(&tools, true);
        assert!(hosted.iter().any(|tool| tool["type"] == "web_search"));
        assert!(hosted.iter().any(|tool| {
            tool["type"] == "code_interpreter" && tool["container"]["type"] == "auto"
        }));
        assert!(
            hosted
                .iter()
                .all(|tool| tool["name"] != "web_search" && tool["name"] != "sandbox")
        );

        let nested = strict_schema(&serde_json::json!({
            "type": "object",
            "properties": {
                "sources": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "excerpt_text": {"type": "string"},
                            "title": {"type": "string"}
                        },
                        "required": ["excerpt_text"]
                    }
                },
                "limit": {"type": ["number", "null"]}
            },
            "required": ["sources"]
        }));
        assert_strict(&nested).expect("nested schema is strict");
        assert_eq!(
            nested["properties"]["sources"]["items"]["properties"]["title"]["type"],
            serde_json::json!(["string", "null"])
        );
        assert_eq!(
            nested["properties"]["limit"]["type"],
            serde_json::json!(["number", "null"])
        );
    }

    #[test]
    fn hosted_tool_history_replays_the_provider_item() {
        use crate::core::ml::llm::{ContentPart, LlmMessage, MessageRole, ToolCall, ToolResult};
        let replay = serde_json::json!({
            "type": "code_interpreter_call",
            "id": "ci_1",
            "status": "completed",
            "code": "print(6*7)",
            "outputs": [{"type": "logs", "logs": "42\n"}]
        });
        let messages = vec![
            LlmMessage::new(
                "session",
                MessageRole::Assistant,
                vec![ContentPart::ToolCall(ToolCall {
                    id: "ci_1".to_owned(),
                    name: "sandbox".to_owned(),
                    input: r#"{"query":null,"code":"print(6*7)"}"#.to_owned(),
                    r#type: "hosted".to_owned(),
                    finished: true,
                    thought_signature: Vec::new(),
                })],
                "",
            ),
            LlmMessage::new(
                "session",
                MessageRole::Tool,
                vec![ContentPart::ToolResult(ToolResult {
                    tool_call_id: "ci_1".to_owned(),
                    content: serde_json::json!({
                        "hosted": true,
                        "tool_name": "sandbox",
                        "stdout": "42\n",
                        "replay": replay,
                    })
                    .to_string(),
                    metadata: String::new(),
                    is_error: false,
                })],
                "",
            ),
        ];
        let input = super::response_input(&messages);
        let items = input.as_array().expect("input array");
        assert!(
            items
                .iter()
                .any(|item| { item["type"] == "code_interpreter_call" && item["id"] == "ci_1" })
        );
        assert!(items.iter().all(|item| {
            item["type"] != "function_call" && item["type"] != "function_call_output"
        }));
    }

    #[test]
    fn provider_status_error_keeps_the_response_body() {
        assert_eq!(
            provider_status_error("OpenAI", 400, ""),
            "OpenAI returned HTTP 400"
        );
        assert_eq!(
            provider_status_error(
                "OpenAI",
                400,
                "  Invalid schema for function \n 'replace_lines'  "
            ),
            "OpenAI returned HTTP 400: Invalid schema for function 'replace_lines'"
        );
    }
}
