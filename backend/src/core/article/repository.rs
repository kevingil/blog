use std::time::Duration;

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::error::AppError;

use super::{Article, ArticleListOptions, ArticleSearchOptions, ArticleVersion};

#[async_trait]
pub trait ArticleRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Article, AppError>;
    async fn find_by_slug(&self, slug: &str) -> Result<Article, AppError>;
    async fn list(&self, options: ArticleListOptions) -> Result<(Vec<Article>, i64), AppError>;
    async fn search(&self, options: ArticleSearchOptions) -> Result<(Vec<Article>, i64), AppError>;
    async fn search_by_embedding(
        &self,
        embedding: &[f32],
        limit: i64,
    ) -> Result<Vec<Article>, AppError>;
    async fn save(&self, article: &mut Article) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
    async fn get_popular_tags(&self, limit: i64) -> Result<Vec<i64>, AppError>;
    async fn slug_exists(&self, slug: &str, exclude_id: Option<Uuid>) -> Result<bool, AppError>;
    async fn save_draft(&self, article: &mut Article) -> Result<(), AppError>;
    async fn publish(
        &self,
        article: &mut Article,
        published_at: Option<DateTime<Utc>>,
    ) -> Result<(), AppError>;
    async fn unpublish(&self, article: &mut Article) -> Result<(), AppError>;
    async fn list_versions(&self, article_id: Uuid) -> Result<Vec<ArticleVersion>, AppError>;
    async fn get_version(&self, version_id: Uuid) -> Result<ArticleVersion, AppError>;
    async fn revert_to_version(&self, article_id: Uuid, version_id: Uuid) -> Result<(), AppError>;
    async fn create_draft_snapshot(&self, article_id: Uuid) -> Result<Uuid, AppError>;
    async fn update_draft_content(
        &self,
        article_id: Uuid,
        html_content: &str,
    ) -> Result<(), AppError>;

    /// Wait for owned fire-and-forget version writes during graceful shutdown or tests.
    async fn drain_background_tasks(&self) -> Result<(), AppError>;

    /// Close background task admission and wait up to `timeout` before aborting stragglers.
    async fn shutdown_background_tasks(&self, timeout: Duration) -> Result<(), AppError>;
}
