use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use chrono::Utc;
use uuid::Uuid;

use crate::error::AppError;

use super::{service::McpConnectorRepository, types::McpConnector};

#[derive(Clone, Default)]
pub struct InMemoryMcpConnectorRepository {
    connectors: Arc<Mutex<Vec<McpConnector>>>,
}

#[async_trait]
impl McpConnectorRepository for InMemoryMcpConnectorRepository {
    async fn list(&self) -> Result<Vec<McpConnector>, AppError> {
        self.connectors
            .lock()
            .map(|connectors| connectors.clone())
            .map_err(|_| AppError::Internal)
    }

    async fn find(&self, id: Uuid) -> Result<McpConnector, AppError> {
        self.connectors
            .lock()
            .map_err(|_| AppError::Internal)?
            .iter()
            .find(|connector| connector.id == id)
            .cloned()
            .ok_or(AppError::NotFound)
    }

    async fn create(&self, connector: &McpConnector) -> Result<McpConnector, AppError> {
        let now = Utc::now();
        let mut saved = connector.clone();
        saved.created_at = Some(now);
        saved.updated_at = Some(now);
        self.connectors
            .lock()
            .map_err(|_| AppError::Internal)?
            .push(saved.clone());
        Ok(saved)
    }

    async fn update(&self, connector: &McpConnector) -> Result<McpConnector, AppError> {
        let mut connectors = self.connectors.lock().map_err(|_| AppError::Internal)?;
        let Some(existing) = connectors.iter_mut().find(|item| item.id == connector.id) else {
            return Err(AppError::NotFound);
        };
        let mut saved = connector.clone();
        saved.created_at = existing.created_at;
        saved.updated_at = Some(Utc::now());
        *existing = saved.clone();
        Ok(saved)
    }

    async fn delete(&self, id: Uuid) -> Result<(), AppError> {
        let mut connectors = self.connectors.lock().map_err(|_| AppError::Internal)?;
        let before = connectors.len();
        connectors.retain(|connector| connector.id != id);
        if connectors.len() == before {
            Err(AppError::NotFound)
        } else {
            Ok(())
        }
    }
}
