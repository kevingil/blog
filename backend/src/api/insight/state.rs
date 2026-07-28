use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{
    core::{
        auth::{AccountId, AccountRepository},
        insight::InsightService,
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
pub struct InsightState {
    service: Arc<InsightService>,
    accounts: Arc<dyn AccountOrganizationPort>,
}

impl InsightState {
    pub fn new(service: Arc<InsightService>, accounts: Arc<dyn AccountOrganizationPort>) -> Self {
        Self { service, accounts }
    }

    pub fn service(&self) -> &InsightService {
        &self.service
    }

    pub async fn organization_id(&self, account_id: AccountId) -> Option<Uuid> {
        self.accounts
            .organization_id(account_id)
            .await
            .ok()
            .flatten()
    }
}
