use std::sync::Arc;

use crate::core::organization::OrganizationService;

#[derive(Clone)]
pub struct OrganizationState {
    service: Arc<OrganizationService>,
}

impl OrganizationState {
    pub const fn new(service: Arc<OrganizationService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &OrganizationService {
        &self.service
    }
}
