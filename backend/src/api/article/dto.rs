use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::{IntoParams, ToSchema};

#[derive(Debug, Deserialize, ToSchema)]
pub struct GenerateArticleRequest {
    #[serde(default)]
    pub prompt: String,
    #[serde(default)]
    pub title: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct GenerateArticleResponse {
    pub article: crate::core::article::Article,
    pub request_id: String,
}

#[derive(Debug, Deserialize, Default, IntoParams)]
#[into_params(parameter_in = Query)]
#[serde(rename_all = "camelCase")]
pub struct ArticleListQuery {
    #[param(default = 1, minimum = 1)]
    pub page: Option<i64>,
    #[param(default = 6, minimum = 1)]
    pub articles_per_page: Option<i64>,
    pub tag: Option<String>,
    #[param(default = "published")]
    pub status: Option<String>,
    pub sort_by: Option<String>,
    pub sort_order: Option<String>,
}

#[derive(Debug, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
#[serde(rename_all = "camelCase")]
pub struct ArticleSearchQuery {
    pub query: String,
    #[param(default = 1, minimum = 1)]
    pub page: Option<i64>,
    pub tag: Option<String>,
    #[param(default = "published")]
    pub status: Option<String>,
    pub sort_by: Option<String>,
    pub sort_order: Option<String>,
}

#[derive(Debug, Deserialize, Default, ToSchema)]
pub struct PublishArticleRequest {
    pub published_at: Option<i64>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PopularTagsResponse {
    pub tags: Vec<String>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DeleteArticleResponse {
    pub success: bool,
}

pub fn timestamp(timestamp: i64) -> Option<DateTime<Utc>> {
    use chrono::TimeZone;
    if timestamp > 1_000_000_000_000 {
        Utc.timestamp_millis_opt(timestamp).single()
    } else {
        Utc.timestamp_opt(timestamp, 0).single()
    }
}
