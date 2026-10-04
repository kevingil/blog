use std::{
    error::Error,
    fmt::Debug,
    sync::{Arc, Mutex, MutexGuard},
};

use async_trait::async_trait;
use blog_backend::{
    core::{
        datasource::{CrawledContent, DataSource, DataSourceRepository},
        insight::{
            ContentTopicMatch, ContentTopicMatchRepository, InsightContentRepository, InsightTopic,
            InsightTopicRepository,
        },
        ml::llm::{AnswerResponse, ResearchPort, WebSearchResponse, WebSearchResult},
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
            return Err(AppError::database("no underlying error was recorded"));
        }
        Ok(lock(&self.topics).clone())
    }

    async fn find_due(&self, limit: i64) -> Result<Vec<InsightTopic>, AppError> {
        let values = self.find_all().await?;
        if limit < 0 {
            return Ok(values);
        }
        Ok(values
            .into_iter()
            .take(usize::try_from(limit).unwrap_or(0))
            .collect())
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
            return Err(AppError::database("no underlying error was recorded"));
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
        .map_err(AppError::internal)
    }

    async fn find_primary_by_topic_id(
        &self,
        topic_id: Uuid,
        offset: i64,
        limit: i64,
    ) -> Result<(Vec<ContentTopicMatch>, i64), AppError> {
        if *lock(&self.fail_matches) {
            return Err(AppError::database("no underlying error was recorded"));
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
    async fn save(&self, content: &mut CrawledContent) -> Result<(), AppError> {
        if *lock(&self.fail_contents) {
            return Err(AppError::database("no underlying error was recorded"));
        }
        if content.id.is_nil() {
            content.id = Uuid::new_v4();
        }
        lock(&self.contents).push(content.clone());
        Ok(())
    }

    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<CrawledContent>, AppError> {
        if *lock(&self.fail_contents) {
            return Err(AppError::database("no underlying error was recorded"));
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
            return Err(AppError::external("no underlying error was recorded"));
        }
        lock(&self.response)
            .clone()
            .ok_or(AppError::external("no underlying error was recorded"))
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
            return Err(AppError::database("no underlying error was recorded"));
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

#[derive(Default)]
struct Research {
    pages: Mutex<Option<Vec<WebSearchResult>>>,
    fail: Mutex<bool>,
}

impl Research {
    fn set_pages(&self, contents: &[CrawledContent]) {
        *lock(&self.pages) = Some(
            contents
                .iter()
                .map(|content| WebSearchResult {
                    id: content.id.to_string(),
                    title: content.title.clone().unwrap_or_default(),
                    url: content.url.clone(),
                    text: content.content.clone(),
                    summary: String::new(),
                    author: String::new(),
                    published_date: content
                        .published_at
                        .map(|value| value.to_rfc3339())
                        .unwrap_or_default(),
                    highlights: Vec::new(),
                    score: 1.0,
                    favicon: String::new(),
                })
                .collect(),
        );
    }
}

#[async_trait]
impl ResearchPort for Research {
    fn is_configured(&self) -> bool {
        true
    }

    async fn search(&self, query: &str) -> Result<WebSearchResponse, AppError> {
        self.deep_search(query, None).await
    }

    async fn answer(&self, _question: &str) -> Result<AnswerResponse, AppError> {
        Err(AppError::external("no underlying error was recorded"))
    }

    async fn deep_search(
        &self,
        query: &str,
        _domain: Option<&str>,
    ) -> Result<WebSearchResponse, AppError> {
        if *lock(&self.fail) {
            return Err(AppError::external("no underlying error was recorded"));
        }
        let results = lock(&self.pages).clone().unwrap_or_else(|| {
            vec![WebSearchResult {
                id: "page".to_owned(),
                title: query.to_owned(),
                url: "https://example.test/page".to_owned(),
                text: "page body".to_owned(),
                summary: String::new(),
                author: String::new(),
                published_date: String::new(),
                highlights: Vec::new(),
                score: 1.0,
                favicon: String::new(),
            }]
        });
        Ok(WebSearchResponse {
            results,
            request_id: "research".to_owned(),
            resolved_search_type: "deep".to_owned(),
            cost_dollars: None,
        })
    }
}

struct EmptySources;

#[async_trait]
impl DataSourceRepository for EmptySources {
    async fn find_by_id(&self, _id: Uuid) -> Result<DataSource, AppError> {
        Err(AppError::NotFound)
    }
    async fn find_by_organization_id(&self, _id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(Vec::new())
    }
    async fn find_by_user_id(&self, _id: Uuid) -> Result<Vec<DataSource>, AppError> {
        Ok(Vec::new())
    }
    async fn find_by_url(&self, _url: &str) -> Result<Option<DataSource>, AppError> {
        Ok(None)
    }
    async fn find_due_to_crawl(&self, _limit: i64) -> Result<Vec<DataSource>, AppError> {
        Ok(Vec::new())
    }
    async fn list(&self, _offset: i64, _limit: i64) -> Result<(Vec<DataSource>, i64), AppError> {
        Ok((Vec::new(), 0))
    }
    async fn save(&self, _source: &mut DataSource) -> Result<(), AppError> {
        Ok(())
    }
    async fn update(&self, _source: &DataSource) -> Result<(), AppError> {
        Ok(())
    }
    async fn update_crawl_status(
        &self,
        _id: Uuid,
        _status: &str,
        _error_message: Option<&str>,
    ) -> Result<(), AppError> {
        Ok(())
    }
    async fn update_next_crawl_at(
        &self,
        _id: Uuid,
        _next_crawl_at: DateTime<Utc>,
    ) -> Result<(), AppError> {
        Ok(())
    }
    async fn increment_content_count(&self, _id: Uuid, _delta: i32) -> Result<(), AppError> {
        Ok(())
    }
    async fn delete(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }
}

struct Fixture {
    generator: RuntimeInsightGenerator,
    store: Arc<Store>,
    research: Arc<Research>,
    text: Arc<Text>,
    writer: Arc<Writer>,
}

fn fixture(now: DateTime<Utc>) -> Fixture {
    let store = Arc::new(Store::default());
    let text = Arc::new(Text::successful());
    let writer = Arc::new(Writer::default());
    let research = Arc::new(Research::default());
    let generator = RuntimeInsightGenerator::new(
        store.clone(),
        Arc::new(EmptySources),
        store.clone(),
        research.clone(),
        text.clone(),
        writer.clone(),
        Arc::new(FixedClock(now)),
    );
    Fixture {
        generator,
        store,
        research,
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
        check_frequency: "daily".to_owned(),
        next_check_at: None,
        is_enabled: true,
    }
}

fn content(index: usize, published_at: Option<DateTime<Utc>>, body: String) -> CrawledContent {
    CrawledContent {
        id: Uuid::new_v4(),
        data_source_id: Some(Uuid::new_v4()),
        topic_id: None,
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

fn seed(fixture: &Fixture, topic: &InsightTopic, contents: Vec<CrawledContent>, _total: i64) {
    lock(&fixture.store.topics).push(topic.clone());
    fixture.research.set_pages(&contents);
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
async fn empty_research_skips_before_the_model() -> TestResult {
    let fixture = fixture(now());
    let topic = topic(None);
    lock(&fixture.store.topics).push(topic.clone());
    *lock(&fixture.research.pages) = Some(Vec::new());
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
async fn cadence_is_not_a_hard_coded_day_window() -> TestResult {
    let fixture = fixture(now());
    let recent_topic = topic(Some(now() - Duration::hours(1)));
    seed(
        &fixture,
        &recent_topic,
        (1..=3)
            .map(|index| content(index, None, format!("body {index}")))
            .collect(),
        3,
    );
    assert_eq!(
        fixture
            .generator
            .generate_for_topic(&recent_topic, &CancellationToken::new())
            .await?,
        InsightTopicResult::Created
    );
    assert_eq!(lock(&fixture.writer.values).len(), 1);
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
    assert_eq!(values[0].source_content_ids.len(), ids.len());
    assert_eq!(values[0].period_start, earliest);
    assert_eq!(values[0].period_end, latest);
    assert_eq!(values[0].organization_id, topic.organization_id);
    assert_eq!(values[0].topic_id, Some(topic.id));
    drop(values);
    assert_eq!(lock(&fixture.store.updated_topics).as_slice(), &[(
        topic.id,
        now()
    )]);
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

    for failure in ["research", "save", "text", "writer", "topic-update"] {
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
            "research" => *lock(&fixture.research.fail) = true,
            "save" => *lock(&fixture.store.fail_contents) = true,
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
            "research" => "failed to research tracker",
            "save" => "failed to save research page",
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
            previous_summary: None,
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
        Err(AppError::External(_))
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
