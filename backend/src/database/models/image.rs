use chrono::{DateTime, Utc};
use diesel::{AsChangeset, Identifiable, Insertable, Queryable, Selectable};
use serde_json::Value;
use uuid::Uuid;

use crate::schema::imagen_request;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = imagen_request)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct ImageRow {
    pub id: Uuid,
    pub prompt: String,
    pub provider: String,
    pub model_name: String,
    pub request_id: Option<String>,
    pub status: Option<String>,
    pub output_url: Option<String>,
    pub file_index_id: Option<Uuid>,
    pub error_message: Option<String>,
    pub meta_data: Option<Value>,
    pub created_at: Option<DateTime<Utc>>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = imagen_request)]
pub struct NewImageRow {
    pub id: Uuid,
    pub prompt: String,
    pub provider: String,
    pub model_name: String,
    pub request_id: Option<String>,
    pub status: String,
    pub output_url: Option<String>,
    pub file_index_id: Option<Uuid>,
    pub error_message: Option<String>,
    pub meta_data: Value,
    pub created_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, AsChangeset)]
#[diesel(table_name = imagen_request)]
#[diesel(treat_none_as_null = true)]
pub struct ImageChangeset {
    pub prompt: String,
    pub provider: String,
    pub model_name: String,
    pub request_id: Option<String>,
    pub status: Option<String>,
    pub output_url: Option<String>,
    pub file_index_id: Option<Uuid>,
    pub error_message: Option<String>,
    pub meta_data: Option<Value>,
    pub completed_at: Option<DateTime<Utc>>,
}
