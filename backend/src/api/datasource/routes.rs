use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_data_source, __path_delete_data_source, __path_discover_data_sources,
        __path_get_data_source, __path_get_data_source_content, __path_list_data_sources,
        __path_recommend_data_sources, __path_trigger_crawl, __path_update_data_source,
        create_data_source, delete_data_source, discover_data_sources, get_data_source,
        get_data_source_content, list_data_sources, recommend_data_sources, trigger_crawl,
        update_data_source,
    },
    state::DataSourceState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    DataSourceState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_data_sources, create_data_source))
        .routes(routes!(recommend_data_sources))
        .routes(routes!(discover_data_sources))
        .routes(routes!(
            get_data_source,
            update_data_source,
            delete_data_source
        ))
        .routes(routes!(trigger_crawl))
        .routes(routes!(get_data_source_content))
}
