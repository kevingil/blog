use chrono::{DateTime, SecondsFormat, Utc};
use serde::{Deserialize, Deserializer, Serialize};
use utoipa::{IntoParams, ToSchema};
use uuid::Uuid;

use crate::core::project::{
    Project as CoreProject, ProjectCreateRequest as CoreCreateRequest,
    ProjectDetail as CoreProjectDetail, ProjectListResult as CoreListResult,
    ProjectUpdateRequest as CoreUpdateRequest,
};

use super::error::ProjectApiError;

#[derive(Debug, Default, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
#[serde(rename_all = "camelCase")]
pub struct ProjectListQuery {
    #[param(default = 1)]
    #[param(value_type = Option<i64>)]
    pub page: Option<String>,
    #[param(default = 20)]
    #[param(value_type = Option<i64>)]
    pub per_page: Option<String>,
}

impl ProjectListQuery {
    pub fn values(&self) -> (i64, i64) {
        (
            parse_i64(self.page.as_deref(), 1),
            parse_i64(self.per_page.as_deref(), 20),
        )
    }
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct ProjectCreateRequest {
    #[serde(default, deserialize_with = "null_default")]
    pub title: String,
    #[serde(default, deserialize_with = "null_default")]
    pub description: String,
    #[serde(default, deserialize_with = "null_default")]
    pub content: String,
    #[serde(default, deserialize_with = "null_default")]
    pub tags: Vec<String>,
    #[serde(default, deserialize_with = "null_default")]
    pub image_url: String,
    #[serde(default, deserialize_with = "null_default")]
    pub url: String,
}

impl ProjectCreateRequest {
    pub fn validate(self) -> Result<CoreCreateRequest, ProjectApiError> {
        let mut issues = Vec::new();
        if let Some(message) = required_length_issue("Title", &self.title, 1, 200) {
            issues.push(("Title", message));
        }
        if let Some(message) = required_length_issue("Description", &self.description, 1, 500) {
            issues.push(("Description", message));
        }
        if let Some(message) = tags_issue(&self.tags) {
            issues.push(("Tags", message));
        }
        if let Some(message) = optional_url_issue("ImageURL", &self.image_url) {
            issues.push(("ImageURL", message));
        }
        if let Some(message) = optional_url_issue("URL", &self.url) {
            issues.push(("URL", message));
        }
        if !issues.is_empty() {
            return Err(ProjectApiError::validations(issues));
        }
        Ok(CoreCreateRequest {
            title: self.title,
            description: self.description,
            content: self.content,
            tags: self.tags,
            image_url: self.image_url,
            url: self.url,
        })
    }
}

#[derive(Debug, Default, Deserialize, ToSchema)]
pub struct ProjectUpdateRequest {
    pub title: Option<String>,
    pub description: Option<String>,
    pub content: Option<String>,
    pub tags: Option<Vec<String>>,
    pub image_url: Option<String>,
    pub url: Option<String>,
}

impl ProjectUpdateRequest {
    pub fn validate(self) -> Result<CoreUpdateRequest, ProjectApiError> {
        let mut issues = Vec::new();
        if let Some(title) = self.title.as_deref()
            && let Some(message) = required_length_issue("Title", title, 1, 200)
        {
            issues.push(("Title", message));
        }
        if let Some(description) = self.description.as_deref()
            && let Some(message) = required_length_issue("Description", description, 1, 500)
        {
            issues.push(("Description", message));
        }
        if let Some(tags) = self.tags.as_deref()
            && let Some(message) = tags_issue(tags)
        {
            issues.push(("Tags", message));
        }
        if let Some(image_url) = self.image_url.as_deref()
            && let Some(message) = optional_url_issue("ImageURL", image_url)
        {
            issues.push(("ImageURL", message));
        }
        if let Some(url) = self.url.as_deref()
            && let Some(message) = optional_url_issue("URL", url)
        {
            issues.push(("URL", message));
        }
        if !issues.is_empty() {
            return Err(ProjectApiError::validations(issues));
        }
        Ok(CoreUpdateRequest {
            title: self.title,
            description: self.description,
            content: self.content,
            tags: self.tags,
            image_url: self.image_url,
            url: self.url,
        })
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ProjectResponse {
    pub id: Uuid,
    pub title: String,
    pub description: String,
    pub content: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub tag_ids: Vec<i64>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub image_url: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub url: String,
    pub created_at: String,
    pub updated_at: String,
}

impl From<CoreProject> for ProjectResponse {
    fn from(value: CoreProject) -> Self {
        Self {
            id: value.id,
            title: value.title,
            description: value.description,
            content: value.content,
            tag_ids: value.tag_ids,
            image_url: value.image_url,
            url: value.url,
            created_at: timestamp(value.created_at),
            updated_at: timestamp(value.updated_at),
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ProjectListResponse {
    pub projects: Vec<ProjectResponse>,
    pub total: i64,
    pub current_page: i64,
    pub per_page: i64,
}

impl From<CoreListResult> for ProjectListResponse {
    fn from(value: CoreListResult) -> Self {
        Self {
            projects: value.projects.into_iter().map(Into::into).collect(),
            total: value.total,
            current_page: value.page,
            per_page: value.per_page,
        }
    }
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ProjectDetailResponse {
    pub project: ProjectResponse,
    pub tags: Vec<String>,
}

impl From<CoreProjectDetail> for ProjectDetailResponse {
    fn from(value: CoreProjectDetail) -> Self {
        Self {
            project: value.project.into(),
            tags: value.tags,
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
    maximum: usize,
) -> Option<String> {
    if value.is_empty() {
        return Some(format!("{field} is required"));
    }
    let length = value.chars().count();
    if length < minimum {
        return Some(format!("{field} must be at least {minimum} characters"));
    }
    if length > maximum {
        return Some(format!("{field} must be at most {maximum} characters"));
    }
    None
}

fn tags_issue(tags: &[String]) -> Option<String> {
    for tag in tags {
        let length = tag.chars().count();
        if length < 1 {
            return Some("Tags must be at least 1 characters".to_owned());
        }
        if length > 50 {
            return Some("Tags must be at most 50 characters".to_owned());
        }
    }
    None
}

fn optional_url_issue(field: &'static str, value: &str) -> Option<String> {
    if !value.is_empty() && reqwest::Url::parse(value).is_err() {
        Some(format!("{field} must be a valid URL"))
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
