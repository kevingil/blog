use axum::{
    Json,
    extract::{Path, Query, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::api::{auth::AuthenticatedAccount, response::SuccessResponse};

use super::{
    dto::{
        CreateSourceRequest, ScrapeSourceRequest, SearchSourcesResponse, SourceListQuery,
        SourceListResponse, SourceResponse, SourceSearchQuery, SourcesResponse, SuccessFlag,
        UpdateSourceRequest,
    },
    error::SourceApiError,
    state::SourceState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, SourceApiError>;

#[utoipa::path(
    get,
    path = "/dashboard/sources",
    params(SourceListQuery),
    responses(
        (status = 200, body = SuccessResponse<SourceListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "listAllSources"
)]
pub async fn list_all_sources(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Query(query): Query<SourceListQuery>,
) -> ApiResult<SourceListResponse> {
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .list(query.page.unwrap_or(1), query.limit.unwrap_or(20))
            .await?
            .into(),
    )))
}

#[utoipa::path(
    post,
    path = "/sources",
    request_body = CreateSourceRequest,
    responses(
        (status = 201, body = SuccessResponse<SourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "createSource"
)]
pub async fn create_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    body: Result<Json<CreateSourceRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<SourceResponse>>), SourceApiError> {
    let Json(request) = body.map_err(invalid_request_body)?;
    let value = state
        .service()
        .create(request.validate().map_err(SourceApiError::validation)?)
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(value.into())),
    ))
}

#[utoipa::path(
    post,
    path = "/sources/scrape",
    request_body = ScrapeSourceRequest,
    responses(
        (status = 201, body = SuccessResponse<SourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "scrapeAndCreateSource"
)]
pub async fn scrape_and_create_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    body: Result<Json<ScrapeSourceRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<SourceResponse>>), SourceApiError> {
    let Json(request) = body.map_err(invalid_request_body)?;
    request.validate().map_err(SourceApiError::validation)?;
    let value = state
        .service()
        .scrape_and_create(request.article_id, &request.url)
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(value.into())),
    ))
}

#[utoipa::path(
    get,
    path = "/sources/article/{articleId}",
    params(("articleId" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SourcesResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "getArticleSources"
)]
pub async fn get_article_sources(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Path(article_id): Path<String>,
) -> ApiResult<SourcesResponse> {
    let article_id = parse_article_id(&article_id)?;
    Ok(Json(SuccessResponse::new(SourcesResponse {
        sources: state
            .service()
            .get_by_article_id(article_id)
            .await?
            .into_iter()
            .map(Into::into)
            .collect(),
    })))
}

#[utoipa::path(
    get,
    path = "/sources/article/{articleId}/search",
    params(("articleId" = Uuid, Path), SourceSearchQuery),
    responses(
        (status = 200, body = SuccessResponse<SearchSourcesResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "searchSimilarSources"
)]
pub async fn search_similar_sources(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Path(article_id): Path<String>,
    Query(query): Query<SourceSearchQuery>,
) -> ApiResult<SearchSourcesResponse> {
    let article_id = parse_article_id(&article_id)?;
    if query.q.is_empty() {
        return Err(crate::error::AppError::InvalidInput(
            "Query parameter 'q' is required".to_owned(),
        )
        .into());
    }
    let limit = query.limit.unwrap_or(5).min(20);
    let sources = state
        .service()
        .search_similar(article_id, &query.q, limit)
        .await?
        .into_iter()
        .map(Into::into)
        .collect();
    Ok(Json(SuccessResponse::new(SearchSourcesResponse {
        sources,
        query: query.q,
    })))
}

#[utoipa::path(
    get,
    path = "/sources/{sourceId}",
    params(("sourceId" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "getSource"
)]
pub async fn get_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Path(source_id): Path<String>,
) -> ApiResult<SourceResponse> {
    let source_id = parse_source_id(&source_id)?;
    Ok(Json(SuccessResponse::new(
        state.service().get_by_id(source_id).await?.into(),
    )))
}

#[utoipa::path(
    put,
    path = "/sources/{sourceId}",
    params(("sourceId" = Uuid, Path)),
    request_body = UpdateSourceRequest,
    responses(
        (status = 200, body = SuccessResponse<SourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "updateSource"
)]
pub async fn update_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Path(source_id): Path<String>,
    body: Result<Json<UpdateSourceRequest>, JsonRejection>,
) -> ApiResult<SourceResponse> {
    let source_id = parse_source_id(&source_id)?;
    let Json(request) = body.map_err(invalid_request_body)?;
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .update(source_id, request.into())
            .await?
            .into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/sources/{sourceId}",
    params(("sourceId" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "sources",
    operation_id = "deleteSource"
)]
pub async fn delete_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<SourceState>,
    Path(source_id): Path<String>,
) -> ApiResult<SuccessFlag> {
    let source_id = parse_source_id(&source_id)?;
    state.service().delete(source_id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

fn parse_article_id(value: &str) -> Result<Uuid, SourceApiError> {
    Uuid::parse_str(value)
        .map_err(|_| crate::error::AppError::InvalidInput("Invalid article ID".to_owned()).into())
}

fn parse_source_id(value: &str) -> Result<Uuid, SourceApiError> {
    Uuid::parse_str(value)
        .map_err(|_| crate::error::AppError::InvalidInput("Invalid source ID".to_owned()).into())
}

fn invalid_request_body(_: JsonRejection) -> SourceApiError {
    crate::error::AppError::InvalidInput("Invalid request body".to_owned()).into()
}
