use axum::{extract::FromRef, routing::post};
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_source, __path_delete_source, __path_get_article_sources, __path_get_source,
        __path_list_all_sources, __path_scrape_and_create_source, __path_search_similar_sources,
        __path_update_source, create_source, delete_source, get_article_sources, get_source,
        list_all_sources, scrape_and_create_source, search_similar_sources, update_source,
    },
    state::SourceState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    SourceState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_all_sources))
        .routes(routes!(create_source))
        // Preserve the active frontend caller, which includes the group-root slash.
        .route("/sources/", post(create_source))
        .routes(routes!(scrape_and_create_source))
        .routes(routes!(get_article_sources))
        .routes(routes!(search_similar_sources))
        .routes(routes!(get_source, update_source, delete_source))
}
