use std::collections::BTreeMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::{
    core::datasource::{
        CrawledContentResponse as CoreCrawledContentResponse,
        DataSourceCreateRequest as CoreCreateRequest,
        DataSourceDiscoveryRecommendationRequest as CoreDiscoveryRequest,
        DataSourceRecommendationRequest as CoreRecommendationRequest,
        DataSourceRecommendationResponse as CoreRecommendationResponse,
        DataSourceRecommendationsResponse as CoreRecommendationsResponse,
        DataSourceResponse as CoreDataSourceResponse, DataSourceUpdateRequest as CoreUpdateRequest,
    },
    error::AppError,
};

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Deserialize, ToSchema)]
pub struct DataSourceCreateRequest {
    #[schema(min_length = 1, max_length = 255)]
    pub name: String,
    #[schema(format = "uri")]
    pub url: String,
    #[schema(format = "uri")]
    pub feed_url: Option<String>,
    #[serde(default)]
    #[schema(pattern = "^(blog|forum|news|rss|newsletter)?$")]
    pub source_type: String,
    #[serde(default)]
    #[schema(pattern = "^(hourly|daily|weekly)?$")]
    pub crawl_frequency: String,
    pub is_enabled: Option<bool>,
}

impl DataSourceCreateRequest {
    pub fn validate(self) -> Result<CoreCreateRequest, AppError> {
        validate_length("name", &self.name, 1, 255)?;
        validate_url("url", &self.url)?;
        if let Some(feed_url) = self.feed_url.as_deref() {
            validate_url("feed_url", feed_url)?;
        }
        if !self.source_type.is_empty()
            && !["blog", "forum", "news", "rss", "newsletter"].contains(&self.source_type.as_str())
        {
            return Err(invalid("source_type"));
        }
        if !self.crawl_frequency.is_empty()
            && !["hourly", "daily", "weekly"].contains(&self.crawl_frequency.as_str())
        {
            return Err(invalid("crawl_frequency"));
        }
        Ok(CoreCreateRequest {
            name: self.name,
            url: self.url,
            feed_url: self.feed_url,
            source_type: self.source_type,
            crawl_frequency: self.crawl_frequency,
            is_enabled: self.is_enabled,
        })
    }
}

#[derive(Debug, Deserialize, Default, ToSchema)]
pub struct DataSourceUpdateRequest {
    pub name: Option<String>,
    pub url: Option<String>,
    pub feed_url: Option<String>,
    pub source_type: Option<String>,
    pub crawl_frequency: Option<String>,
    pub is_enabled: Option<bool>,
}

impl DataSourceUpdateRequest {
    pub fn validate(self) -> Result<CoreUpdateRequest, AppError> {
        if let Some(name) = self.name.as_deref() {
            validate_length("name", name, 1, 255)?;
        }
        if let Some(url) = self.url.as_deref() {
            validate_url("url", url)?;
        }
        if let Some(feed_url) = self.feed_url.as_deref() {
            validate_url("feed_url", feed_url)?;
        }
        if self
            .source_type
            .as_deref()
            .is_some_and(|value| !["blog", "forum", "news", "rss", "newsletter"].contains(&value))
        {
            return Err(invalid("source_type"));
        }
        if self
            .crawl_frequency
            .as_deref()
            .is_some_and(|value| !["hourly", "daily", "weekly"].contains(&value))
        {
            return Err(invalid("crawl_frequency"));
        }
        Ok(CoreUpdateRequest {
            name: self.name,
            url: self.url,
            feed_url: self.feed_url,
            source_type: self.source_type,
            crawl_frequency: self.crawl_frequency,
            is_enabled: self.is_enabled,
        })
    }
}

impl From<DataSourceUpdateRequest> for CoreUpdateRequest {
    fn from(value: DataSourceUpdateRequest) -> Self {
        Self {
            name: value.name,
            url: value.url,
            feed_url: value.feed_url,
            source_type: value.source_type,
            crawl_frequency: value.crawl_frequency,
            is_enabled: value.is_enabled,
        }
    }
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct DataSourceRecommendationRequest {
    #[schema(min_length = 3, max_length = 500)]
    pub query: String,
    #[serde(default)]
    pub limit: i32,
}

impl DataSourceRecommendationRequest {
    pub fn validate(self) -> Result<CoreRecommendationRequest, AppError> {
        validate_length("query", &self.query, 3, 500)?;
        Ok(CoreRecommendationRequest {
            query: self.query,
            limit: self.limit,
        })
    }
}

#[derive(Debug, Deserialize, Default, ToSchema)]
pub struct DataSourceDiscoveryRecommendationRequest {
    #[serde(default)]
    pub limit: i32,
}

impl From<DataSourceDiscoveryRecommendationRequest> for CoreDiscoveryRequest {
    fn from(value: DataSourceDiscoveryRecommendationRequest) -> Self {
        Self { limit: value.limit }
    }
}

#[derive(Debug, Deserialize, Default, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct PaginationQuery {
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 1)]
    pub page: Option<i64>,
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 20)]
    pub limit: Option<i64>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DataSourceResponse {
    pub id: Uuid,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_id: Option<Uuid>,
    pub name: String,
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub feed_url: Option<String>,
    pub source_type: String,
    pub crawl_frequency: String,
    pub is_enabled: bool,
    pub is_discovered: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub discovered_from_id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_crawled_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_crawl_at: Option<DateTime<Utc>>,
    pub crawl_status: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error_message: Option<String>,
    pub content_count: i32,
    pub subscriber_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
    pub created_at: String,
    pub updated_at: String,
}

impl From<CoreDataSourceResponse> for DataSourceResponse {
    fn from(value: CoreDataSourceResponse) -> Self {
        Self {
            id: value.id,
            organization_id: value.organization_id,
            user_id: value.user_id,
            name: value.name,
            url: value.url,
            feed_url: value.feed_url,
            source_type: value.source_type,
            crawl_frequency: value.crawl_frequency,
            is_enabled: value.is_enabled,
            is_discovered: value.is_discovered,
            discovered_from_id: value.discovered_from_id,
            last_crawled_at: value.last_crawled_at,
            next_crawl_at: value.next_crawl_at,
            crawl_status: value.crawl_status,
            error_message: value.error_message,
            content_count: value.content_count,
            subscriber_count: value.subscriber_count,
            meta_data: value.meta_data,
            created_at: timestamp(value.created_at),
            updated_at: timestamp(value.updated_at),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DataSourceRecommendationResponse {
    pub name: String,
    pub url: String,
    pub domain: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub summary: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub reason: String,
    pub source_type: String,
    #[serde(skip_serializing_if = "is_zero_f64")]
    pub score: f64,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub favicon: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub sample_url: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub sample_title: String,
}

impl From<CoreRecommendationResponse> for DataSourceRecommendationResponse {
    fn from(value: CoreRecommendationResponse) -> Self {
        Self {
            name: value.name,
            url: value.url,
            domain: value.domain,
            summary: value.summary,
            reason: value.reason,
            source_type: value.source_type,
            score: value.score,
            favicon: value.favicon,
            sample_url: value.sample_url,
            sample_title: value.sample_title,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DataSourceRecommendationsResponse {
    #[serde(skip_serializing_if = "String::is_empty")]
    pub mode: String,
    pub query: String,
    #[serde(skip_serializing_if = "is_zero_i32")]
    pub seed_count: i32,
    pub recommendations: Vec<DataSourceRecommendationResponse>,
}

impl From<CoreRecommendationsResponse> for DataSourceRecommendationsResponse {
    fn from(value: CoreRecommendationsResponse) -> Self {
        Self {
            mode: value.mode,
            query: value.query,
            seed_count: value.seed_count,
            recommendations: value.recommendations.into_iter().map(Into::into).collect(),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CrawledContentResponse {
    pub id: Uuid,
    pub data_source_id: Uuid,
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub summary: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub author: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub published_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data_source_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data_source_url: Option<String>,
}

impl From<CoreCrawledContentResponse> for CrawledContentResponse {
    fn from(value: CoreCrawledContentResponse) -> Self {
        Self {
            id: value.id,
            data_source_id: value.data_source_id,
            url: value.url,
            title: value.title,
            content: value.content,
            summary: value.summary,
            author: value.author,
            published_at: value.published_at,
            meta_data: value.meta_data,
            created_at: timestamp(value.created_at),
            data_source_name: value.data_source_name,
            data_source_url: value.data_source_url,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessFlag {
    pub success: bool,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CrawlTriggeredResponse {
    pub success: bool,
    pub message: &'static str,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DataSourceContentResponse {
    pub contents: Vec<CrawledContentResponse>,
    pub total: i64,
    pub page: i64,
    pub limit: i64,
}

fn validate_length(field: &str, value: &str, min: usize, max: usize) -> Result<(), AppError> {
    let length = value.chars().count();
    if !(min..=max).contains(&length) {
        return Err(invalid(field));
    }
    Ok(())
}

fn validate_url(field: &str, value: &str) -> Result<(), AppError> {
    let valid = reqwest::Url::parse(value)
        .is_ok_and(|url| !url.scheme().is_empty() && (url.has_host() || url.scheme() == "mailto"));
    if valid { Ok(()) } else { Err(invalid(field)) }
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

fn is_zero_f64(value: &f64) -> bool {
    *value == 0.0
}

fn is_zero_i32(value: &i32) -> bool {
    *value == 0
}

fn optional_i64<'de, D>(deserializer: D) -> Result<Option<i64>, D::Error>
where
    D: serde::Deserializer<'de>,
{
    Option::<String>::deserialize(deserializer)
        .map(|value| value.and_then(|value| value.parse().ok()))
}
