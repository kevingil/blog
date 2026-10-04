use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{
    core::{
        datasource::{CrawledContent, DataSource, DataSourceRepository},
        insight::{InsightContentRepository, InsightService, InsightTopic, InsightTopicRepository},
        ml::llm::ResearchPort,
        worker::{Clock, InsightGenerationPort, InsightTopicResult, WorkerFailure},
    },
    error::AppError,
    integrations::{llm::GroqClient, openai::OpenAiClient},
};

const MAX_PAGES: usize = 8;
const FALLBACK_PERIOD: Duration = Duration::days(7);
const MAX_ARTICLE_CONTENT_CHARS: usize = 1_500;

pub(crate) const INSIGHT_INSTRUCTIONS: &str = r#"You are the research desk for a technical writer. Read the supplied pages and the previous briefing, then return one briefing about what people are saying right now.

Return only a JSON object with this exact schema:
{
  "title": "non-empty string",
  "summary": "2-3 sentences on the current conversation",
  "content": "2-4 paragraphs on what matters and why a writer would care now",
  "key_points": ["3-5 writing angles, each a non-empty sentence"]
}

Use only the supplied pages. If a previous briefing is present, update it instead of repeating it. Do not wrap the JSON in Markdown or add fields."#;

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct InsightGenerationRequest {
    pub topic: InsightTopicContext,
    pub articles: Vec<InsightArticleContext>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct InsightTopicContext {
    pub name: String,
    pub description: Option<String>,
    pub previous_summary: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct InsightArticleContext {
    pub id: Uuid,
    pub title: Option<String>,
    pub url: String,
    pub published_at: Option<String>,
    pub content: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct GeneratedInsight {
    pub title: String,
    pub summary: String,
    pub content: String,
    pub key_points: Vec<String>,
}

impl GeneratedInsight {
    fn validate(self) -> Result<Self, AppError> {
        if self.title.trim().is_empty()
            || self.summary.trim().is_empty()
            || self.content.trim().is_empty()
            || !(3..=5).contains(&self.key_points.len())
            || self.key_points.iter().any(|point| point.trim().is_empty())
        {
            return Err(AppError::InvalidInput(
                "structured insight response does not satisfy the required schema".to_owned(),
            ));
        }
        Ok(self)
    }
}

#[async_trait]
pub trait InsightTextGenerator: Send + Sync {
    fn is_configured(&self) -> bool;

    async fn generate_insight(
        &self,
        request: InsightGenerationRequest,
    ) -> Result<GeneratedInsight, AppError>;
}

#[async_trait]
impl InsightTextGenerator for OpenAiClient {
    fn is_configured(&self) -> bool {
        OpenAiClient::is_configured(self)
    }

    async fn generate_insight(
        &self,
        request: InsightGenerationRequest,
    ) -> Result<GeneratedInsight, AppError> {
        let input = serde_json::to_string(&request).map_err(|_| AppError::Internal)?;
        let response = self.generate_provider_text(&input).await?;
        decode_generated_insight(&response)
    }
}

#[async_trait]
impl InsightTextGenerator for GroqClient {
    fn is_configured(&self) -> bool {
        GroqClient::is_configured(self)
    }

    async fn generate_insight(
        &self,
        request: InsightGenerationRequest,
    ) -> Result<GeneratedInsight, AppError> {
        let input = serde_json::to_string(&request).map_err(|_| AppError::Internal)?;
        let response = self.generate_text(&input).await?;
        decode_generated_insight(&response)
    }
}

pub(crate) fn decode_generated_insight(response: &str) -> Result<GeneratedInsight, AppError> {
    serde_json::from_str::<GeneratedInsight>(response)
        .map_err(|_| AppError::External)?
        .validate()
}

#[derive(Debug, Clone, PartialEq)]
pub struct NewInsight {
    pub organization_id: Option<Uuid>,
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    pub content: String,
    pub key_points: Vec<String>,
    pub source_content_ids: Vec<Uuid>,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
    pub data_source_id: Option<Uuid>,
}

#[async_trait]
pub trait InsightWriter: Send + Sync {
    async fn create(&self, insight: NewInsight) -> Result<(), AppError>;

    async fn previous_summary(
        &self,
        _topic_id: Option<Uuid>,
        _data_source_id: Option<Uuid>,
    ) -> Result<Option<String>, AppError> {
        Ok(None)
    }
}

#[async_trait]
impl InsightWriter for InsightService {
    async fn create(&self, insight: NewInsight) -> Result<(), AppError> {
        self.create_insight(
            insight.organization_id,
            insight.topic_id,
            insight.title,
            insight.summary,
            insight.content,
            Some(insight.key_points),
            Some(insight.source_content_ids),
            Some(insight.period_start),
            Some(insight.period_end),
            insight.data_source_id,
        )
        .await
        .map(|_| ())
    }

    async fn previous_summary(
        &self,
        topic_id: Option<Uuid>,
        data_source_id: Option<Uuid>,
    ) -> Result<Option<String>, AppError> {
        self.latest_briefing_summary(topic_id, data_source_id).await
    }
}

pub struct RuntimeInsightGenerator {
    topics: Arc<dyn InsightTopicRepository>,
    sources: Arc<dyn DataSourceRepository>,
    contents: Arc<dyn InsightContentRepository>,
    research: Arc<dyn ResearchPort>,
    text: Arc<dyn InsightTextGenerator>,
    writer: Arc<dyn InsightWriter>,
    clock: Arc<dyn Clock>,
}

impl RuntimeInsightGenerator {
    pub fn new(
        topics: Arc<dyn InsightTopicRepository>,
        sources: Arc<dyn DataSourceRepository>,
        contents: Arc<dyn InsightContentRepository>,
        research: Arc<dyn ResearchPort>,
        text: Arc<dyn InsightTextGenerator>,
        writer: Arc<dyn InsightWriter>,
        clock: Arc<dyn Clock>,
    ) -> Self {
        Self {
            topics,
            sources,
            contents,
            research,
            text,
            writer,
            clock,
        }
    }

    async fn generate(
        &self,
        topic: &InsightTopic,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        self.research_tracker(
            cancellation,
            &topic.name,
            topic.description.as_deref(),
            None,
            Some(topic.id),
            None,
            topic.organization_id,
        )
        .await
    }

    async fn research_domain(
        &self,
        source: &DataSource,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        let host = reqwest::Url::parse(&source.url)
            .ok()
            .and_then(|url| url.host_str().map(str::to_owned))
            .ok_or_else(|| {
                WorkerFailure::new(format!("failed to extract domain from URL: {}", source.url))
            })?;
        self.research_tracker(
            cancellation,
            &source.name,
            None,
            Some(host.as_str()),
            None,
            Some(source.id),
            source.organization_id,
        )
        .await
    }

    async fn research_tracker(
        &self,
        cancellation: &CancellationToken,
        name: &str,
        description: Option<&str>,
        domain: Option<&str>,
        topic_id: Option<Uuid>,
        data_source_id: Option<Uuid>,
        organization_id: Option<Uuid>,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        if cancellation.is_cancelled() {
            return Err(cancelled());
        }
        let query = if domain.is_some() {
            format!("latest articles and discussions from {name}")
        } else {
            match description {
                Some(description) if !description.is_empty() => format!("{name}. {description}"),
                _ => name.to_owned(),
            }
        };
        tracing::info!(
            tracker = name,
            domain = domain.unwrap_or(""),
            %query,
            "insight research started"
        );
        let searched = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.research.deep_search(&query, domain) => {
                result.map_err(|error| failure("failed to research tracker", error))?
            }
        };
        let pages = searched
            .results
            .into_iter()
            .filter(|result| !result.text.is_empty() || !result.highlights.is_empty())
            .take(MAX_PAGES)
            .collect::<Vec<_>>();
        tracing::info!(
            tracker = name,
            pages = pages.len(),
            "insight research returned pages"
        );
        if pages.is_empty() {
            self.advance_topic(topic_id).await?;
            self.advance_source(data_source_id).await?;
            return Ok(InsightTopicResult::SkippedInsufficient);
        }
        let mut contents = Vec::with_capacity(pages.len());
        for page in pages {
            if cancellation.is_cancelled() {
                return Err(cancelled());
            }
            let body = if page.text.is_empty() {
                page.highlights.join("\n")
            } else {
                page.text
            };
            let mut content = CrawledContent {
                id: Uuid::new_v4(),
                data_source_id,
                topic_id,
                url: page.url,
                title: Some(page.title).filter(|title| !title.is_empty()),
                content: body,
                summary: Some(page.summary).filter(|summary| !summary.is_empty()),
                author: Some(page.author).filter(|author| !author.is_empty()),
                published_at: chrono::DateTime::parse_from_rfc3339(&page.published_date)
                    .ok()
                    .map(|value| value.with_timezone(&chrono::Utc)),
                embedding: None,
                meta_data: None,
                created_at: Some(self.clock.now()),
            };
            tokio::select! {
                biased;
                () = cancellation.cancelled() => return Err(cancelled()),
                result = self.contents.save(&mut content) => {
                    result.map_err(|error| failure("failed to save research page", error))?
                }
            }
            contents.push(content);
        }
        let now = self.clock.now();
        let (period_start, period_end) = insight_period(&contents, now);
        let previous_summary = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.writer.previous_summary(topic_id, data_source_id) => {
                result.map_err(|error| failure("failed to load previous briefing", error))?
            }
        };
        let request = InsightGenerationRequest {
            topic: InsightTopicContext {
                name: name.to_owned(),
                description: description.map(str::to_owned),
                previous_summary,
            },
            articles: contents
                .iter()
                .map(|content| InsightArticleContext {
                    id: content.id,
                    title: content.title.clone(),
                    url: content.url.clone(),
                    published_at: content
                        .published_at
                        .map(|value| value.to_rfc3339_opts(chrono::SecondsFormat::Secs, true)),
                    content: truncate_chars(&content.content, MAX_ARTICLE_CONTENT_CHARS),
                })
                .collect(),
        };
        let generated = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.text.generate_insight(request) => {
                result.map_err(|error| failure("failed to generate structured insight", error))?
            }
        };
        let insight = NewInsight {
            organization_id,
            topic_id,
            title: generated.title,
            summary: generated.summary,
            content: generated.content,
            key_points: generated.key_points,
            source_content_ids: contents.iter().map(|content| content.id).collect(),
            period_start,
            period_end,
            data_source_id,
        };
        let title = insight.title.clone();
        let page_count = contents.len();
        tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.writer.create(insight) => {
                result.map_err(|error| failure("failed to create insight", error))?
            }
        }
        tracing::info!(tracker = name, %title, pages = page_count, "insight briefing saved");
        if let Some(topic_id) = topic_id {
            tokio::select! {
                biased;
                () = cancellation.cancelled() => return Err(cancelled()),
                result = self.topics.update_last_insight_at(topic_id, now) => {
                    result.map_err(|error| failure("failed to update topic insight timestamp", error))?
                }
            }
            self.advance_topic(Some(topic_id)).await?;
        }
        self.advance_source(data_source_id).await?;
        Ok(InsightTopicResult::Created)
    }

    async fn advance_topic(&self, topic_id: Option<Uuid>) -> Result<(), WorkerFailure> {
        let Some(topic_id) = topic_id else {
            return Ok(());
        };
        let mut topic = self
            .topics
            .find_by_id(topic_id)
            .await
            .map_err(|error| failure("failed to load topic schedule", error))?;
        topic.next_check_at = Some(self.clock.now() + check_interval(&topic.check_frequency));
        self.topics
            .update(&topic)
            .await
            .map_err(|error| failure("failed to update topic schedule", error))
    }

    async fn advance_source(&self, source_id: Option<Uuid>) -> Result<(), WorkerFailure> {
        let Some(source_id) = source_id else {
            return Ok(());
        };
        let source = self
            .sources
            .find_by_id(source_id)
            .await
            .map_err(|error| failure("failed to load source schedule", error))?;
        self.sources
            .update_next_crawl_at(
                source_id,
                self.clock.now() + check_interval(&source.crawl_frequency),
            )
            .await
            .map_err(|error| failure("failed to update source schedule", error))
    }
}

#[async_trait]
impl InsightGenerationPort for RuntimeInsightGenerator {
    fn is_configured(&self) -> bool {
        self.text.is_configured()
    }

    async fn topics(&self) -> Result<Vec<InsightTopic>, WorkerFailure> {
        self.topics
            .find_due(100)
            .await
            .map_err(|error| failure("failed to list insight topics", error))
    }

    async fn sources(&self) -> Result<Vec<DataSource>, WorkerFailure> {
        self.sources
            .find_due_to_crawl(100)
            .await
            .map_err(|error| failure("failed to list due sources", error))
    }

    async fn generate_for_source(
        &self,
        source: &DataSource,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        self.research_domain(source, cancellation).await
    }

    async fn generate_for_topic(
        &self,
        topic: &InsightTopic,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        self.generate(topic, cancellation).await
    }
}

fn insight_period(
    contents: &[CrawledContent],
    now: DateTime<Utc>,
) -> (DateTime<Utc>, DateTime<Utc>) {
    let mut timestamps = contents.iter().filter_map(|content| content.published_at);
    let Some(first) = timestamps.next() else {
        return (now - FALLBACK_PERIOD, now);
    };
    timestamps.fold((first, first), |(start, end), timestamp| {
        (start.min(timestamp), end.max(timestamp))
    })
}

fn truncate_chars(value: &str, maximum: usize) -> String {
    if value.chars().count() <= maximum {
        value.to_owned()
    } else {
        let mut truncated = value.chars().take(maximum).collect::<String>();
        truncated.push_str("...");
        truncated
    }
}

fn failure(context: &str, error: AppError) -> WorkerFailure {
    WorkerFailure::new(format!("{context}: {error}"))
}

fn cancelled() -> WorkerFailure {
    WorkerFailure::new("operation cancelled")
}

fn check_interval(frequency: &str) -> Duration {
    match frequency {
        "hourly" => Duration::hours(1),
        "weekly" => Duration::days(7),
        _ => Duration::days(1),
    }
}
