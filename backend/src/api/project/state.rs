use std::sync::Arc;

use crate::core::project::{GithubImportPort, ProjectService};

#[derive(Clone)]
pub struct ProjectState {
    service: Arc<ProjectService>,
    github: Option<Arc<dyn GithubImportPort>>,
}

impl ProjectState {
    pub const fn new(service: Arc<ProjectService>) -> Self {
        Self {
            service,
            github: None,
        }
    }

    pub fn with_github(mut self, github: Arc<dyn GithubImportPort>) -> Self {
        self.github = Some(github);
        self
    }

    pub fn service(&self) -> &ProjectService {
        &self.service
    }

    pub fn github(&self) -> Option<Arc<dyn GithubImportPort>> {
        self.github.clone()
    }
}
