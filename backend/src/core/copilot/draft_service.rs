use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{core::article::ArticleRepository, error::AppError};

#[async_trait]
pub trait ArticleDraftService: Send + Sync {
    async fn create_draft_snapshot(&self, article_id: Uuid) -> Result<Option<Uuid>, AppError>;
    async fn update_draft_content(&self, article_id: Uuid, content: &str) -> Result<(), AppError>;
}

pub struct ArticleDraftAdapter {
    repository: Arc<dyn ArticleRepository>,
}

impl ArticleDraftAdapter {
    pub fn new(repository: Arc<dyn ArticleRepository>) -> Self {
        Self { repository }
    }
}

#[async_trait]
impl ArticleDraftService for ArticleDraftAdapter {
    async fn create_draft_snapshot(&self, article_id: Uuid) -> Result<Option<Uuid>, AppError> {
        self.repository
            .create_draft_snapshot(article_id)
            .await
            .map(Some)
    }

    async fn update_draft_content(&self, article_id: Uuid, content: &str) -> Result<(), AppError> {
        self.repository
            .update_draft_content(article_id, content)
            .await
    }
}

#[async_trait]
impl<T> crate::core::ml::llm::DraftSaver for T
where
    T: ArticleDraftService + Send + Sync + ?Sized,
{
    async fn update_draft_content(
        &self,
        article_id: Uuid,
        markdown_content: &str,
    ) -> Result<(), AppError> {
        ArticleDraftService::update_draft_content(self, article_id, markdown_content).await
    }
}
