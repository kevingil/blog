use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

use crate::core::datasource::CrawledContentResponse;

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightTopic {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub description: Option<String>,
    pub keywords: Option<Vec<String>>,
    pub embedding: Option<Vec<f32>>,
    pub is_auto_generated: bool,
    pub content_count: i32,
    pub last_insight_at: Option<DateTime<Utc>>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ContentTopicMatch {
    pub id: Uuid,
    pub content_id: Uuid,
    pub topic_id: Uuid,
    pub similarity_score: f64,
    pub is_primary: bool,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Insight {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    pub content: Option<String>,
    pub key_points: Option<Vec<String>>,
    pub source_content_ids: Vec<Uuid>,
    pub embedding: Option<Vec<f32>>,
    pub generated_at: Option<DateTime<Utc>>,
    pub period_start: Option<DateTime<Utc>>,
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    pub meta_data: Option<MetaData>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct UserInsightStatus {
    pub id: Uuid,
    pub user_id: Uuid,
    pub insight_id: Uuid,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    pub read_at: Option<DateTime<Utc>>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightTopicCreateRequest {
    pub name: String,
    pub description: Option<String>,
    pub keywords: Option<Vec<String>>,
    pub color: Option<String>,
    pub icon: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct InsightTopicUpdateRequest {
    pub name: Option<String>,
    pub description: Option<String>,
    pub keywords: Option<Vec<String>>,
    pub color: Option<String>,
    pub icon: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightSearchRequest {
    pub query: String,
    pub topic_id: Option<Uuid>,
    pub limit: i64,
    pub is_unread: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightTopicResponse {
    pub id: Uuid,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<Uuid>,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub keywords: Option<Vec<String>>,
    pub is_auto_generated: bool,
    pub content_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_insight_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub color: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icon: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

impl From<InsightTopic> for InsightTopicResponse {
    fn from(value: InsightTopic) -> Self {
        Self {
            id: value.id,
            organization_id: value.organization_id,
            name: value.name,
            description: value.description,
            keywords: value.keywords,
            is_auto_generated: value.is_auto_generated,
            content_count: value.content_count,
            last_insight_at: value.last_insight_at,
            color: value.color,
            icon: value.icon,
            created_at: value.created_at,
            updated_at: value.updated_at,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightResponse {
    pub id: Uuid,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub organization_id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub key_points: Option<Vec<String>>,
    pub source_content_ids: Vec<Uuid>,
    pub generated_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub period_start: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub meta_data: Option<MetaData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_color: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_icon: Option<String>,
}

impl From<Insight> for InsightResponse {
    fn from(value: Insight) -> Self {
        Self {
            id: value.id,
            organization_id: value.organization_id,
            topic_id: value.topic_id,
            title: value.title,
            summary: value.summary,
            content: value.content,
            key_points: value.key_points,
            source_content_ids: value.source_content_ids,
            generated_at: value.generated_at,
            period_start: value.period_start,
            period_end: value.period_end,
            is_read: value.is_read,
            is_pinned: value.is_pinned,
            is_used_in_article: value.is_used_in_article,
            meta_data: value.meta_data,
            topic_name: None,
            topic_color: None,
            topic_icon: None,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct UserInsightStatusResponse {
    pub insight_id: Uuid,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_at: Option<DateTime<Utc>>,
}

impl From<UserInsightStatus> for UserInsightStatusResponse {
    fn from(value: UserInsightStatus) -> Self {
        Self {
            insight_id: value.insight_id,
            is_read: value.is_read,
            is_pinned: value.is_pinned,
            is_used_in_article: value.is_used_in_article,
            read_at: value.read_at,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightWithUserStatus {
    #[serde(flatten)]
    pub insight: InsightResponse,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_status: Option<UserInsightStatusResponse>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct InsightWithSources {
    #[serde(flatten)]
    pub insight: InsightResponse,
    pub source_contents: Vec<CrawledContentResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic: Option<InsightTopicResponse>,
}
