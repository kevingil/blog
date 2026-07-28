use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{core::image::ImageService, error::AppError};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ImageGenerationJob {
    pub image_id: Uuid,
    pub request_id: String,
    pub article_id: Uuid,
    pub prompt: String,
    pub generate_prompt: bool,
}

#[async_trait]
pub trait ImageGenerationQueue: Send + Sync {
    fn provider(&self) -> &str;
    fn model_name(&self) -> &str;
    async fn enqueue(&self, job: ImageGenerationJob) -> Result<(), AppError>;
}

#[derive(Clone)]
pub struct ImageState {
    service: Arc<ImageService>,
    queue: Arc<dyn ImageGenerationQueue>,
}

impl ImageState {
    pub fn new(service: Arc<ImageService>, queue: Arc<dyn ImageGenerationQueue>) -> Self {
        Self { service, queue }
    }

    pub fn service(&self) -> &ImageService {
        &self.service
    }

    pub fn queue(&self) -> &dyn ImageGenerationQueue {
        self.queue.as_ref()
    }
}
