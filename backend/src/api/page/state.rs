use std::sync::Arc;

use crate::core::page::PageService;

#[derive(Clone)]
pub struct PageState {
    service: Arc<PageService>,
}

impl PageState {
    pub const fn new(service: Arc<PageService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &PageService {
        &self.service
    }
}
