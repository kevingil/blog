use std::sync::Arc;

use crate::{core::storage::StorageService, error::AppError};

#[derive(Clone)]
pub struct StorageState {
    service: Arc<StorageService>,
}

impl StorageState {
    pub fn new(service: Arc<StorageService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> Result<&StorageService, AppError> {
        Ok(&self.service)
    }
}
