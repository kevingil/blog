use std::sync::Arc;

use async_trait::async_trait;
use serde_json::json;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

use crate::core::{
    datasource::{
        CrawledContent, CrawledContentRepository, DataSource, DataSourceRepository,
        DataSourceService, RecommendationSearchPort, SearchOptions,
    },
    insight::{EmbeddingPort, InsightService},
    source::FetchExtractPort,
};

use super::{
    StatusService, Worker, WorkerContext, WorkerFailure, WorkerResult, WorkerResultStatus,
    WorkerState,
};

const CRAWL_WORKER_NAME: &str = "crawl";
const DEFAULT_BATCH_SIZE: i64 = 10;

#[async_trait]
pub trait CrawlSourcePort: Send + Sync {
    fn is_configured(&self) -> bool;

    async fn crawl_source(
        &self,
        source: &DataSource,
        cancellation: &CancellationToken,
    ) -> Result<i32, WorkerFailure>;
}

pub struct ContentCrawler {
    search: Arc<dyn RecommendationSearchPort>,
    fetch: Arc<dyn FetchExtractPort>,
    data_sources: Arc<dyn DataSourceRepository>,
    contents: Arc<dyn CrawledContentRepository>,
    embeddings: Arc<dyn EmbeddingPort>,
    insights: Arc<InsightService>,
    topic_threshold: f64,
}

impl ContentCrawler {
    pub fn new(
        search: Arc<dyn RecommendationSearchPort>,
        fetch: Arc<dyn FetchExtractPort>,
        data_sources: Arc<dyn DataSourceRepository>,
        contents: Arc<dyn CrawledContentRepository>,
        embeddings: Arc<dyn EmbeddingPort>,
        insights: Arc<InsightService>,
    ) -> Self {
        Self {
            search,
            fetch,
            data_sources,
            contents,
            embeddings,
            insights,
            topic_threshold: 0.6,
        }
    }

    async fn crawl_items(&self, source: &DataSource) -> Result<Vec<CrawledItem>, WorkerFailure> {
        let domain = reqwest::Url::parse(&source.url)
            .ok()
            .and_then(|url| url.host_str().map(str::to_owned))
            .ok_or_else(|| {
                WorkerFailure::new(format!("failed to extract domain from URL: {}", source.url))
            })?;
        let searched = self
            .search
            .search(
                &format!("site:{domain}"),
                SearchOptions {
                    num_results: 20,
                    include_text: true,
                    include_summary: true,
                    ..SearchOptions::default()
                },
            )
            .await;
        match searched {
            Ok(response) if !response.results.is_empty() => Ok(response
                .results
                .into_iter()
                .filter(|result| !result.text.is_empty())
                .map(|result| CrawledItem {
                    url: result.url,
                    title: result.title,
                    content: result.text,
                })
                .collect()),
            Ok(_) | Err(_) => {
                let scraped = self
                    .fetch
                    .fetch_extract(&source.url)
                    .await
                    .map_err(|error| WorkerFailure::new(format!("failed to fetch URL: {error}")))?;
                if scraped.content.is_empty() {
                    Ok(Vec::new())
                } else {
                    Ok(vec![CrawledItem {
                        url: scraped.url,
                        title: scraped.title,
                        content: scraped.content,
                    }])
                }
            }
        }
    }

    async fn process_item(
        &self,
        source: &DataSource,
        item: CrawledItem,
        cancellation: &CancellationToken,
    ) -> Result<bool, WorkerFailure> {
        let existing = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.contents.find_by_url(source.id, &item.url) => result,
        };
        if matches!(existing, Ok(Some(_))) {
            // The Go worker counts an already-persisted URL as processed because
            // its per-item function returns success before the outer counter is
            // incremented. Preserve that observable metric behavior.
            return Ok(true);
        }
        let embedding_input = truncate_chars(&item.content, 8_000);
        let embedding = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.embeddings.generate_embedding(&embedding_input) => {
                result.map_err(|error| WorkerFailure::new(format!("failed to generate embedding: {error}")))?
            }
        };
        let mut content = CrawledContent {
            id: Uuid::new_v4(),
            data_source_id: source.id,
            url: item.url,
            title: Some(item.title),
            content: item.content,
            summary: None,
            author: None,
            published_at: None,
            embedding: Some(embedding.clone()),
            meta_data: None,
            created_at: None,
        };
        tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.contents.save(&mut content) => {
                result.map_err(|error| WorkerFailure::new(format!("failed to save content: {error}")))?
            }
        }
        let _ = self
            .insights
            .match_content_to_topics(content.id, &embedding, self.topic_threshold)
            .await;
        Ok(true)
    }
}

#[derive(Debug)]
struct CrawledItem {
    url: String,
    title: String,
    content: String,
}

#[async_trait]
impl CrawlSourcePort for ContentCrawler {
    fn is_configured(&self) -> bool {
        self.search.is_configured()
    }

    async fn crawl_source(
        &self,
        source: &DataSource,
        cancellation: &CancellationToken,
    ) -> Result<i32, WorkerFailure> {
        let items = tokio::select! {
            biased;
            () = cancellation.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.crawl_items(source) => result?,
        };
        let mut created = 0_i32;
        for item in items {
            if let Ok(true) = self.process_item(source, item, cancellation).await {
                created += 1;
            }
        }
        if created > 0 {
            let _ = self
                .data_sources
                .increment_content_count(source.id, created)
                .await;
        }
        Ok(created)
    }
}

fn truncate_chars(value: &str, maximum: usize) -> String {
    if value.chars().count() <= maximum {
        value.to_owned()
    } else {
        value.chars().take(maximum).collect()
    }
}

pub struct CrawlWorker {
    status: Arc<StatusService>,
    data_sources: Arc<DataSourceService>,
    crawler: Option<Arc<dyn CrawlSourcePort>>,
    batch_size: i64,
}

impl CrawlWorker {
    pub fn new(
        status: Arc<StatusService>,
        data_sources: Arc<DataSourceService>,
        crawler: Option<Arc<dyn CrawlSourcePort>>,
    ) -> Self {
        Self {
            status,
            data_sources,
            crawler,
            batch_size: DEFAULT_BATCH_SIZE,
        }
    }

    pub fn with_batch_size(mut self, batch_size: i64) -> Result<Self, WorkerFailure> {
        if batch_size <= 0 {
            return Err(WorkerFailure::new(
                "crawl worker batch size must be greater than zero",
            ));
        }
        self.batch_size = batch_size;
        Ok(self)
    }
}

#[async_trait]
impl Worker for CrawlWorker {
    fn name(&self) -> &str {
        CRAWL_WORKER_NAME
    }

    async fn run(&self, context: WorkerContext) -> Result<WorkerResult, WorkerFailure> {
        let Some(crawler) = self
            .crawler
            .as_ref()
            .filter(|crawler| crawler.is_configured())
        else {
            self.status.update_status(
                self.name(),
                WorkerState::Running,
                100,
                "Exa not configured, skipping",
            );
            let mut result = WorkerResult::warning(
                "Exa not configured, crawl skipped",
                vec!["Exa client not configured".to_owned()],
            );
            result
                .metrics
                .insert("sources_considered".to_owned(), json!(0));
            result
                .metrics
                .insert("content_created".to_owned(), json!(0));
            return Ok(result);
        };

        self.status.update_status(
            self.name(),
            WorkerState::Running,
            0,
            "Fetching sources to crawl...",
        );
        let sources = tokio::select! {
            biased;
            () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
            result = self.data_sources.get_due_to_crawl(self.batch_size) => {
                result.map_err(|error| WorkerFailure::new(format!("failed to get sources due for crawling: {error}")))?
            }
        };
        if sources.is_empty() {
            self.status.update_status(
                self.name(),
                WorkerState::Running,
                100,
                "No sources due for crawling",
            );
            let mut result = WorkerResult::warning(
                "No sources due for crawling",
                vec!["No sources were due for crawling".to_owned()],
            );
            result
                .metrics
                .insert("sources_considered".to_owned(), json!(0));
            result
                .metrics
                .insert("content_created".to_owned(), json!(0));
            return Ok(result);
        }

        let total = i32::try_from(sources.len())
            .map_err(|_| WorkerFailure::new("too many crawl sources"))?;
        self.status.set_progress(
            self.name(),
            0,
            total,
            format!("Found {total} sources to crawl"),
        );
        let mut sources_failed = 0_i32;
        let mut sources_succeeded = 0_i32;
        let mut content_created = 0_i32;
        for (index, source) in sources.into_iter().enumerate() {
            if context.cancellation().is_cancelled() {
                return Err(WorkerFailure::new("operation cancelled"));
            }
            let done =
                i32::try_from(index).map_err(|_| WorkerFailure::new("too many crawl sources"))?;
            self.status.set_progress(
                self.name(),
                done,
                total,
                format!("Crawling: {}", source.name),
            );
            let _ = self
                .data_sources
                .update_crawl_status(source.id, "crawling", None)
                .await;
            let crawled = tokio::select! {
                biased;
                () = context.cancelled() => return Err(WorkerFailure::new("operation cancelled")),
                result = crawler.crawl_source(&source, context.cancellation()) => result,
            };
            match crawled {
                Ok(created) => {
                    let _ = self
                        .data_sources
                        .update_crawl_status(source.id, "success", None)
                        .await;
                    let _ = self
                        .data_sources
                        .set_next_crawl_time(source.id, &source.crawl_frequency)
                        .await;
                    sources_succeeded += 1;
                    content_created = content_created.saturating_add(created);
                }
                Err(error) => {
                    let _ = self
                        .data_sources
                        .update_crawl_status(source.id, "failed", Some(error.message()))
                        .await;
                    sources_failed += 1;
                }
            }
        }

        self.status.set_progress(
            self.name(),
            total,
            total,
            format!("Completed crawling {total} sources"),
        );
        let mut result = if content_created == 0 || sources_failed > 0 {
            let mut warnings = Vec::new();
            if content_created == 0 {
                warnings.push("No new content was created".to_owned());
            }
            if sources_failed > 0 {
                warnings.push(format!("{sources_failed} sources failed during crawl"));
            }
            WorkerResult::warning(
                format!("Crawl finished with {content_created} new content items"),
                warnings,
            )
        } else {
            WorkerResult::completed(format!(
                "Crawled {sources_succeeded} sources and created {content_created} content items"
            ))
        };
        result.status = if content_created == 0 || sources_failed > 0 {
            WorkerResultStatus::Warning
        } else {
            WorkerResultStatus::Completed
        };
        result
            .output_summary
            .insert("content_created".to_owned(), json!(content_created));
        result
            .metrics
            .insert("sources_considered".to_owned(), json!(total));
        result
            .metrics
            .insert("sources_succeeded".to_owned(), json!(sources_succeeded));
        result
            .metrics
            .insert("sources_failed".to_owned(), json!(sources_failed));
        result
            .metrics
            .insert("content_created".to_owned(), json!(content_created));
        Ok(result)
    }
}
