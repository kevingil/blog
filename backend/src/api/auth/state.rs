use std::sync::Arc;

use axum::{
    extract::{FromRef, FromRequestParts},
    http::{HeaderMap, header::AUTHORIZATION, request::Parts},
};

use crate::{
    core::auth::{AccountId, AuthService},
    error::AppError,
};

#[derive(Clone)]
pub struct AuthState {
    service: Arc<AuthService>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AuthenticatedAccount(pub AccountId);

impl AuthenticatedAccount {
    pub const fn into_inner(self) -> AccountId {
        self.0
    }
}

impl<S> FromRequestParts<S> for AuthenticatedAccount
where
    S: Send + Sync,
    AuthState: FromRef<S>,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let auth = AuthState::from_ref(state);
        auth.authenticate(&parts.headers).map(Self)
    }
}

impl AuthState {
    pub const fn new(service: Arc<AuthService>) -> Self {
        Self { service }
    }

    pub fn service(&self) -> &AuthService {
        &self.service
    }

    pub fn authenticate(&self, headers: &HeaderMap) -> Result<AccountId, AppError> {
        let header = headers
            .get(AUTHORIZATION)
            .ok_or(AppError::Unauthorized)?
            .to_str()
            .map_err(|_| AppError::Unauthorized)?;
        let token = header
            .strip_prefix("Bearer ")
            .filter(|token| !token.is_empty())
            .ok_or(AppError::Unauthorized)?;
        self.service.account_id_from_token(token)
    }
}
