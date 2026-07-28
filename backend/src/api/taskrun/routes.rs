use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_get_task_run, __path_list_task_run_events, __path_list_task_runs, get_task_run,
        list_task_run_events, list_task_runs,
    },
    state::TaskRunState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    TaskRunState: FromRef<S>,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_task_runs))
        .routes(routes!(get_task_run))
        .routes(routes!(list_task_run_events))
}
