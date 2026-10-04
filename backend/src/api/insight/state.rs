use std::sync::Arc;

use async_trait::async_trait;
use uuid::Uuid;

use crate::{
    core::{
        auth::{AccountId, AccountRepository},
        insight::{InsightService, TrackerCatalog},
        worker::WorkerManager,
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
    trackers: Option<Arc<TrackerCatalog>>,
    workers: Option<Arc<WorkerManager>>,
}

impl InsightState {
    pub fn new(service: Arc<InsightService>, accounts: Arc<dyn AccountOrganizationPort>) -> Self {
        Self {
            service,
            accounts,
            trackers: None,
            workers: None,
        }
    }

    pub fn with_trackers(mut self, trackers: Arc<TrackerCatalog>) -> Self {
        self.trackers = Some(trackers);
        self
    }

    pub fn with_workers(mut self, workers: Arc<WorkerManager>) -> Self {
        self.workers = Some(workers);
        self
    }

    pub fn trackers(&self) -> Result<&TrackerCatalog, AppError> {
        self.trackers.as_deref().ok_or(AppError::Internal)
    }

    pub fn workers(&self) -> Option<&Arc<WorkerManager>> {
        self.workers.as_ref()
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
