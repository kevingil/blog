use axum::{
    Router,
    http::{
        HeaderName, HeaderValue, Method, StatusCode,
        header::{ACCEPT, AUTHORIZATION, CONTENT_TYPE},
    },
    response::IntoResponse,
};
use tower::ServiceBuilder;
use tower_http::{
    cors::CorsLayer,
    limit::RequestBodyLimitLayer,
    request_id::{MakeRequestUuid, PropagateRequestIdLayer, SetRequestIdLayer},
    set_header::SetResponseHeaderLayer,
    timeout::TimeoutLayer,
    trace::TraceLayer,
};
use utoipa_swagger_ui::SwaggerUi;

use crate::{
    api::{auth::AuthState, websocket::WebSocketSupervisorHandle},
    constants::{DEFAULT_REQUEST_TIMEOUT, MAX_REQUEST_BODY_BYTES},
    database::pool::PgPool,
    openapi,
};

#[derive(Clone)]
pub struct AppState {
    pub(crate) pool: PgPool,
    auth: AuthState,
    websocket: WebSocketSupervisorHandle,
}

impl AppState {
    pub fn new(pool: PgPool, auth: AuthState, websocket: WebSocketSupervisorHandle) -> Self {
        Self {
            pool,
            auth,
            websocket,
        }
    }

    pub const fn database_pool(&self) -> &PgPool {
        &self.pool
    }
}

impl axum::extract::FromRef<AppState> for AuthState {
    fn from_ref(state: &AppState) -> Self {
        state.auth.clone()
    }
}

impl axum::extract::FromRef<AppState> for WebSocketSupervisorHandle {
    fn from_ref(state: &AppState) -> Self {
        state.websocket.clone()
    }
}

async fn not_found() -> impl IntoResponse {
    (StatusCode::NOT_FOUND, "Not Found")
}

pub fn router(state: AppState, cors_origins: &[String]) -> anyhow::Result<Router> {
    let (api, document) = openapi::split_for_parts();
    let allowed_origins = cors_origins
        .iter()
        .map(|origin| HeaderValue::from_str(origin))
        .collect::<Result<Vec<_>, _>>()?;
    let cors = CorsLayer::new()
        .allow_origin(allowed_origins)
        .allow_methods([
            Method::GET,
            Method::POST,
            Method::PUT,
            Method::DELETE,
            Method::OPTIONS,
            Method::PATCH,
        ])
        .allow_headers([ACCEPT, AUTHORIZATION, CONTENT_TYPE])
        .allow_credentials(true)
        .max_age(std::time::Duration::from_secs(300));

    let request_id = HeaderName::from_static("x-request-id");
    let middleware = ServiceBuilder::new()
        .layer(SetRequestIdLayer::new(request_id.clone(), MakeRequestUuid))
        .layer(PropagateRequestIdLayer::new(request_id))
        .layer(TraceLayer::new_for_http())
        .layer(TimeoutLayer::with_status_code(
            StatusCode::REQUEST_TIMEOUT,
            DEFAULT_REQUEST_TIMEOUT,
        ))
        .layer(RequestBodyLimitLayer::new(MAX_REQUEST_BODY_BYTES))
        .layer(cors)
        .layer(SetResponseHeaderLayer::if_not_present(
            HeaderName::from_static("x-content-type-options"),
            HeaderValue::from_static("nosniff"),
        ))
        .layer(SetResponseHeaderLayer::if_not_present(
            HeaderName::from_static("x-frame-options"),
            HeaderValue::from_static("DENY"),
        ))
        .layer(SetResponseHeaderLayer::if_not_present(
            HeaderName::from_static("x-xss-protection"),
            HeaderValue::from_static("1; mode=block"),
        ));

    Ok(Router::new()
        .merge(api)
        .merge(SwaggerUi::new("/swagger").url("/api/openapi.json", document))
        .fallback(not_found)
        .layer(middleware)
        .with_state(state))
}
