use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{core::article::ArticleService, error::AppError};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GenerationRequest {
    pub message: String,
    pub article_id: Uuid,
}

#[async_trait]
pub trait ArticleGenerationQueue: Send + Sync {
    async fn submit(&self, request: GenerationRequest) -> Result<String, AppError>;
}

#[derive(Clone)]
pub struct ArticleState {
    service: Arc<ArticleService>,
    generation_queue: Arc<dyn ArticleGenerationQueue>,
}

impl ArticleState {
    pub fn new(
        service: Arc<ArticleService>,
        generation_queue: Arc<dyn ArticleGenerationQueue>,
    ) -> Self {
        Self {
            service,
            generation_queue,
        }
    }

    pub fn service(&self) -> Result<&ArticleService, AppError> {
        Ok(&self.service)
    }

    pub fn generation_queue(&self) -> Result<&dyn ArticleGenerationQueue, AppError> {
        Ok(self.generation_queue.as_ref())
    }
}
