use async_trait::async_trait;
use uuid::Uuid;

use crate::error::AppError;

use super::{ProfileAccount, PublicProfile, SiteSettings};

#[async_trait]
pub trait ProfileRepository: Send + Sync {
    async fn get_public_profile(&self) -> Result<PublicProfile, AppError>;
    async fn is_user_admin(&self, user_id: Uuid) -> Result<bool, AppError>;
}

#[async_trait]
pub trait ProfileAccountRepository: Send + Sync {
    async fn find_profile_account(&self, id: Uuid) -> Result<ProfileAccount, AppError>;
    async fn update_profile_account(&self, account: &ProfileAccount) -> Result<(), AppError>;
}

#[async_trait]
pub trait SiteSettingsRepository: Send + Sync {
    async fn get(&self) -> Result<SiteSettings, AppError>;
    async fn save(&self, settings: &mut SiteSettings) -> Result<(), AppError>;
}
