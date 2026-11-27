use std::{env, error::Error, io, sync::Arc};

use async_trait::async_trait;
use axum::{
    Router,
    body::Body,
    extract::FromRef,
    http::{HeaderMap, Method, Request, StatusCode, header::CONTENT_TYPE},
    middleware,
};
use base64::{Engine, engine::general_purpose::URL_SAFE_NO_PAD};
use blog_backend::{
    api::{
        article::{
            ArticleGenerationQueue, ArticleState, GenerationRequest, router as article_router,
        },
        auth::{AuthState, attach_resource_metadata, router as auth_router},
    },
    core::{
        article::ArticleService,
        auth::{
            Account, AccountId, AccountRepository, AuthService, mcp_token_hash, oauth::pkce_s256,
        },
    },
    database::{
        pool::{PgPool, create_pool},
        repository::{
            account::DieselAccountRepository, article::DieselArticleRepository,
            tag::DieselTagRepository,
        },
    },
    schema::{mcp_access_token, oauth_authorization_code},
};
use diesel::{Connection, ExpressionMethods, PgConnection, QueryDsl};
use diesel_async::RunQueryDsl;
use diesel_migrations::{EmbeddedMigrations, MigrationHarness, embed_migrations};
use http_body_util::BodyExt;
use secrecy::SecretString;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use tower::ServiceExt;
use uuid::Uuid;

const MIGRATIONS: EmbeddedMigrations = embed_migrations!("migrations");
const TEST_SECRET: &str = "oauth-mcp-test-secret";
const PASSWORD: &str = "test-password";
const PUBLIC_BASE: &str = "https://blog.example";
const REDIRECT: &str = "https://claude.ai/api/mcp/auth_callback";
const VERIFIER: &str = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
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

struct HttpResponse {
    status: StatusCode,
    headers: HeaderMap,
    text: String,
    json: Value,
}

fn test_pool() -> TestResult<PgPool> {
    let database_url = env::var("TEST_DATABASE_URL").map_err(|_| {
        io::Error::other(
            "TEST_DATABASE_URL is required for the oauth_mcp target; start the Docker test PostgreSQL service and provide its URL",
        )
    })?;
    let mut connection = PgConnection::establish(&database_url)?;
    connection
        .run_pending_migrations(MIGRATIONS)
        .map_err(|error| io::Error::other(format!("oauth migration failed: {error}")))?;
    Ok(create_pool(&SecretString::from(database_url))?)
}

async fn call(
    router: Router,
    method: Method,
    path: &str,
    content_type: &str,
    body: impl Into<Body>,
    bearer: Option<&str>,
) -> TestResult<HttpResponse> {
    let mut builder = Request::builder()
        .method(method)
        .uri(path)
        .header(CONTENT_TYPE, content_type);
    if let Some(bearer) = bearer {
        builder = builder.header("authorization", bearer);
    }
    let response = router.oneshot(builder.body(body.into())?).await?;
    let status = response.status();
    let headers = response.headers().clone();
    let bytes = response.into_body().collect().await?.to_bytes();
    let text = String::from_utf8(bytes.to_vec())?;
    let json = if text.is_empty() {
        Value::Null
    } else {
        serde_json::from_str(&text).unwrap_or(Value::Null)
    };
    Ok(HttpResponse {
        status,
        headers,
        text,
        json,
    })
}

fn form(pairs: &[(&str, &str)]) -> String {
    pairs
        .iter()
        .map(|(key, value)| format!("{}={}", encode(key), encode(value)))
        .collect::<Vec<_>>()
        .join("&")
}

fn encode(value: &str) -> String {
    let mut encoded = String::new();
    for byte in value.bytes() {
        match byte {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'.' | b'_' | b'~' => {
                encoded.push(char::from(byte));
            }
            _ => encoded.push_str(&format!("%{byte:02X}")),
        }
    }
    encoded
}

fn query_param(url: &str, key: &str) -> Option<String> {
    let query = url.split_once('?')?.1;
    for pair in query.split('&') {
        let (name, value) = pair.split_once('=')?;
        if name == key {
            return Some(percent_decode(value));
        }
    }
    None
}

fn percent_decode(value: &str) -> String {
    let bytes = value.as_bytes();
    let mut decoded = Vec::new();
    let mut index = 0;
    while index < bytes.len() {
        if bytes[index] == b'%' && index + 2 < bytes.len() {
            let hex = &value[index + 1..index + 3];
            if let Ok(byte) = u8::from_str_radix(hex, 16) {
                decoded.push(byte);
                index += 3;
                continue;
            }
        }
        decoded.push(bytes[index]);
        index += 1;
    }
    String::from_utf8(decoded).unwrap_or_default()
}

fn bearer(token: &str) -> String {
    format!("Bearer {token}")
}

fn authorize_form(
    client_id: &str,
    challenge: &str,
    email: &str,
    password: &str,
    decision: &str,
) -> String {
    form(&[
        ("response_type", "code"),
        ("client_id", client_id),
        ("redirect_uri", REDIRECT),
        ("code_challenge", challenge),
        ("code_challenge_method", "S256"),
        ("state", "state-1"),
        ("email", email),
        ("password", password),
        ("decision", decision),
    ])
}

#[tokio::test]
async fn oauth_metadata_registration_and_pkce_issue_a_revocable_token() -> TestResult {
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
        auth: AuthState::with_public_base_url(auth_service.clone(), PUBLIC_BASE),
    };
    let auth = state.auth.clone();
    let router: Router = auth_router::<TestState>()
        .merge(article_router::<TestState>())
        .layer(middleware::from_fn_with_state(
            auth,
            attach_resource_metadata,
        ))
        .with_state(state)
        .into();

    let admin_id = AccountId(Uuid::new_v4());
    let admin = Account {
        id: admin_id,
        name: "OAuth Admin".to_owned(),
        email: format!("oauth-admin-{}@example.com", admin_id.0),
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
        name: "OAuth Member".to_owned(),
        email: format!("oauth-member-{}@example.com", member_id.0),
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

    let resource = call(
        router.clone(),
        Method::GET,
        "/.well-known/oauth-protected-resource",
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(resource.status, StatusCode::OK, "{}", resource.text);
    assert_eq!(resource.json["resource"], format!("{PUBLIC_BASE}/mcp"));
    assert_eq!(
        resource.json["authorization_servers"][0],
        json!(PUBLIC_BASE)
    );
    assert_eq!(resource.json["bearer_methods_supported"][0], "header");
    let resource_path = call(
        router.clone(),
        Method::GET,
        "/.well-known/oauth-protected-resource/mcp",
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(resource_path.json["resource"], resource.json["resource"]);

    let server = call(
        router.clone(),
        Method::GET,
        "/.well-known/oauth-authorization-server",
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(server.status, StatusCode::OK, "{}", server.text);
    assert_eq!(server.json["issuer"], PUBLIC_BASE);
    assert_eq!(
        server.json["authorization_endpoint"],
        format!("{PUBLIC_BASE}/oauth/authorize")
    );
    assert_eq!(
        server.json["token_endpoint"],
        format!("{PUBLIC_BASE}/oauth/token")
    );
    assert_eq!(
        server.json["registration_endpoint"],
        format!("{PUBLIC_BASE}/oauth/register")
    );
    assert_eq!(server.json["code_challenge_methods_supported"][0], "S256");
    assert_eq!(
        server.json["token_endpoint_auth_methods_supported"][0],
        "none"
    );

    let rejected = call(
        router.clone(),
        Method::POST,
        "/oauth/register",
        "application/json",
        json!({
            "client_name": "Claude",
            "redirect_uris": ["http://example.com/callback"]
        })
        .to_string(),
        None,
    )
    .await?;
    assert_eq!(
        rejected.status,
        StatusCode::BAD_REQUEST,
        "{}",
        rejected.text
    );
    assert_eq!(rejected.json["error"], "invalid_redirect_uri");

    let registered = call(
        router.clone(),
        Method::POST,
        "/oauth/register",
        "application/json",
        json!({
            "client_name": "Claude",
            "redirect_uris": [REDIRECT, "http://127.0.0.1:9/callback"],
            "grant_types": ["authorization_code", "refresh_token"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none"
        })
        .to_string(),
        None,
    )
    .await?;
    assert_eq!(
        registered.status,
        StatusCode::CREATED,
        "{}",
        registered.text
    );
    assert!(registered.json.get("client_secret").is_none());
    let client_id = registered.json["client_id"]
        .as_str()
        .ok_or_else(|| io::Error::other("registration omitted client_id"))?
        .to_owned();
    assert_eq!(registered.json["client_name"], "Claude");

    let challenge = pkce_s256(VERIFIER).ok_or_else(|| io::Error::other("verifier was rejected"))?;
    assert_eq!(
        challenge,
        URL_SAFE_NO_PAD.encode(Sha256::digest(VERIFIER.as_bytes()))
    );

    let plain = call(
        router.clone(),
        Method::GET,
        &format!(
            "/oauth/authorize?response_type=code&client_id={client_id}&redirect_uri={}&code_challenge={challenge}&code_challenge_method=plain&state=state-1",
            encode(REDIRECT)
        ),
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(plain.status, StatusCode::FOUND, "{}", plain.text);
    let plain_location = plain
        .headers
        .get("location")
        .and_then(|value| value.to_str().ok())
        .ok_or_else(|| io::Error::other("plain PKCE omitted the redirect"))?;
    assert_eq!(
        query_param(plain_location, "error").as_deref(),
        Some("invalid_request")
    );
    assert!(query_param(plain_location, "code").is_none());

    let unknown = call(
        router.clone(),
        Method::GET,
        "/oauth/authorize?response_type=code&client_id=missing&redirect_uri=https%3A%2F%2Fclaude.ai%2Fcb&code_challenge_method=S256",
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(unknown.status, StatusCode::BAD_REQUEST, "{}", unknown.text);
    assert!(unknown.headers.get("location").is_none());

    let form_page = call(
        router.clone(),
        Method::GET,
        &format!(
            "/oauth/authorize?response_type=code&client_id={client_id}&redirect_uri={}&code_challenge={challenge}&code_challenge_method=S256&state=state-1",
            encode(REDIRECT)
        ),
        "application/json",
        Body::empty(),
        None,
    )
    .await?;
    assert_eq!(form_page.status, StatusCode::OK, "{}", form_page.text);
    assert!(form_page.text.contains("Claude"));
    assert!(form_page.text.contains("Approve"));

    let wrong_password = call(
        router.clone(),
        Method::POST,
        "/oauth/authorize",
        "application/x-www-form-urlencoded",
        authorize_form(&client_id, &challenge, &admin.email, "nope-pass", "approve"),
        None,
    )
    .await?;
    assert_eq!(
        wrong_password.status,
        StatusCode::UNAUTHORIZED,
        "{}",
        wrong_password.text
    );

    let member_attempt = call(
        router.clone(),
        Method::POST,
        "/oauth/authorize",
        "application/x-www-form-urlencoded",
        authorize_form(&client_id, &challenge, &member.email, PASSWORD, "approve"),
        None,
    )
    .await?;
    assert_eq!(
        member_attempt.status,
        StatusCode::FORBIDDEN,
        "{}",
        member_attempt.text
    );

    let approved = call(
        router.clone(),
        Method::POST,
        "/oauth/authorize",
        "application/x-www-form-urlencoded",
        authorize_form(&client_id, &challenge, &admin.email, PASSWORD, "approve"),
        None,
    )
    .await?;
    assert_eq!(approved.status, StatusCode::FOUND, "{}", approved.text);
    let approved_location = header_str(&approved, "location")?;
    assert!(approved_location.starts_with(REDIRECT));
    assert_eq!(
        query_param(approved_location, "state").as_deref(),
        Some("state-1")
    );
    let bad_verifier_code = query_param(approved_location, "code")
        .ok_or_else(|| io::Error::other("approval omitted the code"))?;

    let mut connection = pool.get().await.map_err(io::Error::other)?;
    let stored_code_hash: String = oauth_authorization_code::table
        .filter(oauth_authorization_code::code_hash.eq(mcp_token_hash(&bad_verifier_code)))
        .select(oauth_authorization_code::code_hash)
        .first(&mut connection)
        .await?;
    assert_eq!(stored_code_hash, mcp_token_hash(&bad_verifier_code));
    assert_ne!(stored_code_hash, bad_verifier_code);
    drop(connection);

    let wrong_verifier = token_request(
        &router,
        &form(&[
            ("grant_type", "authorization_code"),
            ("code", &bad_verifier_code),
            ("redirect_uri", REDIRECT),
            ("client_id", &client_id),
            ("code_verifier", &"a".repeat(43)),
        ]),
    )
    .await?;
    assert_eq!(
        wrong_verifier.status,
        StatusCode::BAD_REQUEST,
        "{}",
        wrong_verifier.text
    );
    assert_eq!(wrong_verifier.json["error"], "invalid_grant");

    let reused = token_request(
        &router,
        &form(&[
            ("grant_type", "authorization_code"),
            ("code", &bad_verifier_code),
            ("redirect_uri", REDIRECT),
            ("client_id", &client_id),
            ("code_verifier", VERIFIER),
        ]),
    )
    .await?;
    assert_eq!(reused.status, StatusCode::BAD_REQUEST, "{}", reused.text);
    assert_eq!(reused.json["error"], "invalid_grant");

    let redirect_code = approval_code(&router, &client_id, &challenge, &admin.email).await?;
    let wrong_redirect = token_request(
        &router,
        &form(&[
            ("grant_type", "authorization_code"),
            ("code", &redirect_code),
            ("redirect_uri", "http://127.0.0.1:9/callback"),
            ("client_id", &client_id),
            ("code_verifier", VERIFIER),
        ]),
    )
    .await?;
    assert_eq!(
        wrong_redirect.status,
        StatusCode::BAD_REQUEST,
        "{}",
        wrong_redirect.text
    );
    assert_eq!(wrong_redirect.json["error"], "invalid_grant");

    let code = approval_code(&router, &client_id, &challenge, &admin.email).await?;
    let issued = token_request(
        &router,
        &form(&[
            ("grant_type", "authorization_code"),
            ("code", &code),
            ("redirect_uri", REDIRECT),
            ("client_id", &client_id),
            ("code_verifier", VERIFIER),
        ]),
    )
    .await?;
    assert_eq!(issued.status, StatusCode::OK, "{}", issued.text);
    assert_eq!(issued.json["token_type"], "Bearer");
    assert_eq!(issued.json["expires_in"], 3600);
    let access_token = issued.json["access_token"]
        .as_str()
        .ok_or_else(|| io::Error::other("token response omitted access_token"))?
        .to_owned();
    let refresh_token = issued.json["refresh_token"]
        .as_str()
        .ok_or_else(|| io::Error::other("token response omitted refresh_token"))?
        .to_owned();
    assert!(access_token.starts_with("mcp_"));
    assert!(refresh_token.starts_with("mcr_"));
    assert_eq!(access_token.len(), 68);

    let mut connection = pool.get().await.map_err(io::Error::other)?;
    let (stored_hash, stored_client): (String, Option<String>) = mcp_access_token::table
        .filter(mcp_access_token::token_hash.eq(mcp_token_hash(&access_token)))
        .select((mcp_access_token::token_hash, mcp_access_token::client_name))
        .first(&mut connection)
        .await?;
    assert_eq!(stored_hash, mcp_token_hash(&access_token));
    assert_ne!(stored_hash, access_token);
    assert_eq!(stored_client.as_deref(), Some("Claude"));
    drop(connection);

    let draft = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": format!("OAuth draft {}", Uuid::new_v4()), "markdown": "from oauth"})
            .to_string(),
        Some(&bearer(&access_token)),
    )
    .await?;
    assert_eq!(draft.status, StatusCode::CREATED, "{}", draft.text);
    assert_eq!(
        draft.json["data"]["article"]["author_id"],
        json!(admin_id.0.to_string())
    );
    assert_eq!(
        draft.json["data"]["author"]["id"],
        json!(admin_id.0.to_string())
    );

    let minted = call(
        router.clone(),
        Method::POST,
        "/auth/mcp-tokens",
        "application/json",
        json!({"name": "from-oauth"}).to_string(),
        Some(&bearer(&access_token)),
    )
    .await?;
    assert_eq!(minted.status, StatusCode::UNAUTHORIZED, "{}", minted.text);
    let password_change = call(
        router.clone(),
        Method::PUT,
        "/auth/password",
        "application/json",
        json!({"currentPassword": PASSWORD, "newPassword": "new-password-1"}).to_string(),
        Some(&bearer(&access_token)),
    )
    .await?;
    assert_eq!(
        password_change.status,
        StatusCode::UNAUTHORIZED,
        "{}",
        password_change.text
    );

    let listed = call(
        router.clone(),
        Method::GET,
        "/auth/mcp-tokens",
        "application/json",
        Body::empty(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(listed.status, StatusCode::OK, "{}", listed.text);
    assert!(!listed.text.contains(&access_token));
    assert!(!listed.text.contains(&refresh_token));
    assert!(!listed.text.contains(&stored_hash));
    let item = listed.json["data"]
        .as_array()
        .and_then(|items| items.iter().find(|item| item["clientName"] == "Claude"))
        .ok_or_else(|| io::Error::other("token list omitted the oauth client"))?;
    assert_eq!(item["name"], "Claude");
    assert!(item.get("token").is_none());
    assert!(item.get("tokenHash").is_none());
    let token_id = item["id"]
        .as_str()
        .ok_or_else(|| io::Error::other("listed token omitted id"))?
        .to_owned();

    let changed = call(
        router.clone(),
        Method::PUT,
        "/auth/password",
        "application/json",
        json!({"currentPassword": PASSWORD, "newPassword": "new-password-1"}).to_string(),
        Some(&admin_jwt),
    )
    .await?;
    assert_eq!(changed.status, StatusCode::OK, "{}", changed.text);
    let still = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": format!("After password {}", Uuid::new_v4()), "markdown": "still"})
            .to_string(),
        Some(&bearer(&access_token)),
    )
    .await?;
    assert_eq!(still.status, StatusCode::CREATED, "{}", still.text);

    let refreshed = token_request(
        &router,
        &form(&[
            ("grant_type", "refresh_token"),
            ("refresh_token", &refresh_token),
            ("client_id", &client_id),
        ]),
    )
    .await?;
    assert_eq!(refreshed.status, StatusCode::OK, "{}", refreshed.text);
    let new_access = refreshed.json["access_token"]
        .as_str()
        .ok_or_else(|| io::Error::other("refresh omitted access_token"))?
        .to_owned();
    let new_refresh = refreshed.json["refresh_token"]
        .as_str()
        .ok_or_else(|| io::Error::other("refresh omitted refresh_token"))?
        .to_owned();
    assert_ne!(new_access, access_token);
    assert_ne!(new_refresh, refresh_token);

    let old_refresh = token_request(
        &router,
        &form(&[
            ("grant_type", "refresh_token"),
            ("refresh_token", &refresh_token),
            ("client_id", &client_id),
        ]),
    )
    .await?;
    assert_eq!(
        old_refresh.status,
        StatusCode::BAD_REQUEST,
        "{}",
        old_refresh.text
    );
    assert_eq!(old_refresh.json["error"], "invalid_grant");

    let rotated = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": format!("Rotated {}", Uuid::new_v4()), "markdown": "new"}).to_string(),
        Some(&bearer(&new_access)),
    )
    .await?;
    assert_eq!(rotated.status, StatusCode::CREATED, "{}", rotated.text);
    let old_access = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": format!("Old access {}", Uuid::new_v4()), "markdown": "old"}).to_string(),
        Some(&bearer(&access_token)),
    )
    .await?;
    assert_eq!(
        old_access.status,
        StatusCode::UNAUTHORIZED,
        "{}",
        old_access.text
    );

    let replacement = changed.json["data"]["token"]
        .as_str()
        .ok_or_else(|| io::Error::other("password update omitted the replacement JWT"))?;
    let revoked = call(
        router.clone(),
        Method::DELETE,
        &format!("/auth/mcp-tokens/{token_id}"),
        "application/json",
        Body::empty(),
        Some(&bearer(replacement)),
    )
    .await?;
    assert_eq!(revoked.status, StatusCode::OK, "{}", revoked.text);
    assert_eq!(revoked.json["data"]["clientName"], "Claude");
    assert!(revoked.json["data"]["revokedAt"].is_string());
    assert!(!revoked.text.contains(&new_access));

    let revoked_use = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": format!("Revoked {}", Uuid::new_v4()), "markdown": "no"}).to_string(),
        Some(&bearer(&new_access)),
    )
    .await?;
    let metadata =
        format!("Bearer resource_metadata=\"{PUBLIC_BASE}/.well-known/oauth-protected-resource\"");
    assert_eq!(
        revoked_use.status,
        StatusCode::UNAUTHORIZED,
        "{}",
        revoked_use.text
    );
    assert_eq!(revoked_use.json["error"], "authentication required");
    assert_eq!(revoked_use.json["code"], "UNAUTHORIZED");
    assert_eq!(header_str(&revoked_use, "www-authenticate")?, metadata);

    let anonymous = call(
        router.clone(),
        Method::POST,
        "/blog/drafts",
        "application/json",
        json!({"title": "Missing", "markdown": "no"}).to_string(),
        None,
    )
    .await?;
    assert_eq!(
        anonymous.status,
        StatusCode::UNAUTHORIZED,
        "{}",
        anonymous.text
    );
    assert_eq!(anonymous.json["error"], "authentication required");
    assert_eq!(header_str(&anonymous, "www-authenticate")?, metadata);

    Ok(())
}

async fn token_request(router: &Router, body: &str) -> TestResult<HttpResponse> {
    call(
        router.clone(),
        Method::POST,
        "/oauth/token",
        "application/x-www-form-urlencoded",
        body.to_owned(),
        None,
    )
    .await
}

async fn approval_code(
    router: &Router,
    client_id: &str,
    challenge: &str,
    email: &str,
) -> TestResult<String> {
    let approved = call(
        router.clone(),
        Method::POST,
        "/oauth/authorize",
        "application/x-www-form-urlencoded",
        authorize_form(client_id, challenge, email, PASSWORD, "approve"),
        None,
    )
    .await?;
    if approved.status != StatusCode::FOUND {
        return Err(io::Error::other(format!("approval failed: {}", approved.text)).into());
    }
    let location = header_str(&approved, "location")?;
    query_param(location, "code")
        .ok_or_else(|| io::Error::other("approval omitted the code").into())
}

fn header_str<'a>(response: &'a HttpResponse, name: &str) -> TestResult<&'a str> {
    response
        .headers
        .get(name)
        .and_then(|value| value.to_str().ok())
        .ok_or_else(|| io::Error::other(format!("response omitted {name}")).into())
}
