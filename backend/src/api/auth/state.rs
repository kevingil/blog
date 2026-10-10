use std::sync::Arc;

use axum::{
    extract::{FromRef, FromRequestParts},
    http::{
        HeaderMap,
        header::{AUTHORIZATION, SEC_WEBSOCKET_PROTOCOL},
        request::Parts,
    },
};

use crate::{
    core::auth::{Account, AccountId, AuthService},
    error::AppError,
};

use super::throttle::LoginThrottle;

/// Browsers cannot set headers on a WebSocket handshake, so socket clients
/// offer the subprotocols `bearer` and `<token>`. The server selects `bearer`.
pub const WEBSOCKET_BEARER_PROTOCOL: &str = "bearer";

#[derive(Clone)]
pub struct AuthState {
    service: Arc<AuthService>,
    login_throttle: Arc<LoginThrottle>,
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
        let account = auth.authenticate(&parts.headers).await?;
        Ok(Self(account.id))
    }
}

/// The caller's account on endpoints that are public but show signed-in
/// authors more. A missing, expired, or revoked token reads as anonymous.
#[derive(Debug, Clone, PartialEq)]
pub struct OptionalAccount(pub Option<Account>);

impl<S> FromRequestParts<S> for OptionalAccount
where
    S: Send + Sync,
    AuthState: FromRef<S>,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let auth = AuthState::from_ref(state);
        auth.optional_account(&parts.headers).await.map(Self)
    }
}

impl AuthState {
    pub fn new(service: Arc<AuthService>) -> Self {
        Self {
            service,
            login_throttle: Arc::new(LoginThrottle::default()),
        }
    }

    pub fn service(&self) -> &AuthService {
        &self.service
    }

    pub fn login_throttle(&self) -> &LoginThrottle {
        &self.login_throttle
    }

    pub async fn authenticate(&self, headers: &HeaderMap) -> Result<Account, AppError> {
        let token = bearer_token(headers).ok_or(AppError::Unauthorized)?;
        self.service.authenticate(token).await
    }

    pub async fn optional_account(&self, headers: &HeaderMap) -> Result<Option<Account>, AppError> {
        match self.authenticate(headers).await {
            Ok(account) => Ok(Some(account)),
            Err(AppError::Unauthorized) => Ok(None),
            Err(error) => Err(error),
        }
    }

    pub async fn authenticate_websocket(&self, headers: &HeaderMap) -> Result<Account, AppError> {
        let token = websocket_bearer_token(headers).ok_or(AppError::Unauthorized)?;
        self.service.authenticate(token).await
    }
}

pub fn bearer_token(headers: &HeaderMap) -> Option<&str> {
    headers
        .get(AUTHORIZATION)?
        .to_str()
        .ok()?
        .strip_prefix("Bearer ")
        .filter(|token| !token.is_empty())
}

fn websocket_bearer_token(headers: &HeaderMap) -> Option<&str> {
    let mut protocols = headers
        .get(SEC_WEBSOCKET_PROTOCOL)?
        .to_str()
        .ok()?
        .split(',')
        .map(str::trim);
    protocols.find(|protocol| *protocol == WEBSOCKET_BEARER_PROTOCOL)?;
    protocols.next().filter(|token| !token.is_empty())
}
