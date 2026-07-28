use std::{error::Error, sync::Arc};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::{
        auth::AuthState,
        datasource::{self, DataSourceState},
        insight::{self, InsightState},
        source::{self, SourceState},
    },
    core::{
        auth::{Account, AccountId, AccountRepository, AuthService},
        datasource::{DataSourceService, RecommendationService},
        insight::InsightService,
        source::SourceService,
    },
    database::{
        pool::create_pool,
        repository::{
            article::DieselArticleRepository,
            content_topic_match::DieselContentTopicMatchRepository,
            crawled_content::DieselCrawledContentRepository,
            data_source::DieselDataSourceRepository, insight::DieselInsightRepository,
            insight_topic::DieselInsightTopicRepository, source::DieselSourceRepository,
            user_insight_status::DieselUserInsightStatusRepository,
        },
    },
    error::AppError,
    integrations::{exa::ExaClient, fetch::HttpFetchExtract, openai::OpenAiClient},
};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use tower::ServiceExt;
use utoipa_axum::router::OpenApiRouter;
use uuid::Uuid;

const TEST_SECRET: &str = "data-domain-http-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Clone)]
struct TestState {
    auth: AuthState,
    datasource: DataSourceState,
    insight: InsightState,
    source: SourceState,
}

impl FromRef<TestState> for AuthState {
    fn from_ref(state: &TestState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<TestState> for DataSourceState {
    fn from_ref(state: &TestState) -> Self {
        state.datasource.clone()
    }
}

impl FromRef<TestState> for InsightState {
    fn from_ref(state: &TestState) -> Self {
        state.insight.clone()
    }
}

impl FromRef<TestState> for SourceState {
    fn from_ref(state: &TestState) -> Self {
        state.source.clone()
    }
}

#[derive(Default)]
struct Accounts;

#[async_trait]
impl AccountRepository for Accounts {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(Some(Account {
            id,
            name: "Test Account".to_owned(),
            email: "test@example.test".to_owned(),
            password_hash: String::new(),
            role: "admin".to_owned(),
            created_at: None,
            updated_at: None,
            bio: None,
            profile_image: None,
            email_public: None,
            social_links: None,
            meta_description: None,
            organization_id: None,
        }))
    }

    async fn find_by_email(&self, _email: &str) -> Result<Option<Account>, AppError> {
        Ok(None)
    }

    async fn create(&self, _account: &Account) -> Result<(), AppError> {
        Ok(())
    }

    async fn update_identity(
        &self,
        _id: AccountId,
        _name: &str,
        _email: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn update_password_if_current(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
        _new_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }

    async fn delete_if_password_hash(
        &self,
        _id: AccountId,
        _expected_password_hash: &str,
    ) -> Result<bool, AppError> {
        Ok(false)
    }
}

struct Fixture {
    router: Router,
    bearer: String,
}

fn fixture() -> TestResult<Fixture> {
    // Pool creation is lazy. Rejection-path HTTP tests exercise routing,
    // authentication, and validation without opening a database connection.
    let pool = create_pool(&SecretString::from(
        "postgres://unused:unused@127.0.0.1:1/unused".to_owned(),
    ))?;
    let accounts = Arc::new(Accounts);
    let auth_service = Arc::new(AuthService::new(accounts.clone(), TEST_SECRET)?);
    let bearer = format!(
        "Bearer {}",
        auth_service.issue_token(AccountId(Uuid::new_v4()))?
    );

    let data_sources = Arc::new(DieselDataSourceRepository::new(pool.clone()));
    let crawled_content = Arc::new(DieselCrawledContentRepository::new(pool.clone()));
    let datasource = DataSourceState::new(
        Arc::new(DataSourceService::new(
            data_sources.clone(),
            crawled_content.clone(),
        )),
        Arc::new(RecommendationService::new(
            data_sources,
            Arc::new(ExaClient::new("")?),
        )),
        accounts.clone(),
    );

    let embeddings = Arc::new(OpenAiClient::new("")?);
    let insight = InsightState::new(
        Arc::new(InsightService::new(
            Arc::new(DieselInsightRepository::new(pool.clone())),
            Arc::new(DieselInsightTopicRepository::new(pool.clone())),
            Arc::new(DieselUserInsightStatusRepository::new(pool.clone())),
            crawled_content,
            Arc::new(DieselContentTopicMatchRepository::new(pool.clone())),
            embeddings.clone(),
        )),
        accounts,
    );

    let source = SourceState::new(Arc::new(SourceService::new(
        Arc::new(DieselSourceRepository::new(pool.clone())),
        Arc::new(DieselArticleRepository::new(pool)),
        embeddings,
        Arc::new(HttpFetchExtract::new()?),
    )));

    let state = TestState {
        auth: AuthState::new(auth_service),
        datasource,
        insight,
        source,
    };
    let router: Router = OpenApiRouter::new()
        .merge(datasource::router::<TestState>())
        .merge(insight::router::<TestState>())
        .merge(source::router::<TestState>())
        .with_state(state)
        .into();
    Ok(Fixture { router, bearer })
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    body: Value,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut request = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, "application/json");
    if let Some(bearer) = bearer {
        request = request.header("authorization", bearer);
    }
    let response = router
        .oneshot(request.body(Body::from(serde_json::to_vec(&body)?))?)
        .await?;
    let status = response.status();
    let bytes = response.into_body().collect().await?.to_bytes();
    let body = serde_json::from_slice(&bytes)
        .unwrap_or_else(|_| json!({"_raw": String::from_utf8_lossy(&bytes)}));
    Ok((status, body))
}

#[tokio::test]
async fn every_data_source_insight_and_source_route_requires_authentication() -> TestResult {
    let fixture = fixture()?;
    let id = Uuid::new_v4();
    let routes = [
        (Method::GET, "/data-sources".to_owned()),
        (Method::POST, "/data-sources".to_owned()),
        (Method::POST, "/data-sources/recommendations".to_owned()),
        (
            Method::POST,
            "/data-sources/recommendations/discovery".to_owned(),
        ),
        (Method::GET, format!("/data-sources/{id}")),
        (Method::PUT, format!("/data-sources/{id}")),
        (Method::DELETE, format!("/data-sources/{id}")),
        (Method::POST, format!("/data-sources/{id}/crawl")),
        (Method::GET, format!("/data-sources/{id}/content")),
        (Method::GET, "/insights".to_owned()),
        (Method::GET, "/insights/search?q=test".to_owned()),
        (Method::GET, "/insights/unread-count".to_owned()),
        (Method::GET, "/insights/topics".to_owned()),
        (Method::POST, "/insights/topics".to_owned()),
        (Method::GET, format!("/insights/topics/{id}")),
        (Method::PUT, format!("/insights/topics/{id}")),
        (Method::DELETE, format!("/insights/topics/{id}")),
        (Method::GET, "/insights/content/search?q=test".to_owned()),
        (Method::GET, "/insights/content/recent".to_owned()),
        (Method::GET, format!("/insights/{id}")),
        (Method::DELETE, format!("/insights/{id}")),
        (Method::POST, format!("/insights/{id}/read")),
        (Method::POST, format!("/insights/{id}/pin")),
        (Method::GET, "/dashboard/sources".to_owned()),
        (Method::POST, "/sources".to_owned()),
        (Method::POST, "/sources/".to_owned()),
        (Method::POST, "/sources/scrape".to_owned()),
        (Method::GET, format!("/sources/article/{id}")),
        (Method::GET, format!("/sources/article/{id}/search?q=test")),
        (Method::GET, format!("/sources/{id}")),
        (Method::PUT, format!("/sources/{id}")),
        (Method::DELETE, format!("/sources/{id}")),
    ];
    for (method, path) in routes {
        let (status, response) =
            call(fixture.router.clone(), method, &path, json!({}), None).await?;
        assert_eq!(status, StatusCode::UNAUTHORIZED, "{path}: {response}");
    }
    Ok(())
}

#[tokio::test]
async fn uuid_path_rejections_preserve_domain_specific_messages() -> TestResult {
    let fixture = fixture()?;
    for (method, path, message) in [
        (
            Method::GET,
            "/data-sources/not-a-uuid",
            "Invalid data source ID",
        ),
        (
            Method::GET,
            "/insights/topics/not-a-uuid",
            "Invalid topic ID",
        ),
        (Method::GET, "/insights/not-a-uuid", "Invalid insight ID"),
        (
            Method::GET,
            "/sources/article/not-a-uuid",
            "Invalid article ID",
        ),
        (Method::GET, "/sources/not-a-uuid", "Invalid source ID"),
    ] {
        let (status, response) = call(
            fixture.router.clone(),
            method,
            path,
            json!(null),
            Some(&fixture.bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::BAD_REQUEST, "{path}: {response}");
        assert_eq!(response["code"], "INVALID_INPUT", "{path}: {response}");
        assert_eq!(response["error"], message, "{path}: {response}");
    }
    Ok(())
}

#[tokio::test]
async fn data_source_requests_preserve_body_and_validation_errors() -> TestResult {
    let fixture = fixture()?;
    let (status, response) = call(
        fixture.router.clone(),
        Method::POST,
        "/data-sources",
        json!({"name": "", "url": "not-a-url"}),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{response}");
    assert_eq!(response["code"], "VALIDATION_ERROR");

    let (status, response) = call(
        fixture.router.clone(),
        Method::POST,
        "/data-sources",
        json!({"name": 42, "url": "https://example.test"}),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{response}");
    assert_eq!(response["code"], "INVALID_INPUT");
    assert_eq!(response["error"], "Invalid request body");

    let (status, response) = call(
        fixture.router,
        Method::POST,
        "/data-sources/recommendations",
        json!({"query": "ab"}),
        Some(&fixture.bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{response}");
    assert_eq!(response["code"], "VALIDATION_ERROR");
    Ok(())
}

#[tokio::test]
async fn insight_and_source_rejections_match_the_go_contract() -> TestResult {
    let fixture = fixture()?;
    for (method, path, body, code, message) in [
        (
            Method::GET,
            "/insights?topic_id=not-a-uuid",
            json!(null),
            "INVALID_INPUT",
            "Invalid topic ID",
        ),
        (
            Method::GET,
            "/insights/search?limit=not-an-integer",
            json!(null),
            "INVALID_INPUT",
            "Search query required",
        ),
        (
            Method::POST,
            "/insights/topics",
            json!({"name": ""}),
            "VALIDATION_ERROR",
            "name: failed validation",
        ),
        (
            Method::POST,
            "/sources",
            json!({"article_id": Uuid::nil(), "content": ""}),
            "VALIDATION_ERROR",
            "article_id: failed validation",
        ),
        (
            Method::POST,
            "/sources/scrape",
            json!({"article_id": Uuid::new_v4(), "url": "not-a-url"}),
            "VALIDATION_ERROR",
            "url: failed validation",
        ),
        (
            Method::GET,
            &format!(
                "/sources/article/{}/search?limit=not-an-integer",
                Uuid::new_v4()
            ),
            json!(null),
            "INVALID_INPUT",
            "Query parameter 'q' is required",
        ),
    ] {
        let (status, response) = call(
            fixture.router.clone(),
            method,
            path,
            body,
            Some(&fixture.bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::BAD_REQUEST, "{path}: {response}");
        assert_eq!(response["code"], code, "{path}: {response}");
        assert_eq!(response["error"], message, "{path}: {response}");
    }
    Ok(())
}

#[test]
fn generated_openapi_contains_every_data_domain_operation() -> TestResult {
    let (_, document) = OpenApiRouter::<TestState>::new()
        .merge(datasource::router::<TestState>())
        .merge(insight::router::<TestState>())
        .merge(source::router::<TestState>())
        .split_for_parts();
    let document = serde_json::to_value(document)?;
    let operation_ids = document["paths"]
        .as_object()
        .ok_or("missing OpenAPI paths")?
        .values()
        .flat_map(|path| {
            path.as_object()
                .into_iter()
                .flat_map(|operations| operations.values())
        })
        .filter_map(|operation| operation["operationId"].as_str())
        .collect::<Vec<_>>();
    for operation_id in [
        "listDataSources",
        "createDataSource",
        "recommendDataSources",
        "discoverDataSources",
        "getDataSource",
        "updateDataSource",
        "deleteDataSource",
        "triggerDataSourceCrawl",
        "getDataSourceContent",
        "listInsights",
        "searchInsights",
        "getUnreadInsightCount",
        "listInsightTopics",
        "createInsightTopic",
        "getInsightTopic",
        "updateInsightTopic",
        "deleteInsightTopic",
        "searchInsightContent",
        "getRecentInsightContent",
        "getInsight",
        "deleteInsight",
        "markInsightRead",
        "toggleInsightPinned",
        "listAllSources",
        "createSource",
        "scrapeAndCreateSource",
        "getArticleSources",
        "searchSimilarSources",
        "getSource",
        "updateSource",
        "deleteSource",
    ] {
        assert!(
            operation_ids.contains(&operation_id),
            "missing OpenAPI operation {operation_id}"
        );
    }
    assert_eq!(operation_ids.len(), 31);
    assert_eq!(
        document["components"]["schemas"]["DataSourceCreateRequest"]["properties"]["name"]["minLength"],
        1
    );
    assert_eq!(
        document["components"]["schemas"]["DataSourceRecommendationRequest"]["properties"]["query"]
            ["maxLength"],
        500
    );
    assert_eq!(
        document["components"]["schemas"]["InsightTopicCreateRequest"]["properties"]["color"]["maxLength"],
        20
    );
    assert_eq!(
        document["components"]["schemas"]["CreateSourceRequest"]["properties"]["content"]["minLength"],
        1
    );
    Ok(())
}
