use std::collections::BTreeMap;

use axum::{
    Json,
    extract::{FromRef, FromRequestParts},
    http::{StatusCode, header::AUTHORIZATION, request::Parts},
    response::{IntoResponse, Response},
};

use crate::{
    api::auth::AuthState,
    error::{AppError, ErrorEnvelope},
};

pub struct ProjectApiError {
    status: StatusCode,
    body: ErrorEnvelope,
    cause: String,
}

impl ProjectApiError {
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

    pub fn not_found(message: impl Into<String>) -> Self {
        Self::new(StatusCode::NOT_FOUND, message, "NOT_FOUND", None)
    }

    pub fn bad_gateway(message: impl Into<String>) -> Self {
        Self::new(
            StatusCode::BAD_GATEWAY,
            message,
            "EXTERNAL_SERVICE_ERROR",
            None,
        )
    }

    fn unauthorized(message: &'static str) -> Self {
        Self::new(StatusCode::UNAUTHORIZED, message, "UNAUTHORIZED", None)
    }

    pub(crate) fn with_cause(mut self, cause: impl std::fmt::Display) -> Self {
        self.cause = cause.to_string();
        self
    }

    fn new(
        status: StatusCode,
        error: impl Into<String>,
        code: &'static str,
        details: Option<BTreeMap<String, String>>,
    ) -> Self {
        let error = error.into();
        Self {
            status,
            cause: error.clone(),
            body: ErrorEnvelope {
                error,
                code,
                details,
            },
        }
    }
}

impl From<AppError> for ProjectApiError {
    fn from(error: AppError) -> Self {
        let cause = error.to_string();
        let mut mapped = match error {
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
            AppError::Database(_) => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Database error",
                "DATABASE_ERROR",
                None,
            ),
            AppError::External(_) => Self::new(
                StatusCode::BAD_GATEWAY,
                "external service operation failed",
                "EXTERNAL_SERVICE_ERROR",
                None,
            ),
            AppError::Internal(_) => Self::new(
                StatusCode::INTERNAL_SERVER_ERROR,
                "Internal server error",
                "INTERNAL_ERROR",
                None,
            ),
        };
        mapped.cause = cause;
        mapped
    }
}

impl IntoResponse for ProjectApiError {
    fn into_response(self) -> Response {
        crate::error::log_error_response(self.status, self.body.code, &self.cause);
        (self.status, Json(self.body)).into_response()
    }
}

pub struct ProjectAuthenticated;

impl<S> FromRequestParts<S> for ProjectAuthenticated
where
    S: Send + Sync,
    AuthState: FromRef<S>,
{
    type Rejection = ProjectApiError;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let header = parts
            .headers
            .get(AUTHORIZATION)
            .ok_or_else(|| ProjectApiError::unauthorized("Not authenticated"))?
            .to_str()
            .map_err(|_| ProjectApiError::unauthorized("Invalid token format"))?;
        if !header.starts_with("Bearer ") {
            return Err(ProjectApiError::unauthorized("Invalid token format"));
        }
        AuthState::from_ref(state)
            .authenticate(&parts.headers)
            .map_err(|_| ProjectApiError::unauthorized("Invalid or expired token"))?;
        Ok(Self)
    }
}
