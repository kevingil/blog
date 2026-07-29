use serde::{Deserialize, Serialize};
use utoipa::{IntoParams, ToSchema};

use crate::core::storage::{FileData, FolderData};

#[derive(Debug, Clone, Default, Deserialize, IntoParams)]
#[into_params(parameter_in = Query)]
pub struct ListFilesQuery {
    pub prefix: Option<String>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct ListFilesResponse {
    pub files: Vec<FileDataResponse>,
    pub folders: Vec<FolderDataResponse>,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct FileDataResponse {
    pub key: String,
    pub last_modified: chrono::DateTime<chrono::Utc>,
    pub size: String,
    pub size_raw: i64,
    pub url: String,
    pub is_image: bool,
}

impl From<FileData> for FileDataResponse {
    fn from(file: FileData) -> Self {
        Self {
            key: file.key,
            last_modified: file.last_modified,
            size: file.size,
            size_raw: file.size_raw,
            url: file.url,
            is_image: file.is_image,
        }
    }
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct FolderDataResponse {
    pub name: String,
    pub path: String,
    pub is_hidden: bool,
    pub last_modified: chrono::DateTime<chrono::Utc>,
    pub file_count: i32,
}

impl From<FolderData> for FolderDataResponse {
    fn from(folder: FolderData) -> Self {
        Self {
            name: folder.name,
            path: folder.path,
            is_hidden: folder.is_hidden,
            last_modified: folder.last_modified,
            file_count: folder.file_count,
        }
    }
}

#[derive(Debug, Clone, ToSchema)]
pub struct UploadFileRequest {
    pub key: String,
    #[schema(value_type = String, format = Binary)]
    pub file: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct UploadFileResponse {
    pub success: bool,
    pub url: String,
    pub key: String,
}

#[derive(Debug, Clone, Deserialize, ToSchema)]
pub struct CreateFolderRequest {
    pub path: String,
}

#[derive(Debug, Clone, Deserialize, ToSchema)]
#[serde(rename_all = "camelCase")]
pub struct UpdateFolderRequest {
    pub old_path: String,
    pub new_path: String,
}

#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct SuccessFlagResponse {
    pub success: bool,
}
