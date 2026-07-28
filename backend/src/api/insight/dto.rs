use std::collections::BTreeMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::{
    core::{
        datasource::CrawledContentResponse as CoreCrawledContentResponse,
        insight::{
            InsightResponse as CoreInsightResponse,
            InsightTopicCreateRequest as CoreTopicCreateRequest,
            InsightTopicResponse as CoreTopicResponse,
            InsightTopicUpdateRequest as CoreTopicUpdateRequest,
            InsightWithSources as CoreInsightWithSources,
            InsightWithUserStatus as CoreInsightWithUserStatus,
            UserInsightStatusResponse as CoreUserStatusResponse,
        },
    },
    error::AppError,
};

pub type MetaData = BTreeMap<String, Value>;

#[derive(Debug, Deserialize, Default, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct InsightListQuery {
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 1)]
    pub page: Option<i64>,
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 20)]
    pub limit: Option<i64>,
    pub topic_id: Option<String>,
}

#[derive(Debug, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct SearchQuery {
    #[serde(default)]
    #[param(min_length = 1)]
    pub q: String,
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 10)]
    pub limit: Option<i64>,
}

#[derive(Debug, Deserialize, Default, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct RecentQuery {
    #[serde(default, deserialize_with = "optional_i64")]
    #[param(default = 20)]
    pub limit: Option<i64>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct InsightTopicCreateRequest {
    #[schema(min_length = 1, max_length = 255)]
    pub name: String,
    pub description: Option<String>,
    #[serde(default)]
    pub keywords: Option<Vec<String>>,
    #[schema(max_length = 20)]
    pub color: Option<String>,
    #[schema(max_length = 50)]
    pub icon: Option<String>,
}

impl InsightTopicCreateRequest {
    pub fn validate(self) -> Result<CoreTopicCreateRequest, AppError> {
        validate_length("name", &self.name, 1, 255)?;
        if let Some(color) = self.color.as_deref() {
            validate_max("color", color, 20)?;
        }
        if let Some(icon) = self.icon.as_deref() {
            validate_max("icon", icon, 50)?;
        }
        Ok(CoreTopicCreateRequest {
            name: self.name,
            description: self.description,
            keywords: self.keywords,
            color: self.color,
            icon: self.icon,
        })
    }
}

#[derive(Debug, Deserialize, Default, ToSchema)]
pub struct InsightTopicUpdateRequest {
    pub name: Option<String>,
    pub description: Option<String>,
    pub keywords: Option<Vec<String>>,
    pub color: Option<String>,
    pub icon: Option<String>,
}

impl From<InsightTopicUpdateRequest> for CoreTopicUpdateRequest {
    fn from(value: InsightTopicUpdateRequest) -> Self {
        Self {
            name: value.name,
            description: value.description,
            keywords: value.keywords,
            color: value.color,
            icon: value.icon,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
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
    pub created_at: String,
    pub updated_at: String,
}

impl From<CoreTopicResponse> for InsightTopicResponse {
    fn from(value: CoreTopicResponse) -> Self {
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
            created_at: timestamp(value.created_at),
            updated_at: timestamp(value.updated_at),
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
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
    pub generated_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub period_start: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[schema(value_type = Option<Object>)]
    pub meta_data: Option<MetaData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_color: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic_icon: Option<String>,
}

impl From<CoreInsightResponse> for InsightResponse {
    fn from(value: CoreInsightResponse) -> Self {
        Self {
            id: value.id,
            organization_id: value.organization_id,
            topic_id: value.topic_id,
            title: value.title,
            summary: value.summary,
            content: value.content,
            key_points: value.key_points,
            source_content_ids: value.source_content_ids,
            generated_at: timestamp(value.generated_at),
            period_start: value.period_start,
            period_end: value.period_end,
            is_read: value.is_read,
            is_pinned: value.is_pinned,
            is_used_in_article: value.is_used_in_article,
            meta_data: value.meta_data,
            topic_name: value.topic_name,
            topic_color: value.topic_color,
            topic_icon: value.topic_icon,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct UserInsightStatusResponse {
    pub insight_id: Uuid,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_at: Option<DateTime<Utc>>,
}

impl From<CoreUserStatusResponse> for UserInsightStatusResponse {
    fn from(value: CoreUserStatusResponse) -> Self {
        Self {
            insight_id: value.insight_id,
            is_read: value.is_read,
            is_pinned: value.is_pinned,
            is_used_in_article: value.is_used_in_article,
            read_at: value.read_at,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct InsightWithUserStatus {
    #[serde(flatten)]
    pub insight: InsightResponse,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_status: Option<UserInsightStatusResponse>,
}

impl From<CoreInsightWithUserStatus> for InsightWithUserStatus {
    fn from(value: CoreInsightWithUserStatus) -> Self {
        Self {
            insight: value.insight.into(),
            user_status: value.user_status.map(Into::into),
        }
    }
}

impl From<CoreInsightResponse> for InsightWithUserStatus {
    fn from(value: CoreInsightResponse) -> Self {
        Self {
            insight: value.into(),
            user_status: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
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

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct InsightWithSources {
    #[serde(flatten)]
    pub insight: InsightResponse,
    pub source_contents: Vec<CrawledContentResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub topic: Option<InsightTopicResponse>,
}

impl From<CoreInsightWithSources> for InsightWithSources {
    fn from(value: CoreInsightWithSources) -> Self {
        Self {
            insight: value.insight.into(),
            source_contents: value.source_contents.into_iter().map(Into::into).collect(),
            topic: value.topic.map(Into::into),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct InsightListResponse {
    pub insights: Vec<InsightWithUserStatus>,
    pub total: i64,
    pub page: i64,
    pub limit: i64,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CountResponse {
    pub count: i64,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessFlag {
    pub success: bool,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PinResponse {
    pub success: bool,
    pub is_pinned: bool,
}

fn validate_length(field: &str, value: &str, min: usize, max: usize) -> Result<(), AppError> {
    let length = value.chars().count();
    if !(min..=max).contains(&length) {
        return Err(invalid(field));
    }
    Ok(())
}

fn validate_max(field: &str, value: &str, max: usize) -> Result<(), AppError> {
    if value.chars().count() > max {
        return Err(invalid(field));
    }
    Ok(())
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
