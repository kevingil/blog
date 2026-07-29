use std::{
    collections::VecDeque,
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        datasource::{
            CrawledContent, CrawledContentRepository, DataSource, DataSourceRepository,
            DataSourceService, RecommendationSearchPort, SearchOptions, SearchResponse,
            SearchResult, SimilarOptions,
        },
        insight::InsightTopic,
        taskrun::TaskRunContext,
        worker::{
            CrawlSourcePort, CrawlWorker, DiscoveryWorker, InsightGenerationPort,
            InsightTopicResult, InsightWorker, StatusService, SystemClock, Worker, WorkerContext,
            WorkerFailure, WorkerResultStatus,
        },
    },
    error::AppError,
};
use chrono::{DateTime, Utc};
use serde_json::json;
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct SourceState {
    sources: Vec<DataSource>,
    crawl_updates: Vec<(Uuid, String, Option<String>)>,
    next_crawl_updates: Vec<Uuid>,
}

#[derive(Default)]
struct Sources {
    state: Mutex<SourceState>,
}

impl Sources {
    fn state(&self) -> MutexGuard<'_, SourceState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}

#[async_trait]
impl DataSourceRepository for Sources {
    async fn find_by_id(&self, id: Uuid) -> Result<DataSource, AppError> {
        self.state()
            .sources
            .iter()
            .find(|source| source.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_organization_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(self
            .state()
            .sources
            .iter()
            .filter(|source| source.organization_id == Some(id))
            .cloned()
            .collect())
    }

    async fn find_by_user_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(self
            .state()
            .sources
            .iter()
            .filter(|source| source.user_id == Some(id))
            .cloned()
            .collect())
    }

    async fn find_by_url(&self, url: &str) -> Result<Option<DataSource>, AppError> {
        Ok(self
            .state()
            .sources
            .iter()
            .find(|source| source.url == url)
            .cloned())
    }

    async fn find_due_to_crawl(&self, limit: i64) -> Result<Vec<DataSource>, AppError> {
        let limit =
            usize::try_from(limit).map_err(|_| AppError::InvalidInput("limit".to_owned()))?;
        Ok(self.state().sources.iter().take(limit).cloned().collect())
    }

    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<DataSource>, i64), AppError> {
        let state = self.state();
        let total = i64::try_from(state.sources.len()).map_err(|_| AppError::Internal)?;
        let offset = usize::try_from(offset).map_err(|_| AppError::Internal)?;
        let limit = usize::try_from(limit).map_err(|_| AppError::Internal)?;
        Ok((
            state
                .sources
                .iter()
                .skip(offset)
                .take(limit)
                .cloned()
                .collect(),
            total,
        ))
    }

    async fn save(&self, source: &mut DataSource) -> Result<(), AppError> {
        self.state().sources.push(source.clone());
        Ok(())
    }

    async fn update(&self, source: &DataSource) -> Result<(), AppError> {
        let mut state = self.state();
        let stored = state
            .sources
            .iter_mut()
            .find(|stored| stored.id == source.id)
            .ok_or(AppError::NotFound)?;
        *stored = source.clone();
        Ok(())
    }

    async fn update_crawl_status(
        &self,
        id: Uuid,
        status: &str,
        error_message: Option<&str>,
    ) -> Result<(), AppError> {
        self.state()
            .crawl_updates
            .push((id, status.to_owned(), error_message.map(str::to_owned)));
        Ok(())
    }

    async fn update_next_crawl_at(
        &self,
        id: Uuid,
        _next_crawl_at: DateTime<Utc>,
    ) -> Result<(), AppError> {
        self.state().next_crawl_updates.push(id);
        Ok(())
    }

    async fn increment_content_count(&self, _id: Uuid, _delta: i32) -> Result<(), AppError> {
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        self.state().sources.retain(|source| source.id != id);
        Ok(())
    }
}

#[derive(Default)]
struct Contents;

#[async_trait]
impl CrawledContentRepository for Contents {
    async fn find_by_data_source_id(
        &self,
        _id: Uuid,
        _offset: i64,
        _limit: i64,
    ) -> Result<(Vec<CrawledContent>, i64), AppError> {
        Ok((Vec::new(), 0))
    }

    async fn find_by_ids(&self, _ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        Ok(Vec::new())
    }

    async fn search_similar(
        &self,
        _embedding: &[f32],
        _limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        Ok(Vec::new())
    }

    async fn search_similar_by_org(
        &self,
        _organization_id: Uuid,
        _embedding: &[f32],
        _limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        Ok(Vec::new())
    }

    async fn find_recent_by_org(
        &self,
        _organization_id: Uuid,
        _limit: i64,
    ) -> Result<Vec<CrawledContent>, AppError> {
        Ok(Vec::new())
    }

    async fn find_by_id(&self, _id: Uuid) -> Result<CrawledContent, AppError> {
        Err(AppError::NotFound)
    }

    async fn find_by_url(
        &self,
        _data_source_id: Uuid,
        _url: &str,
    ) -> Result<Option<CrawledContent>, AppError> {
        Ok(None)
    }

    async fn save(&self, _content: &mut CrawledContent) -> Result<(), AppError> {
        Ok(())
    }

    async fn update(&self, _content: &CrawledContent) -> Result<(), AppError> {
        Ok(())
    }

    async fn delete(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }

    async fn delete_by_data_source_id(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }

    async fn count_by_data_source_id(&self, _id: Uuid) -> Result<i64, AppError> {
        Ok(0)
    }
}

fn source(name: &str, url: &str) -> DataSource {
    DataSource {
        id: Uuid::new_v4(),
        organization_id: Some(Uuid::new_v4()),
        user_id: Some(Uuid::new_v4()),
        name: name.to_owned(),
        url: url.to_owned(),
        feed_url: None,
        source_type: "blog".to_owned(),
        crawl_frequency: "daily".to_owned(),
        is_enabled: true,
        is_discovered: false,
        discovered_from_id: None,
        last_crawled_at: None,
        next_crawl_at: None,
        crawl_status: "pending".to_owned(),
        error_message: None,
        content_count: 0,
        subscriber_count: 1,
        meta_data: None,
        created_at: None,
        updated_at: None,
    }
}

fn context() -> WorkerContext {
    WorkerContext::new(CancellationToken::new(), TaskRunContext::default())
}

fn service(sources: Arc<Sources>) -> Arc<DataSourceService> {
    Arc::new(DataSourceService::new(sources, Arc::new(Contents)))
}

struct Crawler {
    results: Mutex<VecDeque<Result<i32, WorkerFailure>>>,
}

#[async_trait]
impl CrawlSourcePort for Crawler {
    fn is_configured(&self) -> bool {
        true
    }

    async fn crawl_source(
        &self,
        _source: &DataSource,
        _cancellation: &CancellationToken,
    ) -> Result<i32, WorkerFailure> {
        self.results
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .pop_front()
            .ok_or_else(|| WorkerFailure::new("missing crawl fixture"))?
    }
}

#[tokio::test]
async fn crawl_worker_keeps_partial_failures_as_warning_results() -> TestResult {
    let sources = Arc::new(Sources::default());
    sources.state().sources.extend([
        source("One", "https://one.test"),
        source("Two", "https://two.test"),
    ]);
    let crawler = Arc::new(Crawler {
        results: Mutex::new(VecDeque::from([
            Ok(3),
            Err(WorkerFailure::new("provider failed")),
        ])),
    });
    let worker = CrawlWorker::new(
        Arc::new(StatusService::new(Arc::new(SystemClock))),
        service(sources.clone()),
        Some(crawler),
    );
    let result = worker.run(context()).await?;
    assert_eq!(result.status, WorkerResultStatus::Warning);
    assert_eq!(result.metrics["sources_succeeded"], json!(1));
    assert_eq!(result.metrics["sources_failed"], json!(1));
    assert_eq!(result.metrics["content_created"], json!(3));
    assert_eq!(sources.state().next_crawl_updates.len(), 1);
    assert_eq!(
        sources
            .state()
            .crawl_updates
            .iter()
            .map(|(_, status, _)| status.as_str())
            .collect::<Vec<_>>(),
        ["crawling", "success", "crawling", "failed"]
    );
    Ok(())
}

struct SimilarSearch {
    results: Vec<SearchResult>,
}

#[async_trait]
impl RecommendationSearchPort for SimilarSearch {
    async fn search(
        &self,
        _query: &str,
        _options: SearchOptions,
    ) -> Result<SearchResponse, AppError> {
        Ok(SearchResponse::default())
    }

    async fn find_similar(
        &self,
        _url: &str,
        _options: SimilarOptions,
    ) -> Result<SearchResponse, AppError> {
        Ok(SearchResponse {
            results: self.results.clone(),
        })
    }

    fn is_configured(&self) -> bool {
        true
    }
}

#[tokio::test]
async fn discovery_worker_filters_same_domains_and_normalizes_tracking_urls() -> TestResult {
    let sources = Arc::new(Sources::default());
    let seed = source("Seed", "https://www.example.com/blog");
    sources.state().sources.push(seed);
    let search = Arc::new(SimilarSearch {
        results: vec![
            SearchResult {
                title: "Same".to_owned(),
                url: "https://news.example.com/post".to_owned(),
                ..SearchResult::default()
            },
            SearchResult {
                title: String::new(),
                url: "https://www.adjacent.test/article/?utm_source=x&keep=yes#part".to_owned(),
                ..SearchResult::default()
            },
        ],
    });
    let worker = DiscoveryWorker::new(
        Arc::new(StatusService::new(Arc::new(SystemClock))),
        service(sources.clone()),
        Some(search),
    );
    let result = worker.run(context()).await?;
    assert_eq!(result.status, WorkerResultStatus::Completed);
    assert_eq!(result.metrics["discovered_sources"], json!(1));
    let discovered = sources
        .state()
        .sources
        .iter()
        .find(|source| source.is_discovered)
        .cloned()
        .ok_or("discovered source missing")?;
    assert_eq!(discovered.name, "Adjacent");
    assert_eq!(discovered.url, "https://www.adjacent.test/article?keep=yes");
    assert!(!discovered.is_enabled);
    Ok(())
}

struct Generator {
    topics: Vec<InsightTopic>,
    results: Mutex<VecDeque<Result<InsightTopicResult, WorkerFailure>>>,
}

#[async_trait]
impl InsightGenerationPort for Generator {
    fn is_configured(&self) -> bool {
        true
    }

    async fn topics(&self) -> Result<Vec<InsightTopic>, WorkerFailure> {
        Ok(self.topics.clone())
    }

    async fn generate_for_topic(
        &self,
        _topic: &InsightTopic,
        _cancellation: &CancellationToken,
    ) -> Result<InsightTopicResult, WorkerFailure> {
        self.results
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .pop_front()
            .ok_or_else(|| WorkerFailure::new("missing insight fixture"))?
    }
}

fn topic(name: &str) -> InsightTopic {
    InsightTopic {
        id: Uuid::new_v4(),
        organization_id: Some(Uuid::new_v4()),
        name: name.to_owned(),
        description: None,
        keywords: None,
        embedding: None,
        is_auto_generated: false,
        content_count: 0,
        last_insight_at: None,
        color: None,
        icon: None,
        created_at: None,
        updated_at: None,
    }
}

#[tokio::test]
async fn insight_worker_isolates_topic_failures_in_warning_results() -> TestResult {
    let generator = Arc::new(Generator {
        topics: vec![topic("Created"), topic("Broken")],
        results: Mutex::new(VecDeque::from([
            Ok(InsightTopicResult::Created),
            Err(WorkerFailure::new("LLM response invalid")),
        ])),
    });
    let worker = InsightWorker::new(
        Arc::new(StatusService::new(Arc::new(SystemClock))),
        Some(generator),
    )?;
    let result = worker.run(context()).await?;
    assert_eq!(result.status, WorkerResultStatus::Warning);
    assert_eq!(result.metrics["topics_considered"], json!(2));
    assert_eq!(result.metrics["topics_failed"], json!(1));
    assert_eq!(result.metrics["insights_created"], json!(1));
    assert_eq!(result.warnings, vec!["1 topics failed during generation"]);
    Ok(())
}
