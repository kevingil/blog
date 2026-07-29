use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use pgvector::Vector;
use serde_json::Value;
use uuid::Uuid;

use crate::schema::crawled_content;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = crawled_content)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct CrawledContentRow {
    pub id: Uuid,
    pub data_source_id: Uuid,
    pub url: String,
    pub title: Option<String>,
    pub content: String,
    pub summary: Option<String>,
    pub author: Option<String>,
    pub published_at: Option<DateTime<Utc>>,
    pub embedding: Option<Vector>,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = crawled_content)]
pub struct NewCrawledContentRow {
    pub id: Uuid,
    pub data_source_id: Uuid,
    pub url: String,
    pub title: Option<String>,
    pub content: String,
    pub summary: Option<String>,
    pub author: Option<String>,
    pub published_at: Option<DateTime<Utc>>,
    pub embedding: Option<Vector>,
    pub meta_data: Value,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = crawled_content)]
#[diesel(treat_none_as_null = true)]
pub struct CrawledContentChangeset {
    pub data_source_id: Uuid,
    pub url: String,
    pub title: Option<String>,
    pub content: String,
    pub summary: Option<String>,
    pub author: Option<String>,
    pub published_at: Option<DateTime<Utc>>,
    pub embedding: Option<Vector>,
    pub meta_data: Option<Value>,
}
