use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Utc};

use crate::{core::storage::StorageService, error::AppError};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExternalImage {
    pub bytes: Vec<u8>,
    pub content_type: String,
    pub extension: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExternalPage {
    pub url: String,
    pub title: String,
    pub excerpt: String,
    pub image: Option<ExternalImage>,
    pub published_at: Option<DateTime<Utc>>,
}

#[async_trait]
pub trait ExternalPagePort: Send + Sync {
    async fn load(&self, url: &str) -> Result<ExternalPage, AppError>;
}

#[async_trait]
pub trait ExternalImageCache: Send + Sync {
    async fn store(
        &self,
        key: &str,
        content_type: &str,
        bytes: Vec<u8>,
    ) -> Result<String, AppError>;
}

pub struct ObjectImageCache {
    storage: Arc<StorageService>,
}

impl ObjectImageCache {
    pub fn new(storage: Arc<StorageService>) -> Self {
        Self { storage }
    }
}

#[async_trait]
impl ExternalImageCache for ObjectImageCache {
    async fn store(
        &self,
        key: &str,
        content_type: &str,
        bytes: Vec<u8>,
    ) -> Result<String, AppError> {
        Ok(self
            .storage
            .put_recorded(key, content_type, bytes, None)
            .await?
            .url)
    }
}
