use std::sync::Arc;

use async_trait::async_trait;
use chrono::Utc;
use tokio_util::sync::CancellationToken;

use crate::error::AppError;

use super::{FileData, FolderData, ObjectListing};

#[async_trait]
pub trait ObjectStore: Send + Sync {
    async fn list(&self, prefix: &str, delimiter: Option<&str>) -> Result<ObjectListing, AppError>;
    async fn put(&self, key: &str, data: Vec<u8>) -> Result<(), AppError>;
    async fn delete(&self, key: &str) -> Result<(), AppError>;
    async fn copy(&self, source_key: &str, destination_key: &str) -> Result<(), AppError>;
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ListResult {
    pub files: Vec<FileData>,
    pub folders: Vec<FolderData>,
}

#[derive(Clone)]
pub struct StorageService {
    store: Arc<dyn ObjectStore>,
    url_prefix: Arc<str>,
    cancellation: CancellationToken,
}

impl StorageService {
    pub fn new(
        store: Arc<dyn ObjectStore>,
        url_prefix: impl Into<Arc<str>>,
        cancellation: CancellationToken,
    ) -> Self {
        Self {
            store,
            url_prefix: url_prefix.into(),
            cancellation,
        }
    }

    pub async fn list_files(&self, prefix: &str) -> Result<ListResult, AppError> {
        let listing = tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.store.list(prefix, Some("/")) => result?,
        };
        let mut files = listing
            .objects
            .into_iter()
            .map(|object| FileData {
                url: format!("{}/{}", self.url_prefix, object.key),
                is_image: is_image_file(&object.key),
                size: format_byte_size(object.size),
                size_raw: object.size,
                key: object.key,
                last_modified: object.last_modified,
            })
            .collect::<Vec<_>>();
        let mut folders = listing
            .common_prefixes
            .into_iter()
            .map(|path| {
                let without_trailing_slash = path.strip_suffix('/').unwrap_or(&path);
                let name = without_trailing_slash
                    .rsplit_once('/')
                    .map(|(_, name)| name)
                    .unwrap_or(without_trailing_slash)
                    .to_owned();
                FolderData {
                    is_hidden: name.starts_with('.'),
                    name,
                    path,
                    last_modified: Utc::now(),
                    file_count: 0,
                }
            })
            .collect::<Vec<_>>();
        files.sort_unstable_by(|left, right| right.last_modified.cmp(&left.last_modified));
        folders.sort_unstable_by(|left, right| left.name.cmp(&right.name));
        Ok(ListResult { files, folders })
    }

    pub async fn upload_file(&self, key: &str, data: Vec<u8>) -> Result<(), AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.store.put(key, data) => result,
        }
    }

    pub async fn delete_file(&self, key: &str) -> Result<(), AppError> {
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => Err(AppError::Internal),
            result = self.store.delete(key) => result,
        }
    }

    pub async fn create_folder(&self, path: &str) -> Result<(), AppError> {
        let key = if path.ends_with('/') {
            path.to_owned()
        } else {
            format!("{path}/")
        };
        self.upload_file(&key, Vec::new()).await
    }

    pub async fn update_folder(&self, old_path: &str, new_path: &str) -> Result<(), AppError> {
        let listing = tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.store.list(old_path, None) => result?,
        };
        for object in listing.objects {
            if self.cancellation.is_cancelled() {
                return Err(AppError::Internal);
            }
            let new_key = object.key.replacen(old_path, new_path, 1);
            tokio::select! {
                biased;
                () = self.cancellation.cancelled() => return Err(AppError::Internal),
                result = self.store.copy(&object.key, &new_key) => result?,
            }
            tokio::select! {
                biased;
                () = self.cancellation.cancelled() => return Err(AppError::Internal),
                result = self.store.delete(&object.key) => result?,
            }
        }
        Ok(())
    }

    pub fn url_prefix(&self) -> &str {
        &self.url_prefix
    }
}

fn format_byte_size(mut size: i64) -> String {
    const UNITS: [&str; 9] = ["B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"];
    let mut unit = 0;
    while size >= 1024 && unit < UNITS.len() - 1 {
        size /= 1024;
        unit += 1;
    }
    format!("{:.2} {}", size as f64, UNITS[unit])
}

fn is_image_file(key: &str) -> bool {
    [".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"]
        .iter()
        .any(|extension| key.ends_with(extension))
}
