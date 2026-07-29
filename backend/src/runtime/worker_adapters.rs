use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Duration, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::{
    core::{
        datasource::CrawledContent,
        insight::{
            ContentTopicMatchRepository, InsightContentRepository, InsightService, InsightTopic,
            InsightTopicRepository,
        },
        worker::{Clock, InsightGenerationPort, InsightTopicResult, WorkerFailure},
    },
    error::AppError,
    integrations::{llm::GroqClient, openai::OpenAiClient},
};

const MIN_CONTENT_COUNT: i64 = 3;
const MAX_CONTENT_PER_GENERATION: i64 = 10;
const RECENT_INSIGHT_WINDOW: Duration = Duration::hours(24);
const FALLBACK_PERIOD: Duration = Duration::days(7);
const MAX_ARTICLE_CONTENT_CHARS: usize = 1_500;

pub(crate) const INSIGHT_INSTRUCTIONS: &str = r#"Analyze the supplied topic and articles and return one insight.

Return only a JSON object with this exact schema:
{
  "title": "non-empty string",
  "summary": "2-3 sentences",
  "content": "2-4 paragraphs",
  "key_points": ["3-5 non-empty takeaways"]
}

Synthesize only information supported by the supplied articles. Do not wrap the JSON in Markdown or add fields."#;

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct InsightGenerationRequest {
    pub topic: InsightTopicContext,
    pub articles: Vec<InsightArticleContext>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub struct InsightTopicContext {
    pub name: String,
    pub description: Option<String>,
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
        let response = self.generate_text(INSIGHT_INSTRUCTIONS, &input).await?;
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
    pub topic_id: Uuid,
    pub title: String,
    pub summary: String,
    pub content: String,
    pub key_points: Vec<String>,
    pub source_content_ids: Vec<Uuid>,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
}

#[async_trait]
pub trait InsightWriter: Send + Sync {
    async fn create(&self, insight: NewInsight) -> Result<(), AppError>;
}

#[async_trait]
impl InsightWriter for InsightService {
    async fn create(&self, insight: NewInsight) -> Result<(), AppError> {
        self.create_insight(
            insight.organization_id,
            Some(insight.topic_id),
            insight.title,
            insight.summary,
            insight.content,
            Some(insight.key_points),
            Some(insight.source_content_ids),
            Some(insight.period_start),
            Some(insight.period_end),
        )
        .await
        .map(|_| ())
    }
}

pub struct RuntimeInsightGenerator {
    topics: Arc<dyn InsightTopicRepository>,
    matches: Arc<dyn ContentTopicMatchRepository>,
    contents: Arc<dyn InsightContentRepository>,
    text: Arc<dyn InsightTextGenerator>,
    writer: Arc<dyn InsightWriter>,
    clock: Arc<dyn Clock>,
}

impl RuntimeInsightGenerator {
    pub fn new(
        topics: Arc<dyn InsightTopicRepository>,
        matches: Arc<dyn ContentTopicMatchRepository>,
        contents: Arc<dyn InsightContentRepository>,
        text: Arc<dyn InsightTextGenerator>,
        writer: Arc<dyn InsightWriter>,
        clock: Arc<dyn Clock>,
    ) -> Self {
        Self {
            topics,
            matches,
            contents,
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
        if cancellation.is_cancelled() {
            return Err(cancelled());
        }
        let (matches, total) = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.matches.find_primary_by_topic_id(
                topic.id,
                0,
                MAX_CONTENT_PER_GENERATION,
            ) => result.map_err(|error| failure("failed to get content matches", error))?,
        };
        if total < MIN_CONTENT_COUNT {
            return Ok(InsightTopicResult::SkippedInsufficient);
        }

        let content_ids = matches
            .into_iter()
            .map(|value| value.content_id)
            .collect::<Vec<_>>();
        let contents = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.contents.find_by_ids(&content_ids) => {
                result.map_err(|error| failure("failed to get content details", error))?
            }
        };
        if i64::try_from(contents.len()).unwrap_or(i64::MAX) < MIN_CONTENT_COUNT {
            return Ok(InsightTopicResult::SkippedInsufficient);
        }

        let now = self.clock.now();
        if topic
            .last_insight_at
            .is_some_and(|last| now.signed_duration_since(last) < RECENT_INSIGHT_WINDOW)
        {
            return Ok(InsightTopicResult::SkippedRecent);
        }

        let (period_start, period_end) = insight_period(&contents, now);
        let request = generation_request(topic, &contents);
        let generated = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.text.generate_insight(request) => {
                result.map_err(|error| failure("failed to generate structured insight", error))?
            }
        };
        let insight = NewInsight {
            organization_id: topic.organization_id,
            topic_id: topic.id,
            title: generated.title,
            summary: generated.summary,
            content: generated.content,
            key_points: generated.key_points,
            source_content_ids: content_ids,
            period_start,
            period_end,
        };
        tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.writer.create(insight) => {
                result.map_err(|error| failure("failed to create insight", error))?
            }
        }
        tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(cancelled()),
            result = self.topics.update_last_insight_at(topic.id, now) => {
                result.map_err(|error| failure("failed to update topic insight timestamp", error))?
            }
        }
        Ok(InsightTopicResult::Created)
    }
}

#[async_trait]
impl InsightGenerationPort for RuntimeInsightGenerator {
    fn is_configured(&self) -> bool {
        self.text.is_configured()
    }

    async fn topics(&self) -> Result<Vec<InsightTopic>, WorkerFailure> {
        self.topics
            .find_all()
            .await
            .map_err(|error| failure("failed to list insight topics", error))
    }

    async fn generate_for_topic(
        &self,
        topic: &InsightTopic,
        cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        self.generate(topic, cancellation).await
    }
}

fn generation_request(
    topic: &InsightTopic,
    contents: &[CrawledContent],
) -> InsightGenerationRequest {
    InsightGenerationRequest {
        topic: InsightTopicContext {
            name: topic.name.clone(),
            description: topic.description.clone(),
        },
        articles: contents
            .iter()
            .map(|content| InsightArticleContext {
                id: content.id,
                title: content.title.clone(),
                url: content.url.clone(),
                published_at: content
                    .published_at
                    .map(|value| value.to_rfc3339_opts(SecondsFormat::Secs, true)),
                content: truncate_chars(&content.content, MAX_ARTICLE_CONTENT_CHARS),
            })
            .collect(),
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
