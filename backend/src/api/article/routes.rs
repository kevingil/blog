use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_article, __path_delete_article, __path_generate_article,
        __path_get_article_data, __path_get_articles, __path_get_popular_tags,
        __path_get_recommended_articles, __path_get_version, __path_list_versions,
        __path_publish_article, __path_revert_to_version, __path_search_articles,
        __path_unpublish_article, __path_update_article, __path_update_article_with_context,
        create_article, delete_article, generate_article, get_article_data, get_articles,
        get_popular_tags, get_recommended_articles, get_version, list_versions, publish_article,
        revert_to_version, search_articles, unpublish_article, update_article,
        update_article_with_context,
    },
    state::ArticleState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    ArticleState: FromRef<S>,
    AuthState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(generate_article))
        .routes(routes!(create_article))
        .routes(routes!(update_article))
        .routes(routes!(update_article_with_context))
        .routes(routes!(get_articles))
        .routes(routes!(search_articles))
        .routes(routes!(get_popular_tags))
        .routes(routes!(get_article_data, delete_article))
        .routes(routes!(get_recommended_articles))
        .routes(routes!(publish_article))
        .routes(routes!(unpublish_article))
        .routes(routes!(list_versions))
        .routes(routes!(get_version))
        .routes(routes!(revert_to_version))
}
