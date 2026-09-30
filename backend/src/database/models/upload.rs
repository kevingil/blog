use chrono::{DateTime, Utc};
use diesel::{Identifiable, Insertable, Queryable, Selectable};
use uuid::Uuid;

use crate::schema::upload_files;

#[derive(Debug, Clone, Queryable, Selectable, Identifiable)]
#[diesel(table_name = upload_files)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct UploadFileRow {
    pub id: Uuid,
    pub s3_key: String,
    pub public_url: String,
    pub filename: String,
    pub directory_path: String,
    pub content_type: String,
    pub byte_size: i64,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub blurhash: Option<String>,
    pub created_by: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Insertable)]
#[diesel(table_name = upload_files)]
pub struct NewUploadFileRow {
    pub id: Uuid,
    pub s3_key: String,
    pub public_url: String,
    pub filename: String,
    pub directory_path: String,
    pub content_type: String,
    pub byte_size: i64,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub blurhash: Option<String>,
    pub created_by: Option<Uuid>,
}
