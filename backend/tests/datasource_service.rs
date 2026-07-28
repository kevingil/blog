use std::{
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::datasource::{
        CrawledContent, CrawledContentRepository, DataSource, DataSourceCreateRequest,
        DataSourceDiscoveryRecommendationRequest, DataSourceRecommendationRequest,
        DataSourceRepository, DataSourceService, DataSourceUpdateRequest, RecommendationSearchPort,
        RecommendationService, SearchOptions, SearchResponse, SearchResult, SimilarOptions,
    },
    error::AppError,
};
use chrono::{DateTime, Utc};
use uuid::Uuid;

type TestResult = Result<(), Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct Store {
    sources: Mutex<Vec<DataSource>>,
    contents: Mutex<Vec<CrawledContent>>,
    search_results: Mutex<Vec<SearchResult>>,
    similar_results: Mutex<Vec<SearchResult>>,
    configured: Mutex<bool>,
}

fn lock<T>(value: &Mutex<T>) -> Result<MutexGuard<'_, T>, AppError> {
    value.lock().map_err(|_| AppError::Internal)
}

fn source(id: Uuid, organization_id: Option<Uuid>, user_id: Option<Uuid>, url: &str) -> DataSource {
    DataSource {
        id,
        organization_id,
        user_id,
        name: "Source".to_owned(),
        url: url.to_owned(),
        feed_url: None,
        source_type: "blog".to_owned(),
        crawl_frequency: "daily".to_owned(),
        is_enabled: true,
        is_discovered: false,
        discovered_from_id: None,
        last_crawled_at: None,
        next_crawl_at: Some(Utc::now()),
        crawl_status: "pending".to_owned(),
        error_message: None,
        content_count: 0,
        subscriber_count: 1,
        meta_data: None,
        created_at: Some(Utc::now()),
        updated_at: Some(Utc::now()),
    }
}

fn content(id: Uuid, data_source_id: Uuid) -> CrawledContent {
    CrawledContent {
        id,
        data_source_id,
        url: "https://example.com/post".to_owned(),
        title: Some("Post".to_owned()),
        content: "Body".to_owned(),
        summary: None,
        author: None,
        published_at: None,
        embedding: None,
        meta_data: None,
        created_at: Some(Utc::now()),
    }
}

fn create_request(url: &str) -> DataSourceCreateRequest {
    DataSourceCreateRequest {
        name: "Created".to_owned(),
        url: url.to_owned(),
        feed_url: None,
        source_type: String::new(),
        crawl_frequency: String::new(),
        is_enabled: None,
    }
}

fn service(store: Arc<Store>) -> DataSourceService {
    DataSourceService::new(store.clone(), store)
}

#[async_trait]
impl DataSourceRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<DataSource, AppError> {
        lock(&self.sources)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_organization_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(lock(&self.sources)?
            .iter()
            .filter(|value| value.organization_id == Some(id))
            .cloned()
            .collect())
    }

    async fn find_by_user_id(&self, id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(lock(&self.sources)?
            .iter()
            .filter(|value| value.user_id == Some(id))
            .cloned()
            .collect())
    }

    async fn find_by_url(&self, url: &str) -> Result<Option<DataSource>, AppError> {
        Ok(lock(&self.sources)?
            .iter()
            .find(|value| value.url == url)
            .cloned())
    }

    async fn find_due_to_crawl(&self, limit: i64) -> Result<Vec<DataSource>, AppError> {
        Ok(lock(&self.sources)?
            .iter()
            .filter(|value| value.is_enabled && value.crawl_status != "crawling")
            .take(limit.max(0) as usize)
            .cloned()
            .collect())
    }

    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<DataSource>, i64), AppError> {
        let values = lock(&self.sources)?;
        Ok((
            values
                .iter()
                .skip(offset.max(0) as usize)
                .take(limit.max(0) as usize)
                .cloned()
                .collect(),
            values.len() as i64,
        ))
    }

    async fn save(&self, source: &mut DataSource) -> Result<(), AppError> {
        lock(&self.sources)?.push(source.clone());
        Ok(())
    }

    async fn update(&self, source: &DataSource) -> Result<(), AppError> {
        let mut values = lock(&self.sources)?;
        let current = values
            .iter_mut()
            .find(|value| value.id == source.id)
            .ok_or(AppError::NotFound)?;
        *current = source.clone();
        Ok(())
    }

    async fn update_crawl_status(
        &self,
        id: Uuid,
        status: &str,
        error_message: Option<&str>,
    ) -> Result<(), AppError> {
        let mut values = lock(&self.sources)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.crawl_status = status.to_owned();
        value.error_message = error_message.map(str::to_owned);
        Ok(())
    }

    async fn update_next_crawl_at(
        &self,
        id: Uuid,
        next_crawl_at: DateTime<Utc>,
    ) -> Result<(), AppError> {
        let mut values = lock(&self.sources)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.next_crawl_at = Some(next_crawl_at);
        Ok(())
    }

    async fn increment_content_count(&self, id: Uuid, delta: i32) -> Result<(), AppError> {
        let mut values = lock(&self.sources)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.content_count += delta;
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.sources)?;
        let before = values.len();
        values.retain(|value| value.id != id);
        if before == values.len() {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}

#[async_trait]
impl CrawledContentRepository for Store {
    async fn find_by_data_source_id(
        &self,
        id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<CrawledContent>, i64), AppError> {
        let values: Vec<_> = lock(&self.contents)?
            .iter()
            .filter(|value| value.data_source_id == id)
            .cloned()
            .collect();
        Ok((
            values
                .iter()
                .skip(offset.max(0) as usize)
                .take(limit.max(0) as usize)
                .cloned()
                .collect(),
            values.len() as i64,
        ))
    }

    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        Ok(lock(&self.contents)?
            .iter()
            .filter(|value| ids.contains(&value.id))
            .cloned()
            .collect())
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

    async fn find_by_id(&self, id: Uuid) -> Result<CrawledContent, AppError> {
        lock(&self.contents)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_url(
        &self,
        data_source_id: Uuid,
        url: &str,
    ) -> Result<Option<CrawledContent>, AppError> {
        Ok(lock(&self.contents)?
            .iter()
            .find(|value| value.data_source_id == data_source_id && value.url == url)
            .cloned())
    }

    async fn save(&self, content: &mut CrawledContent) -> Result<(), AppError> {
        lock(&self.contents)?.push(content.clone());
        Ok(())
    }

    async fn update(&self, content: &CrawledContent) -> Result<(), AppError> {
        let mut values = lock(&self.contents)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == content.id)
            .ok_or(AppError::NotFound)?;
        *value = content.clone();
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        lock(&self.contents)?.retain(|value| value.id != id);
        Ok(())
    }

    async fn delete_by_data_source_id(&self, id: Uuid) -> Result<(), AppError> {
        lock(&self.contents)?.retain(|value| value.data_source_id != id);
        Ok(())
    }

    async fn count_by_data_source_id(&self, id: Uuid) -> Result<i64, AppError> {
        Ok(lock(&self.contents)?
            .iter()
            .filter(|value| value.data_source_id == id)
            .count() as i64)
    }
}

#[async_trait]
impl RecommendationSearchPort for Store {
    async fn search(
        &self,
        _query: &str,
        _options: SearchOptions,
    ) -> Result<SearchResponse, AppError> {
        Ok(SearchResponse {
            results: lock(&self.search_results)?.clone(),
        })
    }

    async fn find_similar(
        &self,
        _url: &str,
        _options: SimilarOptions,
    ) -> Result<SearchResponse, AppError> {
        Ok(SearchResponse {
            results: lock(&self.similar_results)?.clone(),
        })
    }

    fn is_configured(&self) -> bool {
        self.configured.lock().is_ok_and(|value| *value)
    }
}

#[tokio::test]
async fn test_service_get_by_id() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.sources)?.push(source(id, Some(Uuid::new_v4()), None, "https://one.test"));
    assert_eq!(service(store.clone()).get_by_id(id).await?.id, id);
    assert!(matches!(
        service(store).get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_list() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    lock(&store.sources)?.push(source(
        Uuid::new_v4(),
        Some(organization_id),
        None,
        "https://one.test",
    ));
    assert_eq!(service(store.clone()).list(organization_id).await?.len(), 1);
    assert!(service(store).list(Uuid::new_v4()).await?.is_empty());
    Ok(())
}

#[tokio::test]
async fn test_service_list_by_user_id() -> TestResult {
    let store = Arc::new(Store::default());
    let user_id = Uuid::new_v4();
    lock(&store.sources)?.push(source(
        Uuid::new_v4(),
        None,
        Some(user_id),
        "https://one.test",
    ));
    assert_eq!(service(store).list_by_user_id(user_id).await?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn test_service_list_all() -> TestResult {
    let store = Arc::new(Store::default());
    for index in 0..25 {
        lock(&store.sources)?.push(source(
            Uuid::new_v4(),
            None,
            Some(Uuid::new_v4()),
            &format!("https://{index}.test"),
        ));
    }
    let (values, total) = service(store.clone()).list_all(0, 0).await?;
    assert_eq!(values.len(), 20);
    assert_eq!(total, 25);
    assert_eq!(service(store).list_all(1, 101).await?.0.len(), 20);
    Ok(())
}

#[tokio::test]
async fn test_service_create() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let created = service(store.clone())
        .create(
            Some(organization_id),
            None,
            create_request("https://new.test"),
        )
        .await?;
    assert_eq!(created.source_type, "blog");
    assert_eq!(created.crawl_frequency, "daily");
    assert!(created.is_enabled);
    let user_id = Uuid::new_v4();
    service(store.clone())
        .create(None, Some(user_id), create_request("https://user.test"))
        .await?;
    assert!(matches!(
        service(store.clone())
            .create(None, None, create_request("https://none.test"))
            .await,
        Err(AppError::InvalidInput(_))
    ));
    assert!(matches!(
        service(store)
            .create(
                Some(organization_id),
                None,
                create_request("https://new.test")
            )
            .await,
        Err(AppError::Conflict(_))
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_update() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    lock(&store.sources)?.push(source(id, Some(organization_id), None, "https://old.test"));
    lock(&store.sources)?.push(source(
        Uuid::new_v4(),
        Some(organization_id),
        None,
        "https://taken.test",
    ));
    let updated = service(store.clone())
        .update(
            id,
            DataSourceUpdateRequest {
                name: Some("Updated".to_owned()),
                url: Some("https://updated.test".to_owned()),
                crawl_frequency: Some("hourly".to_owned()),
                is_enabled: Some(false),
                ..DataSourceUpdateRequest::default()
            },
        )
        .await?;
    assert_eq!(updated.name, "Updated");
    assert!(!updated.is_enabled);
    assert!(matches!(
        service(store.clone())
            .update(
                id,
                DataSourceUpdateRequest {
                    url: Some("https://taken.test".to_owned()),
                    ..DataSourceUpdateRequest::default()
                }
            )
            .await,
        Err(AppError::Conflict(_))
    ));
    assert!(matches!(
        service(store)
            .update(Uuid::new_v4(), DataSourceUpdateRequest::default())
            .await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_delete() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.sources)?.push(source(id, None, Some(Uuid::new_v4()), "https://one.test"));
    service(store.clone()).delete(id).await?;
    assert!(matches!(
        service(store).delete(id).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_get_content() -> TestResult {
    let store = Arc::new(Store::default());
    let data_source_id = Uuid::new_v4();
    lock(&store.contents)?.push(content(Uuid::new_v4(), data_source_id));
    let (values, total) = service(store).get_content(data_source_id, 0, 0).await?;
    assert_eq!(values.len(), 1);
    assert_eq!(total, 1);
    Ok(())
}

#[tokio::test]
async fn test_service_get_due_to_crawl() -> TestResult {
    let store = Arc::new(Store::default());
    lock(&store.sources)?.push(source(
        Uuid::new_v4(),
        None,
        Some(Uuid::new_v4()),
        "https://one.test",
    ));
    assert_eq!(service(store).get_due_to_crawl(10).await?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn test_service_trigger_crawl() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    let mut value = source(id, None, Some(Uuid::new_v4()), "https://one.test");
    value.crawl_status = "success".to_owned();
    lock(&store.sources)?.push(value);
    service(store.clone()).trigger_crawl(id).await?;
    assert_eq!(lock(&store.sources)?[0].crawl_status, "pending");
    assert!(matches!(
        service(store).trigger_crawl(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_update_crawl_status() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.sources)?.push(source(id, None, Some(Uuid::new_v4()), "https://one.test"));
    service(store.clone())
        .update_crawl_status(id, "failed", Some("error"))
        .await?;
    assert_eq!(
        lock(&store.sources)?[0].error_message.as_deref(),
        Some("error")
    );
    Ok(())
}

#[tokio::test]
async fn test_service_set_next_crawl_time() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.sources)?.push(source(id, None, Some(Uuid::new_v4()), "https://one.test"));
    let before = Utc::now();
    service(store.clone())
        .set_next_crawl_time(id, "weekly")
        .await?;
    assert!(
        lock(&store.sources)?[0]
            .next_crawl_at
            .is_some_and(|time| time > before)
    );
    Ok(())
}

#[tokio::test]
async fn test_service_create_discovered_source() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let from_id = Uuid::new_v4();
    let value = service(store.clone())
        .create_discovered_source(
            Some(organization_id),
            None,
            from_id,
            "Discovered".to_owned(),
            "https://discovered.test".to_owned(),
        )
        .await?;
    assert!(value.is_discovered);
    assert!(!value.is_enabled);
    assert_eq!(value.discovered_from_id, Some(from_id));
    assert!(matches!(
        service(store)
            .create_discovered_source(
                None,
                None,
                from_id,
                "None".to_owned(),
                "https://none.test".to_owned()
            )
            .await,
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}

#[tokio::test]
async fn test_recommendation_service_recommend() -> TestResult {
    let store = Arc::new(Store::default());
    *lock(&store.configured)? = true;
    let organization_id = Uuid::new_v4();
    lock(&store.sources)?.push(source(
        Uuid::new_v4(),
        Some(organization_id),
        None,
        "https://existing.test/path",
    ));
    lock(&store.search_results)?.extend([
        SearchResult {
            title: "Existing".to_owned(),
            url: "https://existing.test/other".to_owned(),
            ..SearchResult::default()
        },
        SearchResult {
            title: "New".to_owned(),
            url: "https://new-site.test/article".to_owned(),
            score: 0.9,
            summary: "Summary".to_owned(),
            ..SearchResult::default()
        },
        SearchResult {
            title: "Duplicate".to_owned(),
            url: "https://new-site.test/another".to_owned(),
            score: 0.8,
            ..SearchResult::default()
        },
    ]);
    let response = RecommendationService::new(store.clone(), store.clone())
        .recommend(
            Some(organization_id),
            None,
            DataSourceRecommendationRequest {
                query: " rust ".to_owned(),
                limit: 0,
            },
        )
        .await?;
    assert_eq!(response.query, "rust");
    assert_eq!(response.recommendations.len(), 1);
    assert_eq!(response.recommendations[0].url, "https://new-site.test");

    lock(&store.similar_results)?.push(SearchResult {
        title: "Adjacent".to_owned(),
        url: "https://adjacent.test/post".to_owned(),
        score: 1.0,
        ..SearchResult::default()
    });
    let discovery = RecommendationService::new(store.clone(), store.clone())
        .recommend_from_existing_sources(
            Some(organization_id),
            None,
            DataSourceDiscoveryRecommendationRequest { limit: 50 },
        )
        .await?;
    assert_eq!(discovery.seed_count, 1);
    assert_eq!(discovery.recommendations.len(), 1);

    *lock(&store.configured)? = false;
    assert!(matches!(
        RecommendationService::new(store.clone(), store)
            .recommend(
                Some(organization_id),
                None,
                DataSourceRecommendationRequest {
                    query: "rust".to_owned(),
                    limit: 1,
                }
            )
            .await,
        Err(AppError::External)
    ));
    Ok(())
}
