use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{
    core::{
        auth::{AccountId, AccountRepository},
        datasource::{DataSourceService, RecommendationService},
    },
    error::AppError,
};

#[async_trait]
pub trait AccountOrganizationPort: Send + Sync {
    async fn organization_id(&self, account_id: AccountId) -> Result<Option<Uuid>, AppError>;
}

#[async_trait]
impl<T> AccountOrganizationPort for T
where
    T: AccountRepository + Send + Sync,
{
    async fn organization_id(&self, account_id: AccountId) -> Result<Option<Uuid>, AppError> {
        self.find_by_id(account_id)
            .await
            .map(|account| account.and_then(|account| account.organization_id))
    }
}

#[derive(Clone)]
pub struct DataSourceState {
    service: Arc<DataSourceService>,
    recommendations: Arc<RecommendationService>,
    accounts: Arc<dyn AccountOrganizationPort>,
}

impl DataSourceState {
    pub fn new(
        service: Arc<DataSourceService>,
        recommendations: Arc<RecommendationService>,
        accounts: Arc<dyn AccountOrganizationPort>,
    ) -> Self {
        Self {
            service,
            recommendations,
            accounts,
        }
    }

    pub fn service(&self) -> &DataSourceService {
        &self.service
    }

    pub fn recommendations(&self) -> &RecommendationService {
        &self.recommendations
    }

    pub async fn organization_id(&self, account_id: AccountId) -> Option<Uuid> {
        // The Go middleware treats account lookup failure as "no organization"
        // and falls back to account-owned data sources.
        self.accounts
            .organization_id(account_id)
            .await
            .ok()
            .flatten()
    }
}
