use std::sync::Arc;

use async_trait::async_trait;
use chrono::Utc;
use tokio_util::sync::CancellationToken;

use crate::error::AppError;

use uuid::Uuid;

use super::{
    FileData, FolderData, ObjectListing, UploadFile, UploadRepository, blurhash_from_bytes,
};

#[async_trait]
pub trait ObjectStore: Send + Sync {
    async fn list(&self, prefix: &str, delimiter: Option<&str>) -> Result<ObjectListing, AppError>;
    async fn put(&self, key: &str, data: Vec<u8>) -> Result<(), AppError>;
    async fn put_with_content_type(
        &self,
        key: &str,
        data: Vec<u8>,
        content_type: &str,
    ) -> Result<(), AppError> {
        let _ = content_type;
        self.put(key, data).await
    }
    async fn delete(&self, key: &str) -> Result<(), AppError>;
    async fn copy(&self, source_key: &str, destination_key: &str) -> Result<(), AppError>;
    async fn get(&self, key: &str) -> Result<Vec<u8>, AppError>;
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ListResult {
    pub files: Vec<FileData>,
    pub folders: Vec<FolderData>,
}

#[derive(Clone)]
pub struct RecordedUpload {
    pub id: Option<Uuid>,
    pub key: String,
    pub url: String,
    pub content_type: String,
    pub byte_size: i64,
    pub width: Option<i32>,
    pub height: Option<i32>,
    pub blurhash: Option<String>,
}

#[derive(Clone)]
pub struct StorageService {
    store: Arc<dyn ObjectStore>,
    uploads: Option<Arc<dyn UploadRepository>>,
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
            uploads: None,
            url_prefix: url_prefix.into(),
            cancellation,
        }
    }

    pub fn with_uploads(mut self, uploads: Arc<dyn UploadRepository>) -> Self {
        self.uploads = Some(uploads);
        self
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
                id: None,
                blurhash: None,
                width: None,
                height: None,
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
        if let Some(uploads) = &self.uploads {
            let keys = files
                .iter()
                .map(|file| file.key.clone())
                .collect::<Vec<_>>();
            if !keys.is_empty() {
                let records = uploads.find_by_keys(&keys).await?;
                for file in &mut files {
                    if let Some(record) = records.iter().find(|record| record.s3_key == file.key) {
                        file.id = Some(record.id);
                        file.blurhash.clone_from(&record.blurhash);
                        file.width = record.width;
                        file.height = record.height;
                    }
                }
            }
        }
        files.sort_unstable_by(|left, right| right.last_modified.cmp(&left.last_modified));
        folders.sort_unstable_by(|left, right| left.name.cmp(&right.name));
        Ok(ListResult { files, folders })
    }

    pub async fn upload_file(&self, key: &str, data: Vec<u8>) -> Result<(), AppError> {
        self.put_recorded(key, "application/octet-stream", data, None)
            .await
            .map(|_| ())
    }

    pub async fn put_recorded(
        &self,
        key: &str,
        content_type: &str,
        data: Vec<u8>,
        created_by: Option<Uuid>,
    ) -> Result<RecordedUpload, AppError> {
        let byte_size = i64::try_from(data.len()).unwrap_or(i64::MAX);
        let content_type = content_type_for(key, content_type);
        let (blurhash, width, height) = if is_raster_image(key, &content_type) {
            blurhash_from_bytes(&data)
                .map(|(hash, width, height)| (Some(hash), Some(width), Some(height)))
                .unwrap_or((None, None, None))
        } else {
            (None, None, None)
        };
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.store.put_with_content_type(key, data, &content_type) => result?,
        }
        let url = public_url(&self.url_prefix, key);
        let id = if key.is_empty() || key.ends_with('/') {
            None
        } else if let Some(uploads) = &self.uploads {
            let filename = filename_from_key(key);
            let saved = uploads
                .upsert(UploadFile {
                    id: Uuid::new_v4(),
                    s3_key: key.to_owned(),
                    public_url: url.clone(),
                    filename,
                    directory_path: directory_path(key),
                    content_type: content_type.clone(),
                    byte_size,
                    width,
                    height,
                    blurhash: blurhash.clone(),
                    created_by,
                })
                .await?;
            Some(saved.id)
        } else {
            None
        };
        Ok(RecordedUpload {
            id,
            key: key.to_owned(),
            url,
            content_type,
            byte_size,
            width,
            height,
            blurhash,
        })
    }

    pub async fn generate_blurhash(&self, key: &str) -> Result<RecordedUpload, AppError> {
        let key = key.trim();
        if key.is_empty() || key.ends_with('/') {
            return Err(AppError::InvalidInput("File key is required".to_owned()));
        }
        let content_type = content_type_for(key, "application/octet-stream");
        if !is_raster_image(key, &content_type) {
            return Err(AppError::InvalidInput(
                "Blurhash can only be generated for images".to_owned(),
            ));
        }
        let bytes = tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.store.get(key) => result?,
        };
        let (hash, width, height) = blurhash_from_bytes(&bytes)
            .ok_or_else(|| AppError::InvalidInput("Could not read this image".to_owned()))?;
        let byte_size = i64::try_from(bytes.len()).unwrap_or(i64::MAX);
        let url = public_url(&self.url_prefix, key);
        let existing = if let Some(uploads) = &self.uploads {
            uploads
                .find_by_keys(&[key.to_owned()])
                .await?
                .into_iter()
                .next()
        } else {
            None
        };
        let saved_content_type = existing
            .as_ref()
            .map(|file| file.content_type.clone())
            .unwrap_or_else(|| content_type.clone());
        let saved_size = existing
            .as_ref()
            .map(|file| file.byte_size)
            .unwrap_or(byte_size);
        let id = if let Some(uploads) = &self.uploads {
            let saved = uploads
                .upsert(UploadFile {
                    id: existing
                        .as_ref()
                        .map(|file| file.id)
                        .unwrap_or_else(Uuid::new_v4),
                    s3_key: key.to_owned(),
                    public_url: url.clone(),
                    filename: filename_from_key(key),
                    directory_path: directory_path(key),
                    content_type: saved_content_type.clone(),
                    byte_size: saved_size,
                    width: Some(width),
                    height: Some(height),
                    blurhash: Some(hash.clone()),
                    created_by: existing.and_then(|file| file.created_by),
                })
                .await?;
            uploads.attach_unlinked_images(&saved).await?;
            Some(saved.id)
        } else {
            None
        };
        Ok(RecordedUpload {
            id,
            key: key.to_owned(),
            url,
            content_type: saved_content_type,
            byte_size: saved_size,
            width: Some(width),
            height: Some(height),
            blurhash: Some(hash),
        })
    }

    pub async fn generate_blurhash_for_id(&self, id: Uuid) -> Result<RecordedUpload, AppError> {
        let uploads = self
            .uploads
            .as_ref()
            .ok_or_else(|| AppError::InvalidInput("image library is unavailable".to_owned()))?;
        let file = uploads.find_by_id(id).await?.ok_or(AppError::NotFound)?;
        self.generate_blurhash(&file.s3_key).await
    }

    pub async fn delete_file(&self, key: &str) -> Result<(), AppError> {
        if let Some(uploads) = &self.uploads
            && let Some(file) = uploads
                .find_by_keys(&[key.to_owned()])
                .await?
                .into_iter()
                .next()
            && uploads.is_referenced(file.id).await?
        {
            return Err(AppError::Conflict(
                "this file is still used by a page".to_owned(),
            ));
        }
        tokio::select! {
            biased;
            () = self.cancellation.cancelled() => return Err(AppError::Internal),
            result = self.store.delete(key) => result?,
        }
        if let Some(uploads) = &self.uploads {
            uploads.delete_by_key(key).await?;
        }
        Ok(())
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
        if let Some(uploads) = &self.uploads {
            uploads
                .rename_prefix(old_path, new_path, &self.url_prefix)
                .await?;
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

fn is_raster_image(key: &str, content_type: &str) -> bool {
    is_image_file(key) || (content_type.starts_with("image/") && content_type != "image/svg+xml")
}

fn content_type_for(key: &str, declared: &str) -> String {
    if !declared.is_empty() && declared != "application/octet-stream" {
        return declared.to_owned();
    }
    let lower = key.rsplit('.').next().unwrap_or("").to_ascii_lowercase();
    match lower.as_str() {
        "jpg" | "jpeg" => "image/jpeg".to_owned(),
        "png" => "image/png".to_owned(),
        "gif" => "image/gif".to_owned(),
        "webp" => "image/webp".to_owned(),
        "bmp" => "image/bmp".to_owned(),
        "svg" => "image/svg+xml".to_owned(),
        _ => {
            if declared.is_empty() {
                "application/octet-stream".to_owned()
            } else {
                declared.to_owned()
            }
        }
    }
}

pub fn public_url(prefix: &str, key: &str) -> String {
    format!(
        "{}/{}",
        prefix.trim_end_matches('/'),
        key.trim_start_matches('/')
    )
}

fn filename_from_key(key: &str) -> String {
    key.rsplit('/')
        .next()
        .filter(|name| !name.is_empty())
        .unwrap_or(key)
        .to_owned()
}

fn directory_path(key: &str) -> String {
    match key.rsplit_once('/') {
        Some((directory, file)) if !directory.is_empty() && !file.is_empty() => {
            directory.to_owned()
        }
        _ => String::new(),
    }
}
