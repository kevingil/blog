use axum::{
    Json,
    extract::{Path, Query, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::api::response::SuccessResponse;

use super::{
    dto::{
        PageCreateRequest, PageListQuery, PageListResponse, PageResponse, PageUpdateRequest,
        SuccessFlag,
    },
    error::{PageApiError, PageAuthenticated},
    state::PageState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, PageApiError>;

fn page_id(value: &str) -> Result<Uuid, PageApiError> {
    Uuid::parse_str(value)
        .map_err(|_| crate::error::AppError::InvalidInput("Invalid page ID".to_owned()).into())
}

#[utoipa::path(
    get,
    path = "/pages/{slug}",
    params(("slug" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<PageResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    tag = "pages",
    operation_id = "getPageBySlug"
)]
pub async fn get_page_by_slug(
    State(state): State<PageState>,
    Path(slug): Path<String>,
) -> ApiResult<PageResponse> {
    if slug.is_empty() {
        return Err(
            crate::error::AppError::InvalidInput("Page slug is required".to_owned()).into(),
        );
    }
    Ok(Json(SuccessResponse::new(
        state.service().get_by_slug(&slug).await?.into(),
    )))
}

#[utoipa::path(
    get,
    path = "/dashboard/pages",
    params(PageListQuery),
    responses(
        (status = 200, body = SuccessResponse<PageListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "pages",
    operation_id = "listPages"
)]
pub async fn list_pages(
    _authenticated: PageAuthenticated,
    State(state): State<PageState>,
    Query(query): Query<PageListQuery>,
) -> ApiResult<PageListResponse> {
    let (page, per_page, is_published) = query.values();
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .list(page, per_page, is_published)
            .await?
            .into(),
    )))
}

#[utoipa::path(
    get,
    path = "/dashboard/pages/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<PageResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "pages",
    operation_id = "getPageById"
)]
pub async fn get_page_by_id(
    _authenticated: PageAuthenticated,
    State(state): State<PageState>,
    Path(id): Path<String>,
) -> ApiResult<PageResponse> {
    Ok(Json(SuccessResponse::new(
        state.service().get_by_id(page_id(&id)?).await?.into(),
    )))
}

#[utoipa::path(
    post,
    path = "/dashboard/pages",
    request_body = PageCreateRequest,
    responses(
        (status = 201, body = SuccessResponse<PageResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 409, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "pages",
    operation_id = "createPage"
)]
pub async fn create_page(
    _authenticated: PageAuthenticated,
    State(state): State<PageState>,
    body: Result<Json<PageCreateRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<PageResponse>>), PageApiError> {
    let Json(request) = body.map_err(|_| PageApiError::invalid_body())?;
    let page = state.service().create(request.validate()?).await?;
    Ok((StatusCode::CREATED, Json(SuccessResponse::new(page.into()))))
}

#[utoipa::path(
    put,
    path = "/dashboard/pages/{id}",
    params(("id" = Uuid, Path)),
    request_body = PageUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<PageResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "pages",
    operation_id = "updatePage"
)]
pub async fn update_page(
    _authenticated: PageAuthenticated,
    State(state): State<PageState>,
    Path(id): Path<String>,
    body: Result<Json<PageUpdateRequest>, JsonRejection>,
) -> ApiResult<PageResponse> {
    let id = page_id(&id)?;
    let Json(request) = body.map_err(|_| PageApiError::invalid_body())?;
    Ok(Json(SuccessResponse::new(
        state.service().update(id, request.into()).await?.into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/dashboard/pages/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "pages",
    operation_id = "deletePage"
)]
pub async fn delete_page(
    _authenticated: PageAuthenticated,
    State(state): State<PageState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    state.service().delete(page_id(&id)?).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}
