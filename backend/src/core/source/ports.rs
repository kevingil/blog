use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{ScrapedContent, Source, SourceListOptions, SourceWithArticle};

#[async_trait]
pub trait SourceRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Source, AppError>;
    async fn find_by_article_id(&self, article_id: Uuid) -> Result<Vec<Source>, AppError>;
    async fn list(
        &self,
        options: SourceListOptions,
    ) -> Result<(Vec<SourceWithArticle>, i64), AppError>;
    async fn save(&self, source: &mut Source) -> Result<(), AppError>;
    async fn update(&self, source: &Source) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
    async fn search_similar(
        &self,
        article_id: Uuid,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Source>, AppError>;
}

#[async_trait]
pub trait ArticleLookupPort: Send + Sync {
    async fn ensure_exists(&self, article_id: Uuid) -> Result<(), AppError>;
}

#[async_trait]
pub trait EmbeddingPort: Send + Sync {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>, AppError>;
}

#[async_trait]
pub trait FetchExtractPort: Send + Sync {
    /// Fetches with the Go oracle's 30 second bound and extracts PDF or HTML text.
    async fn fetch_extract(&self, url: &str) -> Result<ScrapedContent, AppError>;
}
