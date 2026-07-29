use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use super::{
    handlers::{
        __path_delete_account, __path_login, __path_logout, __path_register, __path_update_account,
        __path_update_password, delete_account, login, logout, register, update_account,
        update_password,
    },
    state::AuthState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(login))
        .routes(routes!(register))
        .routes(routes!(logout))
        .routes(routes!(update_account))
        .routes(routes!(update_password))
        .routes(routes!(delete_account))
}
