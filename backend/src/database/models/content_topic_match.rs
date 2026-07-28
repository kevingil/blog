use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::schema::content_topic_match;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = content_topic_match)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ContentTopicMatchRow {
    pub id: Uuid,
    pub content_id: Uuid,
    pub topic_id: Uuid,
    pub similarity_score: f64,
    pub is_primary: Option<bool>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = content_topic_match)]
pub struct NewContentTopicMatchRow {
    pub id: Uuid,
    pub content_id: Uuid,
    pub topic_id: Uuid,
    pub similarity_score: f64,
    pub is_primary: bool,
    pub created_at: DateTime<Utc>,
}
