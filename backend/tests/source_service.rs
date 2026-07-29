use std::{
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::source::{
        AgentResourceSelection, ArticleLookupPort, CreateSourceRequest, EmbeddingPort,
        FetchExtractPort, ScrapedContent, Source, SourceListOptions, SourceRepository,
        SourceService, SourceWithArticle, UpdateSourceRequest,
    },
    error::AppError,
};
use serde_json::Value;
use uuid::Uuid;

type TestResult = Result<(), Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct Store {
    values: Mutex<Vec<Source>>,
}

fn lock<T>(value: &Mutex<T>) -> Result<MutexGuard<'_, T>, AppError> {
    value.lock().map_err(|_| AppError::Internal)
}

fn source(id: Uuid, article_id: Uuid, url: &str) -> Source {
    Source {
        id,
        article_id,
        title: "Source title".to_owned(),
        content: "Source content".to_owned(),
        url: url.to_owned(),
        source_type: "web".to_owned(),
        embedding: Some(vec![0.5; 4]),
        meta_data: None,
        created_at: None,
    }
}

fn resource(article_id: Uuid) -> AgentResourceSelection {
    AgentResourceSelection {
        article_id,
        source_id: None,
        title: "Selected source".to_owned(),
        content: "Full content".to_owned(),
        url: "https://example.com/resource".to_owned(),
        source_type: "web".to_owned(),
        origin_tool: "exa_search".to_owned(),
        origin_query: "rust".to_owned(),
        origin_question: String::new(),
        author: "Author".to_owned(),
        published_date: String::new(),
        selected_excerpt: "Selected excerpt".to_owned(),
        selected_excerpt_id: "excerpt-1".to_owned(),
        request_id: "request-1".to_owned(),
        usage_status: String::new(),
    }
}

fn service(store: Arc<Store>) -> SourceService {
    SourceService::new(store.clone(), store.clone(), store.clone(), store)
}

#[async_trait]
impl SourceRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<Source, AppError> {
        lock(&self.values)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_article_id(&self, article_id: Uuid) -> Result<Vec<Source>, AppError> {
        Ok(lock(&self.values)?
            .iter()
            .filter(|value| value.article_id == article_id)
            .cloned()
            .collect())
    }

    async fn list(
        &self,
        options: SourceListOptions,
    ) -> Result<(Vec<SourceWithArticle>, i64), AppError> {
        let values = lock(&self.values)?;
        let total = values.len() as i64;
        let offset = (options.page.max(1) - 1) * options.per_page;
        Ok((
            values
                .iter()
                .skip(offset.max(0) as usize)
                .take(options.per_page.max(0) as usize)
                .cloned()
                .map(|source| SourceWithArticle {
                    source,
                    article_title: "Article".to_owned(),
                    article_slug: "article".to_owned(),
                })
                .collect(),
            total,
        ))
    }

    async fn save(&self, source: &mut Source) -> Result<(), AppError> {
        lock(&self.values)?.push(source.clone());
        Ok(())
    }

    async fn update(&self, source: &Source) -> Result<(), AppError> {
        let mut values = lock(&self.values)?;
        let current = values
            .iter_mut()
            .find(|value| value.id == source.id)
            .ok_or(AppError::NotFound)?;
        *current = source.clone();
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.values)?;
        let before = values.len();
        values.retain(|value| value.id != id);
        if values.len() == before {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn search_similar(
        &self,
        article_id: Uuid,
        _embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Source>, AppError> {
        Ok(lock(&self.values)?
            .iter()
            .filter(|value| value.article_id == article_id)
            .take(limit.max(0) as usize)
            .cloned()
            .collect())
    }
}

#[async_trait]
impl ArticleLookupPort for Store {
    async fn ensure_exists(&self, _article_id: Uuid) -> Result<(), AppError> {
        Ok(())
    }
}

#[async_trait]
impl EmbeddingPort for Store {
    async fn generate_embedding(&self, _text: &str) -> Result<Vec<f32>, AppError> {
        Ok(vec![0.25; 4])
    }
}

#[async_trait]
impl FetchExtractPort for Store {
    async fn fetch_extract(&self, url: &str) -> Result<ScrapedContent, AppError> {
        Ok(ScrapedContent {
            title: "Scraped".to_owned(),
            content: "Scraped content".to_owned(),
            url: url.to_owned(),
        })
    }
}

#[tokio::test]
async fn test_service_get_by_id() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.values)?.push(source(id, Uuid::new_v4(), ""));
    assert_eq!(service(store.clone()).get_by_id(id).await?.id, id);
    assert!(matches!(
        service(store).get_by_id(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_get_by_article_id() -> TestResult {
    let store = Arc::new(Store::default());
    let article_id = Uuid::new_v4();
    lock(&store.values)?.push(source(Uuid::new_v4(), article_id, ""));
    assert_eq!(service(store).get_by_article_id(article_id).await?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn test_service_list() -> TestResult {
    let store = Arc::new(Store::default());
    let article_id = Uuid::new_v4();
    for _ in 0..21 {
        lock(&store.values)?.push(source(Uuid::new_v4(), article_id, ""));
    }
    let normalized = service(store.clone()).list(0, 0).await?;
    assert_eq!(normalized.page, 1);
    assert_eq!(normalized.sources.len(), 20);
    assert_eq!(normalized.total_pages, 2);
    let capped = service(store).list(1, 101).await?;
    assert_eq!(capped.sources.len(), 20);
    Ok(())
}

#[tokio::test]
async fn test_service_delete() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.values)?.push(source(id, Uuid::new_v4(), ""));
    service(store.clone()).delete(id).await?;
    assert!(lock(&store.values)?.is_empty());
    assert!(matches!(
        service(store).delete(Uuid::new_v4()).await,
        Err(AppError::NotFound)
    ));
    Ok(())
}

#[tokio::test]
async fn test_service_upsert_agent_resource_creates_source() -> TestResult {
    let store = Arc::new(Store::default());
    let value = service(store.clone())
        .upsert_agent_resource(resource(Uuid::new_v4()))
        .await?;
    let metadata = value.meta_data.ok_or(AppError::Internal)?;
    let resource = metadata
        .get("resource")
        .and_then(Value::as_object)
        .ok_or(AppError::Internal)?;
    assert_eq!(
        resource.get("usage_status"),
        Some(&Value::String("used".to_owned()))
    );
    assert_eq!(lock(&store.values)?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn test_service_upsert_agent_resource_updates_existing_source() -> TestResult {
    let store = Arc::new(Store::default());
    let article_id = Uuid::new_v4();
    let id = Uuid::new_v4();
    lock(&store.values)?.push(source(id, article_id, "https://EXAMPLE.com/resource "));
    let value = service(store.clone())
        .upsert_agent_resource(resource(article_id))
        .await?;
    assert_eq!(value.id, id);
    assert_eq!(lock(&store.values)?.len(), 1);
    Ok(())
}

#[tokio::test]
async fn create_update_search_and_scrape_use_injected_ports() -> TestResult {
    let store = Arc::new(Store::default());
    let article_id = Uuid::new_v4();
    let value = service(store.clone())
        .create(CreateSourceRequest {
            article_id,
            title: "Manual".to_owned(),
            content: "Body".to_owned(),
            url: String::new(),
            source_type: String::new(),
            meta_data: None,
        })
        .await?;
    assert_eq!(value.source_type, "manual");
    let updated = service(store.clone())
        .update(
            value.id,
            UpdateSourceRequest {
                content: Some("Updated".to_owned()),
                ..UpdateSourceRequest::default()
            },
        )
        .await?;
    assert_eq!(updated.embedding, Some(vec![0.25; 4]));
    assert_eq!(
        service(store.clone())
            .search_similar(article_id, "query", 5)
            .await?
            .len(),
        1
    );
    let scraped = service(store)
        .scrape_and_create(article_id, "https://example.com/document.pdf")
        .await?;
    assert_eq!(scraped.source_type, "pdf");
    Ok(())
}
