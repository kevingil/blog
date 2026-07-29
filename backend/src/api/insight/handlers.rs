use axum::{
    Json,
    extract::{Path, Query, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::{
    api::{auth::AuthenticatedAccount, response::SuccessResponse},
    core::insight::InsightSearchRequest,
    error::AppError,
};

use super::{
    dto::{
        CountResponse, CrawledContentResponse, InsightListQuery, InsightListResponse,
        InsightResponse, InsightTopicCreateRequest, InsightTopicResponse,
        InsightTopicUpdateRequest, InsightWithSources, InsightWithUserStatus, PinResponse,
        RecentQuery, SearchQuery, SuccessFlag,
    },
    error::InsightApiError,
    state::InsightState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, InsightApiError>;

#[utoipa::path(
    get,
    path = "/insights",
    params(InsightListQuery),
    responses(
        (status = 200, body = SuccessResponse<InsightListResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "listInsights"
)]
pub async fn list_insights(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Query(query): Query<InsightListQuery>,
) -> ApiResult<InsightListResponse> {
    let user_id = authenticated.into_inner().into_inner();
    let page = query.page.unwrap_or(1);
    let limit = query.limit.unwrap_or(20);
    let (values, total) = if let Some(topic_id) = query.topic_id {
        let topic_id = Uuid::parse_str(&topic_id)
            .map_err(|_| AppError::InvalidInput("Invalid topic ID".to_owned()))?;
        let (values, total) = state
            .service()
            .list_insights_by_topic(topic_id, page, limit)
            .await?;
        (
            values
                .into_iter()
                .map(InsightWithUserStatus::from)
                .collect(),
            total,
        )
    } else {
        let (values, total) = state
            .service()
            .list_insights_with_user_status(user_id, page, limit)
            .await?;
        (values.into_iter().map(Into::into).collect(), total)
    };
    Ok(Json(SuccessResponse::new(InsightListResponse {
        insights: values,
        total,
        page,
        limit,
    })))
}

#[utoipa::path(
    get,
    path = "/insights/search",
    params(SearchQuery),
    responses(
        (status = 200, body = SuccessResponse<Vec<InsightResponse>>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "searchInsights"
)]
pub async fn search_insights(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Query(query): Query<SearchQuery>,
) -> ApiResult<Vec<InsightResponse>> {
    if query.q.is_empty() {
        return Err(AppError::InvalidInput("Search query required".to_owned()).into());
    }
    let limit = query.limit.unwrap_or(10);
    let organization_id = state.organization_id(authenticated.into_inner()).await;
    let values = match organization_id {
        Some(id) => {
            state
                .service()
                .search_insights_by_org(id, &query.q, limit)
                .await?
        }
        None => {
            state
                .service()
                .search_insights(InsightSearchRequest {
                    query: query.q,
                    topic_id: None,
                    limit,
                    is_unread: None,
                })
                .await?
        }
    };
    Ok(Json(SuccessResponse::new(
        values.into_iter().map(Into::into).collect(),
    )))
}

#[utoipa::path(
    get,
    path = "/insights/unread-count",
    responses(
        (status = 200, body = SuccessResponse<CountResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "getUnreadInsightCount"
)]
pub async fn get_unread_count(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
) -> ApiResult<CountResponse> {
    let count = state
        .service()
        .count_unread_insights_for_user(authenticated.into_inner().into_inner())
        .await?;
    Ok(Json(SuccessResponse::new(CountResponse { count })))
}

#[utoipa::path(
    get,
    path = "/insights/topics",
    responses(
        (status = 200, body = SuccessResponse<Vec<InsightTopicResponse>>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "listInsightTopics"
)]
pub async fn list_topics(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
) -> ApiResult<Vec<InsightTopicResponse>> {
    let organization_id = state.organization_id(authenticated.into_inner()).await;
    let values = match organization_id {
        Some(id) => state.service().list_topics(id).await?,
        None => state.service().list_all_topics().await?,
    };
    Ok(Json(SuccessResponse::new(
        values.into_iter().map(Into::into).collect(),
    )))
}

#[utoipa::path(
    post,
    path = "/insights/topics",
    request_body = InsightTopicCreateRequest,
    responses(
        (status = 201, body = SuccessResponse<InsightTopicResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "createInsightTopic"
)]
pub async fn create_topic(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    body: Result<Json<InsightTopicCreateRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<InsightTopicResponse>>), InsightApiError> {
    let Json(request) = body.map_err(invalid_request_body)?;
    let organization_id = state.organization_id(authenticated.into_inner()).await;
    let value = state
        .service()
        .create_topic(
            organization_id,
            request.validate().map_err(InsightApiError::validation)?,
        )
        .await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(value.into())),
    ))
}

#[utoipa::path(
    get,
    path = "/insights/topics/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<InsightTopicResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "getInsightTopic"
)]
pub async fn get_topic(
    _authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<InsightTopicResponse> {
    let id = parse_topic_id(&id)?;
    Ok(Json(SuccessResponse::new(
        state.service().get_topic_by_id(id).await?.into(),
    )))
}

#[utoipa::path(
    put,
    path = "/insights/topics/{id}",
    params(("id" = Uuid, Path)),
    request_body = InsightTopicUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<InsightTopicResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "updateInsightTopic"
)]
pub async fn update_topic(
    _authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
    body: Result<Json<InsightTopicUpdateRequest>, JsonRejection>,
) -> ApiResult<InsightTopicResponse> {
    let id = parse_topic_id(&id)?;
    let Json(request) = body.map_err(invalid_request_body)?;
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .update_topic(id, request.into())
            .await?
            .into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/insights/topics/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "deleteInsightTopic"
)]
pub async fn delete_topic(
    _authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    let id = parse_topic_id(&id)?;
    state.service().delete_topic(id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    get,
    path = "/insights/content/search",
    params(SearchQuery),
    responses(
        (status = 200, body = SuccessResponse<Vec<CrawledContentResponse>>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "searchInsightContent"
)]
pub async fn search_crawled_content(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Query(query): Query<SearchQuery>,
) -> ApiResult<Vec<CrawledContentResponse>> {
    if query.q.is_empty() {
        return Err(AppError::InvalidInput("Search query required".to_owned()).into());
    }
    let limit = query.limit.unwrap_or(10);
    let organization_id = state.organization_id(authenticated.into_inner()).await;
    let values = match organization_id {
        Some(id) => {
            state
                .service()
                .search_crawled_content_by_org(id, &query.q, limit)
                .await?
        }
        None => {
            state
                .service()
                .search_crawled_content(&query.q, limit)
                .await?
        }
    };
    Ok(Json(SuccessResponse::new(
        values.into_iter().map(Into::into).collect(),
    )))
}

#[utoipa::path(
    get,
    path = "/insights/content/recent",
    params(RecentQuery),
    responses(
        (status = 200, body = SuccessResponse<Vec<CrawledContentResponse>>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "getRecentInsightContent"
)]
pub async fn get_recent_crawled_content(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Query(query): Query<RecentQuery>,
) -> ApiResult<Vec<CrawledContentResponse>> {
    let organization_id = state
        .organization_id(authenticated.into_inner())
        .await
        .ok_or(AppError::Unauthorized)?;
    let values = state
        .service()
        .get_recent_crawled_content(organization_id, query.limit.unwrap_or(20))
        .await?;
    Ok(Json(SuccessResponse::new(
        values.into_iter().map(Into::into).collect(),
    )))
}

#[utoipa::path(
    get,
    path = "/insights/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<InsightWithSources>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "getInsight"
)]
pub async fn get_insight(
    _authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<InsightWithSources> {
    let id = parse_insight_id(&id)?;
    Ok(Json(SuccessResponse::new(
        state.service().get_insight_with_sources(id).await?.into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/insights/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "deleteInsight"
)]
pub async fn delete_insight(
    _authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    let id = parse_insight_id(&id)?;
    state.service().delete_insight(id).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/insights/{id}/read",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "markInsightRead"
)]
pub async fn mark_insight_as_read(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    let id = parse_insight_id(&id)?;
    state
        .service()
        .mark_insight_as_read_for_user(authenticated.into_inner().into_inner(), id)
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/insights/{id}/pin",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<PinResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "insights",
    operation_id = "toggleInsightPinned"
)]
pub async fn toggle_insight_pinned(
    authenticated: AuthenticatedAccount,
    State(state): State<InsightState>,
    Path(id): Path<String>,
) -> ApiResult<PinResponse> {
    let id = parse_insight_id(&id)?;
    let is_pinned = state
        .service()
        .toggle_insight_pinned_for_user(authenticated.into_inner().into_inner(), id)
        .await?;
    Ok(Json(SuccessResponse::new(PinResponse {
        success: true,
        is_pinned,
    })))
}

fn parse_topic_id(value: &str) -> Result<Uuid, InsightApiError> {
    Uuid::parse_str(value).map_err(|_| AppError::InvalidInput("Invalid topic ID".to_owned()).into())
}

fn parse_insight_id(value: &str) -> Result<Uuid, InsightApiError> {
    Uuid::parse_str(value)
        .map_err(|_| AppError::InvalidInput("Invalid insight ID".to_owned()).into())
}

fn invalid_request_body(_: JsonRejection) -> InsightApiError {
    AppError::InvalidInput("Invalid request body".to_owned()).into()
}
