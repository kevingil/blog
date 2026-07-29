use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_get_my_profile, __path_get_public_profile, __path_get_site_settings,
        __path_update_profile, __path_update_site_settings, get_my_profile, get_public_profile,
        get_site_settings, update_profile, update_site_settings,
    },
    state::ProfileState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    ProfileState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(get_public_profile))
        .routes(routes!(get_my_profile, update_profile))
        .routes(routes!(get_site_settings, update_site_settings))
}
