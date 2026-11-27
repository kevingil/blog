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
        auth::{AuthState, router as auth_router},
    },
    core::{
        article::ArticleService,
        auth::{Account, AccountId, AccountRepository, AuthService, mcp_token_hash},
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            account::DieselAccountRepository, article::DieselArticleRepository,
            tag::DieselTagRepository,
        },
    },
    schema::mcp_access_token,
};
use chrono::{DateTime, Utc};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
const TEST_SECRET: &str = "mcp-access-token-test-secret";
const PASSWORD: &str = "test-password";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

struct RejectingGenerationQueue;

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
            "TEST_DATABASE_URL is required for the mcp_access_token target; start the Docker test PostgreSQL service and provide its URL",
        )
    })?;
    let mut connection = PgConnection::establish(&database_url)?;
    connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("mcp token migration failed: {error}")))?;
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
    let bytes = response.into_body().collect().await?.to_bytes();
    if bytes.is_empty() {
        return Ok((status, Value::Null));
    }
    Ok((status, serde_json::from_slice(&bytes)?))
}

async fn stored_token(
    pool: &PgPool,
    id: Uuid,
) -> TestResult<(String, String, Option<DateTime<Utc>>, Option<DateTime<Utc>>)> {
    let mut connection = pool.get().await.map_err(io::Error::other)?;
    Ok(mcp_access_token::table
        .filter(mcp_access_token::id.eq(id))
        .select((
            mcp_access_token::token_hash,
            mcp_access_token::token_prefix,
            mcp_access_token::last_used_at,
            mcp_access_token::revoked_at,
        ))
        .first(&mut connection)
        .await?)
}

fn bearer(token: &str) -> String {
    format!("Bearer {token}")
}

#[tokio::test]
async fn mcp_access_tokens_authenticate_until_revoked_and_survive_password_changes() -> TestResult {
    let pool = test_pool()?;
    let accounts = Arc::new(DieselAccountRepository::new(pool.clone()));
    let articles = Arc::new(DieselArticleRepository::new(pool.clone()));
    let tags = Arc::new(DieselTagRepository::new(pool.clone()));
    let auth_service = Arc::new(AuthService::new(accounts.clone(), TEST_SECRET)?);
    let state = TestState {
        article: ArticleState::new(
            Arc::new(ArticleService::new(articles, accounts.clone(), tags)),
            Arc::new(RejectingGenerationQueue),
        ),
        auth: AuthState::new(auth_service.clone()),
    };
    let router: Router = auth_router::<TestState>()
        .merge(article_router::<TestState>())
        .with_state(state)
        .into();

    let admin_id = AccountId(Uuid::new_v4());
    let admin = Account {
        id: admin_id,
        name: "MCP Token Admin".to_owned(),
        email: format!("mcp-token-{}@example.com", admin_id.0),
        password_hash: auth_service.hash_password(PASSWORD).await?,
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
    accounts.create(&admin).await?;
    let admin_jwt = bearer(&auth_service.issue_token(&admin)?);

    let member_id = AccountId(Uuid::new_v4());
    let member = Account {
        id: member_id,
        name: "MCP Token Member".to_owned(),
        email: format!("mcp-member-{}@example.com", member_id.0),
        password_hash: auth_service.hash_password(PASSWORD).await?,
        role: "user".to_owned(),
        created_at: None,
        updated_at: None,
        bio: None,
        profile_image: None,
        email_public: None,
        social_links: None,
        meta_description: None,
        organization_id: None,
    };
    accounts.create(&member).await?;
    let member_jwt = bearer(&auth_service.issue_token(&member)?);

    let (status, missing) = call(
        router.clone(),
        Method::POST,
        "/auth/mcp-tokens",
        json!({"name": "cursor"}).to_string(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{missing}");
    assert_eq!(missing["code"], "UNAUTHORIZED");

    let (status, listed_missing) = call(
        router.clone(),
        Method::GET,
        "/auth/mcp-tokens",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{listed_missing}");

    let (status, revoked_missing) = call(
        router.clone(),
        Method::DELETE,
        &format!("/auth/mcp-tokens/{}", Uuid::new_v4()),
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{revoked_missing}");

    let (status, forbidden) = call(
        router.clone(),
        Method::POST,
        "/auth/mcp-tokens",
        json!({"name": "cursor"}).to_string(),
        Some(&member_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::FORBIDDEN, "{forbidden}");
    assert_eq!(forbidden["code"], "FORBIDDEN");

    let (status, created) = call(
        router.clone(),
        Method::POST,
        "/auth/mcp-tokens",
        json!({"name": "cursor"}).to_string(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{created}");
    let plaintext = created["data"]["token"]
        .as_str()
        .ok_or_else(|| io::Error::other("create response omitted the plaintext token"))?
        .to_owned();
    assert!(plaintext.starts_with("mcp_"));
    assert_eq!(plaintext.len(), 68);
    assert_eq!(created.to_string().matches(&plaintext).count(), 1);
    let token_id = Uuid::parse_str(
        created["data"]["id"]
            .as_str()
            .ok_or_else(|| io::Error::other("create response omitted id"))?,
    )?;
    let prefix = created["data"]["tokenPrefix"]
        .as_str()
        .ok_or_else(|| io::Error::other("create response omitted tokenPrefix"))?;
    assert_eq!(prefix, &plaintext[..prefix.len()]);
    assert_eq!(prefix.len(), 12);
    let hash = mcp_token_hash(&plaintext);
    assert!(!created.to_string().contains(&hash));

    let (stored_hash, stored_prefix, last_used_at, revoked_at) =
        stored_token(&pool, token_id).await?;
    assert_eq!(stored_hash, hash);
    assert_ne!(stored_hash, plaintext);
    assert!(!stored_hash.contains(&plaintext));
    assert_eq!(stored_prefix, prefix);
    assert!(last_used_at.is_none());
    assert!(revoked_at.is_none());

    let (status, listed) = call(
        router.clone(),
        Method::GET,
        "/auth/mcp-tokens",
        Body::empty(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed}");
    let listed_text = listed.to_string();
    assert!(!listed_text.contains(&plaintext));
    assert!(!listed_text.contains(&hash));
    let item = listed["data"]
        .as_array()
        .and_then(|items| {
            items
                .iter()
                .find(|item| item["id"].as_str() == Some(&token_id.to_string()))
        })
        .ok_or_else(|| io::Error::other("list omitted the created token"))?;
    assert!(item.get("token").is_none());
    assert!(item.get("tokenHash").is_none());
    assert!(item.get("token_hash").is_none());
    assert_eq!(item["tokenPrefix"], prefix);
    assert_eq!(item["name"], "cursor");
    assert!(item["revokedAt"].is_null());

    let mcp = bearer(&plaintext);
    let (status, minted) = call(
        router.clone(),
        Method::POST,
        "/auth/mcp-tokens",
        json!({"name": "from-token"}).to_string(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{minted}");
    let (status, listed_with_token) = call(
        router.clone(),
        Method::GET,
        "/auth/mcp-tokens",
        Body::empty(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{listed_with_token}");
    let (status, password_denied) = call(
        router.clone(),
        Method::PUT,
        "/auth/password",
        json!({"currentPassword": PASSWORD, "newPassword": "new-password-1"}).to_string(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{password_denied}");

    let (status, draft_missing) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": "Missing credential", "markdown": "no"}).to_string(),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{draft_missing}");
    assert_eq!(draft_missing["code"], "UNAUTHORIZED");

    let draft_title = format!("MCP draft {}", Uuid::new_v4());
    let (status, draft) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": draft_title, "markdown": "Written with an MCP token."}).to_string(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{draft}");
    assert_eq!(
        draft["data"]["article"]["author_id"],
        json!(admin_id.0.to_string())
    );
    assert_eq!(draft["data"]["author"]["id"], json!(admin_id.0.to_string()));
    assert!(draft["data"]["article"]["published_at"].is_null());
    let (_, _, last_used_at, _) = stored_token(&pool, token_id).await?;
    assert!(last_used_at.is_some());

    let (status, changed) = call(
        router.clone(),
        Method::PUT,
        "/auth/password",
        json!({"currentPassword": PASSWORD, "newPassword": "new-password-1"}).to_string(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{changed}");
    let (status, old_jwt) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": format!("Old JWT {}", Uuid::new_v4()), "markdown": "no"}).to_string(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{old_jwt}");

    let (status, after_password) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": format!("After password {}", Uuid::new_v4()), "markdown": "still"})
            .to_string(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::CREATED, "{after_password}");
    assert_eq!(
        after_password["data"]["article"]["author_id"],
        json!(admin_id.0.to_string())
    );

    let (status, revoked) = call(
        router.clone(),
        Method::DELETE,
        &format!("/auth/mcp-tokens/{token_id}"),
        Body::empty(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{revoked}");

    let refreshed = changed["data"]["token"]
        .as_str()
        .ok_or_else(|| io::Error::other("password update omitted the replacement JWT"))?;
    let refreshed = bearer(refreshed);
    let (status, revoked) = call(
        router.clone(),
        Method::DELETE,
        &format!("/auth/mcp-tokens/{token_id}"),
        Body::empty(),
        Some(&refreshed),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{revoked}");
    assert!(revoked["data"].get("token").is_none());
    assert!(revoked["data"]["revokedAt"].is_string());
    assert!(!revoked.to_string().contains(&plaintext));
    assert!(!revoked.to_string().contains(&hash));
    let (_, _, _, revoked_at) = stored_token(&pool, token_id).await?;
    assert!(revoked_at.is_some());

    let (status, revoked_use) = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        json!({"title": format!("Revoked {}", Uuid::new_v4()), "markdown": "no"}).to_string(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{revoked_use}");
    assert_eq!(revoked_use["code"], "UNAUTHORIZED");

    let (status, revoke_with_token) = call(
        router.clone(),
        Method::DELETE,
        &format!("/auth/mcp-tokens/{token_id}"),
        Body::empty(),
        Some(&mcp),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{revoke_with_token}");

    Ok(())
}
