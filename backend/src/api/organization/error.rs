use std::collections::BTreeMap;

use axum::{
    Json,
    extract::{FromRef, FromRequestParts},
    http::{StatusCode, header::AUTHORIZATION, request::Parts},
    response::{IntoResponse, Response},
};

use crate::{
    api::auth::AuthState,
    core::auth::AccountId,
    error::{AppError, ErrorEnvelope},
};

pub struct OrganizationApiError {
    status: StatusCode,
    body: ErrorEnvelope,
}

impl OrganizationApiError {
    pub fn invalid_body() -> Self {
        Self::new(
            StatusCode::BAD_REQUEST,
            "Invalid request body",
            "INVALID_INPUT",
            None,
        )
    }

    pub fn validations(issues: Vec<(&'static str, String)>) -> Self {
        let error = issues
            .first()
            .map(|(field, message)| format!("{field}: {message}"))
            .unwrap_or_else(|| "validation failed".to_owned());
        let details = issues
            .into_iter()
            .map(|(field, message)| (field.to_owned(), message))
            .collect();
        Self::new(
            StatusCode::BAD_REQUEST,
            error,
            "VALIDATION_ERROR",
            Some(details),
        )
    }

    fn unauthorized(message: &'static str) -> Self {
        Self::new(StatusCode::UNAUTHORIZED, message, "UNAUTHORIZED", None)
    }

    fn new(
        status: StatusCode,
        error: impl Into<String>,
        code: &'static str,
        details: Option<BTreeMap<String, String>>,
    ) -> Self {
        Self {
            status,
            body: ErrorEnvelope {
                error: error.into(),
                code,
                details,
            },
        }
    }
}

impl From<AppError> for OrganizationApiError {
    fn from(error: AppError) -> Self {
        match error {
            AppError::InvalidInput(message) => {
                Self::new(StatusCode::BAD_REQUEST, message, "INVALID_INPUT", None)
            }
            AppError::Unauthorized => Self::unauthorized("Not authenticated"),
            AppError::Forbidden => {
                Self::new(StatusCode::FORBIDDEN, "access forbidden", "FORBIDDEN", None)
            }
            AppError::NotFound => Self::new(
                StatusCode::NOT_FOUND,
                "resource not found",
                "NOT_FOUND",
                None,
            ),
            AppError::Conflict(message) => {
                Self::new(StatusCode::CONFLICT, message, "ALREADY_EXISTS", None)
            }
            AppError::Database => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Database error",
                "DATABASE_ERROR",
                None,
            ),
            AppError::External => Self::new(
                StatusCode::BAD_GATEWAY,
                "external service operation failed",
                "EXTERNAL_SERVICE_ERROR",
                None,
            ),
            AppError::Internal => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error",
                "INTERNAL_ERROR",
                None,
            ),
        }
    }
}

impl IntoResponse for OrganizationApiError {
    fn into_response(self) -> Response {
        (self.status, Json(self.body)).into_response()
    }
}

pub struct OrganizationAuthenticated(pub AccountId);

impl<S> FromRequestParts<S> for OrganizationAuthenticated
where
    S: Send + Sync,
    AuthState: FromRef<S>,
{
    type Rejection = OrganizationApiError;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let header = parts
            .headers
            .get(AUTHORIZATION)
            .ok_or_else(|| OrganizationApiError::unauthorized("Not authenticated"))?
            .to_str()
            .map_err(|_| OrganizationApiError::unauthorized("Invalid token format"))?;
        if !header.starts_with("Bearer ") {
            return Err(OrganizationApiError::unauthorized("Invalid token format"));
        }
        let account_id = AuthState::from_ref(state)
            .authenticate(&parts.headers)
            .map_err(|_| OrganizationApiError::unauthorized("Invalid or expired token"))?;
        Ok(Self(account_id))
    }
}
