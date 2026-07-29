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
    },
    core::{
        article::{ArticleRepository, ArticleService, generate_slug},
        auth::{Account, AccountId, AccountRepository, AuthService},
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            account::DieselAccountRepository, article::DieselArticleRepository,
            tag::DieselTagRepository,
        },
    },
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

#[async_trait]
impl ArticleGenerationQueue for RejectingGenerationQueue {
    async fn submit(
        &self,
        _request: GenerationRequest,
    ) -> Result<String, blog_backend::error::AppError> {
        Err(blog_backend::error::AppError::External)
    }
}

#[derive(Clone)]
struct TestState {
    article: ArticleState,
    auth: AuthState,
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
    let tags = Arc::new(DieselTagRepository::new(pool));
    let auth_service = Arc::new(AuthService::new(accounts.clone(), TEST_SECRET)?);
    let auth = AuthState::new(auth_service.clone());
    let article_state = ArticleState::new(
        Arc::new(ArticleService::new(
            articles.clone(),
            accounts.clone(),
            tags,
        )),
        Arc::new(RejectingGenerationQueue),
    );
    let router: Router = article_router::<TestState>()
        .with_state(TestState {
            article: article_state,
            auth,
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
    accounts
        .create(&Account {
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
        })
        .await?;
    let bearer = format!("Bearer {}", auth_service.issue_token(account_id)?);
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

    let (status, fetched) = call(
        router.clone(),
        Method::GET,
        &format!("/blog/articles/{slug}"),
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{fetched}");
    assert_eq!(fetched["data"]["article"]["id"], article_id.to_string());

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
