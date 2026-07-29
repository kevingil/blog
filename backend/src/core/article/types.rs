use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::ToSchema;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct Article {
    pub id: Uuid,
    pub slug: String,
    pub author_id: Uuid,
    pub tag_ids: Option<Vec<i64>>,
    pub draft_title: String,
    pub draft_content: String,
    pub draft_image_url: String,
    pub draft_embedding: Vec<f32>,
    pub published_title: Option<String>,
    pub published_content: Option<String>,
    pub published_image_url: Option<String>,
    pub published_embedding: Vec<f32>,
    pub published_at: Option<DateTime<Utc>>,
    pub current_draft_version_id: Option<Uuid>,
    pub current_published_version_id: Option<Uuid>,
    pub imagen_request_id: Option<Uuid>,
    pub session_memory: Option<BTreeMap<String, Value>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

impl Article {
    pub fn is_published(&self) -> bool {
        self.published_at.is_some()
    }

    pub fn title(&self, for_editing: bool) -> &str {
        if for_editing {
            &self.draft_title
        } else {
            self.published_title.as_deref().unwrap_or(&self.draft_title)
        }
    }

    pub fn content(&self, for_editing: bool) -> &str {
        if for_editing {
            &self.draft_content
        } else {
            self.published_content
                .as_deref()
                .unwrap_or(&self.draft_content)
        }
    }

    pub fn image_url(&self, for_editing: bool) -> &str {
        if for_editing {
            &self.draft_image_url
        } else {
            self.published_image_url
                .as_deref()
                .unwrap_or(&self.draft_image_url)
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize, ToSchema)]
pub struct ArticleVersion {
    pub id: Uuid,
    pub article_id: Uuid,
    pub version_number: i32,
    pub status: String,
    pub title: String,
    pub content: String,
    pub image_url: String,
    pub embedding: Vec<f32>,
    pub edited_by: Option<Uuid>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ArticleSearchOptions {
    pub query: String,
    pub page: i64,
    pub per_page: i64,
    pub published_only: bool,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ArticleListOptions {
    pub page: i64,
    pub per_page: i64,
    pub published_only: bool,
    pub author_id: Option<Uuid>,
    pub tag_id: Option<i64>,
    pub sort_by: String,
    pub sort_order: String,
}
