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
    api::{
        agent::AgentState,
        article::ArticleState,
        auth::{AuthState, attach_resource_metadata},
        datasource::DataSourceState,
        image::ImageState,
        insight::InsightState,
        organization::OrganizationState,
        page::PageState,
        profile::ProfileState,
        project::ProjectState,
        source::SourceState,
        storage::StorageState,
        taskrun::TaskRunState,
        websocket::WebSocketSupervisorHandle,
        worker::WorkerState,
    },
    constants::{DEFAULT_REQUEST_TIMEOUT, MAX_REQUEST_BODY_BYTES, MCP_REQUEST_TIMEOUT},
    database::pool::PgPool,
    openapi,
};

#[derive(Clone)]
pub struct AppState {
    pub(crate) pool: PgPool,
    auth: AuthState,
    agent: AgentState,
    article: ArticleState,
    datasource: DataSourceState,
    image: ImageState,
    insight: InsightState,
    organization: OrganizationState,
    page: PageState,
    profile: ProfileState,
    project: ProjectState,
    source: SourceState,
    storage: StorageState,
    taskrun: TaskRunState,
    websocket: WebSocketSupervisorHandle,
    worker: WorkerState,
}

pub struct AppDependencies {
    pub agent: AgentState,
    pub article: ArticleState,
    pub datasource: DataSourceState,
    pub image: ImageState,
    pub insight: InsightState,
    pub organization: OrganizationState,
    pub page: PageState,
    pub profile: ProfileState,
    pub project: ProjectState,
    pub source: SourceState,
    pub storage: StorageState,
    pub taskrun: TaskRunState,
    pub worker: WorkerState,
}

impl AppState {
    pub fn new(
        pool: PgPool,
        auth: AuthState,
        websocket: WebSocketSupervisorHandle,
        dependencies: AppDependencies,
    ) -> Self {
        Self {
            pool,
            auth,
            agent: dependencies.agent,
            article: dependencies.article,
            datasource: dependencies.datasource,
            image: dependencies.image,
            insight: dependencies.insight,
            organization: dependencies.organization,
            page: dependencies.page,
            profile: dependencies.profile,
            project: dependencies.project,
            source: dependencies.source,
            storage: dependencies.storage,
            taskrun: dependencies.taskrun,
            websocket,
            worker: dependencies.worker,
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

macro_rules! from_app_state {
    ($state:ty, $field:ident) => {
        impl axum::extract::FromRef<AppState> for $state {
            fn from_ref(state: &AppState) -> Self {
                state.$field.clone()
            }
        }
    };
}

from_app_state!(AgentState, agent);
impl axum::extract::FromRef<AppState> for ArticleState {
    fn from_ref(state: &AppState) -> Self {
        state.article.clone()
    }
}
from_app_state!(DataSourceState, datasource);
from_app_state!(ImageState, image);
from_app_state!(InsightState, insight);
from_app_state!(OrganizationState, organization);
from_app_state!(PageState, page);
from_app_state!(ProfileState, profile);
from_app_state!(ProjectState, project);
from_app_state!(SourceState, source);
from_app_state!(StorageState, storage);
from_app_state!(TaskRunState, taskrun);
from_app_state!(WorkerState, worker);

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

    let auth = state.auth.clone();
    let api = with_http_middleware(
        Router::new()
            .merge(api)
            .merge(SwaggerUi::new("/swagger").url("/api/openapi.json", document))
            .fallback(not_found),
        DEFAULT_REQUEST_TIMEOUT,
        cors.clone(),
        auth.clone(),
    )
    .with_state(state.clone());
    let mcp = with_http_middleware(
        crate::api::mcp::router(auth.clone(), api.clone()),
        MCP_REQUEST_TIMEOUT,
        cors,
        auth,
    )
    .with_state(state);
    Ok(Router::new().merge(api).merge(mcp))
}

fn timeout_layer(request_timeout: std::time::Duration) -> TimeoutLayer {
    TimeoutLayer::with_status_code(StatusCode::REQUEST_TIMEOUT, request_timeout)
}

fn with_http_middleware<S>(
    router: Router<S>,
    request_timeout: std::time::Duration,
    cors: CorsLayer,
    auth: AuthState,
) -> Router<S>
where
    S: Clone + Send + Sync + 'static,
{
    let request_id = HeaderName::from_static("x-request-id");
    let middleware = ServiceBuilder::new()
        .layer(SetRequestIdLayer::new(request_id.clone(), MakeRequestUuid))
        .layer(PropagateRequestIdLayer::new(request_id))
        .layer(
            TraceLayer::new_for_http().make_span_with(|request: &axum::extract::Request| {
                let request_id = request
                    .headers()
                    .get("x-request-id")
                    .and_then(|value| value.to_str().ok())
                    .unwrap_or("");
                tracing::info_span!(
                    "request",
                    method = %request.method(),
                    uri = %request.uri(),
                    version = ?request.version(),
                    request_id = %request_id,
                )
            }),
        )
        .layer(timeout_layer(request_timeout))
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

    router
        .layer(axum::middleware::from_fn_with_state(
            auth,
            attach_resource_metadata,
        ))
        .layer(middleware)
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use axum::{
        Router,
        body::Body,
        http::{Request, StatusCode},
        routing::{get, post},
    };
    use tower::ServiceExt;

    use super::*;

    async fn slow_ok() -> &'static str {
        tokio::time::sleep(Duration::from_millis(80)).await;
        "ok"
    }

    #[tokio::test]
    async fn mcp_timeout_is_split_from_the_api_timeout() -> Result<(), Box<dyn std::error::Error>> {
        assert_eq!(DEFAULT_REQUEST_TIMEOUT, Duration::from_secs(30));
        assert_eq!(MCP_REQUEST_TIMEOUT, Duration::from_secs(60));

        let app = Router::new()
            .merge(
                Router::new()
                    .route("/blog/articles", get(slow_ok))
                    .layer(timeout_layer(Duration::from_millis(20))),
            )
            .merge(
                Router::new()
                    .route("/mcp", post(slow_ok))
                    .layer(timeout_layer(Duration::from_millis(200))),
            );

        let api = app
            .clone()
            .oneshot(
                Request::builder()
                    .method("GET")
                    .uri("/blog/articles")
                    .body(Body::empty())?,
            )
            .await?;
        assert_eq!(api.status(), StatusCode::REQUEST_TIMEOUT);

        let mcp = app
            .oneshot(
                Request::builder()
                    .method("POST")
                    .uri("/mcp")
                    .body(Body::empty())?,
            )
            .await?;
        assert_eq!(mcp.status(), StatusCode::OK);
        Ok(())
    }
}
