mod catalog;
mod execute;

use std::{sync::Arc, thread};

use axum::{
    extract::{Request, State},
    http::{HeaderValue, StatusCode, header::WWW_AUTHENTICATE},
    middleware::Next,
    response::{IntoResponse, Response},
};
use rmcp::{
    ErrorData as McpError, ServerHandler,
    handler::server::{tool::Extension, wrapper::Parameters},
    model::{CallToolResult, ContentBlock},
    schemars, tool, tool_handler, tool_router,
    transport::streamable_http_server::{
        StreamableHttpServerConfig, StreamableHttpService, session::local::LocalSessionManager,
    },
};
use serde::Deserialize;
use serde_json::json;

use crate::{
    api::auth::{AuthState, state::bearer_token},
    core::auth::oauth::resource_metadata_header,
    error::AppError,
};

const SERVER_NAME: &str = "blog";
const SERVER_VERSION: &str = env!("CARGO_PKG_VERSION");
const SERVER_INSTRUCTIONS: &str = "Blog MCP server for the blog's own HTTP API. Call search to find operations on articles, pages, projects, sources, and storage, then call execute to run them. search returns the method, path, operation id, parameters, and request and response schemas, and does not call the API. execute runs a short async JavaScript snippet. await blog.request({ method, path, query, body }) and return a JSON value. blog.request dispatches only to those same routes, in-process, with the caller's bearer token. Other routes, network, files, and imports are unavailable.";
const DEFAULT_SEARCH_LIMIT: u32 = 10;
const MAX_SEARCH_LIMIT: u32 = 25;
/// Static tokens and OAuth access tokens issued for this server share this prefix.
const MCP_ACCESS_TOKEN_PREFIX: &str = "mcp_";

#[derive(Debug, Deserialize, schemars::JsonSchema)]
struct ExecuteArgs {
    /// Async function body. await blog.request({ method, path, query, body }) and return JSON.
    code: String,
}

#[derive(Debug, Deserialize, schemars::JsonSchema)]
struct SearchArgs {
    /// Words matched against method, path, operation id, summary, and parameters.
    query: String,
    /// Maximum operations to return. Defaults to 10 and cannot exceed 25.
    #[serde(default)]
    limit: Option<u32>,
}

#[derive(Clone)]
struct BlogMcp {
    api: axum::Router,
}

impl BlogMcp {
    fn new(api: axum::Router) -> Self {
        Self { api }
    }
}

#[tool_router]
impl BlogMcp {
    #[tool(
        description = "Find blog API operations to call with execute. Pass a query such as \"draft\" or \"upload\". The result lists matching articles, pages, projects, sources, and storage operations with method, path, operationId, parameters, and request or response schemas. search does not call the API. Call execute afterwards and use blog.request for the operation you found."
    )]
    fn search(
        &self,
        Parameters(SearchArgs { query, limit }): Parameters<SearchArgs>,
    ) -> Result<CallToolResult, McpError> {
        let query = query.trim();
        if query.is_empty() {
            return Ok(CallToolResult::error(vec![ContentBlock::text(
                "query must not be empty",
            )]));
        }
        let operations = catalog::search(&crate::openapi::document(), query, clamp_limit(limit));
        let payload = json!({ "operations": operations });
        let text = serde_json::to_string_pretty(&payload)
            .unwrap_or_else(|_| "{\"operations\":[]}".to_owned());
        Ok(CallToolResult::success(vec![ContentBlock::text(text)]))
    }

    #[tool(
        description = "Run a short async JavaScript snippet against the blog API. The snippet is an async function body: await blog.request({ method, path, query, body }) and return a JSON value. blog.request calls one articles, pages, projects, sources, or storage route in-process and returns { status, body }, using the same bearer token that called this tool. console.log is returned with the result. fetch, files, timers, and imports are not available. A thrown error, a disallowed path, a timeout, or a limit is a tool error."
    )]
    async fn execute(
        &self,
        Parameters(ExecuteArgs { code }): Parameters<ExecuteArgs>,
        Extension(parts): Extension<http::request::Parts>,
    ) -> Result<CallToolResult, McpError> {
        let code = code.trim();
        if code.is_empty() {
            return Ok(tool_error("code must not be empty"));
        }
        let authorization = match caller_authorization(&parts) {
            Ok(value) => value,
            Err(message) => return Ok(tool_error(message)),
        };
        let job = execute::Job {
            api: self.api.clone(),
            authorization,
            code: code.to_owned(),
        };
        let handle = tokio::runtime::Handle::current();
        let (sender, receiver) = tokio::sync::oneshot::channel();
        if let Err(error) = thread::Builder::new()
            .name("mcp-execute".to_owned())
            .spawn(move || {
                let _ = sender.send(execute::run(handle, job));
            })
        {
            return Ok(tool_error(format!("could not start execution: {error}")));
        }
        match receiver.await {
            Ok(Ok(outcome)) => match execute::tool_text(&outcome) {
                Ok(text) => Ok(CallToolResult::success(vec![ContentBlock::text(text)])),
                Err(message) => Ok(tool_error(message)),
            },
            Ok(Err(message)) => Ok(tool_error(message)),
            Err(_) => Ok(tool_error("execution failed")),
        }
    }
}

#[tool_handler]
impl ServerHandler for BlogMcp {
    fn get_info(&self) -> rmcp::model::ServerConfig {
        rmcp::model::ServerConfig::new(
            rmcp::model::ServerCapabilities::builder()
                .enable_tools()
                .build(),
        )
        .with_server_info(rmcp::model::Implementation::new(
            SERVER_NAME,
            SERVER_VERSION,
        ))
        .with_instructions(SERVER_INSTRUCTIONS)
    }
}

fn clamp_limit(limit: Option<u32>) -> usize {
    let bounded = limit
        .unwrap_or(DEFAULT_SEARCH_LIMIT)
        .clamp(1, MAX_SEARCH_LIMIT);
    usize::try_from(bounded).unwrap_or(10)
}

fn tool_error(message: impl Into<String>) -> CallToolResult {
    CallToolResult::error(vec![ContentBlock::text(message.into())])
}

fn caller_authorization(parts: &http::request::Parts) -> Result<String, &'static str> {
    let token = bearer_token(&parts.headers).ok_or("execute requires the caller's bearer token")?;
    if !token.starts_with(MCP_ACCESS_TOKEN_PREFIX) {
        return Err("execute requires the caller's bearer token");
    }
    Ok(format!("Bearer {token}"))
}

pub fn router<S>(auth: AuthState, api: axum::Router) -> axum::Router<S>
where
    S: Clone + Send + Sync + 'static,
{
    let service = StreamableHttpService::new(
        move || Ok(BlogMcp::new(api.clone())),
        Arc::new(LocalSessionManager::default()),
        StreamableHttpServerConfig::default()
            .with_legacy_session_mode(false)
            .with_json_response(true),
    );
    // `from_fn` cannot see the router state, so the bearer check keeps its own copy.
    axum::Router::<S>::new()
        .nest_service("/mcp", service)
        .layer(axum::middleware::from_fn_with_state(
            auth,
            require_mcp_access,
        ))
}

async fn require_mcp_access(
    State(auth): State<AuthState>,
    request: Request,
    next: Next,
) -> Response {
    if let Err(error) = authenticate(&auth, request.headers()).await {
        return unauthorized(auth.public_base_url(), error);
    }
    next.run(request).await
}

async fn authenticate(auth: &AuthState, headers: &axum::http::HeaderMap) -> Result<(), AppError> {
    let token = bearer_token(headers).ok_or(AppError::Unauthorized)?;
    if !token.starts_with(MCP_ACCESS_TOKEN_PREFIX) {
        return Err(AppError::Unauthorized);
    }
    auth.service().authenticate_bearer(token).await.map(|_| ())
}

fn unauthorized(public_base_url: &str, error: AppError) -> Response {
    let mut response = error.into_response();
    if response.status() == StatusCode::UNAUTHORIZED
        && let Ok(value) = HeaderValue::from_str(&resource_metadata_header(public_base_url))
    {
        response.headers_mut().insert(WWW_AUTHENTICATE, value);
    }
    response
}
