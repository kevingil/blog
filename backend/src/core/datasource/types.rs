use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct DataSource {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub user_id: Option<Uuid>,
    pub name: String,
    pub url: String,
    pub feed_url: Option<String>,
    pub source_type: String,
    pub crawl_frequency: String,
    pub is_enabled: bool,
    pub is_discovered: bool,
    pub discovered_from_id: Option<Uuid>,
    pub last_crawled_at: Option<DateTime<Utc>>,
    pub next_crawl_at: Option<DateTime<Utc>>,
    pub crawl_status: String,
    pub error_message: Option<String>,
    pub content_count: i32,
    pub subscriber_count: i32,
    pub meta_data: Option<MetaData>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct CrawledContent {
    pub id: Uuid,
    pub data_source_id: Uuid,
    pub url: String,
    pub title: Option<String>,
    pub content: String,
    pub summary: Option<String>,
    pub author: Option<String>,
    pub published_at: Option<DateTime<Utc>>,
    pub embedding: Option<Vec<f32>>,
    pub meta_data: Option<MetaData>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DataSourceCreateRequest {
    pub name: String,
    pub url: String,
    pub feed_url: Option<String>,
    pub source_type: String,
    pub crawl_frequency: String,
    pub is_enabled: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct DataSourceUpdateRequest {
    pub name: Option<String>,
    pub url: Option<String>,
    pub feed_url: Option<String>,
    pub source_type: Option<String>,
    pub crawl_frequency: Option<String>,
    pub is_enabled: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DataSourceRecommendationRequest {
    pub query: String,
    pub limit: i32,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct DataSourceDiscoveryRecommendationRequest {
    pub limit: i32,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
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
    pub meta_data: Option<MetaData>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

impl From<DataSource> for DataSourceResponse {
    fn from(value: DataSource) -> Self {
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
            created_at: value.created_at,
            updated_at: value.updated_at,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct DataSourceRecommendationResponse {
    pub name: String,
    pub url: String,
    pub domain: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub summary: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub reason: String,
    pub source_type: String,
    #[serde(skip_serializing_if = "is_zero")]
    pub score: f64,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub favicon: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub sample_url: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub sample_title: String,
}

fn is_zero(value: &f64) -> bool {
    *value == 0.0
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct DataSourceRecommendationsResponse {
    #[serde(skip_serializing_if = "String::is_empty")]
    pub mode: String,
    pub query: String,
    #[serde(skip_serializing_if = "is_zero_i32")]
    pub seed_count: i32,
    pub recommendations: Vec<DataSourceRecommendationResponse>,
}

fn is_zero_i32(value: &i32) -> bool {
    *value == 0
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
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
    pub meta_data: Option<MetaData>,
    pub created_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data_source_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data_source_url: Option<String>,
}

impl From<CrawledContent> for CrawledContentResponse {
    fn from(value: CrawledContent) -> Self {
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
            created_at: value.created_at,
            data_source_name: None,
            data_source_url: None,
        }
    }
}
