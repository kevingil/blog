use axum::{
    Json,
    extract::{Path, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::api::response::SuccessResponse;

use super::{
    dto::{
        OrganizationCreateRequest, OrganizationResponse, OrganizationUpdateRequest, SuccessFlag,
    },
    error::{OrganizationApiError, OrganizationAuthenticated},
    state::OrganizationState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, OrganizationApiError>;

fn organization_id(value: &str) -> Result<Uuid, OrganizationApiError> {
    Uuid::parse_str(value).map_err(|_| {
        crate::error::AppError::InvalidInput("Invalid organization ID".to_owned()).into()
    })
}

#[utoipa::path(
    get,
    path = "/organizations",
    responses(
        (status = 200, body = SuccessResponse<Vec<OrganizationResponse>>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "listOrganizations"
)]
pub async fn list_organizations(
    _authenticated: OrganizationAuthenticated,
    State(state): State<OrganizationState>,
) -> ApiResult<Vec<OrganizationResponse>> {
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .list()
            .await?
            .into_iter()
            .map(Into::into)
            .collect(),
    )))
}

#[utoipa::path(
    post,
    path = "/organizations",
    request_body = OrganizationCreateRequest,
    responses(
        (status = 201, body = SuccessResponse<OrganizationResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 409, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "createOrganization"
)]
pub async fn create_organization(
    _authenticated: OrganizationAuthenticated,
    State(state): State<OrganizationState>,
    body: Result<Json<OrganizationCreateRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<OrganizationResponse>>), OrganizationApiError> {
    let Json(request) = body.map_err(|_| OrganizationApiError::invalid_body())?;
    let organization = state.service().create(request.validate()?).await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(organization.into())),
    ))
}

#[utoipa::path(
    get,
    path = "/organizations/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<OrganizationResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "getOrganization"
)]
pub async fn get_organization(
    _authenticated: OrganizationAuthenticated,
    State(state): State<OrganizationState>,
    Path(id): Path<String>,
) -> ApiResult<OrganizationResponse> {
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .get_by_id(organization_id(&id)?)
            .await?
            .into(),
    )))
}

#[utoipa::path(
    put,
    path = "/organizations/{id}",
    params(("id" = Uuid, Path)),
    request_body = OrganizationUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<OrganizationResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 409, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "updateOrganization"
)]
pub async fn update_organization(
    _authenticated: OrganizationAuthenticated,
    State(state): State<OrganizationState>,
    Path(id): Path<String>,
    body: Result<Json<OrganizationUpdateRequest>, JsonRejection>,
) -> ApiResult<OrganizationResponse> {
    let id = organization_id(&id)?;
    let Json(request) = body.map_err(|_| OrganizationApiError::invalid_body())?;
    Ok(Json(SuccessResponse::new(
        state.service().update(id, request.into()).await?.into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/organizations/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "deleteOrganization"
)]
pub async fn delete_organization(
    _authenticated: OrganizationAuthenticated,
    State(state): State<OrganizationState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    state.service().delete(organization_id(&id)?).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/organizations/{id}/join",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "joinOrganization"
)]
pub async fn join_organization(
    OrganizationAuthenticated(account_id): OrganizationAuthenticated,
    State(state): State<OrganizationState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    state
        .service()
        .join_organization(account_id.into_inner(), organization_id(&id)?)
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/organizations/leave",
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "organizations",
    operation_id = "leaveOrganization"
)]
pub async fn leave_organization(
    OrganizationAuthenticated(account_id): OrganizationAuthenticated,
    State(state): State<OrganizationState>,
) -> ApiResult<SuccessFlag> {
    state
        .service()
        .leave_organization(account_id.into_inner())
        .await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}
