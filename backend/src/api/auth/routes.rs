use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use super::{
    handlers::{
        __path_create_mcp_token, __path_delete_account, __path_list_mcp_tokens, __path_login,
        __path_logout, __path_refresh, __path_register, __path_revoke_mcp_token,
        __path_update_account, __path_update_password, create_mcp_token, delete_account,
        list_mcp_tokens, login, logout, refresh, register, revoke_mcp_token, update_account,
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
        .routes(routes!(refresh))
        .routes(routes!(register))
        .routes(routes!(logout))
        .routes(routes!(update_account))
        .routes(routes!(update_password))
        .routes(routes!(delete_account))
        .routes(routes!(create_mcp_token, list_mcp_tokens))
        .routes(routes!(revoke_mcp_token))
}
