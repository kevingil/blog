use axum::extract::FromRef;
use utoipa_axum::{router::OpenApiRouter, routes};

use crate::api::auth::AuthState;

use super::{
    handlers::{
        __path_create_topic, __path_delete_insight, __path_delete_topic, __path_get_insight,
        __path_get_recent_crawled_content, __path_get_topic, __path_get_unread_count,
        __path_list_insights, __path_list_topics, __path_mark_insight_as_read,
        __path_search_crawled_content, __path_search_insights, __path_toggle_insight_pinned,
        __path_update_topic, create_topic, delete_insight, delete_topic, get_insight,
        get_recent_crawled_content, get_topic, get_unread_count, list_insights, list_topics,
        mark_insight_as_read, search_crawled_content, search_insights, toggle_insight_pinned,
        update_topic,
    },
    state::InsightState,
};

pub fn router<S>() -> OpenApiRouter<S>
where
    S: Clone + Send + Sync + 'static,
    AuthState: FromRef<S>,
    InsightState: FromRef<S>,
{
    OpenApiRouter::new()
        .routes(routes!(list_insights))
        .routes(routes!(search_insights))
        .routes(routes!(get_unread_count))
        .routes(routes!(list_topics, create_topic))
        .routes(routes!(get_topic, update_topic, delete_topic))
        .routes(routes!(search_crawled_content))
        .routes(routes!(get_recent_crawled_content))
        .routes(routes!(get_insight, delete_insight))
        .routes(routes!(mark_insight_as_read))
        .routes(routes!(toggle_insight_pinned))
}
