use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_project, __path_delete_project, __path_get_project, __path_list_projects,
        __path_update_project, create_project, delete_project, get_project, list_projects,
        update_project,
    },
    state::ProjectState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    ProjectState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_projects, create_project))
        .routes(routes!(get_project, update_project, delete_project))
}
