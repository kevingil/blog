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
    title: String,
    sources: Vec<ArticleSourceView>,
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
                title: String::new(),
                sources: Vec::new(),
            })),
            cancellation,
        }
    }

    pub fn document_title(&self) -> Result<String, AppError> {
        self.document
            .read()
            .map(|state| state.title.clone())
            .map_err(|_| AppError::Internal)
    }

    pub fn set_document_title(&self, title: impl Into<String>) -> Result<(), AppError> {
        self.document.write().map_err(|_| AppError::Internal)?.title = title.into();
        Ok(())
    }

    pub fn document_sources(&self) -> Result<Vec<ArticleSourceView>, AppError> {
        self.document
            .read()
            .map(|state| state.sources.clone())
            .map_err(|_| AppError::Internal)
    }

    pub fn set_document_sources(&self, sources: Vec<ArticleSourceView>) -> Result<(), AppError> {
        self.document
            .write()
            .map_err(|_| AppError::Internal)?
            .sources = sources;
        Ok(())
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

    async fn update_draft_title(&self, article_id: Uuid, title: &str) -> Result<(), AppError> {
        let _ = (article_id, title);
        Ok(())
    }
}

/// A source object stored on the article, separate from the body text.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ArticleSourceView {
    pub id: Uuid,
    pub title: String,
    pub url: String,
    pub content: String,
}

#[derive(Debug, Clone, PartialEq)]
pub struct SourceEdit {
    pub id: Option<Uuid>,
    pub title: String,
    pub url: String,
    pub content: String,
    pub replace_content: bool,
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
    pub image: String,
}

/// First http(s) picture among the candidates. Data URLs and other schemes are skipped.
pub fn page_image<'a>(candidates: impl IntoIterator<Item = &'a str>) -> Option<String> {
    candidates.into_iter().find_map(http_image_url)
}

fn http_image_url(value: &str) -> Option<String> {
    let trimmed = value.trim();
    let scheme = trimmed.split_once(':')?.0;
    if !scheme.eq_ignore_ascii_case("http") && !scheme.eq_ignore_ascii_case("https") {
        return None;
    }
    let rest = trimmed.get(scheme.len() + 1..)?;
    if !rest.starts_with("//") || trimmed.chars().any(char::is_whitespace) {
        return None;
    }
    Some(trimmed.to_owned())
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

    /// Multi-source dig. `domain` limits results to one host when the tracker is a site.
    async fn deep_search(
        &self,
        query: &str,
        domain: Option<&str>,
    ) -> Result<WebSearchResponse, AppError> {
        let _ = domain;
        self.search(query).await
    }
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

    /// Add, update, or remove the article's source objects.
    async fn apply_edits(
        &self,
        article_id: Uuid,
        upserts: Vec<SourceEdit>,
        remove_ids: Vec<Uuid>,
    ) -> Result<Vec<SourceResource>, AppError> {
        let _ = (article_id, upserts, remove_ids);
        Err(AppError::InvalidInput(
            "updating sources is not available".to_owned(),
        ))
    }
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
                "origin_tool": "deep_research",
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

    async fn apply_edits(
        &self,
        article_id: Uuid,
        upserts: Vec<SourceEdit>,
        remove_ids: Vec<Uuid>,
    ) -> Result<Vec<SourceResource>, AppError> {
        let mut existing = self.get_by_article_id(article_id).await?;
        for id in &remove_ids {
            if !existing.iter().any(|source| source.id == *id) {
                return Err(AppError::NotFound);
            }
        }
        for edit in &upserts {
            if let Some(id) = edit.id
                && !existing.iter().any(|source| source.id == id)
            {
                return Err(AppError::NotFound);
            }
        }
        for id in &remove_ids {
            self.delete(*id).await?;
            existing.retain(|source| source.id != *id);
        }
        for edit in upserts {
            if let Some(id) = edit.id {
                let Some(current) = existing.iter().find(|source| source.id == id) else {
                    return Err(AppError::NotFound);
                };
                let mut request = crate::core::source::UpdateSourceRequest {
                    title: (!edit.title.trim().is_empty()).then(|| edit.title.clone()),
                    url: Some(edit.url.clone()),
                    ..crate::core::source::UpdateSourceRequest::default()
                };
                if edit.replace_content && !edit.content.trim().is_empty() {
                    request.content = Some(edit.content);
                }
                self.update(current.id, request).await?;
                continue;
            }
            let url_key = edit.url.trim().to_ascii_lowercase();
            if !url_key.is_empty()
                && let Some(current) = existing
                    .iter()
                    .find(|source| source.url.trim().eq_ignore_ascii_case(&url_key))
            {
                let mut request = crate::core::source::UpdateSourceRequest {
                    title: (!edit.title.trim().is_empty()).then(|| edit.title.clone()),
                    ..crate::core::source::UpdateSourceRequest::default()
                };
                if edit.replace_content && !edit.content.trim().is_empty() {
                    request.content = Some(edit.content);
                }
                self.update(current.id, request).await?;
                continue;
            }
            let title = if edit.title.trim().is_empty() {
                if edit.url.trim().is_empty() {
                    return Err(AppError::InvalidInput(
                        "a source needs a title or a url".to_owned(),
                    ));
                }
                edit.url.trim().to_owned()
            } else {
                edit.title.trim().to_owned()
            };
            let content = if edit.content.trim().is_empty() {
                title.clone()
            } else {
                edit.content
            };
            let created = self
                .create(crate::core::source::CreateSourceRequest {
                    article_id,
                    title,
                    content,
                    url: edit.url.trim().to_owned(),
                    source_type: if edit.url.trim().is_empty() {
                        "manual".to_owned()
                    } else {
                        "web".to_owned()
                    },
                    meta_data: None,
                })
                .await?;
            existing.push(created);
        }
        SourceResourcePort::list(self, article_id).await
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
            description: "Read the article. title is a separate field above the body; change it with set_title, never by writing it into the body. sources are separate objects; change them with update_sources, never by writing a Sources section into the body. content is the body with line numbers for replace_lines. The sections array shows each body heading with its line number.".to_owned(),
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
        let title = context.document_title()?;
        let sources = context.document_sources()?;
        let result = json!({
            "title": title,
            "title_field": "The title is stored separately from the body. Use set_title to change it. Do not write the title or a level-1 heading into the body.",
            "content": numbered,
            "total_lines": lines.len(),
            "total_chars": content.len(),
            "sections": sections,
            "sources": source_views_json(&sources),
            "sources_field": "Sources are objects stored on the article, not body text. Use update_sources to add, change, or remove them. Do not write a Sources, References, or Bibliography section into the body.",
            "tool_name": "read_document",
        });
        Ok(ToolResponse::text(
            serde_json::to_string(&result).map_err(|_| AppError::Internal)?,
        ))
    }
}

pub struct ReplaceLinesTool {
    draft_saver: Option<Arc<dyn DraftSaver>>,
    sources: Option<Arc<dyn SourceResourcePort>>,
}

impl ReplaceLinesTool {
    pub fn new(draft_saver: Option<Arc<dyn DraftSaver>>) -> Self {
        Self {
            draft_saver,
            sources: None,
        }
    }

    pub fn with_sources(mut self, sources: Arc<dyn SourceResourcePort>) -> Self {
        self.sources = Some(sources);
        self
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
            description: "Replace lines in the article body by line number. Use read_document to see line numbers and section boundaries. Works for rewriting, insertion, and deletion. The body must not include the title or a Sources section. Change the title with set_title and sources with update_sources. A leading level-1 heading is saved as the title and removed from the body. A trailing Sources, References, or Bibliography section is saved as source objects and removed from the body.".to_owned(),
            parameters: BTreeMap::from([
                ("start_line".to_owned(), json!({"type": "integer"})),
                ("end_line".to_owned(), json!({"type": "integer"})),
                (
                    "new_content".to_owned(),
                    json!({"type": "string", "description": "Replacement markdown for the inclusive line range. Use an empty string to delete those lines."}),
                ),
                ("reason".to_owned(), json!({"type": "string"})),
            ]),
            required: vec![
                "start_line".to_owned(),
                "end_line".to_owned(),
                "new_content".to_owned(),
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
        let committed = commit_draft(
            &context,
            self.draft_saver.as_ref(),
            self.sources.as_ref(),
            new_markdown,
        )
        .await?;

        let result_value = json!({
            "old_str": old_content,
            "new_str": input.new_content,
            "new_markdown": committed.markdown,
            "reason": input.reason,
            "tool_name": "replace_lines",
            "start_line": input.start_line,
            "end_line": end,
            "title_updated": committed.title_updated,
            "new_title": committed.new_title,
            "sources_updated": committed.sources_updated,
            "sources": source_views_json(&committed.sources),
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
            name: "deep_research".to_owned(),
            description: "Advanced multi-source research. Use when web_search is not enough and you need page text, highlights, or a search limited to one domain. Creates citable sources when an article is open.".to_owned(),
            parameters: BTreeMap::from([
                ("query".to_owned(), json!({"type": "string"})),
                (
                    "domain".to_owned(),
                    json!({"type": ["string", "null"], "description": "Optional host to keep the search inside, such as blog.rust-lang.org."}),
                ),
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
        let domain = input
            .get("domain")
            .and_then(Value::as_str)
            .filter(|domain| !domain.is_empty());
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
        let response = self.research.deep_search(query, domain).await?;
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
            "tool_name": "deep_research",
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
    #[serde(default, deserialize_with = "null_as_empty")]
    source_id: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    title: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    url: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    source_type: String,
    excerpt_text: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    excerpt_id: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    content: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    origin_tool: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    origin_query: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    origin_question: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    author: String,
    #[serde(default, deserialize_with = "null_as_empty")]
    published_date: String,
}

fn null_as_empty<'de, D>(deserializer: D) -> Result<String, D::Error>
where
    D: serde::Deserializer<'de>,
{
    Ok(Option::<String>::deserialize(deserializer)?.unwrap_or_default())
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

async fn save_working_markdown(
    context: &ToolContext,
    saver: Option<&Arc<dyn DraftSaver>>,
    markdown: String,
) -> Result<(), AppError> {
    context.update_markdown(markdown.clone())?;
    // Persistence is best effort: the edit stays in the turn's working copy
    // even when the database update fails.
    if let (Some(saver), Some(article_id)) = (saver, context.article_id)
        && let Err(error) = saver.update_draft_content(article_id, &markdown).await
    {
        tracing::warn!(%error, %article_id, "failed to persist copilot draft edit");
    }
    Ok(())
}

struct DraftCommit {
    markdown: String,
    title_updated: bool,
    new_title: String,
    sources_updated: bool,
    sources: Vec<ArticleSourceView>,
}

async fn commit_draft(
    context: &ToolContext,
    saver: Option<&Arc<dyn DraftSaver>>,
    sources: Option<&Arc<dyn SourceResourcePort>>,
    markdown: String,
) -> Result<DraftCommit, AppError> {
    let current_title = context.document_title()?;
    let normalized = normalize_draft(&markdown, &current_title);
    let mut title_updated = false;
    let mut new_title = current_title;
    if let Some(title) = normalized.title {
        context.set_document_title(title.clone())?;
        if let (Some(saver), Some(article_id)) = (saver, context.article_id)
            && let Err(error) = saver.update_draft_title(article_id, &title).await
        {
            tracing::warn!(%error, %article_id, "failed to persist copilot title edit");
        }
        new_title = title;
        title_updated = true;
    }
    save_working_markdown(context, saver, normalized.markdown.clone()).await?;

    let mut saved_sources = context.document_sources()?;
    let mut sources_updated = false;
    if !normalized.extracted_sources.is_empty()
        && let (Some(port), Some(article_id)) = (sources, context.article_id)
    {
        let saved = port
            .apply_edits(article_id, normalized.extracted_sources, Vec::new())
            .await?;
        saved_sources = saved.iter().map(ArticleSourceView::from).collect();
        context.set_document_sources(saved_sources.clone())?;
        sources_updated = true;
    }

    Ok(DraftCommit {
        markdown: normalized.markdown,
        title_updated,
        new_title,
        sources_updated,
        sources: saved_sources,
    })
}

pub struct SetTitleTool {
    draft_saver: Option<Arc<dyn DraftSaver>>,
}

impl SetTitleTool {
    pub fn new(draft_saver: Option<Arc<dyn DraftSaver>>) -> Self {
        Self { draft_saver }
    }
}

#[derive(Debug, Deserialize)]
struct SetTitleInput {
    title: String,
    #[serde(default)]
    reason: String,
}

#[async_trait]
impl Tool for SetTitleTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "set_title".to_owned(),
            description: "Set the article title. The title is a field above the body, not a line in the body. Do not write the title into the article with replace_lines or apply_patch.".to_owned(),
            parameters: BTreeMap::from([
                ("title".to_owned(), json!({"type": "string"})),
                ("reason".to_owned(), json!({"type": "string"})),
            ]),
            required: vec!["title".to_owned(), "reason".to_owned()],
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: SetTitleInput = match serde_json::from_str(&call.input) {
            Ok(input) => input,
            Err(_) => return Ok(ToolResponse::error("Invalid input format")),
        };
        let title = input.title.trim();
        if title.is_empty() || title.contains('\n') || title.contains('\r') {
            return Ok(ToolResponse::error("title must be a single non-empty line"));
        }
        let title = title.to_owned();
        context.set_document_title(title.clone())?;
        if let (Some(saver), Some(article_id)) = (self.draft_saver.as_ref(), context.article_id)
            && let Err(error) = saver.update_draft_title(article_id, &title).await
        {
            tracing::warn!(%error, %article_id, "failed to persist copilot title edit");
        }
        structured_result(json!({
            "new_title": title,
            "title_updated": true,
            "reason": input.reason,
            "tool_name": "set_title",
        }))
    }
}

pub struct UpdateSourcesTool {
    sources: Arc<dyn SourceResourcePort>,
}

impl UpdateSourcesTool {
    pub fn new(sources: Arc<dyn SourceResourcePort>) -> Self {
        Self { sources }
    }
}

#[derive(Debug, Deserialize)]
struct UpdateSourcesInput {
    #[serde(default)]
    sources: Vec<UpdateSourceItem>,
    #[serde(default)]
    remove_ids: Vec<String>,
}

#[derive(Debug, Deserialize)]
struct UpdateSourceItem {
    #[serde(default)]
    id: Option<String>,
    #[serde(default)]
    title: String,
    #[serde(default)]
    url: String,
    #[serde(default)]
    note: String,
}

#[async_trait]
impl Tool for UpdateSourcesTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "update_sources".to_owned(),
            description: "Edit the article's source objects. Sources are not part of the body. Pass sources to add or update, and remove_ids to delete. Use a null id to add a source. Use the id from read_document to update one. Do not write a Sources, References, or Bibliography section into the article text.".to_owned(),
            parameters: BTreeMap::from([
                (
                    "sources".to_owned(),
                    json!({
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "id": {"type": ["string", "null"], "description": "Existing source id. Null adds a source."},
                                "title": {"type": "string"},
                                "url": {"type": "string"},
                                "note": {"type": "string", "description": "Short note stored with the source. Empty leaves the existing note unchanged when updating."}
                            },
                            "required": ["id", "title", "url", "note"]
                        }
                    }),
                ),
                (
                    "remove_ids".to_owned(),
                    json!({"type": "array", "items": {"type": "string"}}),
                ),
            ]),
            required: vec!["sources".to_owned(), "remove_ids".to_owned()],
            parallel_safe: false,
        }
    }

    async fn run(
        &self,
        context: ToolContext,
        call: ToolCallRequest,
    ) -> Result<ToolResponse, AppError> {
        let input: UpdateSourcesInput = match serde_json::from_str(&call.input) {
            Ok(input) => input,
            Err(_) => return Ok(ToolResponse::error("Invalid input format")),
        };
        if input.sources.is_empty() && input.remove_ids.is_empty() {
            return Ok(ToolResponse::error("sources or remove_ids is required"));
        }
        let Some(article_id) = context.article_id else {
            return Ok(ToolResponse::error("No article ID available"));
        };
        let mut upserts = Vec::new();
        for source in input.sources {
            let id = match source
                .id
                .as_deref()
                .map(str::trim)
                .filter(|id| !id.is_empty())
            {
                Some(id) => match Uuid::parse_str(id) {
                    Ok(id) => Some(id),
                    Err(_) => return Ok(ToolResponse::error(format!("Invalid source id: {id}"))),
                },
                None => None,
            };
            let replace_content = !source.note.trim().is_empty();
            upserts.push(SourceEdit {
                id,
                title: source.title,
                url: source.url,
                content: source.note,
                replace_content,
            });
        }
        let mut remove_ids = Vec::new();
        for id in input.remove_ids {
            let id = id.trim();
            if id.is_empty() {
                continue;
            }
            match Uuid::parse_str(id) {
                Ok(id) => remove_ids.push(id),
                Err(_) => return Ok(ToolResponse::error(format!("Invalid source id: {id}"))),
            }
        }
        let saved = match self
            .sources
            .apply_edits(article_id, upserts, remove_ids)
            .await
        {
            Ok(saved) => saved,
            Err(AppError::NotFound) => {
                return Ok(ToolResponse::error(
                    "One or more source ids were not found on this article. Call read_document and use those ids.",
                ));
            }
            Err(AppError::InvalidInput(message)) => return Ok(ToolResponse::error(message)),
            Err(error) => return Err(error),
        };
        let views = saved
            .iter()
            .map(ArticleSourceView::from)
            .collect::<Vec<_>>();
        context.set_document_sources(views.clone())?;
        structured_result(json!({
            "sources": source_views_json(&views),
            "sources_updated": true,
            "tool_name": "update_sources",
        }))
    }
}

pub struct ApplyPatchTool {
    draft_saver: Option<Arc<dyn DraftSaver>>,
    sources: Option<Arc<dyn SourceResourcePort>>,
}

impl ApplyPatchTool {
    pub fn new(draft_saver: Option<Arc<dyn DraftSaver>>) -> Self {
        Self {
            draft_saver,
            sources: None,
        }
    }

    pub fn with_sources(mut self, sources: Arc<dyn SourceResourcePort>) -> Self {
        self.sources = Some(sources);
        self
    }
}

#[derive(Debug, Deserialize)]
struct ApplyPatchInput {
    #[serde(default)]
    patch: String,
    #[serde(default)]
    old_str: String,
    #[serde(default)]
    new_str: String,
    #[serde(default)]
    reason: String,
}

#[async_trait]
impl Tool for ApplyPatchTool {
    fn info(&self) -> ToolInfo {
        ToolInfo {
            name: "apply_patch".to_owned(),
            description: "Apply an edit to the article body on the backend. The editor receives the saved draft. To create a draft in an empty document, set old_str to an empty string and new_str to the full markdown. To edit, set old_str to the exact current body text (it must match once) and new_str to the replacement. Alternatively pass a *** Begin Patch block in patch. Do not include the title or a Sources section. Change the title with set_title and sources with update_sources. A leading level-1 heading is saved as the title and removed from the body. A trailing Sources, References, or Bibliography section is saved as source objects and removed from the body.".to_owned(),
            parameters: BTreeMap::from([
                (
                    "patch".to_owned(),
                    json!({"type": "string", "description": "*** Begin Patch block, or an empty string when using old_str and new_str."}),
                ),
                (
                    "old_str".to_owned(),
                    json!({"type": "string", "description": "Exact article text to replace. Empty only when creating the first draft."}),
                ),
                (
                    "new_str".to_owned(),
                    json!({"type": "string", "description": "Replacement markdown."}),
                ),
                ("reason".to_owned(), json!({"type": "string"})),
            ]),
            required: vec![
                "patch".to_owned(),
                "old_str".to_owned(),
                "new_str".to_owned(),
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
        let input: ApplyPatchInput = match serde_json::from_str(&call.input) {
            Ok(input) => input,
            Err(_) => return Ok(ToolResponse::error("Invalid input format")),
        };
        let document = context.document_markdown()?;
        let applied = if !input.patch.trim().is_empty() {
            apply_begin_patch(&document, &input.patch)
        } else {
            apply_replacement(&document, &input.old_str, &input.new_str)
        };
        let (new_markdown, old_str, new_str) = match applied {
            Ok(applied) => applied,
            Err(error) => return Ok(ToolResponse::error(error)),
        };
        let committed = commit_draft(
            &context,
            self.draft_saver.as_ref(),
            self.sources.as_ref(),
            new_markdown,
        )
        .await?;
        let result_value = json!({
            "old_str": old_str,
            "new_str": new_str,
            "new_markdown": committed.markdown,
            "reason": input.reason,
            "tool_name": "apply_patch",
            "title_updated": committed.title_updated,
            "new_title": committed.new_title,
            "sources_updated": committed.sources_updated,
            "sources": source_views_json(&committed.sources),
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
                    "original": old_str,
                    "proposed": new_str,
                    "reason": input.reason,
                })
                .as_object()
                .cloned()
                .ok_or(AppError::Internal)?,
            }),
        ))
    }
}

fn apply_replacement(
    document: &str,
    old_str: &str,
    new_str: &str,
) -> Result<(String, String, String), String> {
    if old_str.is_empty() {
        if !document.trim().is_empty() {
            return Err(
                "old_str is empty but the article already has content. Copy the exact text to replace."
                    .to_owned(),
            );
        }
        if new_str.trim().is_empty() {
            return Err(
                "new_str is required when creating a draft in an empty article.".to_owned(),
            );
        }
        return Ok((new_str.to_owned(), String::new(), new_str.to_owned()));
    }
    let matches = document.match_indices(old_str).count();
    if matches == 0 {
        return Err(
            "old_str was not found in the article. Call read_document and copy the exact text."
                .to_owned(),
        );
    }
    if matches > 1 {
        return Err(format!(
            "old_str matched {matches} times. Include more surrounding text so it matches once."
        ));
    }
    Ok((
        document.replacen(old_str, new_str, 1),
        old_str.to_owned(),
        new_str.to_owned(),
    ))
}

fn apply_begin_patch(document: &str, patch: &str) -> Result<(String, String, String), String> {
    let mut current = document.to_owned();
    let mut mode = "";
    let mut hunk: Vec<String> = Vec::new();
    let mut saw_file = false;
    let mut original = String::new();
    let mut proposed = String::new();
    let flush = |current: &mut String,
                 mode: &str,
                 hunk: &mut Vec<String>,
                 original: &mut String,
                 proposed: &mut String|
     -> Result<(), String> {
        if hunk.is_empty() {
            return Ok(());
        }
        let lines = std::mem::take(hunk);
        let borrowed = lines.iter().map(String::as_str).collect::<Vec<_>>();
        let (next, old_block, new_block) = match mode {
            "add" => {
                if !current.trim().is_empty() {
                    return Err(
                        "Add File replaces an empty article. Use Update File to edit existing text."
                            .to_owned(),
                    );
                }
                let body = added_lines(&borrowed)?;
                (body.clone(), String::new(), body)
            }
            "update" => apply_hunk(current, &borrowed)?,
            _ => {
                return Err(
                    "patch must name *** Update File: article.md or *** Add File: article.md"
                        .to_owned(),
                );
            }
        };
        if original.is_empty() {
            *original = old_block;
        }
        *proposed = new_block;
        *current = next;
        Ok(())
    };
    for line in patch.lines() {
        let trimmed = line.trim();
        if trimmed == "*** Begin Patch" || trimmed == "*** End Patch" {
            flush(&mut current, mode, &mut hunk, &mut original, &mut proposed)?;
            continue;
        }
        if let Some(path) = trimmed
            .strip_prefix("*** Update File:")
            .or_else(|| trimmed.strip_prefix("*** Update File "))
        {
            flush(&mut current, mode, &mut hunk, &mut original, &mut proposed)?;
            ensure_article_path(path)?;
            mode = "update";
            saw_file = true;
            continue;
        }
        if let Some(path) = trimmed
            .strip_prefix("*** Add File:")
            .or_else(|| trimmed.strip_prefix("*** Add File "))
        {
            flush(&mut current, mode, &mut hunk, &mut original, &mut proposed)?;
            ensure_article_path(path)?;
            mode = "add";
            saw_file = true;
            continue;
        }
        if trimmed.starts_with("***") {
            return Err(
                "unsupported patch command. Use *** Update File: article.md or *** Add File: article.md."
                    .to_owned(),
            );
        }
        if trimmed == "@@" || trimmed.starts_with("@@ ") {
            flush(&mut current, mode, &mut hunk, &mut original, &mut proposed)?;
            continue;
        }
        hunk.push(line.to_owned());
    }
    flush(&mut current, mode, &mut hunk, &mut original, &mut proposed)?;
    if !saw_file {
        return Err(
            "patch must include *** Begin Patch and *** Update File: article.md or *** Add File: article.md"
                .to_owned(),
        );
    }
    Ok((current, original, proposed))
}

fn ensure_article_path(path: &str) -> Result<(), String> {
    let path = path.trim();
    if path.is_empty() || path == "article.md" || path.ends_with("/article.md") {
        Ok(())
    } else {
        Err("patches apply to the article only (article.md)".to_owned())
    }
}

fn added_lines(hunk: &[&str]) -> Result<String, String> {
    let mut lines = Vec::new();
    for line in hunk {
        if let Some(rest) = line.strip_prefix('+') {
            lines.push(rest);
        } else if line.is_empty() {
            lines.push("");
        } else {
            return Err("Add File hunks only contain added lines prefixed with +".to_owned());
        }
    }
    if lines.join("\n").trim().is_empty() {
        return Err("Add File patch did not include any content".to_owned());
    }
    Ok(lines.join("\n"))
}

fn apply_hunk(document: &str, hunk: &[&str]) -> Result<(String, String, String), String> {
    let mut old_lines = Vec::new();
    let mut new_lines = Vec::new();
    for line in hunk {
        if let Some(rest) = line.strip_prefix('+') {
            new_lines.push(rest);
        } else if let Some(rest) = line.strip_prefix('-') {
            old_lines.push(rest);
        } else if let Some(rest) = line.strip_prefix(' ') {
            old_lines.push(rest);
            new_lines.push(rest);
        } else {
            old_lines.push(*line);
            new_lines.push(*line);
        }
    }
    let old_block = old_lines.join("\n");
    let new_block = new_lines.join("\n");
    if old_block.is_empty() {
        if document.trim().is_empty() {
            return Ok((new_block.clone(), String::new(), new_block));
        }
        return Err("patch hunk has no original text to match".to_owned());
    }
    apply_replacement(document, &old_block, &new_block)
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

impl From<&SourceResource> for ArticleSourceView {
    fn from(source: &SourceResource) -> Self {
        Self {
            id: source.id,
            title: source.title.clone(),
            url: source.url.clone(),
            content: source.content.clone(),
        }
    }
}

fn source_views_json(sources: &[ArticleSourceView]) -> Value {
    Value::Array(
        sources
            .iter()
            .map(|source| {
                json!({
                    "id": source.id,
                    "title": source.title,
                    "url": source.url,
                    "note": source.content.chars().take(280).collect::<String>(),
                })
            })
            .collect(),
    )
}

struct NormalizedDraft {
    markdown: String,
    title: Option<String>,
    extracted_sources: Vec<SourceEdit>,
}

fn normalize_draft(markdown: &str, current_title: &str) -> NormalizedDraft {
    let (title, body) = split_leading_title(markdown, current_title);
    let (body, extracted_sources) = split_trailing_sources(&body);
    NormalizedDraft {
        markdown: body,
        title,
        extracted_sources,
    }
}

fn split_leading_title(markdown: &str, current_title: &str) -> (Option<String>, String) {
    let trimmed = markdown.trim_start();
    let Some(first) = trimmed.lines().next() else {
        return (None, markdown.to_owned());
    };
    let rest = trimmed[first.len()..].trim_start_matches(['\r', '\n']);
    if let Some(title) = h1_text(first) {
        return (Some(title), rest.to_owned());
    }
    let current = current_title.trim();
    if !current.is_empty() && first.trim() == current {
        return (None, rest.to_owned());
    }
    (None, markdown.to_owned())
}

fn h1_text(line: &str) -> Option<String> {
    let trimmed = line.trim();
    let hashes = trimmed
        .chars()
        .take_while(|character| *character == '#')
        .count();
    if hashes != 1 {
        return None;
    }
    let rest = trimmed[hashes..].trim();
    if rest.is_empty() {
        return None;
    }
    Some(rest.to_owned())
}

fn split_trailing_sources(markdown: &str) -> (String, Vec<SourceEdit>) {
    let lines = markdown.split('\n').collect::<Vec<_>>();
    let mut start = None;
    for (index, line) in lines.iter().enumerate() {
        if !is_heading(line) {
            continue;
        }
        if sources_heading(line) {
            start = Some(index);
        } else {
            start = None;
        }
    }
    let Some(start) = start else {
        return (markdown.to_owned(), Vec::new());
    };
    let extracted = lines[start + 1..]
        .iter()
        .filter_map(|line| parse_source_line(line))
        .map(|(title, url)| SourceEdit {
            id: None,
            title,
            url,
            content: String::new(),
            replace_content: false,
        })
        .collect::<Vec<_>>();
    let body = lines[..start].join("\n").trim_end().to_owned();
    (body, extracted)
}

fn is_heading(line: &str) -> bool {
    let trimmed = line.trim();
    let hashes = trimmed
        .chars()
        .take_while(|character| *character == '#')
        .count();
    hashes > 0 && hashes <= 6 && !trimmed[hashes..].trim().is_empty()
}

fn sources_heading(line: &str) -> bool {
    let trimmed = line.trim();
    let hashes = trimmed
        .chars()
        .take_while(|character| *character == '#')
        .count();
    if hashes == 0 || hashes > 6 {
        return false;
    }
    let text = trimmed[hashes..]
        .trim()
        .trim_end_matches(':')
        .to_ascii_lowercase();
    matches!(
        text.as_str(),
        "sources" | "references" | "bibliography" | "citations"
    )
}

fn parse_source_line(line: &str) -> Option<(String, String)> {
    let mut text = line.trim();
    if text.is_empty() {
        return None;
    }
    text = text.trim_start_matches(['-', '*', '+']).trim();
    let numbered = text
        .find(|character: char| {
            !(character.is_ascii_digit() || character == '.' || character == ')')
        })
        .unwrap_or(text.len());
    if numbered > 0 && numbered < text.len() {
        let prefix = &text[..numbered];
        if prefix.chars().any(|character| character.is_ascii_digit()) {
            text = text[numbered..].trim();
        }
    }
    if text.is_empty() {
        return None;
    }
    if let Some(rest) = text.strip_prefix('[')
        && let Some((title, after)) = rest.split_once("](")
        && let Some(url) = after.split(')').next()
    {
        let title = title.trim();
        let url = url.trim();
        if !title.is_empty() || !url.is_empty() {
            return Some((
                if title.is_empty() {
                    url.to_owned()
                } else {
                    title.to_owned()
                },
                url.to_owned(),
            ));
        }
    }
    if text.starts_with("http://") || text.starts_with("https://") {
        return Some((text.to_owned(), text.to_owned()));
    }
    for separator in [" — ", " – ", " - ", ": "] {
        if let Some((title, url)) = text.rsplit_once(separator)
            && url.trim().starts_with("http")
        {
            return Some((title.trim().to_owned(), url.trim().to_owned()));
        }
    }
    Some((text.to_owned(), String::new()))
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
