use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::Organization;

#[async_trait]
pub trait OrganizationRepository: Send + Sync {
    async fn find_by_id(&self, id: Uuid) -> Result<Organization, AppError>;
    async fn find_by_slug(&self, slug: &str) -> Result<Organization, AppError>;
    async fn list(&self) -> Result<Vec<Organization>, AppError>;
    async fn save(&self, organization: &mut Organization) -> Result<(), AppError>;
    async fn update(&self, organization: &Organization) -> Result<(), AppError>;
    async fn delete(&self, id: Uuid) -> Result<(), AppError>;
}

#[async_trait]
pub trait OrganizationAccountRepository: Send + Sync {
    async fn set_organization(
        &self,
        account_id: Uuid,
        organization_id: Option<Uuid>,
    ) -> Result<bool, AppError>;
}
