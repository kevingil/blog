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
        agent::AgentState, article::ArticleState, auth::AuthState, datasource::DataSourceState,
        image::ImageState, insight::InsightState, organization::OrganizationState, page::PageState,
        profile::ProfileState, project::ProjectState, source::SourceState, storage::StorageState,
        taskrun::TaskRunState, websocket::WebSocketSupervisorHandle, worker::WorkerState,
    },
    constants::{DEFAULT_REQUEST_TIMEOUT, MAX_REQUEST_BODY_BYTES},
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
