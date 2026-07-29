use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct FileData {
    pub key: String,
    pub last_modified: DateTime<Utc>,
    pub size: String,
    pub size_raw: i64,
    pub url: String,
    pub is_image: bool,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct FolderData {
    pub name: String,
    pub path: String,
    pub is_hidden: bool,
    pub last_modified: DateTime<Utc>,
    pub file_count: i32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ObjectEntry {
    pub key: String,
    pub last_modified: DateTime<Utc>,
    pub size: i64,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ObjectListing {
    pub objects: Vec<ObjectEntry>,
    pub common_prefixes: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct FileIndex {
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
