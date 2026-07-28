use std::sync::Arc;

use crate::core::project::ProjectService;

#[derive(Clone)]
pub struct ProjectState {
    service: Arc<ProjectService>,
}

impl ProjectState {
    pub const fn new(service: Arc<ProjectService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &ProjectService {
        &self.service
    }
}
