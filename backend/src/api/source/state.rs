use std::sync::Arc;

use crate::core::source::SourceService;

#[derive(Clone)]
pub struct SourceState {
    service: Arc<SourceService>,
}

impl SourceState {
    pub const fn new(service: Arc<SourceService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &SourceService {
        &self.service
    }
}
