use std::sync::Arc;

use crate::core::profile::ProfileService;

#[derive(Clone)]
pub struct ProfileState {
    service: Arc<ProfileService>,
}

impl ProfileState {
    pub const fn new(service: Arc<ProfileService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &ProfileService {
        &self.service
    }
}
