use std::collections::BTreeMap;

use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Deserializer, Serialize};
use serde_json::Value;
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::core::page::{
    Page as CorePage, PageCreateRequest as CoreCreateRequest, PageListResult as CoreListResult,
    PageUpdateRequest as CoreUpdateRequest,
};

use super::error::PageApiError;

#[derive(Debug, Default, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
#[serde(rename_all = "camelCase")]
pub struct PageListQuery {
    #[param(default = 1)]
    #[param(value_type = Option<i64>)]
    pub page: Option<String>,
    #[param(default = 20)]
    #[param(value_type = Option<i64>)]
    pub per_page: Option<String>,
    #[param(value_type = Option<bool>)]
    pub is_published: Option<String>,
}

impl PageListQuery {
    pub fn values(&self) -> (i64, i64, Option<bool>) {
        let page = parse_i64(self.page.as_deref(), 1);
        let per_page = parse_i64(self.per_page.as_deref(), 20);
        let is_published = self
            .is_published
            .as_deref()
            .filter(|value| !value.is_empty())
            .map(|value| value.parse().unwrap_or(true));
        (page, per_page, is_published)
    }
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct PageCreateRequest {
    #[serde(default, deserialize_with = "null_default")]
    pub slug: String,
    #[serde(default, deserialize_with = "null_default")]
    pub title: String,
    #[serde(default, deserialize_with = "null_default")]
    pub content: String,
    #[serde(default, deserialize_with = "null_default")]
    pub description: String,
    #[serde(default, deserialize_with = "null_default")]
    pub image_url: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
    #[serde(default, deserialize_with = "null_default")]
    pub is_published: bool,
}

impl PageCreateRequest {
    pub fn validate(self) -> Result<CoreCreateRequest, PageApiError> {
        let mut issues = Vec::new();
        if let Some(message) = required_length_issue("Slug", &self.slug, 3, 100) {
            issues.push(("Slug", message));
        }
        if let Some(message) = required_length_issue("Title", &self.title, 3, 200) {
            issues.push(("Title", message));
        }
        if let Some(message) = required_length_issue("Content", &self.content, 10, usize::MAX) {
            issues.push(("Content", message));
        }
        if let Some(message) = maximum_issue("Description", &self.description, 500) {
            issues.push(("Description", message));
        }
        if !self.image_url.is_empty() && reqwest::Url::parse(&self.image_url).is_err() {
            issues.push(("ImageURL", "ImageURL must be a valid URL".to_owned()));
        }
        if !issues.is_empty() {
            return Err(PageApiError::validations(issues));
        }
        Ok(CoreCreateRequest {
            slug: self.slug,
            title: self.title,
            content: self.content,
            description: self.description,
            image_url: self.image_url,
            meta_data: self.meta_data,
            is_published: self.is_published,
        })
    }
}

#[derive(Debug, Default, Deserialize, ToSchema)]
pub struct PageUpdateRequest {
    pub title: Option<String>,
    pub content: Option<String>,
    pub description: Option<String>,
    pub image_url: Option<String>,
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub is_published: Option<bool>,
}

impl From<PageUpdateRequest> for CoreUpdateRequest {
    fn from(value: PageUpdateRequest) -> Self {
        Self {
            title: value.title,
            content: value.content,
            description: value.description,
            image_url: value.image_url,
            meta_data: value.meta_data,
            is_published: value.is_published,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PageResponse {
    pub id: Uuid,
    pub slug: String,
    pub title: String,
    pub content: String,
    pub description: String,
    pub image_url: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub is_published: bool,
    pub created_at: String,
    pub updated_at: String,
}

impl From<CorePage> for PageResponse {
    fn from(value: CorePage) -> Self {
        Self {
            id: value.id,
            slug: value.slug,
            title: value.title,
            content: value.content,
            description: value.description,
            image_url: value.image_url,
            meta_data: value.meta_data,
            is_published: value.is_published,
            created_at: timestamp(value.created_at),
            updated_at: timestamp(value.updated_at),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PageListResponse {
    pub pages: Vec<PageResponse>,
    pub total: i64,
    pub page: i64,
    pub per_page: i64,
    pub total_pages: i64,
}

impl From<CoreListResult> for PageListResponse {
    fn from(value: CoreListResult) -> Self {
        Self {
            pages: value.pages.into_iter().map(Into::into).collect(),
            total: value.total,
            page: value.page,
            per_page: value.per_page,
            total_pages: value.total_pages,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SuccessFlag {
    pub success: bool,
}

fn parse_i64(value: Option<&str>, fallback: i64) -> i64 {
    value
        .and_then(|value| value.parse().ok())
        .unwrap_or(fallback)
}

fn timestamp(value: Option<DateTime<Utc>>) -> String {
    value
        .map(|value| value.to_rfc3339_opts(SecondsFormat::AutoSi, true))
        .unwrap_or_else(|| "0001-01-01T00:00:00Z".to_owned())
}

fn required_length_issue(
    field: &'static str,
    value: &str,
    minimum: usize,
    maximum_length: usize,
) -> Option<String> {
    if value.is_empty() {
        return Some(format!("{field} is required"));
    }
    let length = value.chars().count();
    if length < minimum {
        return Some(format!("{field} must be at least {minimum} characters"));
    }
    maximum_issue(field, value, maximum_length)
}

fn maximum_issue(field: &'static str, value: &str, maximum: usize) -> Option<String> {
    if value.chars().count() > maximum {
        Some(format!("{field} must be at most {maximum} characters"))
    } else {
        None
    }
}

fn null_default<'de, D, T>(deserializer: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    Option::<T>::deserialize(deserializer).map(Option::unwrap_or_default)
}
