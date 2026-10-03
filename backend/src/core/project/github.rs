use async_trait::async_trait;

use crate::error::AppError;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GithubProjectDraft {
    pub title: String,
    pub description: String,
    pub content: String,
    pub tags: Vec<String>,
    pub image_url: String,
    pub url: String,
}

#[async_trait]
pub trait GithubImportPort: Send + Sync {
    async fn import_repository(&self, url: &str) -> Result<GithubProjectDraft, AppError>;
}
