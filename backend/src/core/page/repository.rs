use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{Page, PageListOptions};

#[async_trait]
pub trait PageRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Page, AppError>;
    async fn find_by_slug(&self, slug: &str) -> Result<Page, AppError>;
    async fn list(&self, options: PageListOptions) -> Result<(Vec<Page>, i64), AppError>;
    async fn save(&self, page: &mut Page) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}
