use axum::{
    Json,
    body::Bytes,
    extract::{Path, Query, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::api::{auth::AuthenticatedAccount, response::SuccessResponse};

use super::{
    dto::{
        CrawlTriggeredResponse, CrawledContentResponse, DataSourceContentResponse,
        DataSourceCreateRequest, DataSourceDiscoveryRecommendationRequest,
        DataSourceRecommendationRequest, DataSourceRecommendationsResponse, DataSourceResponse,
        DataSourceUpdateRequest, PaginationQuery, SuccessFlag,
    },
    error::DataSourceApiError,
    state::DataSourceState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, DataSourceApiError>;

#[utoipa::path(
    get,
    path = "/data-sources",
    params(PaginationQuery),
    responses(
        (status = 200, body = SuccessResponse<Vec<DataSourceResponse>>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "listDataSources"
)]
pub async fn list_data_sources(
    authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Query(_query): Query<PaginationQuery>,
) -> ApiResult<Vec<DataSourceResponse>> {
    let account_id = authenticated.into_inner();
    let organization_id = state.organization_id(account_id).await;
    let values = match organization_id {
        Some(id) => state.service().list(id).await?,
        None => {
            state
                .service()
                .list_by_user_id(account_id.into_inner())
                .await?
        }
    };
    Ok(Json(SuccessResponse::new(
        values.into_iter().map(Into::into).collect(),
    )))
}

#[utoipa::path(
    post,
    path = "/data-sources",
    request_body = DataSourceCreateRequest,
    responses(
        (status = 201, body = SuccessResponse<DataSourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 409, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "createDataSource"
)]
pub async fn create_data_source(
    authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    body: Result<Json<DataSourceCreateRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<DataSourceResponse>>), DataSourceApiError> {
    let Json(request) = body.map_err(invalid_request_body)?;
    let account_id = authenticated.into_inner();
    let organization_id = state.organization_id(account_id).await;
    let user_id = organization_id.is_none().then(|| account_id.into_inner());
    let value = state
        .service()
        .create(
            organization_id,
            user_id,
            request.validate().map_err(DataSourceApiError::validation)?,
        )
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(value.into())),
    ))
}

#[utoipa::path(
    post,
    path = "/data-sources/recommendations",
    request_body = DataSourceRecommendationRequest,
    responses(
        (status = 200, body = SuccessResponse<DataSourceRecommendationsResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "recommendDataSources"
)]
pub async fn recommend_data_sources(
    authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    body: Result<Json<DataSourceRecommendationRequest>, JsonRejection>,
) -> ApiResult<DataSourceRecommendationsResponse> {
    let Json(request) = body.map_err(invalid_request_body)?;
    let account_id = authenticated.into_inner();
    let organization_id = state.organization_id(account_id).await;
    let user_id = organization_id.is_none().then(|| account_id.into_inner());
    let value = state
        .recommendations()
        .recommend(
            organization_id,
            user_id,
            request.validate().map_err(DataSourceApiError::validation)?,
        )
        .await?;
    Ok(Json(SuccessResponse::new(value.into())))
}

#[utoipa::path(
    post,
    path = "/data-sources/recommendations/discovery",
    request_body = Option<DataSourceDiscoveryRecommendationRequest>,
    responses(
        (status = 200, body = SuccessResponse<DataSourceRecommendationsResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "discoverDataSources"
)]
pub async fn discover_data_sources(
    authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    body: Bytes,
) -> ApiResult<DataSourceRecommendationsResponse> {
    let request = if body.is_empty() {
        DataSourceDiscoveryRecommendationRequest::default()
    } else {
        serde_json::from_slice(&body)
            .map_err(|_| crate::error::AppError::InvalidInput("Invalid request body".to_owned()))?
    };
    let account_id = authenticated.into_inner();
    let organization_id = state.organization_id(account_id).await;
    let user_id = organization_id.is_none().then(|| account_id.into_inner());
    let value = state
        .recommendations()
        .recommend_from_existing_sources(organization_id, user_id, request.into())
        .await?;
    Ok(Json(SuccessResponse::new(value.into())))
}

#[utoipa::path(
    get,
    path = "/data-sources/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<DataSourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "getDataSource"
)]
pub async fn get_data_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Path(id): Path<String>,
) -> ApiResult<DataSourceResponse> {
    let id = parse_data_source_id(&id)?;
    Ok(Json(SuccessResponse::new(
        state.service().get_by_id(id).await?.into(),
    )))
}

#[utoipa::path(
    put,
    path = "/data-sources/{id}",
    params(("id" = Uuid, Path)),
    request_body = DataSourceUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<DataSourceResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 409, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "updateDataSource"
)]
pub async fn update_data_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Path(id): Path<String>,
    body: Result<Json<DataSourceUpdateRequest>, JsonRejection>,
) -> ApiResult<DataSourceResponse> {
    let id = parse_data_source_id(&id)?;
    let Json(request) = body.map_err(invalid_request_body)?;
    Ok(Json(SuccessResponse::new(
        state.service().update(id, request.into()).await?.into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/data-sources/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "deleteDataSource"
)]
pub async fn delete_data_source(
    _authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    let id = parse_data_source_id(&id)?;
    state.service().delete(id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/data-sources/{id}/crawl",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<CrawlTriggeredResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "triggerDataSourceCrawl"
)]
pub async fn trigger_crawl(
    _authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Path(id): Path<String>,
) -> ApiResult<CrawlTriggeredResponse> {
    let id = parse_data_source_id(&id)?;
    state.service().trigger_crawl(id).await?;
    Ok(Json(SuccessResponse::new(CrawlTriggeredResponse {
        success: true,
        message: "Crawl triggered successfully",
    })))
}

#[utoipa::path(
    get,
    path = "/data-sources/{id}/content",
    params(("id" = Uuid, Path), PaginationQuery),
    responses(
        (status = 200, body = SuccessResponse<DataSourceContentResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "data-sources",
    operation_id = "getDataSourceContent"
)]
pub async fn get_data_source_content(
    _authenticated: AuthenticatedAccount,
    State(state): State<DataSourceState>,
    Path(id): Path<String>,
    Query(query): Query<PaginationQuery>,
) -> ApiResult<DataSourceContentResponse> {
    let id = parse_data_source_id(&id)?;
    let page = query.page.unwrap_or(1);
    let limit = query.limit.unwrap_or(20);
    let (contents, total) = state.service().get_content(id, page, limit).await?;
    Ok(Json(SuccessResponse::new(DataSourceContentResponse {
        contents: contents
            .into_iter()
            .map(CrawledContentResponse::from)
            .collect(),
        total,
        page,
        limit,
    })))
}

fn parse_data_source_id(value: &str) -> Result<Uuid, DataSourceApiError> {
    Uuid::parse_str(value).map_err(|_| {
        crate::error::AppError::InvalidInput("Invalid data source ID".to_owned()).into()
    })
}

fn invalid_request_body(_: JsonRejection) -> DataSourceApiError {
    crate::error::AppError::InvalidInput("Invalid request body".to_owned()).into()
}
