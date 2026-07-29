use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Source {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: String,
    pub content: String,
    pub url: String,
    pub source_type: String,
    pub embedding: Option<Vec<f32>>,
    pub meta_data: Option<MetaData>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SourceListOptions {
    pub page: i64,
    pub per_page: i64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SourceWithArticle {
    #[serde(flatten)]
    pub source: Source,
    pub article_title: String,
    pub article_slug: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct CreateSourceRequest {
    pub article_id: Uuid,
    pub title: String,
    pub content: String,
    pub url: String,
    pub source_type: String,
    pub meta_data: Option<MetaData>,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct UpdateSourceRequest {
    pub title: Option<String>,
    pub content: Option<String>,
    pub url: Option<String>,
    pub source_type: Option<String>,
    pub meta_data: Option<MetaData>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AgentResourceSelection {
    pub article_id: Uuid,
    pub source_id: Option<Uuid>,
    pub title: String,
    pub content: String,
    pub url: String,
    pub source_type: String,
    pub origin_tool: String,
    pub origin_query: String,
    pub origin_question: String,
    pub author: String,
    pub published_date: String,
    pub selected_excerpt: String,
    pub selected_excerpt_id: String,
    pub request_id: String,
    pub usage_status: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScrapedContent {
    pub title: String,
    pub content: String,
    pub url: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SourceListResponse {
    pub sources: Vec<SourceWithArticle>,
    pub total_pages: i64,
    pub page: i64,
}
