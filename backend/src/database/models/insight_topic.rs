use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use pgvector::Vector;
use serde_json::Value;
use uuid::Uuid;

use crate::schema::insight_topic;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = insight_topic)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct InsightTopicRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub description: Option<String>,
    pub keywords: Option<Value>,
    pub embedding: Option<Vector>,
    pub is_auto_generated: Option<bool>,
    pub content_count: Option<i32>,
    pub last_insight_at: Option<DateTime<Utc>>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = insight_topic)]
pub struct NewInsightTopicRow {
    pub id: Uuid,
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub description: Option<String>,
    pub keywords: Value,
    pub embedding: Option<Vector>,
    pub is_auto_generated: bool,
    pub content_count: i32,
    pub last_insight_at: Option<DateTime<Utc>>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = insight_topic)]
#[diesel(treat_none_as_null = true)]
pub struct InsightTopicChangeset {
    pub organization_id: Option<Uuid>,
    pub name: String,
    pub description: Option<String>,
    pub keywords: Option<Value>,
    pub embedding: Option<Vector>,
    pub is_auto_generated: Option<bool>,
    pub content_count: Option<i32>,
    pub last_insight_at: Option<DateTime<Utc>>,
    pub color: Option<String>,
    pub icon: Option<String>,
    pub updated_at: Option<DateTime<Utc>>,
}
