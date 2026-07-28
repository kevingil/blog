use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_page, __path_delete_page, __path_get_page_by_id, __path_get_page_by_slug,
        __path_list_pages, __path_update_page, create_page, delete_page, get_page_by_id,
        get_page_by_slug, list_pages, update_page,
    },
    state::PageState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    PageState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(get_page_by_slug))
        .routes(routes!(list_pages, create_page))
        .routes(routes!(get_page_by_id, update_page, delete_page))
}
