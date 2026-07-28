use std::collections::BTreeMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::{
    core::source::{
        CreateSourceRequest as CoreCreateRequest, Source as CoreSource,
        SourceListResponse as CoreListResponse, SourceWithArticle as CoreSourceWithArticle,
        UpdateSourceRequest as CoreUpdateRequest,
    },
    error::AppError,
};

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateSourceRequest {
    pub article_id: Uuid,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    #[schema(min_length = 1)]
    pub content: String,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub source_type: String,
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
}

impl CreateSourceRequest {
    pub fn validate(self) -> Result<CoreCreateRequest, AppError> {
        if self.article_id.is_nil() {
            return Err(invalid("article_id"));
        }
        if self.content.is_empty() {
            return Err(invalid("content"));
        }
        Ok(CoreCreateRequest {
            article_id: self.article_id,
            title: self.title,
            content: self.content,
            url: self.url,
            source_type: self.source_type,
            meta_data: self.meta_data,
        })
    }
}

#[derive(Debug, Deserialize, Default, ToSchema)]
pub struct UpdateSourceRequest {
    pub title: Option<String>,
    pub content: Option<String>,
    pub url: Option<String>,
    pub source_type: Option<String>,
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
}

impl From<UpdateSourceRequest> for CoreUpdateRequest {
    fn from(value: UpdateSourceRequest) -> Self {
        Self {
            title: value.title,
            content: value.content,
            url: value.url,
            source_type: value.source_type,
            meta_data: value.meta_data,
        }
    }
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct ScrapeSourceRequest {
    pub article_id: Uuid,
    #[schema(format = "uri")]
    pub url: String,
}

impl ScrapeSourceRequest {
    pub fn validate(&self) -> Result<(), AppError> {
        if self.article_id.is_nil() {
            return Err(invalid("article_id"));
        }
        if !valid_url(&self.url) {
            return Err(invalid("url"));
        }
        Ok(())
    }
}

#[derive(Debug, Deserialize, Default, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct SourceListQuery {
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 1)]
    pub page: Option<i64>,
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 20)]
    pub limit: Option<i64>,
}

#[derive(Debug, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct SourceSearchQuery {
    #[serde(default)]
    #[param(min_length = 1)]
    pub q: String,
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 5)]
    pub limit: Option<i64>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct SourceResponse {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: String,
    pub content: String,
    pub url: String,
    pub source_type: String,
    pub embedding: Option<Vec<f32>>,
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
    pub created_at: String,
}

impl From<CoreSource> for SourceResponse {
    fn from(value: CoreSource) -> Self {
        Self {
            id: value.id,
            article_id: value.article_id,
            title: value.title,
            content: value.content,
            url: value.url,
            source_type: value.source_type,
            embedding: value.embedding,
            meta_data: value.meta_data,
            created_at: timestamp(value.created_at),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SourceWithArticleResponse {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: String,
    pub content: String,
    pub content_preview: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub url: String,
    pub source_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
    pub created_at: String,
    pub article_title: String,
    pub article_slug: String,
}

impl From<CoreSourceWithArticle> for SourceWithArticleResponse {
    fn from(value: CoreSourceWithArticle) -> Self {
        let preview = content_preview(&value.source.content);
        Self {
            id: value.source.id,
            article_id: value.source.article_id,
            title: value.source.title,
            content: value.source.content,
            content_preview: preview,
            url: value.source.url,
            source_type: value.source.source_type,
            meta_data: value.source.meta_data,
            created_at: timestamp(value.source.created_at),
            article_title: value.article_title,
            article_slug: value.article_slug,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SourceListResponse {
    pub sources: Vec<SourceWithArticleResponse>,
    pub total_pages: i64,
    pub page: i64,
}

impl From<CoreListResponse> for SourceListResponse {
    fn from(value: CoreListResponse) -> Self {
        Self {
            sources: value.sources.into_iter().map(Into::into).collect(),
            total_pages: value.total_pages,
            page: value.page,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SourcesResponse {
    pub sources: Vec<SourceResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SearchSourcesResponse {
    pub sources: Vec<SourceResponse>,
    pub query: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessFlag {
    pub success: bool,
}

fn content_preview(content: &str) -> String {
    let mut chars = content.chars();
    let prefix: String = chars.by_ref().take(300).collect();
    if chars.next().is_some() {
        format!("{prefix}...")
    } else {
        prefix
    }
}

fn valid_url(value: &str) -> bool {
    reqwest::Url::parse(value).is_ok_and(|url| !url.scheme().is_empty() && url.has_host())
}

fn invalid(field: &str) -> AppError {
    AppError::InvalidInput(format!("{field}: failed validation"))
}

fn timestamp(value: Option<DateTime<Utc>>) -> String {
    value.map_or_else(
        || "0001-01-01T00:00:00Z".to_owned(),
        |value| value.to_rfc3339_opts(SecondsFormat::AutoSi, true),
    )
}

fn optional_i64<'de, D>(deserializer: D) -> Result<Option<i64>, D::Error>
where
    D: serde::Deserializer<'de>,
{
    Option::<String>::deserialize(deserializer)
        .map(|value| value.and_then(|value| value.parse().ok()))
}
