use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{Project, ProjectListOptions};

#[async_trait]
pub trait ProjectRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Project, AppError>;
    async fn list(&self, options: ProjectListOptions) -> Result<(Vec<Project>, i64), AppError>;
    async fn save(&self, project: &mut Project) -> Result<(), AppError>;
    async fn update(&self, project: &Project) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}
