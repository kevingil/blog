use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::core::storage::ImageAsset;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Project {
    pub id: Uuid,
    pub title: String,
    pub description: String,
    pub content: String,
    #[serde(skip_serializing_if = "Vec::is_empty", default)]
    pub tag_ids: Vec<i64>,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub image_url: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub url: String,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub upload_file_id: Option<Uuid>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub image: Option<ImageAsset>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ProjectListOptions {
    pub page: i64,
    pub per_page: i64,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProjectCreateRequest {
    pub title: String,
    pub description: String,
    #[serde(default)]
    pub content: String,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(default)]
    pub image_url: String,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub image_upload_id: Option<Uuid>,
}

#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProjectUpdateRequest {
    pub title: Option<String>,
    pub description: Option<String>,
    pub content: Option<String>,
    pub tags: Option<Vec<String>>,
    pub image_url: Option<String>,
    pub url: Option<String>,
    #[serde(default)]
    pub image_upload_id: Option<Uuid>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProjectListResult {
    pub projects: Vec<Project>,
    pub total: i64,
    pub page: i64,
    pub per_page: i64,
    pub total_pages: i64,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProjectDetail {
    pub project: Project,
    pub tags: Vec<String>,
}
