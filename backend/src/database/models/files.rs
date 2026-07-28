use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::file_index;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = file_index)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct FileIndexRow {
    pub id: Uuid,
    pub s3_key: String,
    pub filename: String,
    pub directory_path: Option<String>,
    pub file_type: Option<String>,
    pub file_size: Option<i64>,
    pub content_type: Option<String>,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = file_index)]
pub struct NewFileIndexRow {
    pub id: Uuid,
    pub s3_key: String,
    pub filename: String,
    pub directory_path: Option<String>,
    pub file_type: Option<String>,
    pub file_size: Option<i64>,
    pub content_type: Option<String>,
    pub meta_data: Option<Value>,
}
