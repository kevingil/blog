use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use pgvector::Vector;
use serde_json::Value;
use uuid::Uuid;

use crate::schema::article_source;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = article_source)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct SourceRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: Option<String>,
    pub content: String,
    pub url: Option<String>,
    pub source_type: Option<String>,
    pub embedding: Option<Vector>,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = article_source)]
pub struct NewSourceRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub title: Option<String>,
    pub content: String,
    pub url: Option<String>,
    pub source_type: Option<String>,
    pub embedding: Option<Vector>,
    pub meta_data: Value,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = article_source)]
#[diesel(treat_none_as_null = true)]
pub struct SourceChangeset {
    pub article_id: Uuid,
    pub title: Option<String>,
    pub content: String,
    pub url: Option<String>,
    pub source_type: Option<String>,
    pub embedding: Option<Vector>,
    pub meta_data: Option<Value>,
}
