use axum::{
    Json,
    extract::{Path, Query, State, rejection::JsonRejection},
    http::StatusCode,
};
use uuid::Uuid;

use crate::api::response::SuccessResponse;

use crate::error::AppError;

use super::{
    dto::{
        GithubImportRequest, GithubImportResponse, ProjectCreateRequest, ProjectDetailResponse,
        ProjectListQuery, ProjectListResponse, ProjectResponse, ProjectUpdateRequest, SuccessFlag,
    },
    error::{ProjectApiError, ProjectAuthenticated},
    state::ProjectState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, ProjectApiError>;

fn project_id(value: &str) -> Result<Uuid, ProjectApiError> {
    Uuid::parse_str(value)
        .map_err(|_| crate::error::AppError::InvalidInput("Invalid project ID".to_owned()).into())
}

#[utoipa::path(
    get,
    path = "/projects",
    params(ProjectListQuery),
    responses(
        (status = 200, body = SuccessResponse<ProjectListResponse>),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    tag = "projects",
    operation_id = "listProjects"
)]
pub async fn list_projects(
    State(state): State<ProjectState>,
    Query(query): Query<ProjectListQuery>,
) -> ApiResult<ProjectListResponse> {
    let (page, per_page) = query.values();
    Ok(Json(SuccessResponse::new(
        state.service().list(page, per_page).await?.into(),
    )))
}

#[utoipa::path(
    get,
    path = "/projects/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<ProjectDetailResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    tag = "projects",
    operation_id = "getProject"
)]
pub async fn get_project(
    State(state): State<ProjectState>,
    Path(id): Path<String>,
) -> ApiResult<ProjectDetailResponse> {
    Ok(Json(SuccessResponse::new(
        state.service().get_detail(project_id(&id)?).await?.into(),
    )))
}

#[utoipa::path(
    post,
    path = "/projects",
    request_body = ProjectCreateRequest,
    responses(
        (status = 201, body = SuccessResponse<ProjectResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "projects",
    operation_id = "createProject"
)]
pub async fn create_project(
    _authenticated: ProjectAuthenticated,
    State(state): State<ProjectState>,
    body: Result<Json<ProjectCreateRequest>, JsonRejection>,
) -> Result<(StatusCode, Json<SuccessResponse<ProjectResponse>>), ProjectApiError> {
    let Json(request) = body.map_err(|_| ProjectApiError::invalid_body())?;
    let project = state.service().create(request.validate()?).await?;
    Ok((
        StatusCode::CREATED,
        Json(SuccessResponse::new(project.into())),
    ))
}

#[utoipa::path(
    put,
    path = "/projects/{id}",
    params(("id" = Uuid, Path)),
    request_body = ProjectUpdateRequest,
    responses(
        (status = 200, body = SuccessResponse<ProjectResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "projects",
    operation_id = "updateProject"
)]
pub async fn update_project(
    _authenticated: ProjectAuthenticated,
    State(state): State<ProjectState>,
    Path(id): Path<String>,
    body: Result<Json<ProjectUpdateRequest>, JsonRejection>,
) -> ApiResult<ProjectResponse> {
    let id = project_id(&id)?;
    let Json(request) = body.map_err(|_| ProjectApiError::invalid_body())?;
    Ok(Json(SuccessResponse::new(
        state
            .service()
            .update(id, request.validate()?)
            .await?
            .into(),
    )))
}

#[utoipa::path(
    delete,
    path = "/projects/{id}",
    params(("id" = Uuid, Path)),
    responses(
        (status = 200, body = SuccessResponse<SuccessFlag>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "projects",
    operation_id = "deleteProject"
)]
pub async fn delete_project(
    _authenticated: ProjectAuthenticated,
    State(state): State<ProjectState>,
    Path(id): Path<String>,
) -> ApiResult<SuccessFlag> {
    state.service().delete(project_id(&id)?).await?;
    Ok(Json(SuccessResponse::new(SuccessFlag { success: true })))
}

#[utoipa::path(
    post,
    path = "/projects/github-import",
    request_body = GithubImportRequest,
    responses(
        (status = 200, body = SuccessResponse<GithubImportResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 502, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "projects",
    operation_id = "importGithubProject"
)]
pub async fn import_github_project(
    _authenticated: ProjectAuthenticated,
    State(state): State<ProjectState>,
    body: Result<Json<GithubImportRequest>, JsonRejection>,
) -> ApiResult<GithubImportResponse> {
    let Json(request) = body.map_err(|_| ProjectApiError::invalid_body())?;
    let github = state
        .github()
        .ok_or(AppError::internal("GitHub import is not configured"))?;
    let draft = github
        .import_repository(request.url.trim())
        .await
        .map_err(import_error)?;
    Ok(Json(SuccessResponse::new(draft.into())))
}

fn import_error(error: AppError) -> ProjectApiError {
    match error {
        AppError::NotFound => ProjectApiError::not_found("GitHub repository not found"),
        AppError::External(cause) => {
            ProjectApiError::bad_gateway("Could not reach GitHub. Try again in a moment.")
                .with_cause(format!("external service operation failed: {cause}"))
        }
        other => other.into(),
    }
}
