mod catalog;

use std::sync::Arc;

use axum::{
    extract::{Request, State},
    http::{HeaderValue, StatusCode, header::WWW_AUTHENTICATE},
    middleware::Next,
    response::{IntoResponse, Response},
};
use rmcp::{
    ErrorData as McpError, ServerHandler,
    handler::server::wrapper::Parameters,
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
const SERVER_INSTRUCTIONS: &str = "Blog MCP server for the blog's own HTTP API. Use the search tool to discover operations on articles, pages, projects, sources, and storage before the execute tool exists. search returns the method, path, operation id, parameters, and request and response schemas. It does not call the API.";
const DEFAULT_SEARCH_LIMIT: u32 = 10;
const MAX_SEARCH_LIMIT: u32 = 25;
/// Static tokens and OAuth access tokens issued for this server share this prefix.
const MCP_ACCESS_TOKEN_PREFIX: &str = "mcp_";

#[derive(Debug, Deserialize, schemars::JsonSchema)]
struct SearchArgs {
    /// Words matched against method, path, operation id, summary, and parameters.
    query: String,
    /// Maximum operations to return. Defaults to 10 and cannot exceed 25.
    #[serde(default)]
    limit: Option<u32>,
}

#[derive(Clone)]
struct BlogMcp;

impl BlogMcp {
    fn new() -> Self {
        Self
    }
}

#[tool_router]
impl BlogMcp {
    #[tool(
        description = "Discover blog API operations for the upcoming execute tool. Pass a query such as \"draft\" or \"upload\". The result lists matching articles, pages, projects, sources, and storage operations with method, path, operationId, parameters, and request or response schemas. This does not call the API."
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

pub fn router<S>(auth: AuthState) -> axum::Router<S>
where
    S: Clone + Send + Sync + 'static,
{
    let service = StreamableHttpService::new(
        || Ok(BlogMcp::new()),
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
