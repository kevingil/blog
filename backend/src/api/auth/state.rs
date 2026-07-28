use std::sync::Arc;

use crate::core::auth::AuthService;

#[derive(Clone)]
pub struct AuthState {
    service: Arc<AuthService>,
}

impl AuthState {
    pub const fn new(service: Arc<AuthService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &AuthService {
        &self.service
    }
}
