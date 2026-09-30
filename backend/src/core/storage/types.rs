use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use utoipa::ToSchema;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize, ToSchema)]
pub struct ImageAsset {
    pub id: Uuid,
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub blurhash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UploadFile {
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

impl UploadFile {
    pub fn asset(&self) -> ImageAsset {
        ImageAsset {
            id: self.id,
            url: self.public_url.clone(),
            blurhash: self.blurhash.clone(),
            width: self.width,
            height: self.height,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct FileData {
    pub key: String,
    pub last_modified: DateTime<Utc>,
    pub size: String,
    pub size_raw: i64,
    pub url: String,
    pub is_image: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub blurhash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
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
