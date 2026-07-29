use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::ImageGeneration;

#[async_trait]
pub trait ImageRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<ImageGeneration, AppError>;
    async fn find_by_request_id(&self, request_id: &str) -> Result<ImageGeneration, AppError>;
    async fn save(&self, image: &mut ImageGeneration) -> Result<(), AppError>;
    async fn update(&self, image: &ImageGeneration) -> Result<(), AppError>;
}
