use std::{
    error::Error,
    fmt::Debug,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        datasource::CrawledContent,
        insight::{
            ContentTopicMatch, ContentTopicMatchRepository, InsightContentRepository, InsightTopic,
            InsightTopicRepository,
        },
        worker::{Clock, InsightGenerationPort, InsightTopicResult, WorkerFailure},
    },
    error::AppError,
};
use chrono::{DateTime, Duration, Utc};
use tokio_util::sync::CancellationToken;
use uuid::Uuid;

pub use blog_backend::{core, error, integrations};

#[path = "../src/runtime/worker_adapters.rs"]
mod worker_adapters;

use worker_adapters::{
    GeneratedInsight, InsightGenerationRequest, InsightTextGenerator, InsightWriter, NewInsight,
    RuntimeInsightGenerator, decode_generated_insight,
};

type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

fn lock<T>(value: &Mutex<T>) -> MutexGuard<'_, T> {
    value
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

fn worker_failure<T: Debug>(
    result: Result<T, WorkerFailure>,
    context: &str,
) -> TestResult<WorkerFailure> {
    match result {
        Err(error) => Ok(error),
        Ok(value) => Err(format!("{context}; got {value:?}").into()),
    }
}

#[derive(Default)]
struct Store {
    topics: Mutex<Vec<InsightTopic>>,
    matches: Mutex<Vec<ContentTopicMatch>>,
    match_total: Mutex<i64>,
    contents: Mutex<Vec<CrawledContent>>,
    updated_topics: Mutex<Vec<(Uuid, DateTime<Utc>)>>,
    fail_topics: Mutex<bool>,
    fail_matches: Mutex<bool>,
    fail_contents: Mutex<bool>,
    fail_update: Mutex<bool>,
}

#[async_trait]
impl InsightTopicRepository for Store {
    async fn find_by_id(&self, id: Uuid) -> Result<InsightTopic, AppError> {
        lock(&self.topics)
            .iter()
            .find(|topic| topic.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn find_by_organization_id(
        &self,
        organization_id: Uuid,
    ) -> Result<Vec<InsightTopic>, AppError> {
        Ok(lock(&self.topics)
            .iter()
            .filter(|topic| topic.organization_id == Some(organization_id))
            .cloned()
            .collect())
    }

    async fn find_all(&self) -> Result<Vec<InsightTopic>, AppError> {
        if *lock(&self.fail_topics) {
            return Err(AppError::Database);
        }
        Ok(lock(&self.topics).clone())
    }

    async fn search_similar(
        &self,
        _embedding: &[f32],
        _limit: i64,
        _threshold: f64,
    ) -> Result<(Vec<InsightTopic>, Vec<f64>), AppError> {
        Ok((Vec::new(), Vec::new()))
    }

    async fn save(&self, topic: &mut InsightTopic) -> Result<(), AppError> {
        lock(&self.topics).push(topic.clone());
        Ok(())
    }

    async fn update(&self, _topic: &InsightTopic) -> Result<(), AppError> {
        Ok(())
    }

    async fn update_content_count(&self, _id: Uuid, _count: i32) -> Result<(), AppError> {
        Ok(())
    }

    async fn update_last_insight_at(
        &self,
        id: Uuid,
        timestamp: DateTime<Utc>,
    ) -> Result<(), AppError> {
        if *lock(&self.fail_update) {
            return Err(AppError::Database);
        }
        lock(&self.updated_topics).push((id, timestamp));
        Ok(())
    }

    async fn delete(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }
}

#[async_trait]
impl ContentTopicMatchRepository for Store {
    async fn save_batch(&self, _matches: &mut [ContentTopicMatch]) -> Result<(), AppError> {
        Ok(())
    }

    async fn count_by_topic_id(&self, topic_id: Uuid) -> Result<i64, AppError> {
        i64::try_from(
            lock(&self.matches)
                .iter()
                .filter(|value| value.topic_id == topic_id)
                .count(),
        )
        .map_err(|_| AppError::Internal)
    }

    async fn find_primary_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        if *lock(&self.fail_matches) {
            return Err(AppError::Database);
        }
        assert_eq!(offset, 0);
        assert_eq!(limit, 10);
        Ok((
            lock(&self.matches)
                .iter()
                .filter(|value| value.topic_id == topic_id && value.is_primary)
                .cloned()
                .collect(),
            *lock(&self.match_total),
        ))
    }
}

#[async_trait]
impl InsightContentRepository for Store {
    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        if *lock(&self.fail_contents) {
            return Err(AppError::Database);
        }
        let values = lock(&self.contents);
        Ok(ids
            .iter()
            .filter_map(|id| values.iter().find(|value| value.id == *id).cloned())
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
}

struct Text {
    configured: bool,
    response: Mutex<Option<GeneratedInsight>>,
    requests: Mutex<Vec<InsightGenerationRequest>>,
    fail: Mutex<bool>,
}

impl Text {
    fn successful() -> Self {
        Self {
            configured: true,
            response: Mutex::new(Some(GeneratedInsight {
                title: "Generated title".to_owned(),
                summary: "Generated summary".to_owned(),
                content: "Generated content".to_owned(),
                key_points: vec![
                    "First point".to_owned(),
                    "Second point".to_owned(),
                    "Third point".to_owned(),
                ],
            })),
            requests: Mutex::new(Vec::new()),
            fail: Mutex::new(false),
        }
    }
}

#[async_trait]
impl InsightTextGenerator for Text {
    fn is_configured(&self) -> bool {
        self.configured
    }

    async fn generate_insight(
        &self,
        request: InsightGenerationRequest,
    ) -> Result<GeneratedInsight, AppError> {
        lock(&self.requests).push(request);
        if *lock(&self.fail) {
            return Err(AppError::External);
        }
        lock(&self.response).clone().ok_or(AppError::External)
    }
}

#[derive(Default)]
struct Writer {
    values: Mutex<Vec<NewInsight>>,
    fail: Mutex<bool>,
}

#[async_trait]
impl InsightWriter for Writer {
    async fn create(&self, insight: NewInsight) -> Result<(), AppError> {
        if *lock(&self.fail) {
            return Err(AppError::Database);
        }
        lock(&self.values).push(insight);
        Ok(())
    }
}

struct FixedClock(DateTime<Utc>);

impl Clock for FixedClock {
    fn now(&self) -> DateTime<Utc> {
        self.0
    }
}

struct Fixture {
    generator: RuntimeInsightGenerator,
    store: Arc<Store>,
    text: Arc<Text>,
    writer: Arc<Writer>,
}

fn fixture(now: DateTime<Utc>) -> Fixture {
    let store = Arc::new(Store::default());
    let text = Arc::new(Text::successful());
    let writer = Arc::new(Writer::default());
    let generator = RuntimeInsightGenerator::new(
        store.clone(),
        store.clone(),
        store.clone(),
        text.clone(),
        writer.clone(),
        Arc::new(FixedClock(now)),
    );
    Fixture {
        generator,
        store,
        text,
        writer,
    }
}

fn topic(last_insight_at: Option<DateTime<Utc>>) -> InsightTopic {
    InsightTopic {
        id: Uuid::new_v4(),
        organization_id: Some(Uuid::new_v4()),
        name: "Rust systems".to_owned(),
        description: Some("Reliable systems programming".to_owned()),
        keywords: Some(vec!["rust".to_owned()]),
        embedding: None,
        is_auto_generated: false,
        content_count: 3,
        last_insight_at,
        color: None,
        icon: None,
        created_at: None,
        updated_at: None,
    }
}

fn content(index: usize, published_at: Option<DateTime<Utc>>, body: String) -> CrawledContent {
    CrawledContent {
        id: Uuid::new_v4(),
        data_source_id: Uuid::new_v4(),
        url: format!("https://example.test/{index}"),
        title: Some(format!("Article {index}")),
        content: body,
        summary: None,
        author: None,
        published_at,
        embedding: None,
        meta_data: None,
        created_at: None,
    }
}

fn seed(fixture: &Fixture, topic: &InsightTopic, contents: Vec<CrawledContent>, total: i64) {
    *lock(&fixture.store.match_total) = total;
    *lock(&fixture.store.matches) = contents
        .iter()
        .map(|content| ContentTopicMatch {
            id: Uuid::new_v4(),
            content_id: content.id,
            topic_id: topic.id,
            similarity_score: 0.9,
            is_primary: true,
            created_at: None,
        })
        .collect();
    *lock(&fixture.store.contents) = contents;
}

fn now() -> DateTime<Utc> {
    DateTime::<Utc>::UNIX_EPOCH + Duration::days(20_000)
}

#[tokio::test]
async fn topics_and_configuration_are_delegated_without_global_state() -> TestResult {
    let fixture = fixture(now());
    let expected = topic(None);
    lock(&fixture.store.topics).push(expected.clone());
    assert!(fixture.generator.is_configured());
    assert_eq!(fixture.generator.topics().await?, vec![expected]);

    *lock(&fixture.store.fail_topics) = true;
    let error = worker_failure(
        fixture.generator.topics().await,
        "topic repository failure must be blocking",
    )?;
    assert!(error.message().contains("failed to list insight topics"));
    Ok(())
}

#[tokio::test]
async fn insufficient_match_total_or_loaded_content_skips_before_provider() -> TestResult {
    let fixture = fixture(now());
    let topic = topic(None);
    seed(
        &fixture,
        &topic,
        vec![
            content(1, None, "one".to_owned()),
            content(2, None, "two".to_owned()),
        ],
        2,
    );
    assert_eq!(
        fixture
            .generator
            .generate_for_topic(&topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::SkippedInsufficient
    );
    assert!(lock(&fixture.text.requests).is_empty());

    *lock(&fixture.store.match_total) = 3;
    assert_eq!(
        fixture
            .generator
            .generate_for_topic(&topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::SkippedInsufficient
    );
    assert!(lock(&fixture.text.requests).is_empty());
    assert!(lock(&fixture.writer.values).is_empty());
    Ok(())
}

#[tokio::test]
async fn recent_window_skips_but_exact_twenty_four_hour_boundary_generates() -> TestResult {
    let recent_fixture = fixture(now());
    let recent_topic = topic(Some(now() - Duration::hours(23)));
    seed(
        &recent_fixture,
        &recent_topic,
        (1..=3)
            .map(|index| content(index, None, format!("body {index}")))
            .collect(),
        3,
    );
    assert_eq!(
        recent_fixture
            .generator
            .generate_for_topic(&recent_topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::SkippedRecent
    );
    assert!(lock(&recent_fixture.text.requests).is_empty());

    let boundary_fixture = fixture(now());
    let boundary_topic = topic(Some(now() - Duration::hours(24)));
    seed(
        &boundary_fixture,
        &boundary_topic,
        (1..=3)
            .map(|index| content(index, None, format!("body {index}")))
            .collect(),
        3,
    );
    assert_eq!(
        boundary_fixture
            .generator
            .generate_for_topic(&boundary_topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::Created
    );
    assert_eq!(lock(&boundary_fixture.writer.values).len(), 1);
    Ok(())
}

#[tokio::test]
async fn successful_generation_preserves_prompt_ids_unicode_and_published_period() -> TestResult {
    let fixture = fixture(now());
    let topic = topic(None);
    let earliest = now() - Duration::days(5);
    let latest = now() - Duration::days(1);
    let contents = vec![
        content(1, Some(latest), "é".repeat(1_501)),
        content(2, Some(earliest), "second".to_owned()),
        content(3, Some(now() - Duration::days(3)), "third".to_owned()),
    ];
    let ids = contents.iter().map(|value| value.id).collect::<Vec<_>>();
    seed(&fixture, &topic, contents, 3);

    assert_eq!(
        fixture
            .generator
            .generate_for_topic(&topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::Created
    );
    let requests = lock(&fixture.text.requests);
    assert_eq!(requests.len(), 1);
    assert_eq!(requests[0].topic.name, topic.name);
    assert_eq!(requests[0].articles[0].content.chars().count(), 1_503);
    assert!(requests[0].articles[0].content.ends_with("..."));
    drop(requests);

    let values = lock(&fixture.writer.values);
    assert_eq!(values.len(), 1);
    assert_eq!(values[0].source_content_ids, ids);
    assert_eq!(values[0].period_start, earliest);
    assert_eq!(values[0].period_end, latest);
    assert_eq!(values[0].organization_id, topic.organization_id);
    assert_eq!(values[0].topic_id, topic.id);
    drop(values);
    assert_eq!(
        lock(&fixture.store.updated_topics).as_slice(),
        &[(topic.id, now())]
    );
    Ok(())
}

#[tokio::test]
async fn no_published_dates_uses_clocked_seven_day_period() -> TestResult {
    let fixture = fixture(now());
    let topic = topic(None);
    seed(
        &fixture,
        &topic,
        (1..=3)
            .map(|index| content(index, None, format!("body {index}")))
            .collect(),
        3,
    );
    fixture
        .generator
        .generate_for_topic(&topic, &CancellationToken::new())
        .await?;
    let values = lock(&fixture.writer.values);
    assert_eq!(values[0].period_start, now() - Duration::days(7));
    assert_eq!(values[0].period_end, now());
    Ok(())
}

#[tokio::test]
async fn cancellation_and_every_data_boundary_error_are_blocking() -> TestResult {
    let cancelled_fixture = fixture(now());
    let cancelled_topic = topic(None);
    seed(
        &cancelled_fixture,
        &cancelled_topic,
        (1..=3)
            .map(|index| content(index, None, format!("body {index}")))
            .collect(),
        3,
    );
    let cancellation = CancellationToken::new();
    cancellation.cancel();
    let error = worker_failure(
        cancelled_fixture
            .generator
            .generate_for_topic(&cancelled_topic, &cancellation)
            .await,
        "cancellation must stop generation",
    )?;
    assert_eq!(error.message(), "operation cancelled");
    assert!(lock(&cancelled_fixture.text.requests).is_empty());

    for failure in ["matches", "contents", "text", "writer", "topic-update"] {
        let fixture = fixture(now());
        let topic = topic(None);
        seed(
            &fixture,
            &topic,
            (1..=3)
                .map(|index| content(index, None, format!("body {index}")))
                .collect(),
            3,
        );
        match failure {
            "matches" => *lock(&fixture.store.fail_matches) = true,
            "contents" => *lock(&fixture.store.fail_contents) = true,
            "text" => *lock(&fixture.text.fail) = true,
            "writer" => *lock(&fixture.writer.fail) = true,
            "topic-update" => *lock(&fixture.store.fail_update) = true,
            _ => unreachable!(),
        }
        let error = worker_failure(
            fixture
                .generator
                .generate_for_topic(&topic, &CancellationToken::new())
                .await,
            "data boundary errors must be blocking",
        )?;
        let expected = match failure {
            "matches" => "failed to get content matches",
            "contents" => "failed to get content details",
            "text" => "failed to generate structured insight",
            "writer" => "failed to create insight",
            "topic-update" => "failed to update topic insight timestamp",
            _ => unreachable!(),
        };
        assert!(error.message().contains(expected), "{failure}: {error}");
    }
    Ok(())
}

#[test]
fn generated_types_have_no_implicit_or_heuristic_defaults() {
    let malformed = serde_json::from_value::<GeneratedInsight>(serde_json::json!({
        "title": "Title",
        "summary": "Summary",
        "content": "Content",
        "key_points": [],
        "unexpected": true
    }));
    assert!(malformed.is_err());

    let missing = serde_json::from_value::<GeneratedInsight>(serde_json::json!({
        "summary": "Summary",
        "content": "Content",
        "key_points": []
    }));
    assert!(missing.is_err());
}

#[test]
fn insight_adapter_requires_strict_validated_json_and_typed_input() -> TestResult {
    let request = InsightGenerationRequest {
        topic: worker_adapters::InsightTopicContext {
            name: "Rust systems".to_owned(),
            description: Some("Reliable systems programming".to_owned()),
        },
        articles: vec![worker_adapters::InsightArticleContext {
            id: Uuid::new_v4(),
            title: Some("Typed boundaries".to_owned()),
            url: "https://example.test/typed-boundaries".to_owned(),
            published_at: None,
            content: "Article content".to_owned(),
        }],
    };

    let generated = decode_generated_insight(
        &serde_json::json!({
            "title": "Generated title",
            "summary": "Generated summary",
            "content": "Generated content",
            "key_points": ["First point", "Second point", "Third point"]
        })
        .to_string(),
    )?;
    assert_eq!(generated.title, "Generated title");
    assert_eq!(
        serde_json::from_str::<serde_json::Value>(&serde_json::to_string(&request)?)?,
        serde_json::to_value(&request)?
    );

    assert!(matches!(
        decode_generated_insight("TITLE: heuristic output"),
        Err(AppError::External)
    ));
    assert!(matches!(
        decode_generated_insight(
            &serde_json::json!({
                "title": "Generated title",
                "summary": "Generated summary",
                "content": "Generated content",
                "key_points": []
            })
            .to_string()
        ),
        Err(AppError::InvalidInput(_))
    ));
    Ok(())
}
