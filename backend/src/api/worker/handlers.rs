use axum::{
    Json,
    extract::{Path, State},
};

use crate::{
    api::{
        auth::{AuthState, AuthenticatedAccount},
        response::SuccessResponse,
    },
    core::{
        auth::AccountId,
        worker::{RunMetadata, WorkerManagerError},
    },
    error::AppError,
};

use super::{
    dto::{
        AllWorkersStatusResponse, RunWorkerResponse, RunningWorkersResponse, StopWorkerResponse,
        WorkerStatusResponse,
    },
    state::WorkerState,
};

type ApiResult<T> = Result<Json<SuccessResponse<T>>, AppError>;

#[utoipa::path(
    get,
    path = "/workers/status",
    responses(
        (status = 200, body = SuccessResponse<AllWorkersStatusResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "workers",
    operation_id = "getAllWorkerStatus"
)]
pub async fn get_all_worker_status(
    _authenticated: AuthenticatedAccount,
    State(state): State<WorkerState>,
) -> ApiResult<AllWorkersStatusResponse> {
    let workers = state
        .status()
        .snapshot()
        .into_iter()
        .map(|(_, status)| status.into())
        .collect();
    Ok(Json(SuccessResponse::new(AllWorkersStatusResponse {
        workers,
        is_running: state.manager().is_running(),
    })))
}

#[utoipa::path(
    get,
    path = "/workers/running",
    responses(
        (status = 200, body = SuccessResponse<RunningWorkersResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "workers",
    operation_id = "getRunningWorkers"
)]
pub async fn get_running_workers(
    _authenticated: AuthenticatedAccount,
    State(state): State<WorkerState>,
) -> ApiResult<RunningWorkersResponse> {
    Ok(Json(SuccessResponse::new(RunningWorkersResponse {
        workers: state.manager().running_workers(),
    })))
}

#[utoipa::path(
    get,
    path = "/workers/{name}/status",
    params(("name" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<WorkerStatusResponse>),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "workers",
    operation_id = "getWorkerStatus"
)]
pub async fn get_worker_status(
    _authenticated: AuthenticatedAccount,
    State(state): State<WorkerState>,
    Path(name): Path<String>,
) -> ApiResult<WorkerStatusResponse> {
    let status = state.status().status(&name).ok_or(AppError::NotFound)?;
    Ok(Json(SuccessResponse::new(status.into())))
}

#[utoipa::path(
    post,
    path = "/workers/{name}/run",
    params(("name" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<RunWorkerResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "workers",
    operation_id = "runWorker"
)]
pub async fn run_worker(
    authenticated: AuthenticatedAccount,
    State(state): State<WorkerState>,
    State(auth): State<AuthState>,
    Path(name): Path<String>,
) -> ApiResult<RunWorkerResponse> {
    let account_id = authenticated.into_inner();
    let organization_id = organization_id(&auth, account_id).await;
    let run_id = state
        .manager()
        .run_now(
            &name,
            RunMetadata {
                user_id: Some(account_id.into_inner()),
                organization_id,
                triggered_by_user_id: Some(account_id.into_inner()),
                parent_run_id: None,
                trigger_source: "manual".to_owned(),
            },
        )
        .await
        .map_err(map_run_error)?;
    Ok(Json(SuccessResponse::new(RunWorkerResponse {
        started: true,
        message: "Worker started successfully".to_owned(),
        task_run_id: run_id,
    })))
}

#[utoipa::path(
    post,
    path = "/workers/{name}/stop",
    params(("name" = String, Path)),
    responses(
        (status = 200, body = SuccessResponse<StopWorkerResponse>),
        (status = 400, body = crate::error::ErrorEnvelope),
        (status = 401, body = crate::error::ErrorEnvelope),
        (status = 404, body = crate::error::ErrorEnvelope),
        (status = 500, body = crate::error::ErrorEnvelope)
    ),
    security(("bearerAuth" = [])),
    tag = "workers",
    operation_id = "stopWorker"
)]
pub async fn stop_worker(
    _authenticated: AuthenticatedAccount,
    State(state): State<WorkerState>,
    Path(name): Path<String>,
) -> ApiResult<StopWorkerResponse> {
    state.manager().stop(&name).map_err(map_stop_error)?;
    Ok(Json(SuccessResponse::new(StopWorkerResponse {
        stopped: true,
        message: "Worker stopped successfully".to_owned(),
    })))
}

async fn organization_id(auth: &AuthState, account_id: AccountId) -> Option<uuid::Uuid> {
    auth.service()
        .get_account(account_id)
        .await
        .ok()
        .and_then(|account| account.organization_id)
}

fn map_run_error(error: WorkerManagerError) -> AppError {
    match error {
        WorkerManagerError::NotFound => AppError::NotFound,
        WorkerManagerError::AlreadyRunning => {
            AppError::InvalidInput("Worker is already running".to_owned())
        }
        WorkerManagerError::NotRunning => {
            AppError::InvalidInput("Worker is not running".to_owned())
        }
        WorkerManagerError::ShuttingDown
        | WorkerManagerError::InvalidConfig
        | WorkerManagerError::TaskRunPersistence => AppError::Internal,
    }
}

fn map_stop_error(error: WorkerManagerError) -> AppError {
    match error {
        WorkerManagerError::NotRunning => {
            AppError::InvalidInput("Worker is not running".to_owned())
        }
        other => map_run_error(other),
    }
}
