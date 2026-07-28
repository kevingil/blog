use axum::{
    Json,
    extract::{Path, Query, State},
};
use uuid::Uuid;

use crate::{
    api::{
        auth::{AuthState, AuthenticatedAccount},
        response::SuccessResponse,
    },
    core::{
        auth::AccountId,
        taskrun::{TaskRun, TaskRunFilter},
    },
    error::AppError,
};

use super::{
    dto::{
        TaskRunDetailResponse, TaskRunEventResponse, TaskRunEventsResponse, TaskRunListQuery,
        TaskRunListResponse, TaskRunResponse, step_keys,
    },
    state::TaskRunState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

#[utoipa::path(
    get,
    path = "/task-runs",
    params(TaskRunListQuery),
    responses(
        (status = 200, body = SuccessResponse<TaskRunListResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "task-runs",
    operation_id = "listTaskRuns"
)]
pub async fn list_task_runs(
    authenticated: AuthenticatedAccount,
    State(task_runs): State<TaskRunState>,
    State(auth): State<AuthState>,
    Query(query): Query<TaskRunListQuery>,
) -> ApiResult<TaskRunListResponse> {
    let account_id = authenticated.into_inner();
    let organization_id = organization_id(&auth, account_id).await;
    let runs = task_runs
        .service()?
        .list_runs(TaskRunFilter {
            organization_id,
            user_id: organization_id.is_none().then_some(account_id.into_inner()),
            task_name: query.task_name.unwrap_or_default(),
            status: query.status.unwrap_or_default(),
            kind: query.kind.unwrap_or_default(),
            limit: query
                .limit
                .as_deref()
                .and_then(|value| value.parse::<i64>().ok())
                .unwrap_or(50),
        })
        .await?
        .into_iter()
        .map(Into::into)
        .collect();
    Ok(Json(SuccessResponse::new(TaskRunListResponse { runs })))
}

#[utoipa::path(
    get,
    path = "/task-runs/{id}",
    params(("id" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<TaskRunDetailResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "task-runs",
    operation_id = "getTaskRun"
)]
pub async fn get_task_run(
    authenticated: AuthenticatedAccount,
    State(task_runs): State<TaskRunState>,
    State(auth): State<AuthState>,
    Path(id): Path<String>,
) -> ApiResult<TaskRunDetailResponse> {
    let run_id = parse_run_id(&id)?;
    let service = task_runs.service()?;
    let run = service.get_run(run_id).await?;
    ensure_run_access(&auth, authenticated.into_inner(), &run).await?;
    let steps = service.list_steps_by_run_id(run_id).await?;
    let step_keys = step_keys(&steps);
    let events = service
        .list_events_by_run_id(run_id)
        .await?
        .into_iter()
        .map(|event| TaskRunEventResponse::new(event, &step_keys))
        .collect();
    Ok(Json(SuccessResponse::new(TaskRunDetailResponse {
        run: TaskRunResponse::from(run),
        steps: steps.into_iter().map(Into::into).collect(),
        events,
    })))
}

#[utoipa::path(
    get,
    path = "/task-runs/{id}/events",
    params(("id" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<TaskRunEventsResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "task-runs",
    operation_id = "listTaskRunEvents"
)]
pub async fn list_task_run_events(
    authenticated: AuthenticatedAccount,
    State(task_runs): State<TaskRunState>,
    State(auth): State<AuthState>,
    Path(id): Path<String>,
) -> ApiResult<TaskRunEventsResponse> {
    let run_id = parse_run_id(&id)?;
    let service = task_runs.service()?;
    let run = service.get_run(run_id).await?;
    ensure_run_access(&auth, authenticated.into_inner(), &run).await?;
    let steps = service.list_steps_by_run_id(run_id).await?;
    let step_keys = step_keys(&steps);
    let events = service
        .list_events_by_run_id(run_id)
        .await?
        .into_iter()
        .map(|event| TaskRunEventResponse::new(event, &step_keys))
        .collect();
    Ok(Json(SuccessResponse::new(TaskRunEventsResponse { events })))
}

fn parse_run_id(id: &str) -> Result<Uuid, AppError> {
    Uuid::parse_str(id).map_err(|_| AppError::InvalidInput("Invalid task run ID".to_owned()))
}

async fn organization_id(auth: &AuthState, account_id: AccountId) -> Option<Uuid> {
    auth.service()
        .get_account(account_id)
        .await
        .ok()
        .and_then(|account| account.organization_id)
}

async fn ensure_run_access(
    auth: &AuthState,
    account_id: AccountId,
    run: &TaskRun,
) -> Result<(), AppError> {
    if let Some(organization_id) = organization_id(auth, account_id).await {
        if run.organization_id == Some(organization_id) {
            return Ok(());
        }
        return Err(AppError::NotFound);
    }
    if run.user_id == Some(account_id.into_inner()) {
        Ok(())
    } else {
        Err(AppError::NotFound)
    }
}
