use axum::Json;
use serde::Serialize;
use utoipa::ToSchema;
use utoipa_axum::{router::OpenApiRouter, routes};

#[derive(Debug, Serialize, ToSchema)]
pub struct HealthResponse {
    pub status: &'static str,
}

#[utoipa::path(
    get,
    path = "/health",
    operation_id = "healthCheck",
    tag = "system",
    responses((status = 200, body = HealthResponse))
)]
pub async fn health() -> Json<HealthResponse> {
    Json(HealthResponse { status: "ok" })
}

#[derive(Debug, Serialize, ToSchema)]
pub struct RootResponse {
    pub message: &'static str,
}

#[utoipa::path(
    get,
    path = "/",
    operation_id = "rootStatus",
    tag = "system",
    responses((status = 200, body = RootResponse))
)]
pub async fn root() -> Json<RootResponse> {
    Json(RootResponse {
        message: "Blog Agent API",
    })
}

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
{
    OpenApiRouter::new()
        .routes(routes!(root))
        .routes(routes!(health))
}
