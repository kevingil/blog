use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_organization, __path_delete_organization, __path_get_organization,
        __path_join_organization, __path_leave_organization, __path_list_organizations,
        __path_update_organization, create_organization, delete_organization, get_organization,
        join_organization, leave_organization, list_organizations, update_organization,
    },
    state::OrganizationState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    OrganizationState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_organizations, create_organization))
        .routes(routes!(leave_organization))
        .routes(routes!(
            get_organization,
            update_organization,
            delete_organization
        ))
        .routes(routes!(join_organization))
}
