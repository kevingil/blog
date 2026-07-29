use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::data_source;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = data_source)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct DataSourceRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub url: String,
    pub feed_url: Option<String>,
    pub source_type: Option<String>,
    pub crawl_frequency: Option<String>,
    pub is_enabled: Option<bool>,
    pub is_discovered: Option<bool>,
    pub discovered_from_id: Option<Uuid>,
    pub last_crawled_at: Option<DateTime<Utc>>,
    pub next_crawl_at: Option<DateTime<Utc>>,
    pub crawl_status: Option<String>,
    pub error_message: Option<String>,
    pub content_count: Option<i32>,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub user_id: Option<Uuid>,
    pub subscriber_count: Option<i32>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = data_source)]
pub struct NewDataSourceRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub url: String,
    pub feed_url: Option<String>,
    pub source_type: String,
    pub crawl_frequency: String,
    pub is_enabled: bool,
    pub is_discovered: bool,
    pub discovered_from_id: Option<Uuid>,
    pub last_crawled_at: Option<DateTime<Utc>>,
    pub next_crawl_at: Option<DateTime<Utc>>,
    pub crawl_status: String,
    pub error_message: Option<String>,
    pub content_count: i32,
    pub meta_data: Value,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub user_id: Option<Uuid>,
    pub subscriber_count: i32,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = data_source)]
#[diesel(treat_none_as_null = true)]
pub struct DataSourceChangeset {
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub url: String,
    pub feed_url: Option<String>,
    pub source_type: Option<String>,
    pub crawl_frequency: Option<String>,
    pub is_enabled: Option<bool>,
    pub is_discovered: Option<bool>,
    pub discovered_from_id: Option<Uuid>,
    pub last_crawled_at: Option<DateTime<Utc>>,
    pub next_crawl_at: Option<DateTime<Utc>>,
    pub crawl_status: Option<String>,
    pub error_message: Option<String>,
    pub content_count: Option<i32>,
    pub meta_data: Option<Value>,
    pub updated_at: Option<DateTime<Utc>>,
    pub user_id: Option<Uuid>,
    pub subscriber_count: Option<i32>,
}
