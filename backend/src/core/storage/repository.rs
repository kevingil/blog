use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{ImageAsset, UploadFile};

#[async_trait]
pub trait UploadRepository: Send + Sync {
    async fn upsert(&self, file: UploadFile) -> Result<UploadFile, AppError>;
    async fn find_by_id(&self, id: Uuid) -> Result<Option<UploadFile>, AppError>;
    async fn find_by_ids(&self, ids: &[Uuid]) -> Result<Vec<UploadFile>, AppError>;
    async fn find_by_keys(&self, keys: &[String]) -> Result<Vec<UploadFile>, AppError>;
    async fn find_by_public_url(&self, url: &str) -> Result<Option<UploadFile>, AppError>;
    async fn find_by_public_urls(&self, urls: &[String]) -> Result<Vec<UploadFile>, AppError>;
    async fn delete_by_key(&self, key: &str) -> Result<(), AppError>;
    async fn is_referenced(&self, id: Uuid) -> Result<bool, AppError>;
    async fn rename_prefix(
        &self,
        old_prefix: &str,
        new_prefix: &str,
        url_prefix: &str,
    ) -> Result<(), AppError>;
    async fn replace_body_refs(
        &self,
        owner_kind: &str,
        owner_id: Uuid,
        upload_ids: &[Uuid],
    ) -> Result<(), AppError>;
    /// Point articles that already use this image URL at the upload row.
    ///
    /// Older articles stored only `draft_image_url` / `published_image_url`.
    /// Generating a blurhash has to record the upload id or the hash disappears
    /// on the next load.
    async fn attach_unlinked_images(&self, file: &UploadFile) -> Result<(), AppError>;
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ResolvedImage {
    pub url: String,
    pub upload_file_id: Option<Uuid>,
    pub image: Option<ImageAsset>,
}

pub async fn resolve_image(
    uploads: Option<&dyn UploadRepository>,
    selected_id: Option<Uuid>,
    requested_url: &str,
    current_url: &str,
    current_id: Option<Uuid>,
) -> Result<ResolvedImage, AppError> {
    if let Some(id) = selected_id {
        let uploads = uploads
            .ok_or_else(|| AppError::InvalidInput("image library is unavailable".to_owned()))?;
        let file = uploads
            .find_by_id(id)
            .await?
            .ok_or_else(|| AppError::InvalidInput("image was not found".to_owned()))?;
        return Ok(ResolvedImage {
            url: file.public_url.clone(),
            upload_file_id: Some(file.id),
            image: Some(file.asset()),
        });
    }
    let requested_url = requested_url.trim();
    if requested_url.is_empty() {
        return Ok(ResolvedImage {
            url: String::new(),
            upload_file_id: None,
            image: None,
        });
    }
    if let Some(uploads) = uploads
        && let Some(file) = uploads.find_by_public_url(requested_url).await?
    {
        return Ok(ResolvedImage {
            url: file.public_url.clone(),
            upload_file_id: Some(file.id),
            image: Some(file.asset()),
        });
    }
    let upload_file_id = (requested_url == current_url)
        .then_some(current_id)
        .flatten();
    Ok(ResolvedImage {
        url: requested_url.to_owned(),
        upload_file_id,
        image: None,
    })
}
