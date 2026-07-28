use async_trait::async_trait;
use serde_json::Value;
use uuid::Uuid;

use crate::error::AppError;

use super::ChatMessage;

#[async_trait]
pub trait ChatMessageRepository: Send + Sync {
    async fn create(&self, message: &mut ChatMessage) -> Result<(), AppError>;
    async fn find_by_id(&self, id: Uuid) -> Result<ChatMessage, AppError>;
    async fn list_by_article(
        &self,
        article_id: Uuid,
        limit: i64,
    ) -> Result<Vec<ChatMessage>, AppError>;
    async fn list_pending_artifacts(&self, article_id: Uuid) -> Result<Vec<ChatMessage>, AppError>;
    async fn update(&self, message: &ChatMessage) -> Result<(), AppError>;
    async fn update_metadata(&self, id: Uuid, metadata: Value) -> Result<u64, AppError>;
    async fn delete_by_article(&self, article_id: Uuid) -> Result<u64, AppError>;
}
