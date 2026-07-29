use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_get_all_worker_status, __path_get_running_workers, __path_get_worker_status,
        __path_run_worker, __path_stop_worker, get_all_worker_status, get_running_workers,
        get_worker_status, run_worker, stop_worker,
    },
    state::WorkerState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    WorkerState: FromRef<S>,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(get_all_worker_status))
        .routes(routes!(get_running_workers))
        .routes(routes!(get_worker_status))
        .routes(routes!(run_worker))
        .routes(routes!(stop_worker))
}
