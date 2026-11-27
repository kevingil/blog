use std::{
    collections::HashMap,
    error::Error,
    sync::{
        Arc, Mutex,
        atomic::{AtomicUsize, Ordering},
    },
    time::{Duration as StdDuration, Instant},
};

use async_trait::async_trait;
use axum::{
    Json, Router,
    body::Body,
    http::{
        HeaderMap, Request, StatusCode,
        header::{AUTHORIZATION, WWW_AUTHENTICATE},
    },
    response::IntoResponse,
    routing::{get, post as route_post},
};
use blog_backend::{
    api::{auth::AuthState, mcp::router as mcp_router},
    core::auth::{
        Account, AccountId, AccountRepository, AuthService, AuthorizationCode, McpAccessToken,
        NewAuthorizationCode, NewMcpAccessToken, OauthClient, mcp_token_hash,
        oauth::{pkce_s256, resource_metadata_header},
    },
    error::AppError,
};
use chrono::{Duration, Utc};
use http_body_util::BodyExt;
use serde_json::{Value, json};
use tower::ServiceExt;
use uuid::Uuid;

const SECRET: &str = "mcp-server-test-secret";
const VERIFIER: &str = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
const PUBLIC_BASE: &str = "http://localhost:8080";
type TestResult<T = ()> = Result<T, Box<dyn Error + Send + Sync>>;

#[derive(Default)]
struct MemoryAccounts {
    accounts: Mutex<HashMap<Uuid, Account>>,
    clients: Mutex<HashMap<String, OauthClient>>,
    codes: Mutex<HashMap<String, AuthorizationCode>>,
    tokens: Mutex<HashMap<String, McpAccessToken>>,
}

impl MemoryAccounts {
    fn seed_client(&self, client: OauthClient) -> TestResult {
        self.clients
            .lock()
            .map_err(|error| format!("client lock: {error}"))?
            .insert(client.client_id.clone(), client);
        Ok(())
    }

    fn seed_code(&self, code: NewAuthorizationCode) -> TestResult {
        self.codes
            .lock()
            .map_err(|error| format!("code lock: {error}"))?
            .insert(
                code.code_hash,
                AuthorizationCode {
                    client_id: code.client_id,
                    account_id: code.account_id,
                    redirect_uri: code.redirect_uri,
                    code_challenge: code.code_challenge,
                },
            );
        Ok(())
    }
}

#[async_trait]
impl AccountRepository for MemoryAccounts {
    async fn find_by_id(&self, id: AccountId) -> Result<Option<Account>, AppError> {
        Ok(self
            .accounts
            .lock()
            .map_err(AppError::internal)?
            .get(&id.0)
            .cloned())
    }

    async fn find_by_email(&self, _email: &str) -> Result<Option<Account>, AppError> {
        Ok(None)
    }

    async fn create(&self, account: &Account) -> Result<(), AppError> {
        self.accounts
            .lock()
            .map_err(AppError::internal)?
            .insert(account.id.0, account.clone());
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

    async fn insert_mcp_token(&self, token: &NewMcpAccessToken) -> Result<(), AppError> {
        let record = McpAccessToken {
            id: token.id,
            account_id: token.account_id,
            name: token.name.clone(),
            token_prefix: token.token_prefix.clone(),
            created_at: token.created_at,
            last_used_at: None,
            revoked_at: None,
            client_name: token.client_name.clone(),
            expires_at: token.expires_at,
        };
        self.tokens
            .lock()
            .map_err(AppError::internal)?
            .insert(token.token_hash.clone(), record);
        Ok(())
    }

    async fn find_active_mcp_token_by_hash(
        &self,
        token_hash: &str,
    ) -> Result<Option<McpAccessToken>, AppError> {
        let tokens = self.tokens.lock().map_err(AppError::internal)?;
        Ok(tokens
            .get(token_hash)
            .filter(|token| token.revoked_at.is_none())
            .cloned())
    }

    async fn touch_mcp_token_last_used(&self, _id: Uuid) -> Result<(), AppError> {
        Ok(())
    }

    async fn find_oauth_client(&self, client_id: &str) -> Result<Option<OauthClient>, AppError> {
        Ok(self
            .clients
            .lock()
            .map_err(AppError::internal)?
            .get(client_id)
            .cloned())
    }

    async fn consume_authorization_code(
        &self,
        code_hash: &str,
    ) -> Result<Option<AuthorizationCode>, AppError> {
        Ok(self
            .codes
            .lock()
            .map_err(AppError::internal)?
            .remove(code_hash))
    }
}

struct Harness {
    router: Router,
    static_token: String,
    oauth_token: String,
    drafts: Arc<Mutex<Vec<Value>>>,
    forbidden_hits: Arc<AtomicUsize>,
}

async fn harness() -> TestResult<Harness> {
    let accounts = std::sync::Arc::new(MemoryAccounts::default());
    let auth_service = std::sync::Arc::new(AuthService::new(accounts.clone(), SECRET)?);
    let admin = Account {
        id: AccountId(Uuid::new_v4()),
        name: "MCP Server".to_owned(),
        email: "mcp-server@example.com".to_owned(),
        password_hash: "unused".to_owned(),
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
    let static_token = auth_service
        .create_mcp_token(admin.id, "cursor")
        .await?
        .plaintext;

    let client_id = "client-1".to_owned();
    let redirect = "http://127.0.0.1:3000/callback".to_owned();
    accounts.seed_client(OauthClient {
        client_id: client_id.clone(),
        client_name: "Claude".to_owned(),
        redirect_uris: vec![redirect.clone()],
    })?;
    let code = "oauth-code";
    let challenge = pkce_s256(VERIFIER).ok_or("pkce verifier was rejected")?;
    accounts.seed_code(NewAuthorizationCode {
        code_hash: mcp_token_hash(code),
        client_id: client_id.clone(),
        account_id: admin.id,
        redirect_uri: redirect.clone(),
        code_challenge: challenge,
        expires_at: Utc::now() + Duration::minutes(5),
    })?;
    let oauth_token = auth_service
        .exchange_authorization_code(&client_id, code, &redirect, VERIFIER)
        .await
        .map_err(|error| format!("oauth exchange failed: {error:?}"))?
        .access_token;

    let auth = AuthState::with_public_base_url(auth_service, PUBLIC_BASE);
    let (api, drafts, forbidden_hits) = content_api(static_token.clone());
    Ok(Harness {
        router: mcp_router(auth, api),
        static_token,
        oauth_token,
        drafts,
        forbidden_hits,
    })
}

fn content_api(token: String) -> (Router, Arc<Mutex<Vec<Value>>>, Arc<AtomicUsize>) {
    let drafts = Arc::new(Mutex::new(Vec::new()));
    let forbidden_hits = Arc::new(AtomicUsize::new(0));
    let create_drafts = Arc::clone(&drafts);
    let create_token = token.clone();
    let list_drafts = Arc::clone(&drafts);
    let list_token = token;
    let forbidden = Arc::clone(&forbidden_hits);
    let router = Router::new()
        .route(
            "/blog/drafts",
            route_post(move |headers: HeaderMap, Json(body): Json<Value>| {
                let drafts = Arc::clone(&create_drafts);
                let token = create_token.clone();
                async move { store_draft(headers, body, &drafts, &token) }
            }),
        )
        .route(
            "/blog/articles",
            get(move |headers: HeaderMap| {
                let drafts = Arc::clone(&list_drafts);
                let token = list_token.clone();
                async move { list_drafts_response(headers, &drafts, &token) }
            }),
        )
        .route(
            "/auth/me",
            get(move || {
                let forbidden = Arc::clone(&forbidden);
                async move {
                    forbidden.fetch_add(1, Ordering::SeqCst);
                    Json(json!({"leaked": true}))
                }
            }),
        );
    (router, drafts, forbidden_hits)
}

fn bearer_matches(headers: &HeaderMap, token: &str) -> bool {
    let expected = format!("Bearer {token}");
    headers
        .get(AUTHORIZATION)
        .and_then(|value| value.to_str().ok())
        == Some(expected.as_str())
}

fn store_draft(
    headers: HeaderMap,
    body: Value,
    drafts: &Mutex<Vec<Value>>,
    token: &str,
) -> axum::response::Response {
    if !bearer_matches(&headers, token) {
        return (
            StatusCode::UNAUTHORIZED,
            Json(json!({"error": "unauthorized"})),
        )
            .into_response();
    }
    let draft = json!({
        "title": body.get("title").cloned().unwrap_or(Value::Null),
        "markdown": body.get("markdown").cloned().unwrap_or(Value::Null),
    });
    let Ok(mut stored) = drafts.lock() else {
        return (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({"error": "store"})),
        )
            .into_response();
    };
    stored.push(draft.clone());
    (StatusCode::CREATED, Json(json!({"data": draft}))).into_response()
}

fn list_drafts_response(
    headers: HeaderMap,
    drafts: &Mutex<Vec<Value>>,
    token: &str,
) -> axum::response::Response {
    if !bearer_matches(&headers, token) {
        return (
            StatusCode::UNAUTHORIZED,
            Json(json!({"error": "unauthorized"})),
        )
            .into_response();
    }
    let articles = drafts
        .lock()
        .map(|stored| stored.clone())
        .unwrap_or_default();
    Json(json!({"data": {"articles": articles}})).into_response()
}

async fn post(
    router: Router,
    body: Value,
    bearer: Option<&str>,
) -> TestResult<(StatusCode, Option<String>, Value)> {
    let mut builder = Request::builder()
        .method("POST")
        .uri("/mcp")
        .header("host", "localhost")
        .header("content-type", "application/json")
        .header("accept", "application/json, text/event-stream");
    if let Some(bearer) = bearer {
        builder = builder.header("authorization", format!("Bearer {bearer}"));
    }
    let response = router
        .oneshot(builder.body(Body::from(body.to_string()))?)
        .await?;
    let status = response.status();
    let authenticate = response
        .headers()
        .get(WWW_AUTHENTICATE)
        .and_then(|value| value.to_str().ok())
        .map(str::to_owned);
    let content_type = response
        .headers()
        .get("content-type")
        .and_then(|value| value.to_str().ok())
        .unwrap_or("")
        .to_owned();
    let bytes = response.into_body().collect().await?.to_bytes();
    let payload = if bytes.is_empty() {
        Value::Null
    } else {
        decode_rpc(&bytes, &content_type).map_err(|error| {
            format!(
                "{error}; status={status}; content-type={content_type}; body={}",
                String::from_utf8_lossy(&bytes)
            )
        })?
    };
    Ok((status, authenticate, payload))
}

fn decode_rpc(bytes: &[u8], content_type: &str) -> TestResult<Value> {
    if content_type.contains("text/event-stream") {
        let text = String::from_utf8(bytes.to_vec())?;
        let data = text
            .lines()
            .filter_map(|line| line.strip_prefix("data:"))
            .map(str::trim)
            .rfind(|line| !line.is_empty())
            .ok_or("sse response had no data")?;
        return Ok(serde_json::from_str(data)?);
    }
    Ok(serde_json::from_slice(bytes)?)
}

fn rpc(id: i64, method: &str, params: Value) -> Value {
    json!({
        "jsonrpc": "2.0",
        "id": id,
        "method": method,
        "params": params,
    })
}

fn result_of(payload: &Value) -> TestResult<&Value> {
    if let Some(error) = payload.get("error") {
        return Err(format!("json-rpc error: {error}").into());
    }
    payload
        .get("result")
        .ok_or_else(|| "json-rpc response had no result".into())
}

fn assert_content_paths(operations: &[Value]) {
    for operation in operations {
        let path = operation["path"].as_str().unwrap_or("");
        for excluded in [
            "/insights",
            "/data-sources",
            "/workers",
            "/task-runs",
            "/agent",
            "/auth",
            "/mcp",
            "/oauth",
            "/.well-known",
        ] {
            assert!(
                path != excluded && !path.starts_with(&format!("{excluded}/")),
                "{path} should not be searchable"
            );
        }
    }
}

fn operations_from_call(payload: &Value) -> TestResult<Vec<Value>> {
    let result = result_of(payload)?;
    assert_eq!(result["isError"], false, "{result}");
    let text = result["content"]
        .as_array()
        .and_then(|blocks| blocks.first())
        .and_then(|block| block["text"].as_str())
        .ok_or("tool result had no text")?;
    let parsed: Value = serde_json::from_str(text)?;
    parsed["operations"]
        .as_array()
        .cloned()
        .ok_or_else(|| "search result had no operations".into())
}

#[tokio::test]
async fn unauthenticated_mcp_post_challenges_with_resource_metadata() -> TestResult {
    let harness = harness().await?;
    let (status, header, body) = post(
        harness.router.clone(),
        rpc(
            1,
            "initialize",
            json!({
                "protocolVersion": "2025-11-25",
                "capabilities": {},
                "clientInfo": {"name": "blog-test", "version": "0"}
            }),
        ),
        None,
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{body}");
    assert_eq!(
        header.as_deref(),
        Some(resource_metadata_header(PUBLIC_BASE).as_str())
    );

    let (status, header, body) = post(
        harness.router.clone(),
        rpc(1, "initialize", json!({})),
        Some("not-a-token"),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{body}");
    assert_eq!(
        header.as_deref(),
        Some(resource_metadata_header(PUBLIC_BASE).as_str())
    );

    let unknown = format!("mcp_{}", "ab".repeat(32));
    let (status, header, body) = post(
        harness.router,
        rpc(1, "initialize", json!({})),
        Some(&unknown),
    )
    .await?;
    assert_eq!(status, StatusCode::UNAUTHORIZED, "{body}");
    assert_eq!(
        header.as_deref(),
        Some(resource_metadata_header(PUBLIC_BASE).as_str())
    );
    Ok(())
}

#[tokio::test]
async fn static_and_oauth_tokens_list_search_and_hide_excluded_paths() -> TestResult {
    let harness = harness().await?;
    let (status, _, initialized) = post(
        harness.router.clone(),
        rpc(
            1,
            "initialize",
            json!({
                "protocolVersion": "2025-11-25",
                "capabilities": {},
                "clientInfo": {"name": "blog-test", "version": "0"}
            }),
        ),
        Some(&harness.static_token),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{initialized}");
    let result = result_of(&initialized)?;
    assert_eq!(result["serverInfo"]["name"], "blog");
    assert!(
        result["instructions"]
            .as_str()
            .unwrap_or("")
            .contains("execute"),
        "{result}"
    );

    let (status, _, listed) = post(
        harness.router.clone(),
        rpc(2, "tools/list", json!({})),
        Some(&harness.static_token),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{listed}");
    let tools = result_of(&listed)?["tools"]
        .as_array()
        .cloned()
        .ok_or("tools/list had no tools")?;
    let mut names: Vec<&str> = tools
        .iter()
        .filter_map(|tool| tool["name"].as_str())
        .collect();
    names.sort_unstable();
    assert_eq!(names, ["execute", "search"], "{tools:?}");
    let description = tools
        .iter()
        .find(|tool| tool["name"] == "search")
        .and_then(|tool| tool["description"].as_str())
        .unwrap_or("");
    assert!(
        description.contains("execute"),
        "tool description should point at execute: {description}"
    );

    for query in [
        "draft", "insight", "agent", "auth", "login", "worker", "oauth",
    ] {
        let (status, _, called) = post(
            harness.router.clone(),
            rpc(
                3,
                "tools/call",
                json!({"name": "search", "arguments": {"query": query}}),
            ),
            Some(&harness.static_token),
        )
        .await?;
        assert_eq!(status, StatusCode::OK, "{query}: {called}");
        let operations = operations_from_call(&called)?;
        assert_content_paths(&operations);
        if query == "draft" {
            let found = operations.iter().any(|operation| {
                operation["method"] == "POST" && operation["path"] == "/blog/drafts"
            });
            assert!(
                found,
                "draft search missing POST /blog/drafts: {operations:?}"
            );
        }
    }

    let (status, _, oauth_initialized) = post(
        harness.router,
        rpc(
            1,
            "initialize",
            json!({
                "protocolVersion": "2025-11-25",
                "capabilities": {},
                "clientInfo": {"name": "claude", "version": "0"}
            }),
        ),
        Some(&harness.oauth_token),
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{oauth_initialized}");
    assert_eq!(result_of(&oauth_initialized)?["serverInfo"]["name"], "blog");
    Ok(())
}

async fn execute(router: Router, token: &str, code: &str) -> TestResult<(StatusCode, Value)> {
    let (status, _, payload) = post(
        router,
        rpc(
            4,
            "tools/call",
            json!({"name": "execute", "arguments": {"code": code}}),
        ),
        Some(token),
    )
    .await?;
    Ok((status, payload))
}

fn tool_text(payload: &Value) -> TestResult<(bool, &str)> {
    let result = result_of(payload)?;
    let is_error = result["isError"]
        .as_bool()
        .ok_or("tool result had no isError")?;
    let text = result["content"]
        .as_array()
        .and_then(|blocks| blocks.first())
        .and_then(|block| block["text"].as_str())
        .ok_or("tool result had no text")?;
    Ok((is_error, text))
}

#[tokio::test]
async fn execute_creates_a_draft_and_reads_it_back() -> TestResult {
    let harness = harness().await?;
    let code = r#"
        const created = await blog.request({
            method: "POST",
            path: "/blog/drafts",
            body: { title: "From execute", markdown: "hello from execute" }
        });
        if (created.status !== 201) {
            throw new Error("create failed " + created.status);
        }
        console.log("created", created.body.data.title);
        const listed = await blog.request({ method: "GET", path: "/blog/articles" });
        const titles = listed.body.data.articles.map((article) => article.title);
        if (!titles.includes("From execute")) {
            throw new Error("draft missing " + JSON.stringify(listed));
        }
        return { title: created.body.data.title, titles };
    "#;
    let (status, payload) = execute(harness.router, &harness.static_token, code).await?;
    assert_eq!(status, StatusCode::OK, "{payload}");
    let (is_error, text) = tool_text(&payload)?;
    assert!(!is_error, "{text}");
    let body: Value = serde_json::from_str(text)?;
    assert_eq!(body["result"]["title"], "From execute");
    assert!(
        body["console"][0]
            .as_str()
            .unwrap_or("")
            .contains("From execute")
    );
    let stored = harness
        .drafts
        .lock()
        .map_err(|error| format!("draft lock: {error}"))?;
    assert_eq!(stored.len(), 1, "{stored:?}");
    assert_eq!(stored[0]["title"], "From execute");
    assert_eq!(harness.forbidden_hits.load(Ordering::SeqCst), 0);
    Ok(())
}

#[tokio::test]
async fn execute_rejects_a_disallowed_path_without_dispatching() -> TestResult {
    let harness = harness().await?;
    for code in [
        r#"return await blog.request({ method: "GET", path: "/auth/me" });"#,
        r#"return await blog.request({ method: "GET", path: "/mcp" });"#,
        r#"return await blog.request({ method: "POST", path: "/blog/../auth/me" });"#,
        r#"return await blog.request({ method: "GET", path: "/insights/topics" });"#,
    ] {
        let (status, payload) =
            execute(harness.router.clone(), &harness.static_token, code).await?;
        assert_eq!(status, StatusCode::OK, "{code}: {payload}");
        let (is_error, text) = tool_text(&payload)?;
        assert!(is_error, "{code}: {text}");
        assert!(
            text.contains("refused") || text.contains("outside"),
            "{code}: {text}"
        );
        assert!(!text.contains("leaked"), "{code} dispatched: {text}");
    }
    assert_eq!(harness.forbidden_hits.load(Ordering::SeqCst), 0);
    Ok(())
}

#[tokio::test]
async fn execute_interrupts_an_infinite_loop() -> TestResult {
    let harness = harness().await?;
    let started = Instant::now();
    let (status, payload) = tokio::time::timeout(
        StdDuration::from_secs(30),
        execute(harness.router, &harness.static_token, "while (true) {}"),
    )
    .await
    .map_err(|_| "infinite loop was not interrupted within 30 seconds")??;
    assert!(
        started.elapsed() < StdDuration::from_secs(30),
        "loop ran for {:?}",
        started.elapsed()
    );
    assert_eq!(status, StatusCode::OK, "{payload}");
    let (is_error, text) = tool_text(&payload)?;
    assert!(is_error, "{text}");
    assert!(text.contains("interrupted"), "{text}");
    Ok(())
}

#[tokio::test]
async fn execute_stops_a_memory_bomb() -> TestResult {
    let harness = harness().await?;
    let (status, payload) = execute(
        harness.router,
        &harness.static_token,
        r#""x".repeat(32 * 1024 * 1024)"#,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{payload}");
    let (is_error, text) = tool_text(&payload)?;
    assert!(is_error, "{text}");
    assert!(text.to_ascii_lowercase().contains("memory"), "{text}");
    Ok(())
}

#[tokio::test]
async fn execute_returns_a_javascript_exception() -> TestResult {
    let harness = harness().await?;
    let (status, payload) = execute(
        harness.router,
        &harness.static_token,
        r#"throw new Error("boom");"#,
    )
    .await?;
    assert_eq!(status, StatusCode::OK, "{payload}");
    let (is_error, text) = tool_text(&payload)?;
    assert!(is_error, "{text}");
    assert!(text.contains("boom"), "{text}");
    Ok(())
}
