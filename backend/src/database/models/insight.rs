use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use pgvector::Vector;
use serde_json::Value;
use uuid::Uuid;

use crate::schema::insight;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = insight)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct InsightRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    pub content: Option<String>,
    pub key_points: Option<Value>,
    pub source_content_ids: Option<Vec<Option<Uuid>>>,
    pub embedding: Option<Vector>,
    pub generated_at: Option<DateTime<Utc>>,
    pub period_start: Option<DateTime<Utc>>,
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: Option<bool>,
    pub is_pinned: Option<bool>,
    pub is_used_in_article: Option<bool>,
    pub meta_data: Option<Value>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = insight)]
pub struct NewInsightRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    pub content: Option<String>,
    pub key_points: Value,
    pub source_content_ids: Vec<Option<Uuid>>,
    pub embedding: Option<Vector>,
    pub generated_at: DateTime<Utc>,
    pub period_start: Option<DateTime<Utc>>,
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: bool,
    pub is_pinned: bool,
    pub is_used_in_article: bool,
    pub meta_data: Value,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = insight)]
#[diesel(treat_none_as_null = true)]
pub struct InsightChangeset {
    pub organization_id: Option<Uuid>,
    pub topic_id: Option<Uuid>,
    pub title: String,
    pub summary: String,
    pub content: Option<String>,
    pub key_points: Option<Value>,
    pub source_content_ids: Option<Vec<Option<Uuid>>>,
    pub embedding: Option<Vector>,
    pub generated_at: Option<DateTime<Utc>>,
    pub period_start: Option<DateTime<Utc>>,
    pub period_end: Option<DateTime<Utc>>,
    pub is_read: Option<bool>,
    pub is_pinned: Option<bool>,
    pub is_used_in_article: Option<bool>,
    pub meta_data: Option<Value>,
}
