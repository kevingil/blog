use std::{
    collections::BTreeMap,
    error::Error,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        datasource::CrawledContent,
        insight::{
            ContentTopicMatch, ContentTopicMatchRepository, EmbeddingPort, Insight,
            InsightContentRepository, InsightRepository, InsightService, InsightTopic,
            InsightTopicCreateRequest, InsightTopicRepository, UserInsightStatus,
            UserInsightStatusRepository,
        },
    },
    error::AppError,
};
use chrono::{DateTime, Utc};
use uuid::Uuid;

type TestResult = Result<(), Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct Store {
    insights: Mutex<Vec<Insight>>,
    topics: Mutex<Vec<InsightTopic>>,
    embedding_inputs: Mutex<Vec<String>>,
}

fn lock<T>(value: &Mutex<T>) -> Result<MutexGuard<'_, T>, AppError> {
    value.lock().map_err(|_| AppError::Internal)
}

fn insight(id: Uuid, organization_id: Uuid) -> Insight {
    Insight {
        id,
        organization_id: Some(organization_id),
        topic_id: None,
        title: "Insight title".to_owned(),
        summary: "Insight summary".to_owned(),
        content: Some("Insight content".to_owned()),
        key_points: Some(vec!["First".to_owned()]),
        source_content_ids: Vec::new(),
        embedding: Some(vec![0.25; 4]),
        generated_at: Some(Utc::now()),
        period_start: None,
        period_end: None,
        is_read: false,
        is_pinned: false,
        is_used_in_article: false,
        meta_data: None,
    }
}

fn topic(id: Uuid, organization_id: Uuid, name: &str) -> InsightTopic {
    InsightTopic {
        id,
        organization_id: Some(organization_id),
        name: name.to_owned(),
        description: Some("Description".to_owned()),
        keywords: Some(vec!["rust".to_owned()]),
        embedding: Some(vec![0.25; 4]),
        is_auto_generated: false,
        content_count: 0,
        last_insight_at: None,
        color: None,
        icon: None,
        created_at: Some(Utc::now()),
        updated_at: Some(Utc::now()),
    }
}

fn service(store: Arc<Store>) -> InsightService {
    InsightService::new(
        store.clone(),
        store.clone(),
        store.clone(),
        store.clone(),
        store.clone(),
        store,
    )
}

#[async_trait]
impl InsightRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<Insight, AppError> {
        lock(&self.insights)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn list(&self, offset: i64, limit: i64) -> Result<(Vec<Insight>, i64), AppError> {
        let values = lock(&self.insights)?;
        Ok((slice(&values, offset, limit), values.len() as i64))
    }

    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError> {
        let values: Vec<_> = lock(&self.insights)?
            .iter()
            .filter(|value| value.organization_id == Some(organization_id))
            .cloned()
            .collect();
        Ok((slice(&values, offset, limit), values.len() as i64))
    }

    async fn find_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<Insight>, i64), AppError> {
        let values: Vec<_> = lock(&self.insights)?
            .iter()
            .filter(|value| value.topic_id == Some(topic_id))
            .cloned()
            .collect();
        Ok((slice(&values, offset, limit), values.len() as i64))
    }

    async fn find_unread(
        &self,
        organization_id: Uuid,
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        let values: Vec<_> = lock(&self.insights)?
            .iter()
            .filter(|value| value.organization_id == Some(organization_id) && !value.is_read)
            .cloned()
            .collect();
        Ok(slice(&values, 0, limit))
    }

    async fn search_similar(
        &self,
        _embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        Ok(slice(&lock(&self.insights)?, 0, limit))
    }

    async fn search_similar_by_org(
        &self,
        organization_id: Uuid,
        _embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Insight>, AppError> {
        self.find_unread(organization_id, limit).await
    }

    async fn save(&self, value: &mut Insight) -> Result<(), AppError> {
        lock(&self.insights)?.push(value.clone());
        Ok(())
    }

    async fn mark_as_read(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.insights)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.is_read = true;
        Ok(())
    }

    async fn toggle_pinned(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.insights)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.is_pinned = !value.is_pinned;
        Ok(())
    }

    async fn mark_as_used_in_article(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.insights)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.is_used_in_article = true;
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.insights)?;
        let before = values.len();
        values.retain(|value| value.id != id);
        if values.len() == before {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }

    async fn count_unread(&self, organization_id: Uuid) -> Result<i64, AppError> {
        Ok(lock(&self.insights)?
            .iter()
            .filter(|value| value.organization_id == Some(organization_id) && !value.is_read)
            .count() as i64)
    }

    async fn count_all_unread(&self) -> Result<i64, AppError> {
        Ok(lock(&self.insights)?
            .iter()
            .filter(|value| !value.is_read)
            .count() as i64)
    }
}

#[async_trait]
impl InsightTopicRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<InsightTopic, AppError> {
        lock(&self.topics)?
            .iter()
            .find(|value| value.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
    ) -> Result<Vec<InsightTopic>, AppError> {
        Ok(lock(&self.topics)?
            .iter()
            .filter(|value| value.organization_id == Some(organization_id))
            .cloned()
            .collect())
    }

    async fn find_all(&self) -> Result<Vec<InsightTopic>, AppError> {
        Ok(lock(&self.topics)?.clone())
    }

    async fn search_similar(
        &self,
        _embedding: &[f32],
        limit: i64,
        _threshold: f64,
    ) -> Result<(Vec<InsightTopic>, Vec<f64>), AppError> {
        let values = slice(&lock(&self.topics)?, 0, limit);
        Ok((values.clone(), vec![1.0; values.len()]))
    }

    async fn save(&self, value: &mut InsightTopic) -> Result<(), AppError> {
        lock(&self.topics)?.push(value.clone());
        Ok(())
    }

    async fn update(&self, value: &InsightTopic) -> Result<(), AppError> {
        let mut values = lock(&self.topics)?;
        let current = values
            .iter_mut()
            .find(|current| current.id == value.id)
            .ok_or(AppError::NotFound)?;
        *current = value.clone();
        Ok(())
    }

    async fn update_content_count(&self, id: Uuid, count: i32) -> Result<(), AppError> {
        let mut values = lock(&self.topics)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.content_count = count;
        Ok(())
    }

    async fn update_last_insight_at(
        &self,
        id: Uuid,
        timestamp: DateTime<Utc>,
    ) -> Result<(), AppError> {
        let mut values = lock(&self.topics)?;
        let value = values
            .iter_mut()
            .find(|value| value.id == id)
            .ok_or(AppError::NotFound)?;
        value.last_insight_at = Some(timestamp);
        Ok(())
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut values = lock(&self.topics)?;
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
impl UserInsightStatusRepository for Store {
    async fn find_by_user_and_insight(
        &self,
        _user_id: Uuid,
        _insight_id: Uuid,
    ) -> Result<Option<UserInsightStatus>, AppError> {
        Ok(None)
    }

    async fn mark_as_read(&self, _user_id: Uuid, _insight_id: Uuid) -> Result<(), AppError> {
        Ok(())
    }

    async fn toggle_pinned(&self, _user_id: Uuid, _insight_id: Uuid) -> Result<bool, AppError> {
        Ok(true)
    }

    async fn mark_as_used_in_article(
        &self,
        _user_id: Uuid,
        _insight_id: Uuid,
    ) -> Result<(), AppError> {
        Ok(())
    }

    async fn get_status_map_for_insights(
        &self,
        _user_id: Uuid,
        _insight_ids: &[Uuid],
    ) -> Result<BTreeMap<Uuid, UserInsightStatus>, AppError> {
        Ok(BTreeMap::new())
    }

    async fn count_unread_by_user_id(&self, _user_id: Uuid) -> Result<i64, AppError> {
        Ok(0)
    }
}

#[async_trait]
impl InsightContentRepository for Store {
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
}

#[async_trait]
impl ContentTopicMatchRepository for Store {
    async fn save_batch(&self, _matches: &mut [ContentTopicMatch]) -> Result<(), AppError> {
        Ok(())
    }

    async fn count_by_topic_id(&self, _topic_id: Uuid) -> Result<i64, AppError> {
        Ok(0)
    }

    async fn find_primary_by_topic_id(
        &self,
        _topic_id: Uuid,
        _offset: i64,
        _limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        Ok((Vec::new(), 0))
    }
}

#[async_trait]
impl EmbeddingPort for Store {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError> {
        lock(&self.embedding_inputs)?.push(text.to_owned());
        Ok(vec![0.5; 4])
    }
}

fn slice<T: Clone>(values: &[T], offset: i64, limit: i64) -> Vec<T> {
    values
        .iter()
        .skip(offset.max(0) as usize)
        .take(limit.max(0) as usize)
        .cloned()
        .collect()
}

#[tokio::test]
async fn test_service_get_insight_by_id() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let mut saved = insight(id, organization_id);
    saved.title = "Test Insight".to_owned();
    saved.summary = "Test summary".to_owned();
    saved.content = Some("Test content".to_owned());
    saved.key_points = Some(vec!["point1".to_owned(), "point2".to_owned()]);
    let generated_at = saved.generated_at;
    lock(&store.insights)?.push(saved);

    let value = service(store.clone()).get_insight_by_id(id).await?;
    assert_eq!(value.id, id);
    assert_eq!(value.organization_id, Some(organization_id));
    assert_eq!(value.title, "Test Insight");
    assert_eq!(value.summary, "Test summary");
    assert_eq!(value.content.as_deref(), Some("Test content"));
    assert_eq!(
        value.key_points,
        Some(vec!["point1".to_owned(), "point2".to_owned()])
    );
    assert_eq!(value.generated_at, generated_at);
    assert!(!value.is_read);
    assert!(!value.is_pinned);

    let missing = service(store).get_insight_by_id(Uuid::new_v4()).await;
    assert!(matches!(missing, Err(AppError::NotFound)));
    Ok(())
}

#[tokio::test]
async fn test_service_list_insights() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let other_organization_id = Uuid::new_v4();
    for index in 0..25 {
        let mut value = insight(Uuid::new_v4(), organization_id);
        value.title = format!("Insight {index}");
        lock(&store.insights)?.push(value);
    }
    lock(&store.insights)?.push(insight(Uuid::new_v4(), other_organization_id));

    let (values, total) = service(store.clone())
        .list_insights(organization_id, 1, 20)
        .await?;
    assert_eq!(values.len(), 20);
    assert_eq!(total, 25);
    assert_eq!(values[0].title, "Insight 0");
    assert_eq!(values[19].title, "Insight 19");

    let (values, total) = service(store.clone())
        .list_insights(organization_id, 0, 0)
        .await?;
    assert_eq!(values.len(), 20, "invalid pagination uses limit 20");
    assert_eq!(total, 25);

    let (values, total) = service(store)
        .list_insights(organization_id, 1, 500)
        .await?;
    assert_eq!(values.len(), 20, "oversized pagination uses limit 20");
    assert_eq!(total, 25);
    Ok(())
}

#[tokio::test]
async fn test_service_create_insight() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let topic_id = Uuid::new_v4();
    let source_id = Uuid::new_v4();
    lock(&store.topics)?.push(topic(topic_id, organization_id, "Rust"));

    let value = service(store.clone())
        .create_insight(
            Some(organization_id),
            Some(topic_id),
            "New Insight".to_owned(),
            "New summary".to_owned(),
            "New content".to_owned(),
            Some(vec!["point1".to_owned(), "point2".to_owned()]),
            Some(vec![source_id]),
            None,
            None,
        )
        .await?;
    assert_eq!(value.organization_id, Some(organization_id));
    assert_eq!(value.topic_id, Some(topic_id));
    assert_eq!(value.title, "New Insight");
    assert_eq!(value.summary, "New summary");
    assert_eq!(value.content.as_deref(), Some("New content"));
    assert_eq!(
        value.key_points,
        Some(vec!["point1".to_owned(), "point2".to_owned()])
    );
    assert_eq!(value.source_content_ids, vec![source_id]);
    assert!(value.generated_at.is_some());
    assert!(!value.is_read);
    assert!(!value.is_pinned);
    assert!(!value.is_used_in_article);
    assert!(lock(&store.topics)?[0].last_insight_at.is_some());
    assert_eq!(
        lock(&store.embedding_inputs)?.as_slice(),
        ["New Insight New summary"]
    );

    let without_topic = service(store.clone())
        .create_insight(
            Some(organization_id),
            None,
            "Second Insight".to_owned(),
            "Second summary".to_owned(),
            "Second content".to_owned(),
            None,
            None,
            None,
            None,
        )
        .await?;
    assert_eq!(without_topic.topic_id, None);
    assert_eq!(without_topic.key_points, None);
    assert!(without_topic.source_content_ids.is_empty());
    assert_eq!(
        lock(&store.embedding_inputs)?.as_slice(),
        [
            "New Insight New summary".to_owned(),
            "Second Insight Second summary".to_owned()
        ]
    );
    assert_eq!(lock(&store.insights)?.len(), 2);
    Ok(())
}

#[tokio::test]
async fn test_service_delete_insight() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.insights)?.push(insight(id, Uuid::new_v4()));
    service(store.clone()).delete_insight(id).await?;
    assert!(lock(&store.insights)?.is_empty());
    let missing = service(store).delete_insight(Uuid::new_v4()).await;
    assert!(matches!(missing, Err(AppError::NotFound)));
    Ok(())
}

#[tokio::test]
async fn test_service_get_topic_by_id() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    let organization_id = Uuid::new_v4();
    let mut saved = topic(id, organization_id, "Test Topic");
    saved.description = Some("Test description".to_owned());
    saved.keywords = Some(vec!["keyword1".to_owned(), "keyword2".to_owned()]);
    saved.content_count = 5;
    let created_at = saved.created_at;
    let updated_at = saved.updated_at;
    lock(&store.topics)?.push(saved);

    let value = service(store.clone()).get_topic_by_id(id).await?;
    assert_eq!(value.id, id);
    assert_eq!(value.organization_id, Some(organization_id));
    assert_eq!(value.name, "Test Topic");
    assert_eq!(value.description.as_deref(), Some("Test description"));
    assert_eq!(
        value.keywords,
        Some(vec!["keyword1".to_owned(), "keyword2".to_owned()])
    );
    assert_eq!(value.content_count, 5);
    assert_eq!(value.created_at, created_at);
    assert_eq!(value.updated_at, updated_at);

    let missing = service(store).get_topic_by_id(Uuid::new_v4()).await;
    assert!(matches!(missing, Err(AppError::NotFound)));
    Ok(())
}

#[tokio::test]
async fn test_service_list_topics() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let mut first = topic(Uuid::new_v4(), organization_id, "Topic 1");
    first.content_count = 3;
    let mut second = topic(Uuid::new_v4(), organization_id, "Topic 2");
    second.content_count = 7;
    second.is_auto_generated = true;
    lock(&store.topics)?.extend([first, second]);

    let values = service(store.clone()).list_topics(organization_id).await?;
    assert_eq!(values.len(), 2);
    assert_eq!(values[0].name, "Topic 1");
    assert_eq!(values[0].content_count, 3);
    assert_eq!(values[1].name, "Topic 2");
    assert_eq!(values[1].content_count, 7);
    assert!(values[1].is_auto_generated);

    let empty = service(store).list_topics(Uuid::new_v4()).await?;
    assert!(empty.is_empty());
    Ok(())
}

#[tokio::test]
async fn test_service_create_topic() -> TestResult {
    let store = Arc::new(Store::default());
    let organization_id = Uuid::new_v4();
    let value = service(store.clone())
        .create_topic(
            Some(organization_id),
            InsightTopicCreateRequest {
                name: "New Topic".to_owned(),
                description: Some("Test description".to_owned()),
                keywords: Some(vec!["keyword1".to_owned(), "keyword2".to_owned()]),
                color: Some("#FF0000".to_owned()),
                icon: Some("star".to_owned()),
            },
        )
        .await?;
    assert_eq!(value.organization_id, Some(organization_id));
    assert_eq!(value.name, "New Topic");
    assert_eq!(value.description.as_deref(), Some("Test description"));
    assert_eq!(
        value.keywords,
        Some(vec!["keyword1".to_owned(), "keyword2".to_owned()])
    );
    assert_eq!(value.color.as_deref(), Some("#FF0000"));
    assert_eq!(value.icon.as_deref(), Some("star"));
    assert!(!value.is_auto_generated);
    assert_eq!(value.content_count, 0);
    assert_eq!(
        lock(&store.embedding_inputs)?.as_slice(),
        ["New Topic Test description keyword1 keyword2"]
    );

    let minimal = service(store.clone())
        .create_topic(
            Some(organization_id),
            InsightTopicCreateRequest {
                name: "Minimal Topic".to_owned(),
                description: None,
                keywords: None,
                color: None,
                icon: None,
            },
        )
        .await?;
    assert_eq!(minimal.name, "Minimal Topic");
    assert_eq!(minimal.description, None);
    assert_eq!(minimal.keywords, None);
    assert_eq!(minimal.color, None);
    assert_eq!(minimal.icon, None);
    assert_eq!(
        lock(&store.embedding_inputs)?.as_slice(),
        [
            "New Topic Test description keyword1 keyword2".to_owned(),
            "Minimal Topic".to_owned()
        ]
    );
    assert_eq!(lock(&store.topics)?.len(), 2);
    Ok(())
}

#[tokio::test]
async fn test_service_delete_topic() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.topics)?.push(topic(id, Uuid::new_v4(), "Rust"));
    service(store.clone()).delete_topic(id).await?;
    assert!(lock(&store.topics)?.is_empty());
    let missing = service(store).delete_topic(Uuid::new_v4()).await;
    assert!(matches!(missing, Err(AppError::NotFound)));
    Ok(())
}

#[tokio::test]
async fn test_service_mark_insight_as_read() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.insights)?.push(insight(id, Uuid::new_v4()));
    service(store.clone()).mark_insight_as_read(id).await?;
    assert!(lock(&store.insights)?[0].is_read);
    Ok(())
}

#[tokio::test]
async fn test_service_toggle_insight_pinned() -> TestResult {
    let store = Arc::new(Store::default());
    let id = Uuid::new_v4();
    lock(&store.insights)?.push(insight(id, Uuid::new_v4()));
    service(store.clone()).toggle_insight_pinned(id).await?;
    assert!(lock(&store.insights)?[0].is_pinned);
    Ok(())
}

#[tokio::test]
async fn test_service_list_all_topics() -> TestResult {
    let store = Arc::new(Store::default());
    lock(&store.topics)?.push(topic(Uuid::new_v4(), Uuid::new_v4(), "First"));
    lock(&store.topics)?.push(topic(Uuid::new_v4(), Uuid::new_v4(), "Second"));
    assert_eq!(service(store).list_all_topics().await?.len(), 2);
    Ok(())
}
