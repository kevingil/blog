use std::{
    collections::BTreeMap,
    sync::{Arc, RwLock},
};

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value, json};
use uuid::Uuid;

use crate::error::AppError;

use super::super::TextGenerationService;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolInfo {
    pub name: String,
    pub description: String,
    pub parameters: BTreeMap<String, Value>,
    pub required: Vec<String>,
    /// Declarative safety property; the agent may execute a group concurrently
    /// only when every tool in the group opts in.
    pub parallel_safe: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ToolResponseType {
    Text,
    Image,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ArtifactHint {
    #[serde(rename = "type")]
    pub artifact_type: String,
    pub data: Map<String, Value>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolResponse {
    #[serde(rename = "type")]
    pub response_type: ToolResponseType,
    pub content: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub metadata: String,
    pub is_error: bool,
    #[serde(default, skip_serializing_if = "Map::is_empty")]
    pub result: Map<String, Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub artifact: Option<ArtifactHint>,
}

impl ToolResponse {
    pub fn text(content: impl Into<String>) -> Self {
        Self {
            response_type: ToolResponseType::Text,
            content: content.into(),
            metadata: String::new(),
            is_error: false,
            result: Map::new(),
            artifact: None,
        }
    }

    pub fn error(content: impl Into<String>) -> Self {
        Self {
            is_error: true,
            ..Self::text(content)
        }
    }

    pub fn structured(
        content: impl Into<String>,
        result: Map<String, Value>,
        artifact: Option<ArtifactHint>,
    ) -> Self {
        Self {
            result,
            artifact,
            ..Self::text(content)
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ToolCallRequest {
    pub id: String,
    pub name: String,
    pub input: String,
}

#[derive(Debug, Default)]
struct DocumentState {
    html: String,
    markdown: String,
}

#[derive(Clone)]
pub struct ToolContext {
    pub session_id: String,
    pub message_id: String,
    pub request_id: String,
    pub article_id: Option<Uuid>,
    document: Arc<RwLock<DocumentState>>,
    pub cancellation: CancellationToken,
}

use tokio_util::sync::CancellationToken;

impl ToolContext {
    pub fn new(
        session_id: impl Into<String>,
        message_id: impl Into<String>,
        request_id: impl Into<String>,
        article_id: Option<Uuid>,
        html: impl Into<String>,
        markdown: impl Into<String>,
        cancellation: CancellationToken,
    ) -> Self {
        Self {
            session_id: session_id.into(),
            message_id: message_id.into(),
            request_id: request_id.into(),
            article_id,
            document: Arc::new(RwLock::new(DocumentState {
                html: html.into(),
                markdown: unescape_markdown(&markdown.into()),
            })),
            cancellation,
        }
    }

    pub fn document_markdown(&self) -> Result<String, AppError> {
        self.document
            .read()
            .map(|state| state.markdown.clone())
            .map_err(|_| AppError::Internal)
    }

    pub fn document_html(&self) -> Result<String, AppError> {
        self.document
            .read()
            .map(|state| state.html.clone())
            .map_err(|_| AppError::Internal)
    }

    pub fn with_message_id(&self, message_id: impl Into<String>) -> Self {
        let mut context = self.clone();
        context.message_id = message_id.into();
        context
    }

    fn update_markdown(&self, markdown: String) -> Result<(), AppError> {
        self.document
            .write()
            .map_err(|_| AppError::Internal)?
            .markdown = markdown;
        Ok(())
    }
}

#[async_trait]
pub trait Tool: Send + Sync {
    fn info(&self) -> ToolInfo;
    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError>;
}

#[async_trait]
pub trait DraftSaver: Send + Sync {
    async fn update_draft_content(
        &self,
        article_id: Uuid,
        markdown_content: &str,
    ) -> Result<(), AppError>;
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct WebSearchResult {
    pub id: String,
    pub title: String,
    pub url: String,
    pub text: String,
    pub summary: String,
    pub author: String,
    pub published_date: String,
    pub highlights: Vec<String>,
    pub score: f64,
    pub favicon: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct WebSearchResponse {
    pub results: Vec<WebSearchResult>,
    pub request_id: String,
    pub resolved_search_type: String,
    pub cost_dollars: Option<Value>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnswerCitation {
    pub url: String,
    pub title: String,
    pub author: String,
    pub published_date: String,
    pub favicon: String,
    pub text: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AnswerResponse {
    pub answer: String,
    pub citations: Vec<AnswerCitation>,
    pub cost_dollars: Option<Value>,
}

#[async_trait]
pub trait ResearchPort: Send + Sync {
    fn is_configured(&self) -> bool;
    async fn search(&self, query: &str) -> Result<WebSearchResponse, AppError>;
    async fn answer(&self, question: &str) -> Result<AnswerResponse, AppError>;
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SourceResource {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: String,
    pub content: String,
    pub url: String,
    pub source_type: String,
    pub meta_data: BTreeMap<String, Value>,
    pub created_at: Option<chrono::DateTime<chrono::Utc>>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct SourceSelection {
    pub source_id: Option<Uuid>,
    pub title: String,
    pub url: String,
    pub source_type: String,
    pub excerpt_text: String,
    pub excerpt_id: String,
    pub content: String,
    pub origin_tool: String,
    pub origin_query: String,
    pub origin_question: String,
    pub author: String,
    pub published_date: String,
}

#[async_trait]
pub trait SourceResourcePort: Send + Sync {
    async fn create_web_source(
        &self,
        article_id: Uuid,
        query: &str,
        result: &WebSearchResult,
        request_id: &str,
    ) -> Result<SourceResource, AppError>;
    async fn list(&self, article_id: Uuid) -> Result<Vec<SourceResource>, AppError>;
    async fn search_similar(
        &self,
        article_id: Uuid,
        query: &str,
        limit: i64,
    ) -> Result<Vec<SourceResource>, AppError>;
    async fn select(
        &self,
        article_id: Uuid,
        selection: SourceSelection,
        request_id: &str,
    ) -> Result<SourceResource, AppError>;
}

#[async_trait]
impl SourceResourcePort for crate::core::source::SourceService {
    async fn create_web_source(
        &self,
        article_id: Uuid,
        query: &str,
        result: &WebSearchResult,
        request_id: &str,
    ) -> Result<SourceResource, AppError> {
        let meta_data = BTreeMap::from([(
            "resource".to_owned(),
            json!({
                "origin_tool": "search_web_sources",
                "origin_query": query,
                "usage_status": "available",
                "search_result_id": result.id,
                "author": result.author,
                "published_date": result.published_date,
                "created_in_turn": request_id,
            }),
        )]);
        let source = self
            .create(crate::core::source::CreateSourceRequest {
                article_id,
                title: result.title.clone(),
                content: result.text.clone(),
                url: result.url.clone(),
                source_type: "web_search".to_owned(),
                meta_data: Some(meta_data),
            })
            .await?;
        Ok(source.into())
    }

    async fn list(&self, article_id: Uuid) -> Result<Vec<SourceResource>, AppError> {
        Ok(self
            .get_by_article_id(article_id)
            .await?
            .into_iter()
            .map(Into::into)
            .collect())
    }

    async fn search_similar(
        &self,
        article_id: Uuid,
        query: &str,
        limit: i64,
    ) -> Result<Vec<SourceResource>, AppError> {
        Ok(
            crate::core::source::SourceService::search_similar(self, article_id, query, limit)
                .await?
                .into_iter()
                .map(Into::into)
                .collect(),
        )
    }

    async fn select(
        &self,
        article_id: Uuid,
        selection: SourceSelection,
        request_id: &str,
    ) -> Result<SourceResource, AppError> {
        let source = self
            .upsert_agent_resource(crate::core::source::AgentResourceSelection {
                article_id,
                source_id: selection.source_id,
                title: selection.title,
                content: selection.content,
                url: selection.url,
                source_type: selection.source_type,
                origin_tool: selection.origin_tool,
                origin_query: selection.origin_query,
                origin_question: selection.origin_question,
                author: selection.author,
                published_date: selection.published_date,
                selected_excerpt: selection.excerpt_text,
                selected_excerpt_id: selection.excerpt_id,
                request_id: request_id.to_owned(),
                usage_status: "used".to_owned(),
            })
            .await?;
        Ok(source.into())
    }
}

impl From<crate::core::source::Source> for SourceResource {
    fn from(source: crate::core::source::Source) -> Self {
        Self {
            id: source.id,
            article_id: source.article_id,
            title: source.title,
            content: source.content,
            url: source.url,
            source_type: source.source_type,
            meta_data: source.meta_data.unwrap_or_default(),
            created_at: source.created_at,
        }
    }
}

#[derive(Debug, Default)]
pub struct ReadDocumentTool;

#[async_trait]
impl Tool for ReadDocumentTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "read_document".to_owned(),
            description: "Read the full document with line numbers. Use line numbers to reference content for replace_lines. The sections array shows each heading with its line number.".to_owned(),
            parameters: BTreeMap::new(),
            required: Vec::new(),
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        _call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let markdown = context.document_markdown()?;
        let content = if markdown.is_empty() {
            context.document_html()?
        } else {
            markdown
        };
        let lines = if content.is_empty() {
            Vec::new()
        } else {
            content.lines().collect::<Vec<_>>()
        };
        let numbered = lines
            .iter()
            .enumerate()
            .map(|(index, line)| format!("{:4}| {line}", index + 1))
            .collect::<Vec<_>>()
            .join("\n");
        let sections = lines
            .iter()
            .enumerate()
            .filter_map(|(index, line)| {
                let trimmed = line.trim();
                let level = trimmed
                    .chars()
                    .take_while(|character| *character == '#')
                    .count();
                (level > 0 && level <= 6 && !trimmed[level..].trim().is_empty())
                    .then(|| json!({"heading": trimmed, "line": index + 1, "level": level}))
            })
            .collect::<Vec<_>>();
        let result = json!({
            "content": numbered,
            "total_lines": lines.len(),
            "total_chars": content.len(),
            "sections": sections,
            "tool_name": "read_document",
        });
        Ok(ToolResponse::text(
            serde_json::to_string(&result).map_err(|_| AppError::Internal)?,
        ))
    }
}

pub struct ReplaceLinesTool {
    draft_saver: Option<Arc<dyn DraftSaver>>,
}

impl ReplaceLinesTool {
    pub fn new(draft_saver: Option<Arc<dyn DraftSaver>>) -> Self {
        Self { draft_saver }
    }
}

#[derive(Debug, Deserialize)]
struct ReplaceLinesInput {
    start_line: usize,
    end_line: usize,
    #[serde(default)]
    new_content: String,
    reason: String,
}

#[async_trait]
impl Tool for ReplaceLinesTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "replace_lines".to_owned(),
            description: "Replace lines in the document by line number. Use read_document to see line numbers and section boundaries. Works for rewriting, insertion, and deletion.".to_owned(),
            parameters: BTreeMap::from([
                ("start_line".to_owned(), json!({"type": "number"})),
                ("end_line".to_owned(), json!({"type": "number"})),
                ("new_content".to_owned(), json!({"type": "string"})),
                ("reason".to_owned(), json!({"type": "string"})),
            ]),
            required: vec![
                "start_line".to_owned(),
                "end_line".to_owned(),
                "reason".to_owned(),
            ],
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: ReplaceLinesInput = match serde_json::from_str(&call.input) {
            Ok(input) => input,
            Err(_) => return Ok(ToolResponse::error("Invalid input format")),
        };
        if input.start_line < 1 || input.end_line < input.start_line {
            return Ok(ToolResponse::error(format!(
                "Invalid line range: start_line={}, end_line={}. Lines are 1-indexed and end_line must be >= start_line.",
                input.start_line, input.end_line
            )));
        }

        let document = context.document_markdown()?;
        let mut lines = if document.is_empty() {
            Vec::new()
        } else {
            document.split('\n').collect::<Vec<_>>()
        };
        if lines.is_empty() {
            if input.start_line != 1 || input.end_line != 1 {
                return Ok(ToolResponse::error(
                    "Document is empty. To create the first draft, replace line 1 through line 1 with new_content.",
                ));
            }
            if input.new_content.trim().is_empty() {
                return Ok(ToolResponse::error(
                    "new_content is required when creating content in an empty document.",
                ));
            }
        } else if input.start_line > lines.len() {
            return Ok(ToolResponse::error(format!(
                "start_line {} exceeds document length ({} lines). Call read_document to see current line numbers.",
                input.start_line,
                lines.len()
            )));
        }

        let end = input.end_line.min(lines.len().max(1));
        let old_content = if lines.is_empty() {
            String::new()
        } else {
            lines[input.start_line - 1..end].join("\n")
        };
        let new_markdown = if lines.is_empty() {
            input.new_content.clone()
        } else {
            let mut output = lines
                .drain(..input.start_line - 1)
                .map(str::to_owned)
                .collect::<Vec<_>>();
            if !input.new_content.is_empty() {
                output.extend(input.new_content.split('\n').map(str::to_owned));
            }
            output.extend(
                lines
                    .drain(end - (input.start_line - 1)..)
                    .map(str::to_owned),
            );
            output.join("\n")
        };
        context.update_markdown(new_markdown.clone())?;

        // Go treats persistence as best effort: the edit succeeds and remains
        // in the turn's working copy even if the database update fails.
        if let (Some(saver), Some(article_id)) = (&self.draft_saver, context.article_id)
            && let Err(error) = saver.update_draft_content(article_id, &new_markdown).await
        {
            tracing::warn!(%error, %article_id, "failed to persist copilot draft edit");
        }

        let result_value = json!({
            "old_str": old_content,
            "new_str": input.new_content,
            "new_markdown": new_markdown,
            "reason": input.reason,
            "tool_name": "replace_lines",
            "start_line": input.start_line,
            "end_line": end,
        });
        let result = result_value
            .as_object()
            .cloned()
            .ok_or(AppError::Internal)?;
        Ok(ToolResponse::structured(
            serde_json::to_string(&result_value).map_err(|_| AppError::Internal)?,
            result,
            Some(ArtifactHint {
                artifact_type: "diff".to_owned(),
                data: json!({
                    "original": old_content,
                    "proposed": input.new_content,
                    "reason": input.reason,
                })
                .as_object()
                .cloned()
                .ok_or(AppError::Internal)?,
            }),
        ))
    }
}

pub struct GenerateImagePromptTool {
    service: Arc<TextGenerationService>,
}

impl GenerateImagePromptTool {
    pub fn new(service: Arc<TextGenerationService>) -> Self {
        Self { service }
    }
}

#[async_trait]
impl Tool for GenerateImagePromptTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "generate_image_prompt".to_owned(),
            description: "Generate an image prompt based on document content".to_owned(),
            parameters: BTreeMap::from([("content".to_owned(), json!({"type": "string"}))]),
            required: vec!["content".to_owned()],
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        context
            .cancellation
            .run_until_cancelled(async {})
            .await
            .ok_or(AppError::Internal)?;
        let input: Value = match serde_json::from_str(&call.input) {
            Ok(input) => input,
            Err(_) => return Ok(ToolResponse::error("Invalid input format")),
        };
        let Some(content) = input.get("content").and_then(Value::as_str) else {
            return Ok(ToolResponse::error("content is required"));
        };
        if content.is_empty() {
            return Ok(ToolResponse::error("content is required"));
        }
        let prompt = self.service.generate_image_prompt(content).await?;
        let result_value = json!({
            "prompt": prompt,
            "tool_name": "generate_image_prompt",
        });
        Ok(ToolResponse::structured(
            serde_json::to_string(&result_value).map_err(|_| AppError::Internal)?,
            result_value
                .as_object()
                .cloned()
                .ok_or(AppError::Internal)?,
            Some(ArtifactHint {
                artifact_type: "image_prompt".to_owned(),
                data: json!({
                    "prompt": prompt,
                    "content_hint": content.chars().take(200).collect::<String>(),
                })
                .as_object()
                .cloned()
                .ok_or(AppError::Internal)?,
            }),
        ))
    }
}

pub struct AskQuestionTool {
    research: Arc<dyn ResearchPort>,
}

impl AskQuestionTool {
    pub fn new(research: Arc<dyn ResearchPort>) -> Self {
        Self { research }
    }
}

#[async_trait]
impl Tool for AskQuestionTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "ask_question".to_owned(),
            description: "PRIMARY research tool. Ask a factual question and get a web-sourced answer with citations. Use before search_web_sources. Be specific: include names, dates, metrics.".to_owned(),
            parameters: BTreeMap::from([(
                "question".to_owned(),
                json!({"type": "string", "description": "A specific question with names, dates, or metrics for best results."}),
            )]),
            required: vec!["question".to_owned()],
            parallel_safe: true,
        }
    }

    async fn run(
        &self,
        _context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: Value = serde_json::from_str(&call.input)
            .map_err(|_| AppError::InvalidInput("Invalid input format".to_owned()))?;
        let question = input
            .get("question")
            .and_then(Value::as_str)
            .filter(|question| !question.is_empty())
            .ok_or_else(|| AppError::InvalidInput("question is required".to_owned()))?;
        if !self.research.is_configured() {
            return Err(AppError::External);
        }
        let answer = self.research.answer(question).await?;
        let citations = answer
            .citations
            .iter()
            .map(|citation| {
                let mut value = json!({
                    "url": citation.url,
                    "title": citation.title,
                });
                if let Some(object) = value.as_object_mut() {
                    insert_nonempty(object, "author", &citation.author);
                    insert_nonempty(object, "published_date", &citation.published_date);
                    insert_nonempty(object, "favicon", &citation.favicon);
                    if !citation.text.is_empty() {
                        object.insert(
                            "text_preview".to_owned(),
                            Value::String(citation.text.chars().take(300).collect()),
                        );
                    }
                }
                value
            })
            .collect::<Vec<_>>();
        let mut result = json!({
            "answer": answer.answer,
            "citations": citations,
            "question": question,
            "citation_count": citations.len(),
            "tool_name": "ask_question",
        });
        if let Some(cost) = answer.cost_dollars
            && let Some(object) = result.as_object_mut()
        {
            object.insert("cost_info".to_owned(), cost);
        }
        structured_artifact(
            result,
            "answer",
            json!({
                "answer": answer.answer,
                "citations": citations,
                "question": question,
            }),
        )
    }
}

pub struct SearchWebSourcesTool {
    research: Arc<dyn ResearchPort>,
    sources: Arc<dyn SourceResourcePort>,
}

impl SearchWebSourcesTool {
    pub fn new(research: Arc<dyn ResearchPort>, sources: Arc<dyn SourceResourcePort>) -> Self {
        Self { research, sources }
    }
}

#[async_trait]
impl Tool for SearchWebSourcesTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "search_web_sources".to_owned(),
            description: "Broad web search returning multiple source documents. Use ONLY when ask_question doesn't cover the topic broadly enough. Creates citable sources automatically.".to_owned(),
            parameters: BTreeMap::from([
                ("query".to_owned(), json!({"type": "string"})),
                (
                    "create_sources".to_owned(),
                    json!({"type": ["boolean", "null"]}),
                ),
            ]),
            required: vec!["query".to_owned()],
            parallel_safe: true,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: Value = serde_json::from_str(&call.input)
            .map_err(|_| AppError::InvalidInput("Invalid input format".to_owned()))?;
        let query = input
            .get("query")
            .and_then(Value::as_str)
            .filter(|query| !query.is_empty())
            .ok_or_else(|| AppError::InvalidInput("query is required".to_owned()))?;
        if !self.research.is_configured() {
            return Err(AppError::External);
        }
        let mut create_sources = input
            .get("create_sources")
            .and_then(Value::as_bool)
            .unwrap_or(true);
        if context.article_id.is_none() {
            create_sources = false;
        }
        let response = self.research.search(query).await?;
        let search_results = response
            .results
            .iter()
            .map(web_result_json)
            .collect::<Vec<_>>();
        let mut sources_created = Vec::new();
        let mut attempted = 0;
        if create_sources {
            let article_id = context.article_id.ok_or(AppError::Internal)?;
            for result in &response.results {
                if result.text.is_empty() {
                    continue;
                }
                attempted += 1;
                if let Ok(source) = self
                    .sources
                    .create_web_source(article_id, query, result, &context.request_id)
                    .await
                {
                    sources_created.push(json!({
                        "source_id": source.id,
                        "original_title": result.title,
                        "original_url": result.url,
                        "source_created": true,
                        "search_result_id": result.id,
                        "content_length": result.text.len(),
                        "source_type": "web_search",
                        "search_query": query,
                    }));
                }
            }
        }
        let successful = sources_created.len();
        let mut result = json!({
            "search_results": search_results,
            "sources_created": sources_created,
            "query": query,
            "total_found": response.results.len(),
            "results_processed": response.results.len(),
            "sources_attempted": attempted,
            "sources_successful": successful,
            "tool_name": "search_web_sources",
            "exa_request_id": response.request_id,
            "search_type": response.resolved_search_type,
            "message": format!("Found {} search results", response.results.len()),
        });
        if let Some(cost) = response.cost_dollars
            && let Some(object) = result.as_object_mut()
        {
            object.insert("cost_info".to_owned(), cost);
        }
        structured_artifact(
            result,
            "sources",
            json!({
                "search_results": search_results,
                "sources_created": sources_created,
                "query": query,
                "total_found": response.results.len(),
                "sources_successful": successful,
            }),
        )
    }
}

pub struct GetRelevantSourcesTool {
    sources: Arc<dyn SourceResourcePort>,
}

impl GetRelevantSourcesTool {
    pub fn new(sources: Arc<dyn SourceResourcePort>) -> Self {
        Self { sources }
    }
}

#[async_trait]
impl Tool for GetRelevantSourcesTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "get_relevant_sources".to_owned(),
            description:
                "Find relevant source chunks based on a query to provide context for document rewriting"
                    .to_owned(),
            parameters: BTreeMap::from([
                ("query".to_owned(), json!({"type": "string"})),
                ("limit".to_owned(), json!({"type": ["number", "null"]})),
            ]),
            required: vec!["query".to_owned()],
            parallel_safe: true,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: Value = serde_json::from_str(&call.input)
            .map_err(|_| AppError::InvalidInput("Invalid input format".to_owned()))?;
        let query = input
            .get("query")
            .and_then(Value::as_str)
            .filter(|query| !query.is_empty())
            .ok_or_else(|| AppError::InvalidInput("query is required".to_owned()))?;
        let limit = input.get("limit").and_then(Value::as_i64).unwrap_or(5);
        let Some(article_id) = context.article_id else {
            return structured_result(json!({
                "relevant_sources": [],
                "query": query,
                "total_found": 0,
                "tool_name": "get_relevant_sources",
                "warning": "No article ID available - returned empty sources",
            }));
        };
        let sources = self
            .sources
            .search_similar(article_id, query, limit)
            .await?;
        let relevant_sources = sources
            .iter()
            .map(|source| {
                json!({
                    "source_id": source.id,
                    "source_title": source.title,
                    "source_url": source.url,
                    "text_chunk": source.content,
                    "excerpt_text": source.content,
                    "source_type": source.source_type,
                    "excerpt_id": format!("{}:0", source.id),
                })
            })
            .collect::<Vec<_>>();
        let inventory = sources.iter().map(source_resource_json).collect::<Vec<_>>();
        structured_artifact(
            json!({
                "relevant_sources": relevant_sources,
                "source_inventory": inventory,
                "query": query,
                "total_found": relevant_sources.len(),
                "tool_name": "get_relevant_sources",
            }),
            "sources",
            json!({
                "sources": relevant_sources,
                "source_inventory": inventory,
                "query": query,
                "total_found": relevant_sources.len(),
                "inventory_count": sources.len(),
            }),
        )
    }
}

pub struct SelectSourcesForEditTool {
    sources: Arc<dyn SourceResourcePort>,
}

impl SelectSourcesForEditTool {
    pub fn new(sources: Arc<dyn SourceResourcePort>) -> Self {
        Self { sources }
    }
}

#[derive(Debug, Deserialize)]
struct SelectSourcesInput {
    sources: Vec<SelectSourceInput>,
}

#[derive(Debug, Deserialize)]
struct SelectSourceInput {
    #[serde(default)]
    source_id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    url: String,
    #[serde(default)]
    source_type: String,
    excerpt_text: String,
    #[serde(default)]
    excerpt_id: String,
    #[serde(default)]
    content: String,
    #[serde(default)]
    origin_tool: String,
    #[serde(default)]
    origin_query: String,
    #[serde(default)]
    origin_question: String,
    #[serde(default)]
    author: String,
    #[serde(default)]
    published_date: String,
}

#[async_trait]
impl Tool for SelectSourcesForEditTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "select_sources_for_edit".to_owned(),
            description: "Persist selected sources for the pending edit and return the exact excerpts to use as edit context. Use this after research or get_relevant_sources and before replace_lines.".to_owned(),
            parameters: BTreeMap::from([(
                "sources".to_owned(),
                json!({
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "source_id": {"type": ["string", "null"]},
                            "title": {"type": ["string", "null"]},
                            "url": {"type": ["string", "null"]},
                            "source_type": {"type": ["string", "null"]},
                            "excerpt_text": {"type": "string"},
                            "excerpt_id": {"type": ["string", "null"]},
                            "content": {"type": ["string", "null"]},
                            "origin_tool": {"type": ["string", "null"]},
                            "origin_query": {"type": ["string", "null"]},
                            "origin_question": {"type": ["string", "null"]},
                            "author": {"type": ["string", "null"]},
                            "published_date": {"type": ["string", "null"]}
                        },
                        "required": ["excerpt_text"]
                    }
                }),
            )]),
            required: vec!["sources".to_owned()],
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: SelectSourcesInput = serde_json::from_str(&call.input)
            .map_err(|_| AppError::InvalidInput("Invalid input format".to_owned()))?;
        if input.sources.is_empty() {
            return Err(AppError::InvalidInput("sources is required".to_owned()));
        }
        let article_id = context
            .article_id
            .ok_or_else(|| AppError::InvalidInput("article_id is required".to_owned()))?;
        let mut selected = Vec::new();
        for source in input.sources {
            if source.excerpt_text.is_empty() {
                continue;
            }
            let source_id = if source.source_id.is_empty() {
                None
            } else {
                Some(Uuid::parse_str(&source.source_id).map_err(|_| {
                    AppError::InvalidInput(format!("invalid source_id {:?}", source.source_id))
                })?)
            };
            let saved = self
                .sources
                .select(
                    article_id,
                    SourceSelection {
                        source_id,
                        title: source.title,
                        url: source.url,
                        source_type: source.source_type,
                        excerpt_text: source.excerpt_text,
                        excerpt_id: source.excerpt_id,
                        content: source.content,
                        origin_tool: source.origin_tool,
                        origin_query: source.origin_query,
                        origin_question: source.origin_question,
                        author: source.author,
                        published_date: source.published_date,
                    },
                    &context.request_id,
                )
                .await?;
            selected.push(source_resource_json(&saved));
        }
        let inventory = self
            .sources
            .list(article_id)
            .await?
            .iter()
            .map(source_resource_json)
            .collect::<Vec<_>>();
        structured_artifact(
            json!({
                "selected_sources": selected,
                "selected_count": selected.len(),
                "source_inventory": inventory,
                "source_inventory_count": inventory.len(),
                "selected_context": format_selected_context(&selected),
                "inventory_context": format_inventory_context(&inventory),
                "tool_name": "select_sources_for_edit",
            }),
            "sources",
            json!({
                "selected_sources": selected,
                "source_inventory": inventory,
            }),
        )
    }
}

fn structured_result(result: Value) -> Result<ToolResponse, AppError> {
    let object = result.as_object().cloned().ok_or(AppError::Internal)?;
    Ok(ToolResponse::structured(
        serde_json::to_string(&result).map_err(|_| AppError::Internal)?,
        object,
        None,
    ))
}

fn structured_artifact(
    result: Value,
    artifact_type: &str,
    artifact_data: Value,
) -> Result<ToolResponse, AppError> {
    let object = result.as_object().cloned().ok_or(AppError::Internal)?;
    let artifact_data = artifact_data
        .as_object()
        .cloned()
        .ok_or(AppError::Internal)?;
    Ok(ToolResponse::structured(
        serde_json::to_string(&result).map_err(|_| AppError::Internal)?,
        object,
        Some(ArtifactHint {
            artifact_type: artifact_type.to_owned(),
            data: artifact_data,
        }),
    ))
}

fn insert_nonempty(object: &mut Map<String, Value>, key: &str, value: &str) {
    if !value.is_empty() {
        object.insert(key.to_owned(), Value::String(value.to_owned()));
    }
}

fn web_result_json(result: &WebSearchResult) -> Value {
    let mut value = json!({
        "title": result.title,
        "url": result.url,
        "id": result.id,
        "published_date": result.published_date,
        "author": result.author,
        "summary": result.summary,
        "has_full_text": !result.text.is_empty(),
    });
    if let Some(object) = value.as_object_mut() {
        if !result.highlights.is_empty() {
            object.insert(
                "highlights".to_owned(),
                serde_json::to_value(&result.highlights).unwrap_or_default(),
            );
        }
        if !result.text.is_empty() {
            object.insert(
                "text_preview".to_owned(),
                Value::String(result.text.chars().take(500).collect()),
            );
            object.insert("text_length".to_owned(), Value::from(result.text.len()));
        }
    }
    value
}

fn source_resource_json(source: &SourceResource) -> Value {
    json!({
        "source_id": source.id,
        "title": source.title,
        "url": source.url,
        "source_type": source.source_type,
        "preview": source.content.chars().take(220).collect::<String>(),
        "created_at": source.created_at,
    })
}

fn format_selected_context(sources: &[Value]) -> String {
    let mut output = String::from("Selected Sources For This Edit:\n");
    for source in sources {
        let id = source
            .get("source_id")
            .and_then(Value::as_str)
            .unwrap_or_default();
        let title = source
            .get("title")
            .and_then(Value::as_str)
            .unwrap_or("(untitled source)");
        output.push_str(&format!("- [{id}] {title}\n"));
        if let Some(preview) = source.get("preview").and_then(Value::as_str) {
            output.push_str("  excerpt:\n");
            output.push_str(preview);
            output.push('\n');
        }
    }
    output.trim().to_owned()
}

fn format_inventory_context(sources: &[Value]) -> String {
    let mut output = String::from("Available Sources:\n");
    for source in sources {
        let id = source
            .get("source_id")
            .and_then(Value::as_str)
            .unwrap_or_default();
        let title = source
            .get("title")
            .and_then(Value::as_str)
            .unwrap_or("(untitled source)");
        output.push_str(&format!("- [{id}] {title}\n"));
    }
    output.trim().to_owned()
}

fn unescape_markdown(markdown: &str) -> String {
    let mut output = markdown.to_owned();
    for escaped in ["*", "_", "[", "]", "#", ">", "-", "+", "~", "|", "`"] {
        output = output.replace(&format!("\\{escaped}"), escaped);
    }
    output
}
