use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use pgvector::Vector;
use serde_json::Value;
use uuid::Uuid;

use crate::schema::{article, article_version};

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = article)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ArticleRow {
    pub id: Uuid,
    pub slug: String,
    pub author_id: Uuid,
    pub tag_ids: Option<Vec<Option<i32>>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub published_at: Option<DateTime<Utc>>,
    pub imagen_request_id: Option<Uuid>,
    pub session_memory: Option<Value>,
    pub draft_title: Option<String>,
    pub draft_content: Option<String>,
    pub draft_image_url: Option<String>,
    pub draft_embedding: Option<Vector>,
    pub published_title: Option<String>,
    pub published_content: Option<String>,
    pub published_image_url: Option<String>,
    pub published_embedding: Option<Vector>,
    pub current_draft_version_id: Option<Uuid>,
    pub current_published_version_id: Option<Uuid>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = article)]
pub struct NewArticleRow {
    pub id: Uuid,
    pub slug: String,
    pub author_id: Uuid,
    pub tag_ids: Option<Option<Vec<Option<i32>>>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub published_at: Option<DateTime<Utc>>,
    pub imagen_request_id: Option<Uuid>,
    pub session_memory: Value,
    pub draft_title: String,
    pub draft_content: String,
    pub draft_image_url: String,
    pub draft_embedding: Option<Vector>,
    pub published_title: Option<String>,
    pub published_content: Option<String>,
    pub published_image_url: Option<String>,
    pub published_embedding: Option<Vector>,
    pub current_draft_version_id: Option<Uuid>,
    pub current_published_version_id: Option<Uuid>,
}

#[derive(Debug, AsChangeset)]
#[diesel(table_name = article)]
pub struct ArticleChangeset {
    pub slug: String,
    pub author_id: Uuid,
    pub tag_ids: Option<Option<Vec<Option<i32>>>>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: DateTime<Utc>,
    pub published_at: Option<Option<DateTime<Utc>>>,
    pub imagen_request_id: Option<Option<Uuid>>,
    pub session_memory: Option<Option<Value>>,
    pub draft_title: String,
    pub draft_content: String,
    pub draft_image_url: String,
    pub draft_embedding: Option<Option<Vector>>,
    pub published_title: Option<Option<String>>,
    pub published_content: Option<Option<String>>,
    pub published_image_url: Option<Option<String>>,
    pub published_embedding: Option<Option<Vector>>,
    pub current_draft_version_id: Option<Option<Uuid>>,
    pub current_published_version_id: Option<Option<Uuid>>,
}

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = article_version)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ArticleVersionRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub version_number: i32,
    pub status: String,
    pub title: String,
    pub content: Option<String>,
    pub image_url: Option<String>,
    pub embedding: Option<Vector>,
    pub edited_by: Option<Uuid>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = article_version)]
pub struct NewArticleVersionRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub version_number: i32,
    pub status: String,
    pub title: String,
    pub content: Option<String>,
    pub image_url: Option<String>,
    pub embedding: Option<Vector>,
    pub edited_by: Option<Uuid>,
    pub created_at: DateTime<Utc>,
}
