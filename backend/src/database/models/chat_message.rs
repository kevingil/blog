use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::chat_message;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = chat_message)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ChatMessageRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub role: String,
    pub content: String,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = chat_message)]
pub struct NewChatMessageRow {
    pub id: Uuid,
    pub article_id: Uuid,
    pub role: String,
    pub content: String,
    pub meta_data: Value,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = chat_message)]
pub struct ChatMessageChangeset {
    pub article_id: Uuid,
    pub role: String,
    pub content: String,
    pub meta_data: Option<Option<Value>>,
    pub created_at: Option<Option<DateTime<Utc>>>,
}
