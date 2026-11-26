use std::{env, error::Error, io, sync::Arc};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{Method, Request, StatusCode, header::CONTENT_TYPE},
};
use blog_backend::{
    api::{
        article::{
            ArticleGenerationQueue, ArticleState, GenerationRequest, router as article_router,
        },
        auth::AuthState,
        source::{SourceState, router as source_router},
    },
    core::{
        article::{ArticleRepository, ArticleService, generate_slug},
        auth::{Account, AccountId, AccountRepository, AuthService},
        source::{
            ArticleLookupPort, EmbeddingPort, FetchExtractPort, ScrapedContent, SourceService,
        },
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            account::DieselAccountRepository, article::DieselArticleRepository,
            source::DieselSourceRepository, tag::DieselTagRepository,
        },
    },
    error::AppError,
};
use diesel::{Connection, PgConnection};
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
const TEST_SECRET: &str = "article-http-database-test-secret";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

struct RejectingGenerationQueue;

struct FixedEmbedding;

#[async_trait]
impl EmbeddingPort for FixedEmbedding {
    async fn generate_embedding(&self, _text: &str) -> Result<Vec<f32>, AppError> {
        Ok(vec![0.0; 1536])
    }
}

struct UnusedFetch;

#[async_trait]
impl FetchExtractPort for UnusedFetch {
    async fn fetch_extract(&self, _url: &str) -> Result<ScrapedContent, AppError> {
        Err(AppError::internal("fetch is not used by this test"))
    }
}

#[async_trait]
impl ArticleGenerationQueue for RejectingGenerationQueue {
    async fn submit(
        &self,
        _request: GenerationRequest,
    ) -> Result<String, blog_backend::error::AppError> {
        Err(blog_backend::error::AppError::external(
            "no underlying error was recorded",
        ))
    }
}

#[derive(Clone)]
struct TestState {
    article: ArticleState,
    auth: AuthState,
    source: SourceState,
}

impl FromRef<TestState> for ArticleState {
    fn from_ref(state: &TestState) -> Self {
        state.article.clone()
    }
}

impl FromRef<TestState> for AuthState {
    fn from_ref(state: &TestState) -> Self {
        state.auth.clone()
    }
}

impl FromRef<TestState> for SourceState {
    fn from_ref(state: &TestState) -> Self {
        state.source.clone()
    }
}

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the article_http_database target; start the Docker test PostgreSQL service and provide its URL",
        )
    })?;
    let mut connection = PgConnection::establish(&database_url)?;
    connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("article test migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    body: impl Into<Body>,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Value)> {
    let mut builder = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, "application/json");
    if let Some(bearer) = bearer {
        builder = builder.header("authorization", bearer);
    }
    let response = router.oneshot(builder.body(body.into())?).await?;
    let status = response.status();
    let body = response.into_body().collect().await?.to_bytes();
    Ok((status, serde_json::from_slice(&body)?))
}

#[tokio::test]
async fn article_http_routes_use_constructor_injected_postgres_services() -> TestResult {
    let pool = test_pool()?;
    let accounts = Arc::new(DieselAccountRepository::new(pool.clone()));
    let articles = Arc::new(DieselArticleRepository::new(pool.clone()));
    let article_repo: Arc<dyn ArticleRepository> = articles.clone();
    let article_lookup: Arc<dyn ArticleLookupPort> = articles.clone();
    let tags = Arc::new(DieselTagRepository::new(pool.clone()));
    let auth_service = Arc::new(AuthService::new(accounts.clone(), TEST_SECRET)?);
    let auth = AuthState::new(auth_service.clone());
    let article_state = ArticleState::new(
        Arc::new(ArticleService::new(article_repo, accounts.clone(), tags)),
        Arc::new(RejectingGenerationQueue),
    );
    let source_state = SourceState::new(Arc::new(SourceService::new(
        Arc::new(DieselSourceRepository::new(pool)),
        article_lookup,
        Arc::new(FixedEmbedding),
        Arc::new(UnusedFetch),
    )));
    let router: Router = article_router::<TestState>()
        .merge(source_router::<TestState>())
        .with_state(TestState {
            article: article_state,
            auth,
            source: source_state,
        })
        .into();

    let (status, missing) = call(
        router.clone(),
        Method::GET,
        "/blog/articles/parity-missing-slug",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::NOT_FOUND, "{missing}");
    assert_eq!(missing["code"], "NOT_FOUND");

    let account_id = AccountId(Uuid::new_v4());
    let password_hash = auth_service.hash_password("test-password").await?;
    let author = Account {
        id: account_id,
        name: "Article HTTP Author".to_owned(),
        email: format!("article-http-{}@example.com", account_id.0),
        password_hash: password_hash.clone(),
        role: "admin".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: None,
    };
    accounts.create(&author).await?;
    let bearer = format!("Bearer {}", auth_service.issue_token(&author)?);
    let title = format!("Article HTTP {}", Uuid::new_v4());

    let generation_title = format!("Rejected Generation {}", Uuid::new_v4());
    let (status, rejected_generation) = call(
        router.clone(),
        Method::POST,
        "/blog/generate",
        json!({"title": generation_title, "prompt": ""}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(
        status,
        StatusCode::INTERNAL_SERVER_ERROR,
        "{rejected_generation}"
    );
    assert!(
        articles
            .find_by_slug(&generate_slug(&generation_title))
            .await
            .is_err(),
        "a rejected generation request must not leave an orphan draft"
    );

    let (status, unauthorized) = call(
        router.clone(),
        Method::POST,
        "/blog/articles",
        "{malformed",
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized}");
    assert_eq!(unauthorized["code"], "UNAUTHORIZED");
    let (status, invalid) = call(
        router.clone(),
        Method::POST,
        "/blog/articles",
        "{malformed",
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{invalid}");
    assert_eq!(invalid["code"], "INVALID_INPUT");
    assert!(invalid["error"].is_string());
    assert!(invalid.get("details").is_none());

    let (status, created) = call(
        router.clone(),
        Method::POST,
        "/blog/articles",
        json!({
            "title": title,
            "content": "A complete database-backed article body.",
            "image_url": "https://example.com/article.png",
            "tags": ["http-test", "postgres"],
            "publish": false,
            "authorId": account_id.0,
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{created}");
    let article_id = Uuid::parse_str(
        created["data"]["article"]["id"]
            .as_str()
            .ok_or_else(|| io::Error::other("created article id missing"))?,
    )?;
    let slug = created["data"]["article"]["slug"]
        .as_str()
        .ok_or_else(|| io::Error::other("created article slug missing"))?
        .to_owned();
    assert_eq!(created["data"]["author"]["name"], "Article HTTP Author");
    assert_eq!(created["data"]["tags"].as_array().map(Vec::len), Some(2));

    let spoofed_author = Uuid::new_v4();
    let spoofed_title = format!("Spoofed author {}", Uuid::new_v4());
    let (status, spoofed) = call(
        router.clone(),
        Method::POST,
        "/blog/articles",
        json!({
            "title": spoofed_title,
            "content": "Body written by the authenticated account.",
            "publish": false,
            "authorId": spoofed_author,
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{spoofed}");
    assert_ne!(spoofed_author, account_id.0);
    assert_eq!(
        spoofed["data"]["article"]["author_id"],
        account_id.0.to_string()
    );
    assert_eq!(spoofed["data"]["author"]["id"], account_id.0.to_string());
    assert!(spoofed["data"]["article"]["published_at"].is_null());

    let (status, unauthorized_draft) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": "Unsigned draft"}).to_string(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized_draft}");
    assert_eq!(unauthorized_draft["code"], "UNAUTHORIZED");

    let (status, missing_title) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": "  "}).to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::BAD_REQUEST, "{missing_title}");
    assert_eq!(missing_title["code"], "INVALID_INPUT");

    let draft_title = format!("Draft HTTP {}", Uuid::new_v4());
    let (status, draft) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({
            "title": draft_title,
            "markdown": "A draft written as markdown.",
            "tags": ["notes"],
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{draft}");
    assert_eq!(draft["data"]["article"]["draft_title"], draft_title);
    assert_eq!(
        draft["data"]["article"]["draft_content"],
        "A draft written as markdown."
    );
    assert_eq!(
        draft["data"]["article"]["author_id"],
        account_id.0.to_string()
    );
    assert_eq!(draft["data"]["author"]["id"], account_id.0.to_string());
    assert!(draft["data"]["article"]["published_at"].is_null());
    assert_eq!(draft["data"]["tags"].as_array().map(Vec::len), Some(1));

    let sources_path = format!("/sources/article/{article_id}");
    let (status, unauthorized_sources) = call(
        router.clone(),
        Method::PUT,
        &sources_path,
        json!({"sources": []}).to_string(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized_sources}");
    assert_eq!(unauthorized_sources["code"], "UNAUTHORIZED");

    let (status, inserted_sources) = call(
        router.clone(),
        Method::PUT,
        &sources_path,
        json!({
            "sources": [
                {
                    "title": "One",
                    "content": "First source body",
                    "url": "https://Example.com/one",
                    "source_type": "web"
                },
                {
                    "title": "Two",
                    "content": "Second source body",
                    "url": "https://example.com/two",
                    "source_type": "web"
                }
            ]
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{inserted_sources}");
    let inserted = inserted_sources["data"]["sources"]
        .as_array()
        .ok_or_else(|| io::Error::other("inserted sources missing"))?;
    assert_eq!(inserted.len(), 2);
    let first_source_id = inserted[0]["id"]
        .as_str()
        .ok_or_else(|| io::Error::other("inserted source id missing"))?
        .to_owned();

    let (status, updated_sources) = call(
        router.clone(),
        Method::PUT,
        &sources_path,
        json!({
            "sources": [
                {
                    "title": "One revised",
                    "content": "Revised source body",
                    "url": "https://example.com/ONE"
                },
                {
                    "title": "Three",
                    "content": "Third source body",
                    "url": "https://example.com/three"
                }
            ]
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated_sources}");
    let updated = updated_sources["data"]["sources"]
        .as_array()
        .ok_or_else(|| io::Error::other("updated sources missing"))?;
    assert_eq!(updated.len(), 2);
    assert_eq!(updated[0]["id"], first_source_id);
    assert_eq!(updated[0]["title"], "One revised");
    assert_eq!(updated[0]["content"], "Revised source body");
    assert_eq!(updated[0]["url"], "https://example.com/ONE");
    assert_ne!(updated[1]["id"], first_source_id);

    let (status, listed_sources) = call(
        router.clone(),
        Method::GET,
        &sources_path,
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed_sources}");
    let listed = listed_sources["data"]["sources"]
        .as_array()
        .ok_or_else(|| io::Error::other("listed sources missing"))?;
    assert_eq!(listed.len(), 3);
    let mut url_keys = Vec::new();
    for source in listed {
        let url = source["url"]
            .as_str()
            .ok_or_else(|| io::Error::other("listed source url missing"))?
            .to_ascii_lowercase();
        assert!(!url_keys.contains(&url), "{listed_sources}");
        url_keys.push(url);
    }

    let article_path = format!("/blog/articles/{slug}");
    let (status, hidden) = call(
        router.clone(),
        Method::GET,
        &article_path,
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::NOT_FOUND, "{hidden}");
    let (status, fetched) = call(
        router.clone(),
        Method::GET,
        &article_path,
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{fetched}");
    assert_eq!(fetched["data"]["article"]["id"], article_id.to_string());

    let unique_title = title.rsplit(' ').next().unwrap_or(&title);
    let search_path = format!("/blog/articles/search?query={unique_title}&status=all");
    let (status, anonymous_search) = call(
        router.clone(),
        Method::GET,
        &search_path,
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{anonymous_search}");
    assert_eq!(anonymous_search["data"]["include_drafts"], false);
    assert_eq!(anonymous_search["data"]["articles"], json!([]));
    let (status, author_search) = call(
        router.clone(),
        Method::GET,
        &search_path,
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{author_search}");
    assert_eq!(author_search["data"]["include_drafts"], true);
    assert_eq!(
        author_search["data"]["articles"][0]["article"]["id"],
        article_id.to_string()
    );
    let (status, anonymous_list) = call(
        router.clone(),
        Method::GET,
        "/blog/articles?status=all&articlesPerPage=50",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{anonymous_list}");
    assert_eq!(anonymous_list["data"]["include_drafts"], false);
    assert!(
        anonymous_list["data"]["articles"]
            .as_array()
            .is_some_and(|articles| articles
                .iter()
                .all(|item| !item["article"]["published_at"].is_null())),
        "anonymous listing exposed a draft: {anonymous_list}"
    );

    let (status, unauthorized) = call(
        router.clone(),
        Method::DELETE,
        &format!("/blog/articles/{article_id}"),
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{unauthorized}");

    let (status, updated) = call(
        router.clone(),
        Method::POST,
        &format!("/blog/articles/{slug}/update"),
        json!({
            "title": "Updated HTTP Article",
            "content": "The updated database-backed article body.",
            "image_url": "",
            "tags": ["http-test"],
            "published_at": null,
        })
        .to_string(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{updated}");
    let updated_slug = updated["data"]["article"]["slug"]
        .as_str()
        .ok_or_else(|| io::Error::other("updated slug missing"))?;
    assert_eq!(updated_slug, "updated-http-article");

    let (status, published) = call(
        router.clone(),
        Method::POST,
        &format!("/blog/articles/{updated_slug}/publish"),
        "{malformed optional body",
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{published}");
    assert!(!published["data"]["article"]["published_at"].is_null());

    articles.drain_background_tasks().await?;
    for extra in [
        spoofed["data"]["article"]["id"].as_str(),
        draft["data"]["article"]["id"].as_str(),
    ] {
        let id = extra.ok_or_else(|| io::Error::other("extra article id missing"))?;
        let (status, removed) = call(
            router.clone(),
            Method::DELETE,
            &format!("/blog/articles/{id}"),
            Body::empty(),
            Some(&bearer),
        )
        .await?;
        assert_eq!(status, StatusCode::OK, "{removed}");
    }
    let (status, deleted) = call(
        router,
        Method::DELETE,
        &format!("/blog/articles/{article_id}"),
        Body::empty(),
        Some(&bearer),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{deleted}");
    assert_eq!(deleted, json!({"data": {"success": true}}));

    assert!(
        accounts
            .delete_if_password_hash(account_id, &password_hash)
            .await?
    );
    Ok(())
}
