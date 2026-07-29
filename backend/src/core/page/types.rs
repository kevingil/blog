use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Page {
    pub id: Uuid,
    pub slug: String,
    pub title: String,
    pub content: String,
    pub description: String,
    pub image_url: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub is_published: bool,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PageListOptions {
    pub page: i64,
    pub per_page: i64,
    pub is_published: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PageCreateRequest {
    pub slug: String,
    pub title: String,
    pub content: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub image_url: String,
    pub meta_data: Option<BTreeMap<String, Value>>,
    #[serde(default)]
    pub is_published: bool,
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct PageUpdateRequest {
    pub title: Option<String>,
    pub content: Option<String>,
    pub description: Option<String>,
    pub image_url: Option<String>,
    pub meta_data: Option<BTreeMap<String, Value>>,
    pub is_published: Option<bool>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PageListResult {
    pub pages: Vec<Page>,
    pub total: i64,
    pub page: i64,
    pub per_page: i64,
    pub total_pages: i64,
}
