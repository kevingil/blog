use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Utc};

use crate::{core::storage::ObjectStore, error::AppError};

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
    store: Arc<dyn ObjectStore>,
    url_prefix: String,
}

impl ObjectImageCache {
    pub fn new(store: Arc<dyn ObjectStore>, url_prefix: impl Into<String>) -> Self {
        Self {
            store,
            url_prefix: url_prefix.into(),
        }
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
        self.store
            .put_with_content_type(key, bytes, content_type)
            .await?;
        Ok(format!(
            "{}/{}",
            self.url_prefix.trim_end_matches('/'),
            key.trim_start_matches('/')
        ))
    }
}
