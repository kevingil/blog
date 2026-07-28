use async_trait::async_trait;

use crate::error::AppError;

use super::Tag;

#[async_trait]
pub trait TagRepository: Send + Sync {
    async fn find_by_id(&self, id: i32) -> Result<Tag, AppError>;
    async fn find_by_name(&self, name: &str) -> Result<Tag, AppError>;
    async fn find_by_ids(&self, ids: &[i64]) -> Result<Vec<Tag>, AppError>;
    async fn ensure_exists(&self, names: &[String]) -> Result<Vec<i64>, AppError>;
    async fn list(&self) -> Result<Vec<Tag>, AppError>;
    async fn save(&self, tag: &mut Tag) -> Result<(), AppError>;
    async fn delete(&self, id: i32) -> Result<(), AppError>;
    async fn is_used(&self, id: i32) -> Result<bool, AppError>;
}
